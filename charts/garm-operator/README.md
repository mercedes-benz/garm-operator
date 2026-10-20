<!-- SPDX-License-Identifier: MIT -->

# GARM Operator Helm Chart

Helm chart for installing and configuring the [garm-operator](https://github.com/mercedes-benz/garm-operator) in Kubernetes.

## Prerequisites

- Kubernetes 1.27+
- Helm 3.8.0+
- [cert-manager](https://cert-manager.io/) installed (required for admission webhook TLS certificates)
- A running [GARM](https://github.com/cloudbase/garm) server reachable from your Kubernetes cluster

## Installing the Chart

### Using an Existing Secret for GARM Password (Recommended)

First, create a Kubernetes Secret containing your GARM password:

```bash
kubectl create secret generic garm-credentials \
  --namespace garm-operator-system \
  --from-literal=password='YOUR_GARM_PASSWORD'
```

Then install the chart referencing the secret:

```bash
helm install garm-operator ./charts/garm-operator \
  --namespace garm-operator-system \
  --create-namespace \
  --set garm.server="http://garm-server:9997" \
  --set garm.username="admin" \
  --set garm.existingSecret="garm-credentials"
```

### Providing Credentials Directly in Values

```bash
helm install garm-operator ./charts/garm-operator \
  --namespace garm-operator-system \
  --create-namespace \
  --set garm.server="http://garm-server:9997" \
  --set garm.username="admin" \
  --set garm.password="YOUR_GARM_PASSWORD"
```

## Values Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `garm.server` | URL of the GARM server | `""` |
| `garm.username` | GARM admin username | `""` |
| `garm.password` | GARM admin password (ignored if `garm.existingSecret` is set) | `""` |
| `garm.existingSecret` | Name of an existing Secret containing GARM password | `""` |
| `garm.secretKeys.passwordKey` | Key in secret containing password | `password` |
| `garm.secretKeys.usernameKey` | Key in secret containing username | `username` |
| `garm.secretKeys.serverKey` | Key in secret containing server URL | `server` |
| `garm.init` | Initialize GARM on startup | `false` |
| `garm.email` | GARM admin email (required if `garm.init` is true) | `""` |
| `operator.watchNamespace` | Namespace to watch for CRDs (empty watches all) | `""` |
| `operator.metricsBindAddress` | Metrics endpoint bind address | `":8080"` |
| `operator.healthProbeBindAddress` | Health probe bind address | `":8081"` |
| `operator.leaderElection` | Enable leader election | `false` |
| `operator.syncPeriod` | Controller manager sync period | `"5m0s"` |
| `operator.syncRunnersInterval` | Runner sync interval | `"5m0s"` |
| `operator.minIdleRunnersAge` | Minimum idle runners age before deletion | `"5m0s"` |
| `operator.runnerConcurrency` | Runner reconciler concurrency | `20` |
| `operator.repositoryConcurrency` | Repository reconciler concurrency | `5` |
| `operator.organizationConcurrency` | Organization reconciler concurrency | `3` |
| `operator.enterpriseConcurrency` | Enterprise reconciler concurrency | `1` |
| `operator.poolConcurrency` | Pool reconciler concurrency | `10` |
| `operator.runnerReconciliation` | Enable runner reconciliation loop | `true` |
| `operator.logVerbosityLevel` | Log verbosity level (0-5) | `0` |
| `operator.extraArgs` | Extra arguments passed to manager | `[]` |
| `operator.extraEnv` | Extra environment variables | `[]` |
| `operator.extraEnvFrom` | Extra environment variables from secrets/configmaps | `[]` |
| `manager.replicas` | Number of operator replicas | `1` |
| `manager.image.repository` | Image repository | `ghcr.io/mercedes-benz/garm-operator/garm-operator` |
| `manager.image.tag` | Image tag (defaults to `Chart.appVersion`) | `""` |
| `manager.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `manager.resources` | Resource requests and limits | limits: 500m/128Mi, requests: 10m/64Mi |
| `serviceAccount.create` | Whether to create a ServiceAccount | `true` |
| `rbac.create` | Whether to create RBAC resources | `true` |
| `crd.enable` | Whether to install CRDs | `true` |
| `crd.keep` | Retain CRDs on chart uninstall (`helm.sh/resource-policy: keep`) | `true` |
| `webhook.enable` | Enable admission webhooks | `true` |
| `webhook.port` | Admission webhook container port | `9443` |
| `certManager.enable` | Enable cert-manager certificate generation for webhooks | `true` |
| `certManager.issuerRef` | Custom Issuer reference (defaults to self-signed issuer) | `{}` |
| `prometheus.enable` | Enable Prometheus ServiceMonitor | `false` |
| `kubeStateMetrics.createConfigMap` | Deploy CustomResourceStateMetrics ConfigMap | `true` |

## Webhook and Certificates

By default, the chart enables admission webhooks and uses `cert-manager` with a self-signed issuer to provision webhook certificates. If you use an existing ClusterIssuer, specify `certManager.issuerRef`:

```yaml
certManager:
  enable: true
  issuerRef:
    group: cert-manager.io
    kind: ClusterIssuer
    name: my-cluster-issuer
```

## Uninstalling the Chart

```bash
helm uninstall garm-operator --namespace garm-operator-system
```

By default, Custom Resource Definitions (CRDs) have the `helm.sh/resource-policy: keep` annotation enabled, protecting your GARM custom resources from accidental deletion.
