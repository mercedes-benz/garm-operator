<!-- SPDX-License-Identifier: MIT -->

---
date: 2023-07-24
desc: Reduce login API calls
state: rejected
---
<!--
What is the status, such as proposed, accepted, rejected, deprecated, superseded, etc.?
-->

<!--
This is a basic ADR template from [Documenting architecture decisions - Michael Nygard](http://thinkrelevance.com/blog/2011/11/15/documenting-architecture-decisions).

It's possible to manage the ADR files with [adr-tools](https://github.com/npryce/adr-tools).
-->

# Reduce login API calls

## Context and Problem Statement

<!--
Describe the context and the problem that we are trying to solve.
-->

`garm-operator` is getting started with a pair of `username` and `password` to authenticate against a given `garm-server` instance.

The initial `garm-operator` implementation is creating a new `github.com/cloudbase/garm/cmd/garm-cli/client.Client` object for every `CustomResource` within a reconciliation loop.

This means, `garm-operator` requests the `/api/v1/auth/login` endpoint for every `CustomResource` within a reconciliation loop to get a new `JWT` token.
This is not optimal, because the `garm-server` is getting a lot of requests for the same `username` and `password` combination.

## Decision Drivers

<!--
List possible facts which may influence the decision.
-->

* Reduce the number of API calls to the `garm-server` instance
* `garm-operator` code should be easy to understand and maintain

## Considered Options

<!--
List possible options to address the problem or issue.
-->

* keep the current implementation
* create a global `client.Client` object and reuse it for every `CustomResource` within a reconciliation loop and refresh the `JWT` token if it is going to expire soon

## Pros and Cons of the Options

<!--
List the pros and cons of each option.
-->

### keep the current implementation

#### Pros

* The implementation is easy to understand. Debugging might be also easy.

#### Cons

* The `/api/v1/auth/login` endpoint is getting requested more often than necessary.
* The memory footprint of `garm-operator` is increasing, because there is one `client.Client` object for each `CustomResource`.

### global `client.Client`

By caching the `JWT` token, the `garm-server` is getting less requests for the same `username` and `password` combination.

`garm-operator` needs to handle the `JWT` token expiration and refresh it if it is going to expire soon. 
As there isn't an existing `client.Client.RefreshToken()` method (xref [`garm` JWT auth documentation](https://github.com/cloudbase/garm/blob/14586f01541165679cc35a9d2050aee0a595af81/doc/config_jwt_auth.md?plain=1)), we need to re-implement it for our own and propagate the renewed `JWT` token to all current existing `client.Client` objects.

#### Pros

* The `/api/v1/auth/login` endpoint is getting requested less often.
* The memory footprint of `garm-operator` is decreasing, because there is only one `client.Client` object for all `CustomResource` objects.

#### Cons

* The implementation is more complex. Debugging might be also more complex.

## Decision Outcome

<!--
What option was chosen? Why?
-->

As the `garm-operator` code should be easy to understand and maintain and
we do not expect a lot of `CustomResource` objects within a single `garm-operator` instance, we decided to keep the current implementation.
