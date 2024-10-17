// SPDX-License-Identifier: MIT
package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/mercedes-benz/garm-operator/api/v1beta1"
)

var _ conversion.Convertible = &Runner{}

func (r *Runner) ConvertTo(dstRaw conversion.Hub) error {
	return Convert_v1alpha1_Runner_To_v1beta1_Runner(r, dstRaw.(*v1beta1.Runner), nil)
}

func (r *Runner) ConvertFrom(dstRaw conversion.Hub) error {
	return Convert_v1beta1_Runner_To_v1alpha1_Runner(dstRaw.(*v1beta1.Runner), r, nil)
}
