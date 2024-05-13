// SPDX-License-Identifier: MIT

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mercedes-benz/garm-operator/pkg/util/conditions"
)

// OrganizationSpec defines the desired state of Organization
type OrganizationSpec struct {
	CredentialsName string `json:"credentialsName"`

	// WebhookSecretRef represents a secret that should be used for the webhook
	WebhookSecretRef SecretRef `json:"webhookSecretRef"`
}

// OrganizationStatus defines the observed state of Organization
type OrganizationStatus struct {
	ID         string             `json:"id"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=organizations,scope=Namespaced,categories=garm,shortName=org
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Organization ID"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].message",priority=1
//+kubebuilder:printcolumn:name="Pool_Manager_Failure",type="string",JSONPath=`.status.conditions[?(@.reason=='PoolManagerFailure')].message`,priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of Organization"

// Organization is the Schema for the organizations API
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec,omitempty"`
	Status OrganizationStatus `json:"status,omitempty"`
}

func (o *Organization) SetConditions(conditions []metav1.Condition) {
	o.Status.Conditions = conditions
}

func (o *Organization) GetConditions() []metav1.Condition {
	return o.Status.Conditions
}

func (o *Organization) GetCredentialsName() string {
	return o.Spec.CredentialsName
}

func (o *Organization) GetID() string {
	return o.Status.ID
}

func (o *Organization) GetName() string {
	return o.ObjectMeta.Name
}

func (o *Organization) GetPoolManagerIsRunning() bool {
	condition := conditions.Get(o, conditions.PoolManager)
	if condition == nil {
		return false
	}

	return condition.Status == TrueAsString
}

func (o *Organization) GetPoolManagerFailureReason() string {
	condition := conditions.Get(o, conditions.PoolManager)
	if condition == nil {
		return ""
	}

	if condition.Reason == string(conditions.PoolManagerFailureReason) {
		return condition.Message
	}

	return ""
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
