<!-- SPDX-License-Identifier: MIT -->

# kube-state-metrics Configuration

[Here](../../config/kube-state-metrics/configmap.yaml) you will find a sample configuration for [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics) to expose metrics of `garm-operators` custom resources.
If you are using the official [helm chart](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-state-metrics) you can place the contents of `.data.config.yaml` into your helm-charts `values.yaml` file like so:

```yaml
extraArgs:
  - --custom-resource-state-only=true

  rbac:
  extraRules:
    - apiGroups:
        - garm-operator.mercedes-benz.com
      resources:
        - enterprises
        - organizations
        - repositories
        - pools
        - images
        - garmserverconfigs
        - githubendpoints
        - githubcredentials
      verbs:
        - get
        - list
        - watch
    - apiGroups:
        - garm-operator.mercedes-benz.com
      resources:
        - enterprises/status
        - organizations/status
        - repositories/status
        - pools/status
        - images/status
        - garmserverconfigs/status
        - githubendpoints/status
        - githubcredentials/status
      verbs:
        - get
        - list
        - watch

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
            version: "v1beta1"
  [...]
```
> [!NOTE]: this is only a fraction of the kube-state-metrics values file!

# Metrics

The following metrics are exposed with this kube-state-metrics configuration:

### GarmServerConfig
Metric name                                             | Type  | Description                                            | Unit (where applicable)
:-------------------------------------------------------|:------|:-------------------------------------------------------|:-----------------------
`garm_operator_garmserverconfig_created`                | Gauge | Unix creation timestamp.                               | seconds
`garm_operator_garmserverconfig_info`                   | Info  | Information about a garm server config.                |
`garm_operator_garmserverconfig_annotation_paused_info` | Info  | Whether the garmserverconfig reconciliation is paused. |

**Example**
```
garm_operator_garmserverconfig_created{crd_type="garmserverconfig",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="GarmServerConfig",customresource_version="v1beta1",name="garm-server-config",namespace="garm-operator-system"} 1.729081579e+09
garm_operator_garmserverconfig_info{callbackUrl="http://garm-server.garm-server.svc:9997/api/v1/callbacks",controllerId="d85471e3-b234-4cd0-9b10-8fc8c1794b40",crd_type="garmserverconfig",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="GarmServerConfig",customresource_version="v1beta1",metadataUrl="http://garm-server.garm-server.svc:9997/api/v1/metadata",minimumJobAgeBackoff="30",name="garm-server-config",namespace="garm-operator-system",version="v0.1.5",webhookUrl="http://garm-server.garm-server.svc:9997/api/v1/webhook"} 1
```
<br>

### GithubEndpoint
Metric name                                           | Type  | Description                                                          | Unit (where applicable)
:-----------------------------------------------------|:------|:---------------------------------------------------------------------|:-----------------------
`garm_operator_githubendpoint_created`                | Gauge | Unix creation timestamp.                                             | seconds
`garm_operator_githubendpoint_info`                   | Info  | Information about a githubendpoint config.                           |
`garm_operator_githubendpoint_status_conditions`      | Gauge | Displays whether status of each possible condition is True or False. |
`garm_operator_githubendpoint_annotation_paused_info` | Info  | Whether the githubendpoint reconciliation is paused.                 |


**Example**
```
garm_operator_githubendpoint_created{crd_type="githubendpoint",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="GitHubEndpoint",customresource_version="v1beta1",name="github",namespace="garm-operator-system"} 1.729504871e+09
garm_operator_githubendpoint_info{apiBaseUrl="https://api.github.com",baseUrl="https://github.com",crd_type="githubendpoint",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="GitHubEndpoint",customresource_version="v1beta1",name="github",namespace="garm-operator-system",uploadBaseUrl="https://uploads.github.com"} 1
# HELP garm_operator_githubendpoint_status_conditions Displays whether status of each possible condition is True or False.
# TYPE garm_operator_githubendpoint_status_conditions gauge
garm_operator_githubendpoint_status_conditions{crd_type="githubendpoint",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="GitHubEndpoint",customresource_version="v1beta1",name="github",namespace="garm-operator-system",type="Ready"} 1
```
<br>

### GithubCredential
Metric name                                             | Type  | Description                                                          | Unit (where applicable)
:-------------------------------------------------------|:------|:---------------------------------------------------------------------|:-----------------------
`garm_operator_githubcredential_created`                | Gauge | Unix creation timestamp.                                             | seconds
`garm_operator_githubcredential_info`                   | Info  | Information about a githubcredential config.                         |
`garm_operator_githubcredential_status_conditions`      | Gauge | Displays whether status of each possible condition is True or False. |
`garm_operator_githubcredential_annotation_paused_info` | Info  | Whether the githubendpoint reconciliation is paused.                 |


**Example**
```
garm_operator_githubcredential_created{crd_type="githubcredential",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="GitHubCredential",customresource_version="v1beta1",name="github-pat",namespace="garm-operator-system"} 1.72952169e+09
garm_operator_githubcredential_info{apiBaseUrl="https://api.github.com",authType="pat",baseUrl="https://github.com",crd_type="githubcredential",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="GitHubCredential",customresource_version="v1beta1",endpointRefName="github",id="3",name="github-pat",namespace="garm-operator-system",secretRefKey="token",secretRefName="github-pat",uploadBaseUrl="https://uploads.github.com"} 1
garm_operator_githubcredential_status_conditions{crd_type="githubcredential",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="GitHubCredential",customresource_version="v1beta1",name="github-pat",namespace="garm-operator-system",type="EndpointReference"} 1
garm_operator_githubcredential_status_conditions{crd_type="githubcredential",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="GitHubCredential",customresource_version="v1beta1",name="github-pat",namespace="garm-operator-system",type="Ready"} 1
garm_operator_githubcredential_status_conditions{crd_type="githubcredential",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="GitHubCredential",customresource_version="v1beta1",name="github-pat",namespace="garm-operator-system",type="SecretReference"} 1
```
<br>

### Image
Metric name                   | Type  | Description                 | Unit (where applicable)
:-----------------------------|:------|:----------------------------|:-----------------------
`garm_operator_image_created` | Gauge | Unix creation timestamp.    | seconds
`garm_operator_image_info`    | Info  | Information about an image. |

**Example**
```
garm_operator_image_created{crd_type="image",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Image",customresource_version="v1beta1",name="runner-default"} 1.708425683e+09
garm_operator_image_info{crd_type="image",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Image",customresource_version="v1beta1",name="runner-default",tag="runner:linux-ubuntu-22.04-arm64"} 1
```
<br>

### Repository
Metric name                                 | Type  | Description                                                          | Unit (where applicable)
:-------------------------------------------|:------|:---------------------------------------------------------------------|:-----------------------
`garm_operator_repo_created`                | Gauge | Unix creation timestamp.                                             | seconds
`garm_operator_repo_info`                   | Info  | Information about a repository.                                      |
`garm_operator_repo_annotation_paused_info` | Info  | Whether the repo reconciliation is paused.                           |
`garm_operator_repo_status_conditions`      | Gauge | Displays whether status of each possible condition is True or False. |

**Example**
```
garm_operator_repo_created{crd_type="repository",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1beta1",name="my-repo"} 1.709127881e+09
garm_operator_repo_info{crd_type="repository",credentialsRefName="github-pat",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1beta1",id="b33b7f1c-9637-40b5-813f-bba9ec72e92f",name="my-repo",owner="cnt-dev",webhookSecretRefKey="webhookSecret",webhookSecretRefName="org-webhook-secret"} 1
garm_operator_repo_annotation_paused_info{crd_type="repository",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1beta1",name="my-repo",paused_value="true"} 1
garm_operator_repo_status_conditions{crd_type="repository",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1beta1",name="my-repo",type="PoolManager"} 1
garm_operator_repo_status_conditions{crd_type="repository",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1beta1",name="rthalho-test",type="Ready"} 1
garm_operator_repo_status_conditions{crd_type="repository",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Repository",customresource_version="v1beta1",name="rthalho-test",type="SecretReference"} 1
```
<br>

### Organization
Metric name                                | Type  | Description                                                          | Unit (where applicable)
:------------------------------------------|:------|:---------------------------------------------------------------------|:-----------------------
`garm_operator_org_created`                | Gauge | Unix creation timestamp.                                             | seconds
`garm_operator_org_info`                   | Info  | Information about an organization.                                   |
`garm_operator_org_annotation_paused_info` | Info  | Whether the org reconciliation is paused.                            |
`garm_operator_org_status_conditions`      | Gauge | Displays whether status of each possible condition is True or False. |

**Example**
```
garm_operator_org_created{crd_type="organization",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1beta1",name="github-actions"} 1.708425683e+09
garm_operator_org_info{crd_type="organization",credentialsRefName="github-pat",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1beta1",id="3a407238-4599-457f-a9f9-dfe0b294219d",name="github-actions",webhookSecretRefKey="webhookSecret",webhookSecretRefName="org-webhook-secret"} 1
garm_operator_org_annotation_paused_info{crd_type="organization",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1beta1",name="github-actions",paused_value="true"} 1

garm_operator_org_status_conditions{crd_type="organization",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1beta1",name="github-actions",type="PoolManager"} 1
garm_operator_org_status_conditions{crd_type="organization",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1beta1",name="github-actions",type="Ready"} 1
garm_operator_org_status_conditions{crd_type="organization",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Organization",customresource_version="v1beta1",name="github-actions",type="SecretReference"} 1
```
<br>

### Enterprise
Metric name                                       | Type  | Description                                                          | Unit (where applicable)
:-------------------------------------------------|:------|:---------------------------------------------------------------------|:-----------------------
`garm_operator_enterprise_created`                | Gauge | Unix creation timestamp.                                             | seconds
`garm_operator_enterprise_info`                   | Info  | Information about an enterprise.                                     |
`garm_operator_enterprise_annotation_paused_info` | Info  | Whether the enterprise reconciliation is paused.                     |
`garm_operator_enterprise_status_conditions`      | Gauge | Displays whether status of each possible condition is True or False. |

**Example**
```
garm_operator_enterprise_created{crd_type="enterprise",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1beta1",name="mercedes-benz-ag"} 1.708425683e+09
garm_operator_enterprise_info{crd_type="enterprise",credentialsRefName="github-pat",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1beta1",id="3a407238-4599-457f-a9f9-dfe0b294219d",name="mercedes-benz-ag",webhookSecretRefKey="webhookSecret",webhookSecretRefName="enterprise-webhook-secret"} 1
garm_operator_enterprise_annotation_paused_info{crd_type="enterprise",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1beta1",name="mercedes-benz-ag",paused_value="true"} 1

garm_operator_enterprise_status_conditions{crd_type="enterprise",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1beta1",name="mercedes-benz-ag",type="PoolManager"} 1
garm_operator_enterprise_status_conditions{crd_type="enterprise",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1beta1",name="mercedes-benz-ag",type="Ready"} 1
garm_operator_enterprise_status_conditions{crd_type="enterprise",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Enterprise",customresource_version="v1beta1",name="mercedes-benz-ag",type="SecretReference"} 1
```
<br>

### Pool
Metric name                  | Type  | Description              | Unit (where applicable)
:----------------------------|:------|:-------------------------|:-----------------------
`garm_operator_pool_created` | Gauge | Unix creation timestamp. | seconds
`garm_operator_pool_info` | Gauge | Information about a pool.                  |                         |         |
`garm_operator_pool_annotation_paused_info` | Info  | Whether the pool reconciliation is paused. |                         |
`garm_operator_repo_status_conditions` | Gauge | Displays whether status of each possible condition is True or False.       |                         |

**Example**
```
garm_operator_pool_created{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1beta1",name="kubernetes-pool-small-org-github-actions"} 1.708425683e+09
garm_operator_pool_info{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1beta1",enabled="true",githubRunnerGroup="",id="1e52ec4c-0b78-4451-998b-40092344e5a9",imageName="runner-default-1",longRunningIdleRunners="0",maxRunners="2",minIdleRunners="0",name="kubernetes-pool-medium-org-github-actions",osArch="amd64",osType="linux",providerName="kubernetes_external",runnerBootstrapTimeout="2",runnerPrefix="road-runner-k8s-rthalho",scopeKind="Organization",scopeName="github-actions",tags="[medium k8s-dev garm-operator-dev]"} 1
garm_operator_pool_annotation_paused_info{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1beta1",name="kubernetes-pool-medium-org-github-actions",paused_value="true"} 1

garm_operator_pool_status_conditions{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1beta1",name="kubernetes-pool-medium-org-github-actions",type="ImageReference"} 1
garm_operator_pool_status_conditions{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1beta1",name="kubernetes-pool-medium-org-github-actions",type="Ready"} 1
garm_operator_pool_status_conditions{crd_type="pool",customresource_group="garm-operator.mercedes-benz.com",customresource_kind="Pool",customresource_version="v1beta1",name="kubernetes-pool-medium-org-github-actions",type="ScopeReference"} 1
```
