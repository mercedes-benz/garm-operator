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
	"reflect"
	"testing"

	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/params"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	garmoperatorv1alpha1 "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/v1alpha1"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/key"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/mock"
)

func TestOrganizationReconciler_reconcileNormal(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		runtimeObjects    []runtime.Object
		expectGarmRequest func(m *mock.MockOrganizationClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1alpha1.Organization
	}{
		{
			name: "organization exist - no op",
			object: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-organization",
					Namespace: "default",
					Finalizers: []string{
						key.OrganizationFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "foobar",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.OrganizationStatus{
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
			expectedObject: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-organization",
					Namespace: "default",
					Finalizers: []string{
						key.OrganizationFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "foobar",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.OrganizationStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.GetOrganization(
					organizations.NewGetOrgParams().
						WithOrgID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e"),
				).Return(&organizations.GetOrgOK{
					Payload: params.Organization{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-organization",
						CredentialsName: "foobar",
					},
				}, nil)
			},
		},
		{
			name: "organization exist - but spec has changed",
			object: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-organization",
					Namespace: "default",
					Finalizers: []string{
						key.OrganizationFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "has-changed",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.OrganizationStatus{
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
			expectedObject: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-organization",
					Namespace: "default",
					Finalizers: []string{
						key.OrganizationFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "has-changed",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.OrganizationStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.GetOrganization(
					organizations.NewGetOrgParams().
						WithOrgID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e"),
				).Return(&organizations.GetOrgOK{
					Payload: params.Organization{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}, nil)

				m.UpdateOrganization(organizations.NewUpdateOrgParams().
					WithOrgID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").WithBody(params.UpdateEntityParams{
					CredentialsName: "has-changed",
					WebhookSecret:   "has-changed",
				}),
				).Return(&organizations.UpdateOrgOK{
					Payload: params.Organization{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-organization",
						CredentialsName: "has-changed",
						WebhookSecret:   "has-changed",
					},
				}, nil)
			},
		},
		{
			name: "organization exist but pool status has changed - updating status",
			object: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-organization",
					Namespace: "default",
					Finalizers: []string{
						key.OrganizationFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "foobar",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.OrganizationStatus{
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
			expectedObject: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-organization",
					Namespace: "default",
					Finalizers: []string{
						key.OrganizationFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "foobar",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.OrganizationStatus{
					ID:                       "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
					PoolManagerIsRunning:     false,
					PoolManagerFailureReason: "no resources available",
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.GetOrganization(
					organizations.NewGetOrgParams().
						WithOrgID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e"),
				).Return(&organizations.GetOrgOK{
					Payload: params.Organization{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-organization",
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
			name: "organization does not exist - create",
			object: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-organization",
					Namespace: "default",
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "foobar",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
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
			expectedObject: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-organization",
					Namespace: "default",
					Finalizers: []string{
						key.OrganizationFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "foobar",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.OrganizationStatus{
					ID: "9e0da3cb-130b-428d-aa8a-e314d955060e",
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.ListOrganizations(organizations.NewListOrgsParams()).Return(&organizations.ListOrgsOK{Payload: params.Organizations{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "another-non-operator-managed-organization",
						CredentialsName: "totally-unsecure",
					},
				}}, nil)

				m.CreateOrganization(organizations.NewCreateOrgParams().WithBody(
					params.CreateOrgParams{
						Name:            "new-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&organizations.CreateOrgOK{
					Payload: params.Organization{
						Name:            "new-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
					},
				}, nil)
			},
		},
		{
			name: "organization already exist in garm - adopt",
			object: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-organization",
					Namespace: "default",
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "totally-insecure",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
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
			expectedObject: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-organization",
					Namespace: "default",
					Finalizers: []string{
						key.OrganizationFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "totally-insecure",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.OrganizationStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-12345",
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.ListOrganizations(organizations.NewListOrgsParams()).Return(&organizations.ListOrgsOK{Payload: params.Organizations{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "new-organization",
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1alpha1.Organization{}).Build()

			// create a fake reconciler
			reconciler := &OrganizationReconciler{
				Client:   client,
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
				Recorder: record.NewFakeRecorder(3),
			}

			organization := tt.object.DeepCopyObject().(*garmoperatorv1alpha1.Organization)

			mockOrganization := mock.NewMockOrganizationClient(mockCtrl)
			tt.expectGarmRequest(mockOrganization.EXPECT())

			_, err = reconciler.reconcileNormal(context.Background(), mockOrganization, organization)
			if (err != nil) != tt.wantErr {
				t.Errorf("OrganizationReconciler.reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// empty resource version to avoid comparison errors
			organization.ObjectMeta.ResourceVersion = ""
			if !reflect.DeepEqual(organization, tt.expectedObject) {
				t.Errorf("OrganizationReconciler.reconcileNormal() got = %#v, want %#v", organization, tt.expectedObject)
			}
		})
	}
}

func TestOrganizationReconciler_reconcileDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		runtimeObjects    []runtime.Object
		expectGarmRequest func(m *mock.MockOrganizationClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1alpha1.Organization
	}{
		{
			name: "delete organization",
			object: &garmoperatorv1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "delete-organization",
					Namespace: "default",
					Finalizers: []string{
						key.OrganizationFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.OrganizationSpec{
					CredentialsName: "totally-insecure",
					WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1alpha1.OrganizationStatus{
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
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.DeleteOrganization(
					organizations.NewDeleteOrgParams().
						WithOrgID("e1dbf9a6-a9f6-4594-a5ac-12345"),
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1alpha1.Organization{}).Build()

			// create a fake reconciler
			reconciler := &OrganizationReconciler{
				Client:   client,
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
				Recorder: record.NewFakeRecorder(3),
			}

			organization := tt.object.DeepCopyObject().(*garmoperatorv1alpha1.Organization)

			mockOrganization := mock.NewMockOrganizationClient(mockCtrl)
			tt.expectGarmRequest(mockOrganization.EXPECT())

			_, err = reconciler.reconcileDelete(context.Background(), mockOrganization, organization)
			if (err != nil) != tt.wantErr {
				t.Errorf("OrganizationReconciler.reconcileDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// check for mandatory finalizer
			if controllerutil.ContainsFinalizer(organization, key.OrganizationFinalizerName) {
				t.Errorf("OrganizationReconciler.Reconcile() finalizer still exist")
				return
			}
		})
	}
}
