// SPDX-License-Identifier: MIT

package pools

import (
	"github.com/cloudbase/garm/params"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	"github.com/mercedes-benz/garm-operator/pkg/filter"
)

// MatchesGitHubScope returns a predicate that matches pools with the given GitHub scope and ID
func MatchesGitHubScope(scope garmoperatorv1beta1.GitHubScopeKind, id string) filter.Predicate[params.Pool] {
	return func(p params.Pool) bool {
		if scope == garmoperatorv1beta1.EnterpriseScope {
			return p.EnterpriseID == id
		}

		if scope == garmoperatorv1beta1.OrganizationScope {
			return p.OrgID == id
		}

		if scope == garmoperatorv1beta1.RepositoryScope {
			return p.RepoID == id
		}
		return false
	}
}

// MatchesImage returns a predicate that matches pools with the given image
func MatchesImage(image string) filter.Predicate[params.Pool] {
	return func(p params.Pool) bool {
		return p.Image == image
	}
}

// MatchesFlavor returns a predicate that matches pools with the given flavor
func MatchesFlavor(flavor string) filter.Predicate[params.Pool] {
	return func(p params.Pool) bool {
		return p.Flavor == flavor
	}
}

// MatchesProvider returns a predicate that matches pools with the given provider
func MatchesProvider(provider string) filter.Predicate[params.Pool] {
	return func(p params.Pool) bool {
		return p.ProviderName == provider
	}
}
