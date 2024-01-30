package v1alpha1

type ConditionType string

// Common Conditions & Reasons
const (
	ReadyCondition            ConditionType = "Ready"
	SuccessfulReconcileReason               = "ReconcileSuccess"
	ReconcileErrorReason                    = "ReconcileError"
	DeletingReason                          = "Deleting"
	DeletionFailedReason                    = "DeletionFailed"
)

// Pool Conditions & Reasons
const (
	ImageReference                ConditionType = "ImageReference"
	FetchingImageRefSuccessReason               = "FetchingImageRefSuccess"
	FetchingImageRefFailedReason                = "FetchingImageRefFailed"

	ScopeReference                ConditionType = "ScopeReference"
	FetchingScopeRefSuccessReason               = "FetchingScopeRefSuccess"
	FetchingScopeRefFailedReason                = "FetchingScopeRefFailed"
	ScopeRefNotReadyReason                      = "ScopeRefNotReady"
)
