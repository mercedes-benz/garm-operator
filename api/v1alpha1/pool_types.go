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
	"github.com/cloudbase/garm/params"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:Enum=Enterprise;Organization;Repository
type GitHubScope string

const (
	EnterpriseScope   GitHubScope = "Enterprise"
	OrganizationScope GitHubScope = "Organization"
	RepositoryScope   GitHubScope = "Repository"
)

// PoolSpec defines the desired state of Pool
// See: https://github.com/cloudbase/garm/blob/main/params/requests.go#L142
type PoolSpec struct {
	// Defines in which Scope Runners a registered. Valid options are enterprise, organization, and repository
	GitHubScope GitHubScope `json:"github_scope"`

	// Garm Internal ID of the specified scope as reference
	GitHubScopeID string `json:"github_scope_id"`

	ProviderName           string        `json:"provider_name"`
	MaxRunners             uint          `json:"max_runners"`
	MinIdleRunners         uint          `json:"min_idle_runners"`
	Image                  string        `json:"image"`
	Flavor                 string        `json:"flavor"`
	OSType                 params.OSType `json:"os_type"`
	OSArch                 params.OSArch `json:"os_arch"`
	Tags                   []string      `json:"tags"`
	Enabled                bool          `json:"enabled"`
	RunnerBootstrapTimeout uint          `json:"runner_bootstrap_timeout"`
	ForceDeleteRunners     bool          `json:"force_delete_runners"`

	// +optional
	ExtraSpecs string `json:"extra_specs"`

	// +optional
	GitHubRunnerGroup string `json:"github_runner_group"`

	// +optional
	RunnerPrefix string `json:"runner_prefix"`
}

// PoolStatus defines the observed state of Pool
type PoolStatus struct {
	ID            string      `json:"id"`
	Synced        bool        `json:"synced"`
	LastSyncTime  metav1.Time `json:"lastSyncTime"`
	LastSyncError string      `json:"lastSyncError,omitempty"`
	RunnerCount   int         `json:"runnerCount"`
	ActiveRunners int         `json:"activeRunners"`
	IdleRunners   int         `json:"idleRunners"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Flavour",type=string,JSONPath=`.spec.flavor`
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider_name`
// +kubebuilder:printcolumn:name="Scope",type=string,JSONPath=`.spec.github_scope`

// Pool is the Schema for the pools API
type Pool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PoolSpec   `json:"spec,omitempty"`
	Status PoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PoolList contains a list of Pool
type PoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pool{}, &PoolList{})
}

type Predicate func(p Pool) bool

func MatchesImage(image string) Predicate {
	return func(p Pool) bool {
		return p.Spec.Image == image
	}
}

func MatchesFlavour(flavour string) Predicate {
	return func(p Pool) bool {
		return p.Spec.Flavor == flavour
	}
}

func MatchesProvider(provider string) Predicate {
	return func(p Pool) bool {
		return p.Spec.ProviderName == provider
	}
}

func MatchesGitHubScope(scope GitHubScope, id string) Predicate {
	return func(p Pool) bool {
		return p.Spec.GitHubScope == scope && p.Spec.GitHubScopeID == id
	}
}

func (p *PoolList) FilterByFields(predicates ...Predicate) {
	var filteredItems []Pool

	for _, pool := range p.Items {
		match := true
		for _, predicate := range predicates {
			if !predicate(pool) {
				match = false
				break
			}
		}
		if match {
			filteredItems = append(filteredItems, pool)
		}
	}

	p.Items = filteredItems
}

// Todo: Might replace GitHubScope with reference to Enterprise/Org/Repo CRD
