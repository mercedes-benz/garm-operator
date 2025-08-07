// SPDX-License-Identifier: MIT

package v1beta1

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func Test_validateProviderName(t *testing.T) {
	type args struct {
		pool    *Pool
		oldPool *Pool
	}
	tests := []struct {
		name string
		args args
		want *field.Error
	}{
		{
			name: "no provider name change",
			args: args{
				pool: &Pool{
					Spec: PoolSpec{
						ProviderName: "existing-provider",
					},
				},
				oldPool: &Pool{
					Spec: PoolSpec{
						ProviderName: "existing-provider",
					},
				},
			},
			want: nil,
		},
		{
			name: "provider name change - should return error",
			args: args{
				pool: &Pool{
					Spec: PoolSpec{
						ProviderName: "kubernetes",
					},
				},
				oldPool: &Pool{
					Spec: PoolSpec{
						ProviderName: "mesosphere",
					},
				},
			},
			want: field.Invalid(
				field.NewPath("spec").Child("providerName"),
				"kubernetes",
				"can not change provider of an existing pool. Old name: mesosphere, new name: kubernetes",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateProviderName(tt.args.pool, tt.args.oldPool); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateProviderName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateExtraSpec(t *testing.T) {
	type args struct {
		pool *Pool
	}
	tests := []struct {
		name string
		args args
		want *field.Error
	}{
		{
			name: "extraSpec is valid JSON",
			args: args{
				pool: &Pool{
					Spec: PoolSpec{
						ExtraSpecs: `{"key": "value"}`,
					},
				},
			},
			want: nil,
		},
		{
			name: "extraSpec is empty",
			args: args{
				pool: &Pool{
					Spec: PoolSpec{
						ExtraSpecs: `{}`, // this is defined as default and is set by the API server
					},
				},
			},
			want: nil,
		},
		{
			name: "extraSpec is invalid JSON",
			args: args{
				pool: &Pool{
					Spec: PoolSpec{
						ExtraSpecs: `{"key": "value", "provider": "mes`,
					},
				},
			},
			want: field.Invalid(
				field.NewPath("spec").Child("extraSpecs"),
				"{\"key\": \"value\", \"provider\": \"mes",
				"can not unmarshal extraSpecs: unexpected end of JSON input",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateExtraSpec(tt.args.pool); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateExtraSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateGitHubScope(t *testing.T) {
	type args struct {
		pool    *Pool
		oldPool *Pool
	}
	tests := []struct {
		name string
		args args
		want *field.Error
	}{
		{
			name: "no provider name change",
			args: args{
				pool: &Pool{
					Spec: PoolSpec{
						GitHubScopeRef: corev1.TypedLocalObjectReference{
							APIGroup: &GroupVersion.Group,
							Kind:     string(EnterpriseScope),
							Name:     "good-company",
						},
					},
				},
				oldPool: &Pool{
					Spec: PoolSpec{
						GitHubScopeRef: corev1.TypedLocalObjectReference{
							APIGroup: &GroupVersion.Group,
							Kind:     string(EnterpriseScope),
							Name:     "good-company",
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "GitHubScopreRef.name change - should return error",
			args: args{
				pool: &Pool{
					Spec: PoolSpec{
						GitHubScopeRef: corev1.TypedLocalObjectReference{
							APIGroup: &GroupVersion.Group,
							Kind:     string(EnterpriseScope),
							Name:     "good-company",
						},
					},
				},
				oldPool: &Pool{
					Spec: PoolSpec{
						GitHubScopeRef: corev1.TypedLocalObjectReference{
							APIGroup: &GroupVersion.Group,
							Kind:     string(EnterpriseScope),
							Name:     "evil-company",
						},
					},
				},
			},
			want: field.Invalid(
				field.NewPath("spec").Child("githubScopeRef"),
				"good-company",
				"can not change githubScopeRef of an existing pool. Old name: evil-company, new name: good-company",
			),
		},
		{
			name: "GitHubScopreRef.Kind change - should return error",
			args: args{
				pool: &Pool{
					Spec: PoolSpec{
						GitHubScopeRef: corev1.TypedLocalObjectReference{
							APIGroup: &GroupVersion.Group,
							Kind:     string(RepositoryScope),
							Name:     "kubernetes",
						},
					},
				},
				oldPool: &Pool{
					Spec: PoolSpec{
						GitHubScopeRef: corev1.TypedLocalObjectReference{
							APIGroup: &GroupVersion.Group,
							Kind:     string(OrganizationScope),
							Name:     "kubernetes",
						},
					},
				},
			},
			want: field.Invalid(
				field.NewPath("spec").Child("githubScopeRef"),
				"Repository",
				"can not change githubScopeRef of an existing pool. Old Kind: Organization, new Kind: Repository",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateGitHubScope(tt.args.pool, tt.args.oldPool); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateGitHubScope() = %v, want %v", got, tt.want)
			}
		})
	}
}
