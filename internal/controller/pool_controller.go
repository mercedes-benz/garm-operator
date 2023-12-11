// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	garmProviderParams "github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm-provider-common/util"
	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/client/pools"
	"github.com/cloudbase/garm/client/repositories"
	"github.com/cloudbase/garm/params"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/event"
	"github.com/mercedes-benz/garm-operator/pkg/filter"
	poolfilter "github.com/mercedes-benz/garm-operator/pkg/filter/pool"
	"github.com/mercedes-benz/garm-operator/pkg/util/annotations"
)

// PoolReconciler reconciles a Pool object
type PoolReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	BaseURL  string
	Username string
	Password string
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=pools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=images,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=pools/status,verbs=get;update;patch

func (r *PoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	pool := &garmoperatorv1alpha1.Pool{}
	if err := r.Get(ctx, req.NamespacedName, pool); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "cannot fetch Pool")
		return r.handleUpdateError(ctx, pool, err)
	}

	// Ignore objects that are paused
	if annotations.IsPaused(pool) {
		log.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	poolClient := garmClient.NewPoolClient()
	err := poolClient.Login(garmClient.GarmScopeParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
	})
	if err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	instanceClient := garmClient.NewInstanceClient()
	err = instanceClient.Login(garmClient.GarmScopeParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
	})
	if err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	// handle deletion
	if !pool.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, poolClient, pool, instanceClient)
	}

	return r.reconcileNormal(ctx, poolClient, pool, instanceClient)
}

func (r *PoolReconciler) reconcileNormal(ctx context.Context, poolClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, instanceClient garmClient.InstanceClient) (ctrl.Result, error) {
	log := log.FromContext(ctx).
		WithName("reconcileNormal")

	gitHubScopeRef, err := r.fetchGitHubScopeCRD(ctx, pool)
	if err != nil {
		msg := fmt.Sprintf("Error: %s, referenced GitHubScopeRef %s/%s not found", err.Error(), pool.Spec.GitHubScopeRef.Kind, pool.Spec.GitHubScopeRef.Name)
		log.Error(err, msg)
		return r.handleUpdateError(ctx, pool, errors.New(msg))
	}

	if gitHubScopeRef.GetID() == "" {
		return r.handleUpdateError(ctx, pool, fmt.Errorf("referenced GitHubScopeRef %s/%s not ready yet", pool.Spec.GitHubScopeRef.Kind, pool.Spec.GitHubScopeRef.Name))
	}

	if err := r.ensureFinalizer(ctx, pool); err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	// pool status id has been set and id matches existing garm pool, so we know that the pool is persisted in garm and needs to be updated
	if pool.Status.ID != "" && garmPoolExists(poolClient, pool) {
		return r.reconcileUpdate(ctx, poolClient, pool, instanceClient)
	}

	// no pool id yet or the existing pool id is outdated, so pool cr needs by either synced to match garm pool or created
	return r.reconcileCreate(ctx, poolClient, pool, gitHubScopeRef)
}

func (r *PoolReconciler) reconcileCreate(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, gitHubScopeRef garmoperatorv1alpha1.GitHubScope) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("status.ID is empty and pool doesn't exist on garm side. Creating new pool in garm")

	garmPool, err := r.syncPools(ctx, garmClient, pool, gitHubScopeRef)
	if err != nil {
		return r.handleUpdateError(ctx, pool, fmt.Errorf("failed creating pool %s: %s", pool.Name, err.Error()))
	}
	log.Info("creating pool in garm succeeded")
	event.Info(r.Recorder, pool, "creating pool in garm succeeded")
	return r.handleSuccessfulUpdate(ctx, pool, garmPool)
}

func createPool(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, image *garmoperatorv1alpha1.Image, gitHubScopeRef garmoperatorv1alpha1.GitHubScope) (params.Pool, error) {
	log := log.FromContext(ctx)
	log.Info("creating pool")
	poolResult := params.Pool{}

	id := gitHubScopeRef.GetID()
	scope, err := garmoperatorv1alpha1.ToGitHubScopeKind(gitHubScopeRef.GetKind())
	if err != nil {
		return poolResult, err
	}

	extraSpecs := json.RawMessage([]byte{})
	if pool.Spec.ExtraSpecs != "" {
		err := json.Unmarshal([]byte(pool.Spec.ExtraSpecs), &extraSpecs)
		if err != nil {
			return poolResult, err
		}
	}

	poolParams := params.CreatePoolParams{
		RunnerPrefix: params.RunnerPrefix{
			Prefix: pool.Spec.RunnerPrefix,
		},
		ProviderName:           pool.Spec.ProviderName,
		MaxRunners:             pool.Spec.MaxRunners,
		MinIdleRunners:         pool.Spec.MinIdleRunners,
		Image:                  image.Spec.Tag,
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
				WithBody(poolParams))
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
		return params.Pool{}, err
	}

	return poolResult, nil
}

func (r *PoolReconciler) getImage(ctx context.Context, pool *garmoperatorv1alpha1.Pool) (*garmoperatorv1alpha1.Image, error) {
	image := &garmoperatorv1alpha1.Image{}
	if pool.Spec.ImageName != "" {
		if err := r.Get(ctx, types.NamespacedName{Name: pool.Spec.ImageName, Namespace: pool.Namespace}, image); err != nil {
			return nil, err
		}
	}
	return image, nil
}

func (r *PoolReconciler) reconcileUpdate(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, instanceClient garmClient.InstanceClient) (ctrl.Result, error) {
	log := log.FromContext(ctx).
		WithName("reconcileUpdate")
	log.Info("pool on garm side found", "id", pool.Status.ID, "name", pool.Name)

	poolNeedsUpdate, idleRunners, err := r.comparePoolSpecs(ctx, pool, garmClient)
	if err != nil {
		log.Error(err, "error comparing pool specs")
		return r.handleUpdateError(ctx, pool, err)
	}

	if !poolNeedsUpdate {
		log.Info("pool CR differs from pool on garm side. Trigger a garm pool update")

		err = r.updateGarmPool(ctx, garmClient, pool)
		if err != nil {
			log.Error(err, "error updating pool")
			return r.handleUpdateError(ctx, pool, err)
		}
	}

	// update pool idle runners count in status
	pool.Status.MinIdleRunners = uint(len(idleRunners))
	if err := r.updatePoolCRStatus(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}

	// If there are more idle Runners than minIdleRunners are defined in
	// the spec, we force a runner deletion.
	if pool.Status.MinIdleRunners > pool.Spec.MinIdleRunners {
		if err := r.alignIdleRunners(ctx, pool, idleRunners, instanceClient); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// func (r *PoolReconciler) updatePool(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, image *garmoperatorv1alpha1.Image, instanceClient garmClient.InstanceClient) (params.Pool, error) {
func (r *PoolReconciler) updateGarmPool(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool) error {
	log := log.FromContext(ctx)

	image, err := r.getImage(ctx, pool)
	if err != nil {
		log.Error(err, "error getting image")
		return err
	}

	poolParams := params.UpdatePoolParams{
		RunnerPrefix: params.RunnerPrefix{
			Prefix: pool.Spec.RunnerPrefix,
		},
		MaxRunners:             &pool.Spec.MaxRunners,
		MinIdleRunners:         &pool.Spec.MinIdleRunners,
		Image:                  image.Spec.Tag,
		Flavor:                 pool.Spec.Flavor,
		OSType:                 pool.Spec.OSType,
		OSArch:                 pool.Spec.OSArch,
		Tags:                   pool.Spec.Tags,
		Enabled:                &pool.Spec.Enabled,
		RunnerBootstrapTimeout: &pool.Spec.RunnerBootstrapTimeout,
		ExtraSpecs:             json.RawMessage([]byte(pool.Spec.ExtraSpecs)),
		GitHubRunnerGroup:      &pool.Spec.GitHubRunnerGroup,
	}
	_, err = garmClient.UpdatePool(pools.NewUpdatePoolParams().WithPoolID(pool.Status.ID).WithBody(poolParams))
	if err != nil {
		return err
	}

	return nil
}

func (r *PoolReconciler) reconcileDelete(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, instanceClient garmClient.InstanceClient) (ctrl.Result, error) {
	// pool does not exist in garm database yet as ID in Status is empty, so we can safely delete it
	log := log.FromContext(ctx)
	log.Info("Deleting Pool", "pool", pool.Name)
	event.Deleting(r.Recorder, pool, "")

	if pool.Status.ID == "" && controllerutil.ContainsFinalizer(pool, key.PoolFinalizerName) {
		controllerutil.RemoveFinalizer(pool, key.PoolFinalizerName)
		if err := r.Update(ctx, pool); err != nil {
			return r.handleUpdateError(ctx, pool, err)
		}

		log.Info("Successfully deleted pool", "pool", pool.Name)
		return ctrl.Result{}, nil
	}

	if controllerutil.ContainsFinalizer(pool, key.PoolFinalizerName) && pool.Spec.MinIdleRunners != 0 {
		pool.Spec.MinIdleRunners = 0
		pool.Spec.Enabled = false

		err := r.updateGarmPool(ctx, garmClient, pool)
		if err != nil {
			log.Error(err, "error updating pool")
			return r.handleUpdateError(ctx, pool, err)
		}

		if err := r.Update(ctx, pool); err != nil {
			return r.handleUpdateError(ctx, pool, err)
		}
		log.Info("scaling pool down before deleting")
		event.Deleting(r.Recorder, pool, "scaling down runners in pool before deleting")
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, nil
	}

	// get current idle runners
	idleRunners, err := r.getIdleRunners(ctx, pool, instanceClient)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
	}

	// set current idle runners count in status
	pool.Status.MinIdleRunners = uint(len(idleRunners))

	// scale pool down only needed if spec is smaller than current min idle runners
	// doesn't catch the case where minIdleRunners is increased and to many idle runners exist
	if pool.Spec.MinIdleRunners < pool.Status.MinIdleRunners {
		if err := r.alignIdleRunners(ctx, pool, idleRunners, instanceClient); err != nil {
			return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
		}
	}

	// scale pool runners to 0

	err = garmClient.DeletePool(
		pools.NewDeletePoolParams().
			WithPoolID(pool.Status.ID),
	)
	if err != nil {
		return r.handleUpdateError(ctx, pool, fmt.Errorf("error deleting pool %s: %w", pool.Name, err))
	}

	// remove finalizer so k8s can delete resource
	controllerutil.RemoveFinalizer(pool, key.PoolFinalizerName)
	if err := r.Update(ctx, pool); err != nil {
		return r.handleUpdateError(ctx, pool, fmt.Errorf("error deleting pool %s: %w", pool.Name, err))
	}

	log.Info("Successfully deleted pool", "pool", pool.Name)
	return ctrl.Result{}, nil
}

func (r *PoolReconciler) getIdleRunners(ctx context.Context, pool *garmoperatorv1alpha1.Pool, instanceClient garmClient.InstanceClient) ([]params.Instance, error) {
	log := log.FromContext(ctx)
	log.Info("discover idle runners", "pool", pool.Name)

	runners, err := instanceClient.ListPoolInstances(
		instances.NewListPoolInstancesParams().WithPoolID(pool.Status.ID))
	if err != nil {
		return nil, err
	}

	return extractIdleRunners(ctx, runners.GetPayload())
}

// alignIdleRunners scales down the pool to the desired state
// of minIdleRunners. It will delete runners in a deletable state
func (r *PoolReconciler) alignIdleRunners(ctx context.Context, pool *garmoperatorv1alpha1.Pool, idleRunners []params.Instance, instanceClient garmClient.InstanceClient) error {
	log := log.FromContext(ctx)
	log.Info("Scaling pool", "pool", pool.Name)
	event.Scaling(r.Recorder, pool, fmt.Sprintf("scale idle runners down to %d", pool.Spec.MinIdleRunners))

	/*

		alignIdleRunners

			poolMin 20 - 20 runners in a deletable state
				scale down to 10
				10 runners to delete

			poolMin 10 - 10 runners in a deletable state
				scale down to 0
				10 - 0 runners to delete

			poolMin 20 - 8 runners in a deletable state
				scale down to 10
				8 - 10 = -2 runners to delete

			poolMin 10 - 10 runners in a deletable state
				scale down to 0
				10 runners to delete

	*/

	// calculate how many runners need to be deleted
	var removableRunnersCount int
	// if there are more runners than minIdleRunners, delete the difference
	if len(idleRunners)-int(pool.Spec.MinIdleRunners) > 0 {
		removableRunnersCount = len(idleRunners) - int(pool.Spec.MinIdleRunners)
	} else {
		removableRunnersCount = len(idleRunners)
	}

	// this is where the status.minIdleRunners comparison needs to be done
	// if real state is larger than desired state - scale down runners
	for i, runner := range idleRunners {
		// do not delete more runners than minIdleRunners
		if i == removableRunnersCount {
			break
		}
		log.Info("remove runner", "runner", runner.Name, "state", runner.Status, "runner state", runner.RunnerStatus)
		err := instanceClient.DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName(runner.Name))
		if err != nil {
			log.Error(err, "unable to delete runner", "runner", runner.Name)
		}
		if err != nil {
			return err
		}
	}
	log.Info("Successfully scaled pool down", "pool", pool.Name)
	return nil
}

func (r *PoolReconciler) updatePoolCRStatus(ctx context.Context, pool *garmoperatorv1alpha1.Pool) error {
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

func (r *PoolReconciler) syncPools(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, gitHubScopeRef garmoperatorv1alpha1.GitHubScope) (params.Pool, error) {
	log := log.FromContext(ctx)

	// get image cr object by name
	image, err := r.getImage(ctx, pool)
	if err != nil {
		return params.Pool{}, err
	}

	matchingGarmPool, err := r.getExistingGarmPoolBySpecs(ctx, garmClient, pool, image, gitHubScopeRef)
	if err != nil {
		return params.Pool{}, err
	}

	isGarmPoolEmpty := matchingGarmPool.ID == ""

	// we found a garm pool, return it to sync ids
	if !isGarmPoolEmpty {
		log.Info("Found garm pool with matching specs, syncing IDs", "garmID", matchingGarmPool.ID)
		event.Creating(r.Recorder, pool, fmt.Sprintf("found garm pool with matching specs, syncing IDs: %s", matchingGarmPool.ID))
		return matchingGarmPool, err
	}

	return createPool(ctx, garmClient, pool, image, gitHubScopeRef)
}

func (r *PoolReconciler) handleUpdateError(ctx context.Context, pool *garmoperatorv1alpha1.Pool, err error) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Error(err, "error")
	event.Error(r.Recorder, pool, err.Error())

	pool.Status.LastSyncError = err.Error()

	if updateErr := r.updatePoolCRStatus(ctx, pool); updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{}, err
}

func (r *PoolReconciler) handleSuccessfulUpdate(ctx context.Context, pool *garmoperatorv1alpha1.Pool, garmPool params.Pool) (ctrl.Result, error) {
	pool.Status.ID = garmPool.ID
	pool.Status.MinIdleRunners = garmPool.MinIdleRunners
	pool.Status.LastSyncError = ""

	if err := r.updatePoolCRStatus(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *PoolReconciler) getExistingGarmPoolBySpecs(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, image *garmoperatorv1alpha1.Image, gitHubScopeRef garmoperatorv1alpha1.GitHubScope) (params.Pool, error) {
	log := log.FromContext(ctx)
	log.Info("Getting existing garm pools by pool.spec")

	id := gitHubScopeRef.GetID()
	scope, err := garmoperatorv1alpha1.ToGitHubScopeKind(gitHubScopeRef.GetKind())
	if err != nil {
		return params.Pool{}, err
	}

	garmPools, err := garmClient.ListAllPools(pools.NewListPoolsParams())
	if err != nil {
		return params.Pool{}, err
	}
	filteredGarmPools := filter.Match(garmPools.Payload,
		poolfilter.MatchesImage(image.Spec.Tag),
		poolfilter.MatchesFlavor(pool.Spec.Flavor),
		poolfilter.MatchesProvider(pool.Spec.ProviderName),
		poolfilter.MatchesGitHubScope(scope, id),
	)

	if len(filteredGarmPools) > 1 {
		return params.Pool{}, errors.New("can not create pool, multiple instances matching flavour, image and provider found in garm")
	}

	// sync
	if len(filteredGarmPools) == 1 {
		return filteredGarmPools[0], nil
	}

	// create
	return params.Pool{}, nil
}

func garmPoolExists(garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool) bool {
	result, err := garmClient.GetPool(pools.NewGetPoolParams().WithPoolID(pool.Status.ID))
	if err != nil {
		return false
	}
	return result.Payload.ID != ""
}

func (r *PoolReconciler) comparePoolSpecs(ctx context.Context, pool *garmoperatorv1alpha1.Pool, poolClient garmClient.PoolClient) (bool, []params.Instance, error) {
	log := log.FromContext(ctx).
		WithName("comparePoolSpecs")

	image, err := r.getImage(ctx, pool)
	if err != nil {
		log.Error(err, "error getting image")
		return false, []params.Instance{}, err
	}

	// get the current pool from garm
	garmPool, err := poolClient.GetPool(pools.NewGetPoolParams().WithPoolID(pool.Status.ID))
	if err != nil {
		return false, []params.Instance{}, err
	}

	// if pool.Spec.RunnerPrefix is empty, set it to garms default prefix
	if pool.Spec.RunnerPrefix == "" {
		pool.Spec.RunnerPrefix = params.DefaultRunnerPrefix
	}

	gitHubScopeRef, err := r.fetchGitHubScopeCRD(ctx, pool)
	if err != nil {
		return false, []params.Instance{}, err
	}

	// github automatically adds the "self-hosted" tag as well as the OS type (linux, windows, etc)
	// and architecture (arm, x64, etc) to all self hosted runners. When a workflow job comes in, we try
	// to find a pool based on the labels that are set in the workflow. If we don't explicitly define these
	// default tags for each pool, and the user targets these labels, we won't be able to match any pools.
	// The downside is that all pools with the same OS and arch will have these default labels. Users should
	// set distinct and unique labels on each pool, and explicitly target those labels, or risk assigning
	// the job to the wrong worker type.
	ghArch, err := util.ResolveToGithubArch(string(garmPool.Payload.OSArch))
	if err != nil {
		return false, []params.Instance{}, err
	}
	ghOSType, err := util.ResolveToGithubTag(garmPool.Payload.OSType)
	if err != nil {
		return false, []params.Instance{}, err
	}

	labels := []string{
		"self-hosted",
		ghArch,
		ghOSType,
	}

	tags := []params.Tag{}
	for _, tag := range pool.Spec.Tags {
		tags = append(tags, params.Tag{
			Name: tag,
		})
	}

	for _, tag := range labels {
		tags = append(tags, params.Tag{
			Name: tag,
		})
	}

	tmpGarmPool := params.Pool{
		RunnerPrefix: params.RunnerPrefix{
			Prefix: pool.Spec.RunnerPrefix,
		},
		MaxRunners:             pool.Spec.MaxRunners,
		MinIdleRunners:         pool.Spec.MinIdleRunners,
		Image:                  image.Spec.Tag,
		Flavor:                 pool.Spec.Flavor,
		OSType:                 pool.Spec.OSType,
		OSArch:                 pool.Spec.OSArch,
		Tags:                   tags,
		Enabled:                pool.Spec.Enabled,
		RunnerBootstrapTimeout: pool.Spec.RunnerBootstrapTimeout,
		ExtraSpecs:             json.RawMessage([]byte(pool.Spec.ExtraSpecs)),
		GitHubRunnerGroup:      pool.Spec.GitHubRunnerGroup,
		ID:                     pool.Status.ID,
		ProviderName:           pool.Spec.ProviderName,
	}

	switch gitHubScopeRef.GetKind() {
	case string(garmoperatorv1alpha1.EnterpriseScope):
		tmpGarmPool.EnterpriseID = gitHubScopeRef.GetID()
		tmpGarmPool.EnterpriseName = gitHubScopeRef.GetName()
	case string(garmoperatorv1alpha1.OrganizationScope):
		tmpGarmPool.OrgID = gitHubScopeRef.GetID()
		tmpGarmPool.OrgName = gitHubScopeRef.GetName()
	case string(garmoperatorv1alpha1.RepositoryScope):
		tmpGarmPool.RepoID = gitHubScopeRef.GetID()
		tmpGarmPool.RepoName = gitHubScopeRef.GetName()
	}

	idleInstances, err := extractIdleRunners(ctx, garmPool.Payload.Instances)
	if err != nil {
		return false, []params.Instance{}, err
	}

	// empty instances for comparison
	garmPool.Payload.Instances = nil

	log.V(1).Info("Garm pool specification of existing pool", "garmPool", garmPool.Payload)
	log.V(1).Info("Garm pool specification of reconciled pool CR", "poolCR", tmpGarmPool)

	return reflect.DeepEqual(tmpGarmPool, garmPool.Payload), idleInstances, nil
}

func (r *PoolReconciler) fetchGitHubScopeCRD(ctx context.Context, pool *garmoperatorv1alpha1.Pool) (garmoperatorv1alpha1.GitHubScope, error) {
	gitHubScopeNamespacedName := types.NamespacedName{
		Namespace: pool.Namespace,
		Name:      pool.Spec.GitHubScopeRef.Name,
	}

	var gitHubScope client.Object

	switch pool.Spec.GitHubScopeRef.Kind {
	case string(garmoperatorv1alpha1.EnterpriseScope):
		gitHubScope = &garmoperatorv1alpha1.Enterprise{}
		if err := r.Get(ctx, gitHubScopeNamespacedName, gitHubScope); err != nil {
			return nil, err
		}

	case string(garmoperatorv1alpha1.OrganizationScope):
		gitHubScope = &garmoperatorv1alpha1.Organization{}
		if err := r.Get(ctx, gitHubScopeNamespacedName, gitHubScope); err != nil {
			return nil, err
		}

	case string(garmoperatorv1alpha1.RepositoryScope):
		gitHubScope = &garmoperatorv1alpha1.Repository{}
		if err := r.Get(ctx, gitHubScopeNamespacedName, gitHubScope); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported GitHubScopeRef kind: %s", pool.Spec.GitHubScopeRef.Kind)
	}

	return gitHubScope.(garmoperatorv1alpha1.GitHubScope), nil
}

const (
	imageField = "spec.image"
)

// SetupWithManager sets up the controller with the Manager.
func (r *PoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// setup index for image
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &garmoperatorv1alpha1.Pool{}, imageField, func(rawObj client.Object) []string {
		pool := rawObj.(*garmoperatorv1alpha1.Pool)
		if pool.Spec.ImageName == "" {
			return nil
		}
		return []string{pool.Spec.ImageName}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Pool{}).
		Watches(
			&garmoperatorv1alpha1.Image{},
			handler.EnqueueRequestsFromMapFunc(r.findPoolsForImage),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *PoolReconciler) findPoolsForImage(ctx context.Context, obj client.Object) []reconcile.Request {
	image, ok := obj.(*garmoperatorv1alpha1.Image)
	if !ok {
		return nil
	}

	var pools garmoperatorv1alpha1.PoolList
	if err := r.List(ctx, &pools); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, pool := range pools.Items {
		if pool.Spec.ImageName == image.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: pool.Namespace,
					Name:      pool.Name,
				},
			})
		}
	}

	return requests
}

func extractIdleRunners(ctx context.Context, instances []params.Instance) ([]params.Instance, error) {
	log := log.FromContext(ctx)

	// create a list of "deletable runners"
	idleRunners := []params.Instance{}

	// filter runners that are in state that allows deletion
	if len(instances) > 0 {
		for _, runner := range instances {
			switch runner.Status {
			case garmProviderParams.InstanceRunning, garmProviderParams.InstanceError:
				idleRunners = append(idleRunners, runner)
			default:
				log.V(1).Info("Runner is in state that does not allow deletion", "runner", runner.Name, "state", runner.Status)
			}
		}
	}
	return idleRunners, nil
}
