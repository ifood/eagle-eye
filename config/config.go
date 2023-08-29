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
	"bytes"
	"fmt"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

const (
	defaultPort           = 3000
	defaultMaxRequestSize = 52428800
	defaultUpdateInterval = 60
)

type AppConfig struct {
	Aws          AWS
	Scanner      Scanner
	Redis        Redis
	Notification Notification
	HTTPServer   HTTPServer
}

type HTTPServer struct {
	AuthorizationKeys []string
	Profiler          bool
	Swagger           bool
	Metrics           bool
	MaxRequestSize    int
	Port              int
}

type AWS struct {
	Queue    string
	Region   string
	Resolver string
}

type Scanner struct {
	ScanProbabilities map[string]float64
	Cipherpass        string
	Allowlist         map[string][]string
	Virustotal        VirusTotal
	Yara              Yara
	SizeLimit         uint64
	MaxStorageSize    int64
	DebugLog          bool
	InternalBucket    string
}

type Yara struct {
	Rulesdir string
}

type VirusTotal struct {
	Threshold float64
	APIkey    string
	MaxRPM    int
}

type Redis struct {
	URL      string
	Password string
	UseTLS   bool
}

type Notification struct {
	UpdateInterval int
	Slack          Slack
	Phones         []string
}

type Slack struct {
	ChannelID string
	AppToken  string
	Webhook   string
}

func NewConfig() *AppConfig {
	return &AppConfig{
		Aws: AWS{
			Region: "us-east-1",
		},
		Notification: Notification{
			UpdateInterval: defaultUpdateInterval,
		},
		HTTPServer: HTTPServer{
			Port:           defaultPort,
			MaxRequestSize: defaultMaxRequestSize,
		},
	}
}

func validateConfig(config AppConfig) error {
	if config.Redis.URL == "" {
		return fmt.Errorf("no Redis URL specified")
	}

	if config.Aws.Region == "" {
		return fmt.Errorf("no AWS region specified")
	}

	return nil
}

// see supershal approach https://github.com/spf13/viper/issues/188
func LoadConfig() (AppConfig, error) {
	const keyDelimiter = "/"
	v := viper.NewWithOptions(viper.KeyDelimiter(keyDelimiter))

	// set default values in viper.
	// Viper needs to know if a key exists in order to override it.
	// https://github.com/spf13/viper/issues/188
	b, err := yaml.Marshal(NewConfig())
	if err != nil {
		return AppConfig{}, err
	}

	defaultConfig := bytes.NewReader(b)

	v.AddConfigPath(os.Getenv("CONFIG_DIR"))
	v.AddConfigPath("../resources/")
	v.AddConfigPath(".")
	v.AddConfigPath("/app/data/")
	v.AddConfigPath("/app/config/")
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if err := v.MergeConfig(defaultConfig); err != nil {
		return AppConfig{}, err
	}

	// If file not found, return error
	if err := v.MergeInConfig(); err != nil {
		return AppConfig{}, err
	}

	// tell viper to overwrite env variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(keyDelimiter, "_"))
	// refresh configuration with all merged values
	config := AppConfig{}
	err = v.Unmarshal(&config)

	if err != nil {
		return AppConfig{}, err
	}

	err = validateConfig(config)
	if err != nil {
		return AppConfig{}, err
	}

	return config, nil
}
