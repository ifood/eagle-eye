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
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/pkg/awsutils"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"strings"
)

type SMSViewer struct {
	phones  []string
	session *session.Session
}

func NewSMSViewer(awsSession *session.Session, phones []string) *SMSViewer {
	return &SMSViewer{session: awsSession, phones: phones}
}

func (s *SMSViewer) Show(description string, results map[string]entities.ScanResult) error {
	aggregatedResult := s.aggregateScanResults(results)
	message := s.generateMessage(description, len(results), aggregatedResult)
	errors := s.sendSMS(message)

	return fmt.Errorf("%s", strings.Join(errors, "\n"))
}

func (s *SMSViewer) SendMessage(message string) error {
	errors := s.sendSMS(message)

	return fmt.Errorf("%s", strings.Join(errors, "\n"))
}

func (s *SMSViewer) aggregateScanResults(results map[string]entities.ScanResult) entities.ScanResult {
	aggStats := entities.ScanResult{}
	for _, res := range results {
		aggStats = entities.MergeScanResults(aggStats, res)
	}

	return aggStats
}

func (s *SMSViewer) generateMessage(description string, totalBuckets int, aggStats entities.ScanResult) string {
	return fmt.Sprintf(
		"%s:\n"+
			"- %s buckets\n"+
			"- %s matches\n"+
			"- %s errors\n"+
			"- %s bypassed\n"+
			"- %s requests\n",
		description,
		common.ConvertNumberToHumanReadable(totalBuckets),
		common.ConvertNumberToHumanReadable(aggStats.Matches),
		common.ConvertNumberToHumanReadable(aggStats.Errors),
		common.ConvertNumberToHumanReadable(aggStats.Bypassed),
		common.ConvertNumberToHumanReadable(aggStats.Requests))
}

func (s *SMSViewer) sendSMS(message string) []string {
	var errorsList []string

	for _, phone := range s.phones {
		err := awsutils.SendSMS(s.session, nil, phone, message)
		if err != nil {
			errorsList = append(errorsList, err.Error())
		}
	}

	return errorsList
}
