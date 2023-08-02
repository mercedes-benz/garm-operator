package client

import (
	"github.com/cloudbase/garm/client"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/go-openapi/runtime"
)

type OrganizationClient interface {
	ListOrganizations(param *organizations.ListOrgsParams) (*organizations.ListOrgsOK, error)
	CreateOrganization(param *organizations.CreateOrgParams) (*organizations.CreateOrgOK, error)
	GetOrganization(param *organizations.GetOrgParams) (*organizations.GetOrgOK, error)
	DeleteOrganization(param *organizations.DeleteOrgParams) error
}

type organizationClient struct {
	client *client.GarmAPI
	token  runtime.ClientAuthInfoWriter
}

func NewOrganizationClient(garmParams GarmScopeParams) (OrganizationClient, error) {
	garmClient, token, err := newGarmClient(garmParams)
	if err != nil {
		return nil, err
	}

	return &organizationClient{garmClient, token}, nil
}

func (s *organizationClient) ListOrganizations(param *organizations.ListOrgsParams) (*organizations.ListOrgsOK, error) {
	// TODO: track requests by introducing metrics
	organizations, err := s.client.Organizations.ListOrgs(param, s.token)
	if err != nil {
		return nil, err
	}

	return organizations, nil
}

func (s *organizationClient) CreateOrganization(param *organizations.CreateOrgParams) (*organizations.CreateOrgOK, error) {
	organization, err := s.client.Organizations.CreateOrg(param, s.token)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (s *organizationClient) GetOrganization(param *organizations.GetOrgParams) (*organizations.GetOrgOK, error) {
	organization, err := s.client.Organizations.GetOrg(param, s.token)
	if err != nil {
		return nil, err
	}
	return organization, nil
}

func (s *organizationClient) DeleteOrganization(param *organizations.DeleteOrgParams) error {
	err := s.client.Organizations.DeleteOrg(param, s.token)
	if err != nil {
		return err
	}

	return nil
}
