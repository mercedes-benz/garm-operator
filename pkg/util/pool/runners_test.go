// SPDX-License-Identifier: MIT

package pool

import (
	"context"
	"reflect"
	"testing"
	"time"

	garmProviderParams "github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm/params"
)

func TestExtractOldIdleRunners(t *testing.T) {
	type args struct {
		minRunnerAge time.Duration
		instances    []params.Instance
	}
	tests := []struct {
		name string
		args args
		want []params.Instance
	}{
		{
			name: "no runners",
			args: args{
				minRunnerAge: 1 * time.Hour,
				instances:    []params.Instance{},
			},
			want: []params.Instance{},
		},
		{
			name: "no old idle runners",
			args: args{
				minRunnerAge: 1 * time.Hour,
				instances: []params.Instance{
					{
						RunnerStatus: params.RunnerIdle,
						UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
						Name:         "runner1",
					},
					{
						RunnerStatus: params.RunnerIdle,
						UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
						Name:         "runner2",
					},
				},
			},
			want: []params.Instance{},
		},
		{
			name: "two old idle runners",
			args: args{
				minRunnerAge: 10 * time.Minute,
				instances: []params.Instance{
					{
						RunnerStatus: params.RunnerIdle,
						UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
						Name:         "runner1",
					},
					{
						RunnerStatus: params.RunnerIdle,
						UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
						Name:         "runner2",
					},
					{
						RunnerStatus: params.RunnerIdle,
						UpdatedAt:    time.Now().Add(-2 * time.Minute).Round(time.Minute),
						Name:         "runner3",
					},
				},
			},
			want: []params.Instance{
				{
					RunnerStatus: params.RunnerIdle,
					UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
					Name:         "runner1",
				},
				{
					RunnerStatus: params.RunnerIdle,
					UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
					Name:         "runner2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OldIdleRunners(tt.args.minRunnerAge, tt.args.instances); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OldIdleRunners() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractIdleRunners(t *testing.T) {
	type args struct {
		ctx       context.Context
		instances []params.Instance
	}
	tests := []struct {
		name string
		args args
		want []params.Instance
	}{
		{
			name: "no runners",
			args: args{
				ctx:       context.Background(),
				instances: []params.Instance{},
			},
			want: []params.Instance{},
		},
		{
			name: "no idle runners",
			args: args{
				ctx: context.Background(),
				instances: []params.Instance{
					{
						RunnerStatus: params.RunnerActive,
						Status:       garmProviderParams.InstanceRunning,
						Name:         "runner1",
					},
					{
						RunnerStatus: params.RunnerInstalling,
						Status:       garmProviderParams.InstanceRunning,
						Name:         "runner2",
					},
				},
			},
			want: []params.Instance{},
		},
		{
			name: "idle runners",
			args: args{
				ctx: context.Background(),
				instances: []params.Instance{
					{
						RunnerStatus: params.RunnerActive,
						Status:       garmProviderParams.InstanceRunning,
						Name:         "runner1",
					},
					{
						RunnerStatus: params.RunnerInstalling,
						Status:       garmProviderParams.InstanceRunning,
						Name:         "runner2",
					},
					{
						RunnerStatus: params.RunnerIdle,
						Status:       garmProviderParams.InstanceRunning,
						Name:         "runner3",
					},
				},
			},
			want: []params.Instance{
				{
					RunnerStatus: params.RunnerIdle,
					Status:       garmProviderParams.InstanceRunning,
					Name:         "runner3",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IdleRunners(tt.args.ctx, tt.args.instances); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IdleRunners() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractDeletableRunners(t *testing.T) {
	type args struct {
		ctx       context.Context
		instances []params.Instance
	}
	tests := []struct {
		name string
		args args
		want []params.Instance
	}{
		{
			name: "no runners",
			args: args{
				ctx:       context.Background(),
				instances: []params.Instance{},
			},
			want: []params.Instance{},
		},
		{
			name: "no deletable runners",
			args: args{
				ctx: context.Background(),
				instances: []params.Instance{
					{
						Status: garmProviderParams.InstanceCreating,
						Name:   "runner1",
					},
					{
						Status: garmProviderParams.InstancePendingCreate,
						Name:   "runner2",
					},
					{
						Status: garmProviderParams.InstancePendingDelete,
						Name:   "runner3",
					},
				},
			},
			want: []params.Instance{},
		},
		{
			name: "deletable runners",
			args: args{
				ctx: context.Background(),
				instances: []params.Instance{
					{
						Status: garmProviderParams.InstanceRunning,
						Name:   "runner1",
					},
					{
						Status: garmProviderParams.InstancePendingCreate,
						Name:   "runner2",
					},
					{
						Status: garmProviderParams.InstanceError,
						Name:   "runner3",
					},
				},
			},
			want: []params.Instance{
				{
					Status: garmProviderParams.InstanceRunning,
					Name:   "runner1",
				},
				{
					Status: garmProviderParams.InstanceError,
					Name:   "runner3",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeletableRunners(tt.args.ctx, tt.args.instances); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeletableRunners() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractDownscalableRunners(t *testing.T) {
	type args struct {
		minIdleRunners int
		idleRunners    []params.Instance
	}
	tests := []struct {
		name string
		args args
		want []params.Instance
	}{
		{
			name: "delete all runners",
			args: args{
				minIdleRunners: 0,
				idleRunners: []params.Instance{
					{
						Name: "runner1",
					},
					{
						Name: "runner2",
					},
					{
						Name: "runner3",
					},
				},
			},
			want: []params.Instance{
				{
					Name: "runner1",
				},
				{
					Name: "runner2",
				},
				{
					Name: "runner3",
				},
			},
		},
		{
			name: "delete only half of the runners",
			args: args{
				minIdleRunners: 2,
				idleRunners: []params.Instance{
					{
						Name: "runner1",
					},
					{
						Name: "runner2",
					},
					{
						Name: "runner3",
					},
					{
						Name: "runner4",
					},
				},
			},
			want: []params.Instance{
				{
					Name: "runner1",
				},
				{
					Name: "runner2",
				},
			},
		},
		{
			name: "do not delete any runners",
			args: args{
				minIdleRunners: 2,
				idleRunners: []params.Instance{
					{
						Name: "runner1",
					},
				},
			},
			want: []params.Instance{},
		},
		{
			name: "scale down to 0",
			args: args{
				minIdleRunners: 0,
				idleRunners: []params.Instance{
					{
						Name: "runner1",
					},
				},
			},
			want: []params.Instance{
				{
					Name: "runner1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AlignIdleRunners(tt.args.minIdleRunners, tt.args.idleRunners); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AlignIdleRunners() = %v, want %v", got, tt.want)
			}
		})
	}
}
