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

package in

import (
	adapterentities "eagle-eye/adapters/entities"
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/services"
	"eagle-eye/logging"
	"errors"
	"github.com/gofiber/fiber/v2"
	"time"
)

const (
	MIMEApplicationSMS   entities.ViewerMimetype = "application/vnd.eagleeye.scanner.sms.v1"
	MIMEApplicationSlack entities.ViewerMimetype = "application/vnd.eagleeye.scanner.slack.v1"
	MIMEApplicationJSON  entities.ViewerMimetype = "application/json"
)

const (
	errInvalidScanID  = "invalid scan id"
	errScanIDNotFound = "scan id not found"

	errScanInProgress = "scan in progress"
	errScanIsWaiting  = "scan is waiting"
	errScanFailed     = "scan failed"

	errUnknownScanState      = "unknown scan state"
	errUnsupportedAcceptType = "unsupported accept type"
)

type StatisticsController struct {
	scanStatisticsService services.StatisticsService
	logger                logging.Logger
}

func NewStatisticsController(scanStatisticsService services.StatisticsService, logger logging.Logger) StatisticsController {
	return StatisticsController{scanStatisticsService: scanStatisticsService, logger: logger}
}

// GetAggregateResult
// @Summary		Get aggregate bucket scan result
// @Tags		objects
// @Accept		json
// @Produce		json
// @Param		bucket	query	string	false	"Restrict result to specific bucket"
// @Param		period	query	string	false	"Restrict result to single day or month"
// @Param		date	query	string	false	"Reference date with format YYYY-MM-DD"
// @Success		200 {object} adapterentities.ObjectScanResponse
// @Failure		400 {object} adapterentities.ObjectScanResponse
// @Failure		500 {object} adapterentities.ObjectScanResponse
// @Security	ApiKey
// @Router      /objects [get]
func (s *StatisticsController) GetAggregateResult(c *fiber.Ctx) error {
	var response adapterentities.ObjectScanResponse

	bucket := c.Query("bucket", services.NoBucketName)
	date := c.Query("date")
	period := c.Query("period")

	parsedDate, err := common.ParseDate(date, time.Now())
	if err != nil {
		response.Error = err.Error()
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	parsedPeriod, err := entities.ParsePeriod(period)
	if err != nil {
		response.Error = err.Error()
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	acceptType := c.GetReqHeaders()[fiber.HeaderAccept]
	switch acceptType {
	case string(MIMEApplicationJSON), "":
		result, err := s.scanStatisticsService.GetBucketsStatistics(bucket, parsedDate, parsedPeriod)
		if err != nil {
			response.Error = err.Error()
			return c.Status(fiber.StatusInternalServerError).JSON(response)
		}
		response.Result = adapterentities.MapResultToScanResponse(result)

		return c.Status(fiber.StatusOK).JSON(response)
	case string(MIMEApplicationSMS):
		s.scanStatisticsService.Show(MIMEApplicationSMS, bucket, parsedDate, parsedPeriod)
		return c.Status(fiber.StatusOK).JSON(response)
	case string(MIMEApplicationSlack):
		s.scanStatisticsService.Show(MIMEApplicationSlack, bucket, parsedDate, parsedPeriod)
		return c.Status(fiber.StatusOK).JSON(response)
	default:
		response.Error = errUnsupportedAcceptType
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}
}

// GetObjectResult
// @Summary		Get individual object scan result
// @Tags		objects
// @Accept		json
// @Produce		json
// @Param		id	path	string	true	"Scan id returned by the object schedule endpoint"
// @Success		102 {object} adapterentities.ObjectScanResponse
// @Success		200 {object} adapterentities.ObjectScanResponse
// @Failure		400 {object} adapterentities.ObjectScanResponse
// @Failure		404 {object} adapterentities.ObjectScanResponse
// @Failure		500 {object} adapterentities.ObjectScanResponse
// @Security	ApiKey
// @Router      /objects/{id} [get]
func (s *StatisticsController) GetObjectResult(c *fiber.Ctx) error {
	return s.getIndividualResult(c)
}

// GetFileResult
// @Summary		Get individual file scan result
// @Tags		files
// @Accept		json
// @Produce		json
// @Param		id	path	string	true	"Scan id returned by the file schedule endpoint"
// @Success		102 {object} adapterentities.ObjectScanResponse
// @Success		200 {object} adapterentities.ObjectScanResponse
// @Failure		400 {object} adapterentities.ObjectScanResponse
// @Failure		404 {object} adapterentities.ObjectScanResponse
// @Failure		500 {object} adapterentities.ObjectScanResponse
// @Security	ApiKey
// @Router      /files/{id} [get]
func (s *StatisticsController) GetFileResult(c *fiber.Ctx) error {
	return s.getIndividualResult(c)
}

func (s *StatisticsController) getIndividualResult(c *fiber.Ctx) error {
	var response adapterentities.ObjectScanResponse

	scanID := c.Params("id")
	if !common.IsValidUUID(scanID) {
		response.Error = errInvalidScanID
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	result, err := s.scanStatisticsService.GetScanResult(scanID)

	switch {
	case errors.Is(err, services.ErrScanFailed):
		response.Error = errScanFailed
		return c.Status(fiber.StatusInternalServerError).JSON(response)

	case errors.Is(err, services.ErrScanIDNotFound):
		response.Error = errScanIDNotFound
		return c.Status(fiber.StatusNotFound).JSON(response)

	case errors.Is(err, services.ErrScanInProgress):
		response.Error = errScanInProgress
		return c.Status(fiber.StatusProcessing).JSON(response)

	case errors.Is(err, services.ErrScanIsWaiting):
		response.Error = errScanIsWaiting
		return c.Status(fiber.StatusProcessing).JSON(response)

	case errors.Is(err, services.ErrUnknownScanState):
		response.Error = errUnknownScanState
		return c.Status(fiber.StatusInternalServerError).JSON(response)
	}

	response.Result = adapterentities.MapResultToScanResponse(map[string]entities.ScanResult{scanID: result})

	return c.Status(fiber.StatusOK).JSON(response)
}
