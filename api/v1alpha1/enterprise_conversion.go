package v1alpha1

import (
	"github.com/mercedes-benz/garm-operator/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
)

func Convert_v1alpha1_EnterpriseSpec_To_v1beta1_EnterpriseSpec(in *EnterpriseSpec, out *v1beta1.EnterpriseSpec, s apiconversion.Scope) error {
	out.CredentialsRef = v1.TypedLocalObjectReference{
		Name:     in.CredentialsName,
		Kind:     "GitHubCredentials",
		APIGroup: &v1beta1.GroupVersion.Group,
	}

	return autoConvert_v1alpha1_EnterpriseSpec_To_v1beta1_EnterpriseSpec(in, out, s)
}

func Convert_v1beta1_EnterpriseSpec_To_v1alpha1_EnterpriseSpec(in *v1beta1.EnterpriseSpec, out *EnterpriseSpec, s apiconversion.Scope) error {
	out.CredentialsName = in.CredentialsRef.Name

	return autoConvert_v1beta1_EnterpriseSpec_To_v1alpha1_EnterpriseSpec(in, out, s)
}
