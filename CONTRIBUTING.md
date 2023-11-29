<!-- SPDX-License-Identifier: MIT -->

# Contributing 

This document explains how to contribute to this project.
By contributing you will agree that your contribution will be put under the same license as this repository.

<!-- toc -->
- [CLA](#cla)
- [Communication](#communication)
- [Contributions](#contributions)
  - [Semantic Commit Messages](#semantic-commit-messages)
- [Quality](#quality)
  - [Validate and test your code](#validate-and-test-your-code)
<!-- /toc -->

## CLA

Before you can contribute, you will need to sign our cla [Contributor License Agreement](https://github.com/mercedes-benz/foss/blob/master/cla/2022-04-25_MB_FOSS_CLA_MBTI.pdf) and send the signed CLA to <cla-mbti@mercedes-benz.com> 

## Communication

For communication please respect our [FOSS Code of Conduct](https://github.com/mercedes-benz/foss/blob/master/CODE_OF_CONDUCT.md).

The following communication channels exist for this project:
- GitHub for reporting and claiming issues: https://github.com/mercedes-benz/garm-operator/issues

Transparent and open communication is important to us. 
Thus, all project-related communication should happen only through these channels and in English. 
Issue-related communication should happen within the concerned issue.

## Contributions

Contributions are highly welcome.

If you would like to contribute code you can do so through GitHub by forking the repository and sending a pull request.

When submitting code, please make every effort to follow existing conventions and style in order to keep the code as readable as possible.

If you are new to contributing in GitHub, [First Contributions](https://github.com/firstcontributions/first-contributions) might be a good starting point.

Before you can contribute, you will need to sign our Contributor License Agreement (CLA). When you create your first pull request, you will be requested by our CLA-assistant to sign this CLA.

### Semantic Commit Messages

We use [semantic commit messages](https://www.conventionalcommits.org/en/v1.0.0/) in this repository.

They follow this format: `<type>[optional scope]: <description>`

Examples for commit messages following this are:

`feat: allow provided config object to extend other configs`

Here's a list of types that we use in this repository:

| Type  | Explanation                                                   |
|-------|---------------------------------------------------------------|
| feat  | A new feature                                                 |
| fix   | A bug fix                                                     |
| docs  | Documentation only changes                                    |
| test  | Adding missing tests or correcting existing tests             |
| build | Changes that affect the build system or external dependencies |
| chore | Other changes that don't modify src or test files             |

These types are also used for generating the changelog.

## Quality
Please ensure that for all contributions, the corresponding documentation is in-sync and up-to-date. All documentation should be in English language. 

We assume that for every non-trivial contribution, the project has been built and tested prior to the contribution.

### Validate and test your code

Before you commit your code, please make sure that it is valid and tested. The existing tests can be run with `make test` and should give you a rough idea if your code changed any current behavior.

In [`.github/workflows/build.yml`](.github/workflows/build.yml) we also run some checks on your code, but you can also run them locally
before you push by running

```bash
#!/bin/sh

# run the tests
make test
# lint the code
make lint
# verify if any code-generator hasn't run
make verify
```
