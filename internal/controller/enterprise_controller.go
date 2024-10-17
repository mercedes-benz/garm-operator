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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	"github.com/mercedes-benz/garm-operator/pkg/annotations"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/conditions"
	"github.com/mercedes-benz/garm-operator/pkg/event"
	"github.com/mercedes-benz/garm-operator/pkg/secret"
)

// EnterpriseReconciler reconciles a Enterprise object
type EnterpriseReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=enterprises,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=enterprises/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=enterprises/finalizers,verbs=update
// +kubebuilder:rbac:groups="",namespace=xxxxx,resources=secrets,verbs=get;list;watch;

func (r *EnterpriseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	enterprise := &garmoperatorv1beta1.Enterprise{}
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

	// Handle deleted enterprises
	if !enterprise.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, enterpriseClient, enterprise)
	}

	return r.reconcileNormal(ctx, enterpriseClient, enterprise)
}

func (r *EnterpriseReconciler) reconcileNormal(ctx context.Context, client garmClient.EnterpriseClient, enterprise *garmoperatorv1beta1.Enterprise) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	// If the Enterprise doesn't have our finalizer, add it.
	if err := r.ensureFinalizer(ctx, enterprise); err != nil {
		return ctrl.Result{}, err
	}

	webhookSecret, err := secret.FetchRef(ctx, r.Client, &enterprise.Spec.WebhookSecretRef, enterprise.Namespace)
	if err != nil {
		conditions.MarkFalse(enterprise, conditions.ReadyCondition, conditions.FetchingSecretRefFailedReason, err.Error())
		conditions.MarkFalse(enterprise, conditions.SecretReference, conditions.FetchingSecretRefFailedReason, err.Error())
		conditions.MarkUnknown(enterprise, conditions.CredentialsReference, conditions.UnknownReason, conditions.CredentialsNotReconciledYetMsg)
		if conditions.Get(enterprise, conditions.PoolManager) == nil {
			conditions.MarkUnknown(enterprise, conditions.PoolManager, conditions.UnknownReason, conditions.GarmServerNotReconciledYetMsg)
		}
		if err := r.Status().Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	conditions.MarkTrue(enterprise, conditions.SecretReference, conditions.FetchingSecretRefSuccessReason, "")

	credentials, err := r.getCredentialsRef(ctx, enterprise)
	if err != nil {
		conditions.MarkFalse(enterprise, conditions.ReadyCondition, conditions.FetchingCredentialsRefFailedReason, err.Error())
		conditions.MarkFalse(enterprise, conditions.CredentialsReference, conditions.FetchingCredentialsRefFailedReason, err.Error())
		if conditions.Get(enterprise, conditions.PoolManager) == nil {
			conditions.MarkUnknown(enterprise, conditions.PoolManager, conditions.UnknownReason, conditions.GarmServerNotReconciledYetMsg)
		}
		if err := r.Status().Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	conditions.MarkTrue(enterprise, conditions.CredentialsReference, conditions.FetchingCredentialsRefSuccessReason, "")

	garmEnterprise, err := r.getExistingGarmEnterprise(ctx, client, enterprise)
	if err != nil {
		event.Error(r.Recorder, enterprise, err.Error())
		conditions.MarkFalse(enterprise, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
		if err := r.Status().Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// create enterprise on garm side if it does not exist
	if reflect.ValueOf(garmEnterprise).IsZero() {
		garmEnterprise, err = r.createEnterprise(ctx, client, enterprise, webhookSecret)
		if err != nil {
			event.Error(r.Recorder, enterprise, err.Error())
			conditions.MarkFalse(enterprise, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
			if err := r.Status().Update(ctx, enterprise); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
	}

	// update enterprise anytime
	garmEnterprise, err = r.updateEnterprise(ctx, client, garmEnterprise.ID, params.UpdateEntityParams{
		CredentialsName:  credentials.Name,
		WebhookSecret:    webhookSecret,
		PoolBalancerType: enterprise.Spec.PoolBalancerType,
	})
	if err != nil {
		event.Error(r.Recorder, enterprise, err.Error())
		conditions.MarkFalse(enterprise, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
		if err := r.Status().Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// set and update enterprise status
	enterprise.Status.ID = garmEnterprise.ID
	conditions.MarkTrue(enterprise, conditions.ReadyCondition, conditions.SuccessfulReconcileReason, "")
	conditions.MarkTrue(enterprise, conditions.PoolManager, conditions.PoolManagerRunningReason, "")

	if !garmEnterprise.PoolManagerStatus.IsRunning {
		conditions.MarkFalse(enterprise, conditions.ReadyCondition, conditions.PoolManagerFailureReason, "Pool Manager is not running")
		conditions.MarkFalse(enterprise, conditions.PoolManager, conditions.PoolManagerFailureReason, garmEnterprise.PoolManagerStatus.FailureReason)
	}

	if err := r.Status().Update(ctx, enterprise); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciling enterprise successfully done")
	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) createEnterprise(ctx context.Context, client garmClient.EnterpriseClient, enterprise *garmoperatorv1beta1.Enterprise, webhookSecret string) (params.Enterprise, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	log.Info("Enterprise doesn't exist on garm side. Creating new enterprise in garm.")
	event.Creating(r.Recorder, enterprise, "enterprise doesn't exist on garm side")

	retValue, err := client.CreateEnterprise(
		enterprises.NewCreateEnterpriseParams().
			WithBody(params.CreateEnterpriseParams{
				Name:             enterprise.Name,
				CredentialsName:  enterprise.GetCredentialsName(),
				WebhookSecret:    webhookSecret, // gh hook secret
				PoolBalancerType: enterprise.Spec.PoolBalancerType,
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

func (r *EnterpriseReconciler) updateEnterprise(ctx context.Context, client garmClient.EnterpriseClient, statusID string, updateParams params.UpdateEntityParams) (params.Enterprise, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("update credentials and webhook secret in garm enterprise")

	// update credentials and webhook secret
	retValue, err := client.UpdateEnterprise(
		enterprises.NewUpdateEnterpriseParams().
			WithEnterpriseID(statusID).
			WithBody(updateParams))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.UpdateEnterprise error: %s", err))
		return params.Enterprise{}, err
	}

	return retValue.Payload, nil
}

func (r *EnterpriseReconciler) getExistingGarmEnterprise(ctx context.Context, client garmClient.EnterpriseClient, enterprise *garmoperatorv1beta1.Enterprise) (params.Enterprise, error) {
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

func (r *EnterpriseReconciler) reconcileDelete(ctx context.Context, client garmClient.EnterpriseClient, enterprise *garmoperatorv1beta1.Enterprise) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	log.Info("starting enterprise deletion")
	event.Deleting(r.Recorder, enterprise, "starting enterprise deletion")
	conditions.MarkFalse(enterprise, conditions.ReadyCondition, conditions.DeletingReason, conditions.DeletingEnterpriseMsg)
	if err := r.Status().Update(ctx, enterprise); err != nil {
		return ctrl.Result{}, err
	}

	err := client.DeleteEnterprise(
		enterprises.NewDeleteEnterpriseParams().
			WithEnterpriseID(enterprise.Status.ID),
	)
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.DeleteEnterprise error: %s", err))
		event.Error(r.Recorder, enterprise, err.Error())
		conditions.MarkFalse(enterprise, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
		if err := r.Status().Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}
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

func (r *EnterpriseReconciler) getCredentialsRef(ctx context.Context, enterprise *garmoperatorv1beta1.Enterprise) (*garmoperatorv1beta1.GitHubCredentials, error) {
	creds := &garmoperatorv1beta1.GitHubCredentials{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: enterprise.Namespace,
		Name:      enterprise.Spec.CredentialsRef.Name,
	}, creds)
	if err != nil {
		return creds, err
	}
	return creds, nil
}

func (r *EnterpriseReconciler) ensureFinalizer(ctx context.Context, enterprise *garmoperatorv1beta1.Enterprise) error {
	if !controllerutil.ContainsFinalizer(enterprise, key.EnterpriseFinalizerName) {
		controllerutil.AddFinalizer(enterprise, key.EnterpriseFinalizerName)
		return r.Update(ctx, enterprise)
	}
	return nil
}

func (r *EnterpriseReconciler) findEnterprisesForCredentials(ctx context.Context, obj client.Object) []reconcile.Request {
	credentials, ok := obj.(*garmoperatorv1beta1.GitHubCredentials)
	if !ok {
		return nil
	}

	var enterprises garmoperatorv1beta1.EnterpriseList
	if err := r.List(ctx, &enterprises); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, enterprise := range enterprises.Items {
		if enterprise.GetCredentialsName() == credentials.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: enterprise.Namespace,
					Name:      enterprise.Name,
				},
			})
		}
	}

	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnterpriseReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1beta1.Enterprise{}).
		Watches(
			&garmoperatorv1beta1.GitHubCredentials{},
			handler.EnqueueRequestsFromMapFunc(r.findEnterprisesForCredentials),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		WithOptions(options).
		Complete(r)
}
