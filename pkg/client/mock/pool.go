// Code generated by MockGen. DO NOT EDIT.
// Source: ../pool.go

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	enterprises "github.com/cloudbase/garm/client/enterprises"
	organizations "github.com/cloudbase/garm/client/organizations"
	pools "github.com/cloudbase/garm/client/pools"
	repositories "github.com/cloudbase/garm/client/repositories"
	gomock "go.uber.org/mock/gomock"
)

// MockPoolClient is a mock of PoolClient interface.
type MockPoolClient struct {
	ctrl     *gomock.Controller
	recorder *MockPoolClientMockRecorder
}

// MockPoolClientMockRecorder is the mock recorder for MockPoolClient.
type MockPoolClientMockRecorder struct {
	mock *MockPoolClient
}

// NewMockPoolClient creates a new mock instance.
func NewMockPoolClient(ctrl *gomock.Controller) *MockPoolClient {
	mock := &MockPoolClient{ctrl: ctrl}
	mock.recorder = &MockPoolClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPoolClient) EXPECT() *MockPoolClientMockRecorder {
	return m.recorder
}

// CreateEnterprisePool mocks base method.
func (m *MockPoolClient) CreateEnterprisePool(param *enterprises.CreateEnterprisePoolParams) (*enterprises.CreateEnterprisePoolOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateEnterprisePool", param)
	ret0, _ := ret[0].(*enterprises.CreateEnterprisePoolOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateEnterprisePool indicates an expected call of CreateEnterprisePool.
func (mr *MockPoolClientMockRecorder) CreateEnterprisePool(param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateEnterprisePool", reflect.TypeOf((*MockPoolClient)(nil).CreateEnterprisePool), param)
}

// CreateOrgPool mocks base method.
func (m *MockPoolClient) CreateOrgPool(param *organizations.CreateOrgPoolParams) (*organizations.CreateOrgPoolOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrgPool", param)
	ret0, _ := ret[0].(*organizations.CreateOrgPoolOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateOrgPool indicates an expected call of CreateOrgPool.
func (mr *MockPoolClientMockRecorder) CreateOrgPool(param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrgPool", reflect.TypeOf((*MockPoolClient)(nil).CreateOrgPool), param)
}

// CreateRepoPool mocks base method.
func (m *MockPoolClient) CreateRepoPool(param *repositories.CreateRepoPoolParams) (*repositories.CreateRepoPoolOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRepoPool", param)
	ret0, _ := ret[0].(*repositories.CreateRepoPoolOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRepoPool indicates an expected call of CreateRepoPool.
func (mr *MockPoolClientMockRecorder) CreateRepoPool(param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRepoPool", reflect.TypeOf((*MockPoolClient)(nil).CreateRepoPool), param)
}

// DeleteEnterprisePool mocks base method.
func (m *MockPoolClient) DeleteEnterprisePool(param *enterprises.DeleteEnterprisePoolParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteEnterprisePool", param)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteEnterprisePool indicates an expected call of DeleteEnterprisePool.
func (mr *MockPoolClientMockRecorder) DeleteEnterprisePool(param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteEnterprisePool", reflect.TypeOf((*MockPoolClient)(nil).DeleteEnterprisePool), param)
}

// DeletePool mocks base method.
func (m *MockPoolClient) DeletePool(param *pools.DeletePoolParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeletePool", param)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeletePool indicates an expected call of DeletePool.
func (mr *MockPoolClientMockRecorder) DeletePool(param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeletePool", reflect.TypeOf((*MockPoolClient)(nil).DeletePool), param)
}

// GetEnterprisePool mocks base method.
func (m *MockPoolClient) GetEnterprisePool(param *enterprises.GetEnterprisePoolParams) (*enterprises.GetEnterprisePoolOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEnterprisePool", param)
	ret0, _ := ret[0].(*enterprises.GetEnterprisePoolOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEnterprisePool indicates an expected call of GetEnterprisePool.
func (mr *MockPoolClientMockRecorder) GetEnterprisePool(param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEnterprisePool", reflect.TypeOf((*MockPoolClient)(nil).GetEnterprisePool), param)
}

// GetPool mocks base method.
func (m *MockPoolClient) GetPool(param *pools.GetPoolParams) (*pools.GetPoolOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPool", param)
	ret0, _ := ret[0].(*pools.GetPoolOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPool indicates an expected call of GetPool.
func (mr *MockPoolClientMockRecorder) GetPool(param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPool", reflect.TypeOf((*MockPoolClient)(nil).GetPool), param)
}

// ListAllPools mocks base method.
func (m *MockPoolClient) ListAllPools(param *pools.ListPoolsParams) (*pools.ListPoolsOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAllPools", param)
	ret0, _ := ret[0].(*pools.ListPoolsOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAllPools indicates an expected call of ListAllPools.
func (mr *MockPoolClientMockRecorder) ListAllPools(param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAllPools", reflect.TypeOf((*MockPoolClient)(nil).ListAllPools), param)
}

// UpdateEnterprisePool mocks base method.
func (m *MockPoolClient) UpdateEnterprisePool(param *enterprises.UpdateEnterprisePoolParams) (*enterprises.UpdateEnterprisePoolOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateEnterprisePool", param)
	ret0, _ := ret[0].(*enterprises.UpdateEnterprisePoolOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateEnterprisePool indicates an expected call of UpdateEnterprisePool.
func (mr *MockPoolClientMockRecorder) UpdateEnterprisePool(param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateEnterprisePool", reflect.TypeOf((*MockPoolClient)(nil).UpdateEnterprisePool), param)
}

// UpdatePool mocks base method.
func (m *MockPoolClient) UpdatePool(param *pools.UpdatePoolParams) (*pools.UpdatePoolOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdatePool", param)
	ret0, _ := ret[0].(*pools.UpdatePoolOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdatePool indicates an expected call of UpdatePool.
func (mr *MockPoolClientMockRecorder) UpdatePool(param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdatePool", reflect.TypeOf((*MockPoolClient)(nil).UpdatePool), param)
}
