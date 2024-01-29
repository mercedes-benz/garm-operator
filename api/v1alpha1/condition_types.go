package v1alpha1

type ConditionType string

const (
	ReadyCondition     ConditionType = "Ready"
	ImageResourceFound ConditionType = "ImageResourceFound"
)

const (
	SuccessfulReconcileReason = "SuccessfulReconcile"
	FetchingImageRefReason    = "FetchingImageRef"
)
