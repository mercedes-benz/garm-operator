<!-- SPDX-License-Identifier: MIT -->

# Contributing 

This document explains how to contribute to this project.
By contributing you will agree that your contribution will be put under the same license as this repository.

<!-- toc -->
- [Communication](#communication)
- [Contributions](#contributions)
  - [Developer Certificate of Origin](#developer-certificate-of-origin)
  - [Semantic Commit Messages](#semantic-commit-messages)
- [Quality](#quality)
  - [Validate and test your code](#validate-and-test-your-code)
<!-- /toc -->

## Communication

For communication please respect our [FOSS Code of Conduct](https://github.com/mercedes-benz/.github/blob/main/CODE_OF_CONDUCT.md).

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

### Developer Certificate of Origin

Contributors must agree to the [Developer Certificate of Origin Version 1.1 (DCO)](https://developercertificate.org/) and comply with the sign-off procedure for contributing to this project.

All contributions submitted to this repository have to carry a signed-off-by statement in the commit message using the following format:

```text
Signed-off-by: Firstname Lastname <email.address>
```

where the specified email address must be associated with your [github account](https://docs.github.com/en/account-and-profile/concepts/email-addresses). 
If a commit does not carry such a signed-off-by statement, the pull request cannot be merged because by adding signed-off-by statements to the commit messages, you convey that you have the necessary rights to contribute the respective changes.

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
