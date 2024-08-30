// SPDX-License-Identifier: MIT

package v1alpha1

import (
	"github.com/mercedes-benz/garm-operator/pkg/filter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImageSpec defines the desired state of Image
type ImageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Tag is the Name of the image in its registry
	// e.g.
	// - in openstack it can be the image name or id
	// - in k8s it can be the docker image name + tag
	Tag string `json:"tag,omitempty"`
}

// ImageStatus defines the observed state of Image
type ImageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=images,scope=Namespaced,categories=garm
//+kubebuilder:printcolumn:name="Tag",type=string,JSONPath=`.spec.tag`,priority=1
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

func MatchesTag(tag string) filter.Predicate[Image] {
	return func(i Image) bool {
		return i.Spec.Tag == tag
	}
}

func init() {
	SchemeBuilder.Register(&Image{}, &ImageList{})
}
