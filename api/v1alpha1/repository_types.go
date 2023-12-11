// SPDX-License-Identifier: MIT

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepositorySpec defines the desired state of Repository
type RepositorySpec struct {
	CredentialsName string `json:"credentialsName"`
	Owner           string `json:"owner"`

	// WebhookSecretRef represents a secret that should be used for the webhook
	WebhookSecretRef SecretRef `json:"webhookSecretRef"`
}

// RepositoryStatus defines the observed state of Repository
type RepositoryStatus struct {
	ID                       string `json:"id"`
	PoolManagerIsRunning     bool   `json:"poolManagerIsRunning"`
	PoolManagerFailureReason string `json:"poolManagerFailureReason,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=repositories,scope=Namespaced,categories=garm,shortName=repo
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Repository ID"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.poolManagerIsRunning",description="Status of the referenced pool"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.poolManagerFailureReason",description="Error description",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of Repository"

// Repository is the Schema for the repositories API
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

func (r *Repository) GetCredentialsName() string {
	return r.Spec.CredentialsName
}

func (r *Repository) GetID() string {
	return r.Status.ID
}

func (r *Repository) GetName() string {
	return r.ObjectMeta.Name
}

func (r *Repository) GetPoolManagerIsRunning() bool {
	return r.Status.PoolManagerIsRunning
}

func (r *Repository) GetPoolManagerFailureReason() string {
	return r.Status.PoolManagerFailureReason
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
