package client

import (
	"github.com/cloudbase/garm/client"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/go-openapi/runtime"

	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/metrics"
)

type OrganizationClient interface {
	ListOrganizations(param *organizations.ListOrgsParams) (*organizations.ListOrgsOK, error)
	CreateOrganization(param *organizations.CreateOrgParams) (*organizations.CreateOrgOK, error)
	GetOrganization(param *organizations.GetOrgParams) (*organizations.GetOrgOK, error)
	UpdateOrganization(param *organizations.UpdateOrgParams) (*organizations.UpdateOrgOK, error)
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
	metrics.TotalGarmCalls.WithLabelValues("organization.List").Inc()
	organizations, err := s.client.Organizations.ListOrgs(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("organization.List").Inc()
		return nil, err
	}

	return organizations, nil
}

func (s *organizationClient) CreateOrganization(param *organizations.CreateOrgParams) (*organizations.CreateOrgOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("organization.Create").Inc()
	organization, err := s.client.Organizations.CreateOrg(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("organization.Create").Inc()
		return nil, err
	}

	return organization, nil
}

func (s *organizationClient) GetOrganization(param *organizations.GetOrgParams) (*organizations.GetOrgOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("organization.Get").Inc()
	organization, err := s.client.Organizations.GetOrg(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("organization.Get").Inc()
		return nil, err
	}
	return organization, nil
}

func (s *organizationClient) DeleteOrganization(param *organizations.DeleteOrgParams) error {
	metrics.TotalGarmCalls.WithLabelValues("organization.Delete").Inc()
	err := s.client.Organizations.DeleteOrg(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("organization.Delete").Inc()
		return err
	}

	return nil
}

func (s *organizationClient) UpdateOrganization(param *organizations.UpdateOrgParams) (*organizations.UpdateOrgOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("organization.Update").Inc()
	organization, err := s.client.Organizations.UpdateOrg(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("organization.Update").Inc()
		return nil, err
	}

	return organization, nil
}
