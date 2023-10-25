// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/params"
	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
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

	if err = c.Watch(&source.Channel{Source: eventChan}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	return nil
}

func (r *RunnerReconciler) PollRunnerInstances(ctx context.Context, eventChan chan event.GenericEvent) {
	log := log.FromContext(ctx)
	ticker := time.NewTicker(5 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				log.Info("Polling Runners...")
				_ = r.EnqueueRunnerInstances(ctx, eventChan)
			}
		}
	}()
}

func (r *RunnerReconciler) EnqueueRunnerInstances(ctx context.Context, eventChan chan event.GenericEvent) error {
	allRunners := params.Instances{}
	pools := &garmoperatorv1alpha1.PoolList{}

	err := r.List(ctx, pools)
	if err != nil {
		return err
	}

	runnerClient, err := garmClient.NewInstanceClient(garmClient.GarmScopeParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
	})

	var namespace string
	if len(pools.Items) > 0 {
		namespace = pools.Items[0].Namespace
	}

	for _, p := range pools.Items {
		poolRunners, err := runnerClient.ListPoolInstances(instances.NewListPoolInstancesParams().WithPoolID(p.Status.ID))
		if err != nil {
			return err
		}
		allRunners = append(allRunners, poolRunners.Payload...)
	}

	for _, runner := range allRunners {
		// TODO: check if runner already exists as CR in namespace before enqueuing new event
		runnerObj := garmoperatorv1alpha1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      runner.Name,
				Namespace: namespace,
			},
			Spec: garmoperatorv1alpha1.RunnerSpec{},
			Status: garmoperatorv1alpha1.RunnerStatus{
				ID:             runner.ID,
				ProviderID:     runner.ProviderID,
				AgentID:        runner.AgentID,
				Name:           runner.Name,
				OSType:         runner.OSType,
				OSName:         runner.OSName,
				OSVersion:      runner.OSVersion,
				OSArch:         runner.OSArch,
				Addresses:      runner.Addresses,
				Status:         runner.Status,
				InstanceStatus: runner.RunnerStatus,
				PoolID:         runner.PoolID,
				ProviderFault:  runner.ProviderFault,
				//StatusMessages:    runner.StatusMessages,
				//UpdatedAt:         runner.UpdatedAt,
				GitHubRunnerGroup: runner.GitHubRunnerGroup,
			},
		}

		e := event.GenericEvent{
			Object: &runnerObj,
		}

		eventChan <- e
	}
	return nil
}
