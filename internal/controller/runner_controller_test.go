// SPDX-License-Identifier: MIT
package controller

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	commonParams "github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/params"
	"github.com/life4/genesis/slices"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/client/mock"
	"github.com/mercedes-benz/garm-operator/pkg/config"
)

func TestRunnerReconciler_reconcileCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		req               ctrl.Request
		runtimeObjects    []runtime.Object
		expectGarmRequest func(m *mock.MockInstanceClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1alpha1.Runner
	}{
		{
			name: "Create Runner CR",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "road-runner-k8s-FY5snJcv5dzn",
					Namespace: "runner",
				},
			},
			runtimeObjects: []runtime.Object{},
			expectGarmRequest: func(m *mock.MockInstanceClientMockRecorder) {
				response := params.Instances{
					params.Instance{
						Name:         "road-runner-k8s-FY5snJcv5dzn",
						AgentID:      120,
						ID:           "8215f6c6-486e-4893-84df-3231b185a148",
						OSArch:       "amd64",
						OSType:       "linux",
						PoolID:       "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
						ProviderID:   "kubernetes_external",
						Status:       commonParams.InstanceRunning,
						RunnerStatus: params.RunnerIdle,
					},
				}

				m.ListInstances(instances.NewListInstancesParams()).Return(&instances.ListInstancesOK{Payload: response}, nil)
			},
			wantErr: false,
			expectedObject: &garmoperatorv1alpha1.Runner{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Runner",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "road-runner-k8s-fy5snjcv5dzn",
					Namespace: "runner",
					Finalizers: []string{
						key.RunnerFinalizerName,
					},
				},
				Spec: garmoperatorv1alpha1.RunnerSpec{},
				Status: garmoperatorv1alpha1.RunnerStatus{
					Name:           "road-runner-k8s-FY5snJcv5dzn",
					AgentID:        120,
					ID:             "8215f6c6-486e-4893-84df-3231b185a148",
					OSArch:         "amd64",
					OSType:         "linux",
					PoolID:         "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
					ProviderID:     "kubernetes_external",
					Status:         commonParams.InstanceRunning,
					InstanceStatus: params.RunnerIdle,
				},
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
			var runtimeObjects []runtime.Object
			runtimeObjects = append(runtimeObjects, tt.runtimeObjects...)
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1alpha1.Runner{}).Build()

			// create a fake reconciler
			reconciler := &RunnerReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			mockInstanceClient := mock.NewMockInstanceClient(mockCtrl)
			tt.expectGarmRequest(mockInstanceClient.EXPECT())

			_, err = reconciler.reconcile(context.Background(), tt.req, mockInstanceClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunnerReconciler.reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			runner := &garmoperatorv1alpha1.Runner{}
			err = client.Get(context.Background(), types.NamespacedName{Namespace: tt.req.Namespace, Name: strings.ToLower(tt.req.Name)}, runner)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunnerReconciler.reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// empty resource version to avoid comparison errors
			runner.ObjectMeta.ResourceVersion = ""
			if !reflect.DeepEqual(runner, tt.expectedObject) {
				t.Errorf("RunnerReconciler.reconcile() got = %#v, want %#v", runner, tt.expectedObject)
			}
		})
	}
}

func TestRunnerReconciler_reconcileDeleteGarmRunner(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	now := metav1.NewTime(time.Now().UTC())

	tests := []struct {
		name              string
		req               ctrl.Request
		runtimeObjects    []runtime.Object
		expectGarmRequest func(m *mock.MockInstanceClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1alpha1.Runner
	}{
		{
			name: "Delete Runner in Garm DB, when Runner CR is marked with deletion timestamp",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "road-runner-k8s-fy5snjcv5dzn",
					Namespace: "runner",
				},
			},
			runtimeObjects: []runtime.Object{
				&garmoperatorv1alpha1.Runner{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Runner",
						APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "road-runner-k8s-fy5snjcv5dzn",
						Namespace: "runner",
						Finalizers: []string{
							key.RunnerFinalizerName,
						},
						DeletionTimestamp: &now,
					},
					Spec: garmoperatorv1alpha1.RunnerSpec{},
					Status: garmoperatorv1alpha1.RunnerStatus{
						Name:           "road-runner-k8s-FY5snJcv5dzn",
						AgentID:        120,
						ID:             "8215f6c6-486e-4893-84df-3231b185a148",
						OSArch:         "amd64",
						OSType:         "linux",
						PoolID:         "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
						ProviderID:     "kubernetes_external",
						Status:         commonParams.InstanceRunning,
						InstanceStatus: params.RunnerIdle,
					},
				},
			},
			expectGarmRequest: func(m *mock.MockInstanceClientMockRecorder) {
				response := params.Instances{
					params.Instance{
						Name:         "road-runner-k8s-FY5snJcv5dzn",
						AgentID:      120,
						ID:           "8215f6c6-486e-4893-84df-3231b185a148",
						OSArch:       "amd64",
						OSType:       "linux",
						PoolID:       "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
						ProviderID:   "kubernetes_external",
						Status:       commonParams.InstanceRunning,
						RunnerStatus: params.RunnerIdle,
					},
				}

				m.ListInstances(instances.NewListInstancesParams()).Return(&instances.ListInstancesOK{Payload: response}, nil)

				m.DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName("road-runner-k8s-FY5snJcv5dzn")).Return(nil)
			},
			wantErr: false,
			expectedObject: &garmoperatorv1alpha1.Runner{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Runner",
					APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "road-runner-k8s-fy5snjcv5dzn",
					Namespace: "runner",
					Finalizers: []string{
						key.RunnerFinalizerName,
					},
					DeletionTimestamp: &now,
				},
				Spec: garmoperatorv1alpha1.RunnerSpec{},
				Status: garmoperatorv1alpha1.RunnerStatus{
					Name:           "road-runner-k8s-FY5snJcv5dzn",
					AgentID:        120,
					ID:             "8215f6c6-486e-4893-84df-3231b185a148",
					OSArch:         "amd64",
					OSType:         "linux",
					PoolID:         "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
					ProviderID:     "kubernetes_external",
					Status:         commonParams.InstanceRunning,
					InstanceStatus: params.RunnerIdle,
				},
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
			var runtimeObjects []runtime.Object
			runtimeObjects = append(runtimeObjects, tt.runtimeObjects...)
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1alpha1.Runner{}).Build()

			// create a fake reconciler
			reconciler := &RunnerReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			mockInstanceClient := mock.NewMockInstanceClient(mockCtrl)
			tt.expectGarmRequest(mockInstanceClient.EXPECT())

			_, err = reconciler.reconcile(context.Background(), tt.req, mockInstanceClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunnerReconciler.reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			runner := &garmoperatorv1alpha1.Runner{}
			err = client.Get(context.Background(), types.NamespacedName{Namespace: tt.req.Namespace, Name: strings.ToLower(tt.req.Name)}, runner)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunnerReconciler.reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.NotZero(t, runner.ObjectMeta.DeletionTimestamp)
		})
	}
}

func TestRunnerReconciler_reconcileDeleteCR(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		req               ctrl.Request
		runtimeObjects    []runtime.Object
		expectGarmRequest func(m *mock.MockInstanceClientMockRecorder)
		wantErr           bool
		expectedEvents    []event.GenericEvent
	}{
		{
			name: "Delete Runner CR with no matching entry in Garm DB",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "road-runner-k8s-fy5snjcv5dzn",
					Namespace: "runner",
				},
			},
			runtimeObjects: []runtime.Object{
				&garmoperatorv1alpha1.Runner{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Runner",
						APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "road-runner-k8s-fy5snjcv5dzn",
						Namespace: "test-namespace",
						Finalizers: []string{
							key.RunnerFinalizerName,
						},
					},
					Spec: garmoperatorv1alpha1.RunnerSpec{},
					Status: garmoperatorv1alpha1.RunnerStatus{
						Name:           "road-runner-k8s-FY5snJcv5dzn",
						AgentID:        120,
						ID:             "8215f6c6-486e-4893-84df-3231b185a148",
						OSArch:         "amd64",
						OSType:         "linux",
						PoolID:         "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
						ProviderID:     "road-runner-k8s-fy5snjcv5dzn",
						Status:         commonParams.InstanceRunning,
						InstanceStatus: params.RunnerIdle,
					},
				},
				&garmoperatorv1alpha1.Runner{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Runner",
						APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "road-runner-k8s-n6kq2mt3k4qr",
						Namespace: "test-namespace",
						Finalizers: []string{
							key.RunnerFinalizerName,
						},
					},
					Spec: garmoperatorv1alpha1.RunnerSpec{},
					Status: garmoperatorv1alpha1.RunnerStatus{
						Name:           "road-runner-k8s-n6KQ2Mt3k4qr",
						AgentID:        130,
						ID:             "13d31cad-588b-4ea8-8015-052a76ad3dd3",
						OSArch:         "amd64",
						OSType:         "linux",
						PoolID:         "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
						ProviderID:     "road-runner-k8s-n6kq2mt3k4qr",
						Status:         commonParams.InstanceRunning,
						InstanceStatus: params.RunnerIdle,
					},
				},
				&garmoperatorv1alpha1.Pool{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pool",
						APIVersion: garmoperatorv1alpha1.GroupVersion.Group + "/" + garmoperatorv1alpha1.GroupVersion.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-enterprise-pool",
						Namespace: "test-namespace",
					},
					Spec: garmoperatorv1alpha1.PoolSpec{
						GitHubScopeRef: corev1.TypedLocalObjectReference{
							APIGroup: &garmoperatorv1alpha1.GroupVersion.Group,
							Kind:     string(garmoperatorv1alpha1.EnterpriseScope),
							Name:     "my-enterprise",
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
						ID: "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
					},
				},
			},
			expectGarmRequest: func(m *mock.MockInstanceClientMockRecorder) {
				response := &instances.ListPoolInstancesOK{Payload: params.Instances{
					params.Instance{
						Name:         "road-runner-k8s-FY5snJcv5dzn",
						AgentID:      120,
						ID:           "8215f6c6-486e-4893-84df-3231b185a148",
						OSArch:       "amd64",
						OSType:       "linux",
						PoolID:       "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
						ProviderID:   "road-runner-k8s-fy5snjcv5dzn",
						Status:       commonParams.InstanceRunning,
						RunnerStatus: params.RunnerIdle,
					},
				}}
				m.ListPoolInstances(instances.NewListPoolInstancesParams().WithPoolID("a46553c6-ad87-454b-b5f5-a1c468d78c1e")).Return(response, nil)
			},
			wantErr: false,
			expectedEvents: []event.GenericEvent{
				{
					Object: &garmoperatorv1alpha1.Runner{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "road-runner-k8s-fy5snjcv5dzn",
							Namespace: "test-namespace",
						},
					},
				},
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
			var runtimeObjects []runtime.Object
			runtimeObjects = append(runtimeObjects, tt.runtimeObjects...)
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1alpha1.Runner{}).Build()

			// create a fake reconciler
			reconciler := &RunnerReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			mockInstanceClient := mock.NewMockInstanceClient(mockCtrl)
			tt.expectGarmRequest(mockInstanceClient.EXPECT())

			config.Config.Operator.WatchNamespace = "test-namespace"
			fakeChan := make(chan event.GenericEvent)

			go func() {
				err = reconciler.EnqueueRunnerInstances(context.Background(), mockInstanceClient, fakeChan)
				if (err != nil) != tt.wantErr {
					t.Errorf("RunnerReconciler.EnqueueRunnerInstances() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				close(fakeChan)
			}()

			var eventCount int
			for obj := range fakeChan {
				t.Logf("Received Event: %s", obj.Object.GetName())
				filtered := slices.Filter(tt.expectedEvents, func(e event.GenericEvent) bool {
					return e.Object.GetName() == obj.Object.GetName() && e.Object.GetNamespace() == obj.Object.GetNamespace()
				})
				eventCount++
				assert.Equal(t, 1, len(filtered))
			}
			assert.Equal(t, len(tt.expectedEvents), eventCount)
		})
	}
}
