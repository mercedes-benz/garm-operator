/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"errors"
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
var poollog = logf.Log.WithName("pool-resource")

func (r *Pool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-garm-operator-mercedes-benz-com-v1alpha1-pool,mutating=true,failurePolicy=fail,sideEffects=None,groups=garm-operator.mercedes-benz.com,resources=pools,verbs=create;update,versions=v1alpha1,name=mpool.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Pool{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Pool) Default() {
	poollog.Info("default", "name", r.Name)
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-garm-operator-mercedes-benz-com-v1alpha1-pool,mutating=false,failurePolicy=fail,sideEffects=None,groups=garm-operator.mercedes-benz.com,resources=pools,verbs=create;update,versions=v1alpha1,name=vpool.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Pool{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Pool) ValidateCreate() (admission.Warnings, error) {
	poollog.Info("validate create", "name", r.Name)
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Pool) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	var errList field.ErrorList
	poollog.Info("validate update", "name", r.Name)

	oldCRD, ok := old.(*Pool)
	if !ok {
		return nil, errors.New("failed to convert runtime.Object to Pool CRD")
	}

	errList = append(errList, r.validateProviderName(oldCRD))
	if len(errList) == 0 {
		return nil, nil
	}

	err := apierrors.NewInvalid(
		schema.GroupKind{Group: "garm-operator.mercedes-benz.com", Kind: "Pool"},
		r.Name,
		errList,
	)

	return nil, err
}

func (r *Pool) validateProviderName(old *Pool) *field.Error {
	fieldPath := field.NewPath("spec").Child("provider_name")
	n := r.Spec.ProviderName
	o := old.Spec.ProviderName
	if n != o {
		return field.Invalid(
			fieldPath,
			r.Spec.ProviderName,
			fmt.Errorf("provider names do not match. Old name: %s, new name:  %s", o, n).Error(),
		)
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Pool) ValidateDelete() (admission.Warnings, error) {
	poollog.Info("validate delete", "name", r.Name)
	return nil, nil
}
