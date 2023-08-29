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
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/keyauth/v2"
	"strings"
)

func PrepareAuthorizationKeys(authorizationKeys []string) (map[string][]byte, error) {
	keys := make(map[string][]byte)

	for index, access := range authorizationKeys {
		const accessKeyParts = 2
		access := strings.Split(access, ":")

		if len(access) != accessKeyParts {
			return nil, fmt.Errorf("failed to parse access credentials at index %d. "+
				"Credentials should have format <alias>:<secret>, where secret must be prehashed with SHA256"+
				"Recommended method is to generate a secret with openssl, like `openssl rand -hex 32`, then hash it with sha256sum", index)
		}

		const SHA256ExpectedOutputSize = 64
		if len(access[1]) != SHA256ExpectedOutputSize {
			return nil, fmt.Errorf("failed to parse access credentials at index %d. "+
				"Credentials should have format <alias>:<secret>, where secret must be prehashed with SHA256"+
				"Recommended method is to generate a secret with openssl, like `openssl rand -hex 32`, then hash it with sha256sum", index)
		}

		decodedValue, err := hex.DecodeString(access[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse access credentials at index %d. "+
				"Credentials should have format <alias>:<secret>, where secret must be prehashed with SHA256"+
				"Recommended method is to generate a secret with openssl, like `openssl rand -hex 32`, then hash it with sha256sum", index)
		}

		keys[access[0]] = decodedValue
	}

	return keys, nil
}

func FiberAuthFilter(ctx *fiber.Ctx) bool {
	return !strings.HasPrefix(ctx.OriginalURL(), currentVersion) &&
		!strings.HasPrefix(ctx.OriginalURL(), debugPath)
}

func FiberAuthValidator(authorizationKeys map[string][]byte) func(c *fiber.Ctx, key string) (bool, error) {
	return func(c *fiber.Ctx, key string) (bool, error) {
		const equalContents = 1

		hashedKey := sha256.Sum256([]byte(key))

		for user, key := range authorizationKeys {
			if subtle.ConstantTimeCompare(hashedKey[:], key) == equalContents {
				c.Locals("user", user)
				return true, nil
			}
		}

		return false, keyauth.ErrMissingOrMalformedAPIKey
	}
}
