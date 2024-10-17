// SPDX-License-Identifier: MIT
package controller

import (
	"context"
	"reflect"
	"testing"

	"github.com/cloudbase/garm/client/controller_info"
	"github.com/cloudbase/garm/params"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	"github.com/mercedes-benz/garm-operator/pkg/client/mock"
	"github.com/mercedes-benz/garm-operator/pkg/conditions"
)

var controllerID = uuid.New()

func TestGarmServerConfig_reconcile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            *garmoperatorv1beta1.GarmServerConfig
		expectGarmRequest func(m *mock.MockControllerClientMockRecorder)
		runtimeObjects    []runtime.Object
		wantErr           bool
		expectedObject    *garmoperatorv1beta1.GarmServerConfig
	}{
		{
			name: "sync controller info to GarmServerConfig",
			object: &garmoperatorv1beta1.GarmServerConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "garm-server-config",
					Namespace: "default",
				},
				Spec: garmoperatorv1beta1.GarmServerConfigSpec{
					MetadataURL: "http://garm-server.garm-server.svc:9997/api/v1/metadata",
					CallbackURL: "http://garm-server.garm-server.svc:9997/api/v1/callbacks",
					WebhookURL:  "http://garm-server.garm-server.svc:9997/api/v1/webhook",
				},
			},
			expectedObject: &garmoperatorv1beta1.GarmServerConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "garm-server-config",
					Namespace: "default",
				},
				Spec: garmoperatorv1beta1.GarmServerConfigSpec{
					MetadataURL: "http://garm-server.garm-server.svc:9997/api/v1/metadata",
					CallbackURL: "http://garm-server.garm-server.svc:9997/api/v1/callbacks",
					WebhookURL:  "http://garm-server.garm-server.svc:9997/api/v1/webhook",
				},
				Status: garmoperatorv1beta1.GarmServerConfigStatus{
					ControllerID:         controllerID.String(),
					Hostname:             "garm.server.com",
					MetadataURL:          "http://garm-server.garm-server.svc:9997/api/v1/metadata",
					CallbackURL:          "http://garm-server.garm-server.svc:9997/api/v1/callbacks",
					WebhookURL:           "http://garm-server.garm-server.svc:9997/api/v1/webhook",
					ControllerWebhookURL: " http://garm-server.garm-server.svc:9997/api/v1/webhook/BE4B3620-D424-43AC-8EDD-5760DBD516BF",
					MinimumJobAgeBackoff: 30,
					Version:              "v0.1.5",
				},
			},
			runtimeObjects: []runtime.Object{},
			wantErr:        false,
			expectGarmRequest: func(m *mock.MockControllerClientMockRecorder) {
				m.GetControllerInfo().Return(&controller_info.ControllerInfoOK{Payload: params.ControllerInfo{
					ControllerID:         controllerID,
					Hostname:             "garm.server.com",
					MetadataURL:          "http://garm-server.garm-server.svc:9997/api/v1/metadata",
					CallbackURL:          "http://garm-server.garm-server.svc:9997/api/v1/callbacks",
					WebhookURL:           "http://garm-server.garm-server.svc:9997/api/v1/webhook",
					ControllerWebhookURL: " http://garm-server.garm-server.svc:9997/api/v1/webhook/BE4B3620-D424-43AC-8EDD-5760DBD516BF",
					MinimumJobAgeBackoff: 30,
					Version:              "v0.1.5",
				}}, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemeBuilder := runtime.SchemeBuilder{
				garmoperatorv1beta1.AddToScheme,
			}

			err := schemeBuilder.AddToScheme(scheme.Scheme)
			if err != nil {
				t.Fatal(err)
			}
			runtimeObjects := []runtime.Object{tt.object}
			runtimeObjects = append(runtimeObjects, tt.runtimeObjects...)
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1beta1.GarmServerConfig{}).Build()

			// create a fake reconciler
			reconciler := &GarmServerConfigReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			garmServerConfig := tt.object.DeepCopyObject().(*garmoperatorv1beta1.GarmServerConfig)

			mockController := mock.NewMockControllerClient(mockCtrl)
			tt.expectGarmRequest(mockController.EXPECT())

			_, err = reconciler.reconcileNormal(context.Background(), mockController, garmServerConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("GarmServerConfigReconciler.reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// clear out annotations to avoid comparison errors
			garmServerConfig.ObjectMeta.Annotations = nil

			// empty resource version to avoid comparison errors
			garmServerConfig.ObjectMeta.ResourceVersion = ""

			// clear conditions lastTransitionTime to avoid comparison errors
			conditions.NilLastTransitionTime(tt.expectedObject)
			conditions.NilLastTransitionTime(garmServerConfig)

			if !reflect.DeepEqual(garmServerConfig, tt.expectedObject) {
				t.Errorf("GarmServerConfigReconciler.reconcileNormal() \ngot = %#v\n want %#v", garmServerConfig, tt.expectedObject)
			}
		})
	}
}
