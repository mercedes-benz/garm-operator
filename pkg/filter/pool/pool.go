// SPDX-License-Identifier: MIT

package pool

import (
	"github.com/cloudbase/garm/params"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
	"github.com/mercedes-benz/garm-operator/pkg/filter"
)

func MatchesGitHubScope(scope garmoperatorv1alpha1.GitHubScopeKind, id string) filter.Predicate[params.Pool] {
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

func MatchesImage(image string) filter.Predicate[params.Pool] {
	return func(p params.Pool) bool {
		return p.Image == image
	}
}

func MatchesFlavor(flavor string) filter.Predicate[params.Pool] {
	return func(p params.Pool) bool {
		return p.Flavor == flavor
	}
}

func MatchesProvider(provider string) filter.Predicate[params.Pool] {
	return func(p params.Pool) bool {
		return p.ProviderName == provider
	}
}
