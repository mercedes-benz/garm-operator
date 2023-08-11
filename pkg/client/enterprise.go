package client

import (
	"github.com/cloudbase/garm/client"
	"github.com/cloudbase/garm/client/enterprises"
	"github.com/go-openapi/runtime"
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
	// TODO: track requests by introducing metrics
	enterprises, err := s.client.Enterprises.ListEnterprises(param, s.token)
	if err != nil {
		return nil, err
	}

	return enterprises, nil
}

func (s *enterpriseClient) CreateEnterprise(param *enterprises.CreateEnterpriseParams) (*enterprises.CreateEnterpriseOK, error) {
	enterprise, err := s.client.Enterprises.CreateEnterprise(param, s.token)
	if err != nil {
		return nil, err
	}

	return enterprise, nil
}

func (s *enterpriseClient) GetEnterprise(param *enterprises.GetEnterpriseParams) (*enterprises.GetEnterpriseOK, error) {
	enterprise, err := s.client.Enterprises.GetEnterprise(param, s.token)
	if err != nil {
		return nil, err
	}
	return enterprise, nil
}

func (s *enterpriseClient) DeleteEnterprise(param *enterprises.DeleteEnterpriseParams) error {
	err := s.client.Enterprises.DeleteEnterprise(param, s.token)
	if err != nil {
		return err
	}

	return nil
}

func (s *enterpriseClient) UpdateEnterprise(param *enterprises.UpdateEnterpriseParams) (*enterprises.UpdateEnterpriseOK, error) {
	enterprise, err := s.client.Enterprises.UpdateEnterprise(param, s.token)
	if err != nil {
		return nil, err
	}

	return enterprise, nil
}
