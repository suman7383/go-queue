package ringbuffer

import (
	"log"
	"sync"
)

type RingBuffer[T any] struct {
	queue []T
	head  int64
	tail  int64
	size  int64
	cap   int64
	mu    sync.Mutex
}

func NewRingBuffer[T any](capacity int64) *RingBuffer[T] {
	return &RingBuffer[T]{
		queue: make([]T, capacity),
		head:  0,
		tail:  0,
		size:  0,
		cap:   capacity,
	}
}

// insert an element at the tail.
// Returns (size, error)
func (r *RingBuffer[T]) Enqueue(msg T) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size == r.cap {
		// needs resizing
		log.Println("size full. Resizing")
		r.resize()
		log.Printf("resizing done. H: %v, T: %v, size: %v ", r.head, r.tail, r.size)
	}

	r.queue[r.tail] = msg
	r.tail = (r.tail + 1) % r.cap
	r.size++

	return r.size
}

func (r *RingBuffer[T]) Dequeue() (T, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size == 0 || r.head == r.tail {
		var t T
		return t, false
	}

	// take out from head
	item := r.queue[(r.head % r.cap)]

	var t T
	r.queue[(r.head % r.cap)] = t
	r.head = (r.head + 1) % r.cap
	r.size--

	return item, true
}

func (r *RingBuffer[T]) Size() int64 {
	return r.size
}

func (r *RingBuffer[T]) Cap() int64 {
	return r.cap
}

func (r *RingBuffer[T]) resize() {
	var new_queue = make([]T, 2*r.cap)

	for i := range r.size {
		new_queue[i] = r.queue[(r.head+i)%r.cap]
	}

	r.queue = new_queue
	r.head = 0
	r.tail = r.size
	r.cap = 2 * r.cap
}
