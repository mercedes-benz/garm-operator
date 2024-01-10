// SPDX-License-Identifier: MIT

package defaults

import "time"

const (
	// default values for operator configuration
	DefaultMetricsBindAddress     = ":8080"
	DefaultHealthProbeBindAddress = ":8081"
	DefaultLeaderElection         = false
	DefaultSyncPeriod             = 5 * time.Minute
	DefaultWatchNamespace         = ""
	DefaultSyncRunnersInterval    = 5 * time.Second

	// default values for garm configuration
	DefaultGarmInit  = true
	DefaultGarmEmail = "garm-operator@localhost"

	// default values for controller concurrency configuration
	DefaultRunnerConcurrency       = 50
	DefaultRepositoryConcurrency   = 10
	DefaultEnterpriseConcurrency   = 1
	DefaultOrganizationConcurrency = 5
	DefaultPoolConcurrency         = 10

	// default values for controller reconciliation configuration
	DefaultRunnerReconciliation = false
)
