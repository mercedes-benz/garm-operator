// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/cloudbase/garm/client/credentials"
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

func TestGitHubCredentialReconciler_reconcileNormal(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		expectGarmRequest func(m *mock.MockCredentialsClientMockRecorder)
		runtimeObjects    []runtime.Object
		wantErr           bool
		expectedObject    *garmoperatorv1beta1.GitHubCredential
	}{
		{
			name: "github-credential exist - update",
			object: &garmoperatorv1beta1.GitHubCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-credential",
					Namespace: "default",
					Finalizers: []string{
						key.CredentialsFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubCredentialSpec{
					Description: "existing-github-credential",
					AuthType:    params.GithubAuthTypePAT,
					SecretRef: garmoperatorv1beta1.SecretRef{
						Name: "github-token",
						Key:  "token",
					},
					EndpointRef: corev1.TypedLocalObjectReference{
						Kind:     "Endpoint",
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Name:     "existing-github-endpoint",
					},
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "github-token",
					},
					Data: map[string][]byte{
						"token": []byte("foobar"),
					},
				},
				&garmoperatorv1beta1.GitHubEndpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-github-endpoint",
						Namespace: "default",
						Finalizers: []string{
							key.GitHubEndpointFinalizerName,
						},
					},
					Spec: garmoperatorv1beta1.GitHubEndpointSpec{
						Description:           "existing-github-endpoint",
						APIBaseURL:            "https://api.github.com",
						UploadBaseURL:         "https://uploads.github.com",
						BaseURL:               "https://github.com",
						CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{},
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.GitHubCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-credential",
					Namespace: "default",
					Finalizers: []string{
						key.CredentialsFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubCredentialSpec{
					Description: "existing-github-credential",
					AuthType:    params.GithubAuthTypePAT,
					SecretRef: garmoperatorv1beta1.SecretRef{
						Name: "github-token",
						Key:  "token",
					},
					EndpointRef: corev1.TypedLocalObjectReference{
						Kind:     "Endpoint",
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Name:     "existing-github-endpoint",
					},
				},
				Status: garmoperatorv1beta1.GitHubCredentialStatus{
					ID:            1,
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					Repositories:  []string{"foobar-repo", "foobar-repo1"},
					Organizations: nil,
					Enterprises:   nil,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.EndpointReference),
							Reason:             string(conditions.FetchingEndpointRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched GitHubEndpoint CR Ref",
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
			expectGarmRequest: func(m *mock.MockCredentialsClientMockRecorder) {
				m.ListCredentials(credentials.NewListCredentialsParams()).Return(&credentials.ListCredentialsOK{
					Payload: []params.GithubCredentials{
						{
							ID:                 1,
							Name:               "existing-github-credential",
							Description:        "existing-github-credential",
							APIBaseURL:         "https://api.github.com",
							UploadBaseURL:      "https://uploads.github.com",
							BaseURL:            "https://github.com",
							CABundle:           nil,
							AuthType:           params.GithubAuthTypePAT,
							Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
							Organizations:      nil,
							Enterprises:        nil,
							Endpoint:           params.GithubEndpoint{},
							CredentialsPayload: nil,
						},
					},
				}, nil)
				m.GetCredentials(credentials.NewGetCredentialsParams().WithID(1)).Return(&credentials.GetCredentialsOK{
					Payload: params.GithubCredentials{
						ID:                 1,
						Name:               "existing-github-credential",
						Description:        "existing-github-credential",
						APIBaseURL:         "https://api.github.com",
						UploadBaseURL:      "https://uploads.github.com",
						BaseURL:            "https://github.com",
						CABundle:           nil,
						AuthType:           params.GithubAuthTypePAT,
						Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
						Organizations:      nil,
						Enterprises:        nil,
						Endpoint:           params.GithubEndpoint{},
						CredentialsPayload: nil,
					},
				}, nil)
				m.UpdateCredentials(credentials.NewUpdateCredentialsParams().
					WithID(1).
					WithBody(params.UpdateGithubCredentialsParams{
						Name:        util.StringPtr("existing-github-credential"),
						Description: util.StringPtr("existing-github-credential"),
						PAT: &params.GithubPAT{
							OAuth2Token: "foobar",
						},
					})).Return(&credentials.UpdateCredentialsOK{
					Payload: params.GithubCredentials{
						ID:                 1,
						Name:               "existing-github-credential",
						Description:        "existing-github-credential",
						APIBaseURL:         "https://api.github.com",
						UploadBaseURL:      "https://uploads.github.com",
						BaseURL:            "https://github.com",
						CABundle:           nil,
						AuthType:           params.GithubAuthTypePAT,
						Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
						Organizations:      nil,
						Enterprises:        nil,
						Endpoint:           params.GithubEndpoint{},
						CredentialsPayload: nil,
					},
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "github-credential exist but spec has changed - update",
			object: &garmoperatorv1beta1.GitHubCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-credential",
					Namespace: "default",
					Finalizers: []string{
						key.CredentialsFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubCredentialSpec{
					Description: "updated-github-credential",
					AuthType:    params.GithubAuthTypePAT,
					SecretRef: garmoperatorv1beta1.SecretRef{
						Name: "github-token",
						Key:  "token",
					},
					EndpointRef: corev1.TypedLocalObjectReference{
						Kind:     "Endpoint",
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Name:     "existing-github-endpoint",
					},
				},
			},
			runtimeObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "github-token",
					},
					Data: map[string][]byte{
						"token": []byte("new-foobar"),
					},
				},
				&garmoperatorv1beta1.GitHubEndpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-github-endpoint",
						Namespace: "default",
						Finalizers: []string{
							key.GitHubEndpointFinalizerName,
						},
					},
					Spec: garmoperatorv1beta1.GitHubEndpointSpec{
						Description:           "existing-github-endpoint",
						APIBaseURL:            "https://api.github.com",
						UploadBaseURL:         "https://uploads.github.com",
						BaseURL:               "https://github.com",
						CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{},
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.GitHubCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-credential",
					Namespace: "default",
					Finalizers: []string{
						key.CredentialsFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubCredentialSpec{
					Description: "updated-github-credential",
					AuthType:    params.GithubAuthTypePAT,
					SecretRef: garmoperatorv1beta1.SecretRef{
						Name: "github-token",
						Key:  "token",
					},
					EndpointRef: corev1.TypedLocalObjectReference{
						Kind:     "Endpoint",
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Name:     "existing-github-endpoint",
					},
				},
				Status: garmoperatorv1beta1.GitHubCredentialStatus{
					ID:            1,
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					Repositories:  []string{"foobar-repo", "foobar-repo1"},
					Organizations: nil,
					Enterprises:   nil,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.EndpointReference),
							Reason:             string(conditions.FetchingEndpointRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched GitHubEndpoint CR Ref",
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
			expectGarmRequest: func(m *mock.MockCredentialsClientMockRecorder) {
				m.ListCredentials(credentials.NewListCredentialsParams()).Return(&credentials.ListCredentialsOK{
					Payload: []params.GithubCredentials{
						{
							ID:                 1,
							Name:               "existing-github-credential",
							Description:        "existing-github-credential",
							APIBaseURL:         "https://api.github.com",
							UploadBaseURL:      "https://uploads.github.com",
							BaseURL:            "https://github.com",
							CABundle:           nil,
							AuthType:           params.GithubAuthTypePAT,
							Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
							Organizations:      nil,
							Enterprises:        nil,
							Endpoint:           params.GithubEndpoint{},
							CredentialsPayload: nil,
						},
					},
				}, nil)
				m.GetCredentials(credentials.NewGetCredentialsParams().
					WithID(1)).
					Return(&credentials.GetCredentialsOK{
						Payload: params.GithubCredentials{
							ID:                 1,
							Name:               "existing-github-credential",
							Description:        "existing-github-credential",
							APIBaseURL:         "https://api.github.com",
							UploadBaseURL:      "https://uploads.github.com",
							BaseURL:            "https://github.com",
							CABundle:           nil,
							AuthType:           params.GithubAuthTypePAT,
							Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
							Organizations:      nil,
							Enterprises:        nil,
							Endpoint:           params.GithubEndpoint{},
							CredentialsPayload: nil,
						},
					}, nil)
				m.UpdateCredentials(credentials.NewUpdateCredentialsParams().
					WithID(1).
					WithBody(params.UpdateGithubCredentialsParams{
						Name:        util.StringPtr("existing-github-credential"),
						Description: util.StringPtr("updated-github-credential"),
						PAT: &params.GithubPAT{
							OAuth2Token: "new-foobar",
						},
					})).Return(&credentials.UpdateCredentialsOK{
					Payload: params.GithubCredentials{
						ID:                 1,
						Name:               "existing-github-credential",
						Description:        "updated-github-credential",
						APIBaseURL:         "https://api.github.com",
						UploadBaseURL:      "https://uploads.github.com",
						BaseURL:            "https://github.com",
						CABundle:           nil,
						AuthType:           params.GithubAuthTypePAT,
						Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
						Organizations:      nil,
						Enterprises:        nil,
						Endpoint:           params.GithubEndpoint{},
						CredentialsPayload: nil,
					},
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "github-credential does not exist - create and update",
			object: &garmoperatorv1beta1.GitHubCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-github-credential",
					Namespace: "default",
				},
				Spec: garmoperatorv1beta1.GitHubCredentialSpec{
					Description: "new-github-credential",
					AuthType:    params.GithubAuthTypePAT,
					SecretRef: garmoperatorv1beta1.SecretRef{
						Name: "github-token",
						Key:  "token",
					},
					EndpointRef: corev1.TypedLocalObjectReference{
						Kind:     "Endpoint",
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Name:     "existing-github-endpoint",
					},
				},
			},
			expectedObject: &garmoperatorv1beta1.GitHubCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-github-credential",
					Namespace: "default",
					Finalizers: []string{
						key.CredentialsFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubCredentialSpec{
					Description: "new-github-credential",
					AuthType:    params.GithubAuthTypePAT,
					SecretRef: garmoperatorv1beta1.SecretRef{
						Name: "github-token",
						Key:  "token",
					},
					EndpointRef: corev1.TypedLocalObjectReference{
						Kind:     "Endpoint",
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Name:     "existing-github-endpoint",
					},
				},
				Status: garmoperatorv1beta1.GitHubCredentialStatus{
					ID:            2,
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					Repositories:  []string{"foobar-repo", "foobar-repo1"},
					Organizations: nil,
					Enterprises:   nil,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.EndpointReference),
							Reason:             string(conditions.FetchingEndpointRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched GitHubEndpoint CR Ref",
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
						Name:      "github-token",
					},
					Data: map[string][]byte{
						"token": []byte("foobar"),
					},
				},
				&garmoperatorv1beta1.GitHubEndpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-github-endpoint",
						Namespace: "default",
						Finalizers: []string{
							key.GitHubEndpointFinalizerName,
						},
					},
					Spec: garmoperatorv1beta1.GitHubEndpointSpec{
						Description:           "existing-github-endpoint",
						APIBaseURL:            "https://api.github.com",
						UploadBaseURL:         "https://uploads.github.com",
						BaseURL:               "https://github.com",
						CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockCredentialsClientMockRecorder) {
				m.ListCredentials(credentials.NewListCredentialsParams()).Return(&credentials.ListCredentialsOK{
					Payload: []params.GithubCredentials{
						{
							ID:                 1,
							Name:               "existing-github-credential",
							Description:        "existing-github-credential",
							APIBaseURL:         "https://api.github.com",
							UploadBaseURL:      "https://uploads.github.com",
							BaseURL:            "https://github.com",
							CABundle:           nil,
							AuthType:           params.GithubAuthTypePAT,
							Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
							Organizations:      nil,
							Enterprises:        nil,
							Endpoint:           params.GithubEndpoint{},
							CredentialsPayload: nil,
						},
					},
				}, nil)
				m.CreateCredentials(credentials.NewCreateCredentialsParams().
					WithBody(params.CreateGithubCredentialsParams{
						Name:        "new-github-credential",
						Description: "new-github-credential",
						AuthType:    params.GithubAuthTypePAT,
						PAT: params.GithubPAT{
							OAuth2Token: "foobar",
						},
						Endpoint: "existing-github-endpoint",
					})).Return(&credentials.CreateCredentialsOK{
					Payload: params.GithubCredentials{
						ID:                 2,
						Name:               "new-github-credential",
						Description:        "new-github-credential",
						APIBaseURL:         "https://api.github.com",
						UploadBaseURL:      "https://uploads.github.com",
						BaseURL:            "https://github.com",
						CABundle:           nil,
						AuthType:           params.GithubAuthTypePAT,
						Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
						Organizations:      nil,
						Enterprises:        nil,
						Endpoint:           params.GithubEndpoint{},
						CredentialsPayload: nil,
					},
				}, nil)
				m.UpdateCredentials(credentials.NewUpdateCredentialsParams().
					WithID(2).
					WithBody(params.UpdateGithubCredentialsParams{
						Name:        util.StringPtr("new-github-credential"),
						Description: util.StringPtr("new-github-credential"),
						PAT: &params.GithubPAT{
							OAuth2Token: "foobar",
						},
					})).Return(&credentials.UpdateCredentialsOK{
					Payload: params.GithubCredentials{
						ID:                 2,
						Name:               "new-github-credential",
						Description:        "new-github-credential",
						APIBaseURL:         "https://api.github.com",
						UploadBaseURL:      "https://uploads.github.com",
						BaseURL:            "https://github.com",
						CABundle:           nil,
						AuthType:           params.GithubAuthTypePAT,
						Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
						Organizations:      nil,
						Enterprises:        nil,
						Endpoint:           params.GithubEndpoint{},
						CredentialsPayload: nil,
					},
				}, nil)
				m.GetCredentials(credentials.NewGetCredentialsParams().
					WithID(2)).
					Return(&credentials.GetCredentialsOK{
						Payload: params.GithubCredentials{
							ID:                 2,
							Name:               "new-github-credential",
							Description:        "new-github-credential",
							APIBaseURL:         "https://api.github.com",
							UploadBaseURL:      "https://uploads.github.com",
							BaseURL:            "https://github.com",
							CABundle:           nil,
							AuthType:           params.GithubAuthTypePAT,
							Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
							Organizations:      nil,
							Enterprises:        nil,
							Endpoint:           params.GithubEndpoint{},
							CredentialsPayload: nil,
						},
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "github-credential update - bad request error",
			object: &garmoperatorv1beta1.GitHubCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-credential",
					Namespace: "default",
				},
				Spec: garmoperatorv1beta1.GitHubCredentialSpec{
					Description: "existing-github-credential",
					AuthType:    params.GithubAuthTypePAT,
					SecretRef: garmoperatorv1beta1.SecretRef{
						Name: "github-token",
						Key:  "token",
					},
					EndpointRef: corev1.TypedLocalObjectReference{
						Kind:     "Endpoint",
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Name:     "existing-github-endpoint",
					},
				},
				Status: garmoperatorv1beta1.GitHubCredentialStatus{
					ID:            1,
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					Repositories:  []string{"foobar-repo", "foobar-repo1"},
					Organizations: nil,
					Enterprises:   nil,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.EndpointReference),
							Reason:             string(conditions.FetchingEndpointRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched GitHubEndpoint CR Ref",
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
			expectedObject: &garmoperatorv1beta1.GitHubCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-credential",
					Namespace: "default",
					Finalizers: []string{
						key.CredentialsFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubCredentialSpec{
					Description: "existing-github-credential",
					AuthType:    params.GithubAuthTypePAT,
					SecretRef: garmoperatorv1beta1.SecretRef{
						Name: "github-token",
						Key:  "token",
					},
					EndpointRef: corev1.TypedLocalObjectReference{
						Kind:     "Endpoint",
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Name:     "existing-github-endpoint",
					},
				},
				Status: garmoperatorv1beta1.GitHubCredentialStatus{
					ID:            1,
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					Repositories:  []string{"foobar-repo", "foobar-repo1"},
					Organizations: nil,
					Enterprises:   nil,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.GarmAPIErrorReason),
							Status:             metav1.ConditionFalse,
							Message:            "[GET /github/credentials/{id}][400] getCredentialsBadRequest {\"error\":\"\",\"details\":\"\"}",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.EndpointReference),
							Reason:             string(conditions.FetchingEndpointRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched GitHubEndpoint CR Ref",
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
						Name:      "github-token",
					},
					Data: map[string][]byte{
						"token": []byte("foobar"),
					},
				},
				&garmoperatorv1beta1.GitHubEndpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-github-endpoint",
						Namespace: "default",
						Finalizers: []string{
							key.GitHubEndpointFinalizerName,
						},
					},
					Spec: garmoperatorv1beta1.GitHubEndpointSpec{
						Description:           "existing-github-endpoint",
						APIBaseURL:            "https://api.github.com",
						UploadBaseURL:         "https://uploads.github.com",
						BaseURL:               "https://github.com",
						CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockCredentialsClientMockRecorder) {
				m.ListCredentials(credentials.NewListCredentialsParams()).Return(&credentials.ListCredentialsOK{
					Payload: []params.GithubCredentials{
						{
							ID:                 1,
							Name:               "existing-github-credential",
							Description:        "existing-github-credential",
							APIBaseURL:         "https://api.github.com",
							UploadBaseURL:      "https://uploads.github.com",
							BaseURL:            "https://github.com",
							CABundle:           nil,
							AuthType:           params.GithubAuthTypePAT,
							Repositories:       []params.Repository{{ID: strconv.Itoa(1), Owner: "foo-org", Name: "foobar-repo"}, {ID: strconv.Itoa(2), Owner: "foo-org", Name: "foobar-repo1"}},
							Organizations:      nil,
							Enterprises:        nil,
							Endpoint:           params.GithubEndpoint{},
							CredentialsPayload: nil,
						},
					},
				}, nil)

				m.UpdateCredentials(credentials.NewUpdateCredentialsParams().
					WithID(1).
					WithBody(params.UpdateGithubCredentialsParams{
						Name:        util.StringPtr("existing-github-credential"),
						Description: util.StringPtr("existing-github-credential"),
						PAT: &params.GithubPAT{
							OAuth2Token: "foobar",
						},
					})).Return(nil, credentials.NewGetCredentialsBadRequest())
			},
			wantErr: true,
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1beta1.GitHubCredential{}).Build()

			// create a fake reconciler
			reconciler := &GitHubCredentialReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			GitHubCredential := tt.object.DeepCopyObject().(*garmoperatorv1beta1.GitHubCredential)

			mockGitHubCredential := mock.NewMockCredentialsClient(mockCtrl)
			tt.expectGarmRequest(mockGitHubCredential.EXPECT())

			_, err = reconciler.reconcileNormal(context.Background(), mockGitHubCredential, GitHubCredential)
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubCredentialReconciler.reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// clear out annotations to avoid comparison errors
			GitHubCredential.ObjectMeta.Annotations = nil

			// empty resource version to avoid comparison errors
			GitHubCredential.ObjectMeta.ResourceVersion = ""

			// clear conditions lastTransitionTime to avoid comparison errors
			conditions.NilLastTransitionTime(tt.expectedObject)
			conditions.NilLastTransitionTime(GitHubCredential)

			if !reflect.DeepEqual(GitHubCredential, tt.expectedObject) {
				t.Errorf("GitHubCredentialReconciler.reconcileNormal() \ngot = %#v\n want %#v", GitHubCredential, tt.expectedObject)
			}
		})
	}
}

func TestGitHubCredentialReconciler_reconcileDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		name              string
		object            runtime.Object
		runtimeObjects    []runtime.Object
		expectGarmRequest func(m *mock.MockCredentialsClientMockRecorder)
		wantErr           bool
		expectedObject    *garmoperatorv1beta1.GitHubCredential
	}{
		{
			name: "delete github-credential",
			object: &garmoperatorv1beta1.GitHubCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-github-credential",
					Namespace: "default",
					Finalizers: []string{
						key.CredentialsFinalizerName,
					},
				},
				Spec: garmoperatorv1beta1.GitHubCredentialSpec{
					Description: "existing-github-credential",
					AuthType:    params.GithubAuthTypePAT,
					SecretRef: garmoperatorv1beta1.SecretRef{
						Name: "github-token",
						Key:  "token",
					},
					EndpointRef: corev1.TypedLocalObjectReference{
						Kind:     "Endpoint",
						APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
						Name:     "existing-github-endpoint",
					},
				},
				Status: garmoperatorv1beta1.GitHubCredentialStatus{
					ID:            1,
					APIBaseURL:    "https://api.github.com",
					UploadBaseURL: "https://uploads.github.com",
					BaseURL:       "https://github.com",
					Repositories:  []string{"foobar-repo", "foobar-repo1"},
					Organizations: nil,
					Enterprises:   nil,
					Conditions: []metav1.Condition{
						{
							Type:               string(conditions.ReadyCondition),
							Reason:             string(conditions.SuccessfulReconcileReason),
							Status:             metav1.ConditionTrue,
							Message:            "",
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
						{
							Type:               string(conditions.EndpointReference),
							Reason:             string(conditions.FetchingEndpointRefSuccessReason),
							Status:             metav1.ConditionTrue,
							Message:            "Successfully fetched GitHubEndpoint CR Ref",
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
						Name:      "github-token",
					},
					Data: map[string][]byte{
						"token": []byte("foobar"),
					},
				},
				&garmoperatorv1beta1.GitHubEndpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-github-endpoint",
						Namespace: "default",
						Finalizers: []string{
							key.GitHubEndpointFinalizerName,
						},
					},
					Spec: garmoperatorv1beta1.GitHubEndpointSpec{
						Description:           "existing-github-endpoint",
						APIBaseURL:            "https://api.github.com",
						UploadBaseURL:         "https://uploads.github.com",
						BaseURL:               "https://github.com",
						CACertBundleSecretRef: garmoperatorv1beta1.SecretRef{},
					},
				},
			},
			expectGarmRequest: func(m *mock.MockCredentialsClientMockRecorder) {
				m.DeleteCredentials(
					credentials.NewDeleteCredentialsParams().
						WithID(1),
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
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).WithStatusSubresource(&garmoperatorv1beta1.GitHubCredential{}).Build()

			// create a fake reconciler
			reconciler := &GitHubCredentialReconciler{
				Client:   client,
				Recorder: record.NewFakeRecorder(3),
			}

			GitHubCredential := tt.object.DeepCopyObject().(*garmoperatorv1beta1.GitHubCredential)

			mockGitHubCredential := mock.NewMockCredentialsClient(mockCtrl)
			tt.expectGarmRequest(mockGitHubCredential.EXPECT())

			_, err = reconciler.reconcileDelete(context.Background(), mockGitHubCredential, GitHubCredential)
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubCredentialReconciler.reconcileDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// check for mandatory finalizer
			if controllerutil.ContainsFinalizer(GitHubCredential, key.CredentialsFinalizerName) {
				t.Errorf("GitHubCredentialReconciler.Reconcile() finalizer still exist")
				return
			}
		})
	}
}
