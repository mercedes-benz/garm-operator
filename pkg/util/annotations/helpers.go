// SPDX-License-Identifier: MIT

package annotations

import (
	"time"

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

func SetLastSyncTime(o metav1.Object) {
	now := time.Now().UTC()
	newAnnotations := appendAnnotations(o, key.LastSyncTimeAnnotation, now.Format(time.RFC3339))
	o.SetAnnotations(newAnnotations)
}

func appendAnnotations(o metav1.Object, kayValuePair ...string) map[string]string {
	newAnnotations := map[string]string{}
	for k, v := range o.GetAnnotations() {
		newAnnotations[k] = v
	}
	for i := 0; i < len(kayValuePair)-1; i += 2 {
		k := kayValuePair[i]
		v := kayValuePair[i+1]
		newAnnotations[k] = v
	}
	return newAnnotations
}
