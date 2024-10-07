package v1alpha1

import (
	"github.com/mercedes-benz/garm-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

var _ conversion.Convertible = &Image{}

func (i *Image) ConvertTo(dstRaw conversion.Hub) error {
	return Convert_v1alpha1_Image_To_v1beta1_Image(i, dstRaw.(*v1beta1.Image), nil)
}

func (i *Image) ConvertFrom(dstRaw conversion.Hub) error {
	return Convert_v1beta1_Image_To_v1alpha1_Image(dstRaw.(*v1beta1.Image), i, nil)
}
