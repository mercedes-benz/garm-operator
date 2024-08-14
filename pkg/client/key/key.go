// SPDX-License-Identifier: MIT

package key

const (
	groupName                 = "garm-operator.mercedes-benz.com"
	EnterpriseFinalizerName   = groupName + "/enterprise"
	OrganizationFinalizerName = groupName + "/organization"
	RepositoryFinalizerName   = groupName + "/repository"
	PoolFinalizerName         = groupName + "/pool"
	RunnerFinalizerName       = groupName + "/runner"
	EndpointFinalizerName     = groupName + "/endpoint"
	PausedAnnotation          = groupName + "/paused"
)
