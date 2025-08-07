// SPDX-License-Identifier: MIT

package v1beta1

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func Test_validateRepoOwnerName(t *testing.T) {
	type args struct {
		repo    *Repository
		oldRepo *Repository
	}
	tests := []struct {
		name string
		args args
		want *field.Error
	}{
		{
			name: "no owner name change",
			args: args{
				repo: &Repository{
					Spec: RepositorySpec{
						Owner: "company-org",
					},
				},
				oldRepo: &Repository{
					Spec: RepositorySpec{
						Owner: "company-org",
					},
				},
			},
			want: nil,
		},
		{
			name: "no owner name change",
			args: args{
				repo: &Repository{
					Spec: RepositorySpec{
						Owner: "company-org",
					},
				},
				oldRepo: &Repository{
					Spec: RepositorySpec{
						Owner: "private-org",
					},
				},
			},
			want: field.Invalid(
				field.NewPath("spec").Child("owner"),
				"company-org",
				"can not change owner of an existing repository. Old name: private-org, new name: company-org",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateRepoOwnerName(tt.args.repo, tt.args.oldRepo); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateRepoOwnerName() = %v, want %v", got, tt.want)
			}
		})
	}
}
