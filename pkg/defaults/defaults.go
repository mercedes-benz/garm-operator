// SPDX-License-Identifier: MIT

package defaults

import "time"

const (
	// default values for operator configuration
	DefaultMetricsBindAddress     = ":8080"
	DefaultHealthProbeBindAddress = ":8081"
	DefaultLeaderElect            = false
	DefaultSyncPeriod             = 5 * time.Minute
	DefaultNamespace              = ""
	DefaultCreateWebhook          = false
)
