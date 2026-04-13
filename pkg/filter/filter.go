// SPDX-License-Identifier: MIT

package filter

// Predicate is a function that takes an item of type T and returns a boolean indicating whether the item matches certain criteria
type Predicate[T any] func(T) bool

// Match takes a slice of items and a variadic number of predicates, and returns a slice of items that match all the predicates
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
