// SPDX-License-Identifier: MIT

package client

import (
	"testing"

	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/params"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/mercedes-benz/garm-operator/pkg/client/mock"
	"github.com/mercedes-benz/garm-operator/pkg/metrics"
)

func TestGarmClient_Reauthenticate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectGarmRequest := func(m *mock.MockEnterpriseClientMockRecorder, m1 *mock.MockGarmClientMockRecorder) {
		m.GetEnterprise(enterprises.NewGetEnterpriseParams().
			WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e"),
		).Return(nil, instances.NewGetInstanceDefault(401))

		m1.Init().Return(nil)
		m1.Login().Return(nil)

		m.GetEnterprise(
			enterprises.NewGetEnterpriseParams().
				WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e"),
		).Return(&enterprises.GetEnterpriseOK{
			Payload: params.Enterprise{
				ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
				Name:            "existing-enterprise",
				CredentialsName: "foobar",
			},
		}, nil)
	}

	expectedResult := &enterprises.GetEnterpriseOK{
		Payload: params.Enterprise{
			ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
			Name:            "existing-enterprise",
			CredentialsName: "foobar",
		},
	}

	mockEnterpriseClient := mock.NewMockEnterpriseClient(mockCtrl)
	mockBaseClient := mock.NewMockGarmClient(mockCtrl)
	Client = mockBaseClient

	expectGarmRequest(mockEnterpriseClient.EXPECT(), mockBaseClient.EXPECT())

	result, err := EnsureAuth[*enterprises.GetEnterpriseOK](func() (*enterprises.GetEnterpriseOK, error) {
		metrics.TotalGarmCalls.WithLabelValues("enterprises.Get").Inc()

		enterprise, err := mockEnterpriseClient.GetEnterprise(enterprises.NewGetEnterpriseParams().WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e"))
		if err != nil {
			metrics.GarmCallErrors.WithLabelValues("enterprises.Get").Inc()
			return nil, err
		}
		return enterprise, nil
	})

	assert.Equal(t, expectedResult, result)
	assert.Equal(t, nil, err)
}
