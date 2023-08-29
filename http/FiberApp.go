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

package http

import (
	"eagle-eye/logging"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/keyauth/v2"
	"github.com/gofiber/swagger"
	fibertrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gofiber/fiber.v2"
)

const (
	currentVersion = "/v1"
	debugPath      = "/debug"
	swaggerPath    = "/swagger"
)

func CreateFiberApp(fiberConfig FiberConfig, logger logging.Logger) (*fiber.App, error) {
	app := fiber.New(fiber.Config{
		BodyLimit: fiberConfig.MaxRequestSize,
		// Preventing possible security issues when interacting with the authentication filter
		CaseSensitive: true,
		UnescapePath:  false,
		StrictRouting: true,
	})

	// Add datadog tracer middleware
	app.Use(fibertrace.Middleware())

	if len(fiberConfig.AuthorizationKeys) != 0 {
		authorizationKeys, err := PrepareAuthorizationKeys(fiberConfig.AuthorizationKeys)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare keys. %w", err)
		}

		authMiddleware := keyauth.New(keyauth.Config{
			ErrorHandler: func(ctx *fiber.Ctx, err error) error {
				response := struct {
					Error string
				}{
					Error: err.Error(),
				}
				return ctx.Status(fiber.StatusUnauthorized).JSON(response)
			},
			SuccessHandler: func(ctx *fiber.Ctx) error {
				return ctx.Next()
			},
			Filter:    FiberAuthFilter,
			Validator: FiberAuthValidator(authorizationKeys),
		})
		app.Use(authMiddleware)
	} else {
		logger.Infow("No API keys specified, service may be abused by users with network access to its endpoints")
		logger.Infow("Please, consider defining the environment variable HTTPSERVER_AUTHORIZATIONKEYS in your secrets manager.")
	}

	app.Use(fiberConfig.RequestLogger)

	if fiberConfig.Swagger {
		logger.Infow("Swagger endpoint enabled. This is a security sensitive configuration, please keep it disabled unless required")
		logger.Infow("Please, consider enabling endpoint authentication")
		app.Get(fmt.Sprintf("%s/*", swaggerPath), swagger.HandlerDefault)
	}

	if fiberConfig.Profiler {
		logger.Infow("Go profiler is enabled. This is a security sensitive configuration, please keep it disabled unless required")
		logger.Infow("Eg. Request: curl -XGET http://<hostname>:<port>/debug/pprof/profile?seconds=30  --output profile")
		logger.Infow("Check https://pkg.go.dev/net/http/pprof for more examples")
		app.Use(pprof.New())
	}

	app.Get("/healthcheck/readiness", fiberConfig.Readiness)
	app.Get("/healthcheck/liveness", fiberConfig.Liveness)
	app.Get("/metrics", fiberConfig.Metrics)

	v1 := app.Group(currentVersion)
	for _, handler := range fiberConfig.Handlers {
		v1.Add(handler.HTTPMethod, handler.Path, handler.HandlerFunc)
	}

	return app, nil
}
