# -*- mode: Python -*-

# garm-operator is the name of the kind-cluster and therefore usable as k8s context
allow_k8s_contexts('garm-operator')

# TODO: evaluate/test the tilt/kubebuilder extension (https://github.com/tilt-dev/tilt-extensions/tree/master/kubebuilder)
#load('ext://kubebuilder', 'kubebuilder') 
#kubebuilder("mercedes-benz.com", "garm-operator", "v1alpha1", "Enterprise")

# we have to define the `go build and kustomize apply step` in tilt as well as in the makefile
# as it's not possible to define a build-dependency with external local_resources, e.g. an existing (and maintained) make target
# local_resources are only run during initial tilt up or by running them manually via tilt-ui afterwards
# see tilt-issue:
# no way to express build dependencies in Tilt - https://github.com/tilt-dev/tilt/issues/3048

local_resource(
    "install CRDs",
    cmd="make install",
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=True,
    labels=["makefile"],
)

# TODO: evaulate if the `kustomize edit set image controller=${IMG}` command could be run within tilt
# to not have the localhost:5000/controller defined as Kustomization in config/manager/kustomization.yaml
# 
# also the disabled --leader-elect flag should be reverted but it helps to speed up pod start time in local env
# we have to change/improve this later once we push to mb-harbor and deploy into CaaS
templated_yaml = kustomize('config/default')
k8s_yaml(templated_yaml)

docker_build('localhost:5000/controller', '.')
