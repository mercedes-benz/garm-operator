// SPDX-License-Identifier: MIT

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageSpec defines the desired state of Image
type ImageSpec struct {
	// Tag is the Name of the image in its registry
	// e.g.
	// - in openstack it can be the image name or id
	// - in k8s it can be the docker image name + tag
	Tag string `json:"tag,omitempty"`
}

// ImageStatus defines the observed state of Image
type ImageStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:storageversion
//+kubebuilder:resource:path=images,scope=Namespaced,categories=garm
//+kubebuilder:printcolumn:name="Tag",type=string,JSONPath=`.spec.tag`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Image is the Schema for the images API
type Image struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageSpec   `json:"spec,omitempty"`
	Status ImageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImageList contains a list of Image
type ImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Image `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Image{}, &ImageList{})
}
