// SPDX-License-Identifier: MIT

package client

import (
	"github.com/cloudbase/garm/client/enterprises"
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
	BaseClient
}

func NewEnterpriseClient() EnterpriseClient {
	return &enterpriseClient{
		Instance,
	}
}

func (s *enterpriseClient) ListEnterprises(param *enterprises.ListEnterprisesParams) (*enterprises.ListEnterprisesOK, error) {
	return EnsureAuth(func() (*enterprises.ListEnterprisesOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("enterprises.List").Inc()
		enterprises, err := s.Client().Enterprises.ListEnterprises(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("enterprises.List").Inc()
			return nil, err
		}
		return enterprises, nil
	})
}

func (s *enterpriseClient) CreateEnterprise(param *enterprises.CreateEnterpriseParams) (*enterprises.CreateEnterpriseOK, error) {
	return EnsureAuth(func() (*enterprises.CreateEnterpriseOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("enterprises.Create").Inc()
		enterprise, err := s.Client().Enterprises.CreateEnterprise(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("enterprises.Create").Inc()
			return nil, err
		}
		return enterprise, nil
	})
}

func (s *enterpriseClient) GetEnterprise(param *enterprises.GetEnterpriseParams) (*enterprises.GetEnterpriseOK, error) {
	return EnsureAuth(func() (*enterprises.GetEnterpriseOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("enterprises.Get").Inc()
		enterprise, err := s.Client().Enterprises.GetEnterprise(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("enterprises.Get").Inc()
			return nil, err
		}
		return enterprise, nil
	})
}

func (s *enterpriseClient) DeleteEnterprise(param *enterprises.DeleteEnterpriseParams) error {
	_, err := EnsureAuth(func() (interface{}, error) {
		metrics.TotalGarmCalls.WithLabelValues("enterprises.Delete").Inc()
		err := s.Client().Enterprises.DeleteEnterprise(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("enterprises.Delete").Inc()
			return nil, err
		}

		return nil, nil
	})
	return err
}

func (s *enterpriseClient) UpdateEnterprise(param *enterprises.UpdateEnterpriseParams) (*enterprises.UpdateEnterpriseOK, error) {
	return EnsureAuth(func() (*enterprises.UpdateEnterpriseOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("enterprises.Update").Inc()
		enterprise, err := s.Client().Enterprises.UpdateEnterprise(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("enterprises.Update").Inc()
			return nil, err
		}
		return enterprise, nil
	})
}
