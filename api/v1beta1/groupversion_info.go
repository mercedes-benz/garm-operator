// SPDX-License-Identifier: MIT

// Package v1alpha1 contains API Schema definitions for the garm-operator v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=garm-operator.mercedes-benz.com
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "garm-operator.mercedes-benz.com", Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			scheme.AddKnownTypes(GroupVersion, objectTypes...)
			metav1.AddToGroupVersion(scheme, GroupVersion)
			return nil
		},
	)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme

	// objectTypes is a list of all types that are part of this API group-version
	// This list is used to register the types with the scheme and used to get used by init() to register the types with the scheme
	objectTypes = []runtime.Object{}
)
