// SPDX-License-Identifier: MIT

package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/mercedes-benz/garm-operator/pkg/flags"
)

func TestGenerateConfig(t *testing.T) {
	f := flags.InitiateFlags()

	tests := []struct {
		name    string
		wantErr bool
		wantCfg AppConfig
		flags   map[string]string
		envvars map[string]string
	}{
		{
			name:    "expectedFailureGarmConfig",
			wantErr: true,
		},
		{
			name: "ConfigFromDefaultAndEnvs",
			envvars: map[string]string{
				"GARM_SERVER":                    "http://localhost:9997",
				"GARM_USERNAME":                  "admin",
				"GARM_PASSWORD":                  "password",
				"OPERATOR_SYNC_RUNNERS_INTERVAL": "20s",
			},
			wantCfg: AppConfig{
				Operator: OperatorConfig{
					MetricsBindAddress:     ":8080",
					HealthProbeBindAddress: ":8081",
					LeaderElection:         false,
					SyncPeriod:             5 * time.Minute,
					WatchNamespace:         "",
					SyncRunnersInterval:    20 * time.Second,
				},
				Garm: GarmConfig{
					Server:   "http://localhost:9997",
					Username: "admin",
					Password: "password",
					Init:     true,
					Email:    "garm-operator@localhost",
				},
			},
		},
		{
			name: "ConfigFromDefaultAndFlags",
			flags: map[string]string{
				"garm-server":   "http://localhost:9997",
				"garm-username": "admin",
				"garm-password": "password",
			},
			wantCfg: AppConfig{
				Operator: OperatorConfig{
					MetricsBindAddress:     ":8080",
					HealthProbeBindAddress: ":8081",
					LeaderElection:         false,
					SyncPeriod:             5 * time.Minute,
					WatchNamespace:         "",
					SyncRunnersInterval:    5 * time.Second,
				},
				Garm: GarmConfig{
					Server:   "http://localhost:9997",
					Username: "admin",
					Password: "password",
					Init:     true,
					Email:    "garm-operator@localhost",
				},
			},
		},
		{
			name: "ConfigFromDefaultEnvsAndFlags",
			envvars: map[string]string{
				"GARM_SERVER":                    "http://localhost:1234",
				"GARM_USERNAME":                  "admin1234",
				"GARM_PASSWORD":                  "password1234",
				"OPERATOR_SYNC_RUNNERS_INTERVAL": "20s",
			},
			flags: map[string]string{
				"garm-server":                    "http://localhost:9997",
				"garm-username":                  "admin",
				"garm-password":                  "password",
				"operator-sync-runners-interval": "10s",
			},
			wantCfg: AppConfig{
				Operator: OperatorConfig{
					MetricsBindAddress:     ":8080",
					HealthProbeBindAddress: ":8081",
					LeaderElection:         false,
					SyncPeriod:             5 * time.Minute,
					WatchNamespace:         "",
					SyncRunnersInterval:    10 * time.Second,
				},
				Garm: GarmConfig{
					Server:   "http://localhost:9997",
					Username: "admin",
					Password: "password",
					Init:     true,
					Email:    "garm-operator@localhost",
				},
			},
		},
		{
			name: "ConfigFromEnvsFlagsAndFile",
			envvars: map[string]string{
				"GARM_SERVER":   "http://localhost:9997",
				"GARM_USERNAME": "admin",
				"GARM_PASSWORD": "password",
			},
			flags: map[string]string{
				"operator-metrics-bind-address": ":1234",
				"config":                        "test_config.yaml",
			},
			wantCfg: AppConfig{
				Operator: OperatorConfig{
					MetricsBindAddress:     ":7000",
					HealthProbeBindAddress: ":7001",
					LeaderElection:         true,
					SyncPeriod:             10 * time.Minute,
					WatchNamespace:         "garm-operator-namespace",
					SyncRunnersInterval:    15 * time.Second,
				},
				Garm: GarmConfig{
					Server:   "http://garm-server:9997",
					Username: "garm-username",
					Password: "garm-password",
					Init:     true,
					Email:    "garm-operator@localhost",
				},
			},
		},
		{
			name:    "Invalid Polling Interval Config, less than or equal to 5min",
			wantErr: true,
			envvars: map[string]string{
				"GARM_SERVER":   "http://localhost:9997",
				"GARM_USERNAME": "admin",
				"GARM_PASSWORD": "password",
			},
			flags: map[string]string{
				"operator-metrics-bind-address":  ":1234",
				"operator-sync-runners-interval": "10m",
			},
			wantCfg: AppConfig{
				Operator: OperatorConfig{
					MetricsBindAddress:     ":7000",
					HealthProbeBindAddress: ":7001",
					LeaderElection:         true,
					SyncPeriod:             10 * time.Minute,
					WatchNamespace:         "garm-operator-namespace",
					SyncRunnersInterval:    5 * time.Second,
				},
				Garm: GarmConfig{
					Server:   "http://garm-server:9997",
					Username: "garm-username",
					Password: "garm-password",
					Init:     true,
					Email:    "garm-operator@localhost",
				},
			},
		},
		{
			name:    "Invalid Polling Interval Config, greater than or equal to 5s",
			wantErr: true,
			envvars: map[string]string{
				"GARM_SERVER":   "http://localhost:9997",
				"GARM_USERNAME": "admin",
				"GARM_PASSWORD": "password",
			},
			flags: map[string]string{
				"operator-metrics-bind-address":  ":1234",
				"operator-sync-runners-interval": "1s",
			},
			wantCfg: AppConfig{
				Operator: OperatorConfig{
					MetricsBindAddress:     ":7000",
					HealthProbeBindAddress: ":7001",
					LeaderElection:         true,
					SyncPeriod:             10 * time.Minute,
					WatchNamespace:         "garm-operator-namespace",
					SyncRunnersInterval:    5 * time.Second,
				},
				Garm: GarmConfig{
					Server:   "http://garm-server:9997",
					Username: "garm-username",
					Password: "garm-password",
					Init:     true,
					Email:    "garm-operator@localhost",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set env vars
			for k, v := range tt.envvars {
				t.Setenv(k, v)
			}

			var configFile string

			for k, v := range tt.flags {
				f.Set(k, v)
				if k == "config" {
					configFile = v
				}
			}

			if err := GenerateConfig(f, configFile); err != nil {
				if tt.wantErr {
					t.Logf("want error: %s", err.Error())
					return
				}
				t.Errorf("GenerateConfig() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(Config, tt.wantCfg) {
				t.Errorf("GenerateConfig() Config = \n%+v\n, wantCfg = \n%+v\n", Config, tt.wantCfg)
			}
		})
	}
}
