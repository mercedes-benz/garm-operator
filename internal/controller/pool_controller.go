/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	garmClient "github.com/cloudbase/garm/cmd/garm-cli/client"
	"github.com/cloudbase/garm/params"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/v1alpha1"
)

// PoolReconciler reconciles a Pool object
type PoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=pools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=pools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=pools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pool object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile

const finalizer = "pools.garm-operator.mercedes-benz.com/finalizer"

func (r *PoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	pool := &garmoperatorv1alpha1.Pool{}
	if err := r.Get(ctx, req.NamespacedName, pool); err != nil {
		log.Error(err, "cannot fetch Pool")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Todo: Init Garm Client
	var newClient garmClient.Client

	if !pool.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, newClient, pool)
	}

	return r.reconcileNormal(ctx, newClient, pool)
}

func (r *PoolReconciler) reconcileNormal(ctx context.Context, garmClient garmClient.Client, pool *garmoperatorv1alpha1.Pool) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// add finalizer if not present
	if !controllerutil.ContainsFinalizer(pool, finalizer) {
		controllerutil.AddFinalizer(pool, finalizer)
		if err := r.Update(ctx, pool); err != nil {
			return ctrl.Result{}, err
		}
	}

	// create pool
	poolResult, err := r.createPool(*pool, garmClient)
	if err != nil {
		log.Error(err, "failed creating pool", "pool", pool.Spec)
		return ctrl.Result{}, err
	}

	// update status
	pool.Status.ID = poolResult.ID
	if err := r.Status().Update(ctx, pool); err != nil {
		log.Error(err, "unable to update Pool status")
		return ctrl.Result{}, err
	}

	log.Info("created pool", poolResult)
	return ctrl.Result{}, nil
}

func (r *PoolReconciler) createPool(pool garmoperatorv1alpha1.Pool, garmClient garmClient.Client) (params.Pool, error) {
	var result params.Pool
	var err error

	id := pool.Spec.GitHubScopeID
	scope := pool.Spec.GitHubScope

	poolParams := params.CreatePoolParams{
		RunnerPrefix: params.RunnerPrefix{
			Prefix: pool.Spec.RunnerPrefix,
		},
		ProviderName:           pool.Spec.ProviderName,
		MaxRunners:             pool.Spec.MaxRunners,
		MinIdleRunners:         pool.Spec.MinIdleRunners,
		Image:                  pool.Spec.Image,
		Flavor:                 pool.Spec.Flavor,
		OSType:                 pool.Spec.OSType,
		OSArch:                 pool.Spec.OSArch,
		Tags:                   pool.Spec.Tags,
		Enabled:                pool.Spec.Enabled,
		RunnerBootstrapTimeout: pool.Spec.RunnerBootstrapTimeout,
		ExtraSpecs:             pool.Spec.ExtraSpecs,
		GitHubRunnerGroup:      pool.Spec.GitHubRunnerGroup,
	}

	if scope == garmoperatorv1alpha1.EnterpriseScope {
		result, err = garmClient.CreateEnterprisePool(id, poolParams)
	}

	if scope == garmoperatorv1alpha1.OrganizationScope {
		result, err = garmClient.CreateOrgPool(id, poolParams)
	}

	if scope == garmoperatorv1alpha1.RepositoryScope {
		result, err = garmClient.CreateRepoPool(id, poolParams)
	}

	return result, err
}

func (r *PoolReconciler) reconcileDelete(ctx context.Context, garmClient garmClient.Client, pool *garmoperatorv1alpha1.Pool) (ctrl.Result, error) {
	// pool does not exist in garm database yet as ID in Status is empty, so we can safely delete it
	if pool.Status.ID == "" {
		return ctrl.Result{}, nil
	}

	if controllerutil.ContainsFinalizer(pool, finalizer) {
		// only delete pool if no runners are left
		runners, err := garmClient.ListPoolInstances(pool.Status.ID)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Todo: what to return if there are still runners left? => reque after x minutes and try again?
		if len(runners) > 0 {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// remove finalizer so k8s can delete resource
	controllerutil.RemoveFinalizer(pool, finalizer)
	if err := r.Update(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Pool{}).
		Complete(r)
}
