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
	"encoding/json"
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
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Todo: Might replace with reference to Enterprise/Org/Repo CRD
	// Defines in which Scope Runners a registered. Valid options are enterprise, organization, and repository
	GitHubScope GitHubScope `json:"github_scope"`

	// Garm Internal ID of the specified scope as reference
	GitHubScopeID string `json:"github_scope"`

	RunnerPrefix           string          `json:"runner_prefix"`
	ProviderName           string          `json:"provider_name"`
	MaxRunners             uint            `json:"max_runners"`
	MinIdleRunners         uint            `json:"min_idle_runners"`
	Image                  string          `json:"image"`
	Flavor                 string          `json:"flavor"`
	OSType                 params.OSType   `json:"os_type"`
	OSArch                 params.OSArch   `json:"os_arch"`
	Tags                   []string        `json:"tags"`
	Enabled                bool            `json:"enabled"`
	RunnerBootstrapTimeout uint            `json:"runner_bootstrap_timeout"`
	ExtraSpecs             json.RawMessage `json:"extra_specs,omitempty"`
	GitHubRunnerGroup      string          `json:"github-runner-group"`
}

// PoolStatus defines the observed state of Pool
type PoolStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ID string `json:"id"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
