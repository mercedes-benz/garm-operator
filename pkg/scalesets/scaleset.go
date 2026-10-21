// SPDX-License-Identifier: MIT

// Package scalesets holds pure ScaleSet reconcile helpers: payload builders,
// adopt-by-ID lookup and immutable-field drift detection. No API calls here,
// so every function is unit-testable without a cluster.
package scalesets

import (
	"bytes"
	"encoding/json"

	"github.com/cloudbase/garm/params"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
)

// BuildCreateParams maps a ScaleSet CR spec onto GARM create params. The CR
// name becomes the remote scale-set name. Scope travels in the request URL
// (POST /organizations/{id}/scalesets etc), never in the body.
func BuildCreateParams(spec garmoperatorv1beta1.ScaleSetSpec, name string) params.CreateScaleSetParams {
	out := params.CreateScaleSetParams{
		RunnerPrefix:           params.RunnerPrefix{Prefix: spec.RunnerPrefix},
		Name:                   name,
		ProviderName:           spec.ProviderName,
		MaxRunners:             spec.MaxRunners,
		MinIdleRunners:         spec.MinIdleRunners,
		Image:                  spec.Image,
		Flavor:                 spec.Flavor,
		OSType:                 spec.OSType,
		OSArch:                 spec.OSArch,
		Enabled:                spec.Enabled,
		RunnerBootstrapTimeout: spec.RunnerBootstrapTimeout,
		GitHubRunnerGroup:      spec.GitHubRunnerGroup,
		Labels:                 append([]string(nil), spec.Labels...),
	}
	if spec.ExtraSpecs != "" {
		out.ExtraSpecs = json.RawMessage([]byte(spec.ExtraSpecs))
	}
	return out
}

// BuildUpdateParams maps mutable spec fields onto GARM update params.
// Immutable routing (provider, labels, scope) is handled by ImmutableDrift.
//
// live is needed to clear extra specs: GARM treats an omitted value as "leave
// untouched", so an emptied spec.extraSpecs is sent as an empty JSON object,
// and only when the remote still has something to clear. GARM cannot clear a
// runner group, so an empty spec.githubRunnerGroup is simply not sent.
func BuildUpdateParams(spec garmoperatorv1beta1.ScaleSetSpec, live params.ScaleSet) params.UpdateScaleSetParams {
	enabled := spec.Enabled
	maxRunners := spec.MaxRunners
	minIdle := spec.MinIdleRunners
	timeout := spec.RunnerBootstrapTimeout
	var runnerGroup *string
	if spec.GitHubRunnerGroup != "" {
		runnerGroup = &spec.GitHubRunnerGroup
	}
	out := params.UpdateScaleSetParams{
		Enabled:                &enabled,
		MaxRunners:             &maxRunners,
		MinIdleRunners:         &minIdle,
		RunnerBootstrapTimeout: &timeout,
		Image:                  spec.Image,
		Flavor:                 spec.Flavor,
		OSType:                 spec.OSType,
		OSArch:                 spec.OSArch,
		GitHubRunnerGroup:      runnerGroup,
	}
	out.Prefix = spec.RunnerPrefix
	switch {
	case spec.ExtraSpecs != "":
		out.ExtraSpecs = json.RawMessage([]byte(spec.ExtraSpecs))
	case !isEmptyJSON(live.ExtraSpecs):
		out.ExtraSpecs = json.RawMessage(`{}`)
	}
	return out
}

// ImmutableDrift reports whether provider, labels or scope entity differ
// between desired spec and live remote state. True means disable, drain,
// delete and recreate; the update path must not run.
func ImmutableDrift(spec garmoperatorv1beta1.ScaleSetSpec, live params.ScaleSet, scope garmoperatorv1beta1.GitHubScope) bool {
	if live.ProviderName != spec.ProviderName {
		return true
	}
	if !sameEntity(spec, live, scope) {
		return true
	}
	wantMap := make(map[string]struct{}, len(spec.Labels))
	for _, l := range spec.Labels {
		if l != "" {
			wantMap[l] = struct{}{}
		}
	}
	gotMap := make(map[string]struct{}, len(live.Tags))
	for _, tag := range live.Tags {
		if tag.Name != "" {
			gotMap[tag.Name] = struct{}{}
		}
	}
	if len(wantMap) != len(gotMap) {
		return true
	}
	for k := range wantMap {
		if _, ok := gotMap[k]; !ok {
			return true
		}
	}
	return false
}

// MutableDrift reports whether any controller-managed mutable field differs
// between desired spec and live remote state. True means update; false skips
// the call. Enabled included: externally disabled remotes get re-enabled.
//
// An empty spec.githubRunnerGroup is not compared: GARM cannot clear a runner
// group, so treating it as drift would update on every reconcile forever.
func MutableDrift(spec garmoperatorv1beta1.ScaleSetSpec, live params.ScaleSet) bool {
	if live.Enabled != spec.Enabled ||
		live.MinIdleRunners != spec.MinIdleRunners ||
		live.MaxRunners != spec.MaxRunners ||
		live.Image != spec.Image ||
		live.Flavor != spec.Flavor ||
		live.OSType != spec.OSType ||
		live.OSArch != spec.OSArch ||
		live.RunnerBootstrapTimeout != spec.RunnerBootstrapTimeout ||
		live.Prefix != spec.RunnerPrefix ||
		(spec.GitHubRunnerGroup != "" && live.GitHubRunnerGroup != spec.GitHubRunnerGroup) ||
		!extraSpecsEqual(spec.ExtraSpecs, live.ExtraSpecs) {
		return true
	}
	return false
}

// isEmptyJSON reports whether a raw JSON document carries no extra specs:
// nothing, null or an empty object.
func isEmptyJSON(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return false
	}
	return buf.String() == "null" || buf.String() == "{}"
}

// extraSpecsEqual compares desired JSON string against live raw message with
// whitespace normalized. An empty desired value converges against any live
// value that carries no specs (see isEmptyJSON).
func extraSpecsEqual(want string, live json.RawMessage) bool {
	if want == "" {
		return isEmptyJSON(live)
	}
	if len(live) == 0 {
		return false
	}
	var wantBuf, liveBuf bytes.Buffer
	if err := json.Compact(&wantBuf, []byte(want)); err != nil {
		return false
	}
	if err := json.Compact(&liveBuf, live); err != nil {
		return false
	}
	return wantBuf.String() == liveBuf.String()
}

// FindByID adopts remote state by stable numeric ID, never by name.
func FindByID(sets []params.ScaleSet, id uint) *params.ScaleSet {
	for i := range sets {
		if sets[i].ID == id {
			return &sets[i]
		}
	}
	return nil
}

// FindByName locates a remote set for one-time adoption only. After adoption
// status.id binds the CR and all later reads go through FindByID. A rename
// therefore never adopts a different remote set.
func FindByName(sets []params.ScaleSet, name string) *params.ScaleSet {
	for i := range sets {
		if sets[i].Name == name {
			return &sets[i]
		}
	}
	return nil
}

func sameEntity(_ garmoperatorv1beta1.ScaleSetSpec, live params.ScaleSet, scope garmoperatorv1beta1.GitHubScope) bool {
	if scope == nil {
		return true
	}
	switch scope.GetKind() {
	case string(garmoperatorv1beta1.EnterpriseScope):
		return live.EnterpriseID == scope.GetID()
	case string(garmoperatorv1beta1.OrganizationScope):
		return live.OrgID == scope.GetID()
	case string(garmoperatorv1beta1.RepositoryScope):
		return live.RepoID == scope.GetID()
	}
	return false
}
