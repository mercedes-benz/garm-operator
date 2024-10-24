// SPDX-License-Identifier: MIT

package v1beta1

import (
	"github.com/cloudbase/garm/params"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitHubCredentialSpec defines the desired state of GitHubCredential
type GitHubCredentialSpec struct {
	Description string                           `json:"description"`
	EndpointRef corev1.TypedLocalObjectReference `json:"endpointRef"`

	// either pat or app
	AuthType params.GithubAuthType `json:"authType"`

	// if AuthType is app
	AppID          int64 `json:"appId,omitempty"`
	InstallationID int64 `json:"installationId,omitempty"`

	// containing either privateKey or pat token
	SecretRef SecretRef `json:"secretRef,omitempty"`
}

// GitHubCredentialStatus defines the observed state of GitHubCredential
type GitHubCredentialStatus struct {
	ID            int64    `json:"id"`
	APIBaseURL    string   `json:"apiBaseUrl"`
	UploadBaseURL string   `json:"uploadBaseUrl"`
	BaseURL       string   `json:"baseUrl"`
	Repositories  []string `json:"repositories,omitempty"`
	Organizations []string `json:"organizations,omitempty"`
	Enterprises   []string `json:"enterprises,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=githubcredentials,scope=Namespaced,categories=garm,shortName=creds
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Credentials ID"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].message",priority=1
//+kubebuilder:printcolumn:name="AuthType",type="string",JSONPath=`.spec.authType`,description="Authentication type"
//+kubebuilder:printcolumn:name="GitHubEndpoint",type="string",JSONPath=`.spec.endpointRef.name`,description="GitHubEndpoint name these credentials are tied to"
//+kubebuilder:printcolumn:name="Repositories",type="string",JSONPath=`.status.repositories`,description="Repositories these credentials are tied to",priority=1
//+kubebuilder:printcolumn:name="Organizations",type="string",JSONPath=`.status.organizations`,description="Organizations these credentials are tied to",priority=1
//+kubebuilder:printcolumn:name="Enterprises",type="string",JSONPath=`.status.enterprises`,description="Enterprises these credentials are tied to",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of GitHubCredential"

// GitHubCredential is the Schema for the githubcredential API
type GitHubCredential struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitHubCredentialSpec   `json:"spec,omitempty"`
	Status GitHubCredentialStatus `json:"status,omitempty"`
}

func (g *GitHubCredential) SetConditions(conditions []metav1.Condition) {
	g.Status.Conditions = conditions
}

func (g *GitHubCredential) GetConditions() []metav1.Condition {
	return g.Status.Conditions
}

//+kubebuilder:object:root=true

// GitHubCredentialList contains a list of GitHubCredential
type GitHubCredentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitHubCredential `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitHubCredential{}, &GitHubCredentialList{})
}
