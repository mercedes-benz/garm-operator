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
	InstallOrgWebhook(param *organizations.InstallOrgWebhookParams) (*organizations.InstallOrgWebhookOK, error)
	GetOrgWebhookInfo(param *organizations.GetOrgWebhookInfoParams) (*organizations.GetOrgWebhookInfoOK, error)
	UninstallOrgWebhook(param *organizations.UninstallOrgWebhookParams) error
}

type organizationClient struct {
	GarmClient
}

func NewOrganizationClient() OrganizationClient {
	return &organizationClient{
		Client,
	}
}

func (s *organizationClient) ListOrganizations(param *organizations.ListOrgsParams) (*organizations.ListOrgsOK, error) {
	return EnsureAuth(func() (*organizations.ListOrgsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("organization.List").Inc()
		organizations, err := s.GarmAPI().Organizations.ListOrgs(param, s.Token())
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
		organization, err := s.GarmAPI().Organizations.CreateOrg(param, s.Token())
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
		organization, err := s.GarmAPI().Organizations.GetOrg(param, s.Token())
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
		err := s.GarmAPI().Organizations.DeleteOrg(param, s.Token())
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
		organization, err := s.GarmAPI().Organizations.UpdateOrg(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("organization.Update").Inc()
			return nil, err
		}
		return organization, nil
	})
}

func (s *organizationClient) InstallOrgWebhook(param *organizations.InstallOrgWebhookParams) (*organizations.InstallOrgWebhookOK, error) {
	return EnsureAuth(func() (*organizations.InstallOrgWebhookOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("organization.InstallWebhook").Inc()
		webhook, err := s.GarmAPI().Organizations.InstallOrgWebhook(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("organization.InstallWebhook").Inc()
			return nil, err
		}
		return webhook, nil
	})
}

func (s *organizationClient) GetOrgWebhookInfo(param *organizations.GetOrgWebhookInfoParams) (*organizations.GetOrgWebhookInfoOK, error) {
	return EnsureAuth(func() (*organizations.GetOrgWebhookInfoOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("organization.GetWebhookInfo").Inc()
		webhook, err := s.GarmAPI().Organizations.GetOrgWebhookInfo(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("organization.GetWebhookInfo").Inc()
			return nil, err
		}
		return webhook, nil
	})
}

func (s *organizationClient) UninstallOrgWebhook(param *organizations.UninstallOrgWebhookParams) error {
	_, err := EnsureAuth(func() (interface{}, error) {
		metrics.TotalGarmCalls.WithLabelValues("organization.UninstallWebhook").Inc()
		err := s.GarmAPI().Organizations.UninstallOrgWebhook(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("organization.UninstallWebhook").Inc()
			return nil, err
		}
		return nil, nil
	})
	return err
}
