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

        msg, err := topic.Dequeue()
        if err != nil {
            b.Fatalf("Dequeue failed at iteration %d: %v", i, err)
        }

        topic.Acknowledge(msg.ID)
    }
}

