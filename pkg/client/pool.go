package client

import (
	"github.com/cloudbase/garm/cmd/garm-cli/client"
	"github.com/cloudbase/garm/params"
)

type PoolClient interface {
	ListAllPools() ([]params.Pool, error)
	CreateRepoPool(repoId string, param params.CreatePoolParams) (params.Pool, error)
	CreateOrgPool(orgId string, param params.CreatePoolParams) (params.Pool, error)
	CreateEnterprisePool(enterpriseId string, param params.CreatePoolParams) (params.Pool, error)
	UpdatePoolByID(ID string, param params.UpdatePoolParams) (params.Pool, error)
	GetPool(ID string) (params.Pool, error)
	DeletePoolByID(ID string) error
}

type poolClient struct{ client *client.Client }

func NewPoolClient(garmParams GarmScopeParams) (PoolClient, error) {
	garmClient, err := newGarmClient(garmParams)
	if err != nil {
		return nil, err
	}

	return &poolClient{garmClient}, nil
}

func (p *poolClient) ListAllPools() ([]params.Pool, error) {
	pools, err := p.client.ListAllPools()
	if err != nil {
		return []params.Pool{}, err
	}
	return pools, nil
}

func (p *poolClient) CreateRepoPool(repoId string, param params.CreatePoolParams) (params.Pool, error) {
	pool, err := p.client.CreateRepoPool(repoId, param)
	if err != nil {
		return params.Pool{}, err
	}
	return pool, nil
}

func (p *poolClient) CreateOrgPool(orgId string, param params.CreatePoolParams) (params.Pool, error) {
	pool, err := p.client.CreateOrgPool(orgId, param)
	if err != nil {
		return params.Pool{}, err
	}
	return pool, nil
}

func (p *poolClient) CreateEnterprisePool(enterpriseId string, param params.CreatePoolParams) (params.Pool, error) {
	pool, err := p.client.CreateEnterprisePool(enterpriseId, param)
	if err != nil {
		return params.Pool{}, err
	}
	return pool, nil
}

func (p *poolClient) UpdatePoolByID(ID string, param params.UpdatePoolParams) (params.Pool, error) {
	pool, err := p.client.UpdatePoolByID(ID, param)
	if err != nil {
		return params.Pool{}, err
	}
	return pool, nil
}

func (p *poolClient) GetPool(ID string) (params.Pool, error) {
	pool, err := p.client.GetPoolByID(ID)
	if err != nil {
		return params.Pool{}, err
	}
	return pool, nil
}

func (p *poolClient) DeletePoolByID(ID string) error {
	err := p.client.DeletePoolByID(ID)
	if err != nil {
		return err
	}
	return nil
}
