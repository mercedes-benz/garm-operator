// SPDX-License-Identifier: MIT

package finalizers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func EnsureFinalizer(ctx context.Context, c client.Client, o client.Object, finalizer string) (finalizerAdded bool, err error) {
	// Finalizers can only be added when the deletionTimestamp is not set.
	if !o.GetDeletionTimestamp().IsZero() {
		return false, nil
	}

	if controllerutil.ContainsFinalizer(o, finalizer) {
		return false, nil
	}

	controllerutil.AddFinalizer(o, finalizer)

	if err := c.Update(ctx, o); err != nil {
		return false, err
	}

	return true, nil
}
