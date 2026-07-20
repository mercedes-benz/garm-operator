// SPDX-License-Identifier: MIT

package pools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/client/pools"
	"github.com/cloudbase/garm/client/repositories"
	"github.com/cloudbase/garm/params"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
)

func UpdatePool(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1beta1.Pool, image *garmoperatorv1beta1.Image) error {
	log := log.FromContext(ctx).
		WithName("UpdatePool")

	log.Info("updating pool", "pool", pool.Name, "id", pool.Status.ID)

	poolParams := params.UpdatePoolParams{
		RunnerPrefix: params.RunnerPrefix{
			Prefix: pool.Spec.RunnerPrefix,
		},
		MaxRunners:             &pool.Spec.MaxRunners,
		MinIdleRunners:         &pool.Spec.MinIdleRunners,
		Flavor:                 pool.Spec.Flavor,
		OSType:                 pool.Spec.OSType,
		OSArch:                 pool.Spec.OSArch,
		Tags:                   pool.Spec.Tags,
		Enabled:                &pool.Spec.Enabled,
		RunnerBootstrapTimeout: &pool.Spec.RunnerBootstrapTimeout,
		ExtraSpecs:             json.RawMessage([]byte(pool.Spec.ExtraSpecs)),
		GitHubRunnerGroup:      &pool.Spec.GitHubRunnerGroup,
	}
	if image != nil {
		poolParams.Image = image.Spec.Tag
	}

	_, err := garmClient.UpdatePool(pools.NewUpdatePoolParams().WithPoolID(pool.Status.ID).WithBody(poolParams))
	if err != nil {
		return err
	}

	return nil
}

func CreatePool(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1beta1.Pool, image *garmoperatorv1beta1.Image, gitHubScopeRef garmoperatorv1beta1.GitHubScope) (params.Pool, error) {
	log := log.FromContext(ctx).
		WithName("CreatePool")
	log.Info("creating pool", "pool", pool.Name)

	poolResult := params.Pool{}

	fmt.Printf("+%v\n", gitHubScopeRef)

	id := gitHubScopeRef.GetID()

	fmt.Printf("id: %s\n", id)
	fmt.Printf("kind: %s\n", gitHubScopeRef.GetKind())

	scope, err := garmoperatorv1beta1.ToGitHubScopeKind(gitHubScopeRef.GetKind())
	if err != nil {
		return poolResult, err
	}

	extraSpecs := json.RawMessage([]byte{})
	if pool.Spec.ExtraSpecs != "" {
		err := json.Unmarshal([]byte(pool.Spec.ExtraSpecs), &extraSpecs)
		if err != nil {
			return poolResult, err
		}
	}

	poolParams := params.CreatePoolParams{
		RunnerPrefix: params.RunnerPrefix{
			Prefix: pool.Spec.RunnerPrefix,
		},
		ProviderName:           pool.Spec.ProviderName,
		MaxRunners:             pool.Spec.MaxRunners,
		MinIdleRunners:         pool.Spec.MinIdleRunners,
		Image:                  image.Spec.Tag,
		Flavor:                 pool.Spec.Flavor,
		OSType:                 pool.Spec.OSType,
		OSArch:                 pool.Spec.OSArch,
		Tags:                   pool.Spec.Tags,
		Enabled:                pool.Spec.Enabled,
		RunnerBootstrapTimeout: pool.Spec.RunnerBootstrapTimeout,
		ExtraSpecs:             extraSpecs,
		GitHubRunnerGroup:      pool.Spec.GitHubRunnerGroup,
	}

	switch scope {
	case garmoperatorv1beta1.EnterpriseScope:
		result, err := garmClient.CreateEnterprisePool(
			enterprises.NewCreateEnterprisePoolParams().
				WithEnterpriseID(id).
				WithBody(poolParams))
		if err != nil {
			return params.Pool{}, err
		}

		poolResult = result.Payload
	case garmoperatorv1beta1.OrganizationScope:
		result, err := garmClient.CreateOrgPool(
			organizations.NewCreateOrgPoolParams().
				WithOrgID(id).
				WithBody(poolParams))
		if err != nil {
			return params.Pool{}, err
		}
		poolResult = result.Payload
	case garmoperatorv1beta1.RepositoryScope:
		result, err := garmClient.CreateRepoPool(
			repositories.NewCreateRepoPoolParams().
				WithRepoID(id).
				WithBody(poolParams))
		if err != nil {
			return params.Pool{}, err
		}
		poolResult = result.Payload
	default:
		err := fmt.Errorf("no valid scope specified: %s", scope)
		return params.Pool{}, err
	}

	return poolResult, nil
}

func GarmPoolExists(garmClient garmClient.PoolClient, pool *garmoperatorv1beta1.Pool) bool {
	result, err := garmClient.GetPool(pools.NewGetPoolParams().WithPoolID(pool.Status.ID))
	if err != nil {
		return false
	}
	return result.Payload.ID != ""
}
