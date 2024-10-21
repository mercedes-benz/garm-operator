// SPDX-License-Identifier: MIT
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/mercedes-benz/garm-operator/api/v1beta1"
)

var _ conversion.Convertible = &Repository{}

func (r *Repository) ConvertTo(dstRaw conversion.Hub) error {
	return Convert_v1alpha1_Repository_To_v1beta1_Repository(r, dstRaw.(*v1beta1.Repository), nil)
}

func (r *Repository) ConvertFrom(dstRaw conversion.Hub) error {
	return Convert_v1beta1_Repository_To_v1alpha1_Repository(dstRaw.(*v1beta1.Repository), r, nil)
}

func Convert_v1alpha1_RepositorySpec_To_v1beta1_RepositorySpec(in *RepositorySpec, out *v1beta1.RepositorySpec, s apiconversion.Scope) error {
	out.CredentialsRef = corev1.TypedLocalObjectReference{
		Name:     in.CredentialsName,
		Kind:     "GitHubCredential",
		APIGroup: &v1beta1.GroupVersion.Group,
	}

	return autoConvert_v1alpha1_RepositorySpec_To_v1beta1_RepositorySpec(in, out, s)
}

func Convert_v1beta1_RepositorySpec_To_v1alpha1_RepositorySpec(in *v1beta1.RepositorySpec, out *RepositorySpec, s apiconversion.Scope) error {
	out.CredentialsName = in.CredentialsRef.Name

	return autoConvert_v1beta1_RepositorySpec_To_v1alpha1_RepositorySpec(in, out, s)
}
