// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"reflect"

	garmController "github.com/cloudbase/garm/client/controller"
	"github.com/cloudbase/garm/params"
	"github.com/mercedes-benz/garm-operator/pkg/annotations"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/config"
	"github.com/mercedes-benz/garm-operator/pkg/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
)

// GarmServerConfigReconciler reconciles a GarmServerConfig object
type GarmServerConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=garmserverconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=garmserverconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=garmserverconfigs/finalizers,verbs=update

func (r *GarmServerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	controllerClient := garmClient.NewControllerClient()
	return r.reconcile(ctx, req, controllerClient)
}

func (r *GarmServerConfigReconciler) reconcile(ctx context.Context, req ctrl.Request, controllerClient garmClient.ControllerClient) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling GarmServerConfig")

	controllerInfo, err := controllerClient.GetControllerInfo()
	if err != nil {
		log.Error(err, "Failed to get controller info")
		return ctrl.Result{}, err
	}

	// initially create GarmServerConfig CR reflecting values from garm-server controller info
	garmServerConfig := &garmoperatorv1alpha1.GarmServerConfig{}
	if err := r.Get(ctx, req.NamespacedName, garmServerConfig); err != nil {
		return r.handleCreateGarmServerConfigCR(ctx, req, err, &controllerInfo.Payload)
	}

	// Ignore objects that are paused
	if annotations.IsPaused(garmServerConfig) {
		log.Info("Reconciliation is paused for GarmServerConfig: %s", garmServerConfig.Name)
		return ctrl.Result{}, nil
	}

	// sync applied spec with controller info in garm
	newControllerInfo, err := r.updateControllerInfo(ctx, controllerClient, garmServerConfig, &controllerInfo.Payload)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update CR with new state from garm
	if err := r.updateGarmServerConfigStatus(ctx, newControllerInfo, garmServerConfig); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GarmServerConfigReconciler) handleCreateGarmServerConfigCR(ctx context.Context, req ctrl.Request, fetchErr error, controllerInfo *params.ControllerInfo) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	if apierrors.IsNotFound(fetchErr) {
		log.Info("GarmServerConfig was not found")
		if err := r.createGarmServerConfigCR(ctx, controllerInfo, req.Name, req.Namespace); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *GarmServerConfigReconciler) createGarmServerConfigCR(ctx context.Context, controllerInfo *params.ControllerInfo, name, namespace string) error {
	log := log.FromContext(ctx)
	log.Info("Creating GarmServerConfig CR")

	garmServerConfig := &garmoperatorv1alpha1.GarmServerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: garmoperatorv1alpha1.GarmServerConfigSpec{
			MetadataURL: controllerInfo.MetadataURL,
			CallbackURL: controllerInfo.CallbackURL,
			WebhookURL:  controllerInfo.WebhookURL,
		},
	}

	err := r.Create(ctx, garmServerConfig)
	if err != nil {
		log.Error(err, "Failed to create GarmServerConfig CR")
		return err
	}
	return nil
}

func (r *GarmServerConfigReconciler) updateControllerInfo(ctx context.Context, client garmClient.ControllerClient, garmServerConfigCR *garmoperatorv1alpha1.GarmServerConfig, controllerInfo *params.ControllerInfo) (*params.ControllerInfo, error) {
	log := log.FromContext(ctx)

	if garmServerConfigCR.Spec.MetadataURL == controllerInfo.MetadataURL &&
		garmServerConfigCR.Spec.CallbackURL == controllerInfo.CallbackURL &&
		garmServerConfigCR.Spec.WebhookURL == controllerInfo.WebhookURL {
		log.Info("Controller info is up to date")
		return controllerInfo, nil
	}

	params := garmController.NewUpdateControllerParams().WithBody(params.UpdateControllerParams{
		MetadataURL: util.StringPtr(garmServerConfigCR.Spec.MetadataURL),
		CallbackURL: util.StringPtr(garmServerConfigCR.Spec.CallbackURL),
		WebhookURL:  util.StringPtr(garmServerConfigCR.Spec.WebhookURL),
	})

	log.Info("Updating controller info in garm")
	response, err := client.UpdateController(params)
	if err != nil {
		log.Error(err, "Failed to update controller info")
		return nil, err
	}
	return &response.Payload, nil
}

func (r *GarmServerConfigReconciler) updateGarmServerConfigStatus(ctx context.Context, controllerInfo *params.ControllerInfo, garmServerConfigCR *garmoperatorv1alpha1.GarmServerConfig) error {
	log := log.FromContext(ctx)

	if !r.needsStatusUpdate(controllerInfo, garmServerConfigCR) {
		log.Info("GarmServerConfig CR up to date")
		return nil
	}

	log.Info("Updating GarmServerConfig CR")
	garmServerConfigStatus := garmoperatorv1alpha1.GarmServerConfigStatus{
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

func (r *GarmServerConfigReconciler) needsStatusUpdate(controllerInfo *params.ControllerInfo, garmServerConfigCR *garmoperatorv1alpha1.GarmServerConfig) bool {
	tempStatus := garmoperatorv1alpha1.GarmServerConfigStatus{
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

func (r *GarmServerConfigReconciler) ensureFinalizer(ctx context.Context, garmServerConfig *garmoperatorv1alpha1.GarmServerConfig) error {
	if !controllerutil.ContainsFinalizer(garmServerConfig, key.RunnerFinalizerName) {
		controllerutil.AddFinalizer(garmServerConfig, key.RunnerFinalizerName)
		return r.Update(ctx, garmServerConfig)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GarmServerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.GarmServerConfig{}).
		WithOptions(controller.Options{}).
		Build(r)
	if err != nil {
		return err
	}

	eventChan := make(chan event.GenericEvent)
	go func() {
		eventChan <- event.GenericEvent{
			Object: &garmoperatorv1alpha1.GarmServerConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "garm-server-config",
					Namespace: config.Config.Operator.WatchNamespace,
				},
			},
		}
		close(eventChan)
	}()

	return c.Watch(&source.Channel{Source: eventChan}, &handler.EnqueueRequestForObject{})
}
