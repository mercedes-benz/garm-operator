// SPDX-License-Identifier: MIT

package client

import (
	"github.com/cloudbase/garm/client/repositories"
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
	BaseClient
}

func NewRepositoryClient() RepositoryClient {
	return &repositoryClient{
		Instance,
	}
}

func (s *repositoryClient) ListRepositories(param *repositories.ListReposParams) (*repositories.ListReposOK, error) {
	return EnsureAuth(func() (*repositories.ListReposOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("repository.List").Inc()
		repositories, err := s.Client().Repositories.ListRepos(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("repository.List").Inc()
			return nil, err
		}
		return repositories, nil
	})
}

func (s *repositoryClient) CreateRepository(param *repositories.CreateRepoParams) (*repositories.CreateRepoOK, error) {
	return EnsureAuth(func() (*repositories.CreateRepoOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("repository.Create").Inc()
		repository, err := s.Client().Repositories.CreateRepo(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("repository.Create").Inc()
			return nil, err
		}
		return repository, nil
	})
}

func (s *repositoryClient) GetRepository(param *repositories.GetRepoParams) (*repositories.GetRepoOK, error) {
	return EnsureAuth(func() (*repositories.GetRepoOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("repository.Get").Inc()
		repository, err := s.Client().Repositories.GetRepo(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("repository.Get").Inc()
			return nil, err
		}
		return repository, nil
	})
}

func (s *repositoryClient) DeleteRepository(param *repositories.DeleteRepoParams) error {
	_, err := EnsureAuth(func() (interface{}, error) {
		metrics.TotalGarmCalls.WithLabelValues("repository.Delete").Inc()
		err := s.Client().Repositories.DeleteRepo(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("repository.Delete").Inc()
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (s *repositoryClient) UpdateRepository(param *repositories.UpdateRepoParams) (*repositories.UpdateRepoOK, error) {
	return EnsureAuth(func() (*repositories.UpdateRepoOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("repository.Update").Inc()
		repository, err := s.Client().Repositories.UpdateRepo(param, s.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("repository.Update").Inc()
			return nil, err
		}
		return repository, nil
	})
}
