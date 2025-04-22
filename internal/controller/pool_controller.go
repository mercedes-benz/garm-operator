// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/client/pools"
	"github.com/cloudbase/garm/params"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	"github.com/mercedes-benz/garm-operator/pkg/annotations"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/conditions"
	"github.com/mercedes-benz/garm-operator/pkg/config"
	"github.com/mercedes-benz/garm-operator/pkg/event"
	"github.com/mercedes-benz/garm-operator/pkg/finalizers"
	poolUtil "github.com/mercedes-benz/garm-operator/pkg/pools"
	runnerUtil "github.com/mercedes-benz/garm-operator/pkg/runners"
	"github.com/mercedes-benz/garm-operator/pkg/tags"
)

// PoolReconciler reconciles a Pool object
type PoolReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

const (
	imageField = "spec.image"
)

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=pools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=images,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=pools/status,verbs=get;update;patch

func (r *PoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	log := log.FromContext(ctx)

	pool := &garmoperatorv1beta1.Pool{}
	if err := r.Get(ctx, req.NamespacedName, pool); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "cannot fetch Pool")
		event.Error(r.Recorder, pool, err.Error())
		return ctrl.Result{}, err
	}

	orig := pool.DeepCopy()

	// Ignore objects that are paused
	if annotations.IsPaused(pool) {
		log.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	// ensure the finalizer
	if finalizerAdded, err := finalizers.EnsureFinalizer(ctx, r.Client, pool, key.PoolFinalizerName); err != nil || finalizerAdded {
		return ctrl.Result{}, err
	}

	poolClient := garmClient.NewPoolClient()

	instanceClient := garmClient.NewInstanceClient()

	// Initialize conditions to unknown if not set already
	pool.InitializeConditions()

	// always update the status
	defer func() {
		if !reflect.DeepEqual(pool.Status, orig.Status) {
			if err := r.Status().Update(ctx, pool); err != nil {
				log.Error(err, "failed to update status")
				res = ctrl.Result{Requeue: true}
				retErr = err
			}
		}
	}()

	// handle deletion
	if !pool.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, poolClient, pool, instanceClient)
	}

	return r.reconcileNormal(ctx, poolClient, pool, instanceClient)
}

func (r *PoolReconciler) reconcileNormal(ctx context.Context, poolClient garmClient.PoolClient, pool *garmoperatorv1beta1.Pool, instanceClient garmClient.InstanceClient) (ctrl.Result, error) {
	gitHubScopeRef, err := r.fetchGitHubScopeCRD(ctx, pool)
	if err != nil {
		r.errorLog(ctx, pool, err)
		conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		conditions.MarkFalse(pool, conditions.ScopeReference, conditions.FetchingScopeRefFailedReason, err.Error())
		return ctrl.Result{}, err
	}

	if gitHubScopeRef.GetID() == "" {
		err := fmt.Errorf("referenced GitHubScopeRef %s/%s not ready yet", pool.Spec.GitHubScopeRef.Kind, pool.Spec.GitHubScopeRef.Name)
		r.errorLog(ctx, pool, err)
		conditions.MarkFalse(pool, conditions.ScopeReference, conditions.ScopeRefNotReadyReason, err.Error())
		conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		return ctrl.Result{}, err
	}
	conditions.MarkTrue(pool, conditions.ScopeReference, conditions.FetchingScopeRefSuccessReason, fmt.Sprintf("Successfully fetched %s CR Ref", pool.Spec.GitHubScopeRef.Kind))

	// pool status id has been set and id matches existing garm pool, so we know that the pool is persisted in garm and needs to be updated
	if pool.Status.ID != "" && poolUtil.GarmPoolExists(poolClient, pool) {
		return r.reconcileUpdate(ctx, poolClient, pool, instanceClient)
	}

	// no pool id yet or the existing pool id is outdated, so pool cr needs by either synced to match garm pool or created
	return r.reconcileCreate(ctx, poolClient, pool, gitHubScopeRef)
}

func (r *PoolReconciler) reconcileCreate(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1beta1.Pool, gitHubScopeRef garmoperatorv1beta1.GitHubScope) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Pool doesn't exist on garm side. Creating new pool in garm")

	// get image cr object by name
	image, err := pool.GetImageCR(ctx, r.Client)
	if err != nil {
		conditions.MarkFalse(pool, conditions.ImageReference, conditions.FetchingImageRefFailedReason, err.Error())
		conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.FetchingImageRefFailedReason, err.Error())
		r.errorLog(ctx, pool, err)
		return ctrl.Result{}, err
	}
	conditions.MarkTrue(pool, conditions.ImageReference, conditions.FetchingImageRefSuccessReason, "Successfully fetched Image CR Ref")

	// always create new pool in garm
	garmPool, err := poolUtil.CreatePool(ctx, garmClient, pool, image, gitHubScopeRef)
	if err != nil {
		conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		r.errorLog(ctx, pool, fmt.Errorf("failed creating pool %s: %s", pool.Name, err.Error()))
		return ctrl.Result{}, err
	}

	log.Info("creating pool in garm succeeded")
	event.Info(r.Recorder, pool, "creating pool in garm succeeded")

	pool.Status.ID = garmPool.ID
	pool.Status.LongRunningIdleRunners = garmPool.MinIdleRunners

	conditions.MarkTrue(pool, conditions.ReadyCondition, conditions.SuccessfulReconcileReason, "")

	return ctrl.Result{}, nil
}

func (r *PoolReconciler) reconcileUpdate(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1beta1.Pool, instanceClient garmClient.InstanceClient) (ctrl.Result, error) {
	log := log.FromContext(ctx).
		WithName("reconcileUpdate")
	log.Info("pool on garm side found", "id", pool.Status.ID, "name", pool.Name)

	image, err := pool.GetImageCR(ctx, r.Client)
	if err != nil {
		conditions.MarkFalse(pool, conditions.ImageReference, conditions.FetchingImageRefFailedReason, err.Error())
		conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.FetchingImageRefFailedReason, err.Error())
		r.errorLog(ctx, pool, err)
		return ctrl.Result{}, err
	}
	conditions.MarkTrue(pool, conditions.ImageReference, conditions.FetchingImageRefSuccessReason, "Successfully fetched Image CR Ref")

	poolCRdiffersFromGarmPool, idleRunners, err := r.comparePoolSpecs(ctx, pool, image.Spec.Tag, garmClient)
	if err != nil {
		err := fmt.Errorf("error comparing pool specs: %s", err.Error())
		conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		r.errorLog(ctx, pool, err)
		return ctrl.Result{}, err
	}

	if !poolCRdiffersFromGarmPool {
		log.Info("pool CR differs from pool on garm side. Trigger a garm pool update")

		if err = poolUtil.UpdatePool(ctx, garmClient, pool, image); err != nil {
			log.Error(err, "error updating pool")
			conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
			r.errorLog(ctx, pool, err)
			return ctrl.Result{}, err
		}
	}

	longRunningIdleRunnersCount := len(runnerUtil.OldIdleRunners(config.Config.Operator.MinIdleRunnersAge, idleRunners))

	switch {
	case pool.Spec.MinIdleRunners == 0:
		// scale to zero
		// when scale to zero is desired, we immediately scale down to zero by deleting all idle runners

		log.Info("Scaling pool", "pool", pool.Name)
		event.Scaling(r.Recorder, pool, fmt.Sprintf("scale idle runners down to %d", pool.Spec.MinIdleRunners))

		runners := runnerUtil.DeletableRunners(ctx, idleRunners)
		for _, runner := range runners {
			if err := instanceClient.DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName(runner.Name)); err != nil {
				log.Error(err, "unable to delete runner", "runner", runner.Name)
			}
			longRunningIdleRunnersCount--
		}
	default:
		// If there are more old idle Runners than minIdleRunners are defined in
		// the spec, we delete old idle runners

		// get all idle runners that are older than minRunnerAge
		longRunningIdleRunners := runnerUtil.OldIdleRunners(config.Config.Operator.MinIdleRunnersAge, idleRunners)

		// calculate how many old runners need to be deleted to match the desired minIdleRunners
		alignedRunners := runnerUtil.AlignIdleRunners(int(pool.Spec.MinIdleRunners), longRunningIdleRunners)

		// extract runners which are deletable
		runners := runnerUtil.DeletableRunners(ctx, alignedRunners)
		for _, runner := range runners {
			log.Info("Scaling pool", "pool", pool.Name)
			event.Scaling(r.Recorder, pool, fmt.Sprintf("scale long running idle runners down to %d", pool.Spec.MinIdleRunners))

			if err := instanceClient.DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName(runner.Name)); err != nil {
				log.Error(err, "unable to delete runner", "runner", runner.Name)
			}
			longRunningIdleRunnersCount--
		}
	}

	// update pool idle runners count in status
	if pool.Status.LongRunningIdleRunners != uint(longRunningIdleRunnersCount) {
		pool.Status.LongRunningIdleRunners = uint(longRunningIdleRunnersCount)
	}

	conditions.MarkTrue(pool, conditions.ReadyCondition, conditions.SuccessfulReconcileReason, "")
	return ctrl.Result{}, nil
}

func (r *PoolReconciler) reconcileDelete(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1beta1.Pool, instanceClient garmClient.InstanceClient) (ctrl.Result, error) {
	// pool does not exist in garm database yet as ID in Status is empty, so we can safely delete it
	log := log.FromContext(ctx)
	log.Info("Deleting Pool", "pool", pool.Name)
	event.Deleting(r.Recorder, pool, "")
	conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.DeletingReason, conditions.DeletingPoolMsg)
	if err := r.Status().Update(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}

	// this is to make the deletion of a "pending" pool CR possible
	if pool.Status.ID == "" && controllerutil.ContainsFinalizer(pool, key.PoolFinalizerName) {
		controllerutil.RemoveFinalizer(pool, key.PoolFinalizerName)
		if err := r.Update(ctx, pool); err != nil {
			conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
			r.errorLog(ctx, pool, err)
			return ctrl.Result{}, err
		}

		log.Info("Successfully deleted pool", "pool", pool.Name)
		return ctrl.Result{}, nil
	}

	pool.Spec.MinIdleRunners = 0
	pool.Spec.Enabled = false
	if err := r.Update(ctx, pool); err != nil {
		conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		r.errorLog(ctx, pool, err)
		return ctrl.Result{}, err
	}

	if err := poolUtil.UpdatePool(ctx, garmClient, pool, nil); err != nil {
		conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		r.errorLog(ctx, pool, err)
		return ctrl.Result{}, err
	}

	// get all runners
	runners, err := runnerUtil.GetRunnersByPoolID(ctx, pool, instanceClient)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
	}

	// get a list of all idle runners to trigger deletion
	deletableRunners := runnerUtil.DeletableRunners(ctx, runners)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, err
	}

	// set current idle runners count in status
	pool.Status.LongRunningIdleRunners = uint(len(deletableRunners))

	// scale pool down that all idle runners are deleted
	log.Info("Scaling pool", "pool", pool.Name)
	event.Scaling(r.Recorder, pool, fmt.Sprintf("scale idle runners down to %d before deleting", pool.Spec.MinIdleRunners))

	for _, runner := range deletableRunners {
		if err := instanceClient.DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName(runner.Name)); err != nil {
			log.Error(err, "unable to delete runner", "runner", runner.Name)
		}
	}

	// delete pool in garm
	if err = garmClient.DeletePool(pools.NewDeletePoolParams().WithPoolID(pool.Status.ID)); err != nil {
		conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.DeletionFailedReason, err.Error())
		r.errorLog(ctx, pool, err)
		return ctrl.Result{}, err
	}

	// remove finalizer so k8s can delete resource
	controllerutil.RemoveFinalizer(pool, key.PoolFinalizerName)
	if err := r.Update(ctx, pool); err != nil {
		conditions.MarkFalse(pool, conditions.ReadyCondition, conditions.DeletionFailedReason, err.Error())
		r.errorLog(ctx, pool, err)
		return ctrl.Result{}, err
	}

	log.Info("Successfully deleted pool", "pool", pool.Name)
	return ctrl.Result{}, nil
}

func (r *PoolReconciler) errorLog(ctx context.Context, obj client.Object, err error) {
	log := log.FromContext(ctx)

	log.Error(err, "error")
	event.Error(r.Recorder, obj, err.Error())
}

func (r *PoolReconciler) comparePoolSpecs(ctx context.Context, pool *garmoperatorv1beta1.Pool, imageTag string, poolClient garmClient.PoolClient) (bool, []params.Instance, error) {
	log := log.FromContext(ctx).
		WithName("comparePoolSpecs")

	gitHubScopeRef, err := r.fetchGitHubScopeCRD(ctx, pool)
	if err != nil {
		log.Error(err, "error fetching GitHubScopeRef")
		return false, []params.Instance{}, err
	}

	// as there are some "special" tags, which aren't set by the user and aren't part of the pool spec
	// we need to "discover" them and add them to the pool spec before comparing
	poolTags, err := tags.CreateComparableRunnerTags(pool.Spec.Tags, pool.Spec.OSArch, pool.Spec.OSType)
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
		Image:                  imageTag,
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
	case string(garmoperatorv1beta1.EnterpriseScope):
		tmpGarmPool.EnterpriseID = gitHubScopeRef.GetID()
		tmpGarmPool.EnterpriseName = gitHubScopeRef.GetName()
	case string(garmoperatorv1beta1.OrganizationScope):
		tmpGarmPool.OrgID = gitHubScopeRef.GetID()
		tmpGarmPool.OrgName = gitHubScopeRef.GetName()
	case string(garmoperatorv1beta1.RepositoryScope):
		tmpGarmPool.RepoID = gitHubScopeRef.GetID()
		tmpGarmPool.RepoName = gitHubScopeRef.GetName()
	}

	// we are only interested in IdleRunners
	idleInstances := runnerUtil.IdleRunners(ctx, garmPool.Payload.Instances)

	// empty instances for comparison
	garmPool.Payload.Instances = nil

	return reflect.DeepEqual(tmpGarmPool, garmPool.Payload), idleInstances, nil
}

func (r *PoolReconciler) fetchGitHubScopeCRD(ctx context.Context, pool *garmoperatorv1beta1.Pool) (garmoperatorv1beta1.GitHubScope, error) {
	gitHubScopeNamespacedName := types.NamespacedName{
		Namespace: pool.Namespace,
		Name:      pool.Spec.GitHubScopeRef.Name,
	}

	var gitHubScope client.Object

	switch pool.Spec.GitHubScopeRef.Kind {
	case string(garmoperatorv1beta1.EnterpriseScope):
		gitHubScope = &garmoperatorv1beta1.Enterprise{}
		if err := r.Get(ctx, gitHubScopeNamespacedName, gitHubScope); err != nil {
			return nil, err
		}

	case string(garmoperatorv1beta1.OrganizationScope):
		gitHubScope = &garmoperatorv1beta1.Organization{}
		if err := r.Get(ctx, gitHubScopeNamespacedName, gitHubScope); err != nil {
			return nil, err
		}

	case string(garmoperatorv1beta1.RepositoryScope):
		gitHubScope = &garmoperatorv1beta1.Repository{}
		if err := r.Get(ctx, gitHubScopeNamespacedName, gitHubScope); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported GitHubScopeRef kind: %s", pool.Spec.GitHubScopeRef.Kind)
	}

	return gitHubScope.(garmoperatorv1beta1.GitHubScope), nil
}

func (r *PoolReconciler) findPoolsForImage(ctx context.Context, obj client.Object) []reconcile.Request {
	image, ok := obj.(*garmoperatorv1beta1.Image)
	if !ok {
		return nil
	}

	var pools garmoperatorv1beta1.PoolList
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
func (r *PoolReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	// setup index for image
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &garmoperatorv1beta1.Pool{}, imageField, func(rawObj client.Object) []string {
		pool := rawObj.(*garmoperatorv1beta1.Pool)
		if pool.Spec.ImageName == "" {
			return nil
		}
		return []string{pool.Spec.ImageName}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1beta1.Pool{}).
		Watches(
			&garmoperatorv1beta1.Image{},
			handler.EnqueueRequestsFromMapFunc(r.findPoolsForImage),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		WithOptions(options).
		Complete(r)
}
