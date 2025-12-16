// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	garmProviderParams "github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/client/pools"
	"github.com/cloudbase/garm/params"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/client/mock"
	"github.com/mercedes-benz/garm-operator/pkg/conditions"
	"github.com/mercedes-benz/garm-operator/pkg/config"
)

const namespaceName = "test-namespace"

func TestPoolController_ReconcileCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	poolID := "fb2bceeb-f74d-435d-9648-626c75cb23ce"
	enterpriseID := "93068607-2d0d-4b76-a950-0e40d31955b8"
	enterpriseName := "test-enterprise"

	tests := []struct {
		name string
		// the object to reconcile
		object client.Object
		// a list of objects to initialize the fake client with
		// this can be used to define other existing objects that are referenced by the object to reconcile
		// e.g. images or other pools ..
		runtimeObjects    []runtime.Object
		expectGarmRequest func(_ *mock.MockPoolClientMockRecorder, instanceClient *mock.MockInstanceClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1beta1.Pool
	}{
		{
			name: "pool does not exist in garm - create",
			object: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             5,
					MinIdleRunners:         3,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
			},
			expectedObject: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             5,
					MinIdleRunners:         3,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID:                     poolID,
					LongRunningIdleRunners: 3,
					Selector:               "",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Message:            "",
						},
						{
							Type:               string(conditions.ImageReference),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched Image CR Ref",
							Reason:             string(conditions.FetchingImageRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.ScopeReference),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched Enterprise CR Ref",
							Reason:             string(conditions.FetchingScopeRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespaceName,
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("supersecretvalue"),
					},
				},
				&garmoperatorv1beta1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1beta1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1beta1.GroupVersion.Group + "/" + garmoperatorv1beta1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.EnterpriseSpec{
						CredentialsRef: corev1.TypedLocalObjectReference{
							APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
							Kind:     "GitHubCredential",
							Name:     "github-creds",
						},
						WebhookSecretRef: garmoperatorv1beta1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1beta1.EnterpriseStatus{
						ID: enterpriseID,
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
						},
					},
				},
			},
			expectGarmRequest: func(poolClient *mock.MockPoolClientMockRecorder, _ *mock.MockInstanceClientMockRecorder) {
				extraSpecs := json.RawMessage([]byte{})
				poolClient.CreateEnterprisePool(
					enterprises.NewCreateEnterprisePoolParams().WithEnterpriseID(enterpriseID).WithBody(
						params.CreatePoolParams{
							RunnerPrefix: params.RunnerPrefix{
								Prefix: "",
							},
							ProviderName:           "kubernetes_external",
							MaxRunners:             5,
							MinIdleRunners:         3,
							Image:                  "linux-ubuntu-22.04-arm64",
							Flavor:                 "medium",
							OSType:                 "linux",
							OSArch:                 "arm64",
							Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
							Enabled:                true,
							RunnerBootstrapTimeout: 20,
							ExtraSpecs:             extraSpecs,
							GitHubRunnerGroup:      "",
						}),
				).Return(&enterprises.CreateEnterprisePoolOK{
					Payload: params.Pool{
						RunnerPrefix: params.RunnerPrefix{
							Prefix: "",
						},
						ID:             poolID,
						ProviderName:   "kubernetes_external",
						MaxRunners:     5,
						MinIdleRunners: 3,
						Image:          "linux-ubuntu-22.04-arm64",
						Flavor:         "medium",
						OSType:         "linux",
						OSArch:         "arm64",
						Tags: []params.Tag{
							{
								ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6da",
								Name: "kubernetes",
							},
							{
								ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6db",
								Name: "linux",
							},
							{
								ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dc",
								Name: "arm64",
							},
							{
								ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
								Name: "ubuntu",
							},
						},
						Enabled:        true,
						Instances:      []params.Instance{},
						RepoID:         "",
						RepoName:       "",
						OrgID:          "",
						OrgName:        "",
						EnterpriseID:   enterpriseID,
						EnterpriseName: enterpriseName,
					},
				}, nil)
			},
		},
		{
			name: "pool.Status has matching id in garm database, pool.Specs changed - update pool in garm",
			object: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             1,
					MinIdleRunners:         0,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID: poolID,
				},
			},
			expectedObject: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             1,
					MinIdleRunners:         0,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID: poolID,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Message:            "",
						},
						{
							Type:               string(conditions.ImageReference),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched Image CR Ref",
							Reason:             string(conditions.FetchingImageRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.ScopeReference),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched Enterprise CR Ref",
							Reason:             string(conditions.FetchingScopeRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespaceName,
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("supersecretvalue"),
					},
				},
				&garmoperatorv1beta1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1beta1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1beta1.GroupVersion.Group + "/" + garmoperatorv1beta1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.EnterpriseSpec{
						CredentialsRef: corev1.TypedLocalObjectReference{
							APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
							Kind:     "GitHubCredential",
							Name:     "github-creds",
						},
						WebhookSecretRef: garmoperatorv1beta1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1beta1.EnterpriseStatus{
						ID: enterpriseID,
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
						},
					},
				},
			},
			expectGarmRequest: func(poolClient *mock.MockPoolClientMockRecorder, _ *mock.MockInstanceClientMockRecorder) {
				poolClient.GetPool(pools.NewGetPoolParams().WithPoolID(poolID)).Return(&pools.GetPoolOK{Payload: params.Pool{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					ID:             poolID,
					ProviderName:   "kubernetes_external",
					MaxRunners:     5,
					MinIdleRunners: 3,
					Image:          "linux-ubuntu-22.04-arm64",
					Flavor:         "medium",
					OSType:         "linux",
					OSArch:         "arm64",
					Tags: []params.Tag{
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6da",
							Name: "kubernetes",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6db",
							Name: "linux",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dc",
							Name: "arm64",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name: "ubuntu",
						},
					},
					Enabled:        true,
					Instances:      []params.Instance{},
					RepoID:         "",
					RepoName:       "",
					OrgID:          "",
					OrgName:        "",
					EnterpriseID:   enterpriseID,
					EnterpriseName: enterpriseName,
				}}, nil)

				poolClient.GetPool(pools.NewGetPoolParams().WithPoolID(poolID)).Return(&pools.GetPoolOK{Payload: params.Pool{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					ID:             poolID,
					ProviderName:   "kubernetes_external",
					MaxRunners:     5,
					MinIdleRunners: 3,
					Image:          "linux-ubuntu-22.04-arm64",
					Flavor:         "medium",
					OSType:         "linux",
					OSArch:         "arm64",
					Tags: []params.Tag{
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6da",
							Name: "kubernetes",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6db",
							Name: "linux",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dc",
							Name: "arm64",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name: "ubuntu",
						},
					},
					Enabled:        true,
					Instances:      []params.Instance{},
					RepoID:         "",
					RepoName:       "",
					OrgID:          "",
					OrgName:        "",
					EnterpriseID:   enterpriseID,
					EnterpriseName: enterpriseName,
				}}, nil)

				maxRunners := uint(1)
				minIdleRunners := uint(0)
				enabled := true
				runnerBootstrapTimeout := uint(20)
				extraSpecs := json.RawMessage([]byte{})
				gitHubRunnerGroup := ""
				poolClient.UpdatePool(pools.NewUpdatePoolParams().WithPoolID(poolID).WithBody(params.UpdatePoolParams{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					MaxRunners:             &maxRunners,
					MinIdleRunners:         &minIdleRunners,
					Image:                  "linux-ubuntu-22.04-arm64",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                &enabled,
					RunnerBootstrapTimeout: &runnerBootstrapTimeout,
					ExtraSpecs:             extraSpecs,
					GitHubRunnerGroup:      &gitHubRunnerGroup,
				})).Return(&pools.UpdatePoolOK{Payload: params.Pool{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					ID:             poolID,
					ProviderName:   "kubernetes_external",
					MaxRunners:     1,
					MinIdleRunners: 0,
					Image:          "linux-ubuntu-22.04-arm64",
					Flavor:         "medium",
					OSType:         "linux",
					OSArch:         "arm64",
					Tags: []params.Tag{
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6da",
							Name: "kubernetes",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6db",
							Name: "linux",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dc",
							Name: "arm64",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name: "ubuntu",
						},
					},
					Enabled:        true,
					Instances:      []params.Instance{},
					RepoID:         "",
					RepoName:       "",
					OrgID:          "",
					OrgName:        "",
					EnterpriseID:   enterpriseID,
					EnterpriseName: enterpriseName,
				}}, nil)
			},
		},
		{
			name: "scaling idleRunners down to 2 - expect deletion of two old instances",
			object: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             10,
					MinIdleRunners:         2,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID:                     poolID,
					LongRunningIdleRunners: 3,
				},
			},
			expectedObject: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             10,
					MinIdleRunners:         2,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID:                     poolID,
					LongRunningIdleRunners: 2,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Message:            "",
						},
						{
							Type:               string(conditions.ImageReference),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched Image CR Ref",
							Reason:             string(conditions.FetchingImageRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.ScopeReference),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched Enterprise CR Ref",
							Reason:             string(conditions.FetchingScopeRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespaceName,
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("supersecretvalue"),
					},
				},
				&garmoperatorv1beta1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1beta1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1beta1.GroupVersion.Group + "/" + garmoperatorv1beta1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.EnterpriseSpec{
						CredentialsRef: corev1.TypedLocalObjectReference{
							APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
							Kind:     "GitHubCredential",
							Name:     "github-creds",
						},
						WebhookSecretRef: garmoperatorv1beta1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1beta1.EnterpriseStatus{
						ID: enterpriseID,
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
						},
					},
				},
			},
			expectGarmRequest: func(poolClient *mock.MockPoolClientMockRecorder, instanceClient *mock.MockInstanceClientMockRecorder) {
				poolClient.GetPool(pools.NewGetPoolParams().WithPoolID(poolID)).Return(&pools.GetPoolOK{Payload: params.Pool{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					ID:             poolID,
					ProviderName:   "kubernetes_external",
					MaxRunners:     10,
					MinIdleRunners: 5,
					Image:          "linux-ubuntu-22.04-arm64",
					Flavor:         "medium",
					OSType:         "linux",
					OSArch:         "arm64",
					Tags: []params.Tag{
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6da",
							Name: "kubernetes",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6db",
							Name: "linux",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dc",
							Name: "arm64",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name: "ubuntu",
						},
					},
					Enabled: true,
					Instances: []params.Instance{
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name:         "kube-runner-5",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now(),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6de",
							Name:         "kube-runner-4",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now().Add(-1 * time.Hour),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6df",
							Name:         "kube-runner-3",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now().Add(-1 * time.Hour),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dg",
							Name:         "kube-runner-2",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now().Add(-1 * time.Hour),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dh",
							Name:         "kube-runner-1",
							Status:       garmProviderParams.InstancePendingDelete,
							RunnerStatus: params.RunnerTerminated,
							UpdatedAt:    time.Now(),
						},
					},
					RepoID:         "",
					RepoName:       "",
					OrgID:          "",
					OrgName:        "",
					EnterpriseID:   enterpriseID,
					EnterpriseName: enterpriseName,
				}}, nil)

				poolClient.GetPool(pools.NewGetPoolParams().WithPoolID(poolID)).Return(&pools.GetPoolOK{Payload: params.Pool{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					ID:             poolID,
					ProviderName:   "kubernetes_external",
					MaxRunners:     10,
					MinIdleRunners: 5,
					Image:          "linux-ubuntu-22.04-arm64",
					Flavor:         "medium",
					OSType:         "linux",
					OSArch:         "arm64",
					Tags: []params.Tag{
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6da",
							Name: "kubernetes",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6db",
							Name: "linux",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dc",
							Name: "arm64",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name: "ubuntu",
						},
					},
					Enabled: true,
					Instances: []params.Instance{
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name:         "kube-runner-5",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now(),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6de",
							Name:         "kube-runner-4",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now().Add(-1 * time.Hour),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6df",
							Name:         "kube-runner-3",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now().Add(-1 * time.Hour),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dg",
							Name:         "kube-runner-2",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now().Add(-1 * time.Hour),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dh",
							Name:         "kube-runner-1",
							Status:       garmProviderParams.InstancePendingDelete,
							RunnerStatus: params.RunnerTerminated,
							UpdatedAt:    time.Now(),
						},
					},
					RepoID:         "",
					RepoName:       "",
					OrgID:          "",
					OrgName:        "",
					EnterpriseID:   enterpriseID,
					EnterpriseName: enterpriseName,
				}}, nil)

				maxRunners := uint(10)
				minIdleRunners := uint(2)
				enabled := true
				runnerBootstrapTimeout := uint(20)
				extraSpecs := json.RawMessage([]byte{})
				gitHubRunnerGroup := ""
				poolClient.UpdatePool(pools.NewUpdatePoolParams().WithPoolID(poolID).WithBody(params.UpdatePoolParams{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					MaxRunners:             &maxRunners,
					MinIdleRunners:         &minIdleRunners,
					Image:                  "linux-ubuntu-22.04-arm64",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                &enabled,
					RunnerBootstrapTimeout: &runnerBootstrapTimeout,
					ExtraSpecs:             extraSpecs,
					GitHubRunnerGroup:      &gitHubRunnerGroup,
				})).Return(&pools.UpdatePoolOK{Payload: params.Pool{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					ID:             poolID,
					ProviderName:   "kubernetes_external",
					MaxRunners:     10,
					MinIdleRunners: 2,
					Image:          "linux-ubuntu-22.04-arm64",
					Flavor:         "medium",
					OSType:         "linux",
					OSArch:         "arm64",
					Tags: []params.Tag{
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6da",
							Name: "kubernetes",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6db",
							Name: "linux",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dc",
							Name: "arm64",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name: "ubuntu",
						},
					},
					Enabled: true,
					Instances: []params.Instance{
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name:         "kube-runner-5",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now(),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6de",
							Name:         "kube-runner-4",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now().Add(-1 * time.Hour),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6df",
							Name:         "kube-runner-3",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now().Add(-1 * time.Hour),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dg",
							Name:         "kube-runner-2",
							Status:       garmProviderParams.InstanceRunning,
							RunnerStatus: params.RunnerIdle,
							UpdatedAt:    time.Now().Add(-1 * time.Hour),
						},
						{
							ID:           "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dh",
							Name:         "kube-runner-1",
							Status:       garmProviderParams.InstancePendingDelete,
							RunnerStatus: params.RunnerTerminated,
							UpdatedAt:    time.Now(),
						},
					},
					RepoID:         "",
					RepoName:       "",
					OrgID:          "",
					OrgName:        "",
					EnterpriseID:   enterpriseID,
					EnterpriseName: enterpriseName,
				}}, nil)

				instanceClient.DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName("kube-runner-4")).Return(nil)
			},
		},
		{
			name: "pool does not exist in garm - error no image cr found",
			object: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             5,
					MinIdleRunners:         3,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
			},
			expectedObject: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             5,
					MinIdleRunners:         3,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID:                     "",
					LongRunningIdleRunners: 0,
					Selector:               "",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.FetchingImageRefFailedReason),
							Status:             metav1.ConditionFalse,
							Message:            "images.garm-operator.mercedes-benz.com \"ubuntu-image\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.ImageReference),
							Reason:             string(conditions.FetchingImageRefFailedReason),
							Status:             metav1.ConditionFalse,
							Message:            "images.garm-operator.mercedes-benz.com \"ubuntu-image\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.ScopeReference),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched Enterprise CR Ref",
							Reason:             string(conditions.FetchingScopeRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespaceName,
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("supersecretvalue"),
					},
				},
				&garmoperatorv1beta1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1beta1.GroupVersion.Group + "/" + garmoperatorv1beta1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.EnterpriseSpec{
						CredentialsRef: corev1.TypedLocalObjectReference{
							APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
							Kind:     "GitHubCredential",
							Name:     "github-creds",
						},
						WebhookSecretRef: garmoperatorv1beta1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1beta1.EnterpriseStatus{
						ID: enterpriseID,
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
						},
					},
				},
			},
			wantErr: true,
			expectGarmRequest: func(_ *mock.MockPoolClientMockRecorder, _ *mock.MockInstanceClientMockRecorder) {
			},
		},
		{
			name: "pool.Status has matching id in garm database, pool.Specs changed to not existent image cr ref - error no image cr found",
			object: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             1,
					MinIdleRunners:         0,
					ImageName:              "ubuntu-image-not-existent",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID: poolID,
				},
			},
			expectedObject: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             1,
					MinIdleRunners:         0,
					ImageName:              "ubuntu-image-not-existent",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID: poolID,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.FetchingImageRefFailedReason),
							Status:             metav1.ConditionFalse,
							Message:            "images.garm-operator.mercedes-benz.com \"ubuntu-image-not-existent\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.ImageReference),
							Reason:             string(conditions.FetchingImageRefFailedReason),
							Status:             metav1.ConditionFalse,
							Message:            "images.garm-operator.mercedes-benz.com \"ubuntu-image-not-existent\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.ScopeReference),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched Enterprise CR Ref",
							Reason:             string(conditions.FetchingScopeRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespaceName,
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("supersecretvalue"),
					},
				},
				&garmoperatorv1beta1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1beta1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1beta1.GroupVersion.Group + "/" + garmoperatorv1beta1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.EnterpriseSpec{
						CredentialsRef: corev1.TypedLocalObjectReference{
							APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
							Kind:     "GitHubCredential",
							Name:     "github-creds",
						},
						WebhookSecretRef: garmoperatorv1beta1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1beta1.EnterpriseStatus{
						ID: enterpriseID,
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
						},
					},
				},
			},
			wantErr: true,
			expectGarmRequest: func(poolClient *mock.MockPoolClientMockRecorder, _ *mock.MockInstanceClientMockRecorder) {
				poolClient.GetPool(pools.NewGetPoolParams().WithPoolID(poolID)).Return(&pools.GetPoolOK{Payload: params.Pool{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					ID:             poolID,
					ProviderName:   "kubernetes_external",
					MaxRunners:     5,
					MinIdleRunners: 3,
					Image:          "linux-ubuntu-22.04-arm64",
					Flavor:         "medium",
					OSType:         "linux",
					OSArch:         "arm64",
					Tags: []params.Tag{
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6da",
							Name: "kubernetes",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6db",
							Name: "linux",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dc",
							Name: "arm64",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name: "ubuntu",
						},
					},
					Enabled:        true,
					Instances:      []params.Instance{},
					RepoID:         "",
					RepoName:       "",
					OrgID:          "",
					OrgName:        "",
					EnterpriseID:   enterpriseID,
					EnterpriseName: enterpriseName,
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1beta1.Pool{}).Build()

			// create a fake reconciler
			reconciler := &PoolReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			pool := tt.object.DeepCopyObject().(*garmoperatorv1beta1.Pool)

			mockPoolClient := mock.NewMockPoolClient(mockCtrl)
			mockInstanceClient := mock.NewMockInstanceClient(mockCtrl)

			// as the global configuration got initialized in the main function
			// and the defaulting is done in there as well
			// we have to explicitly set the values in here
			config.Config.Operator.MinIdleRunnersAge = time.Duration(30) * time.Minute

			tt.expectGarmRequest(mockPoolClient.EXPECT(), mockInstanceClient.EXPECT())

			_, err = reconciler.reconcileNormal(context.Background(), mockPoolClient, pool, mockInstanceClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("PoolReconciler.reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// clear out annotations to avoid comparison errors
			pool.ObjectMeta.Annotations = nil

			// empty resource version to avoid comparison errors
			pool.ObjectMeta.ResourceVersion = ""

			// clear conditions lastTransitionTime to avoid comparison errors
			conditions.NilLastTransitionTime(tt.expectedObject)
			conditions.NilLastTransitionTime(pool)

			if !reflect.DeepEqual(pool, tt.expectedObject) {
				t.Errorf("PoolReconciler.reconcileNormal() \n got =  %#v \n want = %#v", pool, tt.expectedObject)
			}
		})
	}
}

func TestPoolController_ReconcileDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	poolID := "fb2bceeb-f74d-435d-9648-626c75cb23ce"
	enterpriseID := "93068607-2d0d-4b76-a950-0e40d31955b8"
	enterpriseName := "test-enterprise"

	tests := []struct {
		name string
		// the object to reconcile
		object client.Object
		// a list of objects to initialize the fake client with
		// this can be used to define other existing objects that are referenced by the object to reconcile
		// e.g. images or other pools ..
		runtimeObjects    []runtime.Object
		expectGarmRequest func(_ *mock.MockPoolClientMockRecorder, _ *mock.MockInstanceClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1beta1.Pool
	}{
		{
			name: "delete pool - scaling down runners",
			object: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             5,
					MinIdleRunners:         3,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID: poolID,
				},
			},
			expectedObject: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						// APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind: string(garmoperatorv1beta1.EnterpriseScope),
						Name: enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             5,
					MinIdleRunners:         0,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                false,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID:                     poolID,
					LongRunningIdleRunners: 0,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Status:             metav1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Reason:             string(conditions.DeletingReason),
							Message:            conditions.DeletingPoolMsg,
						},
					},
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespaceName,
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("supersecretvalue"),
					},
				},
				&garmoperatorv1beta1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1beta1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1beta1.GroupVersion.Group + "/" + garmoperatorv1beta1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.EnterpriseSpec{
						CredentialsRef: corev1.TypedLocalObjectReference{
							APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
							Kind:     "GitHubCredential",
							Name:     "github-creds",
						},
						WebhookSecretRef: garmoperatorv1beta1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1beta1.EnterpriseStatus{
						ID: enterpriseID,
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
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockPoolClientMockRecorder, instanceClient *mock.MockInstanceClientMockRecorder) {
				maxRunners := uint(5)
				minIdleRunners := uint(0)
				enabled := false
				runnerBootstrapTimeout := uint(20)
				extraSpecs := json.RawMessage([]byte{})
				gitHubRunnerGroup := ""

				instanceClient.ListPoolInstances(
					instances.NewListPoolInstancesParams().
						WithPoolID(poolID)).
					Return(&instances.ListPoolInstancesOK{
						Payload: params.Instances{},
					},
						nil)

				m.UpdatePool(pools.NewUpdatePoolParams().
					WithPoolID(poolID).
					WithBody(params.UpdatePoolParams{
						RunnerPrefix: params.RunnerPrefix{
							Prefix: "",
						},
						MaxRunners:             &maxRunners,
						MinIdleRunners:         &minIdleRunners,
						Flavor:                 "medium",
						OSType:                 "linux",
						OSArch:                 "arm64",
						Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
						Enabled:                &enabled,
						RunnerBootstrapTimeout: &runnerBootstrapTimeout,
						ExtraSpecs:             extraSpecs,
						GitHubRunnerGroup:      &gitHubRunnerGroup,
					})).Return(&pools.UpdatePoolOK{
					Payload: params.Pool{
						RunnerPrefix: params.RunnerPrefix{
							Prefix: "",
						},
						ID:             poolID,
						ProviderName:   "kubernetes_external",
						MaxRunners:     5,
						MinIdleRunners: 0,
						Image:          "linux-ubuntu-22.04-arm64",
						Flavor:         "medium",
						OSType:         "linux",
						OSArch:         "arm64",
						Tags: []params.Tag{
							{
								ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6da",
								Name: "kubernetes",
							},
							{
								ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6db",
								Name: "linux",
							},
							{
								ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dc",
								Name: "arm64",
							},
							{
								ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
								Name: "ubuntu",
							},
						},
						Enabled:        false,
						Instances:      []params.Instance{},
						RepoID:         "",
						RepoName:       "",
						OrgID:          "",
						OrgName:        "",
						EnterpriseID:   enterpriseID,
						EnterpriseName: enterpriseName,
					},
				}, nil)

				m.DeletePool(pools.NewDeletePoolParams().WithPoolID(poolID)).Return(nil)
			},
		},
		{
			name: "delete pool - deleting garm resource",
			object: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     string(garmoperatorv1beta1.EnterpriseScope),
						Name:     enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             5,
					MinIdleRunners:         0,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                true,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID:                     poolID,
					LongRunningIdleRunners: 0,
				},
			},
			expectedObject: &garmoperatorv1beta1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
				},
				Spec: garmoperatorv1beta1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						// APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind: string(garmoperatorv1beta1.EnterpriseScope),
						Name: enterpriseName,
					},
					ProviderName:           "kubernetes_external",
					MaxRunners:             5,
					MinIdleRunners:         0,
					ImageName:              "ubuntu-image",
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                false,
					RunnerBootstrapTimeout: 20,
					ExtraSpecs:             "",
					GitHubRunnerGroup:      "",
				},
				Status: garmoperatorv1beta1.PoolStatus{
					ID: poolID,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Status:             metav1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Reason:             string(conditions.DeletingReason),
							Message:            conditions.DeletingPoolMsg,
						},
					},
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespaceName,
						Name:      "my-webhook-secret",
					},
					Data: map[string][]byte{
						"webhookSecret": []byte("supersecretvalue"),
					},
				},
				&garmoperatorv1beta1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1beta1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1beta1.GroupVersion.Group + "/" + garmoperatorv1beta1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1beta1.EnterpriseSpec{
						CredentialsRef: corev1.TypedLocalObjectReference{
							APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
							Kind:     "GitHubCredential",
							Name:     "github-creds",
						},
						WebhookSecretRef: garmoperatorv1beta1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1beta1.EnterpriseStatus{
						ID: enterpriseID,
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
						},
					},
				},
			},
			expectGarmRequest: func(poolClient *mock.MockPoolClientMockRecorder, instanceClient *mock.MockInstanceClientMockRecorder) {
				instanceClient.ListPoolInstances(instances.NewListPoolInstancesParams().WithPoolID(poolID)).Return(&instances.ListPoolInstancesOK{Payload: params.Instances{}}, nil)
				poolClient.DeletePool(pools.NewDeletePoolParams().WithPoolID(poolID)).Return(nil)

				poolClient.UpdatePool(pools.NewUpdatePoolParams().WithPoolID(poolID).WithBody(params.UpdatePoolParams{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					MaxRunners:             ptr.To(uint(5)),
					MinIdleRunners:         ptr.To(uint(0)),
					Flavor:                 "medium",
					OSType:                 "linux",
					OSArch:                 "arm64",
					Tags:                   []string{"kubernetes", "linux", "arm64", "ubuntu"},
					Enabled:                ptr.To(false),
					RunnerBootstrapTimeout: ptr.To(uint(20)),
					ExtraSpecs:             json.RawMessage([]byte{}),
					GitHubRunnerGroup:      ptr.To(""),
				})).Return(&pools.UpdatePoolOK{Payload: params.Pool{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					ID:             poolID,
					ProviderName:   "kubernetes_external",
					MaxRunners:     1,
					MinIdleRunners: 0,
					Image:          "linux-ubuntu-22.04-arm64",
					Flavor:         "medium",
					OSType:         "linux",
					OSArch:         "arm64",
					Tags: []params.Tag{
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6da",
							Name: "kubernetes",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6db",
							Name: "linux",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dc",
							Name: "arm64",
						},
						{
							ID:   "b3ea9882-a25c-4eb1-94ba-6c70b9abb6dd",
							Name: "ubuntu",
						},
					},
					Enabled:        false,
					Instances:      []params.Instance{},
					RepoID:         "",
					RepoName:       "",
					OrgID:          "",
					OrgName:        "",
					EnterpriseID:   enterpriseID,
					EnterpriseName: enterpriseName,
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1beta1.Pool{}).Build()

			// create a fake reconciler
			reconciler := &PoolReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			pool := tt.object.DeepCopyObject().(*garmoperatorv1beta1.Pool)

			mockPoolClient := mock.NewMockPoolClient(mockCtrl)
			mockInstanceClient := mock.NewMockInstanceClient(mockCtrl)

			tt.expectGarmRequest(mockPoolClient.EXPECT(), mockInstanceClient.EXPECT())

			_, err = reconciler.reconcileDelete(context.Background(), mockPoolClient, pool, mockInstanceClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("PoolReconciler.reconcileDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// empty resource version to avoid comparison errors
			pool.ObjectMeta.ResourceVersion = ""
			pool.Spec.GitHubScopeRef.APIGroup = nil
			conditions.NilLastTransitionTime(pool)
			conditions.NilLastTransitionTime(tt.expectedObject)

			if !reflect.DeepEqual(pool, tt.expectedObject) {
				t.Errorf("PoolReconciler.reconcileNormal() \n got =  %#v \n want = %#v", pool, tt.expectedObject)
			}
		})
	}
}
