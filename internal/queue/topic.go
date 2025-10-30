package queue

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/suman7383/go-queue/internal/ringbuffer"
)

// Topic represents a named message queue
type Topic struct {
	Name     string
	messages Queue[Message]
	inFlight map[int64]Message // delivered but not yet acked
	nextID   int64
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
		nextID:   1,
		messages: ringbuffer.NewRingBuffer[Message](10000),
		inFlight: make(map[int64]Message),
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
						log.Printf("[Retry] Topic: %s | Msg ID %d | Retry #%d\n", t.Name, msg.ID, msg.Retries)
						t.messages.Enqueue(msg) // Requeue

					} else {
						// max retry reached -> discard the message
						log.Printf("[DROP]: Msg ID %d exceeded max retries (%d). Discarded.\n", msg.ID, t.config.MaxRetries)
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
func (t *Topic) Enqueue(payload string) (int64, error) {

	t.mu.Lock()
	defer t.mu.Unlock()
	msg := Message{
		ID:      t.nextID,
		Payload: payload,
	}

	// Append to WAL
	t.wal.AppendEvent("enqueue", msg)

	t.nextID++
	t.messages.Enqueue(msg)
	// t.messages.enqueue(msg)

	return msg.ID, nil
}

// Dequeue returns the next message if exists(pull)
func (t *Topic) Dequeue() (Message, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// if len(t.messages) == 0 {
	// 	return Message{}, false
	// }

	msg, ok := t.messages.Dequeue()

	if !ok {
		return msg, ok
	}

	t.wal.AppendEvent("deliver", msg)

	// t.messages = t.messages[1:]

	msg.Timestamp = time.Now()
	msg.Acked = false
	t.inFlight[msg.ID] = msg

	return msg, true
}

func (t *Topic) Acknowledge(id int64) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	msg, ok := t.inFlight[id]
	if !ok {
		return false
	}
	msg.Acked = true

	// Append to WAL
	t.wal.AppendEvent("ack", msg)

	delete(t.inFlight, id)

	return true
}

func (t *Topic) replayWAL() {
	path := fmt.Sprintf("data/%s.wal", t.Name)
	file, err := os.Open(path)
	if err != nil {
		log.Println("No WAL found for topic:", t.Name)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	type msgState struct {
		message   Message
		enqueued  bool
		delivered bool
		acked     bool
	}

	msgMap := make(map[int64]*msgState)

	for {
		// --- Decode Type (uint16 length + string) ---
		var typeLen uint16
		if err := binary.Read(reader, binary.LittleEndian, &typeLen); err != nil {
			break // reached EOF
		}

		typeBytes := make([]byte, typeLen)
		if _, err := reader.Read(typeBytes); err != nil {
			break
		}
		entryType := string(typeBytes)

		// --- Decode Message ID ---
		var id int64
		if err := binary.Read(reader, binary.LittleEndian, &id); err != nil {
			break
		}

		// --- Decode Payload (uint32 length + string) ---
		var payloadLen uint32
		if err := binary.Read(reader, binary.LittleEndian, &payloadLen); err != nil {
			break
		}

		payloadBytes := make([]byte, payloadLen)
		if _, err := reader.Read(payloadBytes); err != nil {
			break
		}

		// --- Decode Timestamp ---
		var ts int64
		if err := binary.Read(reader, binary.LittleEndian, &ts); err != nil {
			break
		}

		// --- Decode Acked ---
		var acked uint8
		if err := binary.Read(reader, binary.LittleEndian, &acked); err != nil {
			break
		}

		// --- Decode Retries ---
		var retries int32
		if err := binary.Read(reader, binary.LittleEndian, &retries); err != nil {
			break
		}

		// Build message
		msg := Message{
			ID:        int64(id),
			Payload:   string(payloadBytes),
			Timestamp: time.Unix(0, ts),
			Acked:     acked == 1,
			Retries:   int(retries),
		}

		// Track state
		state, exists := msgMap[msg.ID]
		if !exists {
			state = &msgState{}
			msgMap[msg.ID] = state
		}
		state.message = msg

		switch entryType {
		case "enqueue":
			state.enqueued = true
		case "deliver":
			state.delivered = true
		case "ack":
			state.acked = true
		}

		// Ensure nextID is larger than any ID seen
		if msg.ID >= t.nextID {
			t.nextID = msg.ID + 1
		}
	}

	// Rebuild topic state
	for id, state := range msgMap {
		if state.acked {
			continue // skip fully ACKed messages
		}

		if state.delivered {
			t.inFlight[id] = state.message
		} else if state.enqueued {
			t.messages.Enqueue(state.message)
		}
	}
}
