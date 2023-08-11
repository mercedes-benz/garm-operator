package v1alpha1

import (
	"fmt"
)

type GitHubScopeKind string

const (
	EnterpriseScope   GitHubScopeKind = "Enterprise"
	OrganizationScope GitHubScopeKind = "Organization"
	RepositoryScope   GitHubScopeKind = "Repository"
)

type GitHubScope interface {
	GetKind() string
	GetCredentialsName() string
	GetID() string
	GetPoolManagerIsRunning() bool
	GetPoolManagerFailureReason() string
}

func ToGitHubScopeKind(kind string) (GitHubScopeKind, error) {
	switch kind {
	case string(EnterpriseScope), string(OrganizationScope), string(RepositoryScope):
		return GitHubScopeKind(kind), nil
	default:
		return GitHubScopeKind(""), fmt.Errorf("can not convert kind %s to valid GitHubScopeKind: Enterprise, Organization, Repository", kind)
	}
}

type SecretRef struct {
	Name string `json:"secretName"`
	Key  string `json:"key"`
}
