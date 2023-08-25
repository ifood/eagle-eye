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

package out

import (
	"eagle-eye/domain/entities"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/slack-go/slack"
	"os"
)

type SlackViewer struct {
	token     string
	webhook   string
	channelID string
}

func NewSlackViewer(token, webhook, channelID string) *SlackViewer {
	return &SlackViewer{token: token, webhook: webhook, channelID: channelID}
}

func (s *SlackViewer) Show(description string, results map[string]entities.ScanResult) error {
	textPath, err := s.generateText(results)
	if err != nil {
		return fmt.Errorf("cant send message to slack. %w", err)
	}

	description = fmt.Sprintf("%s\n%s", description, "Tip: Use full screen or download the file to get data formatted")
	_, err = s.sendFileToChannel(description, textPath)

	return err
}

func (s *SlackViewer) SendMessage(message string) error {
	msg := slack.WebhookMessage{
		Username: "tester",
		Channel:  s.channelID,
		Text:     message,
	}

	return slack.PostWebhook(s.webhook, &msg)
}

func (s *SlackViewer) generateText(results map[string]entities.ScanResult) (string, error) {
	tmpPath := fmt.Sprintf("/tmp/%s.txt", uuid.New())

	file, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary result file. %w", err)
	}
	defer file.Close()

	// Header
	_, err = file.WriteString(fmt.Sprintf("%-64s %-15s %-10s %-10s %-15s %-80s %-15s\n",
		"Bucket", "Scanned", "Matches", "Errors", "Bypassed", "Entropy", "Requests"))
	if err != nil {
		return "", fmt.Errorf("failed to write header to result file. %w", err)
	}

	const LineFormat = "%-64s %-15d %-10d %-10d %-15d %-80s %-15d\n"

	// Lines
	for key, value := range results {
		entropy, _ := json.Marshal(value.Entropy)
		_, err = file.WriteString(fmt.Sprintf(LineFormat, key, value.Scanned, value.Matches, value.Errors, value.Bypassed, string(entropy), value.Requests))

		if err != nil {
			return "", fmt.Errorf("failed to write data to result file. %w", err)
		}
	}

	return tmpPath, nil
}

func (s *SlackViewer) sendFileToChannel(bannerMsg, filepath string) (string, error) {
	// Upload image to slack channel
	api := slack.New(s.token)
	parameters := slack.FileUploadParameters{
		File:           filepath,
		Channels:       []string{s.channelID},
		Title:          "Handle results",
		InitialComment: bannerMsg,
	}

	file, err := api.UploadFile(parameters)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to the channel. %s", err)
	}

	var threadID string
	if len(file.Shares.Private) != 0 {
		threadID = file.Shares.Private[s.channelID][0].Ts
	} else if len(file.Shares.Public) != 0 {
		threadID = file.Shares.Public[s.channelID][0].Ts
	}
	return threadID, nil
}
