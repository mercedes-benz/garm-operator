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
	"context"
	"fmt"

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
var poollog = logf.Log.WithName("pool-resource")
var c client.Client

func (r *Pool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
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
	ctx := context.TODO()
	pool := r

	image, err := validateImageName(ctx, pool)
	if err != nil {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
			pool.Name,
			field.ErrorList{err},
		)
	}

	listOpts := &client.ListOptions{
		Namespace: pool.Namespace,
	}

	poolList := &PoolList{}
	if err := c.List(ctx, poolList, listOpts); err != nil {
		poollog.Error(err, "cannot fetch Pools", "error", err)
		return nil, err
	}

	poolList.FilterByFields(
		MatchesFlavour(pool.Spec.Flavor),
		MatchesImage(image.Spec.Tag),
		MatchesProvider(pool.Spec.ProviderName),
		MatchesGitHubScope(pool.Spec.GitHubScopeRef.Name, pool.Spec.GitHubScopeRef.Kind),
	)

	if len(poolList.Items) > 0 {
		existing := poolList.Items[0]
		return nil, apierrors.NewBadRequest(
			fmt.Sprintf("can not create pool, pool=%s with same image=%s , flavor=%s  and provider=%s already exists for specified GitHubScope=%s", existing.Name, existing.Spec.ImageName, existing.Spec.Flavor, existing.Spec.ProviderName, existing.Spec.GitHubScopeRef.Name))
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Pool) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	poollog.Info("validate update", "name", r)

	oldCRD, ok := old.(*Pool)
	if !ok {
		return nil, apierrors.NewBadRequest("failed to convert runtime.Object to Pool CRD")
	}

	_, err := validateImageName(context.Background(), r)
	if err != nil {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: "Pool"},
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

	return nil, nil
}

func validateImageName(ctx context.Context, r *Pool) (*Image, *field.Error) {
	image := Image{}
	err := c.Get(ctx, client.ObjectKey{Name: r.Spec.ImageName, Namespace: r.Namespace}, &image)
	if err != nil {
		return nil, field.Invalid(
			field.NewPath("spec").Child("imageName"),
			r.Spec.ImageName,
			err.Error(),
		)
	}
	return &image, nil
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
