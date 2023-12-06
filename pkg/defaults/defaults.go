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

	// default values for garm configuration
	DefaultGarmInit  = true
	DefaultGarmEmail = "garm-operator@localhost"
)
