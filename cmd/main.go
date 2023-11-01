// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"os"
	"time"

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
	exitCode := 0
	defer func() { os.Exit(exitCode) }()

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
		exitCode = 1
		return
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
		exitCode = 1
		return
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
		exitCode = 1
		return
	}

	if os.Getenv("CREATE_WEBHOOK") == "true" {
		if err = (&garmoperatorv1alpha1.Pool{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Pool")
			exitCode = 1
			return
		}
		if err = (&garmoperatorv1alpha1.Image{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Image")
			exitCode = 1
			return
		}
		if err = (&garmoperatorv1alpha1.Repository{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Repository")
			exitCode = 1
			return
		}
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
		exitCode = 1
		return
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
		exitCode = 1
		return
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
		exitCode = 1
		return
	}

	// fetch runner instances periodically and enqueue reconcile events for runner ctrl if external system has changed
	ctx, cancel := context.WithCancel(context.Background())
	go runnerReconciler.PollRunnerInstances(ctx, eventChan)
	defer cancel()

	if err = (&garmoperatorv1alpha1.Runner{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Runner")
		exitCode = 1
		return
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		exitCode = 1
		return
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		exitCode = 1
		return
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		exitCode = 1
		return
	}
}
