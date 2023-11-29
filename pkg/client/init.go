// SPDX-License-Identifier: MIT

package client

import (
	"context"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type InitClient interface {
	Init(ctx context.Context, garmParams GarmScopeParams) error
}

type initClient struct{}

func NewInitClient() InitClient {
	return &initClient{}
}

func (s *initClient) Init(ctx context.Context, garmParams GarmScopeParams) error {
	metrics.TotalGarmCalls.WithLabelValues("Init").Inc()
	err := initializeGarm(ctx, garmParams)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("Init").Inc()
		return err
	}

	return nil
}
