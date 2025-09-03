package queue

import (
	"testing"
)

// func BenchmarkQueueQps(b *testing.B) {
// 	topic := NewTopic("test", TopicConfig{
// 		AckTimeout: 30 * time.Second,
// 		MaxRetries: 3,
// 	})

// 	maxOps := 1_000_000 // cap at 1M ops
// 	if b.N > maxOps {
// 		b.N = maxOps
// 	}

// 	b.ResetTimer()

// 	go func() {
// 		for i := 0; i < b.N; i++ {
// 			topic.Enqueue(fmt.Sprintf("message %d", i))
// 		}
// 	}()

// 	for i := 0; i < b.N; i++ {
// 		msg, _ := topic.Dequeue()
// 		if msg != nil {

// 			topic.Acknowledge(msg.ID)
// 		}
// 	}
// }

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
				msg, ok := q.Dequeue()
				if ok {
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
