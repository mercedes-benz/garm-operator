package client

import (
	"github.com/cloudbase/garm/client"
	"github.com/cloudbase/garm/client/instances"
	"github.com/go-openapi/runtime"
)

type InstanceClient interface {
	GetInstance(params *instances.GetInstanceParams) (*instances.GetInstanceOK, error)
	ListInstances(params *instances.ListInstancesParams) (*instances.ListInstancesOK, error)
	ListPoolInstances(params *instances.ListPoolInstancesParams) (*instances.ListPoolInstancesOK, error)
}

type instanceClient struct {
	client *client.GarmAPI
	token  runtime.ClientAuthInfoWriter
}

func NewInstanceClient(garmParams GarmScopeParams) (InstanceClient, error) {
	garmClient, token, err := newGarmClient(garmParams)
	if err != nil {
		return nil, err
	}

	return &instanceClient{garmClient, token}, nil
}

func (i *instanceClient) GetInstance(params *instances.GetInstanceParams) (*instances.GetInstanceOK, error) {
	instance, err := i.client.Instances.GetInstance(params, i.token)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (i *instanceClient) ListInstances(params *instances.ListInstancesParams) (*instances.ListInstancesOK, error) {
	instances, err := i.client.Instances.ListInstances(params, i.token)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (i *instanceClient) ListPoolInstances(params *instances.ListPoolInstancesParams) (*instances.ListPoolInstancesOK, error) {
	instances, err := i.client.Instances.ListPoolInstances(params, i.token)
	if err != nil {
		return nil, err
	}
	return instances, nil
}
