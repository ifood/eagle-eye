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

type ScanFlag uint8

const (
	DisableVirusTotal = 1 << iota // Useful for malware eval, because we don't want to burn our possible queries
)

type ResultType string

const (
	// Aggregate Multiple scan requests results gets aggregated to the same key. Useful for buckets
	// for which you want to present a aggregated result. Tracked daily and monthly
	Aggregate ResultType = "Aggregate"

	// Individual Each scan gets saved to a different "row", not possible to get aggregated results daily or
	// monthly
	Individual ResultType = "Individual"
)

type ScanRequest struct {
	Key         []string // Key in the bucket that should be scanned
	Size        uint64   // File size in the bucket
	Bucket      string   // Bucket name
	StorageID   string   // Storage id refers to the local storage abstraction
	StorageType string   // Currently, only supports S3
	MessageID   string   // MessageID from SQS, used to delete the message after processing
	Flags       ScanFlag
	ResultType  ResultType
	ScanID      string
}
