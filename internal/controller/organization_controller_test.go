// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"reflect"
	"testing"
	"time"

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

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/client/mock"
	"github.com/mercedes-benz/garm-operator/pkg/util/conditions"
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
			name: "organization exist - update",
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Message:            "",
						},
						{
							Type:               string(conditions.PoolManager),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.SecretReference),
							Reason:             string(conditions.FetchingSecretRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.ListOrganizations(organizations.NewListOrgsParams()).Return(&organizations.ListOrgsOK{Payload: params.Organizations{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.UpdateOrganization(organizations.NewUpdateOrgParams().
					WithOrgID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&organizations.UpdateOrgOK{
					Payload: params.Organization{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "organization exist but spec has changed - update",
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.PoolManager),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.SecretReference),
							Status:             metav1.ConditionTrue,
							Message:            "",
							Reason:             string(conditions.FetchingSecretRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.ListOrganizations(organizations.NewListOrgsParams()).Return(&organizations.ListOrgsOK{Payload: params.Organizations{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.UpdateOrganization(organizations.NewUpdateOrgParams().
					WithOrgID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "has-changed",
						WebhookSecret:   "has-changed",
					})).Return(&organizations.UpdateOrgOK{
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
			name: "organization exist but pool status has changed - update",
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.PoolManager),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "no resources available",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.SecretReference),
							Status:             metav1.ConditionTrue,
							Message:            "",
							Reason:             string(conditions.FetchingSecretRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.ListOrganizations(organizations.NewListOrgsParams()).Return(&organizations.ListOrgsOK{Payload: params.Organizations{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.UpdateOrganization(organizations.NewUpdateOrgParams().
					WithOrgID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&organizations.UpdateOrgOK{
					Payload: params.Organization{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
						PoolManagerStatus: params.PoolManagerStatus{
							IsRunning:     false,
							FailureReason: "no resources available",
						},
					},
				}, nil)
			},
		},
		{
			name: "organization does not exist - create and update",
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.PoolManager),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.SecretReference),
							Reason:             string(conditions.FetchingSecretRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.ListOrganizations(organizations.NewListOrgsParams()).Return(&organizations.ListOrgsOK{Payload: params.Organizations{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "another-non-operator-managed-organization",
						CredentialsName: "totally-unsecure",
						WebhookSecret:   "foobar",
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
				m.UpdateOrganization(organizations.NewUpdateOrgParams().
					WithOrgID("9e0da3cb-130b-428d-aa8a-e314d955060e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&organizations.UpdateOrgOK{
					Payload: params.Organization{
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
						Name:            "new-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "organization already exist in garm - update",
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
					ID: "e1dbf9a6-a9f6-4594-a5ac-12345",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.PoolManager),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.SecretReference),
							Reason:             string(conditions.FetchingSecretRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
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
				m.UpdateOrganization(organizations.NewUpdateOrgParams().
					WithOrgID("e1dbf9a6-a9f6-4594-a5ac-12345").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&organizations.UpdateOrgOK{
					Payload: params.Organization{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "new-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "organization does not exist in garm - create and update",
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
					ID: "9e0da3cb-130b-428d-aa8a-e314d955060e",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.PoolManager),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.SecretReference),
							Reason:             string(conditions.FetchingSecretRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {
				m.ListOrganizations(organizations.NewListOrgsParams()).Return(&organizations.ListOrgsOK{Payload: params.Organizations{
					{},
				}}, nil)
				m.CreateOrganization(organizations.NewCreateOrgParams().WithBody(
					params.CreateOrgParams{
						Name:            "existing-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&organizations.CreateOrgOK{
					Payload: params.Organization{
						Name:            "existing-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
					},
				}, nil)
				m.UpdateOrganization(organizations.NewUpdateOrgParams().
					WithOrgID("9e0da3cb-130b-428d-aa8a-e314d955060e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&organizations.UpdateOrgOK{
					Payload: params.Organization{
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
						Name:            "existing-organization",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "secret ref not found condition",
			object: &garmoperatorv1alpha1.Organization{
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
				Status: garmoperatorv1alpha1.OrganizationStatus{},
			},
			runtimeObjects: []runtime.Object{},
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.ReconcileErrorReason),
							Status:             metav1.ConditionFalse,
							Message:            "secrets \"my-webhook-secret\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.SecretReference),
							Reason:             string(conditions.FetchingSecretRefFailedReason),
							Status:             metav1.ConditionFalse,
							Message:            "secrets \"my-webhook-secret\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockOrganizationClientMockRecorder) {},
			wantErr:           true,
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

			// clear out annotations to avoid comparison errors
			organization.ObjectMeta.Annotations = nil

			// empty resource version to avoid comparison errors
			organization.ObjectMeta.ResourceVersion = ""

			// clear conditions lastTransitionTime to avoid comparison errors
			conditions.NilLastTransitionTime(tt.expectedObject)
			conditions.NilLastTransitionTime(organization)

			if !reflect.DeepEqual(organization, tt.expectedObject) {
				t.Errorf("OrganizationReconciler.reconcileNormal() \ngot = %#v\n want %#v", organization, tt.expectedObject)
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
