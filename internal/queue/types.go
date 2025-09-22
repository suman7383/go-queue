package queue

import "time"

type Queue[T any] interface {
	Enqueue(T) int64
	Dequeue() (T, bool)
	Size() int64
	Cap() int64
}

type TopicConfig struct {
	AckTimeout time.Duration
	MaxRetries int
}

// Message is a simple struct holding the message and data
type Message struct {
	ID        int64
	Payload   string
	Timestamp time.Time // When it was delivered
	Acked     bool      // Whether it's been acknowledged
	Retries   int
}

type LogEntry struct {
	Type    string // "enqueue" | "deliver" | "ack"
	Message Message
}
