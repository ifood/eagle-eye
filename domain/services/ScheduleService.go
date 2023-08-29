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
	"encoding/json"
	"fmt"
	"github.com/eikenb/pipeat"
	"github.com/google/uuid"
	"io"
	"strings"
	"time"
)

const keyFormat = "schedule-%s"

//go:generate go run -mod=mod github.com/golang/mock/mockgen -destination=../../mocks/mock_scheduler_service.go -package=mocks -source=ScheduleService.go
type Scheduler interface {
	GetSchedule(objectKeyOrScanID string) (entities.ScheduleItemWithState, error)
	IsScheduledScan(bucket string) bool
	ScheduleObject(bucket, key string) (string, error)
	Schedule(filename string, reader io.Reader) (string, error)
	UpdateSchedule(objectKeyOrScanID string, newStatus entities.Status) error
}

type ScheduleService struct {
	bucketName           string
	remoteStorageFactory out.RemoteStorageFactory
	cache                out.Cache
}

func NewScheduleService(remoteStorageFactory out.RemoteStorageFactory, cache out.Cache, bucketName string) *ScheduleService {
	s := ScheduleService{bucketName: bucketName, remoteStorageFactory: remoteStorageFactory, cache: cache}
	return &s
}

func (s *ScheduleService) IsScheduledScan(bucket string) bool {
	return bucket == s.bucketName
}

func (s *ScheduleService) ScheduleObject(bucket, key string) (string, error) {
	storage, err := s.remoteStorageFactory.GetRemoteStorage("s3")
	if err != nil {
		return "", fmt.Errorf("access to service repository failed")
	}

	reader, writer, err := pipeat.Pipe()
	if err != nil {
		return "", fmt.Errorf("failed to create internal pipe")
	}

	if err := storage.Get(bucket, key, writer); err != nil {
		return "", fmt.Errorf("failed to get file from bucket. bucket: %s, key: %s, error: %w", bucket, key, err)
	}

	return s.Schedule(key, reader)
}

func (s *ScheduleService) Schedule(filename string, reader io.Reader) (string, error) {
	if err := s.saveFileToInternalRepository(filename, reader); err != nil {
		return "", err
	}

	requestUUID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("could not generate uuid. err: %w", err)
	}

	scanID := requestUUID.String()
	item := entities.ScheduleItemWithState{
		ScanID:     scanID,
		CreateTime: time.Now(),
		LastUpdate: time.Now(),
		Status:     entities.Waiting,
		Bucket:     s.bucketName,
		Key:        filename,
	}

	if err := s.save(item); err != nil {
		return "", err
	}

	return scanID, nil
}

func (s *ScheduleService) save(item entities.ScheduleItemWithState) error {
	if err := s.cache.Set(fmt.Sprintf(keyFormat, item.ScanID), item, -1); err != nil {
		return fmt.Errorf("failed to save data for scan id. err %w, scanid: %s", err, item.ScanID)
	}

	if err := s.cache.Set(fmt.Sprintf(keyFormat, item.Key), item, -1); err != nil {
		return fmt.Errorf("failed to save for key. err %w, key: %s", err, item.Key)
	}

	return nil
}

func (s *ScheduleService) UpdateSchedule(objectKeyOrScanID string, newStatus entities.Status) error {
	schedule, err := s.GetSchedule(objectKeyOrScanID)
	if err != nil {
		return fmt.Errorf("failed to update scanid to new status. status: %v, error: %w", newStatus, err)
	}

	schedule.Status = newStatus
	schedule.LastUpdate = time.Now()

	err = s.save(schedule)
	if err != nil {
		return fmt.Errorf("failed to update scanid to new status. status: %v, error: %w", newStatus, err)
	}

	return nil
}

func (s *ScheduleService) GetSchedule(objectKeyOrScanID string) (entities.ScheduleItemWithState, error) {
	value, err := s.cache.Get(fmt.Sprintf(keyFormat, objectKeyOrScanID))
	if err != nil {
		return entities.ScheduleItemWithState{}, fmt.Errorf("failed to get status for scan id. err: %w, scanid: %s", err, objectKeyOrScanID)
	}

	var item entities.ScheduleItemWithState
	if err := json.NewDecoder(strings.NewReader(value)).Decode(&item); err != nil {
		return entities.ScheduleItemWithState{}, fmt.Errorf("failed to decode schedule for scan id. err: %w, scanid: %s", err, objectKeyOrScanID)
	}

	if !s.shouldAbort(item) {
		return item, nil
	}

	item.LastUpdate = time.Now()
	item.Status = entities.Error

	return item, s.save(item)
}

func (s *ScheduleService) shouldAbort(item entities.ScheduleItemWithState) bool {
	return (item.Status == entities.Waiting || item.Status == entities.Running) && item.LastUpdate.Before(time.Now().Add(-time.Hour))
}

func (s *ScheduleService) saveFileToInternalRepository(key string, reader io.Reader) error {
	storage, err := s.remoteStorageFactory.GetRemoteStorage("s3")
	if err != nil {
		return fmt.Errorf("access to service repository failed")
	}

	err = storage.Put(s.bucketName, key, reader)
	if err != nil {
		return fmt.Errorf("failed to save file in the service repository")
	}

	return nil
}
