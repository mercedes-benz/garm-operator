// SPDX-License-Identifier: MIT

package v1alpha1

import (
	commonParams "github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm/params"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RunnerSpec defines the desired state of Runner
type RunnerSpec struct{}

// RunnerStatus defines the observed state of Runner
type RunnerStatus struct {
	// ID is the database ID of this instance.
	ID string `json:"id,omitempty"`

	// PeoviderID is the unique ID the provider associated
	// with the compute instance. We use this to identify the
	// instance in the provider.
	ProviderID string `json:"providerId,omitempty"`

	// AgentID is the github runner agent ID.
	AgentID int64 `json:"agentId"`

	// Name is the name associated with an instance. Depending on
	// the provider, this may or may not be useful in the context of
	// the provider, but we can use it internally to identify the
	// instance.
	Name string `json:"name,omitempty"`

	// OSType is the operating system type. For now, only Linux and
	// Windows are supported.
	OSType commonParams.OSType `json:"osType,omitempty"`

	// OSName is the name of the OS. Eg: ubuntu, centos, etc.
	OSName string `json:"osName,omitempty"`

	// OSVersion is the version of the operating system.
	OSVersion string `json:"osVersion,omitempty"`

	// OSArch is the operating system architecture.
	OSArch commonParams.OSArch `json:"osArch,omitempty"`

	// Addresses is a list of IP addresses the provider reports
	// for this instance.
	Addresses []commonParams.Address `json:"addresses,omitempty"`

	// Status is the status of the instance inside the provider (eg: running, stopped, etc)
	Status commonParams.InstanceStatus `json:"status,omitempty"`

	// RunnerStatus is the github runner status as it appears on GitHub.
	InstanceStatus params.RunnerStatus `json:"instanceStatus,omitempty"`

	// PoolID is the ID of the garm pool to which a runner belongs.
	PoolID string `json:"poolId,omitempty"`

	// ProviderFault holds any error messages captured from the IaaS provider that is
	// responsible for managing the lifecycle of the runner.
	ProviderFault string `json:"providerFault,omitempty"`

	// StatusMessages is a list of status messages sent back by the runner as it sets itself
	// up.

	//// UpdatedAt is the timestamp of the last update to this runner.
	// UpdatedAt time.Time `json:"updated_at"`

	// GithubRunnerGroup is the github runner group to which the runner belongs.
	// The runner group must be created by someone with access to the enterprise.
	GitHubRunnerGroup string `json:"githubRunnerGroup"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=runners,scope=Namespaced,categories=garm,shortName=run
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id",description="Runner ID"
//+kubebuilder:printcolumn:name="Pool",type="string",JSONPath=".status.poolId",description="Pool CR Name"
//+kubebuilder:printcolumn:name="Garm Runner Status",type="string",JSONPath=".status.status",description="Garm Runner Status"
//+kubebuilder:printcolumn:name="Provider Runner Status",type="string",JSONPath=".status.instanceStatus",description="Provider Runner Status"
//+kubebuilder:printcolumn:name="Provider ID",type="string",JSONPath=".status.providerId",description="Provider ID",priority=1
//+kubebuilder:printcolumn:name="Agent ID",type="string",JSONPath=".status.agentId",description="Agent ID",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Runner is the Schema for the runners API
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSpec   `json:"spec,omitempty"`
	Status RunnerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RunnerList contains a list of Runner
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Runner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Runner{}, &RunnerList{})
}
