// SPDX-License-Identifier: MIT

package scalesets

import (
	"testing"

	"github.com/cloudbase/garm/params"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
)

type fakeScope struct{ id, name, kind string }

func (f fakeScope) GetKind() string                     { return f.kind }
func (f fakeScope) GetCredentialsName() string          { return "" }
func (f fakeScope) GetID() string                       { return f.id }
func (f fakeScope) GetName() string                     { return f.name }
func (f fakeScope) GetPoolManagerIsRunning() bool       { return true }
func (f fakeScope) GetPoolManagerFailureReason() string { return "" }

func validSpec() garmoperatorv1beta1.ScaleSetSpec {
	return garmoperatorv1beta1.ScaleSetSpec{
		ProviderName:           "azure-notprod",
		Image:                  "gallery-image",
		Flavor:                 "Standard_D8ds_v7",
		OSType:                 "linux",
		OSArch:                 "amd64",
		MaxRunners:             10,
		MinIdleRunners:         1,
		RunnerBootstrapTimeout: 20,
		Labels:                 []string{"garm", "azure-notprod-vms"},
		RunnerPrefix:           "azure-notprod-vms",
		GitHubRunnerGroup:      "hybrid-cloud-fabric-runners",
		Enabled:                true,
	}
}

func orgScope() fakeScope {
	return fakeScope{id: "org-id", name: "geico-private", kind: "Organization"}
}

func TestBuildCreateParams_MapsSpec(t *testing.T) {
	got := BuildCreateParams(validSpec(), "azure-notprod-vms-v3")
	if got.Name != "azure-notprod-vms-v3" {
		t.Fatalf("expected CR name carried over, got %q", got.Name)
	}
	if got.ProviderName != "azure-notprod" || got.Image != "gallery-image" || got.Flavor != "Standard_D8ds_v7" {
		t.Fatalf("unexpected provider/image/flavor mapping: %+v", got)
	}
	if !got.Enabled || got.MinIdleRunners != 1 || got.MaxRunners != 10 {
		t.Fatalf("expected enabled/idle/max preserved: %+v", got)
	}
}

func TestBuildCreateParams_DefaultsAndExtraSpecs(t *testing.T) {
	spec := validSpec()
	spec.ExtraSpecs = `{"capabilities":["nested-virt"]}`
	got := BuildCreateParams(spec, "set-with-extra")
	if got.RunnerBootstrapTimeout != 20 {
		t.Fatalf("expected timeout 20 from spec, got %d", got.RunnerBootstrapTimeout)
	}
	if string(got.ExtraSpecs) != `{"capabilities":["nested-virt"]}` {
		t.Fatalf("expected extraSpecs preserved, got %s", string(got.ExtraSpecs))
	}
}

func TestBuildCreateParams_RejectsBadSpec(t *testing.T) {
	spec := validSpec()
	spec.Image = "__STAGED__"
	if err := spec.Validate(); err == nil {
		t.Fatal("expected placeholder spec to fail validation")
	}
}

func TestBuildUpdateParams(t *testing.T) {
	t.Run("maps all fields with non-empty runner group and extra specs", func(t *testing.T) {
		spec := validSpec()
		spec.ExtraSpecs = `{"foo":"bar"}`
		got := BuildUpdateParams(spec, params.ScaleSet{})
		if got.Enabled == nil || !*got.Enabled {
			t.Fatalf("expected Enabled true, got %v", got.Enabled)
		}
		if got.MaxRunners == nil || *got.MaxRunners != 10 {
			t.Fatalf("expected MaxRunners 10, got %v", got.MaxRunners)
		}
		if got.MinIdleRunners == nil || *got.MinIdleRunners != 1 {
			t.Fatalf("expected MinIdleRunners 1, got %v", got.MinIdleRunners)
		}
		if got.RunnerBootstrapTimeout == nil || *got.RunnerBootstrapTimeout != 20 {
			t.Fatalf("expected RunnerBootstrapTimeout 20, got %v", got.RunnerBootstrapTimeout)
		}
		if got.Image != "gallery-image" || got.Flavor != "Standard_D8ds_v7" || got.OSType != "linux" || got.OSArch != "amd64" {
			t.Fatalf("unexpected image/flavor/os mapping: %+v", got)
		}
		if got.GitHubRunnerGroup == nil || *got.GitHubRunnerGroup != "hybrid-cloud-fabric-runners" {
			t.Fatalf("expected runner group hybrid-cloud-fabric-runners, got %v", got.GitHubRunnerGroup)
		}
		if got.Prefix != "azure-notprod-vms" {
			t.Fatalf("expected runner prefix azure-notprod-vms, got %q", got.Prefix)
		}
		if string(got.ExtraSpecs) != `{"foo":"bar"}` {
			t.Fatalf("expected extra specs, got %s", string(got.ExtraSpecs))
		}
	})

	t.Run("empty runner group and extra specs are omitted when the remote has none", func(t *testing.T) {
		spec := validSpec()
		spec.GitHubRunnerGroup = ""
		spec.ExtraSpecs = ""
		got := BuildUpdateParams(spec, params.ScaleSet{})
		if got.GitHubRunnerGroup != nil {
			t.Fatalf("expected nil runner group for empty string, got %v", got.GitHubRunnerGroup)
		}
		if len(got.ExtraSpecs) != 0 {
			t.Fatalf("expected empty extra specs, got %v", got.ExtraSpecs)
		}
	})

	t.Run("empty extra specs clear what the remote still has", func(t *testing.T) {
		spec := validSpec()
		spec.ExtraSpecs = ""
		live := params.ScaleSet{ExtraSpecs: []byte(`{"a":1}`)}
		got := BuildUpdateParams(spec, live)
		if string(got.ExtraSpecs) != `{}` {
			t.Fatalf("expected {} to clear extra specs, got %q", string(got.ExtraSpecs))
		}
	})

	t.Run("empty extra specs are not resent when the remote holds an empty object", func(t *testing.T) {
		spec := validSpec()
		spec.ExtraSpecs = ""
		got := BuildUpdateParams(spec, params.ScaleSet{ExtraSpecs: []byte(`{}`)})
		if len(got.ExtraSpecs) != 0 {
			t.Fatalf("expected no extra specs to be sent, got %q", string(got.ExtraSpecs))
		}
	})
}

func TestImmutableDrift_DetectsLabelChange(t *testing.T) {
	live := params.ScaleSet{ID: 7, ProviderName: "azure-notprod", OrgID: "org-id"}
	live.Tags = []params.Tag{{Name: "old-label"}}
	if !ImmutableDrift(validSpec(), live, orgScope()) {
		t.Fatal("expected label drift to require recreation")
	}
}

func TestImmutableDrift_CleanWhenMatching(t *testing.T) {
	live := params.ScaleSet{ID: 7, ProviderName: "azure-notprod", OrgID: "org-id"}
	live.Tags = []params.Tag{{Name: "garm"}, {Name: "azure-notprod-vms"}}
	if ImmutableDrift(validSpec(), live, orgScope()) {
		t.Fatal("expected no drift for matching labels/provider/scope")
	}
}

func TestImmutableDrift_HandlesDuplicateLabels(t *testing.T) {
	spec := validSpec()
	spec.Labels = []string{"garm", "garm", "azure-notprod-vms"}
	live := params.ScaleSet{ID: 7, ProviderName: "azure-notprod", OrgID: "org-id"}
	live.Tags = []params.Tag{{Name: "garm"}, {Name: "azure-notprod-vms"}}
	if ImmutableDrift(spec, live, orgScope()) {
		t.Fatal("expected duplicate spec labels to match deduplicated live tags")
	}
}

func TestImmutableDrift_ProviderChange(t *testing.T) {
	spec := validSpec()
	live := params.ScaleSet{ID: 7, ProviderName: "other-provider", OrgID: "org-id"}
	live.Tags = []params.Tag{{Name: "garm"}, {Name: "azure-notprod-vms"}}
	if !ImmutableDrift(spec, live, orgScope()) {
		t.Fatal("expected provider drift to return true")
	}
}

func TestImmutableDrift_ScopeChange(t *testing.T) {
	spec := validSpec()
	live := params.ScaleSet{ID: 7, ProviderName: "azure-notprod", OrgID: "different-org-id"}
	live.Tags = []params.Tag{{Name: "garm"}, {Name: "azure-notprod-vms"}}
	if !ImmutableDrift(spec, live, orgScope()) {
		t.Fatal("expected scope mismatch to return true")
	}
}

func TestImmutableDrift_LabelKeyMismatchWithSameCount(t *testing.T) {
	spec := validSpec()
	spec.Labels = []string{"tag1", "tag2"}
	live := params.ScaleSet{ID: 7, ProviderName: "azure-notprod", OrgID: "org-id"}
	live.Tags = []params.Tag{{Name: "tag1"}, {Name: "tag3"}}
	if !ImmutableDrift(spec, live, orgScope()) {
		t.Fatal("expected label key mismatch with same count to return true")
	}
}

func TestImmutableDrift_IgnoresEmptyLabelStrings(t *testing.T) {
	spec := validSpec()
	spec.Labels = []string{"garm", "", "azure-notprod-vms"}
	live := params.ScaleSet{ID: 7, ProviderName: "azure-notprod", OrgID: "org-id"}
	live.Tags = []params.Tag{{Name: "garm"}, {Name: ""}, {Name: "azure-notprod-vms"}}
	if ImmutableDrift(spec, live, orgScope()) {
		t.Fatal("expected empty strings to be ignored in label comparison")
	}
}

func TestSameEntity(t *testing.T) {
	spec := validSpec()

	t.Run("nil scope returns true", func(t *testing.T) {
		live := params.ScaleSet{ID: 1}
		if !sameEntity(spec, live, nil) {
			t.Fatal("expected nil scope to return true")
		}
	})

	t.Run("EnterpriseScope match and mismatch", func(t *testing.T) {
		entScope := fakeScope{id: "ent-123", name: "my-ent", kind: "Enterprise"}
		matchingLive := params.ScaleSet{EnterpriseID: "ent-123"}
		if !sameEntity(spec, matchingLive, entScope) {
			t.Fatal("expected Enterprise matching ID to return true")
		}
		mismatchLive := params.ScaleSet{EnterpriseID: "ent-456"}
		if sameEntity(spec, mismatchLive, entScope) {
			t.Fatal("expected Enterprise mismatched ID to return false")
		}
	})

	t.Run("OrganizationScope match and mismatch", func(t *testing.T) {
		org := fakeScope{id: "org-123", name: "my-org", kind: "Organization"}
		matchingLive := params.ScaleSet{OrgID: "org-123"}
		if !sameEntity(spec, matchingLive, org) {
			t.Fatal("expected Organization matching ID to return true")
		}
		mismatchLive := params.ScaleSet{OrgID: "org-456"}
		if sameEntity(spec, mismatchLive, org) {
			t.Fatal("expected Organization mismatched ID to return false")
		}
	})

	t.Run("RepositoryScope match and mismatch", func(t *testing.T) {
		repo := fakeScope{id: "repo-123", name: "my-repo", kind: "Repository"}
		matchingLive := params.ScaleSet{RepoID: "repo-123"}
		if !sameEntity(spec, matchingLive, repo) {
			t.Fatal("expected Repository matching ID to return true")
		}
		mismatchLive := params.ScaleSet{RepoID: "repo-456"}
		if sameEntity(spec, mismatchLive, repo) {
			t.Fatal("expected Repository mismatched ID to return false")
		}
	})

	t.Run("unsupported scope kind returns false", func(t *testing.T) {
		unknown := fakeScope{id: "unk-123", name: "unknown", kind: "Unknown"}
		live := params.ScaleSet{ID: 1}
		if sameEntity(spec, live, unknown) {
			t.Fatal("expected unknown scope kind to return false")
		}
	})
}

func TestExtraSpecsEqual(t *testing.T) {
	tests := []struct {
		name string
		want string
		live []byte
		exp  bool
	}{
		{name: "both empty", want: "", live: nil, exp: true},
		{name: "both empty slices", want: "", live: []byte{}, exp: true},
		{name: "want empty, live non-empty", want: "", live: []byte(`{"a":1}`), exp: false},
		{name: "want empty, live empty object", want: "", live: []byte(`{}`), exp: true},
		{name: "want empty, live empty object with whitespace", want: "", live: []byte("{ }"), exp: true},
		{name: "want empty, live null", want: "", live: []byte(`null`), exp: true},
		{name: "want non-empty, live empty", want: `{"a":1}`, live: nil, exp: false},
		{name: "identical json", want: `{"a": 1}`, live: []byte(`{"a": 1}`), exp: true},
		{name: "whitespace difference", want: `{"a":1, "b": 2}`, live: []byte("{\n  \"a\": 1,\n  \"b\": 2\n}"), exp: true},
		{name: "different values", want: `{"a":1}`, live: []byte(`{"a":2}`), exp: false},
		{name: "invalid json in want", want: `{invalid-json`, live: []byte(`{"a":1}`), exp: false},
		{name: "invalid json in live", want: `{"a":1}`, live: []byte(`{invalid-json`), exp: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extraSpecsEqual(tt.want, tt.live)
			if got != tt.exp {
				t.Fatalf("extraSpecsEqual(%q, %q) = %v; want %v", tt.want, string(tt.live), got, tt.exp)
			}
		})
	}
}

func TestMutableDrift_AllFields(t *testing.T) {
	baseSpec := validSpec()
	baseLive := params.ScaleSet{
		ProviderName:           "azure-notprod",
		Image:                  "gallery-image",
		Flavor:                 "Standard_D8ds_v7",
		OSType:                 "linux",
		OSArch:                 "amd64",
		MaxRunners:             10,
		MinIdleRunners:         1,
		RunnerBootstrapTimeout: 20,
		Enabled:                true,
		GitHubRunnerGroup:      "hybrid-cloud-fabric-runners",
	}
	baseLive.Prefix = "azure-notprod-vms"

	tests := []struct {
		name       string
		modifySpec func(s *garmoperatorv1beta1.ScaleSetSpec)
		modifyLive func(l *params.ScaleSet)
	}{
		{
			name:       "MinIdleRunners differs",
			modifyLive: func(l *params.ScaleSet) { l.MinIdleRunners = 5 },
		},
		{
			name:       "MaxRunners differs",
			modifyLive: func(l *params.ScaleSet) { l.MaxRunners = 20 },
		},
		{
			name:       "Image differs",
			modifyLive: func(l *params.ScaleSet) { l.Image = "other-image" },
		},
		{
			name:       "Flavor differs",
			modifyLive: func(l *params.ScaleSet) { l.Flavor = "Standard_B2s" },
		},
		{
			name:       "OSType differs",
			modifyLive: func(l *params.ScaleSet) { l.OSType = "windows" },
		},
		{
			name:       "OSArch differs",
			modifyLive: func(l *params.ScaleSet) { l.OSArch = "arm64" },
		},
		{
			name:       "RunnerBootstrapTimeout differs",
			modifyLive: func(l *params.ScaleSet) { l.RunnerBootstrapTimeout = 45 },
		},
		{
			name:       "RunnerPrefix differs",
			modifyLive: func(l *params.ScaleSet) { l.Prefix = "other-prefix" },
		},
		{
			name:       "GitHubRunnerGroup differs",
			modifyLive: func(l *params.ScaleSet) { l.GitHubRunnerGroup = "other-group" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := baseSpec
			l := baseLive
			if tt.modifySpec != nil {
				tt.modifySpec(&s)
			}
			if tt.modifyLive != nil {
				tt.modifyLive(&l)
			}
			if !MutableDrift(s, l) {
				t.Fatalf("expected MutableDrift to be true for case %s", tt.name)
			}
		})
	}
}

func TestFindByName_AdoptsOnce(t *testing.T) {
	sets := []params.ScaleSet{{ID: 3, Name: "garm-k8s-v4"}, {ID: 7, Name: "other"}}
	got := FindByName(sets, "garm-k8s-v4")
	if got == nil || got.ID != 3 {
		t.Fatalf("expected adoption of ID 3, got %+v", got)
	}
	if FindByName(sets, "missing") != nil {
		t.Fatal("expected unknown name to return nil")
	}
	if FindByName(nil, "missing") != nil {
		t.Fatal("expected nil slice to return nil")
	}
}

func TestMutableDrift_IgnoresRunnerGroupWhenSpecEmpty(t *testing.T) {
	spec := validSpec()
	spec.GitHubRunnerGroup = ""
	live := params.ScaleSet{
		ProviderName: "azure-notprod", Image: "gallery-image", Flavor: "Standard_D8ds_v7",
		OSType: "linux", OSArch: "amd64", MaxRunners: 10, MinIdleRunners: 1,
		RunnerBootstrapTimeout: 20, Enabled: true,
		GitHubRunnerGroup: "Default",
	}
	live.Prefix = "azure-notprod-vms"
	if MutableDrift(spec, live) {
		t.Fatal("an unset runner group must not drift: GARM cannot clear it, so the update would repeat forever")
	}
}

func TestMutableDrift_DetectsExternalDisable(t *testing.T) {
	live := params.ScaleSet{
		ProviderName: "azure-notprod", Image: "gallery-image", Flavor: "Standard_D8ds_v7",
		OSType: "linux", OSArch: "amd64", MaxRunners: 10, MinIdleRunners: 1,
		RunnerBootstrapTimeout: 20, Enabled: false,
		GitHubRunnerGroup: "hybrid-cloud-fabric-runners",
	}
	live.Prefix = "azure-notprod-vms"
	if !MutableDrift(validSpec(), live) {
		t.Fatal("expected externally disabled remote to drift")
	}
	live.Enabled = true
	if MutableDrift(validSpec(), live) {
		t.Fatal("expected matching live state to skip update")
	}
}

func TestMutableDrift_DetectsExtraSpecsGap(t *testing.T) {
	spec := validSpec()
	spec.ExtraSpecs = `{"disable_isolated_networks":true}`
	live := params.ScaleSet{
		ProviderName: "azure-notprod", Image: "gallery-image", Flavor: "Standard_D8ds_v7",
		OSType: "linux", OSArch: "amd64", MaxRunners: 10, MinIdleRunners: 1,
		RunnerBootstrapTimeout: 20, Enabled: true,
		GitHubRunnerGroup: "hybrid-cloud-fabric-runners",
	}
	live.Prefix = "azure-notprod-vms"
	if !MutableDrift(spec, live) {
		t.Fatal("expected missing live extra_specs to drift")
	}
	live.ExtraSpecs = []byte(`{"disable_isolated_networks": true}`)
	if MutableDrift(spec, live) {
		t.Fatal("expected whitespace-only difference to skip update")
	}
}

func TestFindByID(t *testing.T) {
	sets := []params.ScaleSet{{ID: 3}, {ID: 7}}
	got := FindByID(sets, 7)
	if got == nil || got.ID != 7 {
		t.Fatalf("expected ID 7 adopted, got %+v", got)
	}
	if FindByID(sets, 9) != nil {
		t.Fatal("expected unknown ID to return nil")
	}
	if FindByID(nil, 9) != nil {
		t.Fatal("expected nil slice to return nil")
	}
}
