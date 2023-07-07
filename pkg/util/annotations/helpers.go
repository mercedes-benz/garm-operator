// SPDX-License-Identifier: MIT

package annotations

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mercedes-benz/garm-operator/pkg/client/key"
)

// IsPaused returns true if the object has the `paused` annotation.
func IsPaused(o metav1.Object) bool {
	return hasAnnotation(o, key.PausedAnnotation)
}

// hasAnnotation returns true if the object has the specified annotation.
func hasAnnotation(o metav1.Object, annotation string) bool {
	annotations := o.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, ok := annotations[annotation]
	return ok
}
