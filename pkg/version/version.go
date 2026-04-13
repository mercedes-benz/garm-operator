// SPDX-License-Identifier: MIT

package version

import "golang.org/x/mod/semver"

// MinVersion is the minimum required version of garm that the operator supports
const MinVersion = "v0.1.5"

// EnsureMinimalVersion checks if the given version is greater than or equal to the minimum required version
func EnsureMinimalVersion(version string) bool {
	return semver.Compare(version, MinVersion) >= 0
}
