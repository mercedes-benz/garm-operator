package v1alpha1

import (
	v1beta1 "github.com/mercedes-benz/garm-operator/api/v1beta1"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
)

func Convert_v1alpha1_OrganizationSpec_To_v1beta1_OrganizationSpec(in *OrganizationSpec, out *v1beta1.OrganizationSpec, s apiconversion.Scope) error {

	err := autoConvert_v1alpha1_OrganizationSpec_To_v1beta1_OrganizationSpec(in, out, s)
	if err != nil {
		return err
	}

	return nil
}
