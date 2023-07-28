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
	"time"

	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/key"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/poolutil"
	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/client/pools"
	"github.com/cloudbase/garm/client/repositories"
	"github.com/cloudbase/garm/params"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmClient "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client"

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

// +kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=pools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=pools/status,verbs=get;update;patch

func (r *PoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	poolClient, err := garmClient.NewPoolClient(garmClient.GarmScopeParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	pool := &garmoperatorv1alpha1.Pool{}
	if err := r.Get(ctx, req.NamespacedName, pool); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "cannot fetch Pool")
		return ctrl.Result{}, err
	}

	// handle deletion
	if !pool.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, pool, poolClient)
	}

	// pool status id has been set and id matches existing garm pool, so we know that the pool is persisted in garm and needs to be updated
	if pool.Status.ID != "" && r.garmPoolExists(ctx, pool.Status.ID, poolClient) {
		return r.reconcileUpdate(ctx, pool, poolClient)
	}

	// no pool id yet or the existing pool id is outdated, so pool cr needs by either synced to match garm pool or created
	return r.reconcileCreate(ctx, pool, poolClient)
}

// Todo: Write usefull Events foo
// Todo: Write Error to Status Field
func (r *PoolReconciler) reconcileCreate(ctx context.Context, pool *garmoperatorv1alpha1.Pool, garmClient garmClient.PoolClient) (ctrl.Result, error) {
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

func createPool(ctx context.Context, pool *garmoperatorv1alpha1.Pool, garmClient garmClient.PoolClient) (params.Pool, error) {
	log := log.FromContext(ctx)
	log.Info("creating pool")
	poolResult := params.Pool{}

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
		result, err := garmClient.CreateEnterprisePool(
			enterprises.NewCreateEnterprisePoolParams().
				WithEnterpriseID(id).
				WithBody(
					poolParams))
		if err != nil {
			return params.Pool{}, err
		}

		poolResult = result.Payload
	case garmoperatorv1alpha1.OrganizationScope:
		result, err := garmClient.CreateOrgPool(
			organizations.NewCreateOrgPoolParams().
				WithOrgID(id).
				WithBody(poolParams))
		if err != nil {
			return params.Pool{}, err
		}
		poolResult = result.Payload
	case garmoperatorv1alpha1.RepositoryScope:
		result, err := garmClient.CreateRepoPool(
			repositories.NewCreateRepoPoolParams().
				WithRepoID(id).
				WithBody(poolParams))
		if err != nil {
			return params.Pool{}, err
		}
		poolResult = result.Payload
	default:
		err := fmt.Errorf("no valid scope specified: %s", scope)
		if err != nil {
			return params.Pool{}, err
		}
	}

	return poolResult, nil
}

func (r *PoolReconciler) reconcileUpdate(ctx context.Context, pool *garmoperatorv1alpha1.Pool, garmClient garmClient.PoolClient) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("existing pool found in garm, updating...", "ID", pool.Status.ID)

	result, err := updatePool(ctx, pool, garmClient)
	if err != nil {
		log.Error(err, "error updating pool")
		return r.handleUpdateError(ctx, pool, err)
	}

	return r.handleSuccessfulUpdate(ctx, pool, result.ID)
}

func updatePool(ctx context.Context, pool *garmoperatorv1alpha1.Pool, garmClient garmClient.PoolClient) (params.Pool, error) {
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
	result, err := garmClient.UpdatePool(pools.NewUpdatePoolParams().WithPoolID(id).WithBody(poolParams))
	if err != nil {
		return params.Pool{}, err
	}

	return result.Payload, err
}

func (r *PoolReconciler) reconcileDelete(ctx context.Context, pool *garmoperatorv1alpha1.Pool, garmClient garmClient.PoolClient) (ctrl.Result, error) {
	// pool does not exist in garm database yet as ID in Status is empty, so we can safely delete it
	log := log.FromContext(ctx)
	log.Info("Deleting Pool", "pool", pool.Name)
	if pool.Status.ID == "" && controllerutil.ContainsFinalizer(pool, key.PoolFinalizerName) {
		controllerutil.RemoveFinalizer(pool, key.PoolFinalizerName)
		if err := r.Update(ctx, pool); err != nil {
			return r.handleUpdateError(ctx, pool, err)
		}
		log.Info("Successfully deleted pool", "pool", pool.Name)
		return ctrl.Result{}, nil
	}

	if controllerutil.ContainsFinalizer(pool, key.PoolFinalizerName) && pool.Spec.ForceDeleteRunners {
		pool.Spec.MinIdleRunners = 0
		err := r.Update(ctx, pool)
		if err != nil {
			return r.handleUpdateError(ctx, pool, err)
		}
		log.Info("scaling pool down before deleting")
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Minute}, nil
	}

	// remove finalizer so k8s can delete resource
	controllerutil.RemoveFinalizer(pool, key.PoolFinalizerName)
	if err := r.Update(ctx, pool); err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}
	// TODO: handle different pool types (org, repo, enterprise)
	err := garmClient.DeletePool(
		pools.NewDeletePoolParams().
			WithPoolID(pool.Status.ID),
	)
	if err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	log.Info("Successfully deleted pool", "pool", pool.Name)
	return ctrl.Result{}, nil
}

func (r *PoolReconciler) updatePoolStatus(ctx context.Context, pool *garmoperatorv1alpha1.Pool) error {
	log := log.FromContext(ctx)
	if err := r.Status().Update(ctx, pool); err != nil {
		log.Error(err, "unable to update Pool status")
		return err
	}
	return nil
}

func (r *PoolReconciler) ensureFinalizer(ctx context.Context, pool *garmoperatorv1alpha1.Pool) error {
	if !controllerutil.ContainsFinalizer(pool, key.PoolFinalizerName) {
		controllerutil.AddFinalizer(pool, key.PoolFinalizerName)
		return r.Update(ctx, pool)
	}
	return nil
}

func (r *PoolReconciler) syncPools(ctx context.Context, pool *garmoperatorv1alpha1.Pool, garmClient garmClient.PoolClient) (params.Pool, error) {
	log := log.FromContext(ctx)
	matchingGarmPool, err := r.getExistingGarmPoolBySpecs(ctx, pool, garmClient)
	if err != nil {
		return params.Pool{}, err
	}

	isGarmPoolEmpty := matchingGarmPool.ID == ""

	// we found a garm pool, return it to sync ids
	if !isGarmPoolEmpty {
		log.Info("Found garm pool with matching specs", "garmID", matchingGarmPool.ID)
		return matchingGarmPool, err
	}

	log.Info("Pool with specified specs does not yet exist, creating pool in garm")
	// no matching pool in garm found, just create it
	return createPool(ctx, pool, garmClient)
}

func (r *PoolReconciler) handleUpdateError(ctx context.Context, pool *garmoperatorv1alpha1.Pool, err error) (ctrl.Result, error) {
	pool.Status.Synced = false
	pool.Status.LastSyncTime = metav1.Now()
	pool.Status.LastSyncError = err.Error()

	if updateErr := r.updatePoolStatus(ctx, pool); updateErr != nil {
		return ctrl.Result{}, updateErr
	}
	return ctrl.Result{}, err
}

func (r *PoolReconciler) handleSuccessfulUpdate(ctx context.Context, pool *garmoperatorv1alpha1.Pool, poolID string) (ctrl.Result, error) {
	pool.Status.ID = poolID
	pool.Status.Synced = true
	pool.Status.LastSyncTime = metav1.Now()
	pool.Status.LastSyncError = ""

	if err := r.updatePoolStatus(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *PoolReconciler) getExistingGarmPoolBySpecs(ctx context.Context, pool *garmoperatorv1alpha1.Pool, garmClient garmClient.PoolClient) (params.Pool, error) {
	log := log.FromContext(ctx)

	garmPools, err := garmClient.ListAllPools(pools.NewListPoolsParams())
	if err != nil {
		return params.Pool{}, err
	}
	filteredGarmPools := poolutil.FilterGarmPools(garmPools.Payload,
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
		return params.Pool{}, errors.New("can not create pool, multiple instances matching flavour, image and provider found in garm")
	}

	for _, garmPool := range filteredGarmPools {
		for _, poolCR := range poolList.Items {
			// we found some matching garm pool and its matching already existing poolCR => no need to create a pool with the same specs
			if poolCR.Status.ID == garmPool.ID {
				return params.Pool{}, fmt.Errorf("pool with same specs already exists in garm: GarmPoolID=%s PoolCRD=%s", garmPool.ID, poolCR.Name)
			}
		}
		// found matching garm pool, but no pool CR yet => no need for creating, just sync IDs
		return garmPool, nil
	}
	// no pool in garm found, so we need to create one
	return params.Pool{}, nil
}

func (r *PoolReconciler) garmPoolExists(ctx context.Context, poolID string, garmClient garmClient.PoolClient) bool {
	result, err := garmClient.GetPool(pools.NewGetPoolParams().WithPoolID(poolID))
	if err != nil {
		return false
	}
	return result.Payload.ID != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *PoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Pool{}).
		Complete(r)
}
