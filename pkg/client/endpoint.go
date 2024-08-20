package client

import (
	"github.com/cloudbase/garm/client/endpoints"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type EndpointClient interface {
	GetEndpoint(params *endpoints.GetGithubEndpointParams) (*endpoints.GetGithubEndpointOK, error)
	ListEndpoints(params *endpoints.ListGithubEndpointsParams) (*endpoints.ListGithubEndpointsOK, error)
	CreateEndpoint(params *endpoints.CreateGithubEndpointParams) (*endpoints.CreateGithubEndpointOK, error)
	UpdateEndpoint(params *endpoints.UpdateGithubEndpointParams) (*endpoints.UpdateGithubEndpointOK, error)
	DeleteEndpoint(params *endpoints.DeleteGithubEndpointParams) error
}

type endpointClient struct {
	GarmClient
}

func (e *endpointClient) GetEndpoint(params *endpoints.GetGithubEndpointParams) (*endpoints.GetGithubEndpointOK, error) {
	return EnsureAuth(func() (*endpoints.GetGithubEndpointOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("endpoints.Get").Inc()
		endpoint, err := e.GarmAPI().Endpoints.GetGithubEndpoint(params, e.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("endpoints.Get").Inc()
			return nil, err
		}
		return endpoint, nil
	})
}

func (e *endpointClient) ListEndpoints(params *endpoints.ListGithubEndpointsParams) (*endpoints.ListGithubEndpointsOK, error) {
	return EnsureAuth(func() (*endpoints.ListGithubEndpointsOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("endpoints.List").Inc()
		endpoints, err := e.GarmAPI().Endpoints.ListGithubEndpoints(params, e.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("endpoints.List").Inc()
			return nil, err
		}
		return endpoints, nil
	})
}

func (e *endpointClient) CreateEndpoint(params *endpoints.CreateGithubEndpointParams) (*endpoints.CreateGithubEndpointOK, error) {
	return EnsureAuth(func() (*endpoints.CreateGithubEndpointOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("endpoints.Create").Inc()
		endpoint, err := e.GarmAPI().Endpoints.CreateGithubEndpoint(params, e.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("endpoints.Create").Inc()
			return nil, err
		}
		return endpoint, nil
	})
}

func (e *endpointClient) UpdateEndpoint(params *endpoints.UpdateGithubEndpointParams) (*endpoints.UpdateGithubEndpointOK, error) {
	return EnsureAuth(func() (*endpoints.UpdateGithubEndpointOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("endpoints.Update").Inc()
		endpoint, err := e.GarmAPI().Endpoints.UpdateGithubEndpoint(params, e.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("endpoints.Update").Inc()
			return nil, err
		}
		return endpoint, nil
	})
}

func (e *endpointClient) DeleteEndpoint(params *endpoints.DeleteGithubEndpointParams) error {
	_, err := EnsureAuth(func() (interface{}, error) {
		metrics.TotalGarmCalls.WithLabelValues("endpoints.Delete").Inc()
		if err := e.GarmAPI().Endpoints.DeleteGithubEndpoint(params, e.Token()); err != nil {
			metrics.GarmCallErrors.WithLabelValues("endpoints.Delete").Inc()
			return nil, err
		}
		return nil, nil
	})
	return err
}

func NewEndpointClient() EndpointClient {
	return &endpointClient{
		Client,
	}
}
