// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"fmt"
	"github.com/cloudbase/garm/client/endpoints"
	"github.com/cloudbase/garm/params"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/event"
	"github.com/mercedes-benz/garm-operator/pkg/util"
	"github.com/mercedes-benz/garm-operator/pkg/util/annotations"
	"github.com/mercedes-benz/garm-operator/pkg/util/conditions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
)

// EndpointReconciler reconciles a Endpoint object
type EndpointReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=endpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=endpoints/finalizers,verbs=update
// +kubebuilder:rbac:groups="",namespace=xxxxx,resources=secrets,verbs=get;list;watch;

func (r *EndpointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	endpoint := &garmoperatorv1alpha1.Endpoint{}
	if err := r.Get(ctx, req.NamespacedName, endpoint); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Endpoint resource not found.")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Ignore objects that are paused
	if annotations.IsPaused(endpoint) {
		log.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	endpointClient := garmClient.NewEndpointClient()

	// Handle deleted endpoints
	if !endpoint.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, endpointClient, endpoint)
	}

	return r.reconcileNormal(ctx, endpointClient, endpoint)
}

func (r *EndpointReconciler) reconcileNormal(ctx context.Context, client garmClient.EndpointClient, endpoint *garmoperatorv1alpha1.Endpoint) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("endpoint", endpoint.Name)

	// If the Endpoint doesn't have our finalizer, add it.
	if err := r.ensureFinalizer(ctx, endpoint); err != nil {
		return ctrl.Result{}, err
	}

	// get endpoint in garm db with resource name
	garmEndpoint, err := r.getExistingEndpoint(client, endpoint.Name)
	if err != nil {
		event.Error(r.Recorder, endpoint, err.Error())
		conditions.MarkFalse(endpoint, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
		if err := r.Status().Update(ctx, endpoint); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// if not found, create endpoint in garm db
	if reflect.ValueOf(garmEndpoint).IsZero() {
		garmEndpoint, err = r.createEndpoint(ctx, client, endpoint)
		if err != nil {
			event.Error(r.Recorder, endpoint, err.Error())
			conditions.MarkFalse(endpoint, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
			if err := r.Status().Update(ctx, endpoint); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
	}

	// update endpoint cr anytime the endpoint in garm db changes
	garmEndpoint, err = r.updateEndpoint(ctx, client, endpoint)
	if err != nil {
		event.Error(r.Recorder, endpoint, err.Error())
		conditions.MarkFalse(endpoint, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
		if err := r.Status().Update(ctx, endpoint); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// set and update endpoint status
	conditions.MarkTrue(endpoint, conditions.ReadyCondition, conditions.SuccessfulReconcileReason, "")
	if err := r.Status().Update(ctx, endpoint); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciling endpoint successfully done")
	return ctrl.Result{}, nil
}

func (r *EndpointReconciler) getExistingEndpoint(client garmClient.EndpointClient, name string) (params.GithubEndpoint, error) {
	endpoint, err := client.GetEndpoint(endpoints.NewGetGithubEndpointParams().WithName(name))
	if err != nil && garmClient.IsNotFoundError(err) {
		return params.GithubEndpoint{}, nil
	}

	if err != nil {
		return params.GithubEndpoint{}, err
	}

	return endpoint.Payload, nil
}

func (r *EndpointReconciler) createEndpoint(ctx context.Context, client garmClient.EndpointClient, endpoint *garmoperatorv1alpha1.Endpoint) (params.GithubEndpoint, error) {
	log := log.FromContext(ctx)
	log.WithValues("endpoint", endpoint.Name)

	log.Info("Endpoint doesn't exist on garm side. Creating new endpoint in garm.")
	event.Creating(r.Recorder, endpoint, "endpoint doesn't exist on garm side")

	retValue, err := client.CreateEndpoint(endpoints.NewCreateGithubEndpointParams().WithBody(params.CreateGithubEndpointParams{
		Name:          endpoint.Name,
		Description:   endpoint.Spec.Description,
		APIBaseURL:    endpoint.Spec.APIBaseURL,
		UploadBaseURL: endpoint.Spec.UploadBaseURL,
		BaseURL:       endpoint.Spec.BaseURL,
		CACertBundle:  endpoint.Spec.CACertBundle,
	}))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.CreateEndpoint error: %s", err))
		return params.GithubEndpoint{}, err
	}

	log.V(1).Info(fmt.Sprintf("endpoint %s created - return Value %v", endpoint.Name, retValue))

	log.Info("creating endpoint in garm succeeded")
	event.Info(r.Recorder, endpoint, "creating endpoint in garm succeeded")

	return retValue.Payload, nil
}

func (r *EndpointReconciler) updateEndpoint(ctx context.Context, client garmClient.EndpointClient, endpoint *garmoperatorv1alpha1.Endpoint) (params.GithubEndpoint, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("update endpoint")

	retValue, err := client.UpdateEndpoint(
		endpoints.NewUpdateGithubEndpointParams().
			WithName(endpoint.Name).
			WithBody(params.UpdateGithubEndpointParams{
				Description:   util.StringPtr(endpoint.Spec.Description),
				APIBaseURL:    util.StringPtr(endpoint.Spec.APIBaseURL),
				UploadBaseURL: util.StringPtr(endpoint.Spec.UploadBaseURL),
				BaseURL:       util.StringPtr(endpoint.Spec.BaseURL),
				CACertBundle:  endpoint.Spec.CACertBundle,
			}))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.UpdateEndpoint error: %s", err))
		return params.GithubEndpoint{}, err
	}

	return retValue.Payload, nil
}

func (r *EndpointReconciler) reconcileDelete(ctx context.Context, client garmClient.EndpointClient, endpoint *garmoperatorv1alpha1.Endpoint) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("endpoint", endpoint.Name)

	log.Info("starting endpoint deletion")
	event.Deleting(r.Recorder, endpoint, "starting endpoint deletion")
	conditions.MarkFalse(endpoint, conditions.ReadyCondition, conditions.DeletingReason, conditions.DeletingEndpointMsg)
	if err := r.Status().Update(ctx, endpoint); err != nil {
		return ctrl.Result{}, err
	}

	err := client.DeleteEndpoint(
		endpoints.NewDeleteGithubEndpointParams().
			WithName(endpoint.Name),
	)
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.DeleteEndpoint error: %s", err))
		event.Error(r.Recorder, endpoint, err.Error())
		conditions.MarkFalse(endpoint, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
		if err := r.Status().Update(ctx, endpoint); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if controllerutil.ContainsFinalizer(endpoint, key.EndpointFinalizerName) {
		controllerutil.RemoveFinalizer(endpoint, key.EndpointFinalizerName)
		if err := r.Update(ctx, endpoint); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("endpoint deletion done")

	return ctrl.Result{}, nil
}

func (r *EndpointReconciler) ensureFinalizer(ctx context.Context, endpoint *garmoperatorv1alpha1.Endpoint) error {
	if !controllerutil.ContainsFinalizer(endpoint, key.EndpointFinalizerName) {
		controllerutil.AddFinalizer(endpoint, key.EndpointFinalizerName)
		return r.Update(ctx, endpoint)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Endpoint{}).
		Complete(r)
}
