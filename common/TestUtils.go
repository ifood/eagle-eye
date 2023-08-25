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

package common

import (
	"bytes"
	"context"
	dmchttp "eagle-eye/http"
	"eagle-eye/logging"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path"
	"runtime"
	"testing"
)

const EnforceRequestToDisk = 10 * 1024 * 1024

func ChangePathForTesting(t *testing.T) {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("could not get caller")
	}

	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)

	if err != nil {
		panic(err)
	}
}

func LoadFileToStorage(t *testing.T, filename string, writer io.Writer) {
	ChangePathForTesting(t)
	baseDir := "resources/testfiles/"
	src, _ := os.ReadFile(baseDir + filename)

	_, err := io.Copy(writer, bytes.NewReader(src))
	if err != nil {
		panic(err)
	}
}

func LoadFile(t *testing.T, filename string) []byte {
	ChangePathForTesting(t)
	baseDir := "resources/testfiles/"
	src, _ := os.ReadFile(baseDir + filename)

	return src
}

func GetObjectFromJSON[T any](t *testing.T, data []byte) T {
	t.Helper()

	var objects T
	err := json.Unmarshal(data, &objects)

	if err != nil {
		panic(err)
	}

	return objects
}

func GetObjectsFromJSONFile[T any](t *testing.T, filename string) T {
	t.Helper()

	data := LoadFile(t, filename)

	return GetObjectFromJSON[T](t, data)
}

func GetObjectJSON(t *testing.T, data interface{}) string {
	t.Helper()

	jsonData, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return string(jsonData)
}

func RedirectContainerOutput(ctx context.Context, pool *dockertest.Pool, containerID string) {
	err := pool.Client.Logs(docker.LogsOptions{
		Context:      ctx,
		Container:    containerID,
		OutputStream: os.Stdout,
		Follow:       true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
		Timestamps:   true,
	})
	if err != nil {
		log.Println(err)
	}
}

func CreateFiberAppForTest(handlers []dmchttp.Handler) *fiber.App {
	fiberConfig := dmchttp.FiberConfig{
		MaxRequestSize: EnforceRequestToDisk,
		Profiler:       false,
		RequestLogger: func(c *fiber.Ctx) error {
			return c.Next()
		},
		Readiness: func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		},
		Liveness: func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		},
		Handlers: handlers,
	}
	app, err := dmchttp.CreateFiberApp(fiberConfig, logging.NewDiscardLog())

	if err != nil {
		panic(err)
	}

	return app
}

func PrepareRequestBody(t *testing.T, field string, data []byte) (body *bytes.Buffer, format string) {
	t.Helper()

	body = &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	defer writer.Close()

	part, err := writer.CreateFormFile(field, "fakename")
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(part, bytes.NewReader(data))
	if err != nil {
		panic(err)
	}

	return body, writer.FormDataContentType()
}
