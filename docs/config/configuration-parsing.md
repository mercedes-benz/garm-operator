<!-- SPDX-License-Identifier: MIT -->

# Configuration Parsing

The configuration parsing for the Garm Operator is implemented with [koanf](https://github.com/knadh/koanf).

The configuration can be defined with `ENVs`, `Flags` and `config file (yaml)`.

<!-- toc -->
- [Parsing Order](#parsing-order)
- [ENVs](#envs)
- [Flags](#flags)
  - [Additional Flags](#additional-flags)
- [Config File (yaml)](#config-file-yaml)
- [Configuration Default Values](#configuration-default-values)
- [Parsing Validation](#parsing-validation)
<!-- /toc -->

## Parsing Order

Koanf does not specify any order of priority for the various configuration options.

The order is determined by the order in which the Read() function is called.

For the Garm Operator the following order is defined, which is to be considered in ascending order from lowest to highest priority:

1. Defined default values ([see section configuration default values](#configuration-default-values))
1. ENVs
1. Flags
1. Config File (yaml)

## ENVs

All ENVs with the `OPERATOR_` and `GARM_` prefix will be merged by koanf. However, only the following ENVs will be parsed:

```
GARM_SERVER
GARM_USERNAME
GARM_PASSWORD
GARM_INIT
GARM_EMAIL

OPERATOR_METRICS_BIND_ADDRESS
OPERATOR_HEALTH_PROBE_BIND_ADDRESS
OPERATOR_LEADER_ELECTION
OPERATOR_SYNC_PERIOD
OPERATOR_WATCH_NAMESPACE
OPERATOR_SYNC_RUNNERS_INTERVAL
OPERATOR_MIN_IDLE_RUNNERS_AGE

OPERATOR_RUNNER_CONCURRENCY
OPERATOR_REPOSITORY_CONCURRENCY
OPERATOR_ORGANIZATION_CONCURRENCY
OPERATOR_ENTERPRISE_CONCURRENCY
OPERATOR_POOL_CONCURRENCY

OPERATOR_RUNNER_RECONCILATION

OPERATOR_LOG_VERBOSITY_LEVEL
```

## Flags

The following flags will be parsed and can be found in the [flags package](../../pkg/flags/flags.go):

```
--garm-server
--garm-username
--garm-password
--garm-init
--garm-email

--operator-metrics-bind-address
--operator-health-probe-bind-address
--operator-leader-election
--operator-sync-period
--operator-watch-namespace
--operator-sync-runners-interval
--operator-min-idle-runners-age

--operator-runner-concurrency
--operator-repository-concurrency
--operator-organization-concurrency
--operator-enterprise-concurrency
--operator-pool-concurrency

--operator-runner-reconcilation

--operator-log-verbosity-level
```

### Additional Flags

In addition to the previously mentioned flags, there are two additional flags:

```
--config
--dry-run
```

The `--config` flag can be set to specify the path to the `config file (yaml)` which contains the configuration ([see section config file (yaml)](#config-file-yaml)).

The `--dry-run` flag can be set to show the parsed configuration, without starting the Garm Operator. The output can be similar to the following:

```
generated Config as yaml:
garm:
  server: http://garm-server:9997
  username: admin
  password: 123456789
  init: false
  email: ""
operator:
  metricsBindAddress: :8080
  healthProbeBindAddress: :8081
  leaderElection: false
  syncPeriod: 5m0s
  watchNamespace: garm-operator-system
  syncRunnersInterval: 5m0s
  minIdleRunnersAge: 5m0s
  runnerConcurrency: 20
  repositoryConcurrency: 5
  organizationConcurrency: 3
  enterpriseConcurrency: 1
  poolConcurrency: 10
  runnerReconcilation: true
  logVerbosityLevel: 0
```

## Config File (yaml)

The following keys in the config file (yaml) will be parsed:

```yaml
# config.yaml
garm:
  server: "http://garm-server:9997"
  username: "garm-username"
  password: "garm-password"
  init: false
  email: ""

operator:
  metricsBindAddress: ":7000"
  healthProbeBindAddress: ":7001"
  leaderElection: true
  syncPeriod: "10m"
  watchNamespace: "garm-operator-namespace"
  syncRunnersInterval: "5m"
  minIdleRunnersAge: "5m"
  runnerConcurrency: 20
  repositoryConcurrency: 5
  organizationConcurrency: 3
  enterpriseConcurrency: 1
  poolConcurrency: 10
  runnerReconcilation: true
  logVerbosityLevel: 0
```

## Configuration Default Values

The defined default values for the configuration can be found in the [defaults package](../../pkg/defaults/defaults.go).

## Parsing Validation

After the configuration has been parsed by koanf and unmarshalled into a struct, the [validator](https://github.com/go-playground/validator) checks whether the generated struct is valid or not.

For example, if the `Garm Username` is not set, the following error message is returned by the validator:

```
setup "msg"="failed to read config" "error"="invalid config: set with env, flag or in config file: Key: 'AppConfig.Garm.Username' Error:Field validation for 'Username' failed on the 'required' tag"
```
