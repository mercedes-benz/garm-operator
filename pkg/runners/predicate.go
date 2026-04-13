// SPDX-License-Identifier: MIT

package runners

import (
	"strings"

	"github.com/cloudbase/garm/params"

	"github.com/mercedes-benz/garm-operator/pkg/filter"
)

// MatchesName returns a predicate that matches instances with the given name
func MatchesName(name string) filter.Predicate[params.Instance] {
	return func(i params.Instance) bool {
		return strings.EqualFold(i.Name, name)
	}
}
