// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"fmt"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/life4/genesis/slices"
	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
)

// RunnerReconciler reconciles a Runner object
type RunnerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	BaseURL  string
	Username string
	Password string
}

//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=runners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=runners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=runners/finalizers,verbs=update

func (r *RunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling runners...", "Request", req)

	instanceClient := garmClient.NewInstanceClient()
	err := instanceClient.Login(garmClient.GarmScopeParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
		// Debug:    true,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcile(ctx, req, instanceClient)
}

func (r *RunnerReconciler) reconcile(ctx context.Context, req ctrl.Request, instanceClient garmClient.InstanceClient) (ctrl.Result, error) {
	// try fetch runner instance in garm db with events coming from reconcile loop events of RunnerCR or from manually enqueued events of garm api.
	garmRunner, err := r.getGarmRunnerInstance(instanceClient, req.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	// only create RunnerCR if it does not yet exist
	runner := &garmoperatorv1alpha1.Runner{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: strings.ToLower(req.Name)}, runner); err != nil {
		return r.handleCreateRunnerCR(ctx, req, err, garmRunner)
	}

	// delete runner on garm side and remove finalizer if deleted
	if !runner.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instanceClient, runner, garmRunner)
	}

	// sync garm runner status back to RunnerCR
	return r.updateRunnerStatus(ctx, runner, garmRunner)
}

func (r *RunnerReconciler) handleCreateRunnerCR(ctx context.Context, req ctrl.Request, fetchErr error, garmRunner *params.Instance) (ctrl.Result, error) {
	if apierrors.IsNotFound(fetchErr) && garmRunner != nil {
		return r.createRunnerCR(ctx, garmRunner, req.Namespace)
	}
	return ctrl.Result{}, fetchErr
}

func (r *RunnerReconciler) createRunnerCR(ctx context.Context, garmRunner *params.Instance, namespace string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Creating Runner", "Runner", garmRunner.Name)

	runnerObj := &garmoperatorv1alpha1.Runner{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(garmRunner.Name),
			Namespace: namespace,
		},
		Spec: garmoperatorv1alpha1.RunnerSpec{},
	}
	err := r.Create(ctx, runnerObj)
	if err != nil {
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

func (r *RunnerReconciler) reconcileDelete(ctx context.Context, runnerClient garmClient.InstanceClient, runner *garmoperatorv1alpha1.Runner, garmRunner *params.Instance) (ctrl.Result, error) {
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

func (r *RunnerReconciler) cleanupRunnerCR(ctx context.Context, runner *garmoperatorv1alpha1.Runner) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Removing finalizer and cleaning up runner CR in cluster", "Runner CR", runner.Name)
	err := r.Delete(ctx, runner)
	if err != nil {
		return ctrl.Result{}, err
	}
	if controllerutil.ContainsFinalizer(runner, key.RunnerFinalizerName) {
		controllerutil.RemoveFinalizer(runner, key.RunnerFinalizerName)
		if err := r.Update(ctx, runner); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *RunnerReconciler) getGarmRunnerInstance(client garmClient.InstanceClient, name string) (*params.Instance, error) {
	// Problem: req.Name is not always name of garm instance like road-runner-k8s-EyCKa8uPE1to, but name of runner CR from cache which is lowercase road-runner-k8s-eycka8upe1to => GetInstanceByName can match runner in garm with name of CR
	garmRunner, err := client.GetInstanceByName(instances.NewGetInstanceParams().WithInstanceName(name))
	if err != nil && garmClient.IsNotFoundError(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &garmRunner.Payload, err
}

func (r *RunnerReconciler) ensureFinalizer(ctx context.Context, runner *garmoperatorv1alpha1.Runner) error {
	if !controllerutil.ContainsFinalizer(runner, key.RunnerFinalizerName) {
		controllerutil.AddFinalizer(runner, key.RunnerFinalizerName)
		return r.Update(ctx, runner)
	}
	return nil
}

func (r *RunnerReconciler) updateRunnerStatus(ctx context.Context, runner *garmoperatorv1alpha1.Runner, garmRunner *params.Instance) (ctrl.Result, error) {
	if garmRunner == nil {
		return ctrl.Result{}, nil
	}

	log := log.FromContext(ctx)
	log.Info("Update runner status...")

	poolName := garmRunner.PoolID
	pools := &garmoperatorv1alpha1.PoolList{}
	err := r.List(ctx, pools)
	if err == nil {
		pools.FilterByFields(garmoperatorv1alpha1.MatchesID(garmRunner.PoolID))

		if len(pools.Items) > 0 {
			poolName = pools.Items[0].Name
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
	runner.Status.ProviderFault = garmRunner.ProviderFault
	runner.Status.GitHubRunnerGroup = garmRunner.GitHubRunnerGroup

	if err := r.Status().Update(ctx, runner); err != nil {
		log.Error(err, "unable to update Runner status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerReconciler) SetupWithManager(mgr ctrl.Manager, eventChan chan event.GenericEvent) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Runner{}).
		Build(r)
	if err != nil {
		return err
	}

	return c.Watch(&source.Channel{Source: eventChan}, &handler.EnqueueRequestForObject{})
}

func (r *RunnerReconciler) PollRunnerInstances(ctx context.Context, eventChan chan event.GenericEvent) {
	log := log.FromContext(ctx)
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			log.Info("Closing event channel for runners...")
			close(eventChan)
			return
		case _ = <-ticker.C:
			log.Info("Polling Runners...")
			err := r.EnqueueRunnerInstances(ctx, eventChan)
			if err != nil {
				log.Error(err, "Failed polling runner instances")
			}
		}
	}
}

func (r *RunnerReconciler) EnqueueRunnerInstances(ctx context.Context, eventChan chan event.GenericEvent) error {
	garmRunnerInstances := params.Instances{}
	pools := &garmoperatorv1alpha1.PoolList{}

	err := r.List(ctx, pools)
	if err != nil {
		return err
	}

	instanceClient := garmClient.NewInstanceClient()
	err = instanceClient.Login(garmClient.GarmScopeParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
	})
	if err != nil {
		return err
	}

	if len(pools.Items) < 1 {
		return nil
	}
	namespace := pools.Items[0].Namespace
	// fetching runners by pools to ensure only runners belonging to pools in same namespace are being shown
	for _, p := range pools.Items {
		if p.Status.ID == "" {
			continue
		}
		poolRunners, err := instanceClient.ListPoolInstances(instances.NewListPoolInstancesParams().WithPoolID(p.Status.ID))
		if err != nil {
			return err
		}
		garmRunnerInstances = append(garmRunnerInstances, poolRunners.Payload...)
	}

	// compare garmdb with runner cr => delete runnerCRs not present in garmdb
	runnerCRList := &garmoperatorv1alpha1.RunnerList{}
	err = r.List(ctx, runnerCRList)
	if err != nil {
		return err
	}

	runnerCRNameList := slices.Map(runnerCRList.Items, func(runner garmoperatorv1alpha1.Runner) string {
		return runner.Name
	})

	runnerInstanceNameList := slices.Map(garmRunnerInstances, func(runner params.Instance) string {
		return strings.ToLower(runner.Name)
	})

	runnersToDelete := getRunnerDiff(runnerCRNameList, runnerInstanceNameList)
	fmt.Println(runnersToDelete)

	for _, runnerName := range runnersToDelete {
		runner := &garmoperatorv1alpha1.Runner{}
		err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: runnerName}, runner)
		if err != nil {
			return err
		}

		if runner.DeletionTimestamp.IsZero() {
			err = r.Delete(ctx, runner)
			if err != nil {
				return err
			}
		}

		if controllerutil.ContainsFinalizer(runner, key.RunnerFinalizerName) {
			controllerutil.RemoveFinalizer(runner, key.RunnerFinalizerName)
			if err := r.Update(ctx, runner); err != nil {
				return err
			}
		}
	}

	for _, runner := range garmRunnerInstances {
		runnerObj := garmoperatorv1alpha1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      strings.ToLower(runner.Name),
				Namespace: namespace,
			},
		}

		e := event.GenericEvent{
			Object: &runnerObj,
		}

		eventChan <- e
	}
	return nil
}

func getRunnerDiff(runnerCRs, garmRunners []string) []string {
	var diff []string

	for _, runnerCR := range runnerCRs {
		if !slices.Contains(garmRunners, runnerCR) {
			diff = append(diff, runnerCR)
		}
	}
	return diff
}
