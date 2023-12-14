<!-- SPDX-License-Identifier: MIT -->

# Development

<!-- toc -->
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
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
1. Happy debugging 🐛
