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

package main

import (
	"context"
	"eagle-eye/app"
	"log"

	// Docs for swagger
	_ "eagle-eye/docs"
	"os"
)

// @title Scanner service
// @version 1.0
// @description Scanner service is able to scan files as they arrive at your cloud bob storage
// @termsOfService http://swagger.io/terms/
// @contact.name Security Engineering
// @contact.email security-engineering@ifood.com.br
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /v1/
// @securitydefinitions.apikey ApiKey
// @in						   header
// @name					   Authorization
// @description				   Only needed if server was started with enforced authorization. Type \'Bearer\' and then your apikey.

func main() {
	err := app.Start(context.Background())
	if err != nil {
		log.Printf("EagleEye being stopped. Err: %s", err)
		os.Exit(1)
	}
}
