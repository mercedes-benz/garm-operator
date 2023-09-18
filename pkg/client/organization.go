// SPDX-License-Identifier: MIT

package client

import (
	"github.com/cloudbase/garm/client"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/go-openapi/runtime"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type OrganizationClient interface {
	Login(garmParams GarmScopeParams) error
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

func NewOrganizationClient() OrganizationClient {
	return &organizationClient{}
}

func (s *organizationClient) Login(garmParams GarmScopeParams) error {
	metrics.TotalGarmCalls.WithLabelValues("Login").Inc()
	garmClient, token, err := newGarmClient(garmParams)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("Login").Inc()
		return err
	}
	s.client = garmClient
	s.token = token

	return nil
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
