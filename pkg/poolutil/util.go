package poolutil

import (
	"git.i.mercedes-benz.com/GitHub-Actions/garm-operator/api/shared"
	"github.com/cloudbase/garm/params"
)

type Predicate func(params.Pool) bool

func MatchesGitHubScope(scope shared.GitHubScopeKind, id string) Predicate {
	return func(p params.Pool) bool {
		if scope == shared.EnterpriseScope {
			return p.EnterpriseID == id
		}

		if scope == shared.OrganizationScope {
			return p.OrgID == id
		}

		if scope == shared.RepositoryScope {
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

func FilterGarmPools(pools []params.Pool, predicates ...func(pool params.Pool) bool) []params.Pool {
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
