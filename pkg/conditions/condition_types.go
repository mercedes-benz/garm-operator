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
	CredentialsNotReconciledYetMsg    string = "Github Credentials Ref not reconciled yet" // #nosec G101
	GithubEndpointNotReconciledYetMsg string = "Github Endpoint Ref not reconciled yet"    // #nosec G101
	WebhookSecretNotReconciledYetMsg  string = "Webhook Secret Ref not reconciled yet"     // #nosec G101
	DeletingEnterpriseMsg             string = "Deleting enterprise"
	DeletingOrgMsg                    string = "Deleting organization"
	DeletingRepoMsg                   string = "Deleting repository"
	DeletingPoolMsg                   string = "Deleting pool"
	DeletingEndpointMsg               string = "Deleting endpoint"
	DeletingCredentialsMsg            string = "Deleting credentials"              // #nosec G101
	DeletingRunnerMsg                 string = "Deleting runner"                   // #nosec G101
	RunnerIdleAndRunningMsg           string = "Runner is idle and running"        // #nosec G101
	RunnerProvisioningFailedMsg       string = "Provisioning runner failed"        // #nosec G101
	RunnerNotReadyYetMsg              string = "Runner is still being provisioned" // #nosec G101
)
