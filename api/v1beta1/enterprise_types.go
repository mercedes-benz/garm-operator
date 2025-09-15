// SPDX-License-Identifier: MIT

package v1beta1

import (
	"github.com/cloudbase/garm/params"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mercedes-benz/garm-operator/pkg/conditions"
)

// EnterpriseSpec defines the desired state of Enterprise
type EnterpriseSpec struct {
	CredentialsRef corev1.TypedLocalObjectReference `json:"credentialsRef"`

	// WebhookSecretRef represents a secret that should be used for the webhook
	WebhookSecretRef SecretRef               `json:"webhookSecretRef"`
	PoolBalancerType params.PoolBalancerType `json:"poolBalancerType,omitempty"`
}

// EnterpriseStatus defines the observed state of Enterprise
type EnterpriseStatus struct {
	ID         string             `json:"id"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=enterprises,scope=Namespaced,categories=garm,shortName=ent
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Enterprise ID"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].message",priority=1
//+kubebuilder:printcolumn:name="Pool_Manager_Failure",type="string",JSONPath=`.status.conditions[?(@.reason=='PoolManagerFailure')].message`,priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of Enterprise"

// Enterprise is the Schema for the enterprises API
type Enterprise struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnterpriseSpec   `json:"spec,omitempty"`
	Status EnterpriseStatus `json:"status,omitempty"`
}

func (e *Enterprise) InitializeConditions() {
	if conditions.Get(e, conditions.ReadyCondition) == nil {
		conditions.MarkUnknown(e, conditions.ReadyCondition, conditions.UnknownReason, conditions.GarmServerNotReconciledYetMsg)
	}

	if conditions.Get(e, conditions.PoolManager) == nil {
		conditions.MarkUnknown(e, conditions.PoolManager, conditions.UnknownReason, conditions.GarmServerNotReconciledYetMsg)
	}

	if conditions.Get(e, conditions.WebhookSecretReference) == nil {
		conditions.MarkUnknown(e, conditions.WebhookSecretReference, conditions.UnknownReason, conditions.WebhookSecretNotReconciledYetMsg)
	}

	if conditions.Get(e, conditions.GithubCredentialsReference) == nil {
		conditions.MarkUnknown(e, conditions.GithubCredentialsReference, conditions.UnknownReason, conditions.CredentialsNotReconciledYetMsg)
	}
}

func (e *Enterprise) SetConditions(conditions []metav1.Condition) {
	e.Status.Conditions = conditions
}

func (e *Enterprise) GetConditions() []metav1.Condition {
	return e.Status.Conditions
}

func (e *Enterprise) GetCredentialsName() string {
	return e.Spec.CredentialsRef.Name
}

func (e *Enterprise) GetID() string {
	return e.Status.ID
}

func (e *Enterprise) GetName() string {
	return e.ObjectMeta.Name
}

func (e *Enterprise) GetPoolManagerIsRunning() bool {
	condition := conditions.Get(e, conditions.PoolManager)
	if condition == nil {
		return false
	}

	return condition.Status == TrueAsString
}

func (e *Enterprise) GetPoolManagerFailureReason() string {
	condition := conditions.Get(e, conditions.PoolManager)
	if condition == nil {
		return ""
	}

	if condition.Reason == string(conditions.PoolManagerFailureReason) {
		return condition.Message
	}

	return ""
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
