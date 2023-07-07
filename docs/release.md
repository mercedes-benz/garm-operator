<!-- SPDX-License-Identifier: MIT -->

# Release

<!-- toc -->
- [Release creation process](#release-creation-process)
  - [Workflow](#workflow)
<!-- /toc -->

## Release creation process

We use [goreleaser](https://goreleaser.com) for cutting a new release.
The current implementation triggers the release creation process on new git tags starting with `v` (e.g. `v0.0.1`).

### Workflow

1. Fetch the current `main` branch from the remote repository
1. Create a new git tag starting with `v` (e.g. `v0.0.1`)
   ```bash
   git tag -a v0.0.1 -m "Release v0.0.1"
   ```
1. Push the new git tag to the remote repository
   ```bash
   git push origin v0.0.1
   ```
1. Wait for the release creation process to finish
   by checking the [release action workflow](https://github.com/mercedes-benz/garm-operator/actions/workflows/release.yml)