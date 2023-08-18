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
	"strings"

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
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/secret"
)

// OrganizationReconciler reconciles a Organization object
type OrganizationReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	BaseURL  string
	Username string
	Password string
}

//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=organizations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=organizations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=organizations/finalizers,verbs=update

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
	log.WithValues("organization", organization.Name)

	// If the Organization doesn't has our finalizer, add it.
	if !controllerutil.ContainsFinalizer(organization, key.OrganizationFinalizerName) {
		log.V(1).Info(fmt.Sprintf("organization doesn't have the %s finalizer, adding it", key.OrganizationFinalizerName))
		controllerutil.AddFinalizer(organization, key.OrganizationFinalizerName)

		// Update the object immediately
		if err := r.Update(ctx, organization); err != nil {
			return ctrl.Result{}, err
		}
	}

	webhookSecret, err := secret.FetchRef(ctx, r.Client, &organization.Spec.WebhookSecretRef, organization.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	if organization.Status.ID == "" {
		log.Info("status.ID is empty, checking if organization already exists on garm side")
		organizations, err := scope.ListOrganizations(organizations.NewListOrgsParams())
		if err != nil {
			return ctrl.Result{}, err
		}

		log.Info(fmt.Sprintf("%d organizations discovered", len(organizations.Payload)))
		log.V(1).Info(fmt.Sprintf("organizations on garm side: %#v", organizations.Payload))

		for _, garmOrganization := range organizations.Payload {
			if strings.EqualFold(garmOrganization.Name, organization.Name) {

				if !strings.EqualFold(garmOrganization.CredentialsName, organization.Spec.CredentialsName) &&
					!strings.EqualFold(garmOrganization.WebhookSecret, webhookSecret) {
					return ctrl.Result{}, fmt.Errorf("organization with the same name already exists, but credentials and/or webhook secret are different. Please delete the existing organization first")
				}

				log.Info("garm organization object found for given organization CR")

				organization.Status.ID = garmOrganization.ID
				organization.Status.PoolManagerFailureReason = garmOrganization.PoolManagerStatus.FailureReason
				organization.Status.PoolManagerIsRunning = garmOrganization.PoolManagerStatus.IsRunning

				if err := r.Status().Update(ctx, organization); err != nil {
					return ctrl.Result{}, err
				}

				log.Info("existing garm organization object successfully adopted")

				return ctrl.Result{}, nil
			}
		}
	}

	switch {
	case organization.Status.ID == "":

		log.Info("status.ID is empty and organization doesn't exist on garm side. Creating new organization in garm")
		retValue, err := scope.CreateOrganization(
			organizations.NewCreateOrgParams().
				WithBody(params.CreateOrgParams{
					Name:            organization.Name,
					CredentialsName: organization.Spec.CredentialsName,
					WebhookSecret:   webhookSecret, // gh hook secret
				}))
		log.V(1).Info(fmt.Sprintf("organization %s created - return Value %v", organization.Name, retValue))
		if err != nil {
			log.V(1).Info(fmt.Sprintf("client.CreateOrganization error: %s", err))
			return ctrl.Result{}, err
		}

		organization.Status.ID = retValue.Payload.ID
		organization.Status.PoolManagerFailureReason = retValue.Payload.PoolManagerStatus.FailureReason
		organization.Status.PoolManagerIsRunning = retValue.Payload.PoolManagerStatus.IsRunning

		if err := r.Status().Update(ctx, organization); err != nil {
			return ctrl.Result{}, err
		}

		log.Info("creating organization in garm succeeded")

		return ctrl.Result{}, nil

	case organization.Status.ID != "":

		log.Info("comparing organization with garm organization object")
		garmOrganization, err := scope.GetOrganization(
			organizations.NewGetOrgParams().
				WithOrgID(organization.Status.ID),
		)
		if err != nil {
			log.V(1).Info(fmt.Sprintf("client.GetOrganization error: %s", err))
			return ctrl.Result{}, err
		}

		organization.Status.PoolManagerFailureReason = garmOrganization.Payload.PoolManagerStatus.FailureReason
		organization.Status.PoolManagerIsRunning = garmOrganization.Payload.PoolManagerStatus.IsRunning

		log.V(1).Info("compare credentials and webhook secret with garm organization object")

		if organization.Spec.CredentialsName != garmOrganization.Payload.CredentialsName &&
			webhookSecret != garmOrganization.Payload.WebhookSecret {

			log.Info("organization credentials or webhook secret has changed, updating garm organization object")

			// update credentials and webhook secret
			retValue, err := scope.UpdateOrganization(
				organizations.NewUpdateOrgParams().
					WithOrgID(organization.Status.ID).
					WithBody(params.UpdateEntityParams{
						CredentialsName: organization.Spec.CredentialsName,
						WebhookSecret:   webhookSecret, // gh hook secret
					}))
			if err != nil {
				log.V(1).Info(fmt.Sprintf("client.UpdateOrganization error: %s", err))
				return ctrl.Result{}, err
			}

			organization.Status.PoolManagerFailureReason = retValue.Payload.PoolManagerStatus.FailureReason
			organization.Status.PoolManagerIsRunning = retValue.Payload.PoolManagerStatus.IsRunning

			if err := r.Status().Update(ctx, organization); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
	}

	log.Info("reconciling organization successfully done")

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileDelete(ctx context.Context, scope garmClient.OrganizationClient, organization *garmoperatorv1alpha1.Organization) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("organization", organization.Name)

	log.Info("starting organization deletion")

	err := scope.DeleteOrganization(
		organizations.NewDeleteOrgParams().
			WithOrgID(organization.Status.ID),
	)
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.DeleteOrganization error: %s", err))
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

// SetupWithManager sets up the controller with the Manager.
func (r *OrganizationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Organization{}).
		Complete(r)
}
