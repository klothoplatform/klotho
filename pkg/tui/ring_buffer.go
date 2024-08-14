package tui

type RingBuffer[T any] struct {
	buf  []T
	head int
	tail int
}

func NewRingBuffer[T any](size int) *RingBuffer[T] {
	return &RingBuffer[T]{buf: make([]T, size)}
}

func (r *RingBuffer[T]) Push(v T) {
	r.buf[r.head] = v
	r.head = (r.head + 1) % len(r.buf)
	if r.head == r.tail {
		r.tail = (r.tail + 1) % len(r.buf)
	}
}

func (r *RingBuffer[T]) Pop() T {
	if r.head == r.tail {
		return r.buf[r.head]
	}
	v := r.buf[r.tail]
	r.tail = (r.tail + 1) % len(r.buf)
	return v
}

func (r *RingBuffer[T]) Len() int {
	if r.head >= r.tail {
		return r.head - r.tail
	}
	return len(r.buf) - r.tail + r.head
}

func (r *RingBuffer[T]) Cap() int {
	return len(r.buf)
}

func (r *RingBuffer[T]) Get(i int) (T, bool) {
	if i >= r.Len() {
		var zero T
		return zero, false
	}
	if i < 0 {
		i += r.Len()
	}
	return r.buf[(r.tail+i)%len(r.buf)], true
}

func (r *RingBuffer[T]) Set(i int, v T) bool {
	if i >= r.Len() {
		return false
	}
	if i < 0 {
		i += r.Len()
	}
	r.buf[(r.tail+i)%len(r.buf)] = v
	return true
}

func (r *RingBuffer[T]) Clear() {
	r.head = 0
	r.tail = 0
}

func (r *RingBuffer[T]) ForEach(f func(int, T)) {
	for i := 0; i < r.Len(); i++ {
		v, _ := r.Get(i)
		f(i, v)
	}
}
