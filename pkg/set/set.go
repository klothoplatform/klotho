package set

type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(vs ...T) {
	for _, v := range vs {
		s[v] = struct{}{}
	}
}

func (s Set[T]) Remove(v T) bool {
	_, ok := s[v]
	delete(s, v)
	return ok
}

func (s Set[T]) Contains(v T) bool {
	_, ok := s[v]
	return ok
}

func (s Set[T]) ContainsAll(vs ...T) bool {
	for _, v := range vs {
		if !s.Contains(v) {
			return false
		}
	}
	return true
}

func (s Set[T]) ContainsAny(vs ...T) bool {
	for _, v := range vs {
		if s.Contains(v) {
			return true
		}
	}
	return false
}

func (s Set[T]) Len() int {
	return len(s)
}

func (s Set[T]) ToSlice() []T {
	slice := make([]T, 0, len(s))
	for k := range s {
		slice = append(slice, k)
	}
	return slice
}

func (s Set[T]) Union(other Set[T]) Set[T] {
	union := make(Set[T])
	for k := range s {
		union[k] = struct{}{}
	}
	for k := range other {
		union[k] = struct{}{}
	}
	return union
}

func (s Set[T]) Intersection(other Set[T]) Set[T] {
	intersection := make(Set[T])
	for k := range s {
		if _, ok := other[k]; ok {
			intersection.Add(k)
		}
	}
	return intersection
}
