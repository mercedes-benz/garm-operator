// SPDX-License-Identifier: MIT

package version

import "golang.org/x/mod/semver"

const MinVersion = "v0.1.5"

func EnsureMinimalVersion(version string) bool {
	return semver.Compare(version, MinVersion) >= 0
}
