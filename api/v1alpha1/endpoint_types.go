// SPDX-License-Identifier: MIT

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EndpointSpec defines the desired state of Endpoint
type EndpointSpec struct {
	Description   string `json:"description,omitempty"`
	APIBaseURL    string `json:"apiBaseUrl,omitempty"`
	UploadBaseURL string `json:"uploadBaseUrl,omitempty"`
	BaseURL       string `json:"baseUrl,omitempty"`
	CACertBundle  []byte `json:"caCertBundle,omitempty"`
}

// EndpointStatus defines the observed state of Endpoint
type EndpointStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=endpoints,scope=Namespaced,categories=garm,shortName=gep
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.apiBaseUrl",description="API Base URL"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].message",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of Endpoint"

// Endpoint is the Schema for the endpoints API
type Endpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EndpointSpec   `json:"spec,omitempty"`
	Status EndpointStatus `json:"status,omitempty"`
}

func (e *Endpoint) SetConditions(conditions []metav1.Condition) {
	e.Status.Conditions = conditions
}

func (e *Endpoint) GetConditions() []metav1.Condition {
	return e.Status.Conditions
}

//+kubebuilder:object:root=true

// EndpointList contains a list of Endpoint
type EndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Endpoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Endpoint{}, &EndpointList{})
}
