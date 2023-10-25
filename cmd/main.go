// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	"github.com/mercedes-benz/garm-operator/internal/controller"
	"github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/config"
	"github.com/mercedes-benz/garm-operator/pkg/flags"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(garmoperatorv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	ctrl.SetLogger(klogr.New())

	// initiate flags
	f := flags.InitiateFlags()

	// retrieve config flag value for GenerateConfig() function
	configFile := f.Lookup("config").Value.String()

	// call GenerateConfig() function from config package
	if err := config.GenerateConfig(f, configFile); err != nil {
		setupLog.Error(err, "failed to read config")
		os.Exit(1)
	}

	// check if dry-run flag is set to true
	dryRun, _ := f.GetBool("dry-run")

	// perform dry-run if enabled and print out the generated Config as yaml
	if dryRun {
		yamlConfig, err := yaml.Marshal(config.Config)
		if err != nil {
			setupLog.Error(err, "failed to marshal config as yaml")
			os.Exit(1)
		}
		fmt.Printf("generated Config as yaml:\n%s\n", yamlConfig)
		os.Exit(0)
	}

	var watchNamespaces []string
	if config.Config.Operator.WatchNamespace != "" {
		watchNamespaces = []string{config.Config.Operator.WatchNamespace}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     config.Config.Operator.MetricsBindAddress,
		Port:                   9443,
		HealthProbeBindAddress: config.Config.Operator.HealthProbeBindAddress,
		LeaderElection:         config.Config.Operator.LeaderElection,
		LeaderElectionID:       "b608d8b3.mercedes-benz.com",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
		//
		// Default Sync Period = 10 hours.
		// Set default via flag to 5 minutes
		SyncPeriod: &config.Config.Operator.SyncPeriod,
		Cache: cache.Options{
			Namespaces: watchNamespaces,
			SyncPeriod: &config.Config.Operator.SyncPeriod,
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()

	if config.Config.Garm.Init {
		initClient := client.NewInitClient()
		if err = initClient.Init(ctx, client.GarmScopeParams{
			BaseURL:  config.Config.Garm.Server,
			Username: config.Config.Garm.Username,
			Password: config.Config.Garm.Password,
			Email:    config.Config.Garm.Email,
		}); err != nil {
			setupLog.Error(err, "failed to initialize GARM")
			os.Exit(1)
		}
	}

	if err = (&controller.EnterpriseReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("enterprise-controller"),

		BaseURL:  config.Config.Garm.Server,
		Username: config.Config.Garm.Username,
		Password: config.Config.Garm.Password,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Enterprise")
		os.Exit(1)
	}
	if err = (&controller.PoolReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("pool-controller"),

		BaseURL:  config.Config.Garm.Server,
		Username: config.Config.Garm.Username,
		Password: config.Config.Garm.Password,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Pool")
		os.Exit(1)
	}

	if err = (&garmoperatorv1alpha1.Pool{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Pool")
		os.Exit(1)
	}
	if err = (&garmoperatorv1alpha1.Image{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Image")
		os.Exit(1)
	}
	if err = (&garmoperatorv1alpha1.Repository{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Repository")
		os.Exit(1)
	}

	if err = (&controller.OrganizationReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("organization-controller"),

		BaseURL:  config.Config.Garm.Server,
		Username: config.Config.Garm.Username,
		Password: config.Config.Garm.Password,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Organization")
		os.Exit(1)
	}

	if err = (&controller.RepositoryReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("repository-controller"),

		BaseURL:  config.Config.Garm.Server,
		Username: config.Config.Garm.Username,
		Password: config.Config.Garm.Password,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Repository")
		os.Exit(1)
	}

	eventChan := make(chan event.GenericEvent)
	runnerReconciler := &controller.RunnerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),

		BaseURL:  config.Config.Garm.Server,
		Username: config.Config.Garm.Username,
		Password: config.Config.Garm.Password,
	}

	// setup controller so it can reconcile if events from eventChan are queued
	if err = runnerReconciler.SetupWithManager(mgr, eventChan); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Runner")
		os.Exit(1)
	}

	// fetch runner instances periodically and enqueue reconcile events for runner ctrl if external system has changed
	go runnerReconciler.PollRunnerInstances(context.Background(), eventChan)

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
