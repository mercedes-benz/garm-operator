// SPDX-License-Identifier: MIT

package config

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type GarmConfig struct {
	Server   string `koanf:"server" validate:"required,url"`
	Username string `koanf:"username" validate:"required"`
	Password string `koanf:"password" validate:"required"`
	Init     bool   `koanf:"init"`
	Email    string `koanf:"email" validate:"required_if=Init true"`
}

type OperatorConfig struct {
	MetricsBindAddress     string        `koanf:"metrics_bind_address" validate:"required,hostname_port"`
	HealthProbeBindAddress string        `koanf:"health_probe_bind_address" validate:"required,hostname_port"`
	LeaderElection         bool          `koanf:"leader_election"`
	SyncPeriod             time.Duration `koanf:"sync_period" validate:"required"`
	WatchNamespace         string        `koanf:"watch_namespace"`
	SyncRunnersInterval    time.Duration `koanf:"sync_runners_interval" validate:"gte=5s,lte=5m"`
}

type AppConfig struct {
	Garm     GarmConfig     `koanf:"garm"`
	Operator OperatorConfig `koanf:"operator"`
}

var Config AppConfig

func GenerateConfig(f *pflag.FlagSet, configFile string) error {
	// create koanf instance
	k := koanf.New(".")

	// load config from envs with prefix OPERATOR_
	k.Load(env.Provider("OPERATOR_", ".", func(s string) string {
		// Transform env e.g. from OPERATOR_SYNC_PERIOD to operator.syncperiod
		key := strings.Replace(strings.ToLower(s), "_", ".", 1)
		return key
	}), nil)

	// load config from envs with prefix GARM_
	k.Load(env.Provider("GARM_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", 1)
	}), nil)

	// load config from flags
	if f != nil {
		k.Load(posflag.ProviderWithFlag(f, ".", k, func(pf *pflag.Flag) (string, interface{}) {
			// Transform flag e.g. from operator-sync-period to operator.syncperiod
			key := strings.Replace(pf.Name, "-", ".", 1)
			key2 := strings.ReplaceAll(key, "-", "_")

			// Use FlagVal() and then transform the value, or don't use it at all
			// and add custom logic to parse the value.
			val := posflag.FlagVal(f, pf)

			return key2, val
		}), nil)
	}

	// load config from file
	if configFile != "" {
		if err := k.Load(file.Provider(f.Lookup("config").Value.String()), yaml.Parser()); err != nil {
			return errors.Wrap(err, "failed to load config file")
		}
	}

	// unmarshal all koanf config keys into AppConfig struct
	if err := k.Unmarshal("", &Config); err != nil {
		return errors.Wrap(err, "failed to unmarshal config")
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(&Config); err != nil {
		return errors.Wrap(err, "invalid config: set with env, flag or in config file")
	}

	return nil
}
