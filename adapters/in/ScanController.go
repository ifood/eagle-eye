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
	"eagle-eye/domain/services"
	"eagle-eye/logging"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ScanController struct {
	validate         *validator.Validate
	schedulerService services.Scheduler
	logger           logging.Logger
}

func NewScanController(schedulerService services.Scheduler, logger logging.Logger) ScanController {
	return ScanController{schedulerService: schedulerService, logger: logger, validate: validator.New()}
}

// ScanFile
// @Summary		Schedules file for scan
// @Tags		files
// @Accept		json
// @Produce		json
// @Param		file	formData	file	true	"File to be scanned"
// @Success		200 {object} adapterentities.ScheduleResponse
// @Failure		400 {object} adapterentities.ScheduleResponse
// @Failure		500 {object} adapterentities.ScheduleResponse
// @Security	ApiKey
// @Router      /files [post]
func (s *ScanController) ScanFile(c *fiber.Ctx) error {
	resp := adapterentities.ScheduleResponse{}

	file, err := c.FormFile("file")
	if err != nil {
		s.logger.Errorw("no file found", "error", err)
		resp.Error = "no file found"

		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	tempFile, err := file.Open()
	if err != nil {
		s.logger.Errorw("failed to open file", "error", err)
		resp.Error = "failed to open file"

		return c.Status(fiber.StatusInternalServerError).JSON(resp)
	}

	scanID, err := s.schedulerService.Schedule(file.Filename, tempFile)
	if err != nil {
		s.logger.Errorw("failed to schedule file for scanning", "error", err, "filename", file.Filename, "filesize", file.Size)
		resp.Error = "could not schedule file for scan"

		return c.Status(fiber.StatusInternalServerError).JSON(resp)
	}

	resp.ID = scanID

	return c.Status(fiber.StatusOK).JSON(resp)
}

// ScanObject
// @Tags		objects
// @Summary		Schedules object from bucket for scan
// @Accept		json
// @Produce		json
// @Param		request	body	adapterentities.RequestObjectScan	true	"Object to be scanned"
// @Success		200 {object} adapterentities.ScheduleResponse
// @Failure		400 {object} adapterentities.ScheduleResponse
// @Failure		500 {object} adapterentities.ScheduleResponse
// @Security	ApiKey
// @Router      /objects [post]
func (s *ScanController) ScanObject(c *fiber.Ctx) error {
	response := adapterentities.ScheduleResponse{}
	request := &adapterentities.RequestObjectScan{}
	err := c.BodyParser(request)

	if err != nil {
		s.logger.Errorw("Could not parse request", "error", err, "request", request)
		response.Error = err.Error()

		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	if err := s.validate.Struct(request); err != nil {
		s.logger.Errorw("Some field is missing", "error", err)
		response.Error = err.Error()

		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	scanID, err := s.schedulerService.ScheduleObject(request.Bucket, request.Key)
	if err != nil {
		s.logger.Errorw("failed to schedule file for scanning", "bucket", request.Bucket, "key", request.Key)
		response.Error = "could not schedule object for scan"

		return c.Status(fiber.StatusInternalServerError).JSON(response)
	}

	response.ID = scanID

	return c.Status(fiber.StatusOK).JSON(response)
}
