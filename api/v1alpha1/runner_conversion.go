// SPDX-License-Identifier: MIT
package v1alpha1

import (
	apiconversion "k8s.io/apimachinery/pkg/conversion"
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

func Convert_v1beta1_RunnerStatus_To_v1alpha1_RunnerStatus(in *v1beta1.RunnerStatus, out *RunnerStatus, s apiconversion.Scope) error {
	out.ID = in.ID
	out.ProviderID = in.ProviderID
	out.AgentID = in.AgentID
	out.Name = in.Name
	out.OSType = in.OSType
	out.OSName = in.OSName
	out.OSVersion = in.OSVersion
	out.OSArch = in.OSArch
	out.Addresses = nil
	out.Status = in.Status
	out.InstanceStatus = in.InstanceStatus
	out.PoolID = in.PoolID
	out.ProviderFault = in.ProviderFault
	out.GitHubRunnerGroup = in.GitHubRunnerGroup

	return autoConvert_v1beta1_RunnerStatus_To_v1alpha1_RunnerStatus(in, out, s)
}
