// SPDX-License-Identifier: MIT

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitHubEndpointSpec defines the desired state of GitHubEndpoint
type GitHubEndpointSpec struct {
	Description   string `json:"description,omitempty"`
	APIBaseURL    string `json:"apiBaseUrl,omitempty"`
	UploadBaseURL string `json:"uploadBaseUrl,omitempty"`
	BaseURL       string `json:"baseUrl,omitempty"`
	//nolint:godox
	// TODO: This should be a secret reference
	CACertBundle []byte `json:"caCertBundle,omitempty"`
}

// GitHubEndpointStatus defines the observed state of GitHubEndpoint
type GitHubEndpointStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=githubendpoints,scope=Namespaced,categories=garm,shortName=gep
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.apiBaseUrl",description="API Base URL"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].message",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of GitHubEndpoint"

// GitHubEndpoint is the Schema for the githubendpoints API
type GitHubEndpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitHubEndpointSpec   `json:"spec,omitempty"`
	Status GitHubEndpointStatus `json:"status,omitempty"`
}

func (e *GitHubEndpoint) SetConditions(conditions []metav1.Condition) {
	e.Status.Conditions = conditions
}

func (e *GitHubEndpoint) GetConditions() []metav1.Condition {
	return e.Status.Conditions
}

//+kubebuilder:object:root=true

// GitHubEndpointList contains a list of GitHubEndpoint
type GitHubEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitHubEndpoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitHubEndpoint{}, &GitHubEndpointList{})
}
