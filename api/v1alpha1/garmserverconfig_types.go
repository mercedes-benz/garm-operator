// SPDX-License-Identifier: MIT

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GarmServerConfigSpec defines the desired state of GarmServerConfig
type GarmServerConfigSpec struct {
	MetadataURL string `json:"metadataURL,omitempty"`
	CallbackURL string `json:"callbackURL,omitempty"`
	WebhookURL  string `json:"webhookURL,omitempty"`
}

// GarmServerConfigStatus defines the observed state of GarmServerConfig
type GarmServerConfigStatus struct {
	ControllerID         string             `json:"controllerID"`
	Hostname             string             `json:"hostname"`
	MetadataURL          string             `json:"metadataURL"`
	CallbackURL          string             `json:"callbackURL"`
	WebhookURL           string             `json:"webhookURL"`
	ControllerWebhookURL string             `json:"controllerWebhookURL"`
	MinimumJobAgeBackoff uint               `json:"minimumJobAgeBackoff"`
	Version              string             `json:"version"`
	Conditions           []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=garmserverconfigs,scope=Namespaced,categories=garm,shortName=server
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.controllerID",description="Controller ID"
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Garm Version"
//+kubebuilder:printcolumn:name="MetadataURL",type="string",JSONPath=".status.metadataURL",description="MetadataURL",priority=1
//+kubebuilder:printcolumn:name="CallbackURL",type="string",JSONPath=".status.callbackURL",description="CallbackURL",priority=1
//+kubebuilder:printcolumn:name="WebhookURL",type="string",JSONPath=".status.webhookURL",description="WebhookURL",priority=1
//+kubebuilder:printcolumn:name="ControllerWebhookURL",type="string",JSONPath=".status.controllerWebhookURL",description="ControllerWebhookURL",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of GarmServerConfig"

// GarmServerConfig is the Schema for the garmserverconfigs API
type GarmServerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GarmServerConfigSpec   `json:"spec,omitempty"`
	Status GarmServerConfigStatus `json:"status,omitempty"`
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
