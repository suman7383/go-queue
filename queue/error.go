package queue

import "errors"

var (
	ErrEmptyQueue = errors.New("cannot perform dequeue on empty queue")
)
