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
	ID        int64     `json:"id,omitempty"`
	Payload   string    `json:"payload,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"` // When it was delivered
	Acked     bool      `json:"acked,omitempty"`     // Whether it's been acknowledged
	Retries   int       `json:"retries,omitempty"`
}

type LogEntry struct {
	Type    string // "enqueue" | "deliver" | "ack"
	Message Message
}
