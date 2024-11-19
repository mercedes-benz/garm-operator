<!-- SPDX-License-Identifier: MIT -->

# compatibility

<!-- toc -->
- [update to operator 0.4.x](#update-to-operator-04x)
  - [1. update <code>garm</code> to version <code>&gt;=0.1.5</code>](#1-update-garm-to-version-015)
  - [2. update <code>garm-operator</code> to version <code>0.4.x</code>](#2-update-garm-operator-to-version-04x)
  - [3. create new <code>CustomResources</code>](#3-create-new-customresources)
<!-- /toc -->

## update to operator 0.4.x

> [!WARNING]
> `garm-operator` in version `0.4.x` is not compatible with `garm` in version `0.1.4`.
> If you connect `garm-operator` in version `0.4.x` to `garm` in version `0.1.4`, 
> the operator will stop working as we check the `garm` version in the operator.

### 1. update `garm` to version `>=0.1.5`

`garm-operator` in version `0.3.x` is compatible with `garm` in version `0.1.5`, if the `garm` server got updated from `0.1.4` to `0.1.5`.
This is because `garm` is doing some migration steps by moving parts of the configuration from the `garm` server to the `garm` API.
These newly introduced API endpoints are not by `garm-operator` in version `0.3.x`.

Therefore it's possible to still work with the existing `CustomResources`.

### 2. update `garm-operator` to version `0.4.x`

Once `garm` got updated to version `0.1.5`, you can update the `garm-operator` to version `0.4.x`.

### 3. create new `CustomResources`

As `garm` moved some configuration parts to the API, you have to create a few new `CustomResources`.
Without these new `CustomResources`, `garm-operator` is not able to reconcile `Enterprises`, `Organizations` and `Repository` objects.

#### 3.1 create a `GarmServerConfig` object

It's now possible to define the `callbackUrl`, `metadataUrl` and `webhookUrl` via a `GarmServerConfig` object. Therefore it's not needed anymore to restart the `garm` server to apply these changes.

```yaml
apiVersion: garm-operator.mercedes-benz.com/v1beta1
kind: GarmServerConfig
metadata:
  name: garm-server-config
  namespace: garm-operator-system
spec:
  callbackUrl: http://<fqdn-to-your-garm-server-instance>/api/v1/callbacks
  metadataUrl: http://<fqdn-to-your-garm-server-instance>/api/v1/metadata
  webhookUrl: http://<fqdn-to-your-garm-server-instance>/api/v1/webhook
```

By running `kubectl get garmserverconfig` you should see the newly created `GarmServerConfig` object.

```bash
NAME                 ID                                     VERSION   AGE
garm-server-config   dd2524b9-0789-499d-ba7d-3dba65cf9d3f   v0.1.5    16h
```

More information about the defined Urls can be found [in the garm documentation](https://github.com/cloudbase/garm/blob/main/doc/using_garm.md#controller-operations).

#### 3.2 create `GitHubEndpoint` objects

The `GitHubEndpoint` objects are used to define the connection to the GitHub API. It's possible to define multiple `GitHubEndpoint` objects to connect to different GitHub instances.

```yaml
apiVersion: garm-operator.mercedes-benz.com/v1beta1
kind: GitHubEndpoint
metadata:
  name: github-enterprise
  namespace: garm-operator-system
spec:
  description: "github"
  apiBaseUrl: "https://github.com"
  uploadBaseUrl: "https://uploads.github.com"
  baseUrl: "https://api.github.com/"
```

By running `kubectl get githubendpoint` you should see the newly created `GitHubEndpoint` object.

```bash
NAME                URL                  READY   AGE
github              https://github.com   True    3m2s
```

More information about the github endpoints can be found [in the garm documentation](https://github.com/cloudbase/garm/blob/main/doc/using_garm.md#github-endpoints)

> [!NOTE]
> `garm` already ships a default `Github` object. But as this object is immutable, and we do not wanted to reflect the object into the `garm-operator`, we decided to create a new `CustomResource` for this. See the [Reflection of the default GitHub endpoint ADR](docs/adrs/github_default_endpoint.md) for more information.

#### 3.2 create `GitHubCredential` objects

With the new `v1beta1` of `Enterprises`, `Organizations` or `Repositories`, the reference to the Github credential has changed.
In the previous `v1alpha1`, the used credential was referenced by the `.spec.credentialsName` field, which then pointed to a credential object, specified in the `garm` server configuration.

With the new `v1beta1`, the reference to the credential is done by the `.spec.credentialRef` field, which points to a `GitHubCredential` object.

> [!NOTE]
> As long as `GitHubCredential` is not applied in `v1beta1`, the `garm-operator` will quit reconciling `Enterprises`, `Organizations` or `Repositories`.
> The current assumption is, that a `GitHubCredential`, which was set in `.spec.credentialsName` in `v1alpha1`, will be created as `GitHubCredential` in `v1beta1` with the same name.
> If this is not the case, the `garm-operator` will not be able to reconcile the `Enterprises`, `Organizations` or `Repositories` objects.
> This state can be seen in the `status.conditions` field of these objects.
> 
> ```yaml
> status:
>   conditions:
>   - lastTransitionTime: "2024-11-14T20:01:27Z"
>     message: GitHubCredential.garm-operator.mercedes-benz.com "github-pat" not found
>     reason: CredentialsRefFailed
>     status: "False"
>     type: Ready
>   - lastTransitionTime: "2024-11-14T20:01:27Z"
>     message: GitHubCredential.garm-operator.mercedes-benz.com "github-pat" not found
>     reason: CredentialsRefFailed
>     status: "False"
>     type: CredentialsRef
>   [...]
> ```

Create the `GitHubCredential` object and a corresponding `Secret` object with the PAT token.

```yaml
apiVersion: garm-operator.mercedes-benz.com/v1beta1
kind: GitHubCredential
metadata:
  name: github-pat
  namespace: garm-operator-system
spec:
  description: PAT for github
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
  token: <base64 encoded PAT>
```

More information about the github credentials can be found [in the garm documentation](https://github.com/cloudbase/garm/blob/main/doc/using_garm.md#github-credentials)

By running `kubectl get githubcredential` you should see the newly created `GitHubCredential` object.

```bash
NAME         ID    READY   AUTHTYPE   GITHUBENDPOINT      AGE
github-pat   1     True    pat        github              3m51s
```

By running `kubectl get githubcredential -o wide` you see all the `Enterprise`, `Organization` or `Repository` objects which are using this `GitHubCredential`.

```bash
NAME         ID    READY   ERROR   AUTHTYPE   GITHUBENDPOINT      REPOSITORIES   ORGANIZATIONS     ENTERPRISES   AGE
github-pat   1     True            pat        github                             ["my-org"]                      4m46s
```