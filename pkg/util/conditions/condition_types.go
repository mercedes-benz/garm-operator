// SPDX-License-Identifier: MIT

package conditions

type (
	ConditionType   string
	ConditionReason string
)

// Common Conditions & Reasons
const (
	ReadyCondition            ConditionType   = "Ready"
	SuccessfulReconcileReason ConditionReason = "ReconcileSuccess"
	ReconcileErrorReason      ConditionReason = "ReconcileError"
	DeletingReason            ConditionReason = "Deleting"
	DeletionFailedReason      ConditionReason = "DeletionFailed"
)

// Pool Conditions & Reasons
const (
	ImageReference                ConditionType   = "ImageReference"
	FetchingImageRefSuccessReason ConditionReason = "FetchingImageRefSuccess"
	FetchingImageRefFailedReason  ConditionReason = "FetchingImageRefFailed"

	ScopeReference                ConditionType   = "ScopeReference"
	FetchingScopeRefSuccessReason ConditionReason = "FetchingScopeRefSuccess"
	FetchingScopeRefFailedReason  ConditionReason = "FetchingScopeRefFailed"
	ScopeRefNotReadyReason        ConditionReason = "ScopeRefNotReady"
)

// Enterprise, Org & Repo Conditions
const (
	PoolManager              ConditionType   = "PoolManager"
	PoolManagerRunningReason ConditionReason = "PoolManagerRunning"
	PoolManagerFailureReason ConditionReason = "PoolManagerFailure"

	SecretReference                ConditionType   = "SecretReference"
	FetchingSecretRefSuccessReason ConditionReason = "FetchingSecretRefSuccess"
	FetchingSecretRefFailedReason  ConditionReason = "FetchingSecretRefFailed"
)
