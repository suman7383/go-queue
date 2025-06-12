package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type TopicConfig struct {
	AckTimeout time.Duration
	MaxRetries int
}

// Message is a simple struct holding the message and data
type Message struct {
	ID        int
	Payload   string
	Timestamp time.Time // When it was delivered
	Acked     bool      // Whether it's been acknowledged
	Retries   int
}

// Topic represents a named message queue
type Topic struct {
	Name     string
	messages []Message
	inFlight map[int]*Message // delivered but not yet acked
	nextID   int
	mu       sync.Mutex
	config   TopicConfig
	wal      *WAL
}

// Create new topic queue
func NewTopic(name string, config TopicConfig) *Topic {
	wal, err := NewWAL(name)
	if err != nil {
		panic(err)
	}

	t := &Topic{
		Name:     name,
		messages: []Message{},
		inFlight: make(map[int]*Message),
		config:   config,
		wal:      wal,
	}

	// Replay WAL at startup
	t.replayWAL()

	// Retry goroutine
	go func() {
		for {
			time.Sleep(2 * time.Second) // Check periodically

			t.mu.Lock()
			now := time.Now()
			for id, msg := range t.inFlight {
				if !msg.Acked && now.Sub(msg.Timestamp) > t.config.AckTimeout {
					if msg.Retries < t.config.MaxRetries {
						// max retry not reached
						msg.Retries++
						fmt.Printf("[Retry] Topic: %s | Msg ID %d | Retry #%d\n", t.Name, msg.ID, msg.Retries)
						t.messages = append(t.messages, *msg) // Requeue

					} else {
						// max retry reached -> discard the message
						fmt.Printf("[DROP]: Msg ID %d exceeded max retries (%d). Discarded.\n", msg.ID, t.config.MaxRetries)
						// TODO: move to DLQ here later
					}
					delete(t.inFlight, id)
				}
			}
			t.mu.Unlock()
		}
	}()

	return t
}

// Enqueue adds a message to the topic
func (t *Topic) Enqueue(payload string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	msg := Message{
		ID:      t.nextID,
		Payload: payload,
	}

	t.nextID++
	t.messages = append(t.messages, msg)

	// Append to WAL
	if err := t.wal.AppendEvent("enqueue", msg); err != nil {
		fmt.Println("[WAL] Write failed:", err)
	}

	return msg.ID
}

// Dequeue returns the next message if exists(pull)
func (t *Topic) Dequeue() (*Message, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.messages) == 0 {
		return nil, false
	}

	msg := t.messages[0]
	t.messages = t.messages[1:]

	msg.Timestamp = time.Now()
	msg.Acked = false
	t.inFlight[msg.ID] = &msg

	if err := t.wal.AppendEvent("deliver", msg); err != nil {
		fmt.Println("[WAL] Deliver log failed:", err)
	}

	return &msg, true
}

func (t *Topic) Acknowledge(id int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	msg, ok := t.inFlight[id]
	if !ok {
		return false
	}

	msg.Acked = true
	delete(t.inFlight, id)

	// Append to WAL
	if err := t.wal.AppendEvent("ack", *msg); err != nil {
		fmt.Println("[WAL] Write failed:", err)
	}

	return true
}

func (t *Topic) replayWAL() {
	path := fmt.Sprintf("data/%s.wal", t.Name)
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("No WAL found for topic:", t.Name)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	type msgState struct {
		message   Message
		enqueued  bool
		delivered bool
		acked     bool
	}

	msgMap := make(map[int]*msgState)

	for {
		var entry LogEntry
		if err := decoder.Decode(&entry); err != nil {
			break
		}

		id := entry.Message.ID
		state, exists := msgMap[id]

		if !exists {
			state = &msgState{}
			msgMap[id] = state
		}

		state.message = entry.Message

		switch entry.Type {
		case "enqueue":
			state.enqueued = true
		case "deliver":
			state.delivered = true
		case "ack":
			state.acked = true
		}

		if id >= t.nextID {
			t.nextID = id + 1
		}

	}

	for id, state := range msgMap {
		if state.acked {
			continue // skip fully ACKed messages
		}

		if state.delivered {
			t.inFlight[id] = &state.message
		} else if state.enqueued {
			t.messages = append(t.messages, state.message)
		}
	}

}
