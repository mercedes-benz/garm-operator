// SPDX-License-Identifier: MIT

package garmpool

import (
	"github.com/cloudbase/garm/params"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
)

type Predicate func(params.Pool) bool

func MatchesGitHubScope(scope garmoperatorv1alpha1.GitHubScopeKind, id string) Predicate {
	return func(p params.Pool) bool {
		if scope == garmoperatorv1alpha1.EnterpriseScope {
			return p.EnterpriseID == id
		}

		if scope == garmoperatorv1alpha1.OrganizationScope {
			return p.OrgID == id
		}

		if scope == garmoperatorv1alpha1.RepositoryScope {
			return p.RepoID == id
		}
		return false
	}
}

func MatchesImage(image string) Predicate {
	return func(p params.Pool) bool {
		return p.Image == image
	}
}

func MatchesFlavor(flavor string) Predicate {
	return func(p params.Pool) bool {
		return p.Flavor == flavor
	}
}

func MatchesProvider(provider string) Predicate {
	return func(p params.Pool) bool {
		return p.ProviderName == provider
	}
}

func Filter(pools []params.Pool, predicates ...func(pool params.Pool) bool) []params.Pool {
	var filteredPools []params.Pool

	for _, pool := range pools {
		match := true
		for _, predicate := range predicates {
			if !predicate(pool) {
				match = false
				break
			}
		}
		if match {
			filteredPools = append(filteredPools, pool)
		}
	}

	return filteredPools
}
