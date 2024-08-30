// SPDX-License-Identifier: MIT

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mercedes-benz/garm-operator/pkg/image"
	poolUtil "github.com/mercedes-benz/garm-operator/pkg/pools"
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

func (p *Pool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(p).
		Complete()
}

//+kubebuilder:webhook:path=/validate-garm-operator-mercedes-benz-com-v1alpha1-pool,mutating=false,failurePolicy=fail,sideEffects=None,groups=garm-operator.mercedes-benz.com,resources=pools,verbs=create;update,versions=v1alpha1,name=validate.pool.garm-operator.mercedes-benz.com,admissionReviewVersions=v1

var _ webhook.Validator = &Pool{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (p *Pool) ValidateCreate() (admission.Warnings, error) {
	poollog.Info("validate create", "name", p.Name)
	ctx := context.TODO()

	if err := p.validateExtraSpec(); err != nil {
		return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
			p.Name,
			field.ErrorList{err},
		)
	}

	poolImage, err := image.GetByPoolCR(ctx, c, p)
	if err != nil {
		poollog.Error(err, "cannot fetch Image", "error", err)
		return nil, nil
	}

	duplicate, duplicateName, err := poolUtil.CheckDuplicate(ctx, c, p, poolImage)
	if err != nil {
		poollog.Error(err, "error checking for duplicate", "error", err)
		return nil, nil
	}

	if duplicate {
		err := fmt.Sprintf("pool with same image, flavor, provider and github scope already exists: %s", duplicateName)
		return nil, apierrors.NewBadRequest(err)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (p *Pool) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	poollog.Info("validate update", "name", p.Name, "namespace", p.Namespace)

	oldCRD, ok := old.(*Pool)
	if !ok {
		return nil, apierrors.NewBadRequest("failed to convert runtime.Object to Pool CRD")
	}

	// if the object is being deleted, skip validation
	if p.GetDeletionTimestamp() == nil {
		if err := p.validateExtraSpec(); err != nil {
			return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
				p.Name,
				field.ErrorList{err},
			)
		}

		if err := p.validateProviderName(oldCRD); err != nil {
			return nil, apierrors.NewInvalid(
				schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
				p.Name,
				field.ErrorList{err},
			)
		}

		if err := p.validateGitHubScope(oldCRD); err != nil {
			return nil, apierrors.NewInvalid(
				schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
				p.Name,
				field.ErrorList{err},
			)
		}
	}

	return nil, nil
}

func (p *Pool) validateProviderName(old *Pool) *field.Error {
	poollog.Info("validate spec.providerName", "spec.providerName", p.Spec.ProviderName)
	fieldPath := field.NewPath("spec").Child("providerName")
	n := p.Spec.ProviderName
	o := old.Spec.ProviderName
	if n != o {
		return field.Invalid(
			fieldPath,
			p.Spec.ProviderName,
			fmt.Errorf("can not change provider of an existing pool. Old name: %s, new name:  %s", o, n).Error(),
		)
	}
	return nil
}

func (p *Pool) validateExtraSpec() *field.Error {
	extraSpecs := json.RawMessage([]byte{})
	fieldPath := field.NewPath("spec").Child("extraSpecs")
	err := json.Unmarshal([]byte(p.Spec.ExtraSpecs), &extraSpecs)
	if err != nil {
		return field.Invalid(
			fieldPath,
			p.Spec.ExtraSpecs,
			fmt.Errorf("can not unmarshal extraSpecs: %s", err.Error()).Error(),
		)
	}

	return nil
}

func (p *Pool) validateGitHubScope(old *Pool) *field.Error {
	poollog.Info("validate spec.githubScopeRef", "spec.githubScopeRef", p.Spec.GitHubScopeRef)
	fieldPath := field.NewPath("spec").Child("githubScopeRef")
	n := p.Spec.GitHubScopeRef
	o := old.Spec.GitHubScopeRef
	if !reflect.DeepEqual(n, o) {
		return field.Invalid(
			fieldPath,
			p.Spec.ProviderName,
			fmt.Errorf("can not change githubScopeRef of an existing pool. Old name: %+v, new name:  %+v", o, n).Error(),
		)
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (p *Pool) ValidateDelete() (admission.Warnings, error) {
	poollog.Info("validate delete", "name", p.Name)
	return nil, nil
}
