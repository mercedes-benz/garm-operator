// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/cloudbase/garm/client/endpoints"
	"github.com/cloudbase/garm/params"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/client/mock"
	"github.com/mercedes-benz/garm-operator/pkg/conditions"
	"github.com/mercedes-benz/garm-operator/pkg/util"
)

func TestGitHubEndpointReconciler_reconcileNormal(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		expectGarmRequest func(m *mock.MockEndpointClientMockRecorder)
		runtimeObjects    []runtime.Object
		wantErr           bool
		expectedObject    *garmoperatorv1beta1.GitHubEndpoint
	}{
		{
			name: "github-endpoint exist - update",
			object: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "existing-github-endpoint",
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "gh-endpoint-ca-cert-bundle",
						Key:  "caCertBundle",
					},
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "gh-endpoint-ca-cert-bundle",
					},
					Data: map[string][]byte{
						"caCertBundle": []byte("foobar"),
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "existing-github-endpoint",
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "gh-endpoint-ca-cert-bundle",
						Key:  "caCertBundle",
					},
				},
				Status: garmoperatorv1beta1.GitHubEndpointStatus{
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
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
			expectGarmRequest: func(m *mock.MockEndpointClientMockRecorder) {
				m.GetEndpoint(endpoints.NewGetGithubEndpointParams().WithName("existing-github-endpoint")).Return(&endpoints.GetGithubEndpointOK{
					Payload: params.GithubEndpoint{
						Name:          "existing-github-endpoint",
						Description:   "existing-github-endpoint",
						APIBaseURL:    "https://api.github.com",
						UploadBaseURL: "https://uploads.github.com",
						BaseURL:       "https://github.com",
						CACertBundle:  nil,
					},
				}, nil)
				m.UpdateEndpoint(endpoints.NewUpdateGithubEndpointParams().
					WithName("existing-github-endpoint").
					WithBody(params.UpdateGithubEndpointParams{
						Description:   util.StringPtr("existing-github-endpoint"),
						APIBaseURL:    util.StringPtr("https://api.github.com"),
						UploadBaseURL: util.StringPtr("https://uploads.github.com"),
						BaseURL:       util.StringPtr("https://github.com"),
						CACertBundle:  []byte("foobar"),
					})).Return(&endpoints.UpdateGithubEndpointOK{
					Payload: params.GithubEndpoint{
						Name:          "existing-github-endpoint",
						Description:   "existing-github-endpoint",
						APIBaseURL:    "https://api.github.com",
						UploadBaseURL: "https://uploads.github.com",
						BaseURL:       "https://github.com",
						CACertBundle:  []byte("foobar"),
					},
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "github-endpoint exist but spec has changed - update",
			object: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "has-changed",
					APIBaseURL:    "https://api.github-enterprise.com",
					UploadBaseURL: "https://uploads.github-enterprise.com",
					BaseURL:       "https://github-enterprise.com",
					CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "gh-endpoint-ca-cert-bundle",
						Key:  "caCertBundle",
					},
				},
				Status: garmoperatorv1beta1.GitHubEndpointStatus{
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
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
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "gh-endpoint-ca-cert-bundle",
					},
					Data: map[string][]byte{
						"caCertBundle": []byte("foobar"),
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "has-changed",
					APIBaseURL:    "https://api.github-enterprise.com",
					UploadBaseURL: "https://uploads.github-enterprise.com",
					BaseURL:       "https://github-enterprise.com",
					CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "gh-endpoint-ca-cert-bundle",
						Key:  "caCertBundle",
					},
				},
				Status: garmoperatorv1beta1.GitHubEndpointStatus{
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
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
			expectGarmRequest: func(m *mock.MockEndpointClientMockRecorder) {
				m.GetEndpoint(endpoints.NewGetGithubEndpointParams().
					WithName("existing-github-endpoint")).
					Return(&endpoints.GetGithubEndpointOK{
						Payload: params.GithubEndpoint{
							Name:          "existing-github-endpoint",
							Description:   "existing-github-endpoint",
							APIBaseURL:    "https://api.github.com",
							UploadBaseURL: "https://uploads.github.com",
							BaseURL:       "https://github.com",
							CACertBundle:  []byte("foobar"),
						},
					}, nil)
				m.UpdateEndpoint(endpoints.NewUpdateGithubEndpointParams().
					WithName("existing-github-endpoint").
					WithBody(params.UpdateGithubEndpointParams{
						Description:   util.StringPtr("has-changed"),
						APIBaseURL:    util.StringPtr("https://api.github-enterprise.com"),
						UploadBaseURL: util.StringPtr("https://uploads.github-enterprise.com"),
						BaseURL:       util.StringPtr("https://github-enterprise.com"),
						CACertBundle:  []byte("foobar"),
					})).Return(&endpoints.UpdateGithubEndpointOK{
					Payload: params.GithubEndpoint{
						Name:          "existing-github-endpoint",
						Description:   "has-changed",
						APIBaseURL:    "https://api.github-enterprise.com",
						UploadBaseURL: "https://uploads.github-enterprise.com",
						BaseURL:       "https://github-enterprise.com",
						CACertBundle:  []byte("foobar"),
					},
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "github-endpoint does not exist - create and update",
			object: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "new github endpoint",
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "gh-endpoint-ca-cert-bundle",
						Key:  "caCertBundle",
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "new github endpoint",
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "gh-endpoint-ca-cert-bundle",
						Key:  "caCertBundle",
					},
				},
				Status: garmoperatorv1beta1.GitHubEndpointStatus{
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
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
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "gh-endpoint-ca-cert-bundle",
					},
					Data: map[string][]byte{
						"caCertBundle": []byte("foobar"),
					},
				},
			},
			expectGarmRequest: func(m *mock.MockEndpointClientMockRecorder) {
				m.GetEndpoint(endpoints.NewGetGithubEndpointParams().
					WithName("new-github-endpoint")).
					Return(nil, endpoints.NewGetGithubEndpointDefault(404))
				m.CreateEndpoint(endpoints.NewCreateGithubEndpointParams().
					WithBody(params.CreateGithubEndpointParams{
						Name:          "new-github-endpoint",
						Description:   "new github endpoint",
						APIBaseURL:    "https://api.github.com",
						UploadBaseURL: "https://uploads.github.com",
						BaseURL:       "https://github.com",
						CACertBundle:  []byte("foobar"),
					})).Return(&endpoints.CreateGithubEndpointOK{
					Payload: params.GithubEndpoint{
						Name:          "new-github-endpoint",
						Description:   "new github endpoint",
						APIBaseURL:    "https://api.github.com",
						UploadBaseURL: "https://uploads.github.com",
						BaseURL:       "https://github.com",
						CACertBundle:  []byte("foobar"),
					},
				}, nil)
				m.UpdateEndpoint(endpoints.NewUpdateGithubEndpointParams().
					WithName("new-github-endpoint").
					WithBody(params.UpdateGithubEndpointParams{
						Description:   util.StringPtr("new github endpoint"),
						APIBaseURL:    util.StringPtr("https://api.github.com"),
						UploadBaseURL: util.StringPtr("https://uploads.github.com"),
						BaseURL:       util.StringPtr("https://github.com"),
						CACertBundle:  []byte("foobar"),
					})).Return(&endpoints.UpdateGithubEndpointOK{
					Payload: params.GithubEndpoint{
						Name:          "new-github-endpoint",
						Description:   "new github endpoint",
						APIBaseURL:    "https://api.github.com",
						UploadBaseURL: "https://uploads.github.com",
						BaseURL:       "https://github.com",
						CACertBundle:  []byte("foobar"),
					},
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "github-endpoint update - bad request error",
			object: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "",
					APIBaseURL:    "",
					UploadBaseURL: "",
					BaseURL:       "",
				},
			},
			expectedObject: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{},
				Status: garmoperatorv1beta1.GitHubEndpointStatus{
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.GarmAPIErrorReason),
							Status:             metav1.ConditionFalse,
							Message:            "[PUT /github/endpoints/{name}][400] UpdateGithubEndpoint default {\"error\":\"\",\"details\":\"\"}",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			runtimeObjects: []runtime.Object{},
			expectGarmRequest: func(m *mock.MockEndpointClientMockRecorder) {
				m.GetEndpoint(endpoints.NewGetGithubEndpointParams().
					WithName("existing-github-endpoint")).
					Return(&endpoints.GetGithubEndpointOK{
						Payload: params.GithubEndpoint{
							Name:          "existing-github-endpoint",
							Description:   "existing-github-endpoint",
							APIBaseURL:    "https://api.github.com",
							UploadBaseURL: "https://uploads.github.com",
							BaseURL:       "https://github.com",
							CACertBundle:  []byte("foobar"),
						},
					}, nil)
				m.UpdateEndpoint(endpoints.NewUpdateGithubEndpointParams().
					WithName("existing-github-endpoint").
					WithBody(params.UpdateGithubEndpointParams{
						Description:   util.StringPtr(""),
						APIBaseURL:    util.StringPtr(""),
						UploadBaseURL: util.StringPtr(""),
						BaseURL:       util.StringPtr(""),
						CACertBundle:  []byte(""),
					})).Return(nil, endpoints.NewUpdateGithubEndpointDefault(400))
			},
			wantErr: true,
		},
		{
			name: "github-endpoint update - no ca cert bundle secret found",
			object: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "existing-github-endpoint",
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "ca-cert-bundle",
						Key:  "caCertBundle",
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "existing-github-endpoint",
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "ca-cert-bundle",
						Key:  "caCertBundle",
					},
				},
				Status: garmoperatorv1beta1.GitHubEndpointStatus{
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.FetchingSecretRefFailedReason),
							Status:             metav1.ConditionFalse,
							Message:            "secrets \"ca-cert-bundle\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.SecretReference),
							Reason:             string(conditions.FetchingSecretRefFailedReason),
							Status:             metav1.ConditionFalse,
							Message:            "secrets \"ca-cert-bundle\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			runtimeObjects:    []runtime.Object{},
			expectGarmRequest: func(_ *mock.MockEndpointClientMockRecorder) {},
			wantErr:           true,
		},
		{
			name: "github-endpoint update - no ca cert bundle secret defined",
			object: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "existing-github-endpoint",
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
				},
			},
			expectedObject: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "existing-github-endpoint",
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
				},
				Status: garmoperatorv1beta1.GitHubEndpointStatus{
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			runtimeObjects: []runtime.Object{},
			expectGarmRequest: func(m *mock.MockEndpointClientMockRecorder) {
				m.GetEndpoint(endpoints.NewGetGithubEndpointParams().
					WithName("existing-github-endpoint")).
					Return(&endpoints.GetGithubEndpointOK{
						Payload: params.GithubEndpoint{
							Name:          "existing-github-endpoint",
							Description:   "existing-github-endpoint",
							APIBaseURL:    "https://api.github.com",
							UploadBaseURL: "https://uploads.github.com",
							BaseURL:       "https://github.com",
							CACertBundle:  []byte(""),
						},
					}, nil)
				m.UpdateEndpoint(endpoints.NewUpdateGithubEndpointParams().
					WithName("existing-github-endpoint").
					WithBody(params.UpdateGithubEndpointParams{
						Description:   util.StringPtr("existing-github-endpoint"),
						APIBaseURL:    util.StringPtr("https://api.github.com"),
						UploadBaseURL: util.StringPtr("https://uploads.github.com"),
						BaseURL:       util.StringPtr("https://github.com"),
						CACertBundle:  []byte(""),
					})).Return(&endpoints.UpdateGithubEndpointOK{
					Payload: params.GithubEndpoint{
						Name:          "existing-github-endpoint",
						Description:   "existing-github-endpoint",
						APIBaseURL:    "https://api.github.com",
						UploadBaseURL: "https://uploads.github.com",
						BaseURL:       "https://github.com",
						CACertBundle:  []byte(""),
					},
				}, nil)
			},
			wantErr: false,
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1beta1.GitHubEndpoint{}).Build()

			// create a fake reconciler
			reconciler := &GitHubEndpointReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			githubEndpoint := tt.object.DeepCopyObject().(*garmoperatorv1beta1.GitHubEndpoint)

			mockGitHubEndpoint := mock.NewMockEndpointClient(mockCtrl)
			tt.expectGarmRequest(mockGitHubEndpoint.EXPECT())

			_, err = reconciler.reconcileNormal(context.Background(), mockGitHubEndpoint, githubEndpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubEndpointReconciler.reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// clear out annotations to avoid comparison errors
			githubEndpoint.ObjectMeta.Annotations = nil

			// empty resource version to avoid comparison errors
			githubEndpoint.ObjectMeta.ResourceVersion = ""

			// clear conditions lastTransitionTime to avoid comparison errors
			conditions.NilLastTransitionTime(tt.expectedObject)
			conditions.NilLastTransitionTime(githubEndpoint)

			if !reflect.DeepEqual(githubEndpoint, tt.expectedObject) {
				t.Errorf("GitHubEndpointReconciler.reconcileNormal() \ngot = %#v\n want %#v", githubEndpoint, tt.expectedObject)
			}
		})
	}
}

func TestGitHubEndpointReconciler_reconcileDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		runtimeObjects    []runtime.Object
		expectGarmRequest func(m *mock.MockEndpointClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1beta1.GitHubEndpoint
	}{
		{
			name: "delete github-endpoint",
			object: &garmoperatorv1beta1.GitHubEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-endpoint",
					Namespace: "default",
					Finalizers: []string{
						key.GitHubEndpointFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubEndpointSpec{
					Description:   "existing-github-endpoint",
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "gh-endpoint-ca-cert-bundle",
						Key:  "caCertBundle",
					},
				},
				Status: garmoperatorv1beta1.GitHubEndpointStatus{
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
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
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "gh-endpoint-ca-cert-bundle",
					},
					Data: map[string][]byte{
						"caCertBundle": []byte("foobar"),
					},
				},
			},
			expectGarmRequest: func(m *mock.MockEndpointClientMockRecorder) {
				m.DeleteEndpoint(
					endpoints.NewDeleteGithubEndpointParams().
						WithName("existing-github-endpoint"),
				).Return(nil)
			},
			wantErr: false,
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1beta1.GitHubEndpoint{}).Build()

			// create a fake reconciler
			reconciler := &GitHubEndpointReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			githubEndpoint := tt.object.DeepCopyObject().(*garmoperatorv1beta1.GitHubEndpoint)

			mockGitHubEndpoint := mock.NewMockEndpointClient(mockCtrl)
			tt.expectGarmRequest(mockGitHubEndpoint.EXPECT())

			_, err = reconciler.reconcileDelete(context.Background(), mockGitHubEndpoint, githubEndpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubEndpointReconciler.reconcileDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// check for mandatory finalizer
			if controllerutil.ContainsFinalizer(githubEndpoint, key.GitHubEndpointFinalizerName) {
				t.Errorf("GitHubEndpointReconciler.Reconcile() finalizer still exist")
				return
			}
		})
	}
}
