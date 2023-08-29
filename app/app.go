/*
 *    Copyright 2023 iFood
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package app

import (
	"context"
	adaptersin "eagle-eye/adapters/in"
	adaptersout "eagle-eye/adapters/out"
	"eagle-eye/common"
	"eagle-eye/config"
	"eagle-eye/domain/entities"
	portsout "eagle-eye/domain/ports/out"
	"eagle-eye/domain/services"
	"eagle-eye/domain/services/cleanup"
	"eagle-eye/domain/services/filters"
	"eagle-eye/domain/services/notification"
	"eagle-eye/domain/services/preprocess"
	"eagle-eye/domain/services/scan"
	"eagle-eye/domain/services/stages"
	eaglehttp "eagle-eye/http"
	"eagle-eye/logging"
	"eagle-eye/metrics"
	"eagle-eye/pkg/awsutils"
	"fmt"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/uber-go/tally/v4"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
	"io"

	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	virusTotalHourRateLimit   = 20
	virusTotalMinuteRateLimit = 4
	scanWaitList              = "scan-wait-list"
)

//nolint:cyclop
func Start(ctx context.Context) error {
	runtime.GOMAXPROCS(1)

	appConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Enable Datadog tracer
	tracer.Start()
	defer tracer.Stop()

	// Enable Datadog Profiler
	if err = profiler.Start(); err != nil {
		return err
	}
	defer profiler.Stop()

	logger, err := logging.NewZapLogger(appConfig.Scanner.DebugLog)
	if err != nil {
		return err
	}

	var metricsHandler http.Handler
	var metricsScope tally.Scope
	var metricsClose io.Closer

	if appConfig.HTTPServer.Metrics {
		metricsScope, metricsHandler, metricsClose = metrics.NewPrometheusScope()
		defer metricsClose.Close()
	} else {
		metricsScope, metricsHandler, _ = metrics.NewNoopScope()
	}

	var client awsutils.Clients
	session, err := client.Session(appConfig.Aws.Region, appConfig.Aws.Resolver)

	if err != nil {
		return fmt.Errorf("failed to initialize aws client. Error: %s, Region: %s, Resolver: %s", err, appConfig.Aws.Region, appConfig.Aws.Resolver)
	}

	cache := adaptersout.NewCache(appConfig.Redis.URL, appConfig.Redis.Password, appConfig.Redis.UseTLS)

	smsViewer := adaptersout.NewSMSViewer(session, appConfig.Notification.Phones)
	slackViewer := adaptersout.NewSlackViewer(appConfig.Notification.Slack.AppToken, appConfig.Notification.Slack.Webhook, appConfig.Notification.Slack.ChannelID)
	viewers := map[entities.ViewerMimetype]portsout.Viewer{adaptersin.MIMEApplicationSMS: smsViewer, adaptersin.MIMEApplicationSlack: slackViewer}

	localStorageFactory := adaptersout.NewLocalStorageFactory(appConfig.Scanner.MaxStorageSize)
	remoteStorageFactory := adaptersout.NewRemoteStorageFactory(session, nil)

	sqsService := awsutils.SQS{}
	sqsService.Init(session, nil)

	downloadService := services.NewDownloadService(localStorageFactory, remoteStorageFactory, logger)
	scheduleService := services.NewScheduleService(remoteStorageFactory, cache, appConfig.Scanner.InternalBucket)
	decompressService := services.NewDecompressService(logger)

	rateLimiter := common.NewRateLimiter(appConfig.Redis.URL, appConfig.Redis.Password, appConfig.Redis.UseTLS, common.RateLimitConfig{
		Hour:   virusTotalHourRateLimit,
		Minute: virusTotalMinuteRateLimit,
		Key:    "virustotal",
	})

	remoteScan := adaptersout.NewVirusTotalScanner(appConfig.Scanner.Virustotal.APIkey, appConfig.Scanner.Virustotal.Threshold, rateLimiter)

	aggregateRepo := adaptersout.NewCacheAggregateRepo(cache, logger)
	individualRepo := adaptersout.NewCacheIndividualRepository(cache, logger)
	scheduleRepo := adaptersout.NewCacheScheduleScanRepository[entities.ScheduleItem](cache, scanWaitList)

	// Channels
	inputChannel := make(chan *entities.ScanRequest)
	cleanupChannel := make(chan *stages.Cleanup[entities.ScanRequest])

	// Filters
	applicationFilter := filters.NewApplicationFilter(downloadService, localStorageFactory, logger)
	probabilisticFilter := filters.NewProbabilisticFilter(appConfig.Scanner.ScanProbabilities, logger)
	bypassFilter := filters.NewBypassfilter(appConfig.Scanner.Allowlist, appConfig.Scanner.SizeLimit, logger)
	filterHandler := filters.NewFilterHandler([]filters.FilterJob{applicationFilter, probabilisticFilter, bypassFilter}, logger)

	// Preprocessors
	downloader := preprocess.NewDownloader(downloadService)
	preDecryption := preprocess.NewPreDecryption(logger)
	posDecryption := preprocess.NewPostDecryption(localStorageFactory, appConfig.Scanner.Cipherpass, logger)
	decompress := preprocess.NewDecompress(decompressService, localStorageFactory)
	individualScanUpdate := preprocess.NewIndividualScanUpdate(scheduleService, logger)
	preprocessHandler := preprocess.NewPreprocessHandler([]preprocess.Job{downloader, preDecryption, posDecryption, decompress, individualScanUpdate}, logger)

	// Scanners
	yaraScanner, err := scan.NewYaraScanner(appConfig.Scanner.Yara.Rulesdir, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize yara scanner. Error: %s", err)
	}

	entropyScanner := scan.NewEntropyScanner(logger)
	remoteScanner := scan.NewRemoteScanner(remoteScan, time.Minute, scheduleRepo, logger)
	scanService := scan.NewScanService(localStorageFactory, []scan.SyncProcess{entropyScanner, yaraScanner}, []scan.AsyncProcess{remoteScanner}, logger)
	scanHandler := scan.NewScanHandler(scanService, []scan.SyncProcess{entropyScanner, yaraScanner}, []scan.AsyncProcess{remoteScanner}, time.Minute, logger)

	// Notifications
	emergencyService := notification.NewEmergencyService([]portsout.Viewer{slackViewer, smsViewer}, logger)
	aggregateStatistics := notification.NewAggregateStatistics(aggregateRepo, logger)
	individualStatistics := notification.NewIndividualStatistics(individualRepo, logger)
	notificationHandler := notification.NewNotificationHandler([]notification.Job{aggregateStatistics, emergencyService, individualStatistics}, logger)

	// Cleanups
	queueCleanup := cleanup.NewQueueCleanup(appConfig.Aws.Queue, sqsService, logger)
	scheduleCleanup := cleanup.NewScheduleCleanup(scheduleService, logger)
	storageCleanup := cleanup.NewStorageCleanup(localStorageFactory, logger)
	cleanupHandler := cleanup.NewCleanupHandler([]cleanup.Job{&queueCleanup, &storageCleanup, scheduleCleanup}, logger)

	// Stages initialization
	filterStage := stages.NewStage[entities.ScanRequest, entities.ScanRequest](filterHandler, inputChannel, cleanupChannel, logger)
	preprocessStage := stages.NewStage[entities.ScanRequest, entities.ScanRequest](preprocessHandler, filterStage.Output(), cleanupChannel, logger)
	scanStage := stages.NewStage[entities.ScanRequest, entities.ScanResult](scanHandler, preprocessStage.Output(), cleanupChannel, logger)
	notificationStage := stages.NewStage[entities.ScanResult, entities.Empty](notificationHandler, scanStage.Output(), make(chan *stages.Cleanup[entities.ScanResult]), logger)
	cleanupStage := stages.NewStage[stages.Cleanup[entities.ScanRequest], entities.CleanRequest](cleanupHandler, cleanupChannel, chan *stages.Cleanup[stages.Cleanup[entities.ScanRequest]](nil), logger)

	filterStage.Process(ctx)
	preprocessStage.Process(ctx)
	scanStage.Process(ctx)
	notificationStage.Process(ctx)
	cleanupStage.Process(ctx)

	scanHandler.HandleAsync(ctx, scanStage.Output())
	notificationHandler.HandleAsync(ctx, time.Duration(appConfig.Notification.UpdateInterval)*time.Second)

	scanStatisticsService := services.NewScanStatisticsService(aggregateRepo, individualRepo, scheduleService, viewers, logger)

	// Controllers
	queueController := adaptersin.NewQueueController(appConfig.Aws.Queue, localStorageFactory, inputChannel, sqsService, metricsScope, logger)
	go queueController.AsyncScan(ctx)

	scanController := adaptersin.NewScanController(scheduleService, logger)
	statisticsController := adaptersin.NewStatisticsController(scanStatisticsService, logger)

	fiberConfig := eaglehttp.FiberConfig{
		MaxRequestSize:    appConfig.HTTPServer.MaxRequestSize,
		AuthorizationKeys: appConfig.HTTPServer.AuthorizationKeys,
		Profiler:          appConfig.HTTPServer.Profiler,
		Swagger:           appConfig.HTTPServer.Swagger,
		Metrics:           adaptor.HTTPHandler(metricsHandler),
		RequestLogger: func(c *fiber.Ctx) error {
			headers := c.GetReqHeaders()
			// Prevent generating lots of requests because of healthcheck
			if !strings.HasPrefix(c.Path(), "/healthcheck/") && !strings.HasPrefix(c.Path(), "/metrics") {
				logger.Infow("Received webapi request", "caller_type", headers["X-Ifood-Requester-Entity"],
					"requester", headers["X-Ifood-Requester-Service"], "ip", c.IP(), "method", c.Method(), "url", c.BaseURL(), "path", c.Path(),
					"response_status", c.Response().StatusCode())
			}
			return c.Next()
		},
		Readiness: func(c *fiber.Ctx) error {
			if appConfig.Aws.Queue != "" {
				req, err := http.NewRequestWithContext(c.Context(), "GET", appConfig.Aws.Queue, http.NoBody)
				if err != nil {
					logger.Errorw("Failed to create SQS request in readiness.", "error", err)
					return c.Status(fiber.StatusServiceUnavailable).SendString(fmt.Sprintf("Failed to create request %s", err))
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					logger.Errorw("Failed to connect to the SQS in readiness.", "error", err)
					return c.Status(fiber.StatusServiceUnavailable).SendString(fmt.Sprintf("SQS not connectable. %s", err))
				}
				defer resp.Body.Close()
			}

			_, err = cache.List("XXXXX")
			if err != nil {
				logger.Errorw("Failed to connect to the cache.", "error", err)
				return c.Status(fiber.StatusServiceUnavailable).SendString(fmt.Sprintf("Elasticache not connectable. %s", err))
			}

			return c.SendStatus(fiber.StatusOK)
		},
		Liveness: func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		},
		Handlers: []eaglehttp.Handler{
			{HTTPMethod: "POST", Path: "/files", HandlerFunc: scanController.ScanFile},
			{HTTPMethod: "POST", Path: "/objects", HandlerFunc: scanController.ScanObject},
			{HTTPMethod: "GET", Path: "/objects", HandlerFunc: statisticsController.GetAggregateResult},
			{HTTPMethod: "GET", Path: "/files/:id", HandlerFunc: statisticsController.GetFileResult},
			{HTTPMethod: "GET", Path: "/objects/:id", HandlerFunc: statisticsController.GetObjectResult},
		},
	}

	app, err := eaglehttp.CreateFiberApp(fiberConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize fiber framework. Error: %s", err)
	}

	return app.Listen(fmt.Sprintf(":%d", appConfig.HTTPServer.Port))
}
