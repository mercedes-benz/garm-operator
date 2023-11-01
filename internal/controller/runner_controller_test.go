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
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/client/mock"
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
				runner := params.Instance{
					Name:         "road-runner-k8s-FY5snJcv5dzn",
					AgentID:      120,
					ID:           "8215f6c6-486e-4893-84df-3231b185a148",
					OSArch:       "amd64",
					OSType:       "linux",
					PoolID:       "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
					ProviderID:   "kubernetes_external",
					Status:       commonParams.InstanceRunning,
					RunnerStatus: params.RunnerIdle,
				}
				response := instances.NewGetInstanceOK()
				response.Payload = runner
				m.GetInstanceByName(instances.NewGetInstanceParams().WithInstanceName("road-runner-k8s-FY5snJcv5dzn")).Return(response, nil)
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
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
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
			name: "Delete Runner CR",
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
				runner := params.Instance{
					Name:         "road-runner-k8s-FY5snJcv5dzn",
					AgentID:      120,
					ID:           "8215f6c6-486e-4893-84df-3231b185a148",
					OSArch:       "amd64",
					OSType:       "linux",
					PoolID:       "a46553c6-ad87-454b-b5f5-a1c468d78c1e",
					ProviderID:   "kubernetes_external",
					Status:       commonParams.InstanceRunning,
					RunnerStatus: params.RunnerIdle,
				}
				response := instances.NewGetInstanceOK()
				response.Payload = runner
				m.GetInstanceByName(instances.NewGetInstanceParams().WithInstanceName("road-runner-k8s-fy5snjcv5dzn")).Return(response, nil)

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
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
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
			name: "Delete Runner CR",
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
				response := instances.NewGetInstanceDefault(404)
				m.GetInstanceByName(instances.NewGetInstanceParams().WithInstanceName("road-runner-k8s-fy5snjcv5dzn")).Return(nil, response)
			},
			wantErr:        false,
			expectedObject: &garmoperatorv1alpha1.Runner{},
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
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
				Recorder: record.NewFakeRecorder(3),
			}

			mockInstanceClient := mock.NewMockInstanceClient(mockCtrl)
			tt.expectGarmRequest(mockInstanceClient.EXPECT())

			_, err = reconciler.reconcile(context.Background(), tt.req, mockInstanceClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunnerReconciler.reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			runners := &garmoperatorv1alpha1.RunnerList{}
			err = client.List(context.Background(), runners)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunnerReconciler.reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, 0, len(runners.Items))
		})
	}
}

func TestRunnerReconciler_reconcileCleanupCR(t *testing.T) {
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
			name: "Cleanup Runner CR",
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
				response := instances.NewGetInstanceDefault(404)
				m.GetInstanceByName(instances.NewGetInstanceParams().WithInstanceName("road-runner-k8s-fy5snjcv5dzn")).Return(nil, response)
			},
			wantErr:        false,
			expectedObject: &garmoperatorv1alpha1.Runner{},
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
				BaseURL:  "http://domain.does.not.exist:9997",
				Username: "admin",
				Password: "admin",
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
