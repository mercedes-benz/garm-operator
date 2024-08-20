// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"

	"github.com/cloudbase/garm/client/repositories"
	"github.com/cloudbase/garm/params"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	"github.com/mercedes-benz/garm-operator/pkg/annotations"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/conditions"
	"github.com/mercedes-benz/garm-operator/pkg/event"
	"github.com/mercedes-benz/garm-operator/pkg/secret"
)

// RepositoryReconciler reconciles a Repository object
type RepositoryReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=repositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=repositories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=repositories/finalizers,verbs=update

func (r *RepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	repository := &garmoperatorv1alpha1.Repository{}
	err := r.Get(ctx, req.NamespacedName, repository)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("object was not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Ignore objects that are paused
	if annotations.IsPaused(repository) {
		log.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	repositoryClient := garmClient.NewRepositoryClient()

	// Handle deleted repositories
	if !repository.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, repositoryClient, repository)
	}

	return r.reconcileNormal(ctx, repositoryClient, repository)
}

func (r *RepositoryReconciler) reconcileNormal(ctx context.Context, client garmClient.RepositoryClient, repository *garmoperatorv1alpha1.Repository) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("repository", repository.Name)

	// If the Repository doesn't have our finalizer, add it.
	if err := r.ensureFinalizer(ctx, repository); err != nil {
		return ctrl.Result{}, err
	}

	webhookSecret, err := secret.FetchRef(ctx, r.Client, &repository.Spec.WebhookSecretRef, repository.Namespace)
	if err != nil {
		conditions.MarkFalse(repository, conditions.ReadyCondition, conditions.FetchingSecretRefFailedReason, err.Error())
		conditions.MarkFalse(repository, conditions.SecretReference, conditions.FetchingSecretRefFailedReason, err.Error())
		conditions.MarkUnknown(repository, conditions.PoolManager, conditions.UnknownReason, conditions.GarmServerNotReconciledYetMsg)
		conditions.MarkUnknown(repository, conditions.CredentialsReference, conditions.UnknownReason, conditions.CredentialsNotReconciledYetMsg)
		if err := r.Status().Update(ctx, repository); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	conditions.MarkTrue(repository, conditions.SecretReference, conditions.FetchingSecretRefSuccessReason, "")

	credentials, err := r.getCredentialsRef(ctx, repository)
	if err != nil {
		conditions.MarkFalse(repository, conditions.ReadyCondition, conditions.FetchingCredentialsRefFailedReason, err.Error())
		conditions.MarkFalse(repository, conditions.CredentialsReference, conditions.FetchingCredentialsRefFailedReason, err.Error())
		if conditions.Get(repository, conditions.PoolManager) == nil {
			conditions.MarkUnknown(repository, conditions.PoolManager, conditions.UnknownReason, conditions.GarmServerNotReconciledYetMsg)
		}
		if err := r.Status().Update(ctx, repository); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	conditions.MarkTrue(repository, conditions.CredentialsReference, conditions.FetchingCredentialsRefSuccessReason, "")

	garmRepository, err := r.getExistingGarmRepo(ctx, client, repository)
	if err != nil {
		event.Error(r.Recorder, repository, err.Error())
		conditions.MarkFalse(repository, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
		if err := r.Status().Update(ctx, repository); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// create repository on garm side if it does not exist
	if reflect.ValueOf(garmRepository).IsZero() {
		garmRepository, err = r.createRepository(ctx, client, repository, webhookSecret)
		if err != nil {
			event.Error(r.Recorder, repository, err.Error())
			conditions.MarkFalse(repository, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
			if err := r.Status().Update(ctx, repository); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
	}

	// update repository anytime
	garmRepository, err = r.updateRepository(ctx, client, garmRepository.ID, webhookSecret, credentials.Name)
	if err != nil {
		event.Error(r.Recorder, repository, err.Error())
		conditions.MarkFalse(repository, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
		if err := r.Status().Update(ctx, repository); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// set and update repository status
	repository.Status.ID = garmRepository.ID
	conditions.MarkTrue(repository, conditions.ReadyCondition, conditions.SuccessfulReconcileReason, "")
	conditions.MarkTrue(repository, conditions.PoolManager, conditions.PoolManagerRunningReason, "")

	if !garmRepository.PoolManagerStatus.IsRunning {
		conditions.MarkFalse(repository, conditions.ReadyCondition, conditions.PoolManagerFailureReason, "Pool Manager is not running")
		conditions.MarkFalse(repository, conditions.PoolManager, conditions.PoolManagerFailureReason, garmRepository.PoolManagerStatus.FailureReason)
	}

	if err := r.Status().Update(ctx, repository); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciling repository successfully done")

	return ctrl.Result{}, nil
}

func (r *RepositoryReconciler) createRepository(ctx context.Context, client garmClient.RepositoryClient, repository *garmoperatorv1alpha1.Repository, webhookSecret string) (params.Repository, error) {
	log := log.FromContext(ctx)
	log.WithValues("repository", repository.Name)

	log.Info("Repository doesn't exist on garm side. Creating new repository in garm.")
	event.Creating(r.Recorder, repository, "repository doesn't exist on garm side")

	retValue, err := client.CreateRepository(
		repositories.NewCreateRepoParams().
			WithBody(params.CreateRepoParams{
				Name:            repository.Name,
				CredentialsName: repository.GetCredentialsName(),
				Owner:           repository.Spec.Owner,
				WebhookSecret:   webhookSecret, // gh hook secret
			}))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.CreateRepository error: %s", err))
		return params.Repository{}, err
	}

	log.V(1).Info(fmt.Sprintf("repository %s created - return Value %v", repository.Name, retValue))

	log.Info("creating repository in garm succeeded")
	event.Info(r.Recorder, repository, "creating repository in garm succeeded")

	return retValue.Payload, nil
}

func (r *RepositoryReconciler) updateRepository(ctx context.Context, client garmClient.RepositoryClient, statusID, webhookSecret, credentialsName string) (params.Repository, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("update credentials and webhook secret in garm repository")

	// update credentials and webhook secret
	retValue, err := client.UpdateRepository(
		repositories.NewUpdateRepoParams().
			WithRepoID(statusID).
			WithBody(params.UpdateEntityParams{
				CredentialsName: credentialsName,
				WebhookSecret:   webhookSecret, // gh hook secret
			}))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.UpdateRepository error: %s", err))
		return params.Repository{}, err
	}

	return retValue.Payload, nil
}

func (r *RepositoryReconciler) getExistingGarmRepo(ctx context.Context, client garmClient.RepositoryClient, repository *garmoperatorv1alpha1.Repository) (params.Repository, error) {
	log := log.FromContext(ctx)
	log.WithValues("repository", repository.Name)

	log.Info("checking if repository already exists on garm side")
	repositories, err := client.ListRepositories(repositories.NewListReposParams())
	if err != nil {
		return params.Repository{}, fmt.Errorf("getExistingGarmRepo: %w", err)
	}

	log.Info(fmt.Sprintf("%d repositories discovered", len(repositories.Payload)))
	log.V(1).Info(fmt.Sprintf("repositories on garm side: %#v", repositories.Payload))

	for _, garmRepository := range repositories.Payload {
		if strings.EqualFold(garmRepository.Name, repository.Name) {
			return garmRepository, nil
		}
	}
	return params.Repository{}, nil
}

func (r *RepositoryReconciler) reconcileDelete(ctx context.Context, client garmClient.RepositoryClient, repository *garmoperatorv1alpha1.Repository) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("repository", repository.Name)

	log.Info("starting repository deletion")
	event.Deleting(r.Recorder, repository, "starting repository deletion")
	conditions.MarkFalse(repository, conditions.ReadyCondition, conditions.DeletingReason, conditions.DeletingRepoMsg)
	if err := r.Status().Update(ctx, repository); err != nil {
		return ctrl.Result{}, err
	}

	err := client.DeleteRepository(
		repositories.NewDeleteRepoParams().
			WithRepoID(repository.Status.ID),
	)
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.DeleteRepository error: %s", err))
		event.Error(r.Recorder, repository, err.Error())
		conditions.MarkFalse(repository, conditions.ReadyCondition, conditions.GarmAPIErrorReason, err.Error())
		if err := r.Status().Update(ctx, repository); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if controllerutil.ContainsFinalizer(repository, key.RepositoryFinalizerName) {
		controllerutil.RemoveFinalizer(repository, key.RepositoryFinalizerName)

		// update immediately
		if err := r.Update(ctx, repository); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("repository deletion done")

	return ctrl.Result{}, nil
}

func (r *RepositoryReconciler) getCredentialsRef(ctx context.Context, repository *garmoperatorv1alpha1.Repository) (*garmoperatorv1alpha1.GitHubCredentials, error) {
	creds := &garmoperatorv1alpha1.GitHubCredentials{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: repository.Namespace,
		Name:      repository.Spec.CredentialsRef.Name,
	}, creds)
	if err != nil {
		return creds, err
	}
	return creds, nil
}

func (r *RepositoryReconciler) ensureFinalizer(ctx context.Context, pool *garmoperatorv1alpha1.Repository) error {
	if !controllerutil.ContainsFinalizer(pool, key.RepositoryFinalizerName) {
		controllerutil.AddFinalizer(pool, key.RepositoryFinalizerName)
		return r.Update(ctx, pool)
	}
	return nil
}

func (r *RepositoryReconciler) findReposForCredentials(ctx context.Context, obj client.Object) []reconcile.Request {
	credentials, ok := obj.(*garmoperatorv1alpha1.GitHubCredentials)
	if !ok {
		return nil
	}

	var repos garmoperatorv1alpha1.RepositoryList
	if err := r.List(ctx, &repos); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, repo := range repos.Items {
		if repo.GetCredentialsName() == credentials.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: repo.Namespace,
					Name:      repo.Name,
				},
			})
		}
	}

	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepositoryReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Repository{}).
		Watches(
			&garmoperatorv1alpha1.GitHubCredentials{},
			handler.EnqueueRequestsFromMapFunc(r.findReposForCredentials),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		WithOptions(options).
		Complete(r)
}
