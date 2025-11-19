<!-- SPDX-License-Identifier: MIT -->

# Development

<!-- toc -->
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
  - [⚙️ Bootstrap garm-server with garm-provider-k8s for local development](#-bootstrap-garm-server-with-garm-provider-k8s-for-local-development)
  - [🐛 Debugging](#-debugging)
<!-- /toc -->

## Prerequisites

To start developing, you need the following tools installed:

- [Go](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/get-docker/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [Tilt](https://docs.tilt.dev/install.html)

All other remaining tools (e.g. `kustomize`) are getting installed automatically when you start the development environment later on.

## Getting Started

1. Depending on the `garm` server you want to use, you have to specify the server URL and the corresponding username and password in the `config/overlays/local/manager_patch.yaml` file.
1. Start the development environment by running `make tilt-up` in the root directory of this repository.
   
   This will start a local Kubernetes cluster using `kind` (`kind get clusters` will show you a `garm-operator` cluster) and deploy the `garm-operator` into it.

   The `make tilt-up` command will give you the URL to the local tilt environment.
1. Time to start developing. 🎉

### ⚙️ Bootstrap garm-server with garm-provider-k8s for local development

If you need a `garm-server` with a configured [garm-provider-k8s](https://github.com/mercedes-benz/garm-provider-k8s) in your local cluster to spin up some `k8s based runners` for testing, you can do the following:

#### 1. Clone the `garm-provider-k8s` repo:
   ```bash
   git clone https://github.com/mercedes-benz/garm-provider-k8s && cd ./garm-provider-k8s
   ```

#### 2. Generate a GitHub PAT
   Your PAT needs the following permissions:
   If you'll use a PAT (classic), you'll have to grant access for the following scopes. See official [cloudbase/garm](https://github.com/cloudbase/garm/blob/main/doc/github_credentials.md) docs for more information.

   * ```public_repo``` - for access to a repository
   * ```repo``` - for access to a private repository
   * ```admin:org``` - if you plan on using this with an organization to which you have access
   * ```manage_runners:enterprise``` - if you plan to use garm at the enterprise level
   * ```admin:repo_hook``` - if you want to allow GARM to install webhooks on repositories (optional)
   * ```admin:org_hook``` - if you want to allow GARM to install webhooks on organizations (optional)

Fine grained PATs are also supported as long as you grant the required privileges:

* **Repository permissions**:
   * `Administration: Read & write` - needed to generate JIT config/registration token, remove runners, etc.
   * `Metadata: Read-only` - automatically enabled by above
* **Organization permissions**:
   * `Self-hosted runners: Read & write` - needed to manage runners in an organization

#### 3. Create the required `garm-operator` CRs:
You can use the `make template` target in the root directory of the `garm-provider-k8s` repository to generate a new
`garm-operator-crs.yaml` which contains all CRs required for `garm-operator`.

```bash
GARM_GITHUB_ORGANIZATION=my-github-org \
GARM_GITHUB_REPOSITORY=my-github-repo \
GARM_GITHUB_TOKEN=gha_testtoken \
GARM_GITHUB_WEBHOOK_SECRET=supersecret \
make template
```

#### 4. Build and deploy the `garm-server` with `garm-provider-k8s` into your local `kind` cluster:
```bash
make docker-build docker-build-summerwind-runner && kubectl apply -k hack/local-development/kubernetes/
```

#### 5. Deploy the `garm-operator` CRs to your local `kind` cluster:
```bash
kubectl apply -f hack/local-development/kubernetes/garm-operator-crs.yaml
```

### 🐛 Debugging

To improve the local development process, we add [delve](https://github.com/go-delve/delve) into `garm-operator` container image.
This allows us to debug the `garm-operator` running in the local `kind` cluster.

The following steps are required to start debugging the `garm-operator`:

#### 1. set the `mode` variable from `local` to `debug` in the `Tiltfile`

   This will start the `garm-operator` container with the `command` and `args` specified in the [`config/overlays/debug/manager_patch.yaml`](config/overlays/debug/manager_patch.yaml) file. (Ensure that the correct GARM credentials are set.)

   The `garm-operator-controller-manager` pod should log then print the following log message which indicates that you are able to attach a debugger to the `garm-operator`:

   ```
   2023-12-08T15:39:21Z warning layer=rpc Listening for remote connections (connections are not authenticated nor encrypted)
   API server listening at: [::]:2345
   ```

#### 1. IDE configuration
   1. VSCode
      1. Create a `launch.json` file in the `.vscode` directory with the following content:
         ```json
         {
             "version": "0.2.0",
             "configurations": [
                 {
                     "name": "garm-operator - attach",
                     "type": "go",
                     "request": "attach",
                     "mode": "remote",
                     "port": 2345
                 }
             ]
         }
         ```
   1. IntelliJ
      1. Create a `.run` folder in the root of your project if not present. Create a`garm-operator debug.run.xml` file with the following content:
      ```xml
      <component name="ProjectRunConfigurationManager">
         <configuration default="false" name="garm-operator debug" type="GoRemoteDebugConfigurationType" factoryName="Go Remote">
             <option name="disconnectOption" value="ASK" />
             <method v="2" />
         </configuration>
      </component>
      ```
      You can now choose your config in IntelliJs `Run Configurations` and hit `Debug`

      ![img_2.png](docs/assets/intellij-debugging.png)

1. Happy debugging 🐛

