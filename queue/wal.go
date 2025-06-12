package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type WAL struct {
	mu    sync.Mutex
	file  *os.File
	enc   *json.Encoder
	topic string
}

type LogEntry struct {
	Type    string  `json:"type"` // "enqueue" | "deliver"
	Message Message `json:"message"`
}

func NewWAL(topicName string) (*WAL, error) {
	path := fmt.Sprintf("data/%s.wal", topicName)

	// create folder if not exists
	os.MkdirAll("data", os.ModePerm)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &WAL{
		file:  file,
		enc:   json.NewEncoder(file),
		topic: topicName,
	}, nil
}

func (w *WAL) Append(msg Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.enc.Encode(msg)
}

func (w *WAL) AppendEvent(eventType string, msg Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	entry := LogEntry{
		Type:    eventType,
		Message: msg,
	}

	return w.enc.Encode(entry)
}

func (w *WAL) Close() {
	w.file.Close()
}
