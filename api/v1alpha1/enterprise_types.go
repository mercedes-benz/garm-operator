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

// EnterpriseSpec defines the desired state of Enterprise
type EnterpriseSpec struct {
	CredentialsName  string    `json:"credentialsName"`
	WebhookSecretRef SecretRef `json:"webhookSecretRef"`
}

// EnterpriseStatus defines the observed state of Enterprise
type EnterpriseStatus struct {
	ID                       string `json:"id"`
	PoolManagerIsRunning     bool   `json:"poolManagerIsRunning"`
	PoolManagerFailureReason string `json:"poolManagerFailureReason,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=enterprises,scope=Namespaced,categories=garm,shortName=ent
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Enterprise ID"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.poolManagerIsRunning",description="Status of the referenced pool"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.poolManagerFailureReason",description="Error description",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of Enterprise"

// Enterprise is the Schema for the enterprises API
type Enterprise struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnterpriseSpec   `json:"spec,omitempty"`
	Status EnterpriseStatus `json:"status,omitempty"`
}

func (e *Enterprise) GetCredentialsName() string {
	return e.Spec.CredentialsName
}

func (e *Enterprise) GetID() string {
	return e.Status.ID
}

func (e *Enterprise) GetPoolManagerIsRunning() bool {
	return e.Status.PoolManagerIsRunning
}

func (e *Enterprise) GetPoolManagerFailureReason() string {
	return e.Status.PoolManagerFailureReason
}

func (e *Enterprise) GetKind() string {
	return e.Kind
}

//+kubebuilder:object:root=true

// EnterpriseList contains a list of Enterprise
type EnterpriseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Enterprise `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Enterprise{}, &EnterpriseList{})
}
