// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"log"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog/v2/textlogger"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	garmcontroller "github.com/mercedes-benz/garm-operator/internal/controller"
	"github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/config"
	"github.com/mercedes-benz/garm-operator/pkg/flags"
	"github.com/mercedes-benz/garm-operator/pkg/version"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(garmoperatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(garmoperatorv1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// initiate flags
	f := flags.InitiateFlags()

	// retrieve config flag value for GenerateConfig() function
	configFile := f.Lookup("config").Value.String()

	// call GenerateConfig() function from config package
	if err := config.GenerateConfig(f, configFile); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// check if dry-run flag is set to true
	dryRun, _ := f.GetBool("dry-run")

	// perform dry-run if enabled and print out the generated Config as yaml
	if dryRun {
		yamlConfig, err := yaml.Marshal(config.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal config as yaml: %w", err)
		}
		fmt.Printf("generated Config as yaml:\n%s\n", yamlConfig)
		return nil
	}

	ctrl.SetLogger(textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(config.Config.Operator.LogVerbosityLevel))))

	var watchNamespaces map[string]cache.Config
	if config.Config.Operator.WatchNamespace != "" {
		watchNamespaces = map[string]cache.Config{
			config.Config.Operator.WatchNamespace: {},
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: config.Config.Operator.MetricsBindAddress,
		},
		WebhookServer: webhook.NewServer(
			webhook.Options{
				Port: 9443,
			},
		),
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
		Cache: cache.Options{
			DefaultNamespaces: watchNamespaces,
			SyncPeriod:        &config.Config.Operator.SyncPeriod,
		},
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	ctx := ctrl.SetupSignalHandler()

	if err = client.CreateInstance(client.GarmScopeParams{
		BaseURL:  config.Config.Garm.Server,
		Username: config.Config.Garm.Username,
		Password: config.Config.Garm.Password,
		Email:    config.Config.Garm.Email,
	}); err != nil {
		return fmt.Errorf("unable to setup garm: %w", err)
	}

	// create a controller client to get controller info
	// for the version check
	controllerClient := client.NewControllerClient()
	controllerInfo, err := controllerClient.GetControllerInfo()
	if err != nil {
		return fmt.Errorf("unable to get controller info: %w", err)
	}

	// if the version of the controller is not compatible with the Garm version
	// return an error and stop the operator
	if !version.EnsureMinimalVersion(controllerInfo.Payload.Version) {
		return fmt.Errorf("garm-operator is not compatible with Garm version %s. Minimal required version is %s", controllerInfo.Payload.Version, version.MinVersion)
	}

	if err = (&garmcontroller.EnterpriseReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("enterprise-controller"),
	}).SetupWithManager(mgr,
		controller.Options{
			MaxConcurrentReconciles: config.Config.Operator.EnterpriseConcurrency,
		},
	); err != nil {
		return fmt.Errorf("unable to create controller Enterprise: %w", err)
	}

	if err = (&garmcontroller.PoolReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("pool-controller"),
	}).SetupWithManager(mgr,
		controller.Options{
			MaxConcurrentReconciles: config.Config.Operator.PoolConcurrency,
		},
	); err != nil {
		return fmt.Errorf("unable to create controller Pool: %w", err)
	}

	if err = (&garmcontroller.OrganizationReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("organization-controller"),
	}).SetupWithManager(mgr,
		controller.Options{
			MaxConcurrentReconciles: config.Config.Operator.OrganizationConcurrency,
		},
	); err != nil {
		return fmt.Errorf("unable to create controller Organization: %w", err)
	}

	if err = (&garmcontroller.RepositoryReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("repository-controller"),
	}).SetupWithManager(mgr,
		controller.Options{
			MaxConcurrentReconciles: config.Config.Operator.RepositoryConcurrency,
		},
	); err != nil {
		return fmt.Errorf("unable to create controller Repository: %w", err)
	}

	if config.Config.Operator.RunnerReconciliation {
		runnerEvents := make(chan event.GenericEvent)
		runnerReconciler := &garmcontroller.RunnerReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}

		// setup controller so it can reconcile if events from runnerEvents are queued
		if err = runnerReconciler.SetupWithManager(mgr, runnerEvents,
			controller.Options{
				MaxConcurrentReconciles: config.Config.Operator.RunnerConcurrency,
			},
		); err != nil {
			return fmt.Errorf("unable to create controller Runner: %w", err)
		}

		// fetch runner instances periodically and enqueue reconcile events for runner ctrl if external system has changed
		ctx, cancel := context.WithCancel(ctx)
		go runnerReconciler.PollRunnerInstances(ctx, runnerEvents)
		defer cancel()
	}

	if err = (&garmcontroller.GarmServerConfigReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("garm-server-config-controller"),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller GarmServerConfig: %w", err)
	}

	if err = (&garmcontroller.GitHubEndpointReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("github-endpoint-controller"),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller GitHubEndpoint: %w", err)
	}

	if err = (&garmcontroller.GitHubCredentialReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("github-credentials-controller"),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller GitHubCredential: %w", err)
	}

	// webhooks
	if err = (&garmoperatorv1alpha1.Enterprise{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook Enterprise: %w", err)
	}

	if err = (&garmoperatorv1alpha1.Organization{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook Organization: %w", err)
	}

	if err = (&garmoperatorv1alpha1.Pool{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook Pool: %w", err)
	}

	if err = (&garmoperatorv1beta1.Pool{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook Pool: %w", err)
	}

	if err = (&garmoperatorv1alpha1.Image{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook Image: %w", err)
	}

	if err = (&garmoperatorv1beta1.Image{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook Image: %w", err)
	}

	if err = (&garmoperatorv1alpha1.Repository{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook Repository: %w", err)
	}

	if err = (&garmoperatorv1beta1.Repository{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook Repository: %w", err)
	}

	if err = (&garmoperatorv1alpha1.Runner{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create webhook Runner: %w", err)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	return nil
}
