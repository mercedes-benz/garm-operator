// SPDX-License-Identifier: MIT
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	garmoperatorv1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
)

func (o *Organization) ConvertTo(dstRaw conversion.Hub) error {
	return Convert_v1alpha1_Organization_To_v1beta1_Organization(o, dstRaw.(*garmoperatorv1beta1.Organization), nil)
}

func (o *Organization) ConvertFrom(dstRaw conversion.Hub) error {
	return Convert_v1beta1_Organization_To_v1alpha1_Organization(dstRaw.(*garmoperatorv1beta1.Organization), o, nil)
}

func Convert_v1alpha1_OrganizationSpec_To_v1beta1_OrganizationSpec(in *OrganizationSpec, out *garmoperatorv1beta1.OrganizationSpec, s apiconversion.Scope) error {
	out.CredentialsRef = corev1.TypedLocalObjectReference{
		Name:     in.CredentialsName,
		Kind:     "GitHubCredential",
		APIGroup: &garmoperatorv1beta1.GroupVersion.Group,
	}

	return autoConvert_v1alpha1_OrganizationSpec_To_v1beta1_OrganizationSpec(in, out, s)
}

func Convert_v1beta1_OrganizationSpec_To_v1alpha1_OrganizationSpec(in *garmoperatorv1beta1.OrganizationSpec, out *OrganizationSpec, s apiconversion.Scope) error {
	out.CredentialsName = in.CredentialsRef.Name

	return autoConvert_v1beta1_OrganizationSpec_To_v1alpha1_OrganizationSpec(in, out, s)
}
