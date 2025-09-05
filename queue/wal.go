package queue

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type WAL struct {
	file    *os.File
	writer  *bufio.Writer
	topic   string
	walChan chan LogEntry
	wg      sync.WaitGroup
	closeCh chan struct{}
}

type LogEntry struct {
	Type    string // "enqueue" | "deliver" | "ack"
	Message Message
}

func NewWAL(topicName string) (*WAL, error) {
	path := fmt.Sprintf("data/%s.wal", topicName)
	os.MkdirAll("data", os.ModePerm)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	w := &WAL{
		file:    file,
		writer:  bufio.NewWriterSize(file, 1<<20), // 1MB buffer
		topic:   topicName,
		walChan: make(chan LogEntry, 10000),
		closeCh: make(chan struct{}),
	}

	// Start WAL writer goroutine
	w.wg.Add(1)
	go w.runWriter()

	return w, nil
}

// AppendEvent queues an entry for persistence (non-blocking if buffer is available).
// Blocks if the channel is full to ensure no data loss.
func (w *WAL) AppendEvent(eventType string, msg Message) error {
	entry := LogEntry{
		Type:    eventType,
		Message: msg,
	}
	w.walChan <- entry
	return nil
}

// Background WAL writer
func (w *WAL) runWriter() {
	defer w.wg.Done()

	batch := make([]LogEntry, 0, 1000)

	flush := func() {
		for _, e := range batch {
			if err := w.encodeEntry(e); err != nil {
				// In production: push to errChan or panic based on durability needs
				log.Printf("[WAL ERROR] encode failed: %v\n", err)
			}
		}
		batch = batch[:0]
		w.writer.Flush()
	}

	ticker := time.NewTicker(50 * time.Millisecond) // flush interval
	defer ticker.Stop()

	for {
		select {
		case e, ok := <-w.walChan:
			if !ok {
				if len(batch) > 0 {
					flush()
				}
				return
			}
			batch = append(batch, e)
			if len(batch) >= 100 { // batch size
				flush()
			}
		case <-ticker.C:
			if len(batch) > 0 {
				flush()
			}
		case <-w.closeCh:
			if len(batch) > 0 {
				flush()
			}
			return
		}
	}
}

// encodeEntry writes a single LogEntry in binary format.
func (w *WAL) encodeEntry(entry LogEntry) error {
	// Encode Type as length-prefixed string
	if err := binary.Write(w.writer, binary.LittleEndian, uint16(len(entry.Type))); err != nil {
		return err
	}
	if _, err := w.writer.WriteString(entry.Type); err != nil {
		return err
	}

	// Encode Message ID
	if err := binary.Write(w.writer, binary.LittleEndian, int64(entry.Message.ID)); err != nil {
		return err
	}

	// Encode Payload as length-prefixed string
	if err := binary.Write(w.writer, binary.LittleEndian, uint32(len(entry.Message.Payload))); err != nil {
		return err
	}
	if _, err := w.writer.WriteString(entry.Message.Payload); err != nil {
		return err
	}

	// Encode Timestamp (UnixNano)
	if err := binary.Write(w.writer, binary.LittleEndian, entry.Message.Timestamp.UnixNano()); err != nil {
		return err
	}

	// Encode Acked + Retries
	acked := uint8(0)
	if entry.Message.Acked {
		acked = 1
	}
	if err := binary.Write(w.writer, binary.LittleEndian, acked); err != nil {
		return err
	}
	if err := binary.Write(w.writer, binary.LittleEndian, int32(entry.Message.Retries)); err != nil {
		return err
	}

	return nil
}

// Close gracefully shuts down the WAL writer and flushes everything.
func (w *WAL) Close() {
	close(w.closeCh)
	close(w.walChan)
	w.wg.Wait()
	w.writer.Flush()
	w.file.Close()
}
