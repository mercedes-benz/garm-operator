// SPDX-License-Identifier: MIT

package v1alpha1

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
		Complete()
}

//+kubebuilder:webhook:path=/validate-garm-operator-mercedes-benz-com-v1alpha1-pool,mutating=false,failurePolicy=fail,sideEffects=None,groups=garm-operator.mercedes-benz.com,resources=pools,verbs=create;update,versions=v1alpha1,name=validate.pool.garm-operator.mercedes-benz.com,admissionReviewVersions=v1

var _ webhook.Validator = &Pool{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Pool) ValidateCreate() (admission.Warnings, error) {
	poollog.Info("validate create", "name", r.Name)
	ctx := context.TODO()

	err := validateImageName(ctx, r)
	if err != nil {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
			r.Name,
			field.ErrorList{err},
		)
	}

	if err := r.validateExtraSpec(); err != nil {
		return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
			r.Name,
			field.ErrorList{err},
		)
	}

	listOpts := &client.ListOptions{
		Namespace: r.Namespace,
	}

	poolList := &PoolList{}
	if err := c.List(ctx, poolList, listOpts); err != nil {
		poollog.Error(err, "cannot fetch Pools", "error", err)
		return nil, err
	}

	poolList.FilterByFields(
		MatchesFlavour(r.Spec.Flavor),
		MatchesImage(r.Spec.ImageName),
		MatchesProvider(r.Spec.ProviderName),
		MatchesGitHubScope(r.Spec.GitHubScopeRef.Name, r.Spec.GitHubScopeRef.Kind),
	)

	if len(poolList.Items) > 0 {
		existing := poolList.Items[0]
		return nil, apierrors.NewBadRequest(
			fmt.Sprintf("can not create pool, pool=%s with same image=%s, flavor=%s and provider=%s already exists for specified GitHubScope=%s", existing.Name, existing.Spec.ImageName, existing.Spec.Flavor, existing.Spec.ProviderName, existing.Spec.GitHubScopeRef.Name))
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Pool) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	poollog.Info("validate update", "name", r.Name, "namespace", r.Namespace)

	oldCRD, ok := old.(*Pool)
	if !ok {
		return nil, apierrors.NewBadRequest("failed to convert runtime.Object to Pool CRD")
	}

	// if the object is being deleted, skip validation
	if r.GetDeletionTimestamp() == nil {
		err := validateImageName(context.Background(), r)
		if err != nil {
			return nil, apierrors.NewInvalid(
				schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
				r.Name,
				field.ErrorList{err},
			)
		}

		if err := r.validateExtraSpec(); err != nil {
			return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
				r.Name,
				field.ErrorList{err},
			)
		}

		if err := r.validateProviderName(oldCRD); err != nil {
			return nil, apierrors.NewInvalid(
				schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
				r.Name,
				field.ErrorList{err},
			)
		}

		if err := r.validateGitHubScope(oldCRD); err != nil {
			return nil, apierrors.NewInvalid(
				schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
				r.Name,
				field.ErrorList{err},
			)
		}
	}

	return nil, nil
}

func validateImageName(ctx context.Context, r *Pool) *field.Error {
	image := Image{}
	err := c.Get(ctx, client.ObjectKey{Name: r.Spec.ImageName, Namespace: r.Namespace}, &image)
	if err != nil {
		return field.Invalid(
			field.NewPath("spec").Child("imageName"),
			r.Spec.ImageName,
			err.Error(),
		)
	}
	return nil
}

func (r *Pool) validateProviderName(old *Pool) *field.Error {
	poollog.Info("validate spec.providerName", "spec.providerName", r.Spec.ProviderName)
	fieldPath := field.NewPath("spec").Child("providerName")
	n := r.Spec.ProviderName
	o := old.Spec.ProviderName
	if n != o {
		return field.Invalid(
			fieldPath,
			r.Spec.ProviderName,
			fmt.Errorf("can not change provider of an existing pool. Old name: %s, new name:  %s", o, n).Error(),
		)
	}
	return nil
}

func (r *Pool) validateExtraSpec() *field.Error {
	extraSpecs := json.RawMessage([]byte{})
	fieldPath := field.NewPath("spec").Child("extraSpecs")
	err := json.Unmarshal([]byte(r.Spec.ExtraSpecs), &extraSpecs)
	if err != nil {
		return field.Invalid(
			fieldPath,
			r.Spec.ExtraSpecs,
			fmt.Errorf("can not unmarshal extraSpecs: %s", err.Error()).Error(),
		)
	}

	return nil
}

func (r *Pool) validateGitHubScope(old *Pool) *field.Error {
	poollog.Info("validate spec.githubScopeRef", "spec.githubScopeRef", r.Spec.GitHubScopeRef)
	fieldPath := field.NewPath("spec").Child("githubScopeRef")
	n := r.Spec.GitHubScopeRef
	o := old.Spec.GitHubScopeRef
	if !reflect.DeepEqual(n, o) {
		return field.Invalid(
			fieldPath,
			r.Spec.ProviderName,
			fmt.Errorf("can not change githubScopeRef of an existing pool. Old name: %+v, new name:  %+v", o, n).Error(),
		)
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Pool) ValidateDelete() (admission.Warnings, error) {
	poollog.Info("validate delete", "name", r.Name)
	return nil, nil
}
