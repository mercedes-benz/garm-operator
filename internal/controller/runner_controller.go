// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"reflect"
	"strings"
	"time"

	commonParams "github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/params"
	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/conditions"
	"github.com/mercedes-benz/garm-operator/pkg/config"
	"github.com/mercedes-benz/garm-operator/pkg/filter"
	runnerUtil "github.com/mercedes-benz/garm-operator/pkg/runners"
)

// RunnerReconciler reconciles a Runner object
type RunnerReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	ReconcileChan chan event.GenericEvent
}

//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=runners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=runners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=runners/finalizers,verbs=update

func (r *RunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instanceClient := garmClient.NewInstanceClient()
	return r.reconcileNormal(ctx, req, instanceClient)
}

func (r *RunnerReconciler) reconcileNormal(ctx context.Context, req ctrl.Request, instanceClient garmClient.InstanceClient) (res ctrl.Result, retErr error) {
	log := log.FromContext(ctx)

	// try fetch runner instance in garm db with events coming from reconcile loop events of RunnerCR or from manually enqueued events of garm api.
	garmRunner, err := r.getGarmRunnerInstanceByName(instanceClient, req.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	// only create RunnerCR if it does not yet exist
	runner := &garmoperatorv1beta1.Runner{}
	err = r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: strings.ToLower(req.Name)}, runner)

	switch {
	// found Runner CR and matching garm runner or RunnerCR is already in deleting state, continue with reconcile
	case err == nil && garmRunner != nil || !runner.ObjectMeta.DeletionTimestamp.IsZero():
		log.Info("Found Runner CR and matching garm runner, continue with reconcile", "runner", runner.Name)

	// Found RunnerCR but no matching garm runner, delete the RunnerCR
	case err == nil && garmRunner == nil:
		log.Info("Found RunnerCR for event but no matching garm runner, deleting RunnerCR", "runner", runner.Name)
		if err := r.Delete(ctx, runner); err != nil {
			return ctrl.Result{}, err
		}

	// Did not find RunnerCR and found garm runner, create the RunnerCR
	case apierrors.IsNotFound(err) && garmRunner != nil:
		log.Info("Did not find RunnerCR and found garm runner, creating RunnerCR", "runner", garmRunner.Name)
		return r.createRunnerCR(ctx, garmRunner, req.Namespace)

	// Did not find RunnerCR and no matching garm runner, do nothing
	case apierrors.IsNotFound(err) && garmRunner == nil:
		log.Info("No RunnerCR and no garm runner was found for event", "request", req)
		return ctrl.Result{}, nil

	// Reconcile error
	default:
		log.Error(err, "Error reconciling runner", "request", req)
		return ctrl.Result{}, err
	}

	orig := runner.DeepCopy()

	// Initialize conditions to unknown if not set already
	runner.InitializeConditions()

	// always update the status
	defer func() {
		if !reflect.DeepEqual(runner.Status, orig.Status) {
			log.Info("Update runner status...")
			diff := cmp.Diff(orig.Status, runner.Status)
			log.V(1).Info("Runner status changed", "diff", diff)
			if err := r.Status().Update(ctx, runner); err != nil {
				log.Error(err, "failed to update status", "runner", runner.Name)
				res = ctrl.Result{}
				retErr = err
			}
		}
	}()

	// delete runner in garm db
	if !runner.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instanceClient, runner, garmRunner)
	}

	// sync garm runner status back to RunnerCR
	err = r.updateRunnerStatus(ctx, runner, garmRunner)
	if err != nil {
		log.Error(err, "Failed to update runner status", "runner", runner.Name)
	}

	return ctrl.Result{}, err
}

func (r *RunnerReconciler) createRunnerCR(ctx context.Context, garmRunner *params.Instance, namespace string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Creating RunnerCR", "Runner", garmRunner.Name)

	runnerCR := &garmoperatorv1beta1.Runner{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(garmRunner.Name),
			Namespace: namespace,
		},
		Spec: garmoperatorv1beta1.RunnerSpec{},
	}

	if err := r.Create(ctx, runnerCR); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.ensureFinalizer(ctx, runnerCR); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateRunnerStatus(ctx, runnerCR, garmRunner); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RunnerReconciler) reconcileDelete(ctx context.Context, runnerClient garmClient.InstanceClient, runnerCR *garmoperatorv1beta1.Runner, garmRunner *params.Instance) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	if garmRunner != nil {
		log.Info("Deleting Runner in Garm", "Runner Name", garmRunner.Name)
		err := runnerClient.DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName(garmRunner.Name))
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	if controllerutil.ContainsFinalizer(runnerCR, key.RunnerFinalizerName) {
		controllerutil.RemoveFinalizer(runnerCR, key.RunnerFinalizerName)
		if err := r.Update(ctx, runnerCR); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *RunnerReconciler) getGarmRunnerInstanceByName(client garmClient.InstanceClient, name string) (*params.Instance, error) {
	allInstances, err := client.ListInstances(instances.NewListInstancesParams().WithDefaults())
	if err != nil {
		return nil, err
	}

	filteredInstances := filter.Match(allInstances.Payload, runnerUtil.MatchesName(name))
	if len(filteredInstances) == 0 {
		return nil, nil
	}

	return &filteredInstances[0], nil
}

func (r *RunnerReconciler) ensureFinalizer(ctx context.Context, runner *garmoperatorv1beta1.Runner) error {
	if !controllerutil.ContainsFinalizer(runner, key.RunnerFinalizerName) {
		controllerutil.AddFinalizer(runner, key.RunnerFinalizerName)
		return r.Update(ctx, runner)
	}
	return nil
}

func (r *RunnerReconciler) updateRunnerStatus(ctx context.Context, runner *garmoperatorv1beta1.Runner, garmRunner *params.Instance) error {
	if garmRunner == nil {
		return nil
	}

	poolName := garmRunner.PoolID
	pools := &garmoperatorv1beta1.PoolList{}
	if err := r.List(ctx, pools); err == nil {
		filteredPools := filter.Match(pools.Items, garmoperatorv1beta1.MatchesID(garmRunner.PoolID))

		if len(filteredPools) > 0 {
			poolName = filteredPools[0].Name
		}
	}

	runner.Status.ID = garmRunner.ID
	runner.Status.ProviderID = garmRunner.ProviderID
	runner.Status.AgentID = garmRunner.AgentID
	runner.Status.Name = garmRunner.Name
	runner.Status.OSType = garmRunner.OSType
	runner.Status.OSName = garmRunner.OSName
	runner.Status.OSVersion = garmRunner.OSVersion
	runner.Status.OSArch = garmRunner.OSArch
	runner.Status.Addresses = garmRunner.Addresses
	runner.Status.Status = garmRunner.RunnerStatus
	runner.Status.InstanceStatus = garmRunner.Status
	runner.Status.PoolID = poolName
	runner.Status.ProviderFault = string(garmRunner.ProviderFault)
	runner.Status.GitHubRunnerGroup = garmRunner.GitHubRunnerGroup

	if runner.Status.InstanceStatus == commonParams.InstancePendingCreate ||
		runner.Status.InstanceStatus == commonParams.InstanceCreating ||
		runner.Status.Status == params.RunnerInstalling ||
		runner.Status.Status == params.RunnerPending {
		conditions.MarkFalse(runner, conditions.ReadyCondition, conditions.RunnerNotReadyReason, conditions.RunnerNotReadyYetMsg)
	}

	if runner.Status.InstanceStatus == commonParams.InstanceError || runner.Status.Status == params.RunnerFailed {
		conditions.MarkFalse(runner, conditions.ReadyCondition, conditions.RunnerErrorReason, conditions.RunnerProvisioningFailedMsg)
	}

	if runner.Status.InstanceStatus == commonParams.InstanceRunning && runner.Status.Status == params.RunnerIdle {
		conditions.MarkTrue(runner, conditions.ReadyCondition, conditions.RunnerReadyReason, conditions.RunnerIdleAndRunningMsg)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1beta1.Runner{}).
		WithOptions(options).
		WatchesRawSource(source.Channel(r.ReconcileChan, &handler.EnqueueRequestForObject{})).
		Complete(r)
}

func (r *RunnerReconciler) PollRunnerInstances(ctx context.Context) {
	log := log.FromContext(ctx)
	ticker := time.NewTicker(config.Config.Operator.SyncRunnersInterval)
	for {
		select {
		case <-ctx.Done():
			log.Info("Closing event channel for runners...")
			close(r.ReconcileChan)
			return
		case <-ticker.C:
			instanceClient := garmClient.NewInstanceClient()
			err := r.EnqueueRunnerInstances(ctx, instanceClient)
			if err != nil {
				log.Error(err, "Failed polling runner instances")
			}
		}
	}
}

func (r *RunnerReconciler) EnqueueRunnerInstances(ctx context.Context, instanceClient garmClient.InstanceClient) error {
	pools, err := r.fetchPools(ctx)
	if err != nil {
		return err
	}

	// fetching runners by pools to ensure only runners belonging to pools in same namespace are being shown
	garmRunnerInstances, err := r.fetchRunnerInstancesByNamespacedPools(instanceClient, pools)
	if err != nil {
		return err
	}

	runnerCRList := &garmoperatorv1beta1.RunnerList{}
	err = r.List(ctx, runnerCRList)
	if err != nil {
		return err
	}

	var runnerCRNameList []string
	for _, runner := range runnerCRList.Items {
		runnerCRNameList = append(runnerCRNameList, runner.Name)
	}

	var runnerInstanceNameList []string
	for _, runner := range garmRunnerInstances {
		runnerInstanceNameList = append(runnerInstanceNameList, strings.ToLower(runner.Name))
	}

	runnersToDelete := getRunnerDiff(runnerCRNameList, runnerInstanceNameList)

	runnerInstanceNameList = append(runnerInstanceNameList, runnersToDelete...)

	r.enqeueRunnerEvents(runnerInstanceNameList)
	return nil
}

func (r *RunnerReconciler) enqeueRunnerEvents(runners []string) {
	for _, runner := range runners {
		runnerObj := garmoperatorv1beta1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      strings.ToLower(runner),
				Namespace: config.Config.Operator.WatchNamespace,
			},
		}

		e := event.GenericEvent{
			Object: &runnerObj,
		}

		r.ReconcileChan <- e
	}
}

func (r *RunnerReconciler) fetchPools(ctx context.Context) (*garmoperatorv1beta1.PoolList, error) {
	pools := &garmoperatorv1beta1.PoolList{}
	err := r.List(ctx, pools)
	if err != nil {
		return nil, err
	}
	return pools, nil
}

func (r *RunnerReconciler) fetchRunnerInstancesByNamespacedPools(instanceClient garmClient.InstanceClient, pools *garmoperatorv1beta1.PoolList) (params.Instances, error) {
	garmRunnerInstances := params.Instances{}
	for _, p := range pools.Items {
		if p.Status.ID == "" {
			continue
		}
		poolRunners, err := instanceClient.ListPoolInstances(instances.NewListPoolInstancesParams().WithPoolID(p.Status.ID))
		if err != nil {
			return nil, err
		}
		garmRunnerInstances = append(garmRunnerInstances, poolRunners.Payload...)
	}
	return garmRunnerInstances, nil
}

func getRunnerDiff(runnerCRs, garmRunners []string) []string {
	cache := make(map[string]struct{})
	var diff []string

	for _, runner := range garmRunners {
		cache[runner] = struct{}{}
	}

	for _, runnerCR := range runnerCRs {
		if _, found := cache[runnerCR]; !found {
			diff = append(diff, runnerCR)
		}
	}
	return diff
}
