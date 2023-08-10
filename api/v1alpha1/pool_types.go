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
	commonParams "github.com/cloudbase/garm-provider-common/params"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PoolSpec defines the desired state of Pool
// See: https://github.com/cloudbase/garm/blob/main/params/requests.go#L142
type PoolSpec struct {
	// Defines in which Scope Runners a registered. Has a reference to either an Enterprise, Org or Repo CRD
	GitHubScopeRef         corev1.TypedLocalObjectReference `json:"githubScopeRef"`
	ProviderName           string                           `json:"providerName"`
	MaxRunners             uint                             `json:"maxRunners"`
	MinIdleRunners         uint                             `json:"minIdleRunners"`
	Flavor                 string                           `json:"flavor"`
	OSType                 commonParams.OSType              `json:"osType"`
	OSArch                 commonParams.OSArch              `json:"osArch"`
	Tags                   []string                         `json:"tags"`
	Enabled                bool                             `json:"enabled"`
	RunnerBootstrapTimeout uint                             `json:"runnerBootstrapTimeout"`
	ForceDeleteRunners     bool                             `json:"forceDeleteRunners"`

	// The name of the image resource, this image resource must exists in the same namespace as the pool
	ImageName string `json:"imageName"`

	// +optional
	ExtraSpecs string `json:"extraSpecs"`

	// +optional
	GitHubRunnerGroup string `json:"githubRunnerGroup"`

	// +optional
	RunnerPrefix string `json:"runnerPrefix"`
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
//+kubebuilder:resource:path=pools,scope=Namespaced,categories=garm
//+kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="ImageName",type=string,JSONPath=`.spec.imageName`,priority=1
//+kubebuilder:printcolumn:name="Flavour",type=string,JSONPath=`.spec.flavor`,priority=1
//+kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.providerName`,priority=1
//+kubebuilder:printcolumn:name="Scope",type=string,JSONPath=`.spec.githubScopeRef.kind`,priority=1
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.lastSyncError`,priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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

// +k8s:deepcopy-gen=false
type Predicate func(p Pool) bool

func MatchesImage(image string) Predicate {
	return func(p Pool) bool {
		return p.Spec.ImageName == image
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

func MatchesGitHubScope(name, kind string) Predicate {
	return func(p Pool) bool {
		return p.Spec.GitHubScopeRef.Name == name && p.Spec.GitHubScopeRef.Kind == kind
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
