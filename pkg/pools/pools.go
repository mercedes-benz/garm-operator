// SPDX-License-Identifier: MIT

package pools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/client/pools"
	"github.com/cloudbase/garm/client/repositories"
	"github.com/cloudbase/garm/params"
	"sigs.k8s.io/controller-runtime/pkg/log"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	garmClient "github.com/mercedes-benz/garm-operator/pkg/client"
	"github.com/mercedes-benz/garm-operator/pkg/filter"
	imageUtil "github.com/mercedes-benz/garm-operator/pkg/image"
)

func GetGarmPoolBySpecs(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, image *garmoperatorv1alpha1.Image, gitHubScopeRef garmoperatorv1alpha1.GitHubScope) (*params.Pool, error) {
	log := log.FromContext(ctx)
	log.Info("Getting existing garm pools by pool.spec")

	githubScopeRefID := gitHubScopeRef.GetID()
	githubScopeRefName := gitHubScopeRef.GetName()
	scope, err := garmoperatorv1alpha1.ToGitHubScopeKind(gitHubScopeRef.GetKind())
	if err != nil {
		return nil, err
	}

	garmPools, err := garmClient.ListAllPools(pools.NewListPoolsParams())
	if err != nil {
		return nil, err
	}

	filteredGarmPools := filter.Match(garmPools.Payload,
		MatchesImage(image.Spec.Tag),
		MatchesFlavor(pool.Spec.Flavor),
		MatchesProvider(pool.Spec.ProviderName),
		MatchesGitHubScope(scope, githubScopeRefID),
	)

	log.WithValues("image", image.Spec.Tag,
		"flavor", pool.Spec.Flavor,
		"provider", pool.Spec.ProviderName,
		"scope", scope,
		"githubScopeRefId", githubScopeRefID,
		"githubScopeRefName", githubScopeRefName,
	).Info(fmt.Sprintf("%d garm pools with same spec found", len(filteredGarmPools)))

	//nolint TODO: @rafalgalaw - can this happen?
	// i guess it's blocked by the fact that we can't create a pool with the same spec on garm side
	if len(filteredGarmPools) > 1 {
		return nil, errors.New("can not create pool, multiple instances matching flavor, image and provider found in garm")
	}

	// pool with the same specs already exists
	// return the first object in the list
	if len(filteredGarmPools) == 1 {
		return &filteredGarmPools[0], nil
	}

	// create
	return nil, nil
}

func UpdatePool(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, image *garmoperatorv1alpha1.Image) error {
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

func CreatePool(ctx context.Context, garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool, image *garmoperatorv1alpha1.Image, gitHubScopeRef garmoperatorv1alpha1.GitHubScope) (params.Pool, error) {
	log := log.FromContext(ctx).
		WithName("CreatePool")
	log.Info("creating pool", "pool", pool.Name)

	poolResult := params.Pool{}

	id := gitHubScopeRef.GetID()
	scope, err := garmoperatorv1alpha1.ToGitHubScopeKind(gitHubScopeRef.GetKind())
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
	case garmoperatorv1alpha1.EnterpriseScope:
		result, err := garmClient.CreateEnterprisePool(
			enterprises.NewCreateEnterprisePoolParams().
				WithEnterpriseID(id).
				WithBody(poolParams))
		if err != nil {
			return params.Pool{}, err
		}

		poolResult = result.Payload
	case garmoperatorv1alpha1.OrganizationScope:
		result, err := garmClient.CreateOrgPool(
			organizations.NewCreateOrgPoolParams().
				WithOrgID(id).
				WithBody(poolParams))
		if err != nil {
			return params.Pool{}, err
		}
		poolResult = result.Payload
	case garmoperatorv1alpha1.RepositoryScope:
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

func GarmPoolExists(garmClient garmClient.PoolClient, pool *garmoperatorv1alpha1.Pool) bool {
	result, err := garmClient.GetPool(pools.NewGetPoolParams().WithPoolID(pool.Status.ID))
	if err != nil {
		return false
	}
	return result.Payload.ID != ""
}

func CheckDuplicate(ctx context.Context, client client.Client, pool *garmoperatorv1alpha1.Pool, poolImage *garmoperatorv1alpha1.Image) (bool, string, error) {
	poolList := &garmoperatorv1alpha1.PoolList{}
	err := client.List(ctx, poolList)
	if err != nil {
		return false, "", err
	}

	// only get other pools
	filteredPoolList := filter.Match(poolList.Items,
		garmoperatorv1alpha1.MatchesFlavor(pool.Spec.Flavor),
		garmoperatorv1alpha1.MatchesProvider(pool.Spec.ProviderName),
		garmoperatorv1alpha1.MatchesGitHubScope(pool.Spec.GitHubScopeRef.Name, pool.Spec.GitHubScopeRef.Kind),
		garmoperatorv1alpha1.NotMatchingName(pool.Name),
	)

	for _, p := range filteredPoolList {
		pool := p
		image, err := imageUtil.GetByPoolCR(ctx, client, &pool)
		if err != nil {
			continue
		}
		if image.Spec.Tag == poolImage.Spec.Tag {
			return true, fmt.Sprintf("%s/%s", pool.Namespace, pool.Name), nil
		}
	}
	return false, "", nil
}
