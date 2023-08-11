/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OrganizationSpec defines the desired state of Organization
type OrganizationSpec struct {
	CredentialsName string `json:"credentialsName"`

	// WebhookSecretRef represents a secret that should be used for the webhook
	WebhookSecretRef SecretRef `json:"webhookSecretRef"`
}

// OrganizationStatus defines the observed state of Organization
type OrganizationStatus struct {
	ID                       string `json:"id"`
	PoolManagerIsRunning     bool   `json:"poolManagerIsRunning"`
	PoolManagerFailureReason string `json:"poolManagerFailureReason,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=organizations,scope=Namespaced,categories=garm,shortName=org
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Organization ID"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.poolManagerIsRunning",description="Status of the referenced pool"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.poolManagerFailureReason",description="Error description",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of Organization"

// Organization is the Schema for the organizations API
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec,omitempty"`
	Status OrganizationStatus `json:"status,omitempty"`
}

func (o *Organization) GetCredentialsName() string {
	return o.Spec.CredentialsName
}

func (o *Organization) GetID() string {
	return o.Status.ID
}

func (o *Organization) GetPoolManagerIsRunning() bool {
	return o.Status.PoolManagerIsRunning
}

func (o *Organization) GetPoolManagerFailureReason() string {
	return o.Status.PoolManagerFailureReason
}

func (o *Organization) GetKind() string {
	return o.Kind
}

//+kubebuilder:object:root=true

// OrganizationList contains a list of Organization
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}
