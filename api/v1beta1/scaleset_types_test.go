// SPDX-License-Identifier: MIT

package v1beta1

import (
	"testing"

	commonParams "github.com/cloudbase/garm-provider-common/params"
	corev1 "k8s.io/api/core/v1"
)

func validScaleSetSpec() ScaleSetSpec {
	return ScaleSetSpec{
		GitHubScopeRef: corev1.TypedLocalObjectReference{
			APIGroup: func() *string { s := "garm-operator.mercedes-benz.com"; return &s }(),
			Kind:     "Organization",
			Name:     "geico-private",
		},
		ProviderName:           "azure-notprod",
		Image:                  "gallery-image-id",
		Flavor:                 "Standard_D8ds_v7",
		OSType:                 commonParams.OSType("linux"),
		OSArch:                 commonParams.OSArch("amd64"),
		MaxRunners:             10,
		MinIdleRunners:         1,
		RunnerBootstrapTimeout: 20,
		Labels:                 []string{"garm", "azure-notprod-vms"},
		Enabled:                true,
		DeletionPolicy:         ScaleSetDeletionPolicyDelete,
	}
}

func TestScaleSetValidate_AcceptsValidSpec(t *testing.T) {
	if err := validScaleSetSpec().Validate(); err != nil {
		t.Fatalf("expected valid spec, got %v", err)
	}
}

func TestScaleSetValidate_RejectsMinIdleAboveMax(t *testing.T) {
	spec := validScaleSetSpec()
	spec.MinIdleRunners = 11
	if err := spec.Validate(); err == nil {
		t.Fatal("expected minIdleRunners > maxRunners to fail")
	}
}

func TestScaleSetValidate_AcceptsDoubleUnderscoreInImage(t *testing.T) {
	spec := validScaleSetSpec()
	spec.Image = "registry.example.com/runner__base:1"
	if err := spec.Validate(); err != nil {
		t.Fatalf("expected legitimate image name to be accepted, got %v", err)
	}
}

func TestScaleSetValidate_RejectsZeroBootstrapTimeout(t *testing.T) {
	spec := validScaleSetSpec()
	spec.RunnerBootstrapTimeout = 0
	if err := spec.Validate(); err == nil {
		t.Fatal("expected runnerBootstrapTimeout 0 to fail")
	}
}

func TestScaleSetValidate_RejectsUnknownScopeKind(t *testing.T) {
	spec := validScaleSetSpec()
	spec.GitHubScopeRef.Kind = "Pool"
	if err := spec.Validate(); err == nil {
		t.Fatal("expected unsupported scope kind to fail")
	}
}

func TestScaleSetValidate_RejectsForeignScopeAPIGroup(t *testing.T) {
	spec := validScaleSetSpec()
	other := "example.com"
	spec.GitHubScopeRef.APIGroup = &other
	if err := spec.Validate(); err == nil {
		t.Fatal("expected foreign apiGroup to fail")
	}
}

func TestScaleSetValidate_AcceptsOmittedScopeAPIGroup(t *testing.T) {
	spec := validScaleSetSpec()
	spec.GitHubScopeRef.APIGroup = nil
	if err := spec.Validate(); err != nil {
		t.Fatalf("expected omitted apiGroup to be accepted, got %v", err)
	}
}

func TestScaleSetValidate_RejectsEmptyProvider(t *testing.T) {
	spec := validScaleSetSpec()
	spec.ProviderName = ""
	if err := spec.Validate(); err == nil {
		t.Fatal("expected empty provider to fail")
	}
}

func TestScaleSet_InitializeConditions(t *testing.T) {
	s := &ScaleSet{}
	s.InitializeConditions()
	if len(s.Status.Conditions) == 0 {
		t.Fatal("expected Ready condition to be initialized")
	}
}
