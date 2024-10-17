// SPDX-License-Identifier: MIT
package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/mercedes-benz/garm-operator/api/v1beta1"
)

var _ conversion.Convertible = &Image{}

func (i *Image) ConvertTo(dstRaw conversion.Hub) error {
	return Convert_v1alpha1_Image_To_v1beta1_Image(i, dstRaw.(*v1beta1.Image), nil)
}

func (i *Image) ConvertFrom(dstRaw conversion.Hub) error {
	return Convert_v1beta1_Image_To_v1alpha1_Image(dstRaw.(*v1beta1.Image), i, nil)
}
