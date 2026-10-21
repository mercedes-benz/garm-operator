// SPDX-License-Identifier: MIT

package key

const (
	groupName                   = "garm-operator.mercedes-benz.com"
	EnterpriseFinalizerName     = groupName + "/enterprise"
	OrganizationFinalizerName   = groupName + "/organization"
	RepositoryFinalizerName     = groupName + "/repository"
	PoolFinalizerName           = groupName + "/pool"
	ScaleSetFinalizerName       = groupName + "/scaleset"
	RunnerFinalizerName         = groupName + "/runner"
	GitHubEndpointFinalizerName = groupName + "/endpoint"
	CredentialsFinalizerName    = groupName + "/credentials"
	ServerConfigFinalizerName   = groupName + "/serverconfig"
	PausedAnnotation            = groupName + "/paused"
	// AdoptAnnotation opts a ScaleSet into binding to an existing remote scale
	// set with the same name. Without it a name collision is reported, never
	// adopted.
	AdoptAnnotation = groupName + "/adopt"
)
