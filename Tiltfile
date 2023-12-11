# SPDX-License-Identifier: MIT

# -*- mode: Python -*-
load('ext://cert_manager', 'deploy_cert_manager')

# garm-operator is the name of the kind-cluster and therefore usable as k8s context
allow_k8s_contexts('garm-operator')

# we use the `cert_manager` extension to deploy cert-manager into the kind-cluster
# as the plugin has already well written readiness checks we can use it to wait for
deploy_cert_manager(
    kind_cluster_name='garm-operator', # just for security reasons ;-)
    version='v1.12.0' # the version of cert-manager to deploy
)

templated_yaml = kustomize('config/overlays/local')
k8s_yaml(templated_yaml)

# for not having uncategorized resources in the tilt ui
# we explicitly define the resources we want to have grouped together
k8s_resource(
    "garm-operator-controller-manager",
    objects=[
        'garm-operator-system:namespace',
        'enterprises.garm-operator.mercedes-benz.com:customresourcedefinition',
        'images.garm-operator.mercedes-benz.com:customresourcedefinition',
        'organizations.garm-operator.mercedes-benz.com:customresourcedefinition',
        'pools.garm-operator.mercedes-benz.com:customresourcedefinition',
        'runners.garm-operator.mercedes-benz.com:customresourcedefinition',
        'repositories.garm-operator.mercedes-benz.com:customresourcedefinition',
        'garm-operator-controller-manager:serviceaccount',
        'garm-operator-leader-election-role:role',
        'garm-operator-manager-role:clusterrole',
        'garm-operator-manager-role:role',
        'garm-operator-leader-election-rolebinding:rolebinding',
        'garm-operator-manager-rolebinding:rolebinding',
        'garm-operator-serving-cert:certificate',
        'garm-operator-selfsigned-issuer:issuer',
        'garm-operator-validating-webhook-configuration:validatingwebhookconfiguration',
        ],
    labels=["operator"],
)

docker_build('localhost:5000/controller', '.')
