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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/v1alpha1"

	newGarmClient "git.i.mercedes-benz.com/GitHub-Actions/garm-operator/pkg/garmclient"
	garmClient "github.com/cloudbase/garm/cmd/garm-cli/client"
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Enterprise object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *EnterpriseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	newClient, err := newGarmClient.NewGarmClient(newGarmClient.GarmClientParams{
		BaseURL:  r.BaseURL,
		Username: r.Username,
		Password: r.Password,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	enterprise := &garmoperatorv1alpha1.Enterprise{}
	err = r.Get(ctx, req.NamespacedName, enterprise)
	if err != nil {
		log.Info("error Mario", "error", err)
		return ctrl.Result{}, err
	}

	// Handle deleted enterprises
	if !enterprise.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, *newClient)
	}

	return r.reconcileNormal(ctx, *newClient)
}

func (r *EnterpriseReconciler) reconcileNormal(ctx context.Context, garmClient garmClient.Client) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	enterprises, err := garmClient.ListEnterprises()
	if err != nil {
		log.Error(err, "client.ListEnterprises error")
		return ctrl.Result{}, err
	}

	log.Info("enterprise", "discovered enterprises", enterprises)

	credentials, err := garmClient.ListCredentials()
	if err != nil {
		log.Error(err, "client.ListCredentials error")
		return ctrl.Result{}, err
	}

	log.Info("credentials", "discovered credentials", credentials)

	return ctrl.Result{}, nil
}

func (r *EnterpriseReconciler) reconcileDelete(ctx context.Context, garmClient garmClient.Client) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnterpriseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garmoperatorv1alpha1.Enterprise{}).
		Complete(r)
}
