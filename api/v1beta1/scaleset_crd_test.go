// SPDX-License-Identifier: MIT

package v1beta1

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	"sigs.k8s.io/yaml"
)

// Exercise Kubernetes' actual admission checks, including CEL installation cost.
func TestScaleSetCRDValidation(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "config", "crd", "bases", "garm-operator.mercedes-benz.com_scalesets.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var external apiextensionsv1.CustomResourceDefinition
	if err := yaml.UnmarshalStrict(data, &external); err != nil {
		t.Fatal(err)
	}
	apiextensionsv1.SetDefaults_CustomResourceDefinition(&external)
	var crd apiextensions.CustomResourceDefinition
	if err := apiextensionsv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(&external, &crd, nil); err != nil {
		t.Fatal(err)
	}
	crd.Status.StoredVersions = []string{"v1beta1"}
	if errs := validation.ValidateCustomResourceDefinition(context.Background(), &crd); len(errs) != 0 {
		t.Fatalf("invalid ScaleSet CRD: %v", errs.ToAggregate())
	}
}
