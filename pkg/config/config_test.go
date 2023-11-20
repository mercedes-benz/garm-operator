// SPDX-License-Identifier: MIT

package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/mercedes-benz/garm-operator/pkg/flags"
)

func TestReadConfig(t *testing.T) {
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
				"GARM_SERVER":   "http://localhost:9997",
				"GARM_USERNAME": "admin",
				"GARM_PASSWORD": "password",
			},
			wantCfg: AppConfig{
				Operator: OperatorConfig{
					MetricsBindAddress:     ":8080",
					HealthProbeBindAddress: ":8081",
					LeaderElection:         false,
					SyncPeriod:             5 * time.Minute,
					WatchNamespace:         "",
				},
				Garm: GarmConfig{
					Server:   "http://localhost:9997",
					Username: "admin",
					Password: "password",
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
				},
				Garm: GarmConfig{
					Server:   "http://localhost:9997",
					Username: "admin",
					Password: "password",
				},
			},
		},
		{
			name: "ConfigFromDefaultEnvsAndFlags",
			envvars: map[string]string{
				"GARM_SERVER":   "http://localhost:1234",
				"GARM_USERNAME": "admin1234",
				"GARM_PASSWORD": "password1234",
			},
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
				},
				Garm: GarmConfig{
					Server:   "http://localhost:9997",
					Username: "admin",
					Password: "password",
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
				},
				Garm: GarmConfig{
					Server:   "http://garm-server:9997",
					Username: "garm-username",
					Password: "garm-password",
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

			if err := ReadConfig(f, configFile); err != nil {
				if tt.wantErr {
					return
				}
				t.Errorf("ReadConfig() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(Config, tt.wantCfg) {
				t.Errorf("ReadConfig() Config = \n%+v\n, wantCfg = \n%+v\n", Config, tt.wantCfg)
			}
		})
	}
}
