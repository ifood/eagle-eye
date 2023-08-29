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

package awsutils

import (
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	maxIdleConnections     = 100
	idleConnectionsTimeout = 90
	maxIdleConnsPerHost    = 50
	maxConnsPerHost        = 100
)

type Clients struct {
	session *session.Session
}

func (c Clients) Session(region, endpoint string) (*session.Session, error) {
	if c.session != nil {
		return c.session, nil
	}

	transport := http.Transport{
		MaxIdleConns:        maxIdleConnections,
		IdleConnTimeout:     idleConnectionsTimeout * time.Second,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		MaxConnsPerHost:     maxConnsPerHost,
	}

	config := *aws.NewConfig().WithRegion(region).WithS3ForcePathStyle(true).WithHTTPClient(&http.Client{
		Transport: &transport}).WithDisableRestProtocolURICleaning(true)

	if endpoint != "" {
		resolverFn := func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
			return endpoints.ResolvedEndpoint{
				PartitionID:   "aws",
				URL:           endpoint,
				SigningRegion: region,
			}, nil
		}
		config.WithEndpointResolver(endpoints.ResolverFunc(resolverFn))
	}

	var err error
	c.session, err = session.NewSessionWithOptions(session.Options{
		Config:            config,
		SharedConfigState: session.SharedConfigEnable,
	})

	return c.session, err
}
