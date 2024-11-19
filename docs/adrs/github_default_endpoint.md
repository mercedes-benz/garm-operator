<!-- SPDX-License-Identifier: MIT -->

---
date: 2022-11-20
desc: Reflection of the default GitHub endpoint
state: accepted
---
<!--
What is the status, such as proposed, accepted, rejected, deprecated, superseded, etc.?
-->

<!--
This is a basic ADR template from [Documenting architecture decisions - Michael Nygard](http://thinkrelevance.com/blog/2011/11/15/documenting-architecture-decisions).

It's possible to manage the ADR files with [adr-tools](https://github.com/npryce/adr-tools).
-->
# Reflection of the default GitHub endpoint

## Context and Problem Statement

With `garm` in version `0.1.5`, the GitHub API endpoints aren't defined in the configuration file anymore. The GitHub API endpoints are now defined via the `garm` API.

`garm` also ships with a default immutable GitHub API endpoint and points to the public GitHub API.

With the new API version `v1beta1` for all `garm-operator` related resources, we have to make use of the new `GitHubEndpoint` object to define the connection to the GitHub API. 
The `GitHubEndpoint` is referenced in the `GitHubCredential` object.

## Decision Drivers

* The default GitHub API endpoint is immutable.
* The GitHub API endpoint must be configured via a new `GitHubEndpoint` object which then got referenced in the `GitHubCredential` object.

## Considered Options

* Reflecting the default GitHub API endpoint back into the kubernetes cluster by creating a `GitHubEndpoint` object with the default values.
  But as the default GitHub API endpoint is immutable, we have to catch this case in the operator and forbid any update/delete operations on this reflected `GitHubEndpoint`.

## Pros and Cons of the Options

#### Pros:
* The code must not handle the immutability of the default GitHub API endpoint.

#### Cons:
* Everyone who wants to use the `garm-operator` with the default GitHub API endpoint has to create a `GitHubEndpoint` object.

## Decision Outcome

The validation rules in the `CRD` blocks the creation of a `GitHubEndpoint` object with the name `github.com`.