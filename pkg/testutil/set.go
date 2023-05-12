package testutil

func NewSet[T comparable](ss ...T) map[T]struct{} {
	set := make(map[T]struct{}, len(ss))
	for _, s := range ss {
		set[s] = struct{}{}
	}
	return set
}
