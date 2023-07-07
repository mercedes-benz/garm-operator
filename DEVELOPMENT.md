<!-- SPDX-License-Identifier: MIT -->

# Development

<!-- toc -->
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
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
