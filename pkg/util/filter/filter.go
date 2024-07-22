// SPDX-License-Identifier: MIT

package filter

type Predicate[T any] func(T) bool

func Match[T any](items []T, predicates ...Predicate[T]) []T {
	var resultList []T

	for _, item := range items {
		match := true
		for _, predicate := range predicates {
			if !predicate(item) {
				match = false
				break
			}
		}
		if match {
			resultList = append(resultList, item)
		}
	}

	return resultList
}
