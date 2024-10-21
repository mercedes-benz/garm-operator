// SPDX-License-Identifier: MIT
package client

import (
	"github.com/cloudbase/garm/client/credentials"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type CredentialsClient interface {
	GetCredentials(params *credentials.GetCredentialsParams) (*credentials.GetCredentialsOK, error)
	ListCredentials(params *credentials.ListCredentialsParams) (*credentials.ListCredentialsOK, error)
	CreateCredentials(params *credentials.CreateCredentialsParams) (*credentials.CreateCredentialsOK, error)
	UpdateCredentials(params *credentials.UpdateCredentialsParams) (*credentials.UpdateCredentialsOK, error)
	DeleteCredentials(params *credentials.DeleteCredentialsParams) error
}

type credentialClient struct {
	GarmClient
}

func (e *credentialClient) GetCredentials(params *credentials.GetCredentialsParams) (*credentials.GetCredentialsOK, error) {
	return EnsureAuth(func() (*credentials.GetCredentialsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("credentials.Get").Inc()
		endpoint, err := e.GarmAPI().Credentials.GetCredentials(params, e.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("credentials.Get").Inc()
			return nil, err
		}
		return endpoint, nil
	})
}

func (e *credentialClient) ListCredentials(params *credentials.ListCredentialsParams) (*credentials.ListCredentialsOK, error) {
	return EnsureAuth(func() (*credentials.ListCredentialsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("credentials.List").Inc()
		credentials, err := e.GarmAPI().Credentials.ListCredentials(params, e.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("credentials.List").Inc()
			return nil, err
		}
		return credentials, nil
	})
}

func (e *credentialClient) CreateCredentials(params *credentials.CreateCredentialsParams) (*credentials.CreateCredentialsOK, error) {
	return EnsureAuth(func() (*credentials.CreateCredentialsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("credentials.Create").Inc()
		endpoint, err := e.GarmAPI().Credentials.CreateCredentials(params, e.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("credentials.Create").Inc()
			return nil, err
		}
		return endpoint, nil
	})
}

func (e *credentialClient) UpdateCredentials(params *credentials.UpdateCredentialsParams) (*credentials.UpdateCredentialsOK, error) {
	return EnsureAuth(func() (*credentials.UpdateCredentialsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("credentials.Update").Inc()
		endpoint, err := e.GarmAPI().Credentials.UpdateCredentials(params, e.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("credentials.Update").Inc()
			return nil, err
		}
		return endpoint, nil
	})
}

func (e *credentialClient) DeleteCredentials(params *credentials.DeleteCredentialsParams) error {
	_, err := EnsureAuth(func() (interface{}, error) {
		metrics.TotalGarmCalls.WithLabelValues("credentials.Delete").Inc()
		if err := e.GarmAPI().Credentials.DeleteCredentials(params, e.Token()); err != nil {
			metrics.GarmCallErrors.WithLabelValues("credentials.Delete").Inc()
			return nil, err
		}
		return nil, nil
	})
	return err
}

func NewCredentialsClient() CredentialsClient {
	return &credentialClient{
		Client,
	}
}
