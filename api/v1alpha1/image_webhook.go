// SPDX-License-Identifier: MIT

package v1alpha1

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

//+kubebuilder:webhook:path=/validate-garm-operator-mercedes-benz-com-v1alpha1-image,mutating=false,failurePolicy=fail,sideEffects=None,groups=garm-operator.mercedes-benz.com,resources=images,verbs=create;update;delete,versions=v1alpha1,name=validate.image.garm-operator.mercedes-benz.com,admissionReviewVersions=v1

var _ webhook.Validator = &Image{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (i *Image) ValidateCreate() (admission.Warnings, error) {
	imagelog.Info("validate create", "name", i.Name)
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (i *Image) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	imagelog.Info("validate update", "name", i.Name)
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (i *Image) ValidateDelete() (admission.Warnings, error) {
	imagelog.Info("validate delete", "name", i.Name)
	var msg string

	pools, err := i.attachedPools(context.Background())
	if err != nil {
		msg = fmt.Sprintf("imagename=%s with tag=%s can not be deleted, failed to fetch pools: %s", i.Name, i.Spec.Tag, err.Error())
		return nil, apierrors.NewBadRequest(msg)
	}

	if len(pools) > 0 {
		msg = fmt.Sprintf("imagename=%s with tag=%s can not be deleted, as it is still referenced by at least one pool", i.Name, i.Spec.Tag)
		return nil, apierrors.NewBadRequest(msg)
	}
	return nil, nil
}

func (i *Image) attachedPools(ctx context.Context) ([]Pool, error) {
	var pools PoolList
	var result []Pool
	if err := c.List(ctx, &pools); err != nil {
		return result, err
	}

	for _, pool := range pools.Items {
		// we do not care about pools that are already deleted
		if pool.GetDeletionTimestamp() == nil {
			if pool.Spec.ImageName == i.Name {
				result = append(result, pool)
			}
		}
	}

	return result, nil
}
