// SPDX-License-Identifier: MIT

package v1beta1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/mercedes-benz/garm-operator/pkg/filter"
)

func TestPoolList_FilterByFields(t *testing.T) {
	type fields struct {
		TypeMeta metav1.TypeMeta
		ListMeta metav1.ListMeta
		Items    []Pool
	}
	type args struct {
		predicates []filter.Predicate[Pool]
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		length int
	}{
		{
			name: "pool with spec already exist",
			fields: fields{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []Pool{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ubuntu-2004-large",
							Namespace: "test",
						},
						Spec: PoolSpec{
							ImageName:    "ubuntu-2004",
							Flavor:       "large",
							ProviderName: "openstack",
							GitHubScopeRef: corev1.TypedLocalObjectReference{
								Name:     "test",
								Kind:     "Enterprise",
								APIGroup: ptr.To[string]("github.com"),
							},
						},
					},
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ubuntu-2204-large",
							Namespace: "test",
						},
						Spec: PoolSpec{
							ImageName:    "ubuntu-2204",
							Flavor:       "large",
							ProviderName: "openstack",
							GitHubScopeRef: corev1.TypedLocalObjectReference{
								Name:     "test",
								Kind:     "Enterprise",
								APIGroup: ptr.To[string]("github.com"),
							},
						},
					},
				},
			},
			args: args{
				predicates: []filter.Predicate[Pool]{
					MatchesImage("ubuntu-2204"),
					MatchesFlavor("large"),
					MatchesProvider("openstack"),
					MatchesGitHubScope("test", "Enterprise"),
				},
			},
			length: 1,
		},
		{
			name: "pool with spec does not exist",
			fields: fields{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
				Items: []Pool{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ubuntu-2004-large",
							Namespace: "test",
						},
						Spec: PoolSpec{
							ImageName:    "ubuntu-2004",
							Flavor:       "large",
							ProviderName: "openstack",
							GitHubScopeRef: corev1.TypedLocalObjectReference{
								Name:     "test",
								Kind:     "Enterprise",
								APIGroup: ptr.To[string]("github.com"),
							},
						},
					},
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ubuntu-2204-large",
							Namespace: "test",
						},
						Spec: PoolSpec{
							ImageName:    "ubuntu-2204",
							Flavor:       "large",
							ProviderName: "openstack",
							GitHubScopeRef: corev1.TypedLocalObjectReference{
								Name:     "test",
								Kind:     "Enterprise",
								APIGroup: ptr.To[string]("github.com"),
							},
						},
					},
				},
			},
			args: args{
				predicates: []filter.Predicate[Pool]{
					MatchesImage("ubuntu-2404"),
					MatchesFlavor("large"),
					MatchesProvider("openstack"),
					MatchesGitHubScope("test", "Enterprise"),
				},
			},
			length: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PoolList{
				TypeMeta: tt.fields.TypeMeta,
				ListMeta: tt.fields.ListMeta,
				Items:    tt.fields.Items,
			}

			filteredItems := filter.Match(p.Items, tt.args.predicates...)

			if len(filteredItems) != tt.length {
				t.Errorf("FilterByFields() = %v, want %v", len(p.Items), tt.length)
			}
		})
	}
}
