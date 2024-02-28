// SPDX-License-Identifier: MIT

package secret

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
)

// FetchRef fetches a secret for a given garmoperatorv1alpha1.SecretRef and namespace
func FetchRef(ctx context.Context, c client.Client, ref *garmoperatorv1alpha1.SecretRef, namespace string) (string, error) {
	if ref == nil {
		return "", nil
	}

	secret := &corev1.Secret{}
	err := c.Get(
		ctx,
		client.ObjectKey{
			Name:      ref.Name,
			Namespace: namespace,
		},
		secret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", err
		}
		return "", fmt.Errorf("error fetching secret %s/%s: %v", namespace, ref.Name, err)
	}

	tokenBytes, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %q in secret %s/%s not found", ref.Key, namespace, ref.Name)
	}

	return string(tokenBytes), nil
}
