package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	q "github.com/suman7383/go-queue/internal/queue"
	serializepb "github.com/suman7383/go-queue/internal/serialize"
	"google.golang.org/protobuf/proto"
)

type HTTPServer struct {
	Registry *q.TopicRegistry
}

func NewHttpServer(registry *q.TopicRegistry) *HTTPServer {
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

	message, err := extractMessage(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	topic = s.Registry.GetTopic(topicName)
	_, err = topic.Enqueue(message)
	if err != nil {
		http.Error(w, "Failed to enqueue message", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "OK\n")
}

func (s *HTTPServer) handleConsume(w http.ResponseWriter, r *http.Request) {
	topicName := strings.TrimPrefix(r.URL.Path, "/consume/")
	topic := s.Registry.GetTopic(topicName)
	if topic == nil {
		http.Error(w, "Topic not found", http.StatusNotFound)
		return
	}

	msg, ok := topic.Dequeue()
	if !ok {
		http.Error(w, q.ErrEmptyQueue.Error(), http.StatusNoContent)
		return
	}

	encodeAndSendResponse(w, r, msg)
}

// Route -> /ack/[TOPIC-NAME]/[TOPIC-ID]
func (s *HTTPServer) handleAck(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	topicName := parts[2]
	id, err := strconv.ParseInt(parts[3], 10, 64)

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
	fmt.Fprint(w, "OK\n")
}

// Checks for content-type and
// extract message accordingly
func extractMessage(r *http.Request) (string, error) {
	// Check for protobuf
	if isProtoRequest(r) {
		// Read raw bytes from request body
		body, err := io.ReadAll(r.Body)

		if err != nil {
			return "", errors.New("failed to read request body")
		}

		var payload serializepb.Produce
		if err := proto.Unmarshal(body, &payload); err != nil {
			return "", errors.New("failed to unmarshal protobuf")
		}

		return payload.Message, nil
	}

	// Json
	var payload struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return "", errors.New("Invalid body")
	}

	return payload.Message, nil
}

// Checks if content-type is protobuf
// then send response accordingly
func encodeAndSendResponse(w http.ResponseWriter, r *http.Request, msg q.Message) {
	// protobuf
	if acceptProtoResponse(r) {
		msgpb := serializepb.FromMessage(msg)
		data, err := proto.Marshal(msgpb)

		if err != nil {
			http.Error(w, "failed to marshal protobuf", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	} else {
		// json
		json.NewEncoder(w).Encode(msg)
	}
}

// Checks if content type is protobuf
// Header: "Content-Type: application/x-protobuf"
func isProtoRequest(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Content-Type"), "application/x-protobuf")
}

func acceptProtoResponse(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/x-protobuf")
}
