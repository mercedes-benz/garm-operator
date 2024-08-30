package image

import (
	"context"
	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetByPoolCR(ctx context.Context, client client.Client, pool *garmoperatorv1alpha1.Pool) (*garmoperatorv1alpha1.Image, error) {
	image := &garmoperatorv1alpha1.Image{}
	if pool.Spec.ImageName != "" {
		if err := client.Get(ctx, types.NamespacedName{Name: pool.Spec.ImageName, Namespace: pool.Namespace}, image); err != nil {
			return nil, err
		}
	}
	return image, nil
}
