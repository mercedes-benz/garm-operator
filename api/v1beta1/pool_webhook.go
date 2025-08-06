// SPDX-License-Identifier: MIT

package v1beta1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var (
	poollog = logf.Log.WithName("pool-resource")
	c       client.Client
)

func (r *Pool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&PoolValidator{}).
		Complete()
}

//+kubebuilder:webhook:path=/validate-garm-operator-mercedes-benz-com-v1beta1-pool,mutating=false,failurePolicy=fail,sideEffects=None,groups=garm-operator.mercedes-benz.com,resources=pools,verbs=create;update,versions=v1beta1,name=validate.pool.garm-operator.mercedes-benz.com,admissionReviewVersions=v1

type PoolValidator struct{}

var _ webhook.CustomValidator = &PoolValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PoolValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	pool, ok := obj.(*Pool)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected Pool object, got %T", obj))
	}

	poollog.Info("validate create request", "name", pool.Name, "namespace", pool.Namespace)

	if err := validateExtraSpec(pool); err != nil {
		return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
			pool.Name,
			field.ErrorList{err},
		)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PoolValidator) ValidateUpdate(_ context.Context, obj runtime.Object, oldObj runtime.Object) (admission.Warnings, error) {
	pool, ok := obj.(*Pool)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected Pool object, got %T", obj))
	}

	poollog.Info("validate update", "name", pool.Name, "namespace", pool.Namespace)

	oldCRD, ok := oldObj.(*Pool)
	if !ok {
		return nil, apierrors.NewBadRequest("failed to convert runtime.Object to Pool CRD")
	}

	// if the object is being deleted, skip validation
	if err := validateExtraSpec(pool); err != nil {
		return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
			pool.Name,
			field.ErrorList{err},
		)
	}

	if err := validateProviderName(pool, oldCRD); err != nil {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
			pool.Name,
			field.ErrorList{err},
		)
	}

	if err := validateGitHubScope(pool, oldCRD); err != nil {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
			pool.Name,
			field.ErrorList{err},
		)
	}

	return nil, nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PoolValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateProviderName(pool, oldPool *Pool) *field.Error {
	poollog.Info("validate spec.providerName", "spec.providerName", pool.Spec.ProviderName)
	fieldPath := field.NewPath("spec").Child("providerName")
	n := pool.Spec.ProviderName
	o := oldPool.Spec.ProviderName
	if n != o {
		return field.Invalid(
			fieldPath,
			pool.Spec.ProviderName,
			fmt.Errorf("can not change provider of an existing pool. Old name: %s, new name: %s", o, n).Error(),
		)
	}
	return nil
}

func validateExtraSpec(pool *Pool) *field.Error {
	extraSpecs := json.RawMessage([]byte{})
	fieldPath := field.NewPath("spec").Child("extraSpecs")
	err := json.Unmarshal([]byte(pool.Spec.ExtraSpecs), &extraSpecs)
	if err != nil {
		return field.Invalid(
			fieldPath,
			pool.Spec.ExtraSpecs,
			fmt.Errorf("can not unmarshal extraSpecs: %s", err.Error()).Error(),
		)
	}

	return nil
}

func validateGitHubScope(pool, oldPool *Pool) *field.Error {
	// poollog.Info("validate spec.githubScopeRef", "spec.githubScopeRef", pool.Spec.GitHubScopeRef)
	fieldPath := field.NewPath("spec").Child("githubScopeRef")
	n := pool.Spec.GitHubScopeRef
	o := oldPool.Spec.GitHubScopeRef
	if !reflect.DeepEqual(n.Name, o.Name) {
		return field.Invalid(
			fieldPath,
			pool.Spec.GitHubScopeRef.Name,
			fmt.Errorf("can not change githubScopeRef of an existing pool. Old name: %+v, new name: %+v", o.Name, n.Name).Error(),
		)
	}

	if !reflect.DeepEqual(n.Kind, o.Kind) {
		return field.Invalid(
			fieldPath,
			pool.Spec.GitHubScopeRef.Kind,
			fmt.Errorf("can not change githubScopeRef of an existing pool. Old Kind: %+v, new Kind: %+v", o.Kind, n.Kind).Error(),
		)
	}

	return nil
}
