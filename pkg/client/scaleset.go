// SPDX-License-Identifier: MIT

package client

import (
	"strconv"

	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/client/repositories"
	"github.com/cloudbase/garm/client/scalesets"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

// ScaleSetClient wraps the generated GARM scale-set API with auth refresh
// and call metrics, mirroring the PoolClient pattern.
type ScaleSetClient interface {
	GetScaleSet(id uint) (*scalesets.GetScaleSetOK, error)
	ListOrgScaleSets(orgID string) (*organizations.ListOrgScaleSetsOK, error)
	ListRepoScaleSets(repoID string) (*repositories.ListRepoScaleSetsOK, error)
	ListEnterpriseScaleSets(enterpriseID string) (*enterprises.ListEnterpriseScaleSetsOK, error)
	CreateOrgScaleSet(orgID string, param *organizations.CreateOrgScaleSetParams) (*organizations.CreateOrgScaleSetOK, error)
	CreateRepoScaleSet(repoID string, param *repositories.CreateRepoScaleSetParams) (*repositories.CreateRepoScaleSetOK, error)
	CreateEnterpriseScaleSet(enterpriseID string, param *enterprises.CreateEnterpriseScaleSetParams) (*enterprises.CreateEnterpriseScaleSetOK, error)
	UpdateScaleSet(id uint, param *scalesets.UpdateScaleSetParams) (*scalesets.UpdateScaleSetOK, error)
	DeleteScaleSet(id uint) error
}

type scaleSetClient struct {
	GarmClient
}

func NewScaleSetClient() ScaleSetClient {
	return &scaleSetClient{
		Client,
	}
}

func scaleSetID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

func (s *scaleSetClient) GetScaleSet(id uint) (*scalesets.GetScaleSetOK, error) {
	return EnsureAuth(func() (*scalesets.GetScaleSetOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("scaleset.Get").Inc()
		set, err := s.GarmAPI().Scalesets.GetScaleSet(scalesets.NewGetScaleSetParams().WithScalesetID(scaleSetID(id)), s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("scaleset.Get").Inc()
			return nil, err
		}
		return set, nil
	})
}

func (s *scaleSetClient) ListOrgScaleSets(orgID string) (*organizations.ListOrgScaleSetsOK, error) {
	return EnsureAuth(func() (*organizations.ListOrgScaleSetsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("scaleset.ListOrg").Inc()
		sets, err := s.GarmAPI().Organizations.ListOrgScaleSets(organizations.NewListOrgScaleSetsParams().WithOrgID(orgID), s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("scaleset.ListOrg").Inc()
			return nil, err
		}
		return sets, nil
	})
}

func (s *scaleSetClient) ListRepoScaleSets(repoID string) (*repositories.ListRepoScaleSetsOK, error) {
	return EnsureAuth(func() (*repositories.ListRepoScaleSetsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("scaleset.ListRepo").Inc()
		sets, err := s.GarmAPI().Repositories.ListRepoScaleSets(repositories.NewListRepoScaleSetsParams().WithRepoID(repoID), s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("scaleset.ListRepo").Inc()
			return nil, err
		}
		return sets, nil
	})
}

func (s *scaleSetClient) ListEnterpriseScaleSets(enterpriseID string) (*enterprises.ListEnterpriseScaleSetsOK, error) {
	return EnsureAuth(func() (*enterprises.ListEnterpriseScaleSetsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("scaleset.ListEnterprise").Inc()
		sets, err := s.GarmAPI().Enterprises.ListEnterpriseScaleSets(enterprises.NewListEnterpriseScaleSetsParams().WithEnterpriseID(enterpriseID), s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("scaleset.ListEnterprise").Inc()
			return nil, err
		}
		return sets, nil
	})
}

func (s *scaleSetClient) CreateOrgScaleSet(orgID string, param *organizations.CreateOrgScaleSetParams) (*organizations.CreateOrgScaleSetOK, error) {
	return EnsureAuth(func() (*organizations.CreateOrgScaleSetOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("scaleset.CreateOrg").Inc()
		set, err := s.GarmAPI().Organizations.CreateOrgScaleSet(param.WithOrgID(orgID), s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("scaleset.CreateOrg").Inc()
			return nil, err
		}
		return set, nil
	})
}

func (s *scaleSetClient) CreateRepoScaleSet(repoID string, param *repositories.CreateRepoScaleSetParams) (*repositories.CreateRepoScaleSetOK, error) {
	return EnsureAuth(func() (*repositories.CreateRepoScaleSetOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("scaleset.CreateRepo").Inc()
		set, err := s.GarmAPI().Repositories.CreateRepoScaleSet(param.WithRepoID(repoID), s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("scaleset.CreateRepo").Inc()
			return nil, err
		}
		return set, nil
	})
}

func (s *scaleSetClient) CreateEnterpriseScaleSet(enterpriseID string, param *enterprises.CreateEnterpriseScaleSetParams) (*enterprises.CreateEnterpriseScaleSetOK, error) {
	return EnsureAuth(func() (*enterprises.CreateEnterpriseScaleSetOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("scaleset.CreateEnterprise").Inc()
		set, err := s.GarmAPI().Enterprises.CreateEnterpriseScaleSet(param.WithEnterpriseID(enterpriseID), s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("scaleset.CreateEnterprise").Inc()
			return nil, err
		}
		return set, nil
	})
}

func (s *scaleSetClient) UpdateScaleSet(id uint, param *scalesets.UpdateScaleSetParams) (*scalesets.UpdateScaleSetOK, error) {
	return EnsureAuth(func() (*scalesets.UpdateScaleSetOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("scaleset.Update").Inc()
		set, err := s.GarmAPI().Scalesets.UpdateScaleSet(param.WithScalesetID(scaleSetID(id)), s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("scaleset.Update").Inc()
			return nil, err
		}
		return set, nil
	})
}

func (s *scaleSetClient) DeleteScaleSet(id uint) error {
	_, err := EnsureAuth(func() (interface{}, error) {
		metrics.TotalGarmCalls.WithLabelValues("scaleset.Delete").Inc()
		err := s.GarmAPI().Scalesets.DeleteScaleSet(scalesets.NewDeleteScaleSetParams().WithScalesetID(scaleSetID(id)), s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("scaleset.Delete").Inc()
			return nil, err
		}
		return nil, nil
	})
	return err
}
