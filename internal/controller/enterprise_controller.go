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

	if enterprise.Status.ID == "" {
		garmEnterprise, err := r.getExistingGarmEnterprise(ctx, client, enterprise)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !reflect.ValueOf(garmEnterprise).IsZero() {
			return r.syncExistingEnterprise(ctx, garmEnterprise, enterprise, webhookSecret)
		}

		return r.createEnterprise(ctx, client, enterprise, webhookSecret)
	}

	if enterprise.Status.ID != "" {
		return r.updateEnterprise(ctx, client, enterprise, webhookSecret)
	}

	log.Info("reconciling enterprise successfully done")

	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) syncExistingEnterprise(ctx context.Context, garmEnterprise params.Enterprise, enterprise *garmoperatorv1alpha1.Enterprise, webhookSecret string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	if !strings.EqualFold(garmEnterprise.CredentialsName, enterprise.Spec.CredentialsName) &&
		!strings.EqualFold(garmEnterprise.WebhookSecret, webhookSecret) {
		err := fmt.Errorf("enterprise with the same name already exists, but credentials and/or webhook secret are different. Please delete the existing enterprise first")
		event.Error(r.Recorder, enterprise, err.Error())
		return ctrl.Result{}, err
	}

	log.Info("garm enterprise object found for given enterprise CR")

	enterprise.Status.ID = garmEnterprise.ID
	enterprise.Status.PoolManagerFailureReason = garmEnterprise.PoolManagerStatus.FailureReason
	enterprise.Status.PoolManagerIsRunning = garmEnterprise.PoolManagerStatus.IsRunning

	if err := r.Status().Update(ctx, enterprise); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("existing garm enterprise object successfully adopted")
	event.Info(r.Recorder, enterprise, "existing garm enterprise object successfully adopted")

	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) createEnterprise(ctx context.Context, client garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise, webhookSecret string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	log.Info("status.ID is empty and enterprise doesn't exist on garm side. Creating new enterprise in garm")
	event.Creating(r.Recorder, enterprise, "enterprise doesn't exist on garm side")

	retValue, err := client.CreateEnterprise(
		enterprises.NewCreateEnterpriseParams().
			WithBody(params.CreateEnterpriseParams{
				Name:            enterprise.Name,
				CredentialsName: enterprise.Spec.CredentialsName,
				WebhookSecret:   webhookSecret, // gh hook secret
			}))
	log.V(1).Info(fmt.Sprintf("enterprise %s created - return Value %v", enterprise.Name, retValue))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.CreateEnterprise error: %s", err))
		return ctrl.Result{}, err
	}

	enterprise.Status.ID = retValue.Payload.ID
	enterprise.Status.PoolManagerFailureReason = retValue.Payload.PoolManagerStatus.FailureReason
	enterprise.Status.PoolManagerIsRunning = retValue.Payload.PoolManagerStatus.IsRunning

	if err := r.Status().Update(ctx, enterprise); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("creating enterprise in garm succeeded")
	event.Info(r.Recorder, enterprise, "creating enterprise in garm succeeded")

	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) updateEnterprise(ctx context.Context, client garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise, webhookSecret string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	log.Info("comparing enterprise with garm enterprise object")
	garmEnterprise, err := client.GetEnterprise(
		enterprises.NewGetEnterpriseParams().
			WithEnterpriseID(enterprise.Status.ID),
	)
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.GetEnterprise error: %s", err))
		return ctrl.Result{}, err
	}

	enterprise.Status.PoolManagerFailureReason = garmEnterprise.Payload.PoolManagerStatus.FailureReason
	enterprise.Status.PoolManagerIsRunning = garmEnterprise.Payload.PoolManagerStatus.IsRunning

	if err := r.Status().Update(ctx, enterprise); err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("compare credentials and webhook secret with garm enterprise object")

	if enterprise.Spec.CredentialsName != garmEnterprise.Payload.CredentialsName &&
		webhookSecret != garmEnterprise.Payload.WebhookSecret {
		log.Info("enterprise credentials or webhook secret has changed, updating garm enterprise object")
		event.Updating(r.Recorder, enterprise, "enterprise credentials or webhook secret has changed")

		// update credentials and webhook secret
		retValue, err := client.UpdateEnterprise(
			enterprises.NewUpdateEnterpriseParams().
				WithEnterpriseID(enterprise.Status.ID).
				WithBody(params.UpdateEntityParams{
					CredentialsName: enterprise.Spec.CredentialsName,
					WebhookSecret:   webhookSecret, // gh hook secret
				}))
		if err != nil {
			log.V(1).Info(fmt.Sprintf("client.UpdateEnterprise error: %s", err))
			return ctrl.Result{}, err
		}

		enterprise.Status.PoolManagerFailureReason = retValue.Payload.PoolManagerStatus.FailureReason
		enterprise.Status.PoolManagerIsRunning = retValue.Payload.PoolManagerStatus.IsRunning
	}

	if err := r.Status().Update(ctx, enterprise); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) getExistingGarmEnterprise(ctx context.Context, client garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise) (params.Enterprise, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	log.Info("status.ID is empty, checking if enterprise already exists on garm side")
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
	event.Deleting(r.Recorder, enterprise, "starting organization deletion")

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
