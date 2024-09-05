package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mercedes-benz/garm-operator/pkg/filter"
)

func (p *Pool) CheckDuplicate(ctx context.Context, client client.Client, poolImage *Image) (bool, string, error) {
	poolList := &PoolList{}
	err := client.List(ctx, poolList)
	if err != nil {
		return false, "", err
	}

	// only get other pools
	filteredPoolList := filter.Match(poolList.Items,
		MatchesFlavor(p.Spec.Flavor),
		MatchesProvider(p.Spec.ProviderName),
		MatchesGitHubScope(p.Spec.GitHubScopeRef.Name, p.Spec.GitHubScopeRef.Kind),
		NotMatchingName(p.Name),
	)

	for _, p := range filteredPoolList {
		image, err := p.GetImageCR(ctx, client)
		if err != nil {
			continue
		}
		if image.Spec.Tag == poolImage.Spec.Tag {
			return true, fmt.Sprintf("%s/%s", p.Namespace, p.Name), nil
		}
	}
	return false, "", nil
}

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
