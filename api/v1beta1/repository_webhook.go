// SPDX-License-Identifier: MIT

package v1beta1

import (
	"context"
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
		WithValidator(&RepositoryValidator{}).
		Complete()
}

//+kubebuilder:webhook:path=/validate-garm-operator-mercedes-benz-com-v1beta1-repository,mutating=false,failurePolicy=fail,sideEffects=None,groups=garm-operator.mercedes-benz.com,resources=repositories,verbs=update,versions=v1beta1,name=validate.repository.garm-operator.mercedes-benz.com,admissionReviewVersions=v1

type RepositoryValidator struct{}

var _ webhook.CustomValidator = &RepositoryValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *RepositoryValidator) ValidateCreate(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *RepositoryValidator) ValidateUpdate(_ context.Context, obj runtime.Object, oldObj runtime.Object) (admission.Warnings, error) {
	repo, ok := obj.(*Repository)
	if !ok {
		return nil, apierrors.NewBadRequest("failed to convert runtime.Object to Repository CRD")
	}

	repositorylog.Info("validate update", "name", repo.Name, "namespace", repo.Namespace)

	oldCRD, ok := oldObj.(*Repository)
	if !ok {
		return nil, apierrors.NewBadRequest("failed to convert runtime.Object to Repository CRD")
	}

	if err := validateRepoOwnerName(repo, oldCRD); err != nil {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: "Repository"},
			repo.Name,
			field.ErrorList{err},
		)
	}
	return nil, nil
}

func validateRepoOwnerName(repo, oldRepo *Repository) *field.Error {
	repositorylog.Info("validate spec.owner", "spec.owner", repo.Spec.Owner)
	fieldPath := field.NewPath("spec").Child("owner")
	n := repo.Spec.Owner
	o := oldRepo.Spec.Owner
	if n != o {
		return field.Invalid(
			fieldPath,
			repo.Spec.Owner,
			fmt.Errorf("can not change owner of an existing repository. Old name: %s, new name: %s", o, n).Error(),
		)
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *RepositoryValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
