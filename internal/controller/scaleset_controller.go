// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	garmProviderParams "github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/client/repositories"
	"github.com/cloudbase/garm/client/scalesets"
	"github.com/cloudbase/garm/params"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
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
	"github.com/mercedes-benz/garm-operator/pkg/event"
	"github.com/mercedes-benz/garm-operator/pkg/finalizers"
	scaleSetUtil "github.com/mercedes-benz/garm-operator/pkg/scalesets"
)

// drainRequeueInterval is how often a scale set that is waiting for busy
// runners to finish is looked at again.
const drainRequeueInterval = 1 * time.Minute

// ScaleSetReconciler reconciles a ScaleSet object against the GARM scale-set
// API.
//
// A ScaleSet is bound to its remote scale set through status.id (the stable
// remote numeric ID). The CR name is only used to detect a collision with a
// remote set that already exists; such a set is adopted only when the CR opts
// in with the key.AdoptAnnotation annotation, and adoption never deletes or
// recreates anything.
type ScaleSetReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Recorder       events.EventRecorder
	ScaleSetClient garmClient.ScaleSetClient
	InstanceClient garmClient.InstanceClient
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=scalesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=scalesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=scalesets/finalizers,verbs=update

func (r *ScaleSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	log := log.FromContext(ctx)

	set := &garmoperatorv1beta1.ScaleSet{}
	if err := r.Get(ctx, req.NamespacedName, set); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "cannot fetch ScaleSet")
		return ctrl.Result{}, err
	}

	deleting := !set.DeletionTimestamp.IsZero()

	// Paused objects are left alone, except for deletion: skipping it would
	// leave a deleted object stuck on its finalizer until someone notices the
	// annotation. Pause stops the operator from changing the remote set while
	// the object is alive; it does not make the object undeletable.
	if annotations.IsPaused(set) && !deleting {
		log.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	// An invalid spec cannot get better by retrying: report it on the object
	// and wait for the next change instead of backing off forever. Deletion
	// skips validation so an invalid object can always release its finalizer.
	if !deleting {
		if err := set.Spec.Validate(); err != nil {
			log.Info("ScaleSet spec is invalid, waiting for a change", "reason", err.Error())
			conditions.MarkFalse(set, conditions.ReadyCondition, conditions.InvalidSpecReason, err.Error())
			if uerr := r.Status().Update(ctx, set); uerr != nil {
				return ctrl.Result{}, uerr
			}
			r.warn(set, "Validate", err.Error())
			return ctrl.Result{}, nil
		}
	}

	if finalizerAdded, err := finalizers.EnsureFinalizer(ctx, r.Client, set, key.ScaleSetFinalizerName); err != nil || finalizerAdded {
		return ctrl.Result{}, err
	}

	set.InitializeConditions()

	initial := set.DeepCopy()
	defer func() {
		if deleting && !controllerutil.ContainsFinalizer(set, key.ScaleSetFinalizerName) {
			return
		}
		if !reflect.DeepEqual(set.Status, initial.Status) {
			if err := r.Status().Update(ctx, set); err != nil {
				log.Error(err, "failed to update status")
				res = ctrl.Result{}
				retErr = err
			}
		}
	}()

	scaleSetClient := r.ScaleSetClient
	if scaleSetClient == nil {
		scaleSetClient = garmClient.NewScaleSetClient()
	}
	instanceClient := r.InstanceClient
	if instanceClient == nil {
		instanceClient = garmClient.NewInstanceClient()
	}

	if deleting {
		return r.reconcileDelete(ctx, scaleSetClient, instanceClient, set)
	}
	return r.reconcileNormal(ctx, scaleSetClient, instanceClient, set)
}

func (r *ScaleSetReconciler) reconcileNormal(ctx context.Context, scaleSetClient garmClient.ScaleSetClient, instanceClient garmClient.InstanceClient, set *garmoperatorv1beta1.ScaleSet) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	scope, err := r.fetchScope(ctx, set)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// The scope watch reconciles us again once it shows up.
			conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ScopeRefNotFoundReason,
				fmt.Sprintf("referenced scope %s/%s not found", set.Spec.GitHubScopeRef.Kind, set.Spec.GitHubScopeRef.Name))
			return ctrl.Result{}, nil
		}
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		return ctrl.Result{}, err
	}
	if scope.GetID() == "" {
		// The scope watch reconciles us again once the scope has an ID.
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ScopeRefNotReadyReason,
			fmt.Sprintf("referenced scope %s/%s not ready yet", set.Spec.GitHubScopeRef.Kind, set.Spec.GitHubScopeRef.Name))
		return ctrl.Result{}, nil
	}

	// Bound: status.id points at the remote set this object owns.
	if set.Status.ID != "" {
		id, err := strconv.ParseUint(set.Status.ID, 10, 64)
		if err != nil {
			conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
			return ctrl.Result{}, err
		}
		resp, err := scaleSetClient.GetScaleSet(uint(id))
		switch {
		case err == nil:
			live := resp.GetPayload()
			if scaleSetUtil.ImmutableDrift(set.Spec, live, scope) {
				return r.recreate(ctx, scaleSetClient, instanceClient, set, scope, live)
			}
			return r.reconcileUpdate(ctx, scaleSetClient, set, live)
		case garmClient.IsNotFoundError(err):
			// The remote set is gone (deleted out of band). Forget the stale
			// ID and go through the unbound path below, which refuses to
			// create over a name collision.
			log.Info("remote scale set no longer exists, binding anew", "id", set.Status.ID)
			set.Status.ID = ""
		default:
			conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
			return ctrl.Result{}, err
		}
	}

	// Unbound: look for a remote set with our name. A list failure must not
	// fall through to create, or a transient error would produce duplicates.
	sets, err := r.listScopeScaleSets(scaleSetClient, scope)
	if err != nil {
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		return ctrl.Result{}, err
	}
	if live := scaleSetUtil.FindByName(sets, set.Name); live != nil {
		return r.reconcileNameCollision(ctx, scaleSetClient, set, scope, *live)
	}
	return r.reconcileCreate(ctx, scaleSetClient, set, scope)
}

// reconcileNameCollision handles a remote scale set that already carries this
// object's name but is not bound to it. It may belong to someone else (another
// CR, another namespace, a manual setup), so it is only adopted on explicit
// opt-in, and adoption only ever updates mutable fields: when immutable fields
// differ we stop instead of disabling, draining and deleting a set we did not
// create.
func (r *ScaleSetReconciler) reconcileNameCollision(ctx context.Context, scaleSetClient garmClient.ScaleSetClient, set *garmoperatorv1beta1.ScaleSet, scope garmoperatorv1beta1.GitHubScope, live params.ScaleSet) (ctrl.Result, error) {
	if set.Annotations[key.AdoptAnnotation] != "true" {
		msg := fmt.Sprintf("a scale set named %q (id %d) already exists in GARM; annotate this ScaleSet with %s=\"true\" to adopt it, or rename this resource",
			set.Name, live.ID, key.AdoptAnnotation)
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.AdoptionRequiredReason, msg)
		r.warn(set, "Adopt", msg)
		return ctrl.Result{}, nil
	}

	owner, err := r.findBoundScaleSet(ctx, set, live.ID)
	if err != nil {
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		return ctrl.Result{}, err
	}
	if owner != nil {
		msg := fmt.Sprintf("scale set %q (id %d) is already managed by ScaleSet %s/%s", live.Name, live.ID, owner.Namespace, owner.Name)
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.AdoptionRefusedReason, msg)
		r.warn(set, "Adopt", msg)
		return ctrl.Result{}, nil
	}

	if scaleSetUtil.ImmutableDrift(set.Spec, live, scope) {
		msg := fmt.Sprintf("scale set %q (id %d) differs in provider, labels or scope; adoption never deletes and recreates a scale set. "+
			"Align spec.providerName and spec.labels with the existing set, or remove it manually", live.Name, live.ID)
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.AdoptionRefusedReason, msg)
		r.warn(set, "Adopt", msg)
		return ctrl.Result{}, nil
	}

	set.Status.ID = strconv.FormatUint(uint64(live.ID), 10)
	r.info(set, "Adopt", fmt.Sprintf("adopted existing scale set %q (id %d)", live.Name, live.ID))
	return r.reconcileUpdate(ctx, scaleSetClient, set, live)
}

// findBoundScaleSet returns another ScaleSet object that is already bound to
// the given remote scale set ID, if any. It sees whatever the manager cache
// sees, so it is a safeguard, not a lock.
func (r *ScaleSetReconciler) findBoundScaleSet(ctx context.Context, set *garmoperatorv1beta1.ScaleSet, remoteID uint) (*garmoperatorv1beta1.ScaleSet, error) {
	list := &garmoperatorv1beta1.ScaleSetList{}
	if err := r.List(ctx, list); err != nil {
		return nil, err
	}
	want := strconv.FormatUint(uint64(remoteID), 10)
	for i := range list.Items {
		other := &list.Items[i]
		if (set.UID != "" && other.UID == set.UID) || (set.Namespace == other.Namespace && set.Name == other.Name) {
			continue
		}
		if other.Status.ID == want {
			return other, nil
		}
	}
	return nil, nil
}

func (r *ScaleSetReconciler) listScopeScaleSets(scaleSetClient garmClient.ScaleSetClient, scope garmoperatorv1beta1.GitHubScope) ([]params.ScaleSet, error) {
	switch scope.GetKind() {
	case string(garmoperatorv1beta1.OrganizationScope):
		resp, err := scaleSetClient.ListOrgScaleSets(scope.GetID())
		if err != nil {
			return nil, err
		}
		return resp.GetPayload(), nil
	case string(garmoperatorv1beta1.RepositoryScope):
		resp, err := scaleSetClient.ListRepoScaleSets(scope.GetID())
		if err != nil {
			return nil, err
		}
		return resp.GetPayload(), nil
	case string(garmoperatorv1beta1.EnterpriseScope):
		resp, err := scaleSetClient.ListEnterpriseScaleSets(scope.GetID())
		if err != nil {
			return nil, err
		}
		return resp.GetPayload(), nil
	default:
		return nil, fmt.Errorf("unsupported scope kind: %s", scope.GetKind())
	}
}

func (r *ScaleSetReconciler) reconcileCreate(ctx context.Context, scaleSetClient garmClient.ScaleSetClient, set *garmoperatorv1beta1.ScaleSet, scope garmoperatorv1beta1.GitHubScope) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	body := scaleSetUtil.BuildCreateParams(set.Spec, set.Name)

	var id uint
	switch scope.GetKind() {
	case string(garmoperatorv1beta1.OrganizationScope):
		p := organizations.NewCreateOrgScaleSetParams().WithBody(body)
		resp, err := scaleSetClient.CreateOrgScaleSet(scope.GetID(), p)
		if err != nil {
			conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
			return ctrl.Result{}, err
		}
		id = resp.GetPayload().ID
	case string(garmoperatorv1beta1.RepositoryScope):
		p := repositories.NewCreateRepoScaleSetParams().WithBody(body)
		resp, err := scaleSetClient.CreateRepoScaleSet(scope.GetID(), p)
		if err != nil {
			conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
			return ctrl.Result{}, err
		}
		id = resp.GetPayload().ID
	case string(garmoperatorv1beta1.EnterpriseScope):
		p := enterprises.NewCreateEnterpriseScaleSetParams().WithBody(body)
		resp, err := scaleSetClient.CreateEnterpriseScaleSet(scope.GetID(), p)
		if err != nil {
			conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
			return ctrl.Result{}, err
		}
		id = resp.GetPayload().ID
	default:
		err := fmt.Errorf("unsupported scope kind: %s", scope.GetKind())
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		return ctrl.Result{}, err
	}

	set.Status.ID = strconv.FormatUint(uint64(id), 10)
	set.Status.Enabled = body.Enabled
	conditions.MarkTrue(set, conditions.ReadyCondition, conditions.SuccessfulReconcileReason, "")
	log.Info("created scale set in garm", "id", id, "name", set.Name)
	r.info(set, "reconcileCreate", "creating scale set in garm succeeded")
	return ctrl.Result{}, nil
}

func (r *ScaleSetReconciler) reconcileUpdate(ctx context.Context, scaleSetClient garmClient.ScaleSetClient, set *garmoperatorv1beta1.ScaleSet, live params.ScaleSet) (ctrl.Result, error) {
	set.Status.Enabled = live.Enabled
	// Spec is source of truth: any drift, including externally disabled
	// remotes, gets driven back on every reconcile. Skip the call when
	// live already matches to avoid update churn.
	if !scaleSetUtil.MutableDrift(set.Spec, live) {
		conditions.MarkTrue(set, conditions.ReadyCondition, conditions.SuccessfulReconcileReason, "")
		return ctrl.Result{}, nil
	}
	body := scaleSetUtil.BuildUpdateParams(set.Spec, live)
	p := scalesets.NewUpdateScaleSetParams().WithBody(body)
	if _, err := scaleSetClient.UpdateScaleSet(live.ID, p); err != nil {
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		return ctrl.Result{}, err
	}
	conditions.MarkTrue(set, conditions.ReadyCondition, conditions.SuccessfulReconcileReason, "")
	return ctrl.Result{}, nil
}

// recreate handles immutable drift (labels, provider, scope): disable, wait
// for zero instances, delete, then create fresh. Any failure stops before
// deletion, leaving the set disabled but intact.
//
// With deletionPolicy Retain the operator promises not to delete the remote
// set, so it refuses to recreate and reports why instead.
func (r *ScaleSetReconciler) recreate(ctx context.Context, scaleSetClient garmClient.ScaleSetClient, instanceClient garmClient.InstanceClient, set *garmoperatorv1beta1.ScaleSet, scope garmoperatorv1beta1.GitHubScope, live params.ScaleSet) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if set.Spec.DeletionPolicy == garmoperatorv1beta1.ScaleSetDeletionPolicyRetain {
		msg := fmt.Sprintf("provider, labels or scope changed, which requires deleting and recreating scale set %d, but deletionPolicy is Retain. "+
			"Revert the change, or set deletionPolicy to Delete to allow recreation", live.ID)
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.RecreateBlockedReason, msg)
		r.warn(set, "recreate", msg)
		return ctrl.Result{}, nil
	}

	log.Info("immutable fields drifted, recreating scale set", "id", live.ID, "name", set.Name)

	if live.Enabled {
		if err := r.disable(scaleSetClient, live.ID); err != nil {
			conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
			return ctrl.Result{}, err
		}
	}
	// Disabling orphans idle runners without deleting them, so actively
	// remove idle/errored instances. Busy runners are left alone; deletion waits.
	busy, failed := r.deleteDrainableInstances(ctx, live.Instances, instanceClient)
	if len(live.Instances) > 0 {
		r.markDraining(set, "recreate", "before recreating", len(live.Instances), busy, failed)
		return ctrl.Result{RequeueAfter: drainRequeueInterval}, nil
	}
	if err := scaleSetClient.DeleteScaleSet(live.ID); err != nil {
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		return ctrl.Result{}, err
	}
	set.Status.ID = ""
	return r.reconcileCreate(ctx, scaleSetClient, set, scope)
}

func (r *ScaleSetReconciler) reconcileDelete(ctx context.Context, scaleSetClient garmClient.ScaleSetClient, instanceClient garmClient.InstanceClient, set *garmoperatorv1beta1.ScaleSet) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if set.Status.ID == "" {
		controllerutil.RemoveFinalizer(set, key.ScaleSetFinalizerName)
		return ctrl.Result{}, r.Update(ctx, set)
	}

	// Retain policy: stop managing without touching the remote set.
	if set.Spec.DeletionPolicy == garmoperatorv1beta1.ScaleSetDeletionPolicyRetain {
		controllerutil.RemoveFinalizer(set, key.ScaleSetFinalizerName)
		return ctrl.Result{}, r.Update(ctx, set)
	}

	id, err := strconv.ParseUint(set.Status.ID, 10, 64)
	if err != nil {
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		return ctrl.Result{}, err
	}
	resp, err := scaleSetClient.GetScaleSet(uint(id))
	if err != nil {
		if garmClient.IsNotFoundError(err) {
			controllerutil.RemoveFinalizer(set, key.ScaleSetFinalizerName)
			return ctrl.Result{}, r.Update(ctx, set)
		}
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
		return ctrl.Result{}, err
	}
	live := resp.GetPayload()

	if live.Enabled {
		if err := r.disable(scaleSetClient, live.ID); err != nil {
			conditions.MarkFalse(set, conditions.ReadyCondition, conditions.ReconcileErrorReason, err.Error())
			return ctrl.Result{}, err
		}
	}
	busy, failed := r.deleteDrainableInstances(ctx, live.Instances, instanceClient)
	if len(live.Instances) > 0 {
		r.markDraining(set, "reconcileDelete", "before deleting", len(live.Instances), busy, failed)
		return ctrl.Result{RequeueAfter: drainRequeueInterval}, nil
	}
	if err := scaleSetClient.DeleteScaleSet(live.ID); err != nil {
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.DeletionFailedReason, err.Error())
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(set, key.ScaleSetFinalizerName)
	if err := r.Update(ctx, set); err != nil {
		conditions.MarkFalse(set, conditions.ReadyCondition, conditions.DeletionFailedReason, err.Error())
		return ctrl.Result{}, err
	}
	log.Info("Successfully deleted scale set", "name", set.Name)
	return ctrl.Result{}, nil
}

// markDraining reports a drain that is waiting on runners. The condition
// carries the numbers so `kubectl describe` shows why nothing is happening,
// and an event is recorded when the drain starts, not on every requeue.
func (r *ScaleSetReconciler) markDraining(set *garmoperatorv1beta1.ScaleSet, action, when string, remaining, busy, failed int) {
	msg := fmt.Sprintf("waiting for %d runner(s) to finish %s scale set (%d busy, %d deletion request(s) failed)", remaining, when, busy, failed)
	if cond := conditions.Get(set, conditions.ReadyCondition); cond == nil || cond.Reason != string(conditions.DrainingReason) {
		r.info(set, action, msg)
	}
	conditions.MarkFalse(set, conditions.ReadyCondition, conditions.DrainingReason, msg)
}

// deleteDrainableInstances removes idle and errored instances so drain can
// complete. Busy instances are never touched; callers requeue until empty.
// It returns how many instances were left alone because they are busy and how
// many deletion requests failed.
func (r *ScaleSetReconciler) deleteDrainableInstances(ctx context.Context, all []params.Instance, instanceClient garmClient.InstanceClient) (busy, failed int) {
	log := log.FromContext(ctx)
	for _, runner := range all {
		isIdle := runner.RunnerStatus == params.RunnerIdle
		isError := runner.Status == garmProviderParams.InstanceError
		if !isIdle && !isError {
			busy++
			continue
		}
		if err := instanceClient.DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName(runner.Name)); err != nil {
			log.Error(err, "unable to delete runner instance", "runner", runner.Name, "status", runner.Status)
			failed++
		}
	}
	return busy, failed
}

func (r *ScaleSetReconciler) disable(scaleSetClient garmClient.ScaleSetClient, id uint) error {
	disabled := false
	body := params.UpdateScaleSetParams{Enabled: &disabled}
	p := scalesets.NewUpdateScaleSetParams().WithBody(body)
	_, err := scaleSetClient.UpdateScaleSet(id, p)
	return err
}

func (r *ScaleSetReconciler) info(set *garmoperatorv1beta1.ScaleSet, action, msg string) {
	if r.Recorder != nil {
		event.Info(r.Recorder, set, action, msg)
	}
}

func (r *ScaleSetReconciler) warn(set *garmoperatorv1beta1.ScaleSet, action, msg string) {
	if r.Recorder != nil {
		event.Error(r.Recorder, set, action, msg)
	}
}

func (r *ScaleSetReconciler) fetchScope(ctx context.Context, set *garmoperatorv1beta1.ScaleSet) (garmoperatorv1beta1.GitHubScope, error) {
	name := types.NamespacedName{Namespace: set.Namespace, Name: set.Spec.GitHubScopeRef.Name}
	var scope garmoperatorv1beta1.GitHubScope
	switch set.Spec.GitHubScopeRef.Kind {
	case string(garmoperatorv1beta1.EnterpriseScope):
		obj := &garmoperatorv1beta1.Enterprise{}
		if err := r.Get(ctx, name, obj); err != nil {
			return nil, err
		}
		scope = obj
	case string(garmoperatorv1beta1.OrganizationScope):
		obj := &garmoperatorv1beta1.Organization{}
		if err := r.Get(ctx, name, obj); err != nil {
			return nil, err
		}
		scope = obj
	case string(garmoperatorv1beta1.RepositoryScope):
		obj := &garmoperatorv1beta1.Repository{}
		if err := r.Get(ctx, name, obj); err != nil {
			return nil, err
		}
		scope = obj
	default:
		return nil, fmt.Errorf("unsupported GitHubScopeRef kind: %s", set.Spec.GitHubScopeRef.Kind)
	}
	return scope, nil
}

// findScaleSetsForScope maps a scope object (Enterprise, Organization or
// Repository) to the ScaleSets in its namespace that reference it, so a scope
// appearing or becoming ready reconciles them without polling.
func (r *ScaleSetReconciler) findScaleSetsForScope(kind garmoperatorv1beta1.GitHubScopeKind) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		list := &garmoperatorv1beta1.ScaleSetList{}
		if err := r.List(ctx, list, client.InNamespace(obj.GetNamespace())); err != nil {
			log.FromContext(ctx).Error(err, "unable to list ScaleSets for scope", "kind", kind, "name", obj.GetName())
			return nil
		}
		var requests []reconcile.Request
		for _, set := range list.Items {
			ref := set.Spec.GitHubScopeRef
			if ref.Kind == string(kind) && ref.Name == obj.GetName() {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: set.Namespace, Name: set.Name}})
			}
		}
		return requests
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScaleSetReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	scopeChanged := builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1beta1.ScaleSet{}).
		Watches(&garmoperatorv1beta1.Enterprise{}, handler.EnqueueRequestsFromMapFunc(r.findScaleSetsForScope(garmoperatorv1beta1.EnterpriseScope)), scopeChanged).
		Watches(&garmoperatorv1beta1.Organization{}, handler.EnqueueRequestsFromMapFunc(r.findScaleSetsForScope(garmoperatorv1beta1.OrganizationScope)), scopeChanged).
		Watches(&garmoperatorv1beta1.Repository{}, handler.EnqueueRequestsFromMapFunc(r.findScaleSetsForScope(garmoperatorv1beta1.RepositoryScope)), scopeChanged).
		WithOptions(options).
		Complete(r)
}
