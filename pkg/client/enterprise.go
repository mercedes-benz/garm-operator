// SPDX-License-Identifier: MIT

package client

import (
	"github.com/cloudbase/garm/client"
	"github.com/cloudbase/garm/client/enterprises"
	"github.com/go-openapi/runtime"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type EnterpriseClient interface {
	ListEnterprises(param *enterprises.ListEnterprisesParams) (*enterprises.ListEnterprisesOK, error)
	CreateEnterprise(param *enterprises.CreateEnterpriseParams) (*enterprises.CreateEnterpriseOK, error)
	GetEnterprise(param *enterprises.GetEnterpriseParams) (*enterprises.GetEnterpriseOK, error)
	UpdateEnterprise(param *enterprises.UpdateEnterpriseParams) (*enterprises.UpdateEnterpriseOK, error)
	DeleteEnterprise(param *enterprises.DeleteEnterpriseParams) error
}

type enterpriseClient struct {
	client *client.GarmAPI
	token  runtime.ClientAuthInfoWriter
}

func NewEnterpriseClient(garmParams GarmScopeParams) (EnterpriseClient, error) {
	garmClient, token, err := newGarmClient(garmParams)
	if err != nil {
		return nil, err
	}

	return &enterpriseClient{garmClient, token}, nil
}

func (s *enterpriseClient) ListEnterprises(param *enterprises.ListEnterprisesParams) (*enterprises.ListEnterprisesOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("enterprises.List").Inc()
	enterprises, err := s.client.Enterprises.ListEnterprises(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("enterprises.List").Inc()
		return nil, err
	}

	return enterprises, nil
}

func (s *enterpriseClient) CreateEnterprise(param *enterprises.CreateEnterpriseParams) (*enterprises.CreateEnterpriseOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("enterprises.Create").Inc()
	enterprise, err := s.client.Enterprises.CreateEnterprise(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("enterprises.Create").Inc()
		return nil, err
	}

	return enterprise, nil
}

func (s *enterpriseClient) GetEnterprise(param *enterprises.GetEnterpriseParams) (*enterprises.GetEnterpriseOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("enterprises.Get").Inc()
	enterprise, err := s.client.Enterprises.GetEnterprise(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("enterprises.Get").Inc()
		return nil, err
	}
	return enterprise, nil
}

func (s *enterpriseClient) DeleteEnterprise(param *enterprises.DeleteEnterpriseParams) error {
	metrics.TotalGarmCalls.WithLabelValues("enterprises.Delete").Inc()
	err := s.client.Enterprises.DeleteEnterprise(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("enterprises.Delete").Inc()
		return err
	}

	return nil
}

func (s *enterpriseClient) UpdateEnterprise(param *enterprises.UpdateEnterpriseParams) (*enterprises.UpdateEnterpriseOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("enterprises.Update").Inc()
	enterprise, err := s.client.Enterprises.UpdateEnterprise(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("enterprises.Update").Inc()
		return nil, err
	}

	return enterprise, nil
}
