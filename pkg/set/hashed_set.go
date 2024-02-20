package set

import (
	"sort"

	"gopkg.in/yaml.v3"
)

type HashedSet[K comparable, T any] struct {
	Hasher func(T) K
	M      map[K]T
	// Less is used to sort the keys of the set when converting to a slice.
	// If Less is nil, the keys will be sorted in an arbitrary order according to [map] iteration.
	Less func(K, K) bool
}

func HashedSetOf[K comparable, T any](hasher func(T) K, vs ...T) HashedSet[K, T] {
	s := HashedSet[K, T]{Hasher: hasher}
	s.Add(vs...)
	return s
}

func (s *HashedSet[K, T]) Add(vs ...T) {
	if s.M == nil {
		s.M = make(map[K]T)
	}
	for _, v := range vs {
		hash := s.Hasher(v)
		s.M[hash] = v
	}
}

func (s *HashedSet[K, T]) Remove(v T) bool {
	if s.M == nil {
		return false
	}
	hash := s.Hasher(v)
	_, ok := s.M[hash]
	delete(s.M, hash)
	return ok
}

func (s HashedSet[K, T]) Contains(v T) bool {
	if s.M == nil {
		return false
	}
	hash := s.Hasher(v)
	_, ok := s.M[hash]
	return ok
}

func (s HashedSet[K, T]) ContainsAll(vs ...T) bool {
	for _, v := range vs {
		if !s.Contains(v) {
			return false
		}
	}
	return true
}

func (s HashedSet[K, T]) ContainsAny(vs ...T) bool {
	for _, v := range vs {
		if s.Contains(v) {
			return true
		}
	}
	return false
}

func (s HashedSet[K, T]) Len() int {
	return len(s.M)
}

func (s HashedSet[K, T]) ToSlice() []T {
	if s.M == nil {
		return nil
	}
	slice := make([]T, 0, len(s.M))
	if s.Less != nil {
		keys := make([]K, 0, len(s.M))
		for k := range s.M {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return s.Less(keys[i], keys[j])
		})
		for _, k := range keys {
			slice = append(slice, s.M[k])
		}
	} else {
		for k := range s.M {
			slice = append(slice, s.M[k])
		}
	}
	return slice
}

func (s HashedSet[K, T]) ToMap() map[K]T {
	m := make(map[K]T, len(s.M))
	for k, v := range s.M {
		m[k] = v
	}
	return m
}

func (s HashedSet[K, T]) Union(other HashedSet[K, T]) HashedSet[K, T] {
	union := make(map[K]T)
	for k := range s.M {
		v := s.M[k]
		union[k] = v
	}
	for k := range other.M {
		v := other.M[k]
		union[k] = v
	}
	return HashedSet[K, T]{
		Hasher: s.Hasher,
		M:      union,
		Less:   s.Less,
	}
}

func (s HashedSet[K, T]) Intersection(other HashedSet[K, T]) HashedSet[K, T] {
	intersection := HashedSet[K, T]{
		Hasher: s.Hasher,
		M:      make(map[K]T),
		Less:   s.Less,
	}
	for k := range s.M {
		if _, ok := other.M[k]; ok {
			intersection.M[k] = s.M[k]
		}
	}
	return intersection
}

func (s HashedSet[K, T]) MarshalYAML() (interface{}, error) {
	return s.ToSlice(), nil
}

func (s *HashedSet[K, T]) UnmarshalYAML(node *yaml.Node) error {
	var slice []T
	err := node.Decode(&slice)
	if err != nil {
		return err
	}
	s.Add(slice...)
	return nil
}
