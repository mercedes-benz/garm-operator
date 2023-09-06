package client

import (
	"github.com/cloudbase/garm/client"
	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/client/pools"
	"github.com/cloudbase/garm/client/repositories"
	"github.com/go-openapi/runtime"

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
	client *client.GarmAPI
	token  runtime.ClientAuthInfoWriter
}

func NewPoolClient(garmParams GarmScopeParams) (PoolClient, error) {
	garmClient, token, err := newGarmClient(garmParams)
	if err != nil {
		return nil, err
	}

	return &poolClient{garmClient, token}, nil
}

func (p *poolClient) ListAllPools(param *pools.ListPoolsParams) (*pools.ListPoolsOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("pool.List").Inc()
	pools, err := p.client.Pools.ListPools(param, p.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("pool.List").Inc()
		return nil, err
	}
	return pools, nil
}

func (p *poolClient) CreateRepoPool(param *repositories.CreateRepoPoolParams) (*repositories.CreateRepoPoolOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("pool.CreateRepo").Inc()
	pool, err := p.client.Repositories.CreateRepoPool(param, p.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("pool.CreateRepo").Inc()
		return nil, err
	}
	return pool, nil
}

func (p *poolClient) CreateOrgPool(param *organizations.CreateOrgPoolParams) (*organizations.CreateOrgPoolOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("pool.CreateOrg").Inc()
	pool, err := p.client.Organizations.CreateOrgPool(param, p.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("pool.CreateOrg").Inc()
		return nil, err
	}
	return pool, nil
}

func (p *poolClient) CreateEnterprisePool(param *enterprises.CreateEnterprisePoolParams) (*enterprises.CreateEnterprisePoolOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("pool.CreateEnterprise").Inc()
	pool, err := p.client.Enterprises.CreateEnterprisePool(param, p.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("pool.CreateEnterprise").Inc()
		return nil, err
	}
	return pool, nil
}

func (p *poolClient) UpdateEnterprisePool(param *enterprises.UpdateEnterprisePoolParams) (*enterprises.UpdateEnterprisePoolOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("pool.UpdateEnterprise").Inc()
	pool, err := p.client.Enterprises.UpdateEnterprisePool(param, p.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("pool.UpdateEnterprise").Inc()
		return nil, err
	}
	return pool, nil
}

func (p *poolClient) UpdatePool(param *pools.UpdatePoolParams) (*pools.UpdatePoolOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("pool.UpdatePool").Inc()
	pool, err := p.client.Pools.UpdatePool(param, p.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("pool.UpdatePool").Inc()
		return nil, err
	}
	return pool, nil
}

func (p *poolClient) GetEnterprisePool(param *enterprises.GetEnterprisePoolParams) (*enterprises.GetEnterprisePoolOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("pool.GetEnterprise").Inc()
	pool, err := p.client.Enterprises.GetEnterprisePool(param, p.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("pool.GetEnterprise").Inc()
		return nil, err
	}
	return pool, nil
}

func (p *poolClient) GetPool(param *pools.GetPoolParams) (*pools.GetPoolOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("pool.Get").Inc()
	pool, err := p.client.Pools.GetPool(param, p.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("pool.Get").Inc()
		return nil, err
	}
	return pool, nil
}

func (p *poolClient) DeletePool(param *pools.DeletePoolParams) error {
	metrics.TotalGarmCalls.WithLabelValues("pool.Delete").Inc()
	err := p.client.Pools.DeletePool(param, p.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("pool.Delete").Inc()
		return err
	}
	return nil
}

func (p *poolClient) DeleteEnterprisePool(param *enterprises.DeleteEnterprisePoolParams) error {
	metrics.TotalGarmCalls.WithLabelValues("pool.DeleteEnterprise").Inc()
	err := p.client.Enterprises.DeleteEnterprisePool(param, p.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("pool.DeleteEnterprise").Inc()
		return err
	}
	return nil
}
