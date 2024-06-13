// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/cloudbase/garm/client/enterprises"
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
			name: "enterprise exist - update",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: metav1.ObjectMeta{
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
				ObjectMeta: metav1.ObjectMeta{
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "Pool Manager is not running",
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
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.ListEnterprises(enterprises.NewListEnterprisesParams()).Return(&enterprises.ListEnterprisesOK{Payload: params.Enterprises{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.UpdateEnterprise(enterprises.NewUpdateEnterpriseParams().
					WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&enterprises.UpdateEnterpriseOK{
					Payload: params.Enterprise{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "enterprise exist but spec has changed - update",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: metav1.ObjectMeta{
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
				ObjectMeta: metav1.ObjectMeta{
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "Pool Manager is not running",
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
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.ListEnterprises(enterprises.NewListEnterprisesParams()).Return(&enterprises.ListEnterprisesOK{Payload: params.Enterprises{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.UpdateEnterprise(enterprises.NewUpdateEnterpriseParams().
					WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "has-changed",
						WebhookSecret:   "has-changed",
					})).Return(&enterprises.UpdateEnterpriseOK{
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
			name: "enterprise exist but pool status has changed - update",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: metav1.ObjectMeta{
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
				ObjectMeta: metav1.ObjectMeta{
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "Pool Manager is not running",
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
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.UpdateEnterprise(enterprises.NewUpdateEnterpriseParams().
					WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&enterprises.UpdateEnterpriseOK{
					Payload: params.Enterprise{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-enterprise",
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
			name: "enterprise does not exist - create and update",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: metav1.ObjectMeta{
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
				ObjectMeta: metav1.ObjectMeta{
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "Pool Manager is not running",
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
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.CreateEnterprise(enterprises.NewCreateEnterpriseParams().WithBody(
					params.CreateEnterpriseParams{
						Name:            "new-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&enterprises.CreateEnterpriseOK{
					Payload: params.Enterprise{
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
						Name:            "new-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}, nil)
				m.UpdateEnterprise(enterprises.NewUpdateEnterpriseParams().
					WithEnterpriseID("9e0da3cb-130b-428d-aa8a-e314d955060e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&enterprises.UpdateEnterpriseOK{
					Payload: params.Enterprise{
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
						Name:            "new-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "enterprise already exist in garm - update",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: metav1.ObjectMeta{
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
				ObjectMeta: metav1.ObjectMeta{
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "Pool Manager is not running",
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
				m.UpdateEnterprise(enterprises.NewUpdateEnterpriseParams().
					WithEnterpriseID("e1dbf9a6-a9f6-4594-a5ac-12345").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "totally-insecure",
						WebhookSecret:   "foobar",
					})).Return(&enterprises.UpdateEnterpriseOK{
					Payload: params.Enterprise{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "new-enterprise",
						CredentialsName: "totally-insecure",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "enterprise does not exist in garm - create update",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: metav1.ObjectMeta{
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
				ObjectMeta: metav1.ObjectMeta{
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
					ID: "9e0da3cb-130b-428d-aa8a-e314d955060e",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							Message:            "Pool Manager is not running",
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
			expectGarmRequest: func(m *mock.MockEnterpriseClientMockRecorder) {
				m.ListEnterprises(enterprises.NewListEnterprisesParams()).Return(&enterprises.ListEnterprisesOK{Payload: params.Enterprises{
					{},
				}}, nil)
				m.CreateEnterprise(enterprises.NewCreateEnterpriseParams().WithBody(
					params.CreateEnterpriseParams{
						Name:            "existing-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&enterprises.CreateEnterpriseOK{
					Payload: params.Enterprise{
						Name:            "existing-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
					},
				}, nil)
				m.UpdateEnterprise(enterprises.NewUpdateEnterpriseParams().
					WithEnterpriseID("9e0da3cb-130b-428d-aa8a-e314d955060e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					})).Return(&enterprises.UpdateEnterpriseOK{
					Payload: params.Enterprise{
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
						Name:            "existing-enterprise",
						CredentialsName: "foobar",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "secret ref not found condition",
			object: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: metav1.ObjectMeta{
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
				Status: garmoperatorv1alpha1.EnterpriseStatus{},
			},
			runtimeObjects: []runtime.Object{},
			expectedObject: &garmoperatorv1alpha1.Enterprise{
				ObjectMeta: metav1.ObjectMeta{
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
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.FetchingSecretRefFailedReason),
							Status:             metav1.ConditionFalse,
							Message:            "secrets \"my-webhook-secret\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.PoolManager),
							Reason:             string(conditions.UnknownReason),
							Status:             metav1.ConditionUnknown,
							Message:            "GARM server not reconciled yet",
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
			expectGarmRequest: func(_ *mock.MockEnterpriseClientMockRecorder) {},
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1alpha1.Enterprise{}).Build()

			// create a fake reconciler
			reconciler := &EnterpriseReconciler{
				Client:   client,
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

			// clear out annotations to avoid comparison errors
			enterprise.ObjectMeta.Annotations = nil

			// empty resource version to avoid comparison errors
			enterprise.ObjectMeta.ResourceVersion = ""

			// clear conditions lastTransitionTime to avoid comparison errors
			conditions.NilLastTransitionTime(tt.expectedObject)
			conditions.NilLastTransitionTime(enterprise)

			if !reflect.DeepEqual(enterprise, tt.expectedObject) {
				t.Errorf("EnterpriseReconciler.reconcileNormal() \ngot = %#v\n want %#v", enterprise, tt.expectedObject)
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
				ObjectMeta: metav1.ObjectMeta{
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
