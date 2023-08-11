package secret

import (
	"context"
	"fmt"
	garmoperatorv1alpha1 "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// ReadFromFile fetches a secret value from a given file path
func ReadFromFile(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	result := string(bytes)
	result = strings.TrimSpace(result)
	result = strings.Replace(result, "\t", "", -1)
	result = strings.Replace(result, "\n", "", -1)

	return result, err
}

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
		return "", fmt.Errorf("error fetching secret %s/%s: %v", namespace, ref.Name, err)
	}

	tokenBytes, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %q in secret %s/%s not found", ref.Key, namespace, ref.Name)
	}

	return string(tokenBytes), nil
}
