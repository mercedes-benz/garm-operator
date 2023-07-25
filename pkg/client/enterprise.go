package client

import (
	"github.com/cloudbase/garm/cmd/garm-cli/client"
	"github.com/cloudbase/garm/params"
)

type EnterpriseClient interface {
	ListEnterprises() ([]params.Enterprise, error)
	CreateEnterprise(param params.CreateEnterpriseParams) (params.Enterprise, error)
	GetEnterprise(ID string) (params.Enterprise, error)
	DeleteEnterprise(ID string) error
}

type enterpriseClient struct{ client *client.Client }

func NewEnterpriseClient(garmParams GarmScopeParams) (EnterpriseClient, error) {
	garmClient, err := newGarmClient(garmParams)
	if err != nil {
		return nil, err
	}

	return &enterpriseClient{garmClient}, nil
}

func (s *enterpriseClient) ListEnterprises() ([]params.Enterprise, error) {
	// TODO: track requests by introducing metrics
	enterprises, err := s.client.ListEnterprises()
	if err != nil {
		return nil, err
	}

	return enterprises, nil
}

func (s *enterpriseClient) CreateEnterprise(param params.CreateEnterpriseParams) (params.Enterprise, error) {
	enterprise, err := s.client.CreateEnterprise(param)
	if err != nil {
		return params.Enterprise{}, err
	}

	return enterprise, nil
}

func (s *enterpriseClient) GetEnterprise(ID string) (params.Enterprise, error) {
	enterprise, err := s.client.GetEnterprise(ID)
	if err != nil {
		return params.Enterprise{}, err
	}
	return enterprise, nil
}

func (s *enterpriseClient) DeleteEnterprise(ID string) error {
	err := s.client.DeleteEnterprise(ID)
	if err != nil {
		return err
	}

	return nil
}
