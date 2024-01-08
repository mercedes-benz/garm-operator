// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudbase/garm/client/enterprises"
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

// EnterpriseReconciler reconciles a Enterprise object
type EnterpriseReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	BaseURL  string
	Username string
	Password string
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=enterprises,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=enterprises/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=enterprises/finalizers,verbs=update
// +kubebuilder:rbac:groups="",namespace=xxxxx,resources=secrets,verbs=get;list;watch;

func (r *EnterpriseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	enterprise := &garmoperatorv1alpha1.Enterprise{}
	err := r.Get(ctx, req.NamespacedName, enterprise)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("object was not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Ignore objects that are paused
	if annotations.IsPaused(enterprise) {
		log.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	annotations.SetLastSyncTime(enterprise)
	err = r.Update(ctx, enterprise)
	if err != nil {
		log.Error(err, "can not set annotation")
		return ctrl.Result{}, err
	}

	enterpriseClient := garmClient.NewEnterpriseClient()
	err = enterpriseClient.Login(garmClient.GarmScopeParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
		// Debug:    true,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// Handle deleted enterprises
	if !enterprise.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, enterpriseClient, enterprise)
	}

	return r.reconcileNormal(ctx, enterpriseClient, enterprise)
}

func (r *EnterpriseReconciler) reconcileNormal(ctx context.Context, client garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	// If the Enterprise doesn't have our finalizer, add it.
	if err := r.ensureFinalizer(ctx, enterprise); err != nil {
		return ctrl.Result{}, err
	}

	webhookSecret, err := secret.FetchRef(ctx, r.Client, &enterprise.Spec.WebhookSecretRef, enterprise.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	garmEnterprise, err := r.getExistingGarmEnterprise(ctx, client, enterprise)
	if err != nil {
		return ctrl.Result{}, err
	}

	// create enterprise on garm side if it does not exist
	if reflect.ValueOf(garmEnterprise).IsZero() {
		garmEnterprise, err = r.createEnterprise(ctx, client, enterprise, webhookSecret)
		if err != nil {
			event.Error(r.Recorder, enterprise, err.Error())
			return ctrl.Result{}, err
		}
	}

	// update enterprise anytime
	garmEnterprise, err = r.updateEnterprise(ctx, client, garmEnterprise.ID, webhookSecret, enterprise.Spec.CredentialsName)
	if err != nil {
		event.Error(r.Recorder, enterprise, err.Error())
		return ctrl.Result{}, err
	}

	// set and update enterprise status
	enterprise.Status.ID = garmEnterprise.ID
	enterprise.Status.PoolManagerFailureReason = garmEnterprise.PoolManagerStatus.FailureReason
	enterprise.Status.PoolManagerIsRunning = garmEnterprise.PoolManagerStatus.IsRunning

	if err := r.Status().Update(ctx, enterprise); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciling enterprise successfully done")

	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) createEnterprise(ctx context.Context, client garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise, webhookSecret string) (params.Enterprise, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	log.Info("Enterprise doesn't exist on garm side. Creating new enterprise in garm.")
	event.Creating(r.Recorder, enterprise, "enterprise doesn't exist on garm side")

	retValue, err := client.CreateEnterprise(
		enterprises.NewCreateEnterpriseParams().
			WithBody(params.CreateEnterpriseParams{
				Name:            enterprise.Name,
				CredentialsName: enterprise.Spec.CredentialsName,
				WebhookSecret:   webhookSecret, // gh hook secret
			}))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.CreateEnterprise error: %s", err))
		return params.Enterprise{}, err
	}

	log.V(1).Info(fmt.Sprintf("enterprise %s created - return Value %v", enterprise.Name, retValue))

	log.Info("creating enterprise in garm succeeded")
	event.Info(r.Recorder, enterprise, "creating enterprise in garm succeeded")

	return retValue.Payload, nil
}

func (r *EnterpriseReconciler) updateEnterprise(ctx context.Context, client garmClient.EnterpriseClient, statusID, webhookSecret, credentialsName string) (params.Enterprise, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("update credentials and webhook secret in garm enterprise")

	// update credentials and webhook secret
	retValue, err := client.UpdateEnterprise(
		enterprises.NewUpdateEnterpriseParams().
			WithEnterpriseID(statusID).
			WithBody(params.UpdateEntityParams{
				CredentialsName: credentialsName,
				WebhookSecret:   webhookSecret, // gh hook secret
			}))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.UpdateEnterprise error: %s", err))
		return params.Enterprise{}, err
	}

	return retValue.Payload, nil
}

func (r *EnterpriseReconciler) getExistingGarmEnterprise(ctx context.Context, client garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise) (params.Enterprise, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	log.Info("checking if enterprise already exists on garm side")
	enterprises, err := client.ListEnterprises(enterprises.NewListEnterprisesParams())
	if err != nil {
		return params.Enterprise{}, fmt.Errorf("getExistingGarmEnterprise: %w", err)
	}

	log.Info(fmt.Sprintf("%d enterprises discovered", len(enterprises.Payload)))
	log.V(1).Info(fmt.Sprintf("enterprises on garm side: %#v", enterprises.Payload))

	for _, garmEnterprise := range enterprises.Payload {
		if strings.EqualFold(garmEnterprise.Name, enterprise.Name) {
			return garmEnterprise, nil
		}
	}
	return params.Enterprise{}, nil
}

func (r *EnterpriseReconciler) reconcileDelete(ctx context.Context, scope garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	log.Info("starting enterprise deletion")
	event.Deleting(r.Recorder, enterprise, "starting enterprise deletion")

	err := scope.DeleteEnterprise(
		enterprises.NewDeleteEnterpriseParams().
			WithEnterpriseID(enterprise.Status.ID),
	)
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.DeleteEnterprise error: %s", err))
		event.Error(r.Recorder, enterprise, err.Error())
		return ctrl.Result{}, err
	}

	if controllerutil.ContainsFinalizer(enterprise, key.EnterpriseFinalizerName) {
		controllerutil.RemoveFinalizer(enterprise, key.EnterpriseFinalizerName)

		// update immediately
		if err := r.Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("enterprise deletion done")

	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) ensureFinalizer(ctx context.Context, pool *garmoperatorv1alpha1.Enterprise) error {
	if !controllerutil.ContainsFinalizer(pool, key.EnterpriseFinalizerName) {
		controllerutil.AddFinalizer(pool, key.EnterpriseFinalizerName)
		return r.Update(ctx, pool)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnterpriseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Enterprise{}).
		Complete(r)
}
