package key

const (
	groupName                 = "garm-operator.mercedes-benz.com"
	EnterpriseFinalizerName   = groupName + "/enterprise"
	OrganizationFinalizerName = groupName + "/organization"
	PoolFinalizerName         = groupName + "/pool"
	PausedAnnotation          = groupName + "/paused"
)
