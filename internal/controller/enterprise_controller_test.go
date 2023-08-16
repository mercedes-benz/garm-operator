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
	"k8s.io/client-go/tools/record"
	"reflect"
	"testing"

	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/params"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	garmoperatorv1alpha1 "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/v1alpha1"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/key"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/mock"
)

func TestEnterpriseReconciler_reconcileNormal(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		expectGarmRequest func(m *mock.MockEnterpriseClientMockRecorder)
		runtimeObjects    []runtime.Object
		wantErr           bool
		expectedObject    *garmoperatorv1alpha1.Enterprise
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
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("foobar"),
					},
				},
			},
			expectedObject: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "existing-enterprise",
					Namespace: "default",
					Finalizers: []string{
						key.EnterpriseFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "foobar",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
				},
			},
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
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
			},
		},
		{
			name: "enterprise exist - but spec has changed",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "existing-enterprise",
					Namespace: "default",
					Finalizers: []string{
						key.EnterpriseFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "has-changed",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("has-changed"),
					},
				},
			},
			expectedObject: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "existing-enterprise",
					Namespace: "default",
					Finalizers: []string{
						key.EnterpriseFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "has-changed",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
				},
			},
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.GetEnterprise(
					enterprises.NewGetEnterpriseParams().
						WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e"),
				).Return(&enterprises.GetEnterpriseOK{
					Payload: params.Enterprise{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}, nil)

				m.UpdateEnterprise(enterprises.NewUpdateEnterpriseParams().
					WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").WithBody(params.UpdateEntityParams{
					CredentialsName: "has-changed",
					WebhookSecret:   "has-changed",
				}),
				).Return(&enterprises.UpdateEnterpriseOK{
					Payload: params.Enterprise{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-enterprise",
						CredentialsName: "has-changed",
						WebhookSecret:   "has-changed",
					},
				}, nil)
			},
		},
		{
			name: "enterprise exist but pool status has changed - updating status",
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
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
				},
			},
			expectedObject: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "existing-enterprise",
					Namespace: "default",
					Finalizers: []string{
						key.EnterpriseFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "foobar",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID:                       "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
					PoolManagerIsRunning:     false,
					PoolManagerFailureReason: "no resources available",
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("foobar"),
					},
				},
			},
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.GetEnterprise(
					enterprises.NewGetEnterpriseParams().
						WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e"),
				).Return(&enterprises.GetEnterpriseOK{
					Payload: params.Enterprise{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-enterprise",
						CredentialsName: "foobar",
						PoolManagerStatus: params.PoolManagerStatus{
							IsRunning:     false,
							FailureReason: "no resources available",
						},
					},
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
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
			},
			expectedObject: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "new-enterprise",
					Namespace: "default",
					Finalizers: []string{
						key.EnterpriseFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "foobar",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID: "9e0da3cb-130b-428d-aa8a-e314d955060e",
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("foobar"),
					},
				},
			},
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.ListEnterprises(enterprises.NewListEnterprisesParams()).Return(&enterprises.ListEnterprisesOK{Payload: params.Enterprises{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "another-non-operator-managed-enterprise",
						CredentialsName: "totally-unsecure",
					},
				}}, nil)

				m.CreateEnterprise(enterprises.NewCreateEnterpriseParams().WithBody(
					params.CreateEnterpriseParams{
						Name:            "new-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&enterprises.CreateEnterpriseOK{
					Payload: params.Enterprise{
						Name:            "new-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
					},
				}, nil)
			},
		},
		{
			name: "enterprise already exist in garm - adopt",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "new-enterprise",
					Namespace: "default",
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "totally-insecure",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
			},
			expectedObject: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "new-enterprise",
					Namespace: "default",
					Finalizers: []string{
						key.EnterpriseFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "totally-insecure",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-12345",
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("foobar"),
					},
				},
			},
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.ListEnterprises(enterprises.NewListEnterprisesParams()).Return(&enterprises.ListEnterprisesOK{Payload: params.Enterprises{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "new-enterprise",
						CredentialsName: "totally-insecure",
					},
				}}, nil)
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
			runtimeObjects := []runtime.Object{tt.object}
			runtimeObjects = append(runtimeObjects, tt.runtimeObjects...)
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1alpha1.Enterprise{}).Build()

			// create a fake reconciler
			reconciler := &EnterpriseReconciler{
				Client:   client,
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
				Recorder: record.NewFakeRecorder(3),
			}

			enterprise := tt.object.DeepCopyObject().(*garmoperatorv1alpha1.Enterprise)

			mockEnterprise := mock.NewMockEnterpriseClient(mockCtrl)
			tt.expectGarmRequest(mockEnterprise.EXPECT())

			_, err = reconciler.reconcileNormal(context.Background(), mockEnterprise, enterprise)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnterpriseReconciler.reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// empty resource version to avoid comparison errors
			enterprise.ObjectMeta.ResourceVersion = ""
			if !reflect.DeepEqual(enterprise, tt.expectedObject) {
				t.Errorf("EnterpriseReconciler.reconcileNormal() got = %#v, want %#v", enterprise, tt.expectedObject)
			}
		})
	}
}

func TestEnterpriseReconciler_reconcileDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		runtimeObjects    []runtime.Object
		expectGarmRequest func(m *mock.MockEnterpriseClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1alpha1.Enterprise
	}{
		{
			name: "delete enterprise",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: v1.ObjectMeta{
					Name:      "delete-enterprise",
					Namespace: "default",
					Finalizers: []string{
						key.EnterpriseFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.EnterpriseSpec{
					CredentialsName: "totally-insecure",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-12345",
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("foobar"),
					},
				},
			},
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.DeleteEnterprise(
					enterprises.NewDeleteEnterpriseParams().
						WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-12345"),
				).Return(nil)
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

			runtimeObjects := []runtime.Object{tt.object}
			runtimeObjects = append(runtimeObjects, tt.runtimeObjects...)
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1alpha1.Enterprise{}).Build()

			// create a fake reconciler
			reconciler := &EnterpriseReconciler{
				Client:   client,
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
				Recorder: record.NewFakeRecorder(3),
			}

			enterprise := tt.object.DeepCopyObject().(*garmoperatorv1alpha1.Enterprise)

			mockEnterprise := mock.NewMockEnterpriseClient(mockCtrl)
			tt.expectGarmRequest(mockEnterprise.EXPECT())

			_, err = reconciler.reconcileDelete(context.Background(), mockEnterprise, enterprise)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnterpriseReconciler.reconcileDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// check for mandatory finalizer
			if controllerutil.ContainsFinalizer(enterprise, key.EnterpriseFinalizerName) {
				t.Errorf("EnterpriseReconciler.Reconcile() finalizer still exist")
				return
			}
		})
	}
}
