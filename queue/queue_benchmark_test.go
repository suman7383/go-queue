package queue

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkQueueQps(b *testing.B) {
	topic := NewTopic("test", TopicConfig{
		AckTimeout: 30 * time.Second,
		MaxRetries: 3,
	})

	b.ResetTimer()

	go func() {
		for i := 0; i < b.N; i++ {
			topic.Enqueue(fmt.Sprintf("message %d", i))
		}
	}()

	for i := 0; i < b.N; i++ {
		msg, err := topic.Dequeue()
		if err == nil {

			topic.Acknowledge(msg.ID)
		}
	}
}

func BenchmarkQueueProdCons(b *testing.B) {
	q := NewTopic("bench", TopicConfig{})

	done := make(chan struct{})

	// Consumer goroutine
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				msg, err := q.Dequeue()
				if err == nil {
					q.Acknowledge(msg.ID)
				}
			}
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Enqueue("data")
	}

	close(done)
}
