// SPDX-License-Identifier: MIT

package v1alpha1

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var runnerlog = logf.Log.WithName("runner-resource")

func (r *Runner) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-garm-operator-mercedes-benz-com-v1alpha1-runner,mutating=false,failurePolicy=fail,sideEffects=None,groups=garm-operator.mercedes-benz.com,resources=runners,verbs=create;update,versions=v1alpha1,name=vrunner.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Runner{}

func (r *Runner) ValidateCreate() (admission.Warnings, error) {
	runnerlog.Info("validate create", "name", r.Name)

	return nil, apierrors.NewForbidden(
		GroupVersion.WithResource("Runner").GroupResource(),
		r.Name,
		fmt.Errorf("creation of runner CR %s forbitten", r.Name),
	)
}

func (r *Runner) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	runnerlog.Info("validate update", "name", r.Name)

	return nil, apierrors.NewForbidden(
		GroupVersion.WithResource("Runner").GroupResource(),
		r.Name,
		fmt.Errorf("update to runner CR %s forbitten", r.Name),
	)
}

func (r *Runner) ValidateDelete() (admission.Warnings, error) {
	runnerlog.Info("validate delete", "name", r.Name)
	return nil, nil
}
