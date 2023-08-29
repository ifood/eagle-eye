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

package metrics

import (
	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	"io"
	"net/http"
	"time"
)

const applicationName = "eagleeye_scanner"

func NewPrometheusScope() (tally.Scope, http.Handler, io.Closer) {
	reporter := prometheus.NewReporter(prometheus.Options{})
	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		Prefix:         applicationName,
		Separator:      prometheus.DefaultSeparator,
		Tags:           map[string]string{},
		CachedReporter: reporter},
		time.Second)

	return scope, reporter.HTTPHandler(), closer
}
