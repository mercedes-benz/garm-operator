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

	WebhookSecretReference                ConditionType   = "WebhookSecretReference"
	FetchingWebhookSecretRefSuccessReason ConditionReason = "FetchingWebhookSecretRefSuccess"
	FetchingWebhookSecretRefFailedReason  ConditionReason = "FetchingWebhookSecretRefFailed"

	GithubCredentialsReference                ConditionType   = "GithubCredentialsReference"  // #nosec G101
	FetchingGithubCredentialsRefSuccessReason ConditionReason = "GithubCredentialsRefSuccess" // #nosec G101
	FetchingGithubCredentialsRefFailedReason  ConditionReason = "GithubCredentialsRefFailed"  // #nosec G101
)

// Credential Conditions
const (
	GithubEndpointReference                ConditionType   = "GithubEndpointReference"
	FetchingGithubEndpointRefSuccessReason ConditionReason = "FetchingGithubEndpointRefSuccess"
	FetchingGithubEndpointRefFailedReason  ConditionReason = "FetchingGithubEndpointRefFailed"
)

const (
	GarmServerNotReconciledYetMsg     string = "GARM server not reconciled yet"
	CredentialsNotReconciledYetMsg    string = "GithubCredentialsRef not reconciled yet" // #nosec G101
	GithubEndpointNotReconciledYetMsg string = "GithubEndpointRef not reconciled yet"    // #nosec G101
	WebhookSecretNotReconciledYetMsg  string = "WebhookSecretRef not reconciled yet"     // #nosec G101
	DeletingEnterpriseMsg             string = "Deleting enterprise"
	DeletingOrgMsg                    string = "Deleting organization"
	DeletingRepoMsg                   string = "Deleting repository"
	DeletingPoolMsg                   string = "Deleting pool"
	DeletingEndpointMsg               string = "Deleting endpoint"
	DeletingCredentialsMsg            string = "Deleting credentials" // #nosec G101
)
