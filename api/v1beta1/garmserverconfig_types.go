// SPDX-License-Identifier: MIT

package v1beta1

import (
	"github.com/mercedes-benz/garm-operator/pkg/conditions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GarmServerConfigSpec defines the desired state of GarmServerConfig
type GarmServerConfigSpec struct {
	MetadataURL string `json:"metadataUrl,omitempty"`
	CallbackURL string `json:"callbackUrl,omitempty"`
	WebhookURL  string `json:"webhookUrl,omitempty"`
}

// GarmServerConfigStatus defines the observed state of GarmServerConfig
type GarmServerConfigStatus struct {
	ControllerID         string             `json:"controllerId,omitempty"`
	Hostname             string             `json:"hostname,omitempty"`
	MetadataURL          string             `json:"metadataUrl,omitempty"`
	CallbackURL          string             `json:"callbackUrl,omitempty"`
	WebhookURL           string             `json:"webhookUrl,omitempty"`
	ControllerWebhookURL string             `json:"controllerWebhookUrl,omitempty"`
	MinimumJobAgeBackoff uint               `json:"minimumJobAgeBackoff,omitempty"`
	Version              string             `json:"version,omitempty"`
	Conditions           []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=garmserverconfigs,scope=Namespaced,categories=garm
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.controllerId",description="Controller ID"
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Garm Version"
//+kubebuilder:printcolumn:name="MetadataURL",type="string",JSONPath=".status.metadataUrl",description="MetadataURL",priority=1
//+kubebuilder:printcolumn:name="CallbackURL",type="string",JSONPath=".status.callbackUrl",description="CallbackURL",priority=1
//+kubebuilder:printcolumn:name="WebhookURL",type="string",JSONPath=".status.webhookUrl",description="WebhookURL",priority=1
//+kubebuilder:printcolumn:name="ControllerWebhookURL",type="string",JSONPath=".status.controllerWebhookUrl",description="ControllerWebhookURL",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of GarmServerConfig"

// GarmServerConfig is the Schema for the garmserverconfigs API
type GarmServerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GarmServerConfigSpec   `json:"spec,omitempty"`
	Status GarmServerConfigStatus `json:"status,omitempty"`
}

func (g *GarmServerConfig) InitializeConditions() {
	if conditions.Get(g, conditions.ReadyCondition) == nil {
		conditions.MarkUnknown(g, conditions.ReadyCondition, conditions.UnknownReason, conditions.GarmServerNotReconciledYetMsg)
	}
}

func (g *GarmServerConfig) SetConditions(conditions []metav1.Condition) {
	g.Status.Conditions = conditions
}

func (g *GarmServerConfig) GetConditions() []metav1.Condition {
	return g.Status.Conditions
}

//+kubebuilder:object:root=true

// GarmServerConfigList contains a list of GarmServerConfig
type GarmServerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GarmServerConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GarmServerConfig{}, &GarmServerConfigList{})
}
