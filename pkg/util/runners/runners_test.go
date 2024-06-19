// SPDX-License-Identifier: MIT

package runners

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

func TestRunnersToDelete(t *testing.T) {
	garmRunners := []params.Instance{
		{
			Name:         "runner-active-running-1",
			RunnerStatus: params.RunnerActive,
			Status:       garmProviderParams.InstanceRunning,
			UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
		},
		{
			Name:         "runner-active-running-2",
			RunnerStatus: params.RunnerActive,
			Status:       garmProviderParams.InstanceRunning,
			UpdatedAt:    time.Now().Add(-35 * time.Minute).Round(time.Minute),
		},
		{
			Name:         "runner-active-running-3",
			RunnerStatus: params.RunnerActive,
			Status:       garmProviderParams.InstanceRunning,
			UpdatedAt:    time.Now().Add(-40 * time.Minute).Round(time.Minute),
		},
		{
			Name:         "runner-idle-running-1",
			RunnerStatus: params.RunnerIdle,
			Status:       garmProviderParams.InstanceRunning,
			UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
		},
		{
			Name:         "runner-idle-running-2",
			RunnerStatus: params.RunnerIdle,
			Status:       garmProviderParams.InstanceRunning,
			UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
		},
		{
			Name:         "runner-idle-running-3",
			RunnerStatus: params.RunnerIdle,
			Status:       garmProviderParams.InstanceRunning,
			UpdatedAt:    time.Now().Add(-25 * time.Minute).Round(time.Minute),
		},
		{
			Name:         "runner-installing-running-1",
			RunnerStatus: params.RunnerInstalling,
			Status:       garmProviderParams.InstanceRunning,
			UpdatedAt:    time.Now().Add(-15 * time.Minute).Round(time.Minute),
		},
		{
			Name:         "runner-installing-running-2",
			RunnerStatus: params.RunnerInstalling,
			Status:       garmProviderParams.InstanceRunning,
			UpdatedAt:    time.Now().Add(-2 * time.Minute).Round(time.Minute),
		},
		{
			Name:         "runner-installing-pending-create-1",
			RunnerStatus: params.RunnerInstalling,
			Status:       garmProviderParams.InstancePendingCreate,
			UpdatedAt:    time.Now().Add(-2 * time.Minute).Round(time.Minute),
		},
		{
			Name:         "runner-installing-error-1",
			RunnerStatus: params.RunnerInstalling,
			Status:       garmProviderParams.InstanceError,
			UpdatedAt:    time.Now().Add(-2 * time.Minute).Round(time.Minute),
		},
	}

	type args struct {
		minIdleRunners int
		minRunnerAge   time.Duration
		allRunners     []params.Instance
		ctx            context.Context
	}
	tests := []struct {
		name string
		args args
		want []params.Instance
	}{
		{
			name: "delete all runners in a deletable state",
			args: args{
				minIdleRunners: 0,
				minRunnerAge:   10 * time.Minute,
				allRunners:     garmRunners,
				ctx:            context.Background(),
			},
			want: []params.Instance{
				{
					Name:         "runner-active-running-1",
					RunnerStatus: params.RunnerActive,
					Status:       garmProviderParams.InstanceRunning,
					UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
				},
				{
					Name:         "runner-active-running-2",
					RunnerStatus: params.RunnerActive,
					Status:       garmProviderParams.InstanceRunning,
					UpdatedAt:    time.Now().Add(-35 * time.Minute).Round(time.Minute),
				},
				{
					Name:         "runner-active-running-3",
					RunnerStatus: params.RunnerActive,
					Status:       garmProviderParams.InstanceRunning,
					UpdatedAt:    time.Now().Add(-40 * time.Minute).Round(time.Minute),
				},
				{
					Name:         "runner-idle-running-1",
					RunnerStatus: params.RunnerIdle,
					Status:       garmProviderParams.InstanceRunning,
					UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
				},
				{
					Name:         "runner-idle-running-2",
					RunnerStatus: params.RunnerIdle,
					Status:       garmProviderParams.InstanceRunning,
					UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
				},
				{
					Name:         "runner-idle-running-3",
					RunnerStatus: params.RunnerIdle,
					Status:       garmProviderParams.InstanceRunning,
					UpdatedAt:    time.Now().Add(-25 * time.Minute).Round(time.Minute),
				},
				{
					Name:         "runner-installing-running-1",
					RunnerStatus: params.RunnerInstalling,
					Status:       garmProviderParams.InstanceRunning,
					UpdatedAt:    time.Now().Add(-15 * time.Minute).Round(time.Minute),
				},
				{
					Name:         "runner-installing-running-2",
					RunnerStatus: params.RunnerInstalling,
					Status:       garmProviderParams.InstanceRunning,
					UpdatedAt:    time.Now().Add(-2 * time.Minute).Round(time.Minute),
				},
				{
					Name:         "runner-installing-error-1",
					RunnerStatus: params.RunnerInstalling,
					Status:       garmProviderParams.InstanceError,
					UpdatedAt:    time.Now().Add(-2 * time.Minute).Round(time.Minute),
				},
			},
		},
		{
			name: "pool currently scale runners up - do not delete any runners",
			args: args{
				minIdleRunners: 4,
				minRunnerAge:   10 * time.Minute,
				allRunners:     garmRunners,
				ctx:            context.Background(),
			},
			want: []params.Instance{},
		},
		{
			name: "pool currently scale runners up - we scale pool down to 2 runners - delete runners ",
			args: args{
				minIdleRunners: 2,
				minRunnerAge:   10 * time.Minute,
				allRunners:     garmRunners,
				ctx:            context.Background(),
			},
			want: []params.Instance{
				{
					Name:         "runner-idle-running-1",
					RunnerStatus: params.RunnerIdle,
					Status:       garmProviderParams.InstanceRunning,
					UpdatedAt:    time.Now().Add(-30 * time.Minute).Round(time.Minute),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/*
				some cases:
					- minIdleRunners := 0
					  should delete all runners which are in a deletable state
					  no matter how old they are

					- minIdleRunners := >0
					  should delete all runners which are in a deletable state
					  and are older than minRunnerAge
			*/

			var runners []params.Instance

			// idleRunners := IdleRunners(tt.args.ctx, tt.args.allRunners)
			// oldIdleRunners := OldIdleRunners(tt.args.minRunnerAge, idleRunners)

			if tt.args.minIdleRunners == 0 {
				runners = DeletableRunners(tt.args.ctx, tt.args.allRunners)
			} else {
				// let's first extract all runners which are older than minRunnerAge

				// let's first extract all the runners which have to be deleted
				idleRunners := IdleRunners(tt.args.ctx, tt.args.allRunners)
				oldIdleRunners := OldIdleRunners(tt.args.minRunnerAge, idleRunners)
				alignedRunners := AlignIdleRunners(tt.args.minIdleRunners, oldIdleRunners)

				//
				runners = DeletableRunners(tt.args.ctx, alignedRunners)
			}

			if !reflect.DeepEqual(runners, tt.want) {
				for _, runner := range runners {
					t.Logf("runner: %v", runner.Name)
				}
				t.Errorf("AlignIdleRunners() = %v, want %v", runners, tt.want)
			}
		})
	}
}
