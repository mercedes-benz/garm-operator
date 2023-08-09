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

type FetchSecretCallback func() (string, error)

// TryGet takes in a default secret value and function to fetch a secret. If the default secret value is empty, it trys to fetch the secret via the given fetching function
func TryGet(defaultSecretValue string, fetchSecretCallback FetchSecretCallback) (string, error) {
	var err error
	webhookSecret := defaultSecretValue

	if webhookSecret == "" {
		webhookSecret, err = fetchSecretCallback()
		if err != nil {
			return "", fmt.Errorf("can not fetch webhook secret: %s", err.Error())
		}
	}
	return webhookSecret, err
}

// ReadFromFile Returns a function which can fetch a secret from a given file path
func ReadFromFile(filePath string) FetchSecretCallback {
	return func() (string, error) {
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
}

// FetchRef returns a function which can fetch a secret for a given garmoperatorv1alpha1.SecretRef and namespace
func FetchRef(ctx context.Context, c client.Client, ref *garmoperatorv1alpha1.SecretRef, namespace string) FetchSecretCallback {
	return func() (string, error) {
		if ref == nil {
			return "", nil
		}

		secret := &corev1.Secret{}
		err := c.Get(
			ctx,
			client.ObjectKey{
				Name:      ref.SecretName,
				Namespace: namespace,
			},
			secret)
		if err != nil {
			return "", fmt.Errorf("error fetching secret %s/%s: %v", namespace, ref.SecretName, err)
		}

		tokenBytes, ok := secret.Data[ref.Key]
		if !ok {
			return "", fmt.Errorf("key %q in secret %s/%s not found", ref.Key, namespace, ref.SecretName)
		}

		return string(tokenBytes), nil
	}
}
