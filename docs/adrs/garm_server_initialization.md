<!-- SPDX-License-Identifier: MIT -->

---
date: 2023-12-04
desc: Garm Server Initialization
state: accepted
---
<!--
What is the status, such as proposed, accepted, rejected, deprecated, superseded, etc.?
-->

<!--
This is a basic ADR template from [Documenting architecture decisions - Michael Nygard](http://thinkrelevance.com/blog/2011/11/15/documenting-architecture-decisions).

It's possible to manage the ADR files with [adr-tools](https://github.com/npryce/adr-tools).
-->
# Garm Server Initialization

## Context and Problem Statement

<!--
Describe the context and the problem that we are trying to solve.
-->

A newly deployed garm-server instance must first be initialized before a login can take place.
Therefore, the `garm-operator` cannot perform a login if the garm-server instance has not yet been initialized.

## Decision Drivers

<!--
List possible facts which may influence the decision.
-->

* The usability of the garm-server initialization functionality
* The maintenance effort

## Considered Options

<!--
List possible options to address the problem or issue.
-->

* Attach a init container to the garm-server pod
* Attach a sidecar container to the garm-server pod
* Create entrypoint script for the garm-server image
* Implement the init functionality in the `garm-operator`

## Pros and Cons of the Options

<!--
List the pros and cons of each option.
-->

### Attach an init container to the garm-server pod

#### Pros

* The initialization is executed before the garm-server instance starts
* The init container only needs more cluster resources for a short time until the initialization has been completed

#### Cons

* The init container would have to contain a script, `garm` and `garm-cli` binary, which would lead to a higher maintenance effort
* We would have to build and maintain a garm-server-init container image
* The garm-server deployment is currently not provided by us, so other users would have to implement it by themselves

### Attach an sidecar container to the garm-server pod

#### Pros

* The sidecar container can constantly monitor the garm-server instance and initialize it if necessary
* Requires only a script and the `garm-cli` binary

#### Cons

* The sidecar container would run constantly and consume additional cluster resources, although it is only needed for initialization
* We would have to build and maintain a garm-server-sidecar container image
* The garm-server deployment is currently not provided by us, so other users would have to implement it by themselves

### Create entrypoint script for the garm-server image

#### Pros

* We only have to maintain one image
* The garm-server instance can be initialized at any time when the entrypoint script is running in the background

#### Cons

* The garm-server image is currently not provided by us, so other users would have to implement it themselves

### Implement the init functionality in the `garm-operator`

#### Pros

* The initialization functionality can be used by anyone who uses the `garm-operator`
* It allows us to implement better error handling
* With [koanf](https://github.com/knadh/koanf) we can more easily control whether initialization should be performed or not

#### Cons

* Adding an additional functionality which is not part of the core functionality of the `garm-operator`

## Decision Outcome

<!--
What option was chosen? Why?
-->

The initialization of the garm-server should be usable for everyone, for this reason we decided to implement this functionality in the `garm-operator`.
We would also like to reduce further maintaining overhead by adding more images which would be necessary, for example, with the use of an init or sidecar container.
