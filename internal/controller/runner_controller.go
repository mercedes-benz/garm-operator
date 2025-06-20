// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/params"
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
	return r.reconcile(ctx, req, instanceClient)
}

func (r *RunnerReconciler) reconcile(ctx context.Context, req ctrl.Request, instanceClient garmClient.InstanceClient) (ctrl.Result, error) {
	// try fetch runner instance in garm db with events coming from reconcile loop events of RunnerCR or from manually enqueued events of garm api.
	garmRunner, err := r.getGarmRunnerInstanceByName(instanceClient, req.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	// only create RunnerCR if it does not yet exist
	runner := &garmoperatorv1beta1.Runner{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: strings.ToLower(req.Name)}, runner); err != nil {
		return r.handleCreateRunnerCR(ctx, req, err, garmRunner)
	}

	// delete runner in garm db
	if !runner.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instanceClient, garmRunner)
	}

	// sync garm runner status back to RunnerCR
	return r.updateRunnerStatus(ctx, runner, garmRunner)
}

func (r *RunnerReconciler) handleCreateRunnerCR(ctx context.Context, req ctrl.Request, fetchErr error, garmRunner *params.Instance) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	if apierrors.IsNotFound(fetchErr) && garmRunner != nil {
		return r.createRunnerCR(ctx, garmRunner, req.Namespace)
	}

	if apierrors.IsNotFound(fetchErr) {
		log.Info("object was not found")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, fetchErr
}

func (r *RunnerReconciler) createRunnerCR(ctx context.Context, garmRunner *params.Instance, namespace string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Creating Runner", "Runner", garmRunner.Name)

	runnerObj := &garmoperatorv1beta1.Runner{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(garmRunner.Name),
			Namespace: namespace,
		},
		Spec: garmoperatorv1beta1.RunnerSpec{},
	}

	if err := r.Create(ctx, runnerObj); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.ensureFinalizer(ctx, runnerObj); err != nil {
		return ctrl.Result{}, err
	}

	if _, err := r.updateRunnerStatus(ctx, runnerObj, garmRunner); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RunnerReconciler) reconcileDelete(ctx context.Context, runnerClient garmClient.InstanceClient, garmRunner *params.Instance) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	if garmRunner != nil {
		log.Info("Deleting Runner in Garm", "Runner Name", garmRunner.Name)
		err := runnerClient.DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName(garmRunner.Name))
		if err != nil {
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

func (r *RunnerReconciler) updateRunnerStatus(ctx context.Context, runner *garmoperatorv1beta1.Runner, garmRunner *params.Instance) (ctrl.Result, error) {
	if garmRunner == nil {
		return ctrl.Result{}, nil
	}

	if r.runnerSpecsEqual(*runner, garmRunner) {
		return ctrl.Result{}, nil
	}

	log := log.FromContext(ctx)
	log.Info("Update runner status...")

	poolName := garmRunner.PoolID
	pools := &garmoperatorv1beta1.PoolList{}
	err := r.List(ctx, pools)
	if err == nil {
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
	runner.Status.Status = garmRunner.Status
	runner.Status.InstanceStatus = garmRunner.RunnerStatus
	runner.Status.PoolID = poolName
	runner.Status.ProviderFault = string(garmRunner.ProviderFault)
	runner.Status.GitHubRunnerGroup = garmRunner.GitHubRunnerGroup

	if err := r.Status().Update(ctx, runner); err != nil {
		log.Error(err, "unable to update Runner status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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

	// compares garm db with RunnerCRs and deletes RunnerCRs not present in garm db
	err = r.cleanUpNotMatchingRunnerCRs(ctx, garmRunnerInstances)
	if err != nil {
		return err
	}

	// triggers controller to reconcile based on instances in garm db
	r.enqeueRunnerEvents(garmRunnerInstances)
	return nil
}

func (r *RunnerReconciler) enqeueRunnerEvents(garmRunnerInstances params.Instances) {
	for _, runner := range garmRunnerInstances {
		runnerObj := garmoperatorv1beta1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      strings.ToLower(runner.Name),
				Namespace: config.Config.Operator.WatchNamespace,
			},
		}

		e := event.GenericEvent{
			Object: &runnerObj,
		}

		r.ReconcileChan <- e
	}
}

func (r *RunnerReconciler) cleanUpNotMatchingRunnerCRs(ctx context.Context, garmRunnerInstances params.Instances) error {
	runnerCRList := &garmoperatorv1beta1.RunnerList{}
	err := r.List(ctx, runnerCRList)
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
	log.Log.V(1).Info("Deleting runners: ", "Runners", runnersToDelete)

	for _, runnerName := range runnersToDelete {
		runner := &garmoperatorv1beta1.Runner{}
		err := r.Get(ctx, types.NamespacedName{Namespace: config.Config.Operator.WatchNamespace, Name: runnerName}, runner)
		if err != nil {
			return err
		}

		if runner.DeletionTimestamp.IsZero() {
			err = r.Delete(ctx, runner)
			if err != nil {
				return err
			}
		}

		// getting RunnerCR from cache again before removing finalizer, as in the meantime object has changed
		err = r.Get(ctx, types.NamespacedName{Namespace: config.Config.Operator.WatchNamespace, Name: runnerName}, runner)
		if err != nil {
			return err
		}

		if controllerutil.ContainsFinalizer(runner, key.RunnerFinalizerName) {
			controllerutil.RemoveFinalizer(runner, key.RunnerFinalizerName)
			if err := r.Update(ctx, runner); err != nil {
				return err
			}
		}
	}
	return nil
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

func (r *RunnerReconciler) runnerSpecsEqual(runner garmoperatorv1beta1.Runner, garmRunner *params.Instance) bool {
	runner.Status.PoolID = ""
	tmpRunnerStatus := garmoperatorv1beta1.RunnerStatus{
		ID:                garmRunner.ID,
		ProviderID:        garmRunner.ProviderID,
		AgentID:           garmRunner.AgentID,
		Name:              garmRunner.Name,
		OSType:            garmRunner.OSType,
		OSName:            garmRunner.OSName,
		OSVersion:         garmRunner.OSVersion,
		OSArch:            garmRunner.OSArch,
		Addresses:         garmRunner.Addresses,
		Status:            garmRunner.Status,
		InstanceStatus:    garmRunner.RunnerStatus,
		ProviderFault:     string(garmRunner.ProviderFault),
		GitHubRunnerGroup: garmRunner.GitHubRunnerGroup,
		PoolID:            "",
	}

	return reflect.DeepEqual(runner.Status, tmpRunnerStatus)
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
