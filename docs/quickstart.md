<!-- SPDX-License-Identifier: MIT -->

# Quickstart

To get started, you need to have the following prerequisites in place:
1. A running `garm-server` instance in your kubernetes cluster or somewhere else (needs to be reachable by garm-operator)
2. A running `garm-operator` instance in your kubernetes cluster
3. A configured Enterprise, Organization or Repository `webhook` on your GitHub Instance. [See official Garm Docs](https://github.com/cloudbase/garm/blob/main/doc/webhooks.md)

Each `garm-operator` is tied to one `garm-server`. Make sure to apply the following `CustomResources (CRs)` to the same `namespace`, your `garm-operator` is running in. 
In the following examples, our `garm-operator` is deployed in the namespace `garm-operator-system`

<!-- toc -->
- [1. Webhook Secret](#1-webhook-secret)
- [2. Enterprise / Organization / Repository CR](#2-enterprise--organization--repository-cr)
- [3. Spin up a <code>Pool</code> with runners](#3-spin-up-a-pool-with-runners)
<!-- /toc -->

## 1. Webhook Secret
First, apply a normal [kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/) which holds your previously, in your `GitHub webhook` configured secret as a `base64`encoded value:
```bash
$ cat <<EOF | kubectl apply -f -
---
apiVersion: v1
kind: Secret
metadata:
  name: enterprise-webhook-secret
  namespace: garm-operator-system
data:
  webhookSecret: bXlzdXBlcnNlY3JldHdlYmhvb2tzZWNyZXQ= #mysupersecretwebhooksecret
EOF
```

## 2. Enterprise / Organization / Repository CR
Depending on which `GitHub scope` you registered your `webhook` and want to spin runners, apply one of the following `Enterprise / Organization / Repository CRs`.
See [/config/samples/](../config/samples) for more example `CustomResources`.

In the following we are configuring our `garm-server`, so it can spin up runners on an `Enterprise scope` in GitHub.
1. `.spec.credentialsName` must be the name of your configured GitHub credentials in [config.toml](https://github.com/cloudbase/garm/blob/main/doc/github_credentials.md?plain=1#L25) of your `garm-server`.
```bash
$ cat <<EOF | kubectl apply -f -
---
apiVersion: garm-operator.mercedes-benz.com/v1alpha1
kind: Enterprise
metadata:
  labels:
    app.kubernetes.io/name: enterprise
    app.kubernetes.io/instance: enterprise-sample
    app.kubernetes.io/part-of: garm-operator
  name: enterprise-sample
  namespace: garm-operator-system
spec:
  credentialsName: GitHub-Actions
  webhookSecretRef:
    key: "webhookSecret"
    name: "enterprise-webhook-secret"
EOF
```

After applying your `Enterprise / Organization / Repository CR`, you should see a populated `.status.id` field when querying with `kubectl`. 
This `ID` comes from the `garm-server`, after syncing the `Enterprise` object to its internal database and which gets reflected back to the applied `Enterprise CR`.
```bash
$ kubectl get enterprise -o wide

NAME                ID                                     READY   ERROR   AGE
enterprise-sample   d6afb512-77d0-45d2-b8b3-b94f3dc62511   true            1m
```

## 3. Spin up a `Pool` with runners
To spin up a Pool, you need to apply an `Image CR` first. Essentially one `Image CR` can be referenced by multiple `Pool CRs`. Each `Image CR` holds an image tag, which
the associated `Provider` of the `Pool` can create a `Runner Instance` off.

```bash
$ cat <<EOF | kubectl apply -f -
---
apiVersion: garm-operator.mercedes-benz.com/v1alpha1
kind: Image
metadata:
  labels:
    app.kubernetes.io/name: image
    app.kubernetes.io/instance: image-sample
    app.kubernetes.io/part-of: garm-operator
  name: runner-default
  namespace: garm-operator-system
spec:
  tag: linux-ubuntu-22.04
EOF
```

Next apply a `Pool CR`:
```bash
$ cat <<EOF | kubectl apply -f -
---
apiVersion: garm-operator.mercedes-benz.com/v1alpha1
kind: Pool
metadata:
  labels:
    app.kubernetes.io/instance: pool-sample
    app.kubernetes.io/name: pool
    app.kubernetes.io/part-of: garm-operator
  name: openstack-small-pool-enterprise
  namespace: garm-operator-system
spec:
  githubScopeRef:
    apiGroup: garm-operator.mercedes-benz.com
    kind: Enterprise
    name: enterprise-sample
  enabled: true
  extraSpecs: '{}'
  flavor: small
  githubRunnerGroup: ""
  imageName: runner-default
  maxRunners: 4
  minIdleRunners: 2
  osArch: amd64
  osType: linux
  providerName: openstack
  runnerBootstrapTimeout: 20
  runnerPrefix: ""
  tags:
    - linux
    - small
    - ubuntu
EOF
```
Take care of the following:
1. `.spec.githubScopeRef.name` and `.spec.githubScopeRef.kind` should reference the previously applied `Enterprise / Organization / Repository CR`, so its `Runners` are getting registered in the correct scope.
2. `.spec.providerName` should be the same name as your desired provider configured in your [config.toml](https://github.com/cloudbase/garm/blob/main/doc/providers.md?plain=1#L26) of your `garm-server`.
3. `.spec.imageName` should reference the previously applied `Image CRs` `.metadata.name` field

After that you should see the following output, where `ID` gets reflected back from `garm-server` to the `.status.id` field of your `Pool CR`:

```bash
$ kubectl get pool

NAME                                 ID                                     MINIDLERUNNERS   MAXRUNNERS   AGE
openstack-small-pool-enterprise      0ff3f052-5901-46ac-902c-28f2f38a64ec   2                4            1m
```