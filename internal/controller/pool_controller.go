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
	"encoding/json"
	"errors"
	"fmt"
	newGarmClient "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/garmclient"
	poolutil "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/pool"
	garmClient "github.com/cloudbase/garm/cmd/garm-cli/client"
	"github.com/cloudbase/garm/params"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
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

	BaseURL  string
	Username string
	Password string
}

const finalizer = "pools.garm-operator.mercedes-benz.com/finalizer"

// +kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=pools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=pools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=pools/finalizers,verbs=update

func (r *PoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	newClient, err := newGarmClient.NewGarmClient(newGarmClient.GarmClientParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	pool := &garmoperatorv1alpha1.Pool{}
	if err := r.Get(ctx, req.NamespacedName, pool); err != nil {
		log.Error(err, "cannot fetch Pool")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// handle deletion
	if !pool.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, *newClient, pool)
	}

	// pool status id has been set, so we know that the pool is persisted in garm and needs to be updated
	if pool.Status.ID != "" {
		return r.reconcileUpdate(ctx, *newClient, pool)
	}

	// no pool id yet, so pool needs to be created
	return r.reconcileCreate(ctx, *newClient, pool)
}

// Todo: Write usefull Events foo
// Todo: Write Error to Status Field
func (r *PoolReconciler) reconcileCreate(ctx context.Context, garmClient garmClient.Client, pool *garmoperatorv1alpha1.Pool) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("trying to create pool...")

	if err := r.ensureFinalizer(ctx, pool); err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	result, err := r.syncPools(ctx, pool, garmClient)
	if err != nil {
		log.Error(err, "failed creating pool", "error", err)
		return r.handleUpdateError(ctx, pool, err)
	}
	log.Info("successfully created pool")
	return r.handleSuccessfulUpdate(ctx, pool, result.ID)
}

func (r *PoolReconciler) createPool(ctx context.Context, pool garmoperatorv1alpha1.Pool, garmClient garmClient.Client) (params.Pool, error) {
	log := log.FromContext(ctx)
	log.Info("creating pool")
	result := params.Pool{}
	var err error

	id := pool.Spec.GitHubScopeID
	scope := pool.Spec.GitHubScope

	extraSpecs := json.RawMessage([]byte{})
	if pool.Spec.ExtraSpecs != "" {
		_ = json.Unmarshal([]byte(pool.Spec.ExtraSpecs), &extraSpecs)
	}

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
		ExtraSpecs:             extraSpecs,
		GitHubRunnerGroup:      pool.Spec.GitHubRunnerGroup,
	}

	switch scope {
	case garmoperatorv1alpha1.EnterpriseScope:
		result, err = garmClient.CreateEnterprisePool(id, poolParams)
	case garmoperatorv1alpha1.OrganizationScope:
		result, err = garmClient.CreateOrgPool(id, poolParams)
	case garmoperatorv1alpha1.RepositoryScope:
		result, err = garmClient.CreateRepoPool(id, poolParams)
	default:
		err = fmt.Errorf("no valid scope specified: %s", scope)
	}

	if reflect.DeepEqual(result, params.Pool{}) {
		return result, errors.New("could not create pool in garm - check garm logs")
	}
	return result, err
}

func (r *PoolReconciler) reconcileUpdate(ctx context.Context, garmClient garmClient.Client, pool *garmoperatorv1alpha1.Pool) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("trying to update pool...")

	result, err := r.updatePool(*pool, garmClient)
	if err != nil {
		log.Error(err, "error updating pool")
		return r.handleUpdateError(ctx, pool, err)
	}

	return r.handleSuccessfulUpdate(ctx, pool, result.ID)
}

func (r *PoolReconciler) updatePool(pool garmoperatorv1alpha1.Pool, garmClient garmClient.Client) (params.Pool, error) {
	result := params.Pool{}
	var err error
	id := pool.Status.ID

	extraSpecs := json.RawMessage([]byte{})
	if pool.Spec.ExtraSpecs != "" {
		_ = json.Unmarshal([]byte(pool.Spec.ExtraSpecs), &extraSpecs)
	}

	poolParams := params.UpdatePoolParams{
		RunnerPrefix: params.RunnerPrefix{
			Prefix: pool.Spec.RunnerPrefix,
		},
		MaxRunners:             &pool.Spec.MaxRunners,
		MinIdleRunners:         &pool.Spec.MinIdleRunners,
		Image:                  pool.Spec.Image,
		Flavor:                 pool.Spec.Flavor,
		OSType:                 pool.Spec.OSType,
		OSArch:                 pool.Spec.OSArch,
		Tags:                   pool.Spec.Tags,
		Enabled:                &pool.Spec.Enabled,
		RunnerBootstrapTimeout: &pool.Spec.RunnerBootstrapTimeout,
		ExtraSpecs:             extraSpecs,
		GitHubRunnerGroup:      &pool.Spec.GitHubRunnerGroup,
	}
	result, err = garmClient.UpdatePoolByID(id, poolParams)

	return result, err
}

func (r *PoolReconciler) reconcileDelete(ctx context.Context, garmClient garmClient.Client, pool *garmoperatorv1alpha1.Pool) (ctrl.Result, error) {
	// pool does not exist in garm database yet as ID in Status is empty, so we can safely delete it
	log := log.FromContext(ctx)
	log.Info("Deleting Pool", "pool", pool.Name)
	if pool.Status.ID == "" && controllerutil.ContainsFinalizer(pool, finalizer) {
		controllerutil.RemoveFinalizer(pool, finalizer)
		if err := r.Update(ctx, pool); err != nil {
			return r.handleUpdateError(ctx, pool, err)
		}
		log.Info("Successfully deleted pool", "pool", pool.Name)
		return ctrl.Result{}, nil
	}

	if controllerutil.ContainsFinalizer(pool, finalizer) {

		if pool.Spec.ForceDeleteRunners {
			pool.Spec.MinIdleRunners = 0
			err := r.Update(ctx, pool)
			if err != nil {
				return r.handleUpdateError(ctx, pool, err)
			}
			return ctrl.Result{Requeue: true}, nil
		}

		// only delete pool if no runners are left
		runners, err := garmClient.ListPoolInstances(pool.Status.ID)
		if err != nil {
			return r.handleUpdateError(ctx, pool, err)
		}

		// Todo: what to return if there are still runners left? => reque after x minutes and try again?
		if len(runners) > 0 {
			return ctrl.Result{Requeue: true}, nil
		}

	}

	// remove finalizer so k8s can delete resource
	controllerutil.RemoveFinalizer(pool, finalizer)
	if err := r.Update(ctx, pool); err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}
	err := garmClient.DeletePoolByID(pool.Status.ID)
	if err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	log.Info("Successfully deleted pool", "pool", pool.Name)
	return ctrl.Result{}, nil
}

func (r *PoolReconciler) updatePoolStatus(ctx context.Context, pool *garmoperatorv1alpha1.Pool, newStatus *garmoperatorv1alpha1.PoolStatus) error {
	log := log.FromContext(ctx)
	pool.Status = *newStatus
	if err := r.Status().Update(ctx, pool); err != nil {
		log.Error(err, "unable to update Pool status")
		return err
	}
	return nil
}

func (r *PoolReconciler) ensureFinalizer(ctx context.Context, pool *garmoperatorv1alpha1.Pool) error {
	if !controllerutil.ContainsFinalizer(pool, finalizer) {
		controllerutil.AddFinalizer(pool, finalizer)
		return r.Update(ctx, pool)
	}
	return nil
}

func (r *PoolReconciler) syncPools(ctx context.Context, pool *garmoperatorv1alpha1.Pool, garmClient garmClient.Client) (params.Pool, error) {
	log := log.FromContext(ctx)

	garmPools, err := garmClient.ListAllPools()
	if err != nil {
		return params.Pool{}, err
	}
	filteredGarmPools := poolutil.FilterGarmPools(garmPools,
		poolutil.MatchesImageFlavorAndProvider(pool.Spec.Image, pool.Spec.Flavor, pool.Spec.ProviderName),
		poolutil.MatchesGitHubScope(pool.Spec.GitHubScope, pool.Spec.GitHubScopeID),
	)

	listOpts := &client.ListOptions{
		Namespace: pool.Namespace,
	}

	poolList := &garmoperatorv1alpha1.PoolList{}
	if err := r.List(ctx, poolList, listOpts); err != nil {
		log.Error(err, "cannot fetch Pools", "error", err)
		return params.Pool{}, err
	}

	poolList.FilterByFields(
		garmoperatorv1alpha1.MatchesFlavour(pool.Spec.Flavor),
		garmoperatorv1alpha1.MatchesImage(pool.Spec.Image),
		garmoperatorv1alpha1.MatchesProvider(pool.Spec.ProviderName),
		garmoperatorv1alpha1.MatchesGitHubScope(pool.Spec.GitHubScope, pool.Spec.GitHubScopeID),
	)

	if len(filteredGarmPools) > 1 {
		r.handleUpdateError(ctx, pool, errors.New("can not create pool, multiple instances matching flavour, image and provider found in garm"))
	}

	for _, garmPool := range filteredGarmPools {
		for _, poolCR := range poolList.Items {
			// we found a matching garm pool and its matching already existing poolCR => no need to create a pool with the same specs
			if poolCR.Status.ID == garmPool.ID {
				return params.Pool{}, fmt.Errorf("pool with same specs already exists in garm: GarmPoolID=%s PoolCRD=%s", garmPool.ID, poolCR.Name)
			}
		}
		// found matching garm pool, but no pool CR yet => no need for creating, just sync IDs
		return garmPool, nil
	}

	// no matching pool in garm found, just create it
	return r.createPool(ctx, *pool, garmClient)
}

func (r *PoolReconciler) handleUpdateError(ctx context.Context, pool *garmoperatorv1alpha1.Pool, err error) (ctrl.Result, error) {
	status := pool.Status
	status.Synced = false
	status.LastSyncError = err.Error()

	if updateErr := r.updatePoolStatus(ctx, pool, &status); updateErr != nil {
		return ctrl.Result{}, updateErr
	}
	return ctrl.Result{}, err
}

func (r *PoolReconciler) handleSuccessfulUpdate(ctx context.Context, pool *garmoperatorv1alpha1.Pool, poolID string) (ctrl.Result, error) {
	status := pool.Status
	status.ID = poolID
	status.Synced = true
	status.LastSyncTime = metav1.Now()
	status.LastSyncError = ""

	if err := r.updatePoolStatus(ctx, pool, &status); err != nil {
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
