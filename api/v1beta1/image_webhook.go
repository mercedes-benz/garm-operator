// SPDX-License-Identifier: MIT

package v1beta1

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var imagelog = logf.Log.WithName("image-resource")

func (i *Image) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(i).
		Complete()
}

//+kubebuilder:webhook:path=/validate-garm-operator-mercedes-benz-com-v1beta1-image,mutating=false,failurePolicy=fail,sideEffects=None,groups=garm-operator.mercedes-benz.com,resources=images,verbs=delete,versions=v1beta1,name=validate.image.garm-operator.mercedes-benz.com,admissionReviewVersions=v1

type ImageValidator struct {
}

var _ webhook.CustomValidator = &ImageValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (i *ImageValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (i *ImageValidator) ValidateUpdate(ctx context.Context, _ runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (i *ImageValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var msg string

	image, ok := obj.(*Image)
	if !ok {
		msg = fmt.Sprintf("expected Image object, got %T", obj)
		return nil, apierrors.NewBadRequest(msg)
	}

	imagelog.Info("validate delete", "name", image.Name, "namespace", image.Namespace)

	pools, err := attachedPools(context.Background(), image.Name)
	if err != nil {
		msg = fmt.Sprintf("imagename=%s with tag=%s can not be deleted, failed to fetch pools: %s", image.Name, image.Spec.Tag, err.Error())
		return nil, apierrors.NewBadRequest(msg)
	}

	if len(pools) > 0 {
		msg = fmt.Sprintf("imagename=%s with tag=%s can not be deleted, as it is still referenced by at least one pool", image.Name, image.Spec.Tag)
		return nil, apierrors.NewBadRequest(msg)
	}
	return nil, nil
}

func attachedPools(ctx context.Context, imageName string) ([]Pool, error) {
	var pools PoolList
	var result []Pool
	if err := c.List(ctx, &pools); err != nil {
		return result, err
	}

	for _, pool := range pools.Items {
		// we do not care about pools that are already deleted
		if pool.GetDeletionTimestamp() == nil {
			if pool.Spec.ImageName == imageName {
				result = append(result, pool)
			}
		}
	}

	return result, nil
}
