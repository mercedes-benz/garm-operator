// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"errors"
	"reflect"

	garmapiserverparams "github.com/cloudbase/garm/apiserver/params"
	garmcontroller "github.com/cloudbase/garm/client/controller"
	"github.com/cloudbase/garm/client/controller_info"
	"github.com/cloudbase/garm/params"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	"github.com/mercedes-benz/garm-operator/pkg/annotations"
	garmclient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/util"
)

// GarmServerConfigReconciler reconciles a GarmServerConfig object
type GarmServerConfigReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=garmserverconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=garmserverconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=garmserverconfigs/finalizers,verbs=update

func (r *GarmServerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling GarmServerConfig")

	controllerClient := garmclient.NewControllerClient()

	garmServerConfig := &garmoperatorv1beta1.GarmServerConfig{}
	if err := r.Get(ctx, req.NamespacedName, garmServerConfig); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("object was not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Ignore objects that are paused
	if annotations.IsPaused(garmServerConfig) {
		log.Info("Reconciliation is paused for GarmServerConfig")
		return ctrl.Result{}, nil
	}

	return r.reconcileNormal(ctx, controllerClient, garmServerConfig)
}

func (r *GarmServerConfigReconciler) reconcileNormal(ctx context.Context, controllerClient garmclient.ControllerClient, garmServerConfig *garmoperatorv1beta1.GarmServerConfig) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	controllerInfo, err := r.getControllerInfo(controllerClient)
	if err != nil {
		log.Error(err, "Failed to get controller info")
		return ctrl.Result{}, err
	}

	// sync applied spec with controller info in garm
	newControllerInfo, err := r.updateControllerInfo(ctx, controllerClient, garmServerConfig, &controllerInfo)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update CR with new state from garm
	if err := r.updateGarmServerConfigStatus(ctx, newControllerInfo, garmServerConfig); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GarmServerConfigReconciler) updateControllerInfo(ctx context.Context, client garmclient.ControllerClient, garmServerConfigCR *garmoperatorv1beta1.GarmServerConfig, controllerInfo *params.ControllerInfo) (*params.ControllerInfo, error) {
	log := log.FromContext(ctx)

	if garmServerConfigCR.Spec.MetadataURL == controllerInfo.MetadataURL &&
		garmServerConfigCR.Spec.CallbackURL == controllerInfo.CallbackURL &&
		garmServerConfigCR.Spec.WebhookURL == controllerInfo.WebhookURL {
		log.Info("Controller info is up to date")
		return controllerInfo, nil
	}

	updateParams := garmcontroller.NewUpdateControllerParams().WithBody(params.UpdateControllerParams{
		MetadataURL: util.StringPtr(garmServerConfigCR.Spec.MetadataURL),
		CallbackURL: util.StringPtr(garmServerConfigCR.Spec.CallbackURL),
		WebhookURL:  util.StringPtr(garmServerConfigCR.Spec.WebhookURL),
	})

	log.Info("Updating controller info in garm")
	response, err := client.UpdateController(updateParams)
	if err != nil {
		log.Error(err, "Failed to update controller info")
		return nil, err
	}
	return &response.Payload, nil
}

func (r *GarmServerConfigReconciler) updateGarmServerConfigStatus(ctx context.Context, controllerInfo *params.ControllerInfo, garmServerConfigCR *garmoperatorv1beta1.GarmServerConfig) error {
	log := log.FromContext(ctx)

	if !r.needsStatusUpdate(controllerInfo, garmServerConfigCR) {
		log.Info("GarmServerConfig CR up to date")
		return nil
	}

	log.Info("Updating GarmServerConfig CR")
	garmServerConfigStatus := garmoperatorv1beta1.GarmServerConfigStatus{
		ControllerID:         controllerInfo.ControllerID.String(),
		Hostname:             controllerInfo.Hostname,
		MetadataURL:          controllerInfo.MetadataURL,
		CallbackURL:          controllerInfo.CallbackURL,
		WebhookURL:           controllerInfo.WebhookURL,
		ControllerWebhookURL: controllerInfo.ControllerWebhookURL,
		MinimumJobAgeBackoff: controllerInfo.MinimumJobAgeBackoff,
		Version:              controllerInfo.Version,
	}

	garmServerConfigCR.Status = garmServerConfigStatus
	err := r.Status().Update(ctx, garmServerConfigCR)
	if err != nil {
		log.Error(err, "Failed to update GarmServerConfig CR")
		return err
	}
	return nil
}

func (r *GarmServerConfigReconciler) needsStatusUpdate(controllerInfo *params.ControllerInfo, garmServerConfigCR *garmoperatorv1beta1.GarmServerConfig) bool {
	tempStatus := garmoperatorv1beta1.GarmServerConfigStatus{
		ControllerID:         controllerInfo.ControllerID.String(),
		Hostname:             controllerInfo.Hostname,
		MetadataURL:          controllerInfo.MetadataURL,
		CallbackURL:          controllerInfo.CallbackURL,
		WebhookURL:           controllerInfo.WebhookURL,
		ControllerWebhookURL: controllerInfo.ControllerWebhookURL,
		MinimumJobAgeBackoff: controllerInfo.MinimumJobAgeBackoff,
		Version:              controllerInfo.Version,
	}

	return !reflect.DeepEqual(garmServerConfigCR.Status, tempStatus)
}

func (r *GarmServerConfigReconciler) getControllerInfo(client garmclient.ControllerClient) (params.ControllerInfo, error) {
	controllerInfo, err := client.GetControllerInfo()
	if err == nil {
		return controllerInfo.Payload, nil
	}

	var conflictErr *controller_info.ControllerInfoConflict
	if !errors.As(err, &conflictErr) {
		return params.ControllerInfo{}, err
	}

	if reflect.DeepEqual(conflictErr.Payload, garmapiserverparams.URLsRequired) {
		return params.ControllerInfo{}, nil
	}

	return params.ControllerInfo{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *GarmServerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1beta1.GarmServerConfig{}).
		WithOptions(controller.Options{}).
		Complete(r)
}
