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

package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestLoadNotificationConfig(t *testing.T) {
	os.Setenv("NOTIFICATION_SLACK_WEBHOOK", "https://hooks.slack.com/services/T03XXXXXX/A02AA5AAAA4/invalid")
	os.Setenv("NOTIFICATION_SLACK_APPTOKEN", "xoxb-token")
	os.Setenv("NOTIFICATION_PHONES", "+55111111111111,+55222222222222")
	os.Setenv("REDIS_PASSWORD", "password")
	os.Setenv("SCANNER_CIPHERPASS", "cipherpass")
	os.Setenv("SCANNER_VIRUSTOTAL_APIKEY", "vtkey")
	os.Setenv("HTTPSERVER_AUTHORIZATIONKEYS", "alias1:key1,alias2:key2")

	cfg, err := LoadConfig()

	assert.NoError(t, err)
	assert.Equal(t, generateSampleConfig(), cfg)
}

func generateSampleConfig() AppConfig {
	config := AppConfig{
		HTTPServer: HTTPServer{
			AuthorizationKeys: []string{"alias1:key1", "alias2:key2"},
			Port:              3000,
			Profiler:          false,
			Swagger:           false,
			Metrics:           true,
			MaxRequestSize:    52428800,
		},
		Aws: AWS{
			Queue:    "https://sqs.us-east-1.amazonaws.com/000000000100/sqs-queue",
			Region:   "us-east-1",
			Resolver: "test",
		},
		Scanner: Scanner{
			InternalBucket:    "scanner-internal-bucket",
			ScanProbabilities: map[string]float64{"samples-scanner-sandbox": 1.0, "bucket.com.br": 0.30},
			Cipherpass:        "cipherpass",
			Allowlist: map[string][]string{
				"samples-scanner-sandbox": {"/notscan", "/notscan2"},
			},
			Virustotal: VirusTotal{
				APIkey:    "vtkey",
				MaxRPM:    4,
				Threshold: 10.0,
			},
			Yara: Yara{
				Rulesdir: "/app/data/rules",
			},
			DebugLog:       false,
			SizeLimit:      10737418240,
			MaxStorageSize: 21474836480,
		},
		Redis: Redis{
			URL:      "master.app-name.xxx1xx.use1.cache.amazonaws.com:6379",
			Password: "password",
			UseTLS:   true,
		},
		Notification: Notification{
			UpdateInterval: 10,
			Slack: Slack{
				ChannelID: "XXXXXXXXX",
				AppToken:  "xoxb-token",
				Webhook:   "https://hooks.slack.com/services/T03XXXXXX/A02AA5AAAA4/invalid",
			},
			Phones: []string{"+55111111111111", "+55222222222222"},
		},
	}

	return config
}
