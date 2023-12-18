// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

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

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/client/mock"
)

const namespaceName = "test-namespace"

func TestPoolController_ReconcileCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	outdatedPoolID := "a9f48897-77c3-4293-8462-732a22a908f1"
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
		expectGarmRequest func(m *mock.MockPoolClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1alpha1.Pool
	}{
		{
			name: "pool does not exist in garm - create",
			object: &garmoperatorv1alpha1.Pool{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pool",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
				},
				Spec: garmoperatorv1alpha1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(garmoperatorv1alpha1.EnterpriseScope),
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
			expectedObject: &garmoperatorv1alpha1.Pool{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pool",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(garmoperatorv1alpha1.EnterpriseScope),
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
				Status: garmoperatorv1alpha1.PoolStatus{
					ID:            poolID,
					IdleRunners:   3,
					Selector:      "",
					LastSyncError: "",
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
				&garmoperatorv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1alpha1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.EnterpriseSpec{
						CredentialsName: "foobar",
						WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1alpha1.EnterpriseStatus{
						ID:                       enterpriseID,
						PoolManagerIsRunning:     false,
						PoolManagerFailureReason: "no resources available",
					},
				},
			},
			expectGarmRequest: func(m *mock.MockPoolClientMockRecorder) {
				m.ListAllPools(pools.NewListPoolsParams()).Return(&pools.ListPoolsOK{Payload: params.Pools{}}, nil)

				extraSpecs := json.RawMessage([]byte{})
				m.CreateEnterprisePool(
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
			name: "pool with matching specs exists in garm, outdated garmId in pool.Status - sync ids",
			object: &garmoperatorv1alpha1.Pool{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pool",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
				},
				Spec: garmoperatorv1alpha1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(garmoperatorv1alpha1.EnterpriseScope),
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
				Status: garmoperatorv1alpha1.PoolStatus{
					ID:            outdatedPoolID,
					LastSyncError: "",
				},
			},
			expectedObject: &garmoperatorv1alpha1.Pool{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pool",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(garmoperatorv1alpha1.EnterpriseScope),
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
				Status: garmoperatorv1alpha1.PoolStatus{
					ID:            poolID,
					IdleRunners:   3,
					Selector:      "",
					LastSyncError: "",
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
				&garmoperatorv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1alpha1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.EnterpriseSpec{
						CredentialsName: "foobar",
						WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1alpha1.EnterpriseStatus{
						ID:                       enterpriseID,
						PoolManagerIsRunning:     false,
						PoolManagerFailureReason: "no resources available",
					},
				},
			},
			expectGarmRequest: func(m *mock.MockPoolClientMockRecorder) {
				m.GetPool(pools.NewGetPoolParams().WithPoolID(outdatedPoolID)).Return(&pools.GetPoolOK{Payload: params.Pool{}}, nil)

				m.ListAllPools(pools.NewListPoolsParams()).Return(&pools.ListPoolsOK{Payload: params.Pools{
					{
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
				}}, nil)
			},
		},
		{
			name: "pool.Status has matching id in garm database, pool.Specs changed - update pool in garm",
			object: &garmoperatorv1alpha1.Pool{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pool",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
				},
				Spec: garmoperatorv1alpha1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(garmoperatorv1alpha1.EnterpriseScope),
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
				Status: garmoperatorv1alpha1.PoolStatus{
					ID:            poolID,
					LastSyncError: "",
				},
			},
			expectedObject: &garmoperatorv1alpha1.Pool{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pool",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(garmoperatorv1alpha1.EnterpriseScope),
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
				Status: garmoperatorv1alpha1.PoolStatus{
					ID:            poolID,
					LastSyncError: "",
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
				&garmoperatorv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1alpha1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.EnterpriseSpec{
						CredentialsName: "foobar",
						WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1alpha1.EnterpriseStatus{
						ID:                       enterpriseID,
						PoolManagerIsRunning:     false,
						PoolManagerFailureReason: "no resources available",
					},
				},
			},
			expectGarmRequest: func(m *mock.MockPoolClientMockRecorder) {
				m.GetPool(pools.NewGetPoolParams().WithPoolID(poolID)).Return(&pools.GetPoolOK{Payload: params.Pool{
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

				m.GetPool(pools.NewGetPoolParams().WithPoolID(poolID)).Return(&pools.GetPoolOK{Payload: params.Pool{
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
				m.UpdatePool(pools.NewUpdatePoolParams().WithPoolID(poolID).WithBody(params.UpdatePoolParams{
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1alpha1.Pool{}).Build()

			// create a fake reconciler
			reconciler := &PoolReconciler{
				Client:   client,
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
				Recorder: record.NewFakeRecorder(3),
			}

			pool := tt.object.DeepCopyObject().(*garmoperatorv1alpha1.Pool)

			mockPoolClient := mock.NewMockPoolClient(mockCtrl)
			mockInstanceClient := mock.NewMockInstanceClient(mockCtrl)

			tt.expectGarmRequest(mockPoolClient.EXPECT())

			_, err = reconciler.reconcileNormal(context.Background(), mockPoolClient, pool, mockInstanceClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("PoolReconciler.reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// empty resource version to avoid comparison errors
			pool.ObjectMeta.ResourceVersion = ""
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
		expectGarmRequest func(poolClient *mock.MockPoolClientMockRecorder, instanceClient *mock.MockInstanceClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1alpha1.Pool
	}{
		{
			name: "delete pool - scaling down runners",
			object: &garmoperatorv1alpha1.Pool{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pool",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(garmoperatorv1alpha1.EnterpriseScope),
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
				Status: garmoperatorv1alpha1.PoolStatus{
					ID:            poolID,
					LastSyncError: "",
				},
			},
			expectedObject: &garmoperatorv1alpha1.Pool{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pool",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "my-enterprise-pool",
					Namespace:  namespaceName,
					Finalizers: []string{},
				},
				Spec: garmoperatorv1alpha1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						// APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind: string(garmoperatorv1alpha1.EnterpriseScope),
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
				Status: garmoperatorv1alpha1.PoolStatus{
					ID:            poolID,
					LastSyncError: "",
					IdleRunners:   0,
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
				&garmoperatorv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1alpha1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.EnterpriseSpec{
						CredentialsName: "foobar",
						WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1alpha1.EnterpriseStatus{
						ID:                       enterpriseID,
						PoolManagerIsRunning:     false,
						PoolManagerFailureReason: "no resources available",
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
						Image:                  "linux-ubuntu-22.04-arm64",
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
			object: &garmoperatorv1alpha1.Pool{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pool",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-enterprise-pool",
					Namespace: namespaceName,
					Finalizers: []string{
						key.PoolFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(garmoperatorv1alpha1.EnterpriseScope),
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
				Status: garmoperatorv1alpha1.PoolStatus{
					ID:            poolID,
					LastSyncError: "",
					Runners:       0,
					IdleRunners:   0,
				},
			},
			expectedObject: &garmoperatorv1alpha1.Pool{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pool",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "my-enterprise-pool",
					Namespace:  namespaceName,
					Finalizers: []string{},
				},
				Spec: garmoperatorv1alpha1.PoolSpec{
					GitHubScopeRef: corev1.TypedLocalObjectReference{
						// APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind: string(garmoperatorv1alpha1.EnterpriseScope),
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
				Status: garmoperatorv1alpha1.PoolStatus{
					ID:            poolID,
					LastSyncError: "",
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
				&garmoperatorv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
				&garmoperatorv1alpha1.Enterprise{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Enterprise",
						APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      enterpriseName,
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.EnterpriseSpec{
						CredentialsName: "foobar",
						WebhookSecretRef: garmoperatorv1alpha1.SecretRef{
							Name: "my-webhook-secret",
							Key:  "webhookSecret",
						},
					},
					Status: garmoperatorv1alpha1.EnterpriseStatus{
						ID:                       enterpriseID,
						PoolManagerIsRunning:     false,
						PoolManagerFailureReason: "no resources available",
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
					Image:                  "linux-ubuntu-22.04-arm64",
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
				garmoperatorv1alpha1.AddToScheme,
			}

			err := schemeBuilder.AddToScheme(scheme.Scheme)
			if err != nil {
				t.Fatal(err)
			}
			runtimeObjects := []runtime.Object{tt.object}
			runtimeObjects = append(runtimeObjects, tt.runtimeObjects...)
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1alpha1.Pool{}).Build()

			// create a fake reconciler
			reconciler := &PoolReconciler{
				Client:   client,
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
				Recorder: record.NewFakeRecorder(3),
			}

			pool := tt.object.DeepCopyObject().(*garmoperatorv1alpha1.Pool)

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

			if !reflect.DeepEqual(pool, tt.expectedObject) {
				t.Errorf("PoolReconciler.reconcileNormal() \n got =  %#v \n want = %#v", pool, tt.expectedObject)
			}
		})
	}
}
