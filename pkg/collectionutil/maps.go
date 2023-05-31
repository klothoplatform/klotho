package collectionutil

func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetOneEntry gets an entry from the given map. If the map contains multiple entries, it's undefined which this
// returns. If the map is empty, this will return the zero value for both the key and map.
func GetOneEntry[K comparable, V any](m map[K]V) (K, V) {
	for k, v := range m {
		return k, v
	}
	var k K
	var v V
	return k, v
}

type ExtendingMap[K comparable, V any] map[K]V

func Extend[K comparable, V any](m map[K]V) ExtendingMap[K, V] {
	return ExtendingMap[K, V](m)
}

func (source ExtendingMap[K, V]) Into(target map[K]V) {
	for k, v := range source {
		target[k] = v
	}
}

func CopyMap[K comparable, V any](m map[K]V) map[K]V {
	cp := make(map[K]V, len(m))
	Extend(m).Into(cp)
	return cp
}
