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

package controller

import (
	"context"
	"fmt"

	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/params"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/v1alpha1"

	garmClient "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/key"
)

// OrganizationReconciler reconciles a Organization object
type OrganizationReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	BaseURL  string
	Username string
	Password string
}

//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=organizations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=organizations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=organizations/finalizers,verbs=update

func (r *OrganizationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	organization := &garmoperatorv1alpha1.Organization{}
	err := r.Get(ctx, req.NamespacedName, organization)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("object was not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	scope, err := garmClient.NewOrganizationClient(garmClient.GarmScopeParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
		// Debug:    true,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// Handle deleted organizations
	if !organization.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, scope, organization)
	}

	return r.reconcileNormal(ctx, scope, organization)
}

// func (r *OrganizationReconciler) reconcileNormal(ctx context.Context, scope garmClient.OrganizationScope) (ctrl.Result, error) {
func (r *OrganizationReconciler) reconcileNormal(ctx context.Context, scope garmClient.OrganizationClient, organization *garmoperatorv1alpha1.Organization) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// If the Organization doesn't has our finalizer, add it.
	if !controllerutil.ContainsFinalizer(organization, key.OrganizationFinalizerName) {

		controllerutil.AddFinalizer(organization, key.OrganizationFinalizerName)

		// Update the object immediately
		if err := r.Update(ctx, organization); err != nil {
			return ctrl.Result{}, err
		}
	}

	// if organization has no ID, it means it was not created yet or it was already created and the status field is just empty (after e.g. restore)
	// let's check if we could discover the ID by listing all organizations
	if organization.Status.ID == "" {
		// try to adopt existing organization
		adopted, err := r.adoptExisting(ctx, scope, organization)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !adopted {
			// if adoption failed, create new organization
			if err := r.createOrUpdate(ctx, scope, organization); err != nil {
				return ctrl.Result{}, err
			}
		}

		if err := r.Status().Update(ctx, organization); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// update status fields
	// reflect organization status into CR
	garmOrganization, err := scope.GetOrganization(
		organizations.NewGetOrgParams().
			WithOrgID(organization.Status.ID),
	)

	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Updating organization status")
	organization.Status.PoolManagerFailureReason = garmOrganization.Payload.PoolManagerStatus.FailureReason
	organization.Status.PoolManagerIsRunning = garmOrganization.Payload.PoolManagerStatus.IsRunning
	if err := r.Status().Update(ctx, organization); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileDelete(ctx context.Context, scope garmClient.OrganizationClient, organization *garmoperatorv1alpha1.Organization) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	log.Info("start deleting organization")

	err := scope.DeleteOrganization(
		organizations.NewDeleteOrgParams().
			WithOrgID(organization.Status.ID),
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	if controllerutil.ContainsFinalizer(organization, key.OrganizationFinalizerName) {
		controllerutil.RemoveFinalizer(organization, key.OrganizationFinalizerName)

		// update immediately
		if err := r.Update(ctx, organization); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("organization deletion done")

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) createOrUpdate(ctx context.Context, scope garmClient.OrganizationClient, organization *garmoperatorv1alpha1.Organization) error {
	log := log.FromContext(ctx)

	log.Info("organization doesn't exist yet, creating...")
	retValue, err := scope.CreateOrganization(
		organizations.NewCreateOrgParams().
			WithBody(params.CreateOrgParams{
				Name:            organization.Name,
				CredentialsName: organization.Spec.CredentialsName,
				WebhookSecret:   organization.Spec.WebhookSecret, // gh hook secret
			}))
	log.V(1).Info(fmt.Sprintf("organization %s created - return Value %v", organization.Name, retValue))
	if err != nil {
		log.Info("DEBUG", "client.CreateOrganization error", err)
		return err
	}

	// reflect organization status into CR
	organization.Status.ID = retValue.Payload.ID
	organization.Status.PoolManagerFailureReason = retValue.Payload.PoolManagerStatus.FailureReason
	organization.Status.PoolManagerIsRunning = retValue.Payload.PoolManagerStatus.IsRunning
	if err := r.Status().Update(ctx, organization); err != nil {
		return err
	}

	return nil
}

func (r *OrganizationReconciler) adoptExisting(ctx context.Context, scope garmClient.OrganizationClient, organization *garmoperatorv1alpha1.Organization) (bool, error) {
	log := log.FromContext(ctx)

	organizations, err := scope.ListOrganizations(organizations.NewListOrgsParams())
	if err != nil {
		return false, err
	}

	// TODO: better logging and additional information in debug log
	log.Info(fmt.Sprintf("%d organizations discovered", len(organizations.Payload)))

	for _, garmOrganization := range organizations.Payload {
		if garmOrganization.Name == organization.Name {
			log.Info("garm organization object found for given organization CR")
			log.Info(fmt.Sprintf("organization: %#v", garmOrganization))
			organization.Status.ID = garmOrganization.ID
			organization.Status.PoolManagerFailureReason = garmOrganization.PoolManagerStatus.FailureReason
			organization.Status.PoolManagerIsRunning = garmOrganization.PoolManagerStatus.IsRunning
			if err := r.Status().Update(ctx, organization); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrganizationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Organization{}).
		Complete(r)
}
