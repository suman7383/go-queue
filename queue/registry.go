package queue

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

type TopicRegistry struct {
	topics map[string]*Topic
	mu     sync.RWMutex
	config TopicConfig
}

// Creates a new empty registry
func  NewTopicRegistry(config TopicConfig) *TopicRegistry {
	return &TopicRegistry{
		topics: make(map[string]*Topic),
		config: config,
	}
}

// creates a new topic
func (r *TopicRegistry) CreateTopic(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.topics[name]; !exists {
		r.topics[name] = NewTopic(name, r.config)
		fmt.Println("Topic created:", name)
	}
}

// Returns an existing topic
func (r *TopicRegistry) GetTopic(name string) *Topic {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.topics[name]
}

func (r *TopicRegistry) LoadTopicFromDisk(defaultConfig TopicConfig) {
	files, err := os.ReadDir("data")
	if err != nil {
		fmt.Println("No WALs found on disk.")
		return
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".wal") {
			topicName := strings.TrimSuffix(file.Name(), ".wal")

			r.mu.Lock()
			if _, exists := r.topics[topicName]; !exists {
				r.topics[topicName] = NewTopic(topicName, defaultConfig)
				fmt.Printf("[Recovery] Topic '%s' loaded from WAL.\n", topicName)
			}
			r.mu.Unlock()
		}
	}
}
