// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/params"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/event"
	"github.com/mercedes-benz/garm-operator/pkg/secret"
	"github.com/mercedes-benz/garm-operator/pkg/util/annotations"
)

// OrganizationReconciler reconciles a Organization object
type OrganizationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	BaseURL  string
	Username string
	Password string
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
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

	// Ignore objects that are paused
	if annotations.IsPaused(organization) {
		log.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	organizationClient := garmClient.NewOrganizationClient()
	err = organizationClient.Login(garmClient.GarmScopeParams{
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
		return r.reconcileDelete(ctx, organizationClient, organization)
	}

	return r.reconcileNormal(ctx, organizationClient, organization)
}

func (r *OrganizationReconciler) reconcileNormal(ctx context.Context, client garmClient.OrganizationClient, organization *garmoperatorv1alpha1.Organization) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("organization", organization.Name)

	// If the Organization doesn't has our finalizer, add it.
	if err := r.ensureFinalizer(ctx, organization); err != nil {
		return ctrl.Result{}, err
	}

	webhookSecret, err := secret.FetchRef(ctx, r.Client, &organization.Spec.WebhookSecretRef, organization.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	garmOrganization, err := r.getExistingGarmOrg(ctx, client, organization)
	if err != nil {
		return ctrl.Result{}, err
	}

	// create organization on garm side if it does not exist
	if reflect.ValueOf(garmOrganization).IsZero() {
		garmOrganization, err = r.createOrganization(ctx, client, organization, webhookSecret)
		if err != nil {
			event.Error(r.Recorder, organization, err.Error())
			return ctrl.Result{}, err
		}
	}

	// update organization anytime
	garmOrganization, err = r.updateOrganization(ctx, client, garmOrganization.ID, webhookSecret, organization.Spec.CredentialsName)
	if err != nil {
		event.Error(r.Recorder, organization, err.Error())
		return ctrl.Result{}, err
	}

	// set and update organization status
	organization.Status.ID = garmOrganization.ID
	organization.Status.PoolManagerFailureReason = garmOrganization.PoolManagerStatus.FailureReason
	organization.Status.PoolManagerIsRunning = garmOrganization.PoolManagerStatus.IsRunning

	if err := r.Status().Update(ctx, organization); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciling organization successfully done")

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) createOrganization(ctx context.Context, client garmClient.OrganizationClient, organization *garmoperatorv1alpha1.Organization, webhookSecret string) (params.Organization, error) {
	log := log.FromContext(ctx)
	log.WithValues("organization", organization.Name)

	log.Info("Organization doesn't exist on garm side. Creating new organization in garm.")
	event.Creating(r.Recorder, organization, "organization doesn't exist on garm side")

	retValue, err := client.CreateOrganization(
		organizations.NewCreateOrgParams().
			WithBody(params.CreateOrgParams{
				Name:            organization.Name,
				CredentialsName: organization.Spec.CredentialsName,
				WebhookSecret:   webhookSecret, // gh hook secret
			}))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.CreateOrganization error: %s", err))
		return params.Organization{}, err
	}

	log.V(1).Info(fmt.Sprintf("organization %s created - return Value %v", organization.Name, retValue))

	log.Info("creating organization in garm succeeded")
	event.Info(r.Recorder, organization, "creating organization in garm succeeded")

	return retValue.Payload, nil
}

func (r *OrganizationReconciler) updateOrganization(ctx context.Context, client garmClient.OrganizationClient, statusID, webhookSecret, credentialsName string) (params.Organization, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("update credentials and webhook secret in garm organization")

	// update credentials and webhook secret
	retValue, err := client.UpdateOrganization(
		organizations.NewUpdateOrgParams().
			WithOrgID(statusID).
			WithBody(params.UpdateEntityParams{
				CredentialsName: credentialsName,
				WebhookSecret:   webhookSecret, // gh hook secret
			}))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.UpdateOrganization error: %s", err))
		return params.Organization{}, err
	}

	return retValue.Payload, nil
}

func (r *OrganizationReconciler) getExistingGarmOrg(ctx context.Context, client garmClient.OrganizationClient, organization *garmoperatorv1alpha1.Organization) (params.Organization, error) {
	log := log.FromContext(ctx)
	log.WithValues("organization", organization.Name)

	log.Info("checking if organization already exists on garm side")
	organizations, err := client.ListOrganizations(organizations.NewListOrgsParams())
	if err != nil {
		return params.Organization{}, fmt.Errorf("getExistingGarmOrg: %w", err)
	}

	log.Info(fmt.Sprintf("%d organizations discovered", len(organizations.Payload)))
	log.V(1).Info(fmt.Sprintf("organizations on garm side: %#v", organizations.Payload))

	for _, garmOrganization := range organizations.Payload {
		if strings.EqualFold(garmOrganization.Name, organization.Name) {
			return garmOrganization, nil
		}
	}
	return params.Organization{}, nil
}

func (r *OrganizationReconciler) reconcileDelete(ctx context.Context, scope garmClient.OrganizationClient, organization *garmoperatorv1alpha1.Organization) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("organization", organization.Name)

	log.Info("starting organization deletion")
	event.Deleting(r.Recorder, organization, "starting organization deletion")

	err := scope.DeleteOrganization(
		organizations.NewDeleteOrgParams().
			WithOrgID(organization.Status.ID),
	)
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.DeleteOrganization error: %s", err))
		event.Error(r.Recorder, organization, err.Error())
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

func (r *OrganizationReconciler) ensureFinalizer(ctx context.Context, pool *garmoperatorv1alpha1.Organization) error {
	if !controllerutil.ContainsFinalizer(pool, key.OrganizationFinalizerName) {
		controllerutil.AddFinalizer(pool, key.OrganizationFinalizerName)
		return r.Update(ctx, pool)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrganizationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Organization{}).
		Complete(r)
}
