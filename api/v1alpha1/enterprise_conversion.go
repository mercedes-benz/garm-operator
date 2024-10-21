// SPDX-License-Identifier: MIT
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/mercedes-benz/garm-operator/api/v1beta1"
)

var _ conversion.Convertible = &Enterprise{}

func (e *Enterprise) ConvertTo(dstRaw conversion.Hub) error {
	return Convert_v1alpha1_Enterprise_To_v1beta1_Enterprise(e, dstRaw.(*v1beta1.Enterprise), nil)
}

func (e *Enterprise) ConvertFrom(dstRaw conversion.Hub) error {
	return Convert_v1beta1_Enterprise_To_v1alpha1_Enterprise(dstRaw.(*v1beta1.Enterprise), e, nil)
}

func Convert_v1alpha1_EnterpriseSpec_To_v1beta1_EnterpriseSpec(in *EnterpriseSpec, out *v1beta1.EnterpriseSpec, s apiconversion.Scope) error {
	out.CredentialsRef = corev1.TypedLocalObjectReference{
		Name:     in.CredentialsName,
		Kind:     "GitHubCredential",
		APIGroup: &v1beta1.GroupVersion.Group,
	}

	return autoConvert_v1alpha1_EnterpriseSpec_To_v1beta1_EnterpriseSpec(in, out, s)
}

func Convert_v1beta1_EnterpriseSpec_To_v1alpha1_EnterpriseSpec(in *v1beta1.EnterpriseSpec, out *EnterpriseSpec, s apiconversion.Scope) error {
	out.CredentialsName = in.CredentialsRef.Name

	return autoConvert_v1beta1_EnterpriseSpec_To_v1alpha1_EnterpriseSpec(in, out, s)
}
