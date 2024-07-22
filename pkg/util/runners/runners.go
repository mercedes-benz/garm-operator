// SPDX-License-Identifier: MIT

package runners

import (
	"context"
	"time"

	garmProviderParams "github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/params"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
)

func GetRunnersByPoolID(ctx context.Context, pool *garmoperatorv1alpha1.Pool, instanceClient garmClient.InstanceClient) ([]params.Instance, error) {
	log := log.FromContext(ctx)
	log.Info("discover idle runners", "pool", pool.Name)

	runners, err := instanceClient.ListPoolInstances(
		instances.NewListPoolInstancesParams().WithPoolID(pool.Status.ID))
	if err != nil {
		return nil, err
	}

	return runners.Payload, nil
}

// IdleRunners returns a list of runners that are in github state idle
func IdleRunners(ctx context.Context, instances []params.Instance) []params.Instance {
	log := log.FromContext(ctx)

	// create a list of "deletable runners"
	idleRunners := []params.Instance{}

	// filter runners that are idle
	for _, runner := range instances {
		// only consider runners that are idle
		switch runner.RunnerStatus {
		case params.RunnerIdle:
			idleRunners = append(idleRunners, runner)
		default:
			log.V(1).Info("Runner is not idle", "runner", runner.Name, "state", runner.Status)
		}
	}

	return idleRunners
}

// OldIdleRunners returns a list of runners that are older than minRunnerAge
func OldIdleRunners(minRunnerAge time.Duration, instances []params.Instance) []params.Instance {
	oldIdleRunners := []params.Instance{}

	// filter runners that are in state that allows deletion
	for _, runner := range instances {
		if time.Since(runner.UpdatedAt) > minRunnerAge {
			oldIdleRunners = append(oldIdleRunners, runner)
		}
	}
	return oldIdleRunners
}

// DeletableRunners returns a list of runners that are in a deletable state from a garm perspective
func DeletableRunners(ctx context.Context, instances []params.Instance) []params.Instance {
	log := log.FromContext(ctx)

	// create a list of "deletable runners"
	deletableRunners := []params.Instance{}

	// filter runners that are in state that allows deletion
	for _, runner := range instances {
		// only consider runners that are in a deletable state from a garm perspective
		switch runner.Status {
		case garmProviderParams.InstanceRunning, garmProviderParams.InstanceError:
			deletableRunners = append(deletableRunners, runner)
		default:
			log.V(1).Info("Runner is in state that does not allow deletion", "runner", runner.Name, "state", runner.Status)
		}
	}

	return deletableRunners
}

// AlignIdleRunners scales down the pool to the desired state of minIdleRunners.
// It will delete as many runners as needed to reach the desired state.
func AlignIdleRunners(minIdleRunners int, idleRunners []params.Instance) []params.Instance {
	deletableRunners := []params.Instance{}

	removableRunnersCount := len(idleRunners) - minIdleRunners

	// set to 0 if negative
	if removableRunnersCount < 0 {
		removableRunnersCount = 0
	}

	// this is where the status.idleRunners comparison needs to be done
	// if real state is larger than desired state - scale down runners
	for i, runner := range idleRunners {
		// do not delete more runners than minIdleRunners
		if i == removableRunnersCount {
			break
		}
		deletableRunners = append(deletableRunners, runner)
	}
	return deletableRunners
}
