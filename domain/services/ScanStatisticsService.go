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

package services

import (
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/logging"
	"errors"
	"fmt"
	"reflect"
	"time"
)

const (
	NoBucketName string = ""
)

var (
	ErrScanIDNotFound   = errors.New("scanID not found")
	ErrScanFailed       = errors.New("scan failed")
	ErrScanInProgress   = errors.New("scan not completed yet")
	ErrScanIsWaiting    = errors.New("scan is waiting in the queue")
	ErrUnknownScanState = errors.New("scan has unknown state")
)

//go:generate go run -mod=mod github.com/golang/mock/mockgen -destination=../../mocks/mock_statistics_service.go -package=mocks -source=ScanStatisticsService.go
type StatisticsService interface {
	Show(mimetype entities.ViewerMimetype, bucketName string, date time.Time, period entities.Period)
	GetBucketsStatistics(bucketName string, date time.Time, period entities.Period) (map[string]entities.ScanResult, error)
	GetScanResult(scanID string) (entities.ScanResult, error)
}

type ScanStatisticsService struct {
	aggregateRepository  out.AggregateScanRepository
	individualRepository out.IndividualScanRepository
	scheduleService      Scheduler
	viewers              map[entities.ViewerMimetype]out.Viewer
	logger               logging.Logger
}

func NewScanStatisticsService(aggregateRepository out.AggregateScanRepository, individualRepository out.IndividualScanRepository, scheduleService Scheduler, viewers map[entities.ViewerMimetype]out.Viewer, logger logging.Logger) *ScanStatisticsService {
	return &ScanStatisticsService{aggregateRepository: aggregateRepository, individualRepository: individualRepository, scheduleService: scheduleService, viewers: viewers, logger: logger}
}

func (s *ScanStatisticsService) GetScanResult(scanID string) (entities.ScanResult, error) {
	item, err := s.scheduleService.GetSchedule(scanID)
	if err != nil {
		return entities.ScanResult{}, ErrScanIDNotFound // NotFound -> 400
	}

	switch item.Status {
	case entities.Waiting:
		return entities.ScanResult{}, ErrScanIsWaiting
	case entities.Running:
		return entities.ScanResult{}, ErrScanInProgress
	case entities.Completed:
		res, err := s.individualRepository.Get(scanID)
		res.Bucket = "no bucket specified"

		return res, err
	case entities.Error:
		return entities.ScanResult{}, ErrScanFailed
	default:
		return entities.ScanResult{}, ErrUnknownScanState
	}
}

func (s *ScanStatisticsService) Show(mimetype entities.ViewerMimetype, bucket string, date time.Time, period entities.Period) {
	description, err := s.generateDescription(date, period)
	if err != nil {
		s.logger.Errorw("Failed to generate description for external service", "error", err)
		return
	}

	results, err := s.GetBucketsStatistics(bucket, date, period)
	if err != nil {
		s.logger.Errorw("Failed to get bucket statistics", "error", err)
		return
	}

	if _, ok := s.viewers[mimetype]; !ok {
		s.logger.Errorw("Could not find viewer for mimetype", "error", err, "mimetype", mimetype)
		return
	}

	err = s.viewers[mimetype].Show(description, results)
	if err != nil {
		s.logger.Errorw("Failed to send message through viewer", "error", err, "viewer", reflect.ValueOf(s.viewers[mimetype]).Type())
	}
}

func (s *ScanStatisticsService) GetBucketsStatistics(bucketName string, date time.Time, period entities.Period) (map[string]entities.ScanResult, error) {
	results, err := s.prepareStatistics(date, period)
	if err != nil {
		s.logger.Errorw("Failed to consolidate statistics", "error", err)
		return map[string]entities.ScanResult{}, err
	}

	if bucketName == NoBucketName {
		return results, nil
	}

	if val, ok := results[bucketName]; ok {
		return map[string]entities.ScanResult{bucketName: val}, nil
	}

	return map[string]entities.ScanResult{}, fmt.Errorf("not found results for bucket %s on period %s", bucketName, period)
}

func (s *ScanStatisticsService) generateDescription(date time.Time, period entities.Period) (string, error) {
	switch period {
	case entities.Day:
		return fmt.Sprintf("Handle results %s", date.Format("02-01-2006")), nil
	case entities.Month:
		return fmt.Sprintf("Handle results %s", date.Format("Jan 2006")), nil
	default:
		return "", fmt.Errorf("invalid period")
	}
}

func (s *ScanStatisticsService) prepareStatistics(date time.Time, period entities.Period) (map[string]entities.ScanResult, error) {
	var results map[string]entities.ScanResult
	var err error

	switch period {
	case entities.Day:
		results, err = s.aggregateRepository.GetByDate(date.Day(), int(date.Month()))
	case entities.Month:
		results, err = s.aggregateRepository.GetByMonth(int(date.Month()))
	default:
		results = nil
		err = fmt.Errorf("invalid period")
	}

	return results, err
}
