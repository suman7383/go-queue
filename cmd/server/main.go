package main

import (
	"time"

	s "github.com/suman7383/go-queue/internal/http"
	"github.com/suman7383/go-queue/internal/queue"
)

func main() {
	config := queue.TopicConfig{
		AckTimeout: 30 * time.Second,
		MaxRetries: 3,
	}

	// create a registry
	registry := queue.NewTopicRegistry(config)

	// Recover topics from disk BEFORE producers/consumers
	registry.LoadTopicFromDisk(config)

	server := s.NewHttpServer(registry)
	server.Start(":8080")
}
