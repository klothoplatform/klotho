package async

import "sync"

type (
	ConcurrentMap[K comparable, V any] struct {
		mu sync.RWMutex
		m  map[K]V
	}

	MapEntry[K comparable, V any] struct {
		Key   K
		Value V
	}
)

// initForWrite caller must hold a write lock. Read operations do not need to call this - they must all be compatible
// with a nil map.
func (cf *ConcurrentMap[K, V]) initForWrite() {
	if cf.m == nil {
		cf.m = make(map[K]V)
	}
}

func (cf *ConcurrentMap[K, V]) Len() int {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	return len(cf.m)
}

func (cf *ConcurrentMap[K, V]) Set(k K, v V) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	cf.initForWrite()
	cf.m[k] = v
}

func (cf *ConcurrentMap[K, V]) AddAll(entries map[K]V) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	cf.initForWrite()
	for key, value := range entries {
		cf.m[key] = value
	}
}

// Compute sets the value of key 'k' to the result of the supplied computeFunc.
// If the value of 'ok' is 'false', the entry for key 'k' will be removed from the ConcurrentMap.
func (cf *ConcurrentMap[K, V]) Compute(k K, computeFunc func(k K, v V) (val V, ok bool)) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	cf.initForWrite()
	if val, ok := computeFunc(k, cf.m[k]); ok {
		cf.m[k] = val
	} else {
		cf.Delete(k)
	}
}

func (cf *ConcurrentMap[K, V]) Delete(k K) (v V, existed bool) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	if cf.m != nil {
		v, existed = cf.m[k]
		delete(cf.m, k)
	}
	return
}

func (cf *ConcurrentMap[K, V]) Get(k K) (v V, ok bool) {
	cf.mu.RLock()
	defer cf.mu.RUnlock()
	if cf.m != nil {
		v, ok = cf.m[k]
	}
	return
}

func (cf *ConcurrentMap[K, V]) Keys() []K {
	cf.mu.RLock()
	defer cf.mu.RUnlock()
	if cf.m == nil {
		return nil
	}
	ks := make([]K, 0, len(cf.m))
	for k := range cf.m {
		ks = append(ks, k)
	}
	return ks
}

func (cf *ConcurrentMap[K, V]) Values() []V {
	cf.mu.RLock()
	defer cf.mu.RUnlock()
	if cf.m == nil {
		return nil
	}
	vs := make([]V, 0, len(cf.m))
	for _, v := range cf.m {
		vs = append(vs, v)
	}
	return vs
}

func (cf *ConcurrentMap[K, V]) Entries() []MapEntry[K, V] {
	cf.mu.RLock()
	defer cf.mu.RUnlock()
	if cf.m == nil {
		return nil
	}
	kvs := make([]MapEntry[K, V], 0, len(cf.m))
	for k, v := range cf.m {
		kvs = append(kvs, MapEntry[K, V]{Key: k, Value: v})
	}
	return kvs
}

// Each executes `f` for each key-value pair in the map, while holding the lock.
// ! Avoid doing expensive operations in `f`, instead create a copy (eg via `Entries()`).
func (cf *ConcurrentMap[K, V]) Each(f func(k K, v V) (stop bool)) {
	cf.mu.RLock()
	defer cf.mu.RUnlock()
	for k, v := range cf.m {
		if stop := f(k, v); stop {
			return
		}
	}
}
