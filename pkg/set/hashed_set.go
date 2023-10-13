package set

type HashedSet[K comparable, T any] struct {
	Hasher func(T) K
	m      map[K]T
}

func (s *HashedSet[K, T]) initialize() {
	if s.m == nil {
		s.m = make(map[K]T)
	}
}

func (s *HashedSet[K, T]) Add(vs ...T) {
	s.initialize()
	for _, v := range vs {
		hash := s.Hasher(v)
		s.m[hash] = v
	}
}

func (s *HashedSet[K, T]) Remove(v T) bool {
	s.initialize()
	hash := s.Hasher(v)
	_, ok := s.m[hash]
	delete(s.m, hash)
	return ok
}

func (s HashedSet[K, T]) Contains(v T) bool {
	s.initialize()
	hash := s.Hasher(v)
	_, ok := s.m[hash]
	return ok
}

func (s HashedSet[K, T]) ContainsAll(vs ...T) bool {
	s.initialize()
	for _, v := range vs {
		if !s.Contains(v) {
			return false
		}
	}
	return true
}

func (s HashedSet[K, T]) ContainsAny(vs ...T) bool {
	s.initialize()
	for _, v := range vs {
		if s.Contains(v) {
			return true
		}
	}
	return false
}

func (s HashedSet[K, T]) Len() int {
	s.initialize()
	return len(s.m)
}

func (s HashedSet[K, T]) ToSlice() []T {
	s.initialize()
	slice := make([]T, 0, len(s.m))
	for k := range s.m {
		slice = append(slice, s.m[k])
	}
	return slice
}

func (s HashedSet[K, T]) Union(other HashedSet[K, T]) HashedSet[K, T] {
	s.initialize()
	union := make(map[K]T)
	for k := range s.m {
		v := s.m[k]
		union[k] = v
	}
	for k := range other.m {
		v := other.m[k]
		union[k] = v
	}
	return HashedSet[K, T]{
		Hasher: s.Hasher,
		m:      union,
	}
}

func (s HashedSet[K, T]) Intersection(other HashedSet[K, T]) HashedSet[K, T] {
	s.initialize()
	intersection := HashedSet[K, T]{
		Hasher: s.Hasher,
		m:      make(map[K]T),
	}
	for k := range s.m {
		if _, ok := other.m[k]; ok {
			intersection.Add(s.m[k])
		}
	}
	return intersection
}

// marshall to yaml
func (s HashedSet[K, T]) MarshalYAML() (interface{}, error) {
	return s.ToSlice(), nil
}
