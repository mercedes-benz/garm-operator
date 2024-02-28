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

#### Image
Metric name | Type  | Description                 | Unit (where applicable) |
:-----------|:------|:----------------------------|:-----------------------|
`garm_operator_image_created` | Gauge | Unix creation timestamp.    | seconds |
`garm_operator_image_info` | Info  | Information about an image. | |
<br>

#### Repository
Metric name | Type  | Description                                      | Unit (where applicable) |
:-----------|:------|:-------------------------------------------------|:-----------------------|
`garm_operator_repo_created` | Gauge | Unix creation timestamp.                         | seconds |
`garm_operator_repo_pool_manager_running` | Gauge | Whether the repositories poolManager is running. | |
`garm_operator_repo_info` | Info  | Information about a repository.                  | |
`garm_operator_repo_annotation_paused_info` | Info  | Whether the repo reconciliation is paused.       | |
<br>

#### Organization
Metric name | Type  | Description                               | Unit (where applicable) |
:-----------|:------|:------------------------------------------|:-----------------------|
`garm_operator_org_created` | Gauge | Unix creation timestamp.                  | seconds |
`garm_operator_org_pool_manager_running` | Gauge | Whether the orgs poolManager is running.  | |
`garm_operator_org_info` | Info  | Information about an organization.        | |
`garm_operator_org_annotation_paused_info` | Info  | Whether the org reconciliation is paused. | |
<br>

#### Enterprise
Metric name | Type  | Description                                      | Unit (where applicable) |
:-----------|:------|:-------------------------------------------------|:-----------------------|
`garm_operator_enterprise_created` | Gauge | Unix creation timestamp.                         | seconds |
`garm_operator_enterprise_pool_manager_running` | Gauge | Whether the enterprises poolManager is running.  | |
`garm_operator_enterprise_info` | Info  | Information about an enterprise.                 | |
`garm_operator_enterprise_annotation_paused_info` | Info  | Whether the enterprise reconciliation is paused. | |
<br>

#### Pool
Metric name | Type  | Description                                | Unit (where applicable) |
:-----------|:------|:-------------------------------------------|:-----------------------|
`garm_operator_pool_created` | Gauge | Unix creation timestamp.                   | seconds |
`garm_operator_pool_info` | Gauge | Information about a pool.                  | |
`garm_operator_pool_annotation_paused_info` | Info  | Whether the pool reconciliation is paused. | |


