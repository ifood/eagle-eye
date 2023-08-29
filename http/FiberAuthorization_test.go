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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAuthorizationKeysParser(t *testing.T) {
	tests := []struct {
		name string
		keys []string
	}{
		{
			name: "valid keys",
			keys: []string{"alias1:32141506e8178f0a3675cff255acf6a5a83adac7b33a9d7a3a37574e6a90927c", "alias2:f23e662bcffacd71f2dd6899430a5f264a5c403eeb23595f261aa075627b257c"},
		},
		{
			name: "no keys",
			keys: []string{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			parsedKeys, err := PrepareAuthorizationKeys(tt.keys)
			assert.Equal(t, len(tt.keys), len(parsedKeys))
			assert.NoError(t, err)
		})
	}
}

func TestInvalidAuthorizationKeys(t *testing.T) {
	tests := []struct {
		name string
		keys []string
	}{
		{
			name: "empty",
			keys: []string{""},
		},
		{
			name: "invalid secret size",
			keys: []string{"alias:cafe"},
		},
		{
			name: "invalid characters",
			keys: []string{"alias:32141506e8178f0a3675cff255acf6a5a83adac7b33a9d7a3a37574e6a9092!@"},
		},
		{
			name: "no alias",
			keys: []string{"32141506e8178f0a3675cff255acf6a5a83adac7b33a9d7a3a37574e6a90927c"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			parsedKeys, err := PrepareAuthorizationKeys(tt.keys)
			assert.Nil(t, parsedKeys)
			assert.Error(t, err)
		})
	}
}
