// SPDX-License-Identifier: MIT

package v1beta1

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var repositorylog = logf.Log.WithName("repository-resource")

func (r *Repository) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-garm-operator-mercedes-benz-com-v1alpha1-repository,mutating=false,failurePolicy=fail,sideEffects=None,groups=garm-operator.mercedes-benz.com,resources=repositories,verbs=create;update,versions=v1alpha1,name=validate.repository.garm-operator.mercedes-benz.com,admissionReviewVersions=v1

var _ webhook.Validator = &Repository{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Repository) ValidateCreate() (admission.Warnings, error) {
	repositorylog.Info("validate create", "name", r.Name, "namespace", r.Namespace)
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Repository) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	repositorylog.Info("validate update", "name", r.Name, "namespace", r.Namespace)

	oldCRD, ok := old.(*Repository)
	if !ok {
		return nil, apierrors.NewBadRequest("failed to convert runtime.Object to Repository CRD")
	}

	if err := r.validateRepoOwnerName(oldCRD); err != nil {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: "Repository"},
			r.Name,
			field.ErrorList{err},
		)
	}
	return nil, nil
}

func (r *Repository) validateRepoOwnerName(old *Repository) *field.Error {
	repositorylog.Info("validate spec.owner", "spec.owner", r.Spec.Owner)
	fieldPath := field.NewPath("spec").Child("owner")
	n := r.Spec.Owner
	o := old.Spec.Owner
	if n != o {
		return field.Invalid(
			fieldPath,
			r.Spec.Owner,
			fmt.Errorf("cannot change owner of repository resource. Old name: %s, new name:  %s", o, n).Error(),
		)
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Repository) ValidateDelete() (admission.Warnings, error) {
	repositorylog.Info("validate delete", "name", r.Name, "namespace", r.Namespace)
	return nil, nil
}
