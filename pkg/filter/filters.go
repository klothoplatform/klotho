package filter

import (
	"github.com/klothoplatform/klotho/pkg/filter/predicate"
)

// Filter is a generic interface for a filter that is applied to a slice of type 'T' and returns a subset of the original input as []T
type Filter[T any] interface {
	Apply(...T) []T
	Find(...T) (T, bool)
}

// SimpleFilter is a filter that filters based on a supplied predicate (Predicate)
type SimpleFilter[T any] struct {
	Predicate predicate.Predicate[T]
}

// Apply returns the subset of inputs matching the SimpleFilter's Predicate
func (f SimpleFilter[T]) Apply(inputs ...T) []T {
	var result []T
	for _, input := range inputs {
		if f.Predicate(input) {
			result = append(result, input)
		}
	}
	return result
}

func (f SimpleFilter[T]) Find(inputs ...T) (T, bool) {
	for _, input := range inputs {
		if f.Predicate(input) {
			return input, true
		}
	}
	var zero T
	return zero, false
}

// NewSimpleFilter returns a SimpleFilter matching each supplied predicate.Predicate sequentially on a per-input basis
func NewSimpleFilter[T any](predicates ...predicate.Predicate[T]) Filter[T] {
	return SimpleFilter[T]{Predicate: predicate.AllOf(predicates...)}
}
