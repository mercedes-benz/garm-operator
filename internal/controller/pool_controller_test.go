package controller

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/shared"
	garmoperatorv1alpha1 "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/v1alpha1"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/key"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/mock"
	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/pools"
	"github.com/cloudbase/garm/params"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPoolController_ReconcileCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	outdatedPoolId := "a9f48897-77c3-4293-8462-732a22a908f1"
	poolId := "fb2bceeb-f74d-435d-9648-626c75cb23ce"
	enterpriseId := "93068607-2d0d-4b76-a950-0e40d31955b8"
	enterpriseName := "test-enterprise"
	namespaceName := "test-namespace"

	tests := []struct {
		name string
		// the object to reconcile
		object         client.Object
		gitHubScopeRef client.Object
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
					GitHubScopeRef: v1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(shared.EnterpriseScope),
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
					GitHubScopeRef: v1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(shared.EnterpriseScope),
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
					ID:            poolId,
					Synced:        true,
					LastSyncTime:  metav1.Time{},
					LastSyncError: "",
					RunnerCount:   0,
					ActiveRunners: 0,
					IdleRunners:   0,
				},
			},
			runtimeObjects: []runtime.Object{
				&garmoperatorv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
			},
			gitHubScopeRef: &garmoperatorv1alpha1.Enterprise{
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
					WebhookSecret:   "foobar",
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID:                       enterpriseId,
					PoolManagerIsRunning:     false,
					PoolManagerFailureReason: "no resources available",
				},
			},
			expectGarmRequest: func(m *mock.MockPoolClientMockRecorder) {
				m.ListAllPools(pools.NewListPoolsParams()).Return(&pools.ListPoolsOK{Payload: params.Pools{}}, nil)

				extraSpecs := json.RawMessage([]byte{})
				m.CreateEnterprisePool(
					enterprises.NewCreateEnterprisePoolParams().WithEnterpriseID(enterpriseId).WithBody(
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
						ID:             poolId,
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
						EnterpriseID:   enterpriseId,
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
					GitHubScopeRef: v1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(shared.EnterpriseScope),
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
					ID:            outdatedPoolId,
					Synced:        true,
					LastSyncTime:  metav1.Time{},
					LastSyncError: "",
					RunnerCount:   0,
					ActiveRunners: 0,
					IdleRunners:   0,
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
					GitHubScopeRef: v1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(shared.EnterpriseScope),
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
					ID:            poolId,
					Synced:        true,
					LastSyncTime:  metav1.Time{},
					LastSyncError: "",
					RunnerCount:   0,
					ActiveRunners: 0,
					IdleRunners:   0,
				},
			},
			runtimeObjects: []runtime.Object{
				&garmoperatorv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
			},
			gitHubScopeRef: &garmoperatorv1alpha1.Enterprise{
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
					WebhookSecret:   "foobar",
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID:                       enterpriseId,
					PoolManagerIsRunning:     false,
					PoolManagerFailureReason: "no resources available",
				},
			},
			expectGarmRequest: func(m *mock.MockPoolClientMockRecorder) {
				m.GetPool(pools.NewGetPoolParams().WithPoolID(outdatedPoolId)).Return(&pools.GetPoolOK{Payload: params.Pool{}}, nil)

				m.ListAllPools(pools.NewListPoolsParams()).Return(&pools.ListPoolsOK{Payload: params.Pools{
					{
						RunnerPrefix: params.RunnerPrefix{
							Prefix: "",
						},
						ID:             poolId,
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
						EnterpriseID:   enterpriseId,
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
					GitHubScopeRef: v1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(shared.EnterpriseScope),
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
					ID:            poolId,
					Synced:        true,
					LastSyncTime:  metav1.Time{},
					LastSyncError: "",
					RunnerCount:   0,
					ActiveRunners: 0,
					IdleRunners:   0,
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
					GitHubScopeRef: v1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
						Kind:     string(shared.EnterpriseScope),
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
					ID:            poolId,
					Synced:        true,
					LastSyncTime:  metav1.Time{},
					LastSyncError: "",
					RunnerCount:   0,
					ActiveRunners: 0,
					IdleRunners:   0,
				},
			},
			runtimeObjects: []runtime.Object{
				&garmoperatorv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ubuntu-image",
						Namespace: namespaceName,
					},
					Spec: garmoperatorv1alpha1.ImageSpec{
						Tag: "linux-ubuntu-22.04-arm64",
					},
				},
			},
			gitHubScopeRef: &garmoperatorv1alpha1.Enterprise{
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
					WebhookSecret:   "foobar",
				},
				Status: garmoperatorv1alpha1.EnterpriseStatus{
					ID:                       enterpriseId,
					PoolManagerIsRunning:     false,
					PoolManagerFailureReason: "no resources available",
				},
			},
			expectGarmRequest: func(m *mock.MockPoolClientMockRecorder) {
				m.GetPool(pools.NewGetPoolParams().WithPoolID(poolId)).Return(&pools.GetPoolOK{Payload: params.Pool{
					RunnerPrefix: params.RunnerPrefix{
						Prefix: "",
					},
					ID:             poolId,
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
					EnterpriseID:   enterpriseId,
					EnterpriseName: enterpriseName,
				}}, nil)

				maxRunners := uint(1)
				minIdleRunners := uint(0)
				enabled := true
				runnerBootstrapTimeout := uint(20)
				extraSpecs := json.RawMessage([]byte{})
				gitHubRunnerGroup := ""
				m.UpdatePool(pools.NewUpdatePoolParams().WithPoolID(poolId).WithBody(params.UpdatePoolParams{
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
					ID:             poolId,
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
					EnterpriseID:   enterpriseId,
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
			}

			pool := tt.object.DeepCopyObject().(*garmoperatorv1alpha1.Pool)
			gitHubScopeRef := tt.gitHubScopeRef.(shared.GitHubScope)

			mockPoolClient := mock.NewMockPoolClient(mockCtrl)
			tt.expectGarmRequest(mockPoolClient.EXPECT())

			_, err = reconciler.reconcileNormal(context.Background(), mockPoolClient, pool, gitHubScopeRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("PoolReconciler.reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// empty resource version to avoid comparison errors
			pool.ObjectMeta.ResourceVersion = ""
			pool.Status.LastSyncTime = metav1.Time{}
			if !reflect.DeepEqual(pool, tt.expectedObject) {
				t.Errorf("PoolReconciler.reconcileNormal() \n got =  %#v \n want = %#v", pool, tt.expectedObject)
			}

		})
	}
}
