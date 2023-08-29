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

package scan

import (
	"context"
	"eagle-eye/domain/entities"
)

type SyncProcess interface {
	Scan(ctx context.Context, scanContext scanContext) (entities.ScanResult, error)
}

type AsyncProcess interface {
	ScheduleScan(ctx context.Context, scanContext scanContext) (entities.ScanResult, error)
	GetResults(ctx context.Context) []entities.ScanResult
}
