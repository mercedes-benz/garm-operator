// SPDX-License-Identifier: MIT

package instance

import (
	"strings"

	"github.com/cloudbase/garm/params"

	"github.com/mercedes-benz/garm-operator/pkg/filter"
)

func MatchesName(name string) filter.Predicate[params.Instance] {
	return func(i params.Instance) bool {
		return strings.EqualFold(i.Name, name)
	}
}
