// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudbase/garm/client/repositories"
	"github.com/cloudbase/garm/params"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/event"
	"github.com/mercedes-benz/garm-operator/pkg/secret"
	"github.com/mercedes-benz/garm-operator/pkg/util/annotations"
)

// RepositoryReconciler reconciles a Repository object
type RepositoryReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	BaseURL  string
	Username string
	Password string
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
	err = repositoryClient.Login(garmClient.GarmScopeParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
		// Debug:    true,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

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
		return ctrl.Result{}, err
	}

	if repository.Status.ID == "" {
		garmRepository, err := r.getExistingGarmRepo(ctx, client, repository)
		if err != nil {
			return ctrl.Result{}, err
		}

		repoExists := !reflect.ValueOf(garmRepository).IsZero()

		if repoExists {
			return r.syncExistingRepo(ctx, garmRepository, repository, webhookSecret)
		}

		return r.createRepository(ctx, client, repository, webhookSecret)
	}

	if repository.Status.ID != "" {
		return r.updateRepository(ctx, client, repository, webhookSecret)
	}

	log.Info("reconciling repository successfully done")

	return ctrl.Result{}, nil
}

func (r *RepositoryReconciler) syncExistingRepo(ctx context.Context, garmRepository params.Repository, repository *garmoperatorv1alpha1.Repository, webhookSecret string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("repository", repository.Name)

	if !strings.EqualFold(garmRepository.CredentialsName, repository.Spec.CredentialsName) &&
		!strings.EqualFold(garmRepository.WebhookSecret, webhookSecret) {
		err := fmt.Errorf("repository with the same name already exists, but credentials and/or webhook secret are different. Please delete the existing repository first")
		event.Error(r.Recorder, repository, err.Error())
		return ctrl.Result{}, err
	}

	log.Info("garm repository object found for given repository CR")

	repository.Status.ID = garmRepository.ID
	repository.Status.PoolManagerFailureReason = garmRepository.PoolManagerStatus.FailureReason
	repository.Status.PoolManagerIsRunning = garmRepository.PoolManagerStatus.IsRunning

	if err := r.Status().Update(ctx, repository); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("existing garm repository object successfully adopted")
	event.Info(r.Recorder, repository, "existing garm repository object successfully adopted")

	return ctrl.Result{}, nil
}

func (r *RepositoryReconciler) createRepository(ctx context.Context, client garmClient.RepositoryClient, repository *garmoperatorv1alpha1.Repository, webhookSecret string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("repository", repository.Name)

	log.Info("status.ID is empty and repository doesn't exist on garm side. Creating new repository in garm")
	event.Creating(r.Recorder, repository, "repository doesn't exist on garm side")

	retValue, err := client.CreateRepository(
		repositories.NewCreateRepoParams().
			WithBody(params.CreateRepoParams{
				Name:            repository.Name,
				CredentialsName: repository.Spec.CredentialsName,
				Owner:           repository.Spec.Owner,
				WebhookSecret:   webhookSecret, // gh hook secret
			}))
	log.V(1).Info(fmt.Sprintf("repository %s created - return Value %v", repository.Name, retValue))
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.CreateRepository error: %s", err))
		return ctrl.Result{}, err
	}

	repository.Status.ID = retValue.Payload.ID
	repository.Status.PoolManagerFailureReason = retValue.Payload.PoolManagerStatus.FailureReason
	repository.Status.PoolManagerIsRunning = retValue.Payload.PoolManagerStatus.IsRunning

	if err := r.Status().Update(ctx, repository); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("creating repository in garm succeeded")
	event.Info(r.Recorder, repository, "creating repository in garm succeeded")

	return ctrl.Result{}, nil
}

func (r *RepositoryReconciler) updateRepository(ctx context.Context, client garmClient.RepositoryClient, repository *garmoperatorv1alpha1.Repository, webhookSecret string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("repository", repository.Name)

	log.Info("comparing repository with garm repository object")
	garmRepository, err := client.GetRepository(
		repositories.NewGetRepoParams().
			WithRepoID(repository.Status.ID),
	)
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.GetRepository error: %s", err))
		return ctrl.Result{}, err
	}

	repository.Status.PoolManagerFailureReason = garmRepository.Payload.PoolManagerStatus.FailureReason
	repository.Status.PoolManagerIsRunning = garmRepository.Payload.PoolManagerStatus.IsRunning

	log.V(1).Info("compare credentials and webhook secret with garm repository object")

	if repository.Spec.CredentialsName != garmRepository.Payload.CredentialsName &&
		webhookSecret != garmRepository.Payload.WebhookSecret {
		log.Info("repository credentials or webhook secret has changed, updating garm repository object")
		event.Updating(r.Recorder, repository, "repository credentials or webhook secret has changed")

		// update credentials and webhook secret
		retValue, err := client.UpdateRepository(
			repositories.NewUpdateRepoParams().
				WithRepoID(repository.Status.ID).
				WithBody(params.UpdateEntityParams{
					CredentialsName: repository.Spec.CredentialsName,
					WebhookSecret:   webhookSecret, // gh hook secret
				}))
		if err != nil {
			log.V(1).Info(fmt.Sprintf("client.UpdateRepository error: %s", err))
			return ctrl.Result{}, err
		}

		repository.Status.PoolManagerFailureReason = retValue.Payload.PoolManagerStatus.FailureReason
		repository.Status.PoolManagerIsRunning = retValue.Payload.PoolManagerStatus.IsRunning
	}

	if err := r.Status().Update(ctx, repository); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RepositoryReconciler) getExistingGarmRepo(ctx context.Context, client garmClient.RepositoryClient, repository *garmoperatorv1alpha1.Repository) (params.Repository, error) {
	log := log.FromContext(ctx)
	log.WithValues("repository", repository.Name)

	log.Info("status.ID is empty, checking if repository already exists on garm side")
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

func (r *RepositoryReconciler) reconcileDelete(ctx context.Context, scope garmClient.RepositoryClient, repository *garmoperatorv1alpha1.Repository) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("repository", repository.Name)

	log.Info("starting repository deletion")
	event.Deleting(r.Recorder, repository, "starting repository deletion")

	err := scope.DeleteRepository(
		repositories.NewDeleteRepoParams().
			WithRepoID(repository.Status.ID),
	)
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.DeleteRepository error: %s", err))
		event.Error(r.Recorder, repository, err.Error())
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

func (r *RepositoryReconciler) ensureFinalizer(ctx context.Context, pool *garmoperatorv1alpha1.Repository) error {
	if !controllerutil.ContainsFinalizer(pool, key.RepositoryFinalizerName) {
		controllerutil.AddFinalizer(pool, key.RepositoryFinalizerName)
		return r.Update(ctx, pool)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Repository{}).
		Complete(r)
}
