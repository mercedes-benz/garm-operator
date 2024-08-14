// SPDX-License-Identifier: MIT

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mercedes-benz/garm-operator/pkg/conditions"
)

// RepositorySpec defines the desired state of Repository
type RepositorySpec struct {
	CredentialsRef corev1.TypedLocalObjectReference `json:"credentialsRef"`
	Owner          string                           `json:"owner"`

	// WebhookSecretRef represents a secret that should be used for the webhook
	WebhookSecretRef SecretRef `json:"webhookSecretRef"`
}

// RepositoryStatus defines the observed state of Repository
type RepositoryStatus struct {
	ID         string             `json:"id"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=repositories,scope=Namespaced,categories=garm,shortName=repo
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Repository ID"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].message",priority=1
//+kubebuilder:printcolumn:name="Pool_Manager_Failure",type="string",JSONPath=`.status.conditions[?(@.reason=='PoolManagerFailure')].message`,priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of Repository"

// Repository is the Schema for the repositories API
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

func (r *Repository) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

func (r *Repository) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

func (r *Repository) GetCredentialsName() string {
	return r.Spec.CredentialsRef.Name
}

func (r *Repository) GetID() string {
	return r.Status.ID
}

func (r *Repository) GetName() string {
	return r.ObjectMeta.Name
}

func (r *Repository) GetPoolManagerIsRunning() bool {
	condition := conditions.Get(r, conditions.PoolManager)
	if condition == nil {
		return false
	}

	return condition.Status == TrueAsString
}

func (r *Repository) GetPoolManagerFailureReason() string {
	condition := conditions.Get(r, conditions.PoolManager)
	if condition == nil {
		return ""
	}

	if condition.Reason == string(conditions.PoolManagerFailureReason) {
		return condition.Message
	}

	return ""
}

func (r *Repository) GetKind() string {
	return r.Kind
}

//+kubebuilder:object:root=true

// RepositoryList contains a list of Repository
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
}
