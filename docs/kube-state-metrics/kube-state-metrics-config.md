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
```
# HELP garm_operator_image_created Unix creation timestamp.
# TYPE garm_operator_image_created gauge

# HELP garm_operator_image_info Information about an image.
# TYPE garm_operator_image_info info
```

#### Repository
```
# HELP garm_operator_repo_created Unix creation timestamp.
# TYPE garm_operator_repo_created gauge

# HELP garm_operator_repo_pool_manager_running Whether the repositories poolManager is running.
# TYPE garm_operator_repo_pool_manager_running gauge

# HELP garm_operator_repo_info Information about a repository.
# TYPE garm_operator_repo_info info

# HELP garm_operator_repo_annotation_paused_info Whether the repo reconciliation is paused.
# TYPE garm_operator_repo_annotation_paused_info info
```

#### Organization
```
# HELP garm_operator_org_created Unix creation timestamp.
# TYPE garm_operator_org_created gauge

# HELP garm_operator_org_pool_manager_running Whether the orgs poolManager is running.
# TYPE garm_operator_org_pool_manager_running gauge

# HELP garm_operator_org_info Information about an enterprise.
# TYPE garm_operator_org_info info

# HELP garm_operator_org_annotation_paused_info Whether the org reconciliation is paused.
# TYPE garm_operator_org_annotation_paused_info info
```

#### Enterprise
```
# HELP garm_operator_enterprise_created Unix creation timestamp.
# TYPE garm_operator_enterprise_created gauge

# HELP garm_operator_enterprise_pool_manager_running Whether the enterprises poolManager is running.
# TYPE garm_operator_enterprise_pool_manager_running gauge

# HELP garm_operator_enterprise_info Information about an enterprise.
# TYPE garm_operator_enterprise_info info

# HELP garm_operator_enterprise_annotation_paused_info Whether the enterprise reconciliation is paused.
# TYPE garm_operator_enterprise_annotation_paused_info info
```

#### Pool
```
# HELP garm_operator_pool_created Unix creation timestamp.
# TYPE garm_operator_pool_created gauge

# HELP garm_operator_pool_info Information about a pool.
# TYPE garm_operator_pool_info info

# HELP garm_operator_pool_annotation_paused_info Whether the pool reconciliation is paused.
# TYPE garm_operator_pool_annotation_paused_info info
```

