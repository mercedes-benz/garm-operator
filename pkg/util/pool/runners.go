// SPDX-License-Identifier: MIT

package pool

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

func GetAllRunners(ctx context.Context, pool *garmoperatorv1alpha1.Pool, instanceClient garmClient.InstanceClient) ([]params.Instance, error) {
	log := log.FromContext(ctx)
	log.Info("discover idle runners", "pool", pool.Name)

	runners, err := instanceClient.ListPoolInstances(
		instances.NewListPoolInstancesParams().WithPoolID(pool.Status.ID))
	if err != nil {
		return nil, err
	}

	return runners.Payload, nil
}

// ExtractIdleRunners returns a list of runners that are in a state that allows deletion
func ExtractIdleRunners(ctx context.Context, instances []params.Instance) []params.Instance {
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

func ExtractDeletableRunners(ctx context.Context, instances []params.Instance) []params.Instance {
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

// ExtractOldIdleRunners returns a list of runners that are older than minRunnerAge
func ExtractOldIdleRunners(minRunnerAge time.Duration, instances []params.Instance) []params.Instance {
	oldIdleRunners := []params.Instance{}

	// filter runners that are in state that allows deletion
	for _, runner := range instances {
		if time.Since(runner.UpdatedAt) > minRunnerAge {
			oldIdleRunners = append(oldIdleRunners, runner)
		}
	}
	return oldIdleRunners
}

// AlignIdleRunners scales down the pool to the desired state
// of minIdleRunners. It will delete runners in a deletable state
func AlignIdleRunners(ctx context.Context, pool *garmoperatorv1alpha1.Pool, idleRunners []params.Instance, instanceClient garmClient.InstanceClient) error {
	log := log.FromContext(ctx)

	/*

		alignIdleRunners

			poolMin 20: 20 runners in a deletable state
				scale down to 10
				10 runners to delete

			poolMin 20: 8 runners in a deletable state
				scale down to 10
				8 - 10 = -2 runners to delete

			poolMin 10: 10 runners in a deletable state
				scale down to 0
				10 runners to delete

	*/

	// calculate how many runners need to be deleted
	var removableRunnersCount int
	// if there are more runners than minIdleRunners, delete the difference
	if len(idleRunners)-int(pool.Spec.MinIdleRunners) > 0 {
		removableRunnersCount = len(idleRunners) - int(pool.Spec.MinIdleRunners)
	} else {
		removableRunnersCount = len(idleRunners)
	}

	// this is where the status.idleRunners comparison needs to be done
	// if real state is larger than desired state - scale down runners
	for i, runner := range idleRunners {
		// do not delete more runners than minIdleRunners
		if i == removableRunnersCount {
			break
		}
		log.Info("remove runner", "runner", runner.Name, "state", runner.Status, "runner state", runner.RunnerStatus)

		err := instanceClient.DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName(runner.Name))
		if err != nil {
			log.Error(err, "unable to delete runner", "runner", runner.Name)
			return err
		}
	}
	log.Info("Successfully scaled pool down", "pool", pool.Name)
	return nil
}

// AlignIdleRunners scales down the pool to the desired state
// of minIdleRunners. It will delete runners in a deletable state
func ExtractDownscalableRunners(minIdleRunners int, idleRunners []params.Instance) []params.Instance {
	deletableRunners := []params.Instance{}

	// calculate how many runners need to be deleted
	var removableRunnersCount int
	// if there are more runners than minIdleRunners, delete the difference
	if len(idleRunners)-minIdleRunners > 0 {
		removableRunnersCount = len(idleRunners) - minIdleRunners
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
