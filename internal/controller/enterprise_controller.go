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
	"strings"

	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/secret"

	"github.com/cloudbase/garm/client/enterprises"
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

//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=enterprises,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=enterprises/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=garm-operator.mercedes-benz.com,namespace=xxxxx,resources=enterprises/finalizers,verbs=update

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
	log.WithValues("enterprise", enterprise.Name)

	// If the Enterprise doesn't has our finalizer, add it.
	if !controllerutil.ContainsFinalizer(enterprise, key.EnterpriseFinalizerName) {
		log.V(1).Info(fmt.Sprintf("enterprise doesn't have the %s finalizer, adding it", key.EnterpriseFinalizerName))
		controllerutil.AddFinalizer(enterprise, key.EnterpriseFinalizerName)

		// Update the object immediately
		if err := r.Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}
	}

	webhookSecret, err := secret.FetchRef(ctx, r.Client, &enterprise.Spec.WebhookSecretRef, enterprise.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	if enterprise.Status.ID == "" {
		log.Info("status.ID is empty, checking if enterprise already exists on garm side")
		enterprises, err := scope.ListEnterprises(enterprises.NewListEnterprisesParams())
		if err != nil {
			return ctrl.Result{}, err
		}

		log.Info(fmt.Sprintf("%d enterprises discovered", len(enterprises.Payload)))
		log.V(1).Info(fmt.Sprintf("enterprises on garm side: %#v", enterprises.Payload))

		for _, garmEnterprise := range enterprises.Payload {
			if strings.EqualFold(garmEnterprise.Name, enterprise.Name) {

				if !strings.EqualFold(garmEnterprise.CredentialsName, enterprise.Spec.CredentialsName) &&
					!strings.EqualFold(garmEnterprise.WebhookSecret, webhookSecret) {
					return ctrl.Result{}, fmt.Errorf("enterprise with the same name already exists, but credentials and/or webhook secret are different. Please delete the existing enterprise first")
				}

				log.Info("garm enterprise object found for given enterprise CR")

				enterprise.Status.ID = garmEnterprise.ID
				enterprise.Status.PoolManagerFailureReason = garmEnterprise.PoolManagerStatus.FailureReason
				enterprise.Status.PoolManagerIsRunning = garmEnterprise.PoolManagerStatus.IsRunning

				if err := r.Status().Update(ctx, enterprise); err != nil {
					return ctrl.Result{}, err
				}

				log.Info("existing garm enterprise object successfully adopted")

				return ctrl.Result{}, nil
			}
		}
	}

	switch {
	case enterprise.Status.ID == "":

		log.Info("status.ID is empty and enterprise doesn't exist on garm side. Creating new enterprise in garm")

		retValue, err := scope.CreateEnterprise(
			enterprises.NewCreateEnterpriseParams().
				WithBody(params.CreateEnterpriseParams{
					Name:            enterprise.Name,
					CredentialsName: enterprise.Spec.CredentialsName,
					WebhookSecret:   webhookSecret, // gh hook secret
				}))
		log.V(1).Info(fmt.Sprintf("enterprise %s created - return Value %v", enterprise.Name, retValue))
		if err != nil {
			log.V(1).Info(fmt.Sprintf("client.CreateEnterprise error: %s", err))
			return ctrl.Result{}, err
		}

		enterprise.Status.ID = retValue.Payload.ID
		enterprise.Status.PoolManagerFailureReason = retValue.Payload.PoolManagerStatus.FailureReason
		enterprise.Status.PoolManagerIsRunning = retValue.Payload.PoolManagerStatus.IsRunning

		if err := r.Status().Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}

		log.Info("creating enterprise in garm succeeded")

		return ctrl.Result{}, nil

	case enterprise.Status.ID != "":

		log.Info("comparing enterprise with garm enterprise object")
		garmEnterprise, err := scope.GetEnterprise(
			enterprises.NewGetEnterpriseParams().
				WithEnterpriseID(enterprise.Status.ID),
		)
		if err != nil {
			log.V(1).Info(fmt.Sprintf("client.GetEnterprise error: %s", err))
			return ctrl.Result{}, err
		}

		enterprise.Status.PoolManagerFailureReason = garmEnterprise.Payload.PoolManagerStatus.FailureReason
		enterprise.Status.PoolManagerIsRunning = garmEnterprise.Payload.PoolManagerStatus.IsRunning

		if err := r.Status().Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}

		log.V(1).Info("compare credentials and webhook secret with garm enterprise object")

		if enterprise.Spec.CredentialsName != garmEnterprise.Payload.CredentialsName &&
			webhookSecret != garmEnterprise.Payload.WebhookSecret {

			log.Info("enterprise credentials or webhook secret has changed, updating garm enterprise object")

			// update credentials and webhook secret
			retValue, err := scope.UpdateEnterprise(
				enterprises.NewUpdateEnterpriseParams().
					WithEnterpriseID(enterprise.Status.ID).
					WithBody(params.UpdateEntityParams{
						CredentialsName: enterprise.Spec.CredentialsName,
						WebhookSecret:   webhookSecret, // gh hook secret
					}))
			if err != nil {
				log.V(1).Info(fmt.Sprintf("client.UpdateEnterprise error: %s", err))
				return ctrl.Result{}, err
			}

			enterprise.Status.PoolManagerFailureReason = retValue.Payload.PoolManagerStatus.FailureReason
			enterprise.Status.PoolManagerIsRunning = retValue.Payload.PoolManagerStatus.IsRunning

			if err := r.Status().Update(ctx, enterprise); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
	}

	log.Info("reconciling enterprise successfully done")

	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) reconcileDelete(ctx context.Context, scope garmClient.EnterpriseClient, enterprise *garmoperatorv1alpha1.Enterprise) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.WithValues("enterprise", enterprise.Name)

	log.Info("starting enterprise deletion")

	err := scope.DeleteEnterprise(
		enterprises.NewDeleteEnterpriseParams().
			WithEnterpriseID(enterprise.Status.ID),
	)
	if err != nil {
		log.V(1).Info(fmt.Sprintf("client.DeleteEnterprise error: %s", err))
		return ctrl.Result{}, err
	}

	if controllerutil.ContainsFinalizer(enterprise, key.EnterpriseFinalizerName) {
		controllerutil.RemoveFinalizer(enterprise, key.EnterpriseFinalizerName)

		// update immediately
		if err := r.Update(ctx, enterprise); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("enterprise deletion done")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnterpriseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Enterprise{}).
		Complete(r)
}
