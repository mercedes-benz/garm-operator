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
- [2. Create a Garm Server Configuration](#2-create-a-garm-server-configuration)
- [3. Create a Github Endpoint Configuration](#3-create-a-github-endpoint-configuration)
- [4. Create a Github Credentials Configuration](#4-create-a-github-credentials-configuration)
- [5. Enterprise / Organization / Repository CR](#5-enterprise--organization--repository-cr)
- [6. Spin up a <code>Pool</code> with runners](#6-spin-up-a-pool-with-runners)
<!-- /toc -->

## 1. Webhook Secret
First, apply a normal [kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/) which holds your previously, in your `GitHub webhook` configured secret as a `base64`encoded value:
```bash
$ cat <<EOF | kubectl apply -f -
---
apiVersion: v1
kind: Secret
metadata:
  name: webhook-secret
  namespace: garm-operator-system
data:
  webhookSecret: bXlzdXBlcnNlY3JldHdlYmhvb2tzZWNyZXQ= #mysupersecretwebhooksecret
EOF
```

## 2. Create a Garm Server Configuration
Next, apply a `GarmServerConfig` CR to configure your `garm-server` instance. (Please read [here](https://github.com/cloudbase/garm/blob/main/doc/using_garm.md#updating-controller-settings) about the different urls and which one has to be reachable from a runner or from Github)

```bash
$ cat << EOF | kubectl apply -f -
---
apiVersion: garm-operator.mercedes-benz.com/v1beta1
kind: GarmServerConfig
metadata:
  name: garm-server-config
  namespace: garm-operator-system
spec:
  callbackUrl: http://garm-server.garm-server.svc:9997/api/v1/callbacks
  metadataUrl: http://garm-server.garm-server.svc:9997/api/v1/metadata
  webhookUrl: http://garm-server.garm-server.svc:9997/webhook
```

After applying your `GarmServerConfig` CR, you should see a populated `.status.id` field when querying with `kubectl`. 
This `ID` comes from the `garm-server`. Once `garm` is able to create a resource, every object will receive a unique `ID` which gets reflected back to the applied `GarmServerConfig` CR.
```bash
$ kubectl get garmserverconfig
NAME                 ID                                     VERSION          AGE
garm-server-config   d85471e3-b234-4cd0-9b10-8fc8c1794b40   v0.1.5           8d
```

## 3. Create a Github Endpoint Configuration
Garm is able to handle multiple Github endpoints. You can configure each of them with the `GithubEndpoint` CR.

> [!IMPORTANT]: Garm itself ships with a default Github endpoint configuration. As we do not want to reflect the existing configuration in the `garm-operator` we need to create a new one. 
> The default configuration is write protected and can not be modified.
```bash
$ cat << EOF | kubectl apply -f -
---
apiVersion: garm-operator.mercedes-benz.com/v1beta1
kind: GitHubEndpoint
metadata:
  name: github
  namespace: garm-operator-system
spec:
  description: "github.com"
  apiBaseUrl: "https://api.github.com"
  uploadBaseUrl: "https://uploads.github.com"
  baseUrl: "https://github.com"
```

After applying your `GitHubEndpoint` CR, you should see the endpoint configuration in `ready=true` state when querying with `kubectl`.
```bash
$ kubectl get githubendpoint
NAME                URL                               READY   ERROR   AGE
github              https://api.github.com            True            3d2h
```

## 4. Create a Github Credentials Configuration
As Garm needs to authenticate against Github, you need to create a `GitHubCredentials` CR. This CR holds either the `Personal Access Token` (PAT) or an App configuration which is used to authenticate against Github (for further information, please read the [garm documentation](https://github.com/cloudbase/garm/blob/v0.1.5/doc/github_credentials.md#adding-github-credentials))

```bash
$ cat << EOF | kubectl apply -f -
---
apiVersion: garm-operator.mercedes-benz.com/v1beta1
kind: GitHubCredential
metadata:
  name: github-pat
  namespace: garm-operator-system
spec:
  description: credentials for mercedes-benz github
  endpointRef:
    apiGroup: garm-operator.mercedes-benz.com
    kind: GitHubEndpoint
    name: github
  authType: pat
  secretRef:
    name: github-pat
    key: token
---
apiVersion: v1
kind: Secret
metadata:
  name: github-pat
  namespace: garm-operator-system
data:
  token: <base64 encoded token>
```

After applying your `GitHubCredential` CR, you should see the endpoint configuration in `ready=true` state when querying with `kubectl`.
```bash
$ kubectl get githubcredential -o wide
NAME         ID    READY   ERROR   AUTHTYPE   GITHUBENDPOINT      AGE
github-pat   2     True            pat        github-enterprise   2d22h
```

## 5. Enterprise / Organization / Repository CR
Depending on which `GitHub scope` you registered your `webhook` and want to spin runners, apply one of the following `Enterprise / Organization / Repository CRs`.
See [/config/samples/](../config/samples) for more example `CustomResources`.

In the following we are configuring our `garm-server`, so it can spin up runners on an `Organization scope` in GitHub.

```bash
$ cat <<EOF | kubectl apply -f -
---
apiVersion: garm-operator.mercedes-benz.com/v1beta1
kind: Organization
metadata:
  name: my-org
  namespace: garm-operator-system
spec:
  credentialsRef:
    apiGroup: garm-operator.mercedes-benz.com                     
    kind: GitHubCredentials
    name: github-pat
  webhookSecretRef:
    key: "webhookSecret"
    name: "webhook-secret"
EOF
```

After applying your `Enterprise / Organization / Repository CR`, you should see a populated `.status.id` field when querying with `kubectl`. 
This `ID` comes from the `garm-server`, after syncing the `Enterprise` object to its internal database and which gets reflected back to the applied `Enterprise CR`.
```bash
$ kubectl get org -o wide

NAME        ID                                     READY   ERROR   AGE
my-org      d6afb512-77d0-45d2-b8b3-b94f3dc62511   true            1m
```

## 6. Spin up a `Pool` with runners
To spin up a Pool, you need to apply an `Image CR` first. Essentially one `Image CR` can be referenced by multiple `Pool CRs`. Each `Image CR` holds an image tag, which
the associated `Provider` of the `Pool` can create a `Runner Instance` off.

```bash
$ cat <<EOF | kubectl apply -f -
---
apiVersion: garm-operator.mercedes-benz.com/v1beta1
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
apiVersion: garm-operator.mercedes-benz.com/v1beta1
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
    kind: Organization
    name: my-org
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