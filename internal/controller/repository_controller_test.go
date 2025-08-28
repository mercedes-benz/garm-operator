// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/cloudbase/garm/client/repositories"
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
)

func TestRepositoryReconciler_reconcileNormal(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		runtimeObjects    []runtime.Object
		expectGarmRequest func(m *mock.MockRepositoryClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1beta1.Repository
	}{
		{
			name: "repository exist - update",
			object: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "github-creds",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1beta1.RepositoryStatus{
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
				&garmoperatorv1beta1.GitHubCredential{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github-creds",
						Namespace: "default",
					},
					Spec: garmoperatorv1beta1.GitHubCredentialSpec{
						Description: "github-creds",
						EndpointRef: corev1.TypedLocalObjectReference{},
						AuthType:    "pat",
						SecretRef: garmoperatorv1beta1.SecretRef{
							Name: "github-secret",
							Key:  "token",
						},
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "github-creds",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1beta1.RepositoryStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Message:            "Pool Manager is not running",
						},
						{
							Type:               string(conditions.GithubCredentialsReference),
							Reason:             string(conditions.FetchingGithubCredentialsRefSuccessReason),
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
							Type:               string(conditions.WebhookSecretReference),
							Reason:             string(conditions.FetchingWebhookSecretRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockRepositoryClientMockRecorder) {
				m.ListRepositories(repositories.NewListReposParams()).Return(&repositories.ListReposOK{Payload: params.Repositories{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-repository",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.UpdateRepository(repositories.NewUpdateRepoParams().
					WithRepoID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					})).Return(&repositories.UpdateRepoOK{
					Payload: params.Repository{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-repository",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "repository exist but spec has changed - update",
			object: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "has-changed",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1beta1.RepositoryStatus{
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
				&garmoperatorv1beta1.GitHubCredential{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "has-changed",
						Namespace: "default",
					},
					Spec: garmoperatorv1beta1.GitHubCredentialSpec{
						Description: "github-creds",
						EndpointRef: corev1.TypedLocalObjectReference{},
						AuthType:    "pat",
						SecretRef: garmoperatorv1beta1.SecretRef{
							Name: "github-secret",
							Key:  "token",
						},
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "has-changed",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1beta1.RepositoryStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Message:            "Pool Manager is not running",
						},
						{
							Type:               string(conditions.GithubCredentialsReference),
							Reason:             string(conditions.FetchingGithubCredentialsRefSuccessReason),
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
							Type:               string(conditions.WebhookSecretReference),
							Reason:             string(conditions.FetchingWebhookSecretRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockRepositoryClientMockRecorder) {
				m.ListRepositories(repositories.NewListReposParams()).Return(&repositories.ListReposOK{Payload: params.Repositories{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-repository",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.UpdateRepository(repositories.NewUpdateRepoParams().
					WithRepoID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "has-changed",
						WebhookSecret:   "has-changed",
					})).Return(&repositories.UpdateRepoOK{
					Payload: params.Repository{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-repository",
						Owner:           "test-repo",
						CredentialsName: "has-changed",
						WebhookSecret:   "has-changed",
					},
				}, nil)
			},
		},
		{
			name: "repository exist but pool status has changed - update",
			object: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "github-creds",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1beta1.RepositoryStatus{
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
				&garmoperatorv1beta1.GitHubCredential{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github-creds",
						Namespace: "default",
					},
					Spec: garmoperatorv1beta1.GitHubCredentialSpec{
						Description: "github-creds",
						EndpointRef: corev1.TypedLocalObjectReference{},
						AuthType:    "pat",
						SecretRef: garmoperatorv1beta1.SecretRef{
							Name: "github-secret",
							Key:  "token",
						},
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "github-creds",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1beta1.RepositoryStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Message:            "Pool Manager is not running",
						},
						{
							Type:               string(conditions.GithubCredentialsReference),
							Reason:             string(conditions.FetchingGithubCredentialsRefSuccessReason),
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
							Type:               string(conditions.WebhookSecretReference),
							Reason:             string(conditions.FetchingWebhookSecretRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockRepositoryClientMockRecorder) {
				m.ListRepositories(repositories.NewListReposParams()).Return(&repositories.ListReposOK{Payload: params.Repositories{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-repository",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.UpdateRepository(repositories.NewUpdateRepoParams().
					WithRepoID("e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					})).Return(&repositories.UpdateRepoOK{
					Payload: params.Repository{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-repository",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
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
			name: "repository does not exist - create and update",
			object: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "github-creds",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
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
				&garmoperatorv1beta1.GitHubCredential{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github-creds",
						Namespace: "default",
					},
					Spec: garmoperatorv1beta1.GitHubCredentialSpec{
						Description: "github-creds",
						EndpointRef: corev1.TypedLocalObjectReference{},
						AuthType:    "pat",
						SecretRef: garmoperatorv1beta1.SecretRef{
							Name: "github-secret",
							Key:  "token",
						},
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "github-creds",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1beta1.RepositoryStatus{
					ID: "9e0da3cb-130b-428d-aa8a-e314d955060e",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Message:            "Pool Manager is not running",
						},
						{
							Type:               string(conditions.GithubCredentialsReference),
							Reason:             string(conditions.FetchingGithubCredentialsRefSuccessReason),
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
							Type:               string(conditions.WebhookSecretReference),
							Reason:             string(conditions.FetchingWebhookSecretRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockRepositoryClientMockRecorder) {
				m.ListRepositories(repositories.NewListReposParams()).Return(&repositories.ListReposOK{Payload: params.Repositories{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-ae78a8f27a3e",
						Name:            "existing-repository",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					},
				}}, nil)
				m.CreateRepository(repositories.NewCreateRepoParams().WithBody(
					params.CreateRepoParams{
						Name:            "new-repository",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
						Owner:           "test-repo",
					})).Return(&repositories.CreateRepoOK{
					Payload: params.Repository{
						Name:            "new-repository",
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					},
				}, nil)
				m.UpdateRepository(repositories.NewUpdateRepoParams().
					WithRepoID("9e0da3cb-130b-428d-aa8a-e314d955060e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					})).Return(&repositories.UpdateRepoOK{
					Payload: params.Repository{
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
						Name:            "new-repository",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "repository already exist in garm - update",
			object: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "github-creds",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
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
				&garmoperatorv1beta1.GitHubCredential{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github-creds",
						Namespace: "default",
					},
					Spec: garmoperatorv1beta1.GitHubCredentialSpec{
						Description: "github-creds",
						EndpointRef: corev1.TypedLocalObjectReference{},
						AuthType:    "pat",
						SecretRef: garmoperatorv1beta1.SecretRef{
							Name: "github-secret",
							Key:  "token",
						},
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "github-creds",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1beta1.RepositoryStatus{
					ID: "e1dbf9a6-a9f6-4594-a5ac-12345",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Message:            "Pool Manager is not running",
						},
						{
							Type:               string(conditions.GithubCredentialsReference),
							Reason:             string(conditions.FetchingGithubCredentialsRefSuccessReason),
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
							Type:               string(conditions.WebhookSecretReference),
							Reason:             string(conditions.FetchingWebhookSecretRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockRepositoryClientMockRecorder) {
				m.ListRepositories(repositories.NewListReposParams()).Return(&repositories.ListReposOK{Payload: params.Repositories{
					{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "new-repository",
						Owner:           "test-repo",
						CredentialsName: "totally-insecure",
					},
				}}, nil)
				m.UpdateRepository(repositories.NewUpdateRepoParams().
					WithRepoID("e1dbf9a6-a9f6-4594-a5ac-12345").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					})).Return(&repositories.UpdateRepoOK{
					Payload: params.Repository{
						ID:              "e1dbf9a6-a9f6-4594-a5ac-12345",
						Name:            "new-repository",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "repository does not exist in garm - create and update",
			object: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "github-creds",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1beta1.RepositoryStatus{
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
				&garmoperatorv1beta1.GitHubCredential{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github-creds",
						Namespace: "default",
					},
					Spec: garmoperatorv1beta1.GitHubCredentialSpec{
						Description: "github-creds",
						EndpointRef: corev1.TypedLocalObjectReference{},
						AuthType:    "pat",
						SecretRef: garmoperatorv1beta1.SecretRef{
							Name: "github-secret",
							Key:  "token",
						},
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
					CredentialsRef: corev1.TypedLocalObjectReference{
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Kind:     "GitHubCredential",
						Name:     "github-creds",
					},
					Owner: "test-repo",
					WebhookSecretRef: garmoperatorv1beta1.SecretRef{
						Name: "my-webhook-secret",
						Key:  "webhookSecret",
					},
				},
				Status: garmoperatorv1beta1.RepositoryStatus{
					ID: "9e0da3cb-130b-428d-aa8a-e314d955060e",
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.PoolManagerFailureReason),
							Status:             metav1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(time.Now()),
							Message:            "Pool Manager is not running",
						},
						{
							Type:               string(conditions.GithubCredentialsReference),
							Reason:             string(conditions.FetchingGithubCredentialsRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.PoolManager),
							Status:             metav1.ConditionFalse,
							Message:            "",
							Reason:             string(conditions.PoolManagerFailureReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.WebhookSecretReference),
							Status:             metav1.ConditionTrue,
							Message:            "",
							Reason:             string(conditions.FetchingWebhookSecretRefSuccessReason),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockRepositoryClientMockRecorder) {
				m.ListRepositories(repositories.NewListReposParams()).Return(&repositories.ListReposOK{Payload: params.Repositories{
					{},
				}}, nil)
				m.CreateRepository(repositories.NewCreateRepoParams().WithBody(
					params.CreateRepoParams{
						Name:            "existing-repository",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
						Owner:           "test-repo",
					})).Return(&repositories.CreateRepoOK{
					Payload: params.Repository{
						Name:            "existing-repository",
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					},
				}, nil)
				m.UpdateRepository(repositories.NewUpdateRepoParams().
					WithRepoID("9e0da3cb-130b-428d-aa8a-e314d955060e").
					WithBody(params.UpdateEntityParams{
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					})).Return(&repositories.UpdateRepoOK{
					Payload: params.Repository{
						ID:              "9e0da3cb-130b-428d-aa8a-e314d955060e",
						Name:            "existing-repository",
						Owner:           "test-repo",
						CredentialsName: "github-creds",
						WebhookSecret:   "foobar",
					},
				}, nil)
			},
		},
		{
			name: "secret ref not found condition",
			object: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
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
				Status: garmoperatorv1beta1.RepositoryStatus{},
			},
			runtimeObjects: []runtime.Object{},
			expectedObject: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
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
				Status: garmoperatorv1beta1.RepositoryStatus{
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.FetchingWebhookSecretRefFailedReason),
							Status:             metav1.ConditionFalse,
							Message:            "secrets \"my-webhook-secret\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.GithubCredentialsReference),
							Reason:             string(conditions.UnknownReason),
							Status:             metav1.ConditionUnknown,
							Message:            conditions.CredentialsNotReconciledYetMsg,
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.PoolManager),
							Reason:             string(conditions.UnknownReason),
							Status:             metav1.ConditionUnknown,
							Message:            conditions.GarmServerNotReconciledYetMsg,
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.WebhookSecretReference),
							Reason:             string(conditions.FetchingWebhookSecretRefFailedReason),
							Status:             metav1.ConditionFalse,
							Message:            "secrets \"my-webhook-secret\" not found",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
				},
			},
			expectGarmRequest: func(_ *mock.MockRepositoryClientMockRecorder) {},
			wantErr:           true,
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1beta1.Repository{}).Build()

			// create a fake reconciler
			reconciler := &RepositoryReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			repository := tt.object.DeepCopyObject().(*garmoperatorv1beta1.Repository)

			mockRepository := mock.NewMockRepositoryClient(mockCtrl)
			tt.expectGarmRequest(mockRepository.EXPECT())

			_, err = reconciler.reconcileNormal(context.Background(), mockRepository, repository)
			if (err != nil) != tt.wantErr {
				t.Errorf("RepositoryReconciler.reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// clear out annotations to avoid comparison errors
			repository.ObjectMeta.Annotations = nil

			// empty resource version to avoid comparison errors
			repository.ObjectMeta.ResourceVersion = ""

			// clear conditions lastTransitionTime to avoid comparison errors
			conditions.NilLastTransitionTime(tt.expectedObject)
			conditions.NilLastTransitionTime(repository)

			if !reflect.DeepEqual(repository, tt.expectedObject) {
				t.Errorf("RepositoryReconciler.reconcileNormal() \ngot = %#v\n want %#v", repository, tt.expectedObject)
			}
		})
	}
}

func TestRepositoryReconciler_reconcileDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		runtimeObjects    []runtime.Object
		expectGarmRequest func(m *mock.MockRepositoryClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1beta1.Repository
	}{
		{
			name: "delete repository",
			object: &garmoperatorv1beta1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "delete-repository",
					Namespace: "default",
					Finalizers: []string{
						key.RepositoryFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.RepositorySpec{
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
				Status: garmoperatorv1beta1.RepositoryStatus{
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
				&garmoperatorv1beta1.GitHubCredential{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "github-creds",
						Namespace: "default",
					},
					Spec: garmoperatorv1beta1.GitHubCredentialSpec{
						Description: "github-creds",
						EndpointRef: corev1.TypedLocalObjectReference{},
						AuthType:    "pat",
						SecretRef: garmoperatorv1beta1.SecretRef{
							Name: "github-secret",
							Key:  "token",
						},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockRepositoryClientMockRecorder) {
				m.DeleteRepository(
					repositories.NewDeleteRepoParams().
						WithRepoID("e1dbf9a6-a9f6-4594-a5ac-12345"),
				).Return(nil)
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1beta1.Repository{}).Build()

			// create a fake reconciler
			reconciler := &RepositoryReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			repository := tt.object.DeepCopyObject().(*garmoperatorv1beta1.Repository)

			mockRepository := mock.NewMockRepositoryClient(mockCtrl)
			tt.expectGarmRequest(mockRepository.EXPECT())

			_, err = reconciler.reconcileDelete(context.Background(), mockRepository, repository)
			if (err != nil) != tt.wantErr {
				t.Errorf("RepositoryReconciler.reconcileDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// check for mandatory finalizer
			if controllerutil.ContainsFinalizer(repository, key.RepositoryFinalizerName) {
				t.Errorf("RepositoryReconciler.Reconcile() finalizer still exist")
				return
			}
		})
	}
}
