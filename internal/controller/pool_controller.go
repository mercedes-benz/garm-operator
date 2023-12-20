// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/cloudbase/garm/client/pools"
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
	"github.com/mercedes-benz/garm-operator/pkg/util/annotations"
	poolUtil "github.com/mercedes-benz/garm-operator/pkg/util/pool"
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

const (
	imageField = "spec.image"
)

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
	if pool.Status.ID != "" && poolUtil.GarmPoolExists(poolClient, pool) {
		return r.reconcileUpdate(ctx, poolClient, pool, instanceClient)
	}

	// no pool id yet or the existing pool id is outdated, so pool cr needs by either synced to match garm pool or created
	return r.reconcileCreate(ctx, poolClient, pool, gitHubScopeRef)
}

func (r *PoolReconciler) reconcileCreate(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, gitHubScopeRef garmoperatorv1alpha1.GitHubScope) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Pool doesn't exist on garm side. Creating new pool in garm")

	// get image cr object by name
	image, err := r.getImage(ctx, pool)
	if err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	// check if there is already a pool with the same spec on garm side
	matchingGarmPool, err := poolUtil.GetGarmPoolBySpecs(ctx, garmClient, pool, image, gitHubScopeRef)
	if err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	if matchingGarmPool != nil {
		log.Info("Found garm pool with matching specs, syncing IDs", "garmID", matchingGarmPool.ID)
		event.Creating(r.Recorder, pool, fmt.Sprintf("found garm pool with matching specs, syncing IDs: %s", matchingGarmPool.ID))
		return r.handleSuccessfulUpdate(ctx, pool, *matchingGarmPool)
	}

	// create new pool in garm
	garmPool, err := poolUtil.CreatePool(ctx, garmClient, pool, image, gitHubScopeRef)
	if err != nil {
		return r.handleUpdateError(ctx, pool, fmt.Errorf("failed creating pool %s: %s", pool.Name, err.Error()))
	}
	log.Info("creating pool in garm succeeded")
	event.Info(r.Recorder, pool, "creating pool in garm succeeded")
	return r.handleSuccessfulUpdate(ctx, pool, garmPool)
}

func (r *PoolReconciler) reconcileUpdate(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, instanceClient garmClient.InstanceClient) (ctrl.Result, error) {
	log := log.FromContext(ctx).
		WithName("reconcileUpdate")
	log.Info("pool on garm side found", "id", pool.Status.ID, "name", pool.Name)

	poolNeedsUpdate, runners, err := r.comparePoolSpecs(ctx, pool, garmClient)
	if err != nil {
		log.Error(err, "error comparing pool specs")
		return r.handleUpdateError(ctx, pool, err)
	}

	idleRunners, err := poolUtil.ExtractIdleRunners(ctx, runners)
	if err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	if !poolNeedsUpdate {
		log.Info("pool CR differs from pool on garm side. Trigger a garm pool update")

		image, err := r.getImage(ctx, pool)
		if err != nil {
			log.Error(err, "error getting image")
			return r.handleUpdateError(ctx, pool, err)
		}

		if err = poolUtil.UpdatePool(ctx, garmClient, pool, image); err != nil {
			log.Error(err, "error updating pool")
			return r.handleUpdateError(ctx, pool, err)
		}
	}

	// update pool idle runners count in status
	pool.Status.IdleRunners = uint(len(idleRunners))
	pool.Status.Runners = uint(len(runners))
	if err := r.updatePoolCRStatus(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}

	// If there are more idle Runners than minIdleRunners are defined in
	// the spec, we force a runner deletion.
	if pool.Status.IdleRunners > pool.Spec.MinIdleRunners {
		log.Info("Scaling pool", "pool", pool.Name)
		event.Scaling(r.Recorder, pool, fmt.Sprintf("scale idle runners down to %d", pool.Spec.MinIdleRunners))
		if err := poolUtil.AlignIdleRunners(ctx, pool, idleRunners, instanceClient); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *PoolReconciler) reconcileDelete(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, instanceClient garmClient.InstanceClient) (ctrl.Result, error) {
	// pool does not exist in garm database yet as ID in Status is empty, so we can safely delete it
	log := log.FromContext(ctx)
	log.Info("Deleting Pool", "pool", pool.Name)
	event.Deleting(r.Recorder, pool, "")

	// this is to make the deletion of a "pending" pool CR possible
	if pool.Status.ID == "" && controllerutil.ContainsFinalizer(pool, key.PoolFinalizerName) {
		controllerutil.RemoveFinalizer(pool, key.PoolFinalizerName)
		if err := r.Update(ctx, pool); err != nil {
			return r.handleUpdateError(ctx, pool, err)
		}

		log.Info("Successfully deleted pool", "pool", pool.Name)
		return ctrl.Result{}, nil
	}

	pool.Spec.MinIdleRunners = 0
	pool.Spec.Enabled = false
	if err := r.Update(ctx, pool); err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	image, err := r.getImage(ctx, pool)
	if err != nil {
		log.Error(err, "error getting image")
		return r.handleUpdateError(ctx, pool, err)
	}

	if err = poolUtil.UpdatePool(ctx, garmClient, pool, image); err != nil {
		log.Error(err, "error updating pool")
		return r.handleUpdateError(ctx, pool, err)
	}

	// get current idle runners
	runners, err := poolUtil.GetAllRunners(ctx, pool, instanceClient)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
	}

	idleRunners, err := poolUtil.ExtractIdleRunners(ctx, runners)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
	}

	// set current idle runners count in status
	pool.Status.IdleRunners = uint(len(idleRunners))
	pool.Status.Runners = uint(len(runners))
	if err := r.updatePoolCRStatus(ctx, pool); err != nil {
		return r.handleUpdateError(ctx, pool, err)
	}

	// scale pool down only needed if spec is smaller than current min idle runners
	// doesn't catch the case where minIdleRunners is increased and to many idle runners exist
	if pool.Status.IdleRunners > pool.Spec.MinIdleRunners {
		log.Info("Scaling pool", "pool", pool.Name)
		event.Scaling(r.Recorder, pool, fmt.Sprintf("scale idle runners down to %d", pool.Spec.MinIdleRunners))
		if err := poolUtil.AlignIdleRunners(ctx, pool, idleRunners, instanceClient); err != nil {
			return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
		}
	}

	// if the pool still contains runners, we need
	if pool.Status.Runners != 0 {
		log.Info("scaling pool down before deleting")
		event.Info(r.Recorder, pool, fmt.Sprintf("pool still contains %d runners. Reconcile again", pool.Status.Runners))
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
	}

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

func (r *PoolReconciler) updatePoolCRStatus(ctx context.Context, pool *garmoperatorv1alpha1.Pool) error {
	log := log.FromContext(ctx)
	if err := r.Status().Update(ctx, pool); err != nil {
		log.Error(err, "unable to update Pool status")
		return err
	}
	return nil
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
	pool.Status.IdleRunners = garmPool.MinIdleRunners
	pool.Status.LastSyncError = ""

	if err := r.updatePoolCRStatus(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *PoolReconciler) ensureFinalizer(ctx context.Context, pool *garmoperatorv1alpha1.Pool) error {
	if !controllerutil.ContainsFinalizer(pool, key.PoolFinalizerName) {
		controllerutil.AddFinalizer(pool, key.PoolFinalizerName)
		return r.Update(ctx, pool)
	}
	return nil
}

func (r *PoolReconciler) comparePoolSpecs(ctx context.Context, pool *garmoperatorv1alpha1.Pool, poolClient garmClient.PoolClient) (bool, []params.Instance, error) {
	log := log.FromContext(ctx).
		WithName("comparePoolSpecs")

	image, err := r.getImage(ctx, pool)
	if err != nil {
		log.Error(err, "error getting image")
		return false, []params.Instance{}, err
	}

	gitHubScopeRef, err := r.fetchGitHubScopeCRD(ctx, pool)
	if err != nil {
		log.Error(err, "error fetching GitHubScopeRef")
		return false, []params.Instance{}, err
	}

	// as there are some "special" tags, which aren't set by the user and aren't part of the pool spec
	// we need to "discover" them and add them to the pool spec before comparing
	poolTags, err := poolUtil.CreateComparableRunnerTags(pool.Spec.Tags, pool.Spec.OSArch, pool.Spec.OSType)
	if err != nil {
		return false, []params.Instance{}, err
	}

	// get the current pool from garm
	garmPool, err := poolClient.GetPool(pools.NewGetPoolParams().WithPoolID(pool.Status.ID))
	if err != nil {
		return false, []params.Instance{}, err
	}

	// sort tags to ensure that the order is always the same
	// and remove the ID field as it is not relevant for comparison
	sort.Slice(garmPool.Payload.Tags, func(i, j int) bool {
		garmPool.Payload.Tags[i].ID = ""
		garmPool.Payload.Tags[j].ID = ""
		return garmPool.Payload.Tags[i].Name < garmPool.Payload.Tags[j].Name
	})

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
		Tags:                   poolTags,
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

	idleInstances, err := poolUtil.ExtractIdleRunners(ctx, garmPool.Payload.Instances)
	if err != nil {
		return false, []params.Instance{}, err
	}

	// empty instances for comparison
	garmPool.Payload.Instances = nil

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

func (r *PoolReconciler) getImage(ctx context.Context, pool *garmoperatorv1alpha1.Pool) (*garmoperatorv1alpha1.Image, error) {
	image := &garmoperatorv1alpha1.Image{}
	if pool.Spec.ImageName != "" {
		if err := r.Get(ctx, types.NamespacedName{Name: pool.Spec.ImageName, Namespace: pool.Namespace}, image); err != nil {
			return nil, err
		}
	}
	return image, nil
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
