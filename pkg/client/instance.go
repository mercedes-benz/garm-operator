// SPDX-License-Identifier: MIT

package client

import (
	"github.com/cloudbase/garm/client/instances"
	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

type InstanceClient interface {
	GetInstance(params *instances.GetInstanceParams) (*instances.GetInstanceOK, error)
	ListInstances(params *instances.ListInstancesParams) (*instances.ListInstancesOK, error)
	ListPoolInstances(params *instances.ListPoolInstancesParams) (*instances.ListPoolInstancesOK, error)
	DeleteInstance(params *instances.DeleteInstanceParams) error
}

type instanceClient struct {
	BaseClient
}

func NewInstanceClient() InstanceClient {
	return &instanceClient{
		Instance,
	}
}

func (i *instanceClient) GetInstance(params *instances.GetInstanceParams) (*instances.GetInstanceOK, error) {
	return EnsureAuth(func() (*instances.GetInstanceOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("instances.Get").Inc()
		instance, err := i.Client().Instances.GetInstance(params, i.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("instances.Get").Inc()
			return nil, err
		}
		return instance, nil
	})
}

func (i *instanceClient) ListInstances(params *instances.ListInstancesParams) (*instances.ListInstancesOK, error) {
	return EnsureAuth(func() (*instances.ListInstancesOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("instances.List").Inc()
		instances, err := i.Client().Instances.ListInstances(params, i.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("instances.List").Inc()
			return nil, err
		}
		return instances, nil
	})
}

func (i *instanceClient) ListPoolInstances(params *instances.ListPoolInstancesParams) (*instances.ListPoolInstancesOK, error) {
	return EnsureAuth(func() (*instances.ListPoolInstancesOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("instances.ListPool").Inc()
		instances, err := i.Client().Instances.ListPoolInstances(params, i.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("instances.ListPool").Inc()
			return nil, err
		}
		return instances, nil
	})
}

func (i *instanceClient) DeleteInstance(params *instances.DeleteInstanceParams) error {
	_, err := EnsureAuth(func() (interface{}, error) {
		metrics.TotalGarmCalls.WithLabelValues("instances.Delete").Inc()
		err := i.Client().Instances.DeleteInstance(params, i.Token())
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("instances.ListPool").Inc()
			return nil, err
		}
		return nil, nil
	})
	return err
}
