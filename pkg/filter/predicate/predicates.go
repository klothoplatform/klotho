package predicate

import (
	"regexp"
)

type Predicate[T any] func(p T) bool

// Not negates the supplied Predicate
func Not[T any](predicate Predicate[T]) Predicate[T] {
	return func(p T) bool {
		return !predicate(p)
	}
}

// AnyOf returns true if any of the supplied predicates returns true or else returns false
func AnyOf[T any](predicates ...Predicate[T]) Predicate[T] {
	return func(p T) bool {
		for _, predicate := range predicates {
			if predicate(p) {
				return true
			}
		}
		return false
	}
}

// AllOf returns true if all the supplied predicates return true or else returns false
func AllOf[T any](predicates ...Predicate[T]) Predicate[T] {
	return func(p T) bool {
		for _, predicate := range predicates {
			if !predicate(p) {
				return false
			}
		}
		return true
	}
}

func StringMatchesPattern(pattern string) Predicate[string] {
	return func(target string) bool {
		return regexp.MustCompile(pattern).MatchString(target)
	}
}
