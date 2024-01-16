// SPDX-License-Identifier: MIT

package annotations

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mercedes-benz/garm-operator/pkg/client/key"
)

// IsPaused returns true if the object has the `paused` annotation.
func IsPaused(o metav1.Object) bool {
	return HasAnnotation(o, key.PausedAnnotation)
}

// HasAnnotation returns true if the object has the specified annotation.
func HasAnnotation(o metav1.Object, annotation string) bool {
	annotations := o.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, ok := annotations[annotation]
	return ok
}

func SetLastSyncTime(o client.Object, client client.Client) error {
	now := time.Now().UTC()
	newAnnotations := appendAnnotations(o, key.LastSyncTimeAnnotation, now.Format(time.RFC3339))
	o.SetAnnotations(newAnnotations)
	return client.Update(context.Background(), o)
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
