// SPDX-License-Identifier: MIT

package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	garmProviderParams "github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm/client/enterprises"
	"github.com/cloudbase/garm/client/instances"
	"github.com/cloudbase/garm/client/organizations"
	"github.com/cloudbase/garm/client/repositories"
	"github.com/cloudbase/garm/client/scalesets"
	"github.com/cloudbase/garm/params"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	"github.com/mercedes-benz/garm-operator/pkg/client/key"
	"github.com/mercedes-benz/garm-operator/pkg/client/mock"
	"github.com/mercedes-benz/garm-operator/pkg/conditions"
)

const (
	testScaleSetNamespace = "test-namespace"
	testScaleSetOrgName   = "test-org"
	testScaleSetOrgID     = "11111111-1111-1111-1111-111111111111"
	testScaleSetRepoName  = "test-repo"
	testScaleSetRepoID    = "22222222-2222-2222-2222-222222222222"
	testScaleSetEntName   = "test-enterprise"
	testScaleSetEntID     = "33333333-3333-3333-3333-333333333333"
)

func scaleSetTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		t.Fatalf("scheme registration failed: %v", err)
	}
	if err := garmoperatorv1beta1.AddToScheme(s); err != nil {
		t.Fatalf("scheme registration failed: %v", err)
	}
	return s
}

func newTestOrg() *garmoperatorv1beta1.Organization {
	return &garmoperatorv1beta1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testScaleSetOrgName,
			Namespace: testScaleSetNamespace,
		},
		Status: garmoperatorv1beta1.OrganizationStatus{
			ID: testScaleSetOrgID,
		},
	}
}

func newTestRepo() *garmoperatorv1beta1.Repository {
	return &garmoperatorv1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testScaleSetRepoName,
			Namespace: testScaleSetNamespace,
		},
		Status: garmoperatorv1beta1.RepositoryStatus{
			ID: testScaleSetRepoID,
		},
	}
}

func newTestEnterprise() *garmoperatorv1beta1.Enterprise {
	return &garmoperatorv1beta1.Enterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testScaleSetEntName,
			Namespace: testScaleSetNamespace,
		},
		Status: garmoperatorv1beta1.EnterpriseStatus{
			ID: testScaleSetEntID,
		},
	}
}

func newTestScaleSet(name string, opts ...func(*garmoperatorv1beta1.ScaleSet)) *garmoperatorv1beta1.ScaleSet {
	ss := &garmoperatorv1beta1.ScaleSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  testScaleSetNamespace,
			Finalizers: []string{key.ScaleSetFinalizerName},
		},
		Spec: garmoperatorv1beta1.ScaleSetSpec{
			GitHubScopeRef: corev1.TypedLocalObjectReference{
				APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
				Kind:     string(garmoperatorv1beta1.OrganizationScope),
				Name:     testScaleSetOrgName,
			},
			ProviderName:           "kubernetes_external",
			Image:                  "ubuntu:22.04",
			Flavor:                 "medium",
			OSType:                 "linux",
			OSArch:                 "amd64",
			MaxRunners:             5,
			MinIdleRunners:         2,
			RunnerBootstrapTimeout: 20,
			Enabled:                true,
			Labels:                 []string{"large", "ubuntu"},
		},
	}
	for _, opt := range opts {
		opt(ss)
	}
	return ss
}

func TestScaleSetReconcile_NotFound(t *testing.T) {
	r := &ScaleSetReconciler{Client: fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).Build()}
	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: "missing"},
	})
	if err != nil {
		t.Fatalf("expected nil error for missing object, got %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Fatal("expected no requeue for missing object")
	}
}

func TestScaleSetReconcile_Paused(t *testing.T) {
	set := newTestScaleSet("paused-set", func(s *garmoperatorv1beta1.ScaleSet) {
		s.Annotations = map[string]string{
			key.PausedAnnotation: "true",
		}
	})
	r := &ScaleSetReconciler{
		Client: fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).Build(),
	}
	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
	})
	if err != nil {
		t.Fatalf("expected nil error for paused object, got %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Fatal("expected no requeue for paused object")
	}
}

func TestScaleSetReconcile_InvalidSpecFails(t *testing.T) {
	set := newTestScaleSet("invalid-spec", func(s *garmoperatorv1beta1.ScaleSet) {
		s.Spec.MaxRunners = 0 // maxRunners cannot be 0
	})
	r := &ScaleSetReconciler{
		Client: fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
	}
	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
	})
	if err != nil {
		t.Fatalf("expected nil error on permanent validation failure, got: %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Fatal("expected no requeue for invalid spec")
	}

	got := &garmoperatorv1beta1.ScaleSet{}
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got); err != nil {
		t.Fatalf("failed to get set: %v", err)
	}
	cond := conditions.Get(got, conditions.ReadyCondition)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.InvalidSpecReason) {
		t.Fatalf("expected ReadyCondition to be False/InvalidSpec, got %#v", cond)
	}
}

func TestScaleSetReconcile_EnsuresFinalizer(t *testing.T) {
	set := newTestScaleSet("no-finalizer", func(s *garmoperatorv1beta1.ScaleSet) {
		s.Finalizers = nil
	})
	r := &ScaleSetReconciler{
		Client: fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
	}
	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
	})
	if err != nil {
		t.Fatalf("expected nil error on adding finalizer, got %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Fatal("expected no requeue")
	}
	got := &garmoperatorv1beta1.ScaleSet{}
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got); err != nil {
		t.Fatalf("failed to get set: %v", err)
	}
	if len(got.Finalizers) != 1 || got.Finalizers[0] != key.ScaleSetFinalizerName {
		t.Fatalf("expected finalizer %s, got %v", key.ScaleSetFinalizerName, got.Finalizers)
	}
}

func TestScaleSetReconcile_ScopeValidation(t *testing.T) {
	t.Run("scope not found in cluster", func(t *testing.T) {
		set := newTestScaleSet("missing-scope")
		r := &ScaleSetReconciler{
			Client:   fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
			Recorder: events.NewFakeRecorder(10),
		}
		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("expected nil error on missing scope to avoid backoff spam, got %v", err)
		}
		if res.RequeueAfter != 0 {
			t.Fatal("expected no requeue when scope not found (watch will trigger reconciliation)")
		}
		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.ScopeRefNotFoundReason) {
			t.Fatalf("expected ReadyCondition False/ScopeRefNotFound, got %#v", cond)
		}
	})

	t.Run("scope not ready yet (missing ID)", func(t *testing.T) {
		org := newTestOrg()
		org.Status.ID = ""
		set := newTestScaleSet("scope-not-ready")
		r := &ScaleSetReconciler{
			Client:   fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder: events.NewFakeRecorder(10),
		}
		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("expected nil error on scope missing ID to avoid backoff spam, got %v", err)
		}
		if res.RequeueAfter != 0 {
			t.Fatal("expected no requeue when scope not ready (watch will trigger reconciliation)")
		}
		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.ScopeRefNotReadyReason) {
			t.Fatalf("expected ReadyCondition False/ScopeRefNotReady, got %#v", cond)
		}
	})

	t.Run("unsupported scope kind", func(t *testing.T) {
		set := newTestScaleSet("unsupported-scope", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Spec.GitHubScopeRef.Kind = "Cluster"
		})
		r := &ScaleSetReconciler{
			Client:   fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
			Recorder: events.NewFakeRecorder(10),
		}
		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("expected nil error for permanent invalid spec, got %v", err)
		}
		if res.RequeueAfter != 0 {
			t.Fatal("expected no requeue for invalid spec")
		}
		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.InvalidSpecReason) {
			t.Fatalf("expected ReadyCondition False/InvalidSpec, got %#v", cond)
		}
	})
}

func TestScaleSetReconcile_Create(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Run("organization scope create", func(t *testing.T) {
		org := newTestOrg()
		set := newTestScaleSet("org-scaleset")

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		mockScaleSet.EXPECT().ListOrgScaleSets(testScaleSetOrgID).Return(&organizations.ListOrgScaleSetsOK{
			Payload: []params.ScaleSet{},
		}, nil)

		mockScaleSet.EXPECT().CreateOrgScaleSet(testScaleSetOrgID, gomock.Any()).DoAndReturn(
			func(orgID string, param *organizations.CreateOrgScaleSetParams) (*organizations.CreateOrgScaleSetOK, error) {
				if param.Body.Name != set.Name {
					t.Fatalf("expected body name %s, got %s", set.Name, param.Body.Name)
				}
				if param.Body.ProviderName != set.Spec.ProviderName {
					t.Fatalf("expected provider %s, got %s", set.Spec.ProviderName, param.Body.ProviderName)
				}
				return &organizations.CreateOrgScaleSetOK{
					Payload: params.ScaleSet{
						ID:           101,
						Name:         set.Name,
						ProviderName: set.Spec.ProviderName,
						Enabled:      set.Spec.Enabled,
					},
				}, nil
			})

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.RequeueAfter != 0 {
			t.Fatal("expected no requeue")
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		if err := r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got); err != nil {
			t.Fatalf("failed to get set: %v", err)
		}
		if got.Status.ID != "101" {
			t.Fatalf("expected Status.ID 101, got %s", got.Status.ID)
		}
		if !got.Status.Enabled {
			t.Fatal("expected Status.Enabled true")
		}
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionTrue {
			t.Fatalf("expected ReadyCondition True, got %#v", cond)
		}
	})

	t.Run("repository scope create", func(t *testing.T) {
		repo := newTestRepo()
		set := newTestScaleSet("repo-scaleset", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Spec.GitHubScopeRef.Kind = string(garmoperatorv1beta1.RepositoryScope)
			s.Spec.GitHubScopeRef.Name = testScaleSetRepoName
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		mockScaleSet.EXPECT().ListRepoScaleSets(testScaleSetRepoID).Return(&repositories.ListRepoScaleSetsOK{
			Payload: []params.ScaleSet{},
		}, nil)

		mockScaleSet.EXPECT().CreateRepoScaleSet(testScaleSetRepoID, gomock.Any()).Return(&repositories.CreateRepoScaleSetOK{
			Payload: params.ScaleSet{
				ID:      201,
				Name:    set.Name,
				Enabled: true,
			},
		}, nil)

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, repo).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if got.Status.ID != "201" {
			t.Fatalf("expected Status.ID 201, got %s", got.Status.ID)
		}
	})

	t.Run("enterprise scope create", func(t *testing.T) {
		ent := newTestEnterprise()
		set := newTestScaleSet("ent-scaleset", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Spec.GitHubScopeRef.Kind = string(garmoperatorv1beta1.EnterpriseScope)
			s.Spec.GitHubScopeRef.Name = testScaleSetEntName
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		mockScaleSet.EXPECT().ListEnterpriseScaleSets(testScaleSetEntID).Return(&enterprises.ListEnterpriseScaleSetsOK{
			Payload: []params.ScaleSet{},
		}, nil)

		mockScaleSet.EXPECT().CreateEnterpriseScaleSet(testScaleSetEntID, gomock.Any()).Return(&enterprises.CreateEnterpriseScaleSetOK{
			Payload: params.ScaleSet{
				ID:      301,
				Name:    set.Name,
				Enabled: true,
			},
		}, nil)

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, ent).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if got.Status.ID != "301" {
			t.Fatalf("expected Status.ID 301, got %s", got.Status.ID)
		}
	})
}

func TestScaleSetReconcile_AdoptionByName(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	org := newTestOrg()

	makeLive := func(name string) params.ScaleSet {
		return params.ScaleSet{
			ID:                     42,
			Name:                   name,
			OrgID:                  testScaleSetOrgID,
			ProviderName:           "kubernetes_external",
			Image:                  "ubuntu:22.04",
			Flavor:                 "medium",
			OSType:                 "linux",
			OSArch:                 "amd64",
			MaxRunners:             5,
			MinIdleRunners:         2,
			RunnerBootstrapTimeout: 20,
			Enabled:                true,
			Tags: []params.Tag{
				{Name: "large"},
				{Name: "ubuntu"},
			},
		}
	}

	t.Run("refuses adoption without opt-in annotation", func(t *testing.T) {
		set := newTestScaleSet("adopt-me-no-annot")
		live := makeLive(set.Name)

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		mockScaleSet.EXPECT().ListOrgScaleSets(testScaleSetOrgID).Return(&organizations.ListOrgScaleSetsOK{
			Payload: []params.ScaleSet{live},
		}, nil)

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.RequeueAfter != 0 {
			t.Fatal("expected no requeue")
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if got.Status.ID != "" {
			t.Fatalf("expected empty Status.ID without annotation, got %s", got.Status.ID)
		}
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.AdoptionRequiredReason) {
			t.Fatalf("expected ReadyCondition False/AdoptionRequired, got %#v", cond)
		}
	})

	t.Run("successfully adopts matching remote set when annotation is set", func(t *testing.T) {
		set := newTestScaleSet("adopt-me-with-annot", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Annotations = map[string]string{key.AdoptAnnotation: "true"}
		})
		live := makeLive(set.Name)

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		mockScaleSet.EXPECT().ListOrgScaleSets(testScaleSetOrgID).Return(&organizations.ListOrgScaleSetsOK{
			Payload: []params.ScaleSet{live},
		}, nil)

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if got.Status.ID != "42" {
			t.Fatalf("expected adopted Status.ID 42, got %s", got.Status.ID)
		}
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionTrue {
			t.Fatalf("expected ReadyCondition True, got %#v", cond)
		}
	})

	t.Run("refuses adoption if another ScaleSet already owns the remote ID", func(t *testing.T) {
		ownerSet := newTestScaleSet("existing-owner", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Status.ID = "42"
		})
		set := newTestScaleSet("adopt-collision", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Annotations = map[string]string{key.AdoptAnnotation: "true"}
		})
		live := makeLive(set.Name)

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		mockScaleSet.EXPECT().ListOrgScaleSets(testScaleSetOrgID).Return(&organizations.ListOrgScaleSetsOK{
			Payload: []params.ScaleSet{live},
		}, nil)

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(ownerSet, set, org).WithStatusSubresource(ownerSet, set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.RequeueAfter != 0 {
			t.Fatal("expected no requeue")
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if got.Status.ID != "" {
			t.Fatalf("expected Status.ID to remain empty, got %s", got.Status.ID)
		}
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.AdoptionRefusedReason) {
			t.Fatalf("expected ReadyCondition False/AdoptionRefused, got %#v", cond)
		}
	})

	t.Run("refuses adoption and never destructive-recreates if remote has immutable drift", func(t *testing.T) {
		set := newTestScaleSet("adopt-drift", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Annotations = map[string]string{key.AdoptAnnotation: "true"}
			s.Spec.ProviderName = "different_provider" // immutable drift
		})
		live := makeLive(set.Name) // ProviderName is "lxd"

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		mockScaleSet.EXPECT().ListOrgScaleSets(testScaleSetOrgID).Return(&organizations.ListOrgScaleSetsOK{
			Payload: []params.ScaleSet{live},
		}, nil)

		// DeleteScaleSet must NEVER be called here!

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.RequeueAfter != 0 {
			t.Fatal("expected no requeue")
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if got.Status.ID != "" {
			t.Fatalf("expected Status.ID to remain empty, got %s", got.Status.ID)
		}
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.AdoptionRefusedReason) {
			t.Fatalf("expected ReadyCondition False/AdoptionRefused, got %#v", cond)
		}
	})
}

func TestScaleSetReconcile_Update(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Run("mutable drift triggers UpdateScaleSet", func(t *testing.T) {
		org := newTestOrg()
		set := newTestScaleSet("update-drift", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Status.ID = "42"
			s.Spec.MinIdleRunners = 5 // changed from 2 to 5
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		live := params.ScaleSet{
			ID:                     42,
			Name:                   set.Name,
			OrgID:                  testScaleSetOrgID,
			ProviderName:           set.Spec.ProviderName,
			Image:                  set.Spec.Image,
			Flavor:                 set.Spec.Flavor,
			OSType:                 set.Spec.OSType,
			OSArch:                 set.Spec.OSArch,
			MaxRunners:             set.Spec.MaxRunners,
			MinIdleRunners:         2, // differs from spec.MinIdleRunners (5)
			RunnerBootstrapTimeout: set.Spec.RunnerBootstrapTimeout,
			Enabled:                true,
			Tags: []params.Tag{
				{Name: "large"},
				{Name: "ubuntu"},
			},
		}

		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(&scalesets.GetScaleSetOK{
			Payload: live,
		}, nil)

		mockScaleSet.EXPECT().UpdateScaleSet(uint(42), gomock.Any()).DoAndReturn(
			func(id uint, param *scalesets.UpdateScaleSetParams) (*scalesets.UpdateScaleSetOK, error) {
				if param.Body.MinIdleRunners == nil || *param.Body.MinIdleRunners != 5 {
					t.Fatalf("expected MinIdleRunners 5, got %v", param.Body.MinIdleRunners)
				}
				live.MinIdleRunners = 5
				return &scalesets.UpdateScaleSetOK{Payload: live}, nil
			})

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionTrue {
			t.Fatalf("expected ReadyCondition True, got %#v", cond)
		}
	})

	t.Run("no drift skips UpdateScaleSet", func(t *testing.T) {
		org := newTestOrg()
		set := newTestScaleSet("no-drift", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Status.ID = "42"
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		live := params.ScaleSet{
			ID:                     42,
			Name:                   set.Name,
			OrgID:                  testScaleSetOrgID,
			ProviderName:           set.Spec.ProviderName,
			Image:                  set.Spec.Image,
			Flavor:                 set.Spec.Flavor,
			OSType:                 set.Spec.OSType,
			OSArch:                 set.Spec.OSArch,
			MaxRunners:             set.Spec.MaxRunners,
			MinIdleRunners:         set.Spec.MinIdleRunners,
			RunnerBootstrapTimeout: set.Spec.RunnerBootstrapTimeout,
			Enabled:                set.Spec.Enabled,
			Tags: []params.Tag{
				{Name: "large"},
				{Name: "ubuntu"},
			},
		}

		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(&scalesets.GetScaleSetOK{
			Payload: live,
		}, nil)

		// UpdateScaleSet should NOT be called

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestScaleSetReconcile_Recreate_ImmutableDrift(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Run("drains idle/error instances, requeues when runners remain", func(t *testing.T) {
		org := newTestOrg()
		set := newTestScaleSet("recreate-drain", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Status.ID = "42"
			s.Spec.ProviderName = "new_provider" // immutable drift
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		live := params.ScaleSet{
			ID:           42,
			Name:         set.Name,
			OrgID:        testScaleSetOrgID,
			ProviderName: "old_provider", // differs from spec.ProviderName
			Enabled:      true,
			Instances: []params.Instance{
				{
					Name:         "runner-idle",
					Status:       garmProviderParams.InstanceRunning,
					RunnerStatus: params.RunnerIdle,
				},
				{
					Name:         "runner-error",
					Status:       garmProviderParams.InstanceError,
					RunnerStatus: params.RunnerPending,
				},
			},
			Tags: []params.Tag{{Name: "large"}, {Name: "ubuntu"}},
		}

		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(&scalesets.GetScaleSetOK{
			Payload: live,
		}, nil)

		// Disables the live set
		mockScaleSet.EXPECT().UpdateScaleSet(uint(42), gomock.Any()).DoAndReturn(
			func(id uint, p *scalesets.UpdateScaleSetParams) (*scalesets.UpdateScaleSetOK, error) {
				if p.Body.Enabled == nil || *p.Body.Enabled != false {
					t.Fatalf("expected Enabled false in disable call")
				}
				live.Enabled = false
				return &scalesets.UpdateScaleSetOK{Payload: live}, nil
			})

		// Deletes drainable instances (idle and error)
		mockInstance.EXPECT().DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName("runner-idle")).Return(nil)
		mockInstance.EXPECT().DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName("runner-error")).Return(nil)

		// DeleteScaleSet must NOT be called yet because instances > 0

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.RequeueAfter != 1*time.Minute {
			t.Fatalf("expected RequeueAfter 1m, got %v", res.RequeueAfter)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.DrainingReason) {
			t.Fatalf("expected ReadyCondition False/Draining, got %#v", cond)
		}
	})

	t.Run("refuses to delete and recreate when deletionPolicy is Retain", func(t *testing.T) {
		org := newTestOrg()
		set := newTestScaleSet("recreate-retain", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Status.ID = "42"
			s.Spec.DeletionPolicy = garmoperatorv1beta1.ScaleSetDeletionPolicyRetain
			s.Spec.ProviderName = "new_provider" // immutable drift
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		live := params.ScaleSet{
			ID:           42,
			Name:         set.Name,
			OrgID:        testScaleSetOrgID,
			ProviderName: "old_provider",
			Enabled:      true,
			Tags:         []params.Tag{{Name: "large"}, {Name: "ubuntu"}},
		}

		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(&scalesets.GetScaleSetOK{
			Payload: live,
		}, nil)

		// DeleteScaleSet must NOT be called

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.RequeueAfter != 0 {
			t.Fatal("expected no requeue when recreate is blocked")
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.RecreateBlockedReason) {
			t.Fatalf("expected ReadyCondition False/RecreateBlocked, got %#v", cond)
		}
	})

	t.Run("deletes scale set and creates fresh when 0 instances remain", func(t *testing.T) {
		org := newTestOrg()
		set := newTestScaleSet("recreate-empty", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Status.ID = "42"
			s.Spec.ProviderName = "new_provider" // immutable drift
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		live := params.ScaleSet{
			ID:           42,
			Name:         set.Name,
			OrgID:        testScaleSetOrgID,
			ProviderName: "old_provider",
			Enabled:      false, // already disabled
			Instances:    []params.Instance{},
			Tags:         []params.Tag{{Name: "large"}, {Name: "ubuntu"}},
		}

		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(&scalesets.GetScaleSetOK{
			Payload: live,
		}, nil)

		// Since Enabled is false, disable is NOT called again
		mockScaleSet.EXPECT().DeleteScaleSet(uint(42)).Return(nil)

		// Fresh scale set created
		mockScaleSet.EXPECT().CreateOrgScaleSet(testScaleSetOrgID, gomock.Any()).Return(&organizations.CreateOrgScaleSetOK{
			Payload: params.ScaleSet{
				ID:           84,
				Name:         set.Name,
				ProviderName: "new_provider",
				Enabled:      true,
			},
		}, nil)

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.RequeueAfter != 0 {
			t.Fatal("expected no requeue")
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if got.Status.ID != "84" {
			t.Fatalf("expected updated Status.ID 84, got %s", got.Status.ID)
		}
	})
}

func TestScaleSetReconcile_404_Recreates(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Run("404 with no existing set creates new", func(t *testing.T) {
		org := newTestOrg()
		set := newTestScaleSet("stale-id", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Status.ID = "42"
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(nil, scalesets.NewGetScaleSetDefault(404))

		mockScaleSet.EXPECT().ListOrgScaleSets(testScaleSetOrgID).Return(&organizations.ListOrgScaleSetsOK{
			Payload: []params.ScaleSet{},
		}, nil)

		mockScaleSet.EXPECT().CreateOrgScaleSet(testScaleSetOrgID, gomock.Any()).Return(&organizations.CreateOrgScaleSetOK{
			Payload: params.ScaleSet{
				ID:      99,
				Name:    set.Name,
				Enabled: true,
			},
		}, nil)

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if got.Status.ID != "99" {
			t.Fatalf("expected Status.ID 99, got %s", got.Status.ID)
		}
	})

	t.Run("404 with existing set by name adopts it when adopt annotation set", func(t *testing.T) {
		org := newTestOrg()
		set := newTestScaleSet("stale-id-adopt", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Status.ID = "42"
			s.Annotations = map[string]string{key.AdoptAnnotation: "true"}
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(nil, scalesets.NewGetScaleSetDefault(404))

		live := params.ScaleSet{
			ID:                     77,
			Name:                   set.Name,
			OrgID:                  testScaleSetOrgID,
			ProviderName:           set.Spec.ProviderName,
			Image:                  set.Spec.Image,
			Flavor:                 set.Spec.Flavor,
			OSType:                 set.Spec.OSType,
			OSArch:                 set.Spec.OSArch,
			MaxRunners:             set.Spec.MaxRunners,
			MinIdleRunners:         set.Spec.MinIdleRunners,
			RunnerBootstrapTimeout: set.Spec.RunnerBootstrapTimeout,
			Enabled:                set.Spec.Enabled,
			Tags:                   []params.Tag{{Name: "large"}, {Name: "ubuntu"}},
		}

		mockScaleSet.EXPECT().ListOrgScaleSets(testScaleSetOrgID).Return(&organizations.ListOrgScaleSetsOK{
			Payload: []params.ScaleSet{live},
		}, nil)

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if got.Status.ID != "77" {
			t.Fatalf("expected adopted Status.ID 77, got %s", got.Status.ID)
		}
	})
}

func TestScaleSetReconcile_Delete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	now := metav1.Now()

	t.Run("empty status ID removes finalizer without calling GARM", func(t *testing.T) {
		set := newTestScaleSet("delete-empty-id", func(s *garmoperatorv1beta1.ScaleSet) {
			s.DeletionTimestamp = &now
			s.Status.ID = ""
		})

		r := &ScaleSetReconciler{
			Client: fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if len(got.Finalizers) != 0 {
			t.Fatalf("expected finalizer removed, got %v", got.Finalizers)
		}
	})

	t.Run("deletion policy retain removes finalizer without calling GARM", func(t *testing.T) {
		set := newTestScaleSet("delete-retain", func(s *garmoperatorv1beta1.ScaleSet) {
			s.DeletionTimestamp = &now
			s.Status.ID = "42"
			s.Spec.DeletionPolicy = garmoperatorv1beta1.ScaleSetDeletionPolicyRetain
		})

		r := &ScaleSetReconciler{
			Client: fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if len(got.Finalizers) != 0 {
			t.Fatalf("expected finalizer removed, got %v", got.Finalizers)
		}
	})

	t.Run("remote returns 404 removes finalizer", func(t *testing.T) {
		set := newTestScaleSet("delete-404", func(s *garmoperatorv1beta1.ScaleSet) {
			s.DeletionTimestamp = &now
			s.Status.ID = "42"
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(nil, scalesets.NewGetScaleSetDefault(404))

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
			ScaleSetClient: mockScaleSet,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if len(got.Finalizers) != 0 {
			t.Fatalf("expected finalizer removed, got %v", got.Finalizers)
		}
	})

	t.Run("drains runners and requeues before deletion", func(t *testing.T) {
		set := newTestScaleSet("delete-drain", func(s *garmoperatorv1beta1.ScaleSet) {
			s.DeletionTimestamp = &now
			s.Status.ID = "42"
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		live := params.ScaleSet{
			ID:      42,
			Name:    set.Name,
			Enabled: true,
			Instances: []params.Instance{
				{
					Name:         "idle-runner",
					Status:       garmProviderParams.InstanceRunning,
					RunnerStatus: params.RunnerIdle,
				},
				{
					Name:         "error-runner",
					Status:       garmProviderParams.InstanceError,
					RunnerStatus: params.RunnerPending,
				},
				{
					Name:         "busy-runner",
					Status:       garmProviderParams.InstanceRunning,
					RunnerStatus: params.RunnerActive,
				},
			},
		}

		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(&scalesets.GetScaleSetOK{
			Payload: live,
		}, nil)

		// Disables the live set
		mockScaleSet.EXPECT().UpdateScaleSet(uint(42), gomock.Any()).Return(&scalesets.UpdateScaleSetOK{Payload: live}, nil)

		// Deletes idle and errored runners, but NOT busy runner
		mockInstance.EXPECT().DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName("idle-runner")).Return(nil)
		mockInstance.EXPECT().DeleteInstance(instances.NewDeleteInstanceParams().WithInstanceName("error-runner")).Return(nil)

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		res, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.RequeueAfter != 1*time.Minute {
			t.Fatalf("expected RequeueAfter 1m, got %v", res.RequeueAfter)
		}

		// Finalizer must NOT be removed yet
		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if len(got.Finalizers) == 0 {
			t.Fatal("expected finalizer to remain while draining")
		}
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.DrainingReason) {
			t.Fatalf("expected ReadyCondition False/Draining, got %#v", cond)
		}
	})

	t.Run("paused scale set deletion still proceeds and removes finalizer", func(t *testing.T) {
		set := newTestScaleSet("delete-paused", func(s *garmoperatorv1beta1.ScaleSet) {
			s.DeletionTimestamp = &now
			s.Status.ID = "42"
			s.Annotations = map[string]string{key.PausedAnnotation: "true"}
			s.Spec.DeletionPolicy = garmoperatorv1beta1.ScaleSetDeletionPolicyRetain
		})

		r := &ScaleSetReconciler{
			Client: fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if len(got.Finalizers) != 0 {
			t.Fatalf("expected finalizer removed, got %v", got.Finalizers)
		}
	})

	t.Run("invalid spec on delete path does not block finalizer removal", func(t *testing.T) {
		set := newTestScaleSet("delete-invalid-spec", func(s *garmoperatorv1beta1.ScaleSet) {
			s.DeletionTimestamp = &now
			s.Status.ID = "42"
			s.Spec.MaxRunners = 0 // invalid spec!
			s.Spec.DeletionPolicy = garmoperatorv1beta1.ScaleSetDeletionPolicyRetain
		})

		r := &ScaleSetReconciler{
			Client: fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if len(got.Finalizers) != 0 {
			t.Fatalf("expected finalizer removed even with invalid spec, got %v", got.Finalizers)
		}
	})

	t.Run("deletes remote set and removes finalizer when 0 instances", func(t *testing.T) {
		set := newTestScaleSet("delete-complete", func(s *garmoperatorv1beta1.ScaleSet) {
			s.DeletionTimestamp = &now
			s.Status.ID = "42"
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockInstance := mock.NewMockInstanceClient(mockCtrl)

		live := params.ScaleSet{
			ID:        42,
			Name:      set.Name,
			Enabled:   false, // already disabled
			Instances: []params.Instance{},
		}

		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(&scalesets.GetScaleSetOK{
			Payload: live,
		}, nil)

		mockScaleSet.EXPECT().DeleteScaleSet(uint(42)).Return(nil)

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
			ScaleSetClient: mockScaleSet,
			InstanceClient: mockInstance,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		if len(got.Finalizers) != 0 {
			t.Fatalf("expected finalizer removed, got %v", got.Finalizers)
		}
	})
}

func TestScaleSetReconcile_APIErrorHandling(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	org := newTestOrg()

	t.Run("GetScaleSet returns API error", func(t *testing.T) {
		set := newTestScaleSet("get-api-error", func(s *garmoperatorv1beta1.ScaleSet) {
			s.Status.ID = "42"
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(nil, errors.New("garm server connection refused"))

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err == nil {
			t.Fatal("expected error from GetScaleSet")
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse {
			t.Fatalf("expected ReadyCondition False, got %#v", cond)
		}
	})

	t.Run("CreateOrgScaleSet returns API error", func(t *testing.T) {
		set := newTestScaleSet("create-api-error")

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		mockScaleSet.EXPECT().ListOrgScaleSets(testScaleSetOrgID).Return(&organizations.ListOrgScaleSetsOK{
			Payload: []params.ScaleSet{},
		}, nil)

		mockScaleSet.EXPECT().CreateOrgScaleSet(testScaleSetOrgID, gomock.Any()).Return(nil, errors.New("cannot create scale set: quota exceeded"))

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set, org).WithStatusSubresource(set).Build(),
			Recorder:       events.NewFakeRecorder(10),
			ScaleSetClient: mockScaleSet,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err == nil {
			t.Fatal("expected error from CreateOrgScaleSet")
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse {
			t.Fatalf("expected ReadyCondition False, got %#v", cond)
		}
	})

	t.Run("DeleteScaleSet returns API error on delete", func(t *testing.T) {
		now := metav1.Now()
		set := newTestScaleSet("delete-api-error", func(s *garmoperatorv1beta1.ScaleSet) {
			s.DeletionTimestamp = &now
			s.Status.ID = "42"
		})

		mockScaleSet := mock.NewMockScaleSetClient(mockCtrl)
		live := params.ScaleSet{
			ID:        42,
			Name:      set.Name,
			Enabled:   false,
			Instances: []params.Instance{},
		}
		mockScaleSet.EXPECT().GetScaleSet(uint(42)).Return(&scalesets.GetScaleSetOK{Payload: live}, nil)
		mockScaleSet.EXPECT().DeleteScaleSet(uint(42)).Return(errors.New("garm database locked"))

		r := &ScaleSetReconciler{
			Client:         fake.NewClientBuilder().WithScheme(scaleSetTestScheme(t)).WithObjects(set).WithStatusSubresource(set).Build(),
			ScaleSetClient: mockScaleSet,
		}

		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name},
		})
		if err == nil {
			t.Fatal("expected error from DeleteScaleSet")
		}

		got := &garmoperatorv1beta1.ScaleSet{}
		_ = r.Get(context.Background(), types.NamespacedName{Namespace: testScaleSetNamespace, Name: set.Name}, got)
		cond := conditions.Get(got, conditions.ReadyCondition)
		if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(conditions.DeletionFailedReason) {
			t.Fatalf("expected ReadyCondition False/DeletionFailedReason, got %#v", cond)
		}
	})
}
