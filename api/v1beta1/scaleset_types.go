// SPDX-License-Identifier: MIT

package v1beta1

import (
	"fmt"

	commonParams "github.com/cloudbase/garm-provider-common/params"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mercedes-benz/garm-operator/pkg/conditions"
)

// ScaleSetDeletionPolicy controls what happens to the remote scale set when
// the CR is deleted. Retain additionally forbids the operator from deleting
// and recreating the remote scale set to apply a change to an immutable field.
type ScaleSetDeletionPolicy string

// scaleSetScopeAPIGroup is the only API group a githubScopeRef may point to.
const scaleSetScopeAPIGroup = "garm-operator.mercedes-benz.com"

const (
	ScaleSetDeletionPolicyDelete ScaleSetDeletionPolicy = "Delete"
	ScaleSetDeletionPolicyRetain ScaleSetDeletionPolicy = "Retain"
)

// +kubebuilder:validation:XValidation:rule="self.minIdleRunners <= self.maxRunners",message="minIdleRunners must be less than or equal to maxRunners"
type ScaleSetSpec struct {
	// Scope runners register in. References an Enterprise, Organization or Repository CR.
	// +kubebuilder:validation:XValidation:rule="self.kind in ['Enterprise', 'Organization', 'Repository']",message="kind must be Enterprise, Organization or Repository"
	// +kubebuilder:validation:XValidation:rule="!has(self.apiGroup) || self.apiGroup == 'garm-operator.mercedes-benz.com'",message="apiGroup must be garm-operator.mercedes-benz.com"
	GitHubScopeRef corev1.TypedLocalObjectReference `json:"githubScopeRef"`
	// ProviderName must match a provider configured on the GARM server.
	// Immutable server-side routing: changing it requires recreation.
	ProviderName string `json:"providerName"`
	// Image is the provider image reference (gallery ID, Glance name, runner image).
	Image  string              `json:"image"`
	Flavor string              `json:"flavor"`
	OSType commonParams.OSType `json:"osType"`
	OSArch commonParams.OSArch `json:"osArch"`

	// +kubebuilder:validation:Minimum=1
	MaxRunners uint `json:"maxRunners"`
	// +kubebuilder:default=0
	MinIdleRunners uint `json:"minIdleRunners"`
	// Minutes after which a non-joining runner is considered failed.
	// +kubebuilder:default=20
	// +kubebuilder:validation:Minimum=1
	RunnerBootstrapTimeout uint `json:"runnerBootstrapTimeout"`

	// Labels are immutable server-side: a change requires recreation.
	Labels []string `json:"labels,omitempty"`

	// ExtraSpecs is a provider specific JSON document. When empty, any extra
	// specs set on the remote scale set are cleared.
	// +optional
	ExtraSpecs string `json:"extraSpecs,omitempty"`
	// GitHubRunnerGroup places runners in the given runner group. When empty
	// the remote value is left untouched: GARM cannot clear a runner group
	// once it has been set.
	// +optional
	GitHubRunnerGroup string `json:"githubRunnerGroup,omitempty"`
	// +optional
	RunnerPrefix string `json:"runnerPrefix,omitempty"`

	Enabled bool `json:"enabled"`
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	DeletionPolicy ScaleSetDeletionPolicy `json:"deletionPolicy,omitempty"`
}

// Validate rejects specs the reconciler could never converge on. These are
// permanent errors: callers must not retry until the spec changes. The CRD
// carries the same rules as CEL; this guards objects that bypassed admission.
func (s ScaleSetSpec) Validate() error {
	if s.ProviderName == "" || s.Image == "" || s.Flavor == "" {
		return fmt.Errorf("providerName, image and flavor are required")
	}
	if s.MaxRunners == 0 {
		return fmt.Errorf("maxRunners cannot be 0")
	}
	if s.MinIdleRunners > s.MaxRunners {
		return fmt.Errorf("minIdleRunners must be less than or equal to maxRunners")
	}
	if s.RunnerBootstrapTimeout == 0 {
		return fmt.Errorf("runnerBootstrapTimeout cannot be 0")
	}
	switch s.GitHubScopeRef.Kind {
	case string(EnterpriseScope), string(OrganizationScope), string(RepositoryScope):
	default:
		return fmt.Errorf("githubScopeRef.kind must be Enterprise, Organization or Repository, got %q", s.GitHubScopeRef.Kind)
	}
	if g := s.GitHubScopeRef.APIGroup; g != nil && *g != scaleSetScopeAPIGroup {
		return fmt.Errorf("githubScopeRef.apiGroup must be %s, got %q", scaleSetScopeAPIGroup, *g)
	}
	return nil
}

// ScaleSetStatus defines the observed state of ScaleSet.
type ScaleSetStatus struct {
	// ID is the stable remote GARM scale-set ID. Adoption keys off this ID,
	// never the CR name.
	ID string `json:"id,omitempty"`
	// Enabled mirrors the live remote enabled state, not the desired spec.
	Enabled bool `json:"enabled,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (s *ScaleSet) InitializeConditions() {
	if conditions.Get(s, conditions.ReadyCondition) == nil {
		conditions.MarkUnknown(s, conditions.ReadyCondition, conditions.UnknownReason, conditions.GarmServerNotReconciledYetMsg)
	}
}

func (s *ScaleSet) SetConditions(c []metav1.Condition) {
	s.Status.Conditions = c
}

func (s *ScaleSet) GetConditions() []metav1.Condition {
	return s.Status.Conditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:resource:path=scalesets,scope=Namespaced,categories=garm
//+kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="MinIdleRunners",type=string,JSONPath=`.spec.minIdleRunners`
//+kubebuilder:printcolumn:name="MaxRunners",type=string,JSONPath=`.spec.maxRunners`
//+kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.providerName`,priority=1
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="Enabled",type=boolean,JSONPath=`.spec.enabled`,priority=1
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// ScaleSet is the Schema for the scalesets API.
type ScaleSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScaleSetSpec   `json:"spec,omitempty"`
	Status ScaleSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ScaleSetList contains a list of ScaleSet.
type ScaleSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScaleSet `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &ScaleSet{}, &ScaleSetList{})
}
