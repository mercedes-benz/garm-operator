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
	GarmAPIErrorReason        ConditionReason = "GarmAPIError"
	UnknownReason             ConditionReason = "UnknownReason"
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
	DuplicatePoolReason           ConditionReason = "DuplicatePoolFound"
)

// Enterprise, Org & Repo Conditions
const (
	PoolManager              ConditionType   = "PoolManager"
	PoolManagerRunningReason ConditionReason = "PoolManagerRunning"
	PoolManagerFailureReason ConditionReason = "PoolManagerFailure"

	SecretReference                ConditionType   = "SecretReference"
	FetchingSecretRefSuccessReason ConditionReason = "FetchingSecretRefSuccess"
	FetchingSecretRefFailedReason  ConditionReason = "FetchingSecretRefFailed"

	CredentialsReference                ConditionType   = "CredentialsReference"
	FetchingCredentialsRefSuccessReason ConditionReason = "CredentialsRefSuccess"
	FetchingCredentialsRefFailedReason  ConditionReason = "CredentialsRefFailed"
)

// Credential Conditions
const (
	EndpointReference                ConditionType   = "EndpointReference"
	FetchingEndpointRefSuccessReason ConditionReason = "FetchingEndpointRefSuccess"
	FetchingEndpointRefFailedReason  ConditionReason = "FetchingEndpointRefFailed"
)

const (
	GarmServerNotReconciledYetMsg string = "GARM server not reconciled yet"
	DeletingEnterpriseMsg         string = "Deleting enterprise"
	DeletingOrgMsg                string = "Deleting organization"
	DeletingRepoMsg               string = "Deleting repository"
	DeletingPoolMsg               string = "Deleting pool"
	DeletingEndpointMsg           string = "Deleting endpoint"
	DeletingCredentialsMsg        string = "Deleting credentials"
)
