// SPDX-License-Identifier: MIT

package v1beta1

import (
	"context"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mercedes-benz/garm-operator/pkg/filter"
)

func (p *Pool) GetImageCR(ctx context.Context, client client.Client) (*Image, error) {
	image := &Image{}
	if p.Spec.ImageName != "" {
		if err := client.Get(ctx, types.NamespacedName{Name: p.Spec.ImageName, Namespace: p.Namespace}, image); err != nil {
			return nil, err
		}
	}
	return image, nil
}

func MatchesImage(image string) filter.Predicate[Pool] {
	return func(p Pool) bool {
		return p.Spec.ImageName == image
	}
}

func MatchesFlavor(flavor string) filter.Predicate[Pool] {
	return func(p Pool) bool {
		return p.Spec.Flavor == flavor
	}
}

func MatchesProvider(provider string) filter.Predicate[Pool] {
	return func(p Pool) bool {
		return p.Spec.ProviderName == provider
	}
}

func MatchesGitHubScope(name, kind string) filter.Predicate[Pool] {
	return func(p Pool) bool {
		return p.Spec.GitHubScopeRef.Name == name && p.Spec.GitHubScopeRef.Kind == kind
	}
}

func MatchesID(id string) filter.Predicate[Pool] {
	return func(p Pool) bool {
		return p.Status.ID == id
	}
}

func NotMatchingName(name string) filter.Predicate[Pool] {
	return func(p Pool) bool {
		return p.Name != name
	}
}
