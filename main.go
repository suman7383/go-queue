package main

import (
	"time"

	"github.com/suman7383/go-queue/queue"
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

	server := queue.NewHttpServer(registry)
	server.Start(":8080")
}
