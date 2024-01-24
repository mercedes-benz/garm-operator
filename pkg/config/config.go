// SPDX-License-Identifier: MIT

package config

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/iancoleman/strcase"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type GarmConfig struct {
	Server   string `koanf:"server" validate:"required,url" yaml:"server"`
	Username string `koanf:"username" validate:"required" yaml:"username"`
	Password string `koanf:"password" validate:"required" yaml:"password"`
	Init     bool   `koanf:"init" yaml:"init"`
	Email    string `koanf:"email" validate:"required_if=Init true" yaml:"email"`
}

type OperatorConfig struct {
	MetricsBindAddress      string        `koanf:"metricsBindAddress" validate:"required,hostname_port" yaml:"metricsBindAddress"`
	HealthProbeBindAddress  string        `koanf:"healthProbeBindAddress" validate:"required,hostname_port" yaml:"healthProbeBindAddress"`
	LeaderElection          bool          `koanf:"leaderElection" yaml:"leaderElection"`
	SyncPeriod              time.Duration `koanf:"syncPeriod" validate:"required" yaml:"syncPeriod"`
	WatchNamespace          string        `koanf:"watchNamespace" yaml:"watchNamespace"`
	SyncRunnersInterval     time.Duration `koanf:"syncRunnersInterval" validate:"gte=5s,lte=5m" yaml:"syncRunnersInterval"`
	MinIdleRunnersAge       time.Duration `koanf:"minIdleRunnersAge" yaml:"minIdleRunnersAge"`
	RunnerConcurrency       int           `koanf:"runnerConcurrency" validate:"gte=1" yaml:"runnerConcurrency"`
	RepositoryConcurrency   int           `koanf:"repositoryConcurrency" validate:"gte=1" yaml:"repositoryConcurrency"`
	EnterpriseConcurrency   int           `koanf:"enterpriseConcurrency" validate:"gte=1" yaml:"enterpriseConcurrency"`
	OrganizationConcurrency int           `koanf:"organizationConcurrency" validate:"gte=1" yaml:"organizationConcurrency"`
	PoolConcurrency         int           `koanf:"poolConcurrency" validate:"gte=1" yaml:"poolConcurrency"`
	RunnerReconciliation    bool          `koanf:"runnerReconciliation" yaml:"runnerReconciliation"`
	LogVerbosityLevel       int           `koanf:"logVerbosityLevel" validate:"gte=0,lte=5" yaml:"logVerbosityLevel"`
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
		// Transform env e.g. from OPERATOR_SYNC_PERIOD to operator.syncPeriod (camel case)
		key := strings.SplitN(s, "_", 2)

		for i := range key {
			key[i] = strcase.ToLowerCamel(key[i])
		}

		res := strings.Join(key, ".")

		return res
	}), nil)

	// load config from envs with prefix GARM_
	k.Load(env.Provider("GARM_", ".", func(s string) string {
		// Transform env e.g. from GARM_SERVER to garm.server (camel case)
		key := strings.SplitN(s, "_", 2)

		for i := range key {
			key[i] = strcase.ToLowerCamel(key[i])
		}

		res := strings.Join(key, ".")

		return res
	}), nil)

	// load config from flags
	if f != nil {
		k.Load(posflag.ProviderWithFlag(f, ".", k, func(pf *pflag.Flag) (string, interface{}) {
			// Transform flag e.g. from operator-sync-period to operator.syncPeriod (camel case)
			key := strings.SplitN(pf.Name, "-", 2)

			val := posflag.FlagVal(f, pf)

			// Check array length to prevent failing if flag consists only of a single string (e.g. config flag)
			if len(key) == 2 {
				for i := range key {
					key[i] = strcase.ToLowerCamel(key[i])
				}
				res := strings.Join(key, ".")
				return res, val
			}

			res := key[0]

			return res, val
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
