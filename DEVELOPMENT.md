<!-- SPDX-License-Identifier: MIT -->

# Development

<!-- toc -->
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
  - [🐛 Debugging](#-debugging)
  - [⚙️ Bootstrap garm-server with garm-provider-k8s for local development](#-bootstrap-garm-server-with-garm-provider-k8s-for-local-development)
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


### 🐛 Debugging

To improve the local development process, we add [delve](https://github.com/go-delve/delve) into `garm-operator` container image.
This allows us to debug the `garm-operator` running in the local `kind` cluster.

The following steps are required to start debugging the `garm-operator`:

1. set the `mode` variable from `local` to `debug` in the `Tiltfile`

   This will start the `garm-operator` container with the `command` and `args` specified in the [`config/overlays/debug/manager_patch.yaml`](config/overlays/debug/manager_patch.yaml) file. (Ensure that the correct GARM credentials are set.)

   The `garm-operator-controller-manager` pod should log then print the following log message which indicates that you are able to attach a debugger to the `garm-operator`:

   ```
   2023-12-08T15:39:21Z warning layer=rpc Listening for remote connections (connections are not authenticated nor encrypted)
   API server listening at: [::]:2345
   ```

1. IDE configuration
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


### ⚙️ Bootstrap garm-server with garm-provider-k8s for local development

If you need a `garm-server` with a configured [garm-provider-k8s](https://github.com/mercedes-benz/garm-provider-k8s) in your local cluster to spin up some `k8s based runners` for testing, you can do the following:

Clone the `garm-provider-k8s` repo:
   ```bash
   $ git clone https://github.com/mercedes-benz/garm-provider-k8s && cd ./garm-provider-k8s
   ```
And follow this [guide](https://github.com/mercedes-benz/garm-provider-k8s/blob/main/DEVELOPMENT.md). But `instead` of the `make tilt-up` in the `garm-provider-k8s` repo, execute the folling command. Make sure you are in your `kind-garm-operator` kubernetes context:
   ```bash
   $ make build copy docker-build docker-build-summerwind-runner && kubectl apply -k hack/local-development/kubernetes/
   ```
Essentially this does the same as the `make tilt-up` target in `garm-provider-k8s`, but in your local garm-operator cluster. Otherwise, a separate cluster will be spawned with the latest garm-operator release.
