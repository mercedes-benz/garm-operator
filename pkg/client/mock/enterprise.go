// SPDX-License-Identifier: MIT
// Code generated by MockGen. DO NOT EDIT.
// Source: ../enterprise.go
//
// Generated by this command:
//
//	mockgen -package mock -destination=enterprise.go -source=../enterprise.go Enterprise
//

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	enterprises "github.com/cloudbase/garm/client/enterprises"
	gomock "go.uber.org/mock/gomock"
)

// MockEnterpriseClient is a mock of EnterpriseClient interface.
type MockEnterpriseClient struct {
	ctrl     *gomock.Controller
	recorder *MockEnterpriseClientMockRecorder
}

// MockEnterpriseClientMockRecorder is the mock recorder for MockEnterpriseClient.
type MockEnterpriseClientMockRecorder struct {
	mock *MockEnterpriseClient
}

// NewMockEnterpriseClient creates a new mock instance.
func NewMockEnterpriseClient(ctrl *gomock.Controller) *MockEnterpriseClient {
	mock := &MockEnterpriseClient{ctrl: ctrl}
	mock.recorder = &MockEnterpriseClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEnterpriseClient) EXPECT() *MockEnterpriseClientMockRecorder {
	return m.recorder
}

// CreateEnterprise mocks base method.
func (m *MockEnterpriseClient) CreateEnterprise(param *enterprises.CreateEnterpriseParams) (*enterprises.CreateEnterpriseOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateEnterprise", param)
	ret0, _ := ret[0].(*enterprises.CreateEnterpriseOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateEnterprise indicates an expected call of CreateEnterprise.
func (mr *MockEnterpriseClientMockRecorder) CreateEnterprise(param any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateEnterprise", reflect.TypeOf((*MockEnterpriseClient)(nil).CreateEnterprise), param)
}

// DeleteEnterprise mocks base method.
func (m *MockEnterpriseClient) DeleteEnterprise(param *enterprises.DeleteEnterpriseParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteEnterprise", param)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteEnterprise indicates an expected call of DeleteEnterprise.
func (mr *MockEnterpriseClientMockRecorder) DeleteEnterprise(param any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteEnterprise", reflect.TypeOf((*MockEnterpriseClient)(nil).DeleteEnterprise), param)
}

// GetEnterprise mocks base method.
func (m *MockEnterpriseClient) GetEnterprise(param *enterprises.GetEnterpriseParams) (*enterprises.GetEnterpriseOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEnterprise", param)
	ret0, _ := ret[0].(*enterprises.GetEnterpriseOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEnterprise indicates an expected call of GetEnterprise.
func (mr *MockEnterpriseClientMockRecorder) GetEnterprise(param any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEnterprise", reflect.TypeOf((*MockEnterpriseClient)(nil).GetEnterprise), param)
}

// ListEnterprises mocks base method.
func (m *MockEnterpriseClient) ListEnterprises(param *enterprises.ListEnterprisesParams) (*enterprises.ListEnterprisesOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEnterprises", param)
	ret0, _ := ret[0].(*enterprises.ListEnterprisesOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEnterprises indicates an expected call of ListEnterprises.
func (mr *MockEnterpriseClientMockRecorder) ListEnterprises(param any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEnterprises", reflect.TypeOf((*MockEnterpriseClient)(nil).ListEnterprises), param)
}

// UpdateEnterprise mocks base method.
func (m *MockEnterpriseClient) UpdateEnterprise(param *enterprises.UpdateEnterpriseParams) (*enterprises.UpdateEnterpriseOK, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateEnterprise", param)
	ret0, _ := ret[0].(*enterprises.UpdateEnterpriseOK)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateEnterprise indicates an expected call of UpdateEnterprise.
func (mr *MockEnterpriseClientMockRecorder) UpdateEnterprise(param any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateEnterprise", reflect.TypeOf((*MockEnterpriseClient)(nil).UpdateEnterprise), param)
}
