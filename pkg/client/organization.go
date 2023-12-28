// SPDX-License-Identifier: MIT

package client

import (
	"github.com/cloudbase/garm/client/organizations"
	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type OrganizationClient interface {
	ListOrganizations(param *organizations.ListOrgsParams) (*organizations.ListOrgsOK, error)
	CreateOrganization(param *organizations.CreateOrgParams) (*organizations.CreateOrgOK, error)
	GetOrganization(param *organizations.GetOrgParams) (*organizations.GetOrgOK, error)
	UpdateOrganization(param *organizations.UpdateOrgParams) (*organizations.UpdateOrgOK, error)
	DeleteOrganization(param *organizations.DeleteOrgParams) error
}

type organizationClient struct {
	BaseClient
}

func NewOrganizationClient() OrganizationClient {
	return &organizationClient{
		Instance,
	}
}

func (s *organizationClient) ListOrganizations(param *organizations.ListOrgsParams) (*organizations.ListOrgsOK, error) {
	return EnsureAuth(func() (*organizations.ListOrgsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("organization.List").Inc()
		organizations, err := s.Client().Organizations.ListOrgs(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("organization.List").Inc()
			return nil, err
		}
		return organizations, nil
	})
}

func (s *organizationClient) CreateOrganization(param *organizations.CreateOrgParams) (*organizations.CreateOrgOK, error) {
	return EnsureAuth(func() (*organizations.CreateOrgOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("organization.Create").Inc()
		organization, err := s.Client().Organizations.CreateOrg(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("organization.Create").Inc()
			return nil, err
		}
		return organization, nil
	})
}

func (s *organizationClient) GetOrganization(param *organizations.GetOrgParams) (*organizations.GetOrgOK, error) {
	return EnsureAuth(func() (*organizations.GetOrgOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("organization.Get").Inc()
		organization, err := s.Client().Organizations.GetOrg(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("organization.Get").Inc()
			return nil, err
		}
		return organization, nil
	})
}

func (s *organizationClient) DeleteOrganization(param *organizations.DeleteOrgParams) error {
	_, err := EnsureAuth(func() (interface{}, error) {
		metrics.TotalGarmCalls.WithLabelValues("organization.Delete").Inc()
		err := s.Client().Organizations.DeleteOrg(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("organization.Delete").Inc()
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (s *organizationClient) UpdateOrganization(param *organizations.UpdateOrgParams) (*organizations.UpdateOrgOK, error) {
	return EnsureAuth(func() (*organizations.UpdateOrgOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("organization.Update").Inc()
		organization, err := s.Client().Organizations.UpdateOrg(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("organization.Update").Inc()
			return nil, err
		}
		return organization, nil
	})
}
