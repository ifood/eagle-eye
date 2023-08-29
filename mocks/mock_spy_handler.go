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

package mocks

import (
	"context"
	"eagle-eye/domain/entities"
)

// Coding because gomock still does not support generics properly. Even a derived interface embedding the generic one didn't work.
type SpyHandler struct {
	Counter map[string]int
}

func NewSpyHandler() *SpyHandler {
	return &SpyHandler{Counter: make(map[string]int)}
}

func (m *SpyHandler) Handle(ctx context.Context, request *entities.ScanRequest, w *entities.OutputWriter[entities.ScanRequest]) error {
	m.Counter["Handle"] += 1
	return nil
}

func (m *SpyHandler) Name() string {
	m.Counter["Name"] += 1
	return "SpyHandler"
}
