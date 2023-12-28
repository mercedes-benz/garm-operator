// SPDX-License-Identifier: MIT

package client

import (
	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/client/pools"
	"github.com/cloudbase/garm/client/repositories"
	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type PoolClient interface {
	ListAllPools(param *pools.ListPoolsParams) (*pools.ListPoolsOK, error)
	CreateRepoPool(param *repositories.CreateRepoPoolParams) (*repositories.CreateRepoPoolOK, error)
	CreateOrgPool(param *organizations.CreateOrgPoolParams) (*organizations.CreateOrgPoolOK, error)
	CreateEnterprisePool(param *enterprises.CreateEnterprisePoolParams) (*enterprises.CreateEnterprisePoolOK, error)
	UpdatePool(param *pools.UpdatePoolParams) (*pools.UpdatePoolOK, error)
	UpdateEnterprisePool(param *enterprises.UpdateEnterprisePoolParams) (*enterprises.UpdateEnterprisePoolOK, error)
	GetPool(param *pools.GetPoolParams) (*pools.GetPoolOK, error)
	GetEnterprisePool(param *enterprises.GetEnterprisePoolParams) (*enterprises.GetEnterprisePoolOK, error)
	DeletePool(param *pools.DeletePoolParams) error
	DeleteEnterprisePool(param *enterprises.DeleteEnterprisePoolParams) error
}

type poolClient struct {
	BaseClient
}

func NewPoolClient() PoolClient {
	return &poolClient{
		Instance,
	}
}

func (p *poolClient) ListAllPools(param *pools.ListPoolsParams) (*pools.ListPoolsOK, error) {
	return EnsureAuth(func() (*pools.ListPoolsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("pool.List").Inc()
		pools, err := p.Client().Pools.ListPools(param, p.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("pool.List").Inc()
			return nil, err
		}
		return pools, nil
	})
}

func (p *poolClient) CreateRepoPool(param *repositories.CreateRepoPoolParams) (*repositories.CreateRepoPoolOK, error) {
	return EnsureAuth(func() (*repositories.CreateRepoPoolOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("pool.CreateRepo").Inc()
		pool, err := p.Client().Repositories.CreateRepoPool(param, p.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("pool.CreateRepo").Inc()
			return nil, err
		}
		return pool, nil
	})
}

func (p *poolClient) CreateOrgPool(param *organizations.CreateOrgPoolParams) (*organizations.CreateOrgPoolOK, error) {
	return EnsureAuth(func() (*organizations.CreateOrgPoolOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("pool.CreateOrg").Inc()
		pool, err := p.Client().Organizations.CreateOrgPool(param, p.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("pool.CreateOrg").Inc()
			return nil, err
		}
		return pool, nil
	})
}

func (p *poolClient) CreateEnterprisePool(param *enterprises.CreateEnterprisePoolParams) (*enterprises.CreateEnterprisePoolOK, error) {
	return EnsureAuth(func() (*enterprises.CreateEnterprisePoolOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("pool.CreateEnterprise").Inc()
		pool, err := p.Client().Enterprises.CreateEnterprisePool(param, p.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("pool.CreateEnterprise").Inc()
			return nil, err
		}
		return pool, nil
	})
}

func (p *poolClient) UpdateEnterprisePool(param *enterprises.UpdateEnterprisePoolParams) (*enterprises.UpdateEnterprisePoolOK, error) {
	return EnsureAuth(func() (*enterprises.UpdateEnterprisePoolOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("pool.UpdateEnterprise").Inc()
		pool, err := p.Client().Enterprises.UpdateEnterprisePool(param, p.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("pool.UpdateEnterprise").Inc()
			return nil, err
		}
		return pool, nil
	})
}

func (p *poolClient) UpdatePool(param *pools.UpdatePoolParams) (*pools.UpdatePoolOK, error) {
	return EnsureAuth(func() (*pools.UpdatePoolOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("pool.UpdatePool").Inc()
		pool, err := p.Client().Pools.UpdatePool(param, p.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("pool.UpdatePool").Inc()
			return nil, err
		}
		return pool, nil
	})
}

func (p *poolClient) GetEnterprisePool(param *enterprises.GetEnterprisePoolParams) (*enterprises.GetEnterprisePoolOK, error) {
	return EnsureAuth(func() (*enterprises.GetEnterprisePoolOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("pool.GetEnterprise").Inc()
		pool, err := p.Client().Enterprises.GetEnterprisePool(param, p.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("pool.GetEnterprise").Inc()
			return nil, err
		}
		return pool, nil
	})
}

func (p *poolClient) GetPool(param *pools.GetPoolParams) (*pools.GetPoolOK, error) {
	return EnsureAuth(func() (*pools.GetPoolOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("pool.Get").Inc()
		pool, err := p.Client().Pools.GetPool(param, p.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("pool.Get").Inc()
			return nil, err
		}
		return pool, nil
	})
}

func (p *poolClient) DeletePool(param *pools.DeletePoolParams) error {
	_, err := EnsureAuth(func() (interface{}, error) {
		metrics.TotalGarmCalls.WithLabelValues("pool.Delete").Inc()
		err := p.Client().Pools.DeletePool(param, p.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("pool.Delete").Inc()
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (p *poolClient) DeleteEnterprisePool(param *enterprises.DeleteEnterprisePoolParams) error {
	_, err := EnsureAuth(func() (interface{}, error) {
		metrics.TotalGarmCalls.WithLabelValues("pool.DeleteEnterprise").Inc()
		err := p.Client().Enterprises.DeleteEnterprisePool(param, p.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("pool.DeleteEnterprise").Inc()
			return nil, err
		}
		return nil, nil
	})
	return err
}
