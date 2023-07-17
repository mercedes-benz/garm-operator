/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"testing"

	garmoperatorv1alpha1 "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/v1alpha1"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/key"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/mock"
	"github.com/cloudbase/garm/params"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestEnterpriseReconciler_Reconcile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		expectGarmRequest func(m *mock.MockEnterpriseClientMockRecorder)
		wantErr           bool
	}{
		{
			name: "enterprise exist - no op",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "existing-enterprise",
					Namespace: "default",
					Finalizers: []string{
						key.EnterpriseFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "foobar",
					WebhookSecret:   "foobar",
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
				},
			},
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.GetEnterprise("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").Return(params.Enterprise{
					ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
					Name:            "another-non-operator-managed-enterprise",
					CredentialsName: "existing-enterprise",
				}, nil)
			},
		},
		{
			name: "enterprise does not exist - create",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "new-enterprise",
					Namespace: "default",
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "foobar",
					WebhookSecret:   "foobar",
				},
			},
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.ListEnterprises().Return([]params.Enterprise{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "another-non-operator-managed-enterprise",
						CredentialsName: "totally-unsecure",
					},
				}, nil)

				m.CreateEnterprise(params.CreateEnterpriseParams{
					Name:            "new-enterprise",
					CredentialsName: "foobar",
					WebhookSecret:   "foobar",
				}).Return(params.Enterprise{
					Name:            "new-enterprise",
					CredentialsName: "foobar",
					WebhookSecret:   "foobar",
					ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
				}, nil)
			},
		},
		{
			name: "enterprise exist in garm - adopt",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "new-enterprise",
					Namespace: "default",
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "foobar",
					WebhookSecret:   "foobar",
				},
			},
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.ListEnterprises().Return([]params.Enterprise{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "new-enterprise",
						CredentialsName: "totally-unsecure",
					},
				}, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			schemeBuilder := runtime.SchemeBuilder{
				garmoperatorv1alpha1.AddToScheme,
			}

			err := schemeBuilder.AddToScheme(scheme.Scheme)
			if err != nil {
				t.Fatal(err)
			}
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(tt.object).Build()

			// create a fake reconciler
			reconciler := &EnterpriseReconciler{
				Client:   client,
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
			}

			enterprise := tt.object.DeepCopyObject().(*garmoperatorv1alpha1.Enterprise)

			mockEnterprise := mock.NewMockEnterpriseClient(mockCtrl)
			tt.expectGarmRequest(mockEnterprise.EXPECT())

			_, err = reconciler.reconcileNormal(context.Background(), mockEnterprise, enterprise)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnterpriseReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			//check for mandatory finalizer
			if !controllerutil.ContainsFinalizer(enterprise, key.EnterpriseFinalizerName) {
				t.Errorf("EnterpriseReconciler.Reconcile() finalizer not found")
				return
			}

		})
	}
}
