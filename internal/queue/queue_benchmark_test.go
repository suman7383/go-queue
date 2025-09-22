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

	for i := 0; i < b.N; i++ {
		topic.Enqueue(fmt.Sprintf("message %d", i))

		msg, ok := topic.Dequeue()
		if !ok {
			b.Fatalf("queue empty %d", i)
		}

		topic.Acknowledge(msg.ID)
	}
}
