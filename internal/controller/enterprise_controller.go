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
	"fmt"

	"github.com/cloudbase/garm/params"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/v1alpha1"

	garmClient "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client"
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/client/key"
)

// EnterpriseReconciler reconciles a Enterprise object
type EnterpriseReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	BaseURL  string
	Username string
	Password string
}

//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=enterprises,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=enterprises/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,resources=enterprises/finalizers,verbs=update

func (r *EnterpriseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	enterprise := &garmoperatorv1alpha1.Enterprise{}
	err := r.Get(ctx, req.NamespacedName, enterprise)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("object was not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	scope, err := garmClient.NewEnterpriseClient(garmClient.GarmScopeParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
		// Debug:    true,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// Handle deleted enterprises
	if !enterprise.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, scope, enterprise)
	}

	return r.reconcileNormal(ctx, scope, enterprise)
}

// func (r *EnterpriseReconciler) reconcileNormal(ctx context.Context, scope garmClient.EnterpriseScope) (ctrl.Result, error) {
func (r *EnterpriseReconciler) reconcileNormal(ctx context.Context, scope garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// If the Enterprise doesn't has our finalizer, add it.
	if !controllerutil.ContainsFinalizer(enterprise, key.EnterpriseFinalizerName) {

		controllerutil.AddFinalizer(enterprise, key.EnterpriseFinalizerName)

		// Update the object immediately
		if err := r.Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}
	}

	// if enterprise has no ID, it means it was not created yet or it was already created and the status field is just empty (after e.g. restore)
	// let's check if we could discover the ID by listing all enterprises
	if enterprise.Status.ID == "" {
		// try to adopt existing enterprise
		adopted, err := r.adoptExisting(ctx, scope, enterprise)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !adopted {
			// if adoption failed, create new enterprise
			if err := r.createOrUpdate(ctx, scope, enterprise); err != nil {
				return ctrl.Result{}, err
			}
		}

		// todo: after creation we might have all status information available, so we could update the CR here
		return ctrl.Result{}, nil
	}

	// update status fields
	// reflect enterprise status into CR
	garmEnterprise, err := scope.GetEnterprise(enterprise.Status.ID)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Updating enterprise status")
	//enterprise.Status.PoolManagerFailureReason = garmEnterprise.PoolManagerStatus.FailureReason
	enterprise.Status.PoolManagerFailureReason = "mario was here"
	enterprise.Status.PoolManagerIsRunning = garmEnterprise.PoolManagerStatus.IsRunning
	r.Status().Update(ctx, enterprise)

	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) reconcileDelete(ctx context.Context, scope garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	log.Info("start deleting enterprise")

	err := scope.DeleteEnterprise(enterprise.Status.ID)
	if err != nil {
		return ctrl.Result{}, err
	}

	if controllerutil.ContainsFinalizer(enterprise, key.EnterpriseFinalizerName) {
		controllerutil.RemoveFinalizer(enterprise, key.EnterpriseFinalizerName)
	}

	// update immediately
	if err := r.Update(ctx, enterprise); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("enterprise deletion done")

	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) createOrUpdate(ctx context.Context, scope garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise) error {
	log := log.FromContext(ctx)

	log.Info("enterprise doesn't exist yet, creating...")
	retValue, err := scope.CreateEnterprise(
		params.CreateEnterpriseParams{
			Name:            enterprise.Name,
			CredentialsName: enterprise.Spec.CredentialsName,
			WebhookSecret:   enterprise.Spec.WebhookSecret, // gh hook secret
		},
	)
	log.V(1).Info(fmt.Sprintf("enterprise %s created - return Value %v", enterprise.Name, retValue))
	if err != nil {
		log.Info("DEBUG", "client.CreateEnterprise error", err)
		//enterprise.Status.PoolManagerFailureReason = err.Error()
		//enterprise.Status.PoolManagerIsRunning = false
		r.Status().Update(ctx, enterprise)
		return err
	}

	// reflect enterprise status into CR
	enterprise.Status.ID = retValue.ID
	enterprise.Status.PoolManagerFailureReason = retValue.PoolManagerStatus.FailureReason
	enterprise.Status.PoolManagerIsRunning = retValue.PoolManagerStatus.IsRunning

	return nil
}

func (r *EnterpriseReconciler) adoptExisting(ctx context.Context, scope garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise) (bool, error) {
	log := log.FromContext(ctx)

	enterprises, err := scope.ListEnterprises()
	if err != nil {
		return false, err
	}

	// TODO: better logging and additional information in debug log
	log.Info(fmt.Sprintf("%d enterprises discovered", len(enterprises)))

	for _, garmEnterprise := range enterprises {
		if garmEnterprise.Name == enterprise.Name {
			log.Info("garm enterprise object found for given enterprise CR")
			enterprise.Status.ID = garmEnterprise.ID
			enterprise.Status.PoolManagerFailureReason = garmEnterprise.PoolManagerStatus.FailureReason
			enterprise.Status.PoolManagerIsRunning = garmEnterprise.PoolManagerStatus.IsRunning
			r.Status().Update(ctx, enterprise)
			return true, err
		}
	}

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnterpriseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Enterprise{}).
		Complete(r)
}
