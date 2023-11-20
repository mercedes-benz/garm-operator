// SPDX-License-Identifier: MIT

package flags

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/mercedes-benz/garm-operator/pkg/defaults"
)

func InitiateFlags() *pflag.FlagSet {
	f := pflag.NewFlagSet("config", pflag.PanicOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// configure f for koanf
	f.String("config", "", "path to .yaml config file")
	f.String("operator-metrics-bind-address", defaults.DefaultMetricsBindAddress, "The address the metric endpoint binds to.")
	f.String("operator-health-probe-bind-address", defaults.DefaultHealthProbeBindAddress, "The address the probe endpoint binds to.")
	f.Bool("operator-leader-elect", defaults.DefaultLeaderElect, "Enable leader election for controller manager. "+"Enabling this will ensure there is only one active controller manager.")
	f.Duration("operator-sync-period", defaults.DefaultSyncPeriod, "The minimum interval at which watched resources are reconciled (e.g. 5m)")
	f.String("operator-namespace", defaults.DefaultNamespace, "Namespace that the controller watches to reconcile garm objects. "+"If unspecified, the controller watches for garm objects across all namespaces.")
	f.Bool("operator-create-webhook", defaults.DefaultCreateWebhook, "If true, create a validating webhook for garm objects.")

	f.String("garm-server", "", "The address of the GARM server")
	f.String("garm-username", "", "The username for the GARM server")
	f.String("garm-password", "", "The password for the GARM server")

	f.Bool("dry-run", false, "If true, only print the object that would be sent, without sending it.")

	f.Parse(os.Args[1:])

	return f
}
