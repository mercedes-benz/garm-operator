package v1alpha1

import (
	"github.com/mercedes-benz/garm-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

var _ conversion.Convertible = &Pool{}

func (p *Pool) ConvertTo(dstRaw conversion.Hub) error {
	return Convert_v1alpha1_Pool_To_v1beta1_Pool(p, dstRaw.(*v1beta1.Pool), nil)
}

func (p *Pool) ConvertFrom(dstRaw conversion.Hub) error {
	return Convert_v1beta1_Pool_To_v1alpha1_Pool(dstRaw.(*v1beta1.Pool), p, nil)
}
