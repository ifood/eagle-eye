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
	"fmt"
)

type Cleanup[T any] struct {
	Request *T
	Error   error
}

type Stage[T, V any] struct {
	handler      entities.Handler[T, V]
	inputChannel <-chan *T
	logger       logging.Logger
	output       chan *V
	cleanup      chan *Cleanup[T]
}

func NewStage[T any, V any](handler entities.Handler[T, V], inputChannel chan *T, cleanupChannel chan *Cleanup[T], logger logging.Logger) Stage[T, V] {
	output := make(chan *V)

	return Stage[T, V]{
		handler:      handler,
		inputChannel: inputChannel,
		logger:       logger,
		output:       output,
		cleanup:      cleanupChannel,
	}
}

func (s *Stage[T, V]) Output() chan *V {
	return s.output
}

func (s *Stage[T, V]) Process(ctx context.Context) {
	s.logger.Infow("Start of stage")
	s.logger.Infow("Initializing handler", "handler", s.handler.Name())

	go s.doProcess(ctx)
}

func (s *Stage[T, V]) doProcess(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("End of stage")
			return
		case input := <-s.inputChannel:
			s.safeHandle(ctx, input)
		}
	}
}

func (s *Stage[T, V]) safeHandle(ctx context.Context, input *T) {
	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("%v", r)
			s.logger.Errorw("Panic catch during handler execution", "err", panicErr)
			s.cleanup <- &Cleanup[T]{Request: input, Error: panicErr}
		}
	}()

	writer := entities.NewOutputWriter[V](s.output)

	err := s.handler.Handle(ctx, input, writer)
	if err != nil {
		s.cleanup <- &Cleanup[T]{Request: input, Error: err}
		return
	}
}
