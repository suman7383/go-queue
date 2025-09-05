package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type HTTPServer struct {
	Registry *TopicRegistry
}

func NewHttpServer(registry *TopicRegistry) *HTTPServer {
	return &HTTPServer{Registry: registry}
}

func (s *HTTPServer) Start(addr string) {
	http.HandleFunc("/produce/", s.handleProduce)
	http.HandleFunc("/consume/", s.handleConsume)
	http.HandleFunc("/ack/", s.handleAck)

	log.Println("[HTTP] Server running at", addr)
	http.ListenAndServe(addr, nil)
}

func (s *HTTPServer) handleProduce(w http.ResponseWriter, r *http.Request) {
	topicName := strings.TrimPrefix(r.URL.Path, "/produce/")
	topic := s.Registry.GetTopic(topicName)

	if topic == nil {
		s.Registry.CreateTopic(topicName)
	}

	var payload struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	// log.Print(payload)

	topic = s.Registry.GetTopic(topicName)
	_, err := topic.Enqueue(payload.Message)
	if err != nil {
		http.Error(w, "Failed to enqueue message", http.StatusInternalServerError)
		return
	}
	// log.Fprintf(w, "Message enqueued with ID %d\n", id)
}

func (s *HTTPServer) handleConsume(w http.ResponseWriter, r *http.Request) {
	topicName := strings.TrimPrefix(r.URL.Path, "/consume/")
	topic := s.Registry.GetTopic(topicName)
	if topic == nil {
		http.Error(w, "Topic not found", http.StatusNotFound)
		return
	}

	msg, err := topic.Dequeue()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNoContent)
		return
	}

	json.NewEncoder(w).Encode(msg)
}

// Route -> /ack/[TOPIC-NAME]/[TOPIC-ID]
func (s *HTTPServer) handleAck(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	topicName := parts[2]
	id, err := strconv.Atoi(parts[3])

	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	topic := s.Registry.GetTopic(topicName)
	if topic == nil {
		http.Error(w, "Topic not found", http.StatusNotFound)
		return
	}

	topic.Acknowledge(id)
	fmt.Fprintf(w, "Acknowledged message ID %d\n", id)
}
