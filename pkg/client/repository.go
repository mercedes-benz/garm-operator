// SPDX-License-Identifier: MIT

package client

import (
	"github.com/cloudbase/garm/client"
	"github.com/cloudbase/garm/client/repositories"
	"github.com/go-openapi/runtime"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type RepositoryClient interface {
	ListRepositories(param *repositories.ListReposParams) (*repositories.ListReposOK, error)
	CreateRepository(param *repositories.CreateRepoParams) (*repositories.CreateRepoOK, error)
	GetRepository(param *repositories.GetRepoParams) (*repositories.GetRepoOK, error)
	UpdateRepository(param *repositories.UpdateRepoParams) (*repositories.UpdateRepoOK, error)
	DeleteRepository(param *repositories.DeleteRepoParams) error
}

type repositoryClient struct {
	client *client.GarmAPI
	token  runtime.ClientAuthInfoWriter
}

func NewRepositoryClient(garmParams GarmScopeParams) (RepositoryClient, error) {
	garmClient, token, err := newGarmClient(garmParams)
	if err != nil {
		return nil, err
	}

	return &repositoryClient{garmClient, token}, nil
}

func (s *repositoryClient) ListRepositories(param *repositories.ListReposParams) (*repositories.ListReposOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("repository.List").Inc()
	repositories, err := s.client.Repositories.ListRepos(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("repository.List").Inc()
		return nil, err
	}

	return repositories, nil
}

func (s *repositoryClient) CreateRepository(param *repositories.CreateRepoParams) (*repositories.CreateRepoOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("repository.Create").Inc()
	repository, err := s.client.Repositories.CreateRepo(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("repository.Create").Inc()
		return nil, err
	}

	return repository, nil
}

func (s *repositoryClient) GetRepository(param *repositories.GetRepoParams) (*repositories.GetRepoOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("repository.Get").Inc()
	repository, err := s.client.Repositories.GetRepo(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("repository.Get").Inc()
		return nil, err
	}
	return repository, nil
}

func (s *repositoryClient) DeleteRepository(param *repositories.DeleteRepoParams) error {
	metrics.TotalGarmCalls.WithLabelValues("repository.Delete").Inc()
	err := s.client.Repositories.DeleteRepo(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("repository.Delete").Inc()
		return err
	}

	return nil
}

func (s *repositoryClient) UpdateRepository(param *repositories.UpdateRepoParams) (*repositories.UpdateRepoOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("repository.Update").Inc()
	repository, err := s.client.Repositories.UpdateRepo(param, s.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("repository.Update").Inc()
		return nil, err
	}

	return repository, nil
}
