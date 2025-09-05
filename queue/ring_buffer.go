package queue

import (
	"log"
	"sync"
)

type RingBuffer[T any] struct {
	queue []T
	head  int
	tail  int
	size  int
	cap   int
	mu    sync.RWMutex
}

func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
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
func (r *RingBuffer[T]) Enqueue(msg T) int {
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

func (r *RingBuffer[T]) Dequeue() (T, error) {
	var msg T
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size == 0 {
		return msg, ErrEmptyQueue
	}

	// take out from head
	msg = r.queue[(r.head%r.cap)]
	r.head = (r.head + 1)%r.cap
	r.size--

	return msg, nil
}

func (r *RingBuffer[T]) Size() int {

	return r.size
}

func (r *RingBuffer[T]) Cap() int {

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
	r.cap = cap(r.queue)
}
