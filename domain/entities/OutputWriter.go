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

package entities

import "context"

type OutputWriter[T any] struct {
	ch chan *T
}

func NewOutputWriter[T any](ch chan *T) *OutputWriter[T] {
	return &OutputWriter[T]{ch}
}

func (w *OutputWriter[T]) Write(ctx context.Context, value *T) {
	select {
	case <-ctx.Done():
	case w.ch <- value:
	}
}
