// SPDX-License-Identifier: MIT

package filter

type Predicate[T any] func(T) bool

func Match[T any](items []T, predicates ...Predicate[T]) []T {
	var filteredPools []T

	for _, pool := range items {
		match := true
		for _, predicate := range predicates {
			if !predicate(pool) {
				match = false
				break
			}
		}
		if match {
			filteredPools = append(filteredPools, pool)
		}
	}

	return filteredPools
}
