// SPDX-License-Identifier: MIT

package client

import (
	"github.com/cloudbase/garm/client"
	"github.com/cloudbase/garm/client/instances"
	"github.com/go-openapi/runtime"

	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type InstanceClient interface {
	Login(garmParams GarmScopeParams) error
	GetInstance(params *instances.GetInstanceParams) (*instances.GetInstanceOK, error)
	ListInstances(params *instances.ListInstancesParams) (*instances.ListInstancesOK, error)
	ListPoolInstances(params *instances.ListPoolInstancesParams) (*instances.ListPoolInstancesOK, error)
	DeleteInstance(params *instances.DeleteInstanceParams) error
}

type instanceClient struct {
	client *client.GarmAPI
	token  runtime.ClientAuthInfoWriter
}

func NewInstanceClient() InstanceClient {
	return &instanceClient{}
}

func (i *instanceClient) Login(garmParams GarmScopeParams) error {
	metrics.TotalGarmCalls.WithLabelValues("Login").Inc()
	garmClient, token, err := newGarmClient(garmParams)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("Login").Inc()
		return err
	}
	i.client = garmClient
	i.token = token

	return nil
}

func (i *instanceClient) GetInstance(params *instances.GetInstanceParams) (*instances.GetInstanceOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("instances.Get").Inc()
	instance, err := i.client.Instances.GetInstance(params, i.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("instances.Get").Inc()
		return nil, err
	}
	return instance, nil
}

func (i *instanceClient) ListInstances(params *instances.ListInstancesParams) (*instances.ListInstancesOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("instances.List").Inc()
	instances, err := i.client.Instances.ListInstances(params, i.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("instances.List").Inc()
		return nil, err
	}
	return instances, nil
}

func (i *instanceClient) ListPoolInstances(params *instances.ListPoolInstancesParams) (*instances.ListPoolInstancesOK, error) {
	metrics.TotalGarmCalls.WithLabelValues("instances.ListPool").Inc()
	instances, err := i.client.Instances.ListPoolInstances(params, i.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("instances.ListPool").Inc()
		return nil, err
	}
	return instances, nil
}

func (i *instanceClient) DeleteInstance(params *instances.DeleteInstanceParams) error {
	metrics.TotalGarmCalls.WithLabelValues("instances.Delete").Inc()
	err := i.client.Instances.DeleteInstance(params, i.token)
	if err != nil {
		metrics.GarmCallErrors.WithLabelValues("instances.ListPool").Inc()
		return err
	}
	return nil
}
