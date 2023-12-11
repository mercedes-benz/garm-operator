// SPDX-License-Identifier: MIT

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

// +k8s:deepcopy-gen=false
type GitHubScope interface {
	GetKind() string
	GetCredentialsName() string
	GetID() string
	GetName() string
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
	// Name of the kubernetes secret to use
	Name string `json:"name"`
	// Key is the key in the secret's data map for this value
	Key string `json:"key"`
}
