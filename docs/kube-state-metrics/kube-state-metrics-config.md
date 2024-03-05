<!-- SPDX-License-Identifier: MIT -->

# Kube State Metrics Configuration

[Here](../../config/kube-state-metrics/configmap.yaml) you will find a sample configuration for a kube-state-metrics agent to expose metrics of `garm-operators` custom resources.
If you are using the official [helm chart](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-state-metrics) you can place the contents of `.data.config.yaml` into your helm-charts `values.yaml` file like so:

```yaml
extraArgs:
  - --custom-resource-state-only=true

# Enabling support for customResourceState, will create a configMap including your config that will be read from kube-state-metrics
customResourceState:
  enabled: true
  config:
    kind: CustomResourceStateMetrics
    spec:
      resources:
        - groupVersionKind:
            group: garm-operator.mercedes-benz.com
            kind: "Enterprise"
            version: "v1alpha1"
  ...
```
> Note: this is only a fraction of the kube-state-metrics values file!

# Metrics

The following metrics are exposed with this kube-state-metrics configuration:

### Image
Metric name | Type  | Description                 | Unit (where applicable) |
:-----------|:------|:----------------------------|:------------------------|
`garm_operator_image_created` | Gauge | Unix creation timestamp.    | seconds                 |
`garm_operator_image_info` | Info  | Information about an image. |                         |

**Example**
```
garm_operator_image_created{crd_type="image",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Image",customresource_version="v1alpha1",name="runner-default"} 1.708425683e+09
garm_operator_image_info{crd_type="image",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Image",customresource_version="v1alpha1",name="runner-default",tag="runner:linux-ubuntu-22.04-arm64"} 1
```
<br>

### Repository
Metric name | Type  | Description                                                          | Unit (where applicable) |
:-----------|:------|:---------------------------------------------------------------------|:------------------------|
`garm_operator_repo_created` | Gauge | Unix creation timestamp.                                             | seconds                 |
`garm_operator_repo_info` | Info  | Information about a repository.                                      |                         |
`garm_operator_repo_annotation_paused_info` | Info  | Whether the repo reconciliation is paused.                           |                         |
`garm_operator_repo_status_conditions` | Gauge | Displays whether status of each possible condition is True or False. |                         |

**Example**
```
garm_operator_repo_created{crd_type="repository",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1alpha1",name="my-repo"} 1.709127881e+09
garm_operator_repo_info{crd_type="repository",credentialsName="github-pat",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1alpha1",id="b33b7f1c-9637-40b5-813f-bba9ec72e92f",name="my-repo",owner="cnt-dev",webhookSecretRefKey="webhookSecret",webhookSecretRefName="org-webhook-secret"} 1
garm_operator_repo_annotation_paused_info{crd_type="repository",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1alpha1",name="my-repo",paused_value="true"} 1

garm_operator_repo_status_conditions{crd_type="repository",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1alpha1",name="my-repo",type="PoolManager"} 1
garm_operator_repo_status_conditions{crd_type="repository",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1alpha1",name="rthalho-test",type="Ready"} 1
garm_operator_repo_status_conditions{crd_type="repository",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1alpha1",name="rthalho-test",type="SecretReference"} 1
```
<br>

### Organization
Metric name | Type  | Description                               | Unit (where applicable) |
:-----------|:------|:------------------------------------------|:------------------------|
`garm_operator_org_created` | Gauge | Unix creation timestamp.                  | seconds                 |
`garm_operator_org_info` | Info  | Information about an organization.        |                         |
`garm_operator_org_annotation_paused_info` | Info  | Whether the org reconciliation is paused. |                         |
`garm_operator_org_status_conditions` | Gauge | Displays whether status of each possible condition is True or False.       |                         |

**Example**
```
garm_operator_org_created{crd_type="organization",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1alpha1",name="github-actions"} 1.708425683e+09
garm_operator_org_info{crd_type="organization",credentialsName="github-pat",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1alpha1",id="3a407238-4599-457f-a9f9-dfe0b294219d",name="github-actions",webhookSecretRefKey="webhookSecret",webhookSecretRefName="org-webhook-secret"} 1
garm_operator_org_annotation_paused_info{crd_type="organization",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1alpha1",name="github-actions",paused_value="true"} 1

garm_operator_org_status_conditions{crd_type="organization",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1alpha1",name="github-actions",type="PoolManager"} 1
garm_operator_org_status_conditions{crd_type="organization",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1alpha1",name="github-actions",type="Ready"} 1
garm_operator_org_status_conditions{crd_type="organization",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1alpha1",name="github-actions",type="SecretReference"} 1
```
<br>

### Enterprise
Metric name | Type  | Description                                      | Unit (where applicable) |
:-----------|:------|:-------------------------------------------------|:------------------------|
`garm_operator_enterprise_created` | Gauge | Unix creation timestamp.                         | seconds                 |
`garm_operator_enterprise_info` | Info  | Information about an enterprise.                 |                         |
`garm_operator_enterprise_annotation_paused_info` | Info  | Whether the enterprise reconciliation is paused. |                         |
`garm_operator_enterprise_status_conditions` | Gauge | Displays whether status of each possible condition is True or False.       |                         |

**Example**
```
garm_operator_enterprise_created{crd_type="enterprise",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1alpha1",name="mercedes-benz-ag"} 1.708425683e+09
garm_operator_enterprise_info{crd_type="enterprise",credentialsName="github-pat",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1alpha1",id="3a407238-4599-457f-a9f9-dfe0b294219d",name="mercedes-benz-ag",webhookSecretRefKey="webhookSecret",webhookSecretRefName="enterprise-webhook-secret"} 1
garm_operator_enterprise_annotation_paused_info{crd_type="enterprise",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1alpha1",name="mercedes-benz-ag",paused_value="true"} 1

garm_operator_enterprise_status_conditions{crd_type="enterprise",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1alpha1",name="mercedes-benz-ag",type="PoolManager"} 1
garm_operator_enterprise_status_conditions{crd_type="enterprise",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1alpha1",name="mercedes-benz-ag",type="Ready"} 1
garm_operator_enterprise_status_conditions{crd_type="enterprise",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1alpha1",name="mercedes-benz-ag",type="SecretReference"} 1
```
<br>

### Pool
Metric name | Type  | Description                                | Unit (where applicable) |
:-----------|:------|:-------------------------------------------|:------------------------|
`garm_operator_pool_created` | Gauge | Unix creation timestamp.                   | seconds                 |
`garm_operator_pool_info` | Gauge | Information about a pool.                  |                         |         |
`garm_operator_pool_annotation_paused_info` | Info  | Whether the pool reconciliation is paused. |                         |
`garm_operator_repo_status_conditions` | Gauge | Displays whether status of each possible condition is True or False.       |                         |

**Example**
```
garm_operator_pool_created{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1alpha1",name="kubernetes-pool-small-org-github-actions"} 1.708425683e+09
garm_operator_pool_info{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1alpha1",enabled="true",githubRunnerGroup="",id="1e52ec4c-0b78-4451-998b-40092344e5a9",imageName="runner-default-1",longRunningIdleRunners="0",maxRunners="2",minIdleRunners="0",name="kubernetes-pool-medium-org-github-actions",osArch="amd64",osType="linux",providerName="kubernetes_external",runnerBootstrapTimeout="2",runnerPrefix="road-runner-k8s-rthalho",scopeKind="Organization",scopeName="github-actions",tags="[medium k8s-dev garm-operator-dev]"} 1
garm_operator_pool_annotation_paused_info{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1alpha1",name="kubernetes-pool-medium-org-github-actions",paused_value="true"} 1

garm_operator_pool_status_conditions{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1alpha1",name="kubernetes-pool-medium-org-github-actions",type="ImageReference"} 1
garm_operator_pool_status_conditions{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1alpha1",name="kubernetes-pool-medium-org-github-actions",type="Ready"} 1
garm_operator_pool_status_conditions{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1alpha1",name="kubernetes-pool-medium-org-github-actions",type="ScopeReference"} 1
```
