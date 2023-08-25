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

package stages

import (
	"context"
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"eagle-eye/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStageProcess(t *testing.T) {
	t.Run("handler executed for each input", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		ctx := context.Background()

		requests := []entities.ScanRequest{{
			Key:    []string{"file"},
			Bucket: "bucket",
		},
			{
				Key:    []string{"file2"},
				Bucket: "bucket",
			},
		}

		handler := mocks.NewSpyHandler()
		inputChannel := make(chan *entities.ScanRequest, len(requests))
		cleanupChannel := make(chan *Cleanup[entities.ScanRequest])
		stage := NewStage[entities.ScanRequest, entities.ScanRequest](handler, inputChannel, cleanupChannel, logging.NewDiscardLog())

		for _, req := range requests {
			req := req
			entities.NewOutputWriter(make(chan *entities.ScanRequest))
			inputChannel <- &req
		}

		stage.Process(ctx)

		require.Eventually(t, func() bool { return handler.Counter["Name"] == 1 }, 5*time.Second, time.Second)
		require.Eventually(t, func() bool { return handler.Counter["Handle"] == 2 }, 5*time.Second, time.Second)
	})

	t.Run("stage stops when context is canceled", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)

		handler := mocks.NewSpyHandler()
		inputChannel := make(chan *entities.ScanRequest, 1)
		cleanupChannel := make(chan *Cleanup[entities.ScanRequest])
		stage := NewStage[entities.ScanRequest, entities.ScanRequest](handler, inputChannel, cleanupChannel, logging.NewDiscardLog())

		requests := []entities.ScanRequest{{
			Key:    []string{"file"},
			Bucket: "bucket",
		}}

		cancel()
		stage.Process(ctx)
		time.Sleep(time.Second)
		inputChannel <- &requests[0]

		require.Equal(t, handler.Counter["Name"], 1)
		require.Equal(t, handler.Counter["Handle"], 0)
	})
}
