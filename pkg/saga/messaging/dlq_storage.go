// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package messaging

import (
	"context"
	"fmt"
	"sync"
)

// DLQStorage defines the storage abstraction for dead-letter queue messages.
// It backs the cleanup and recovery flows of DeadLetterQueueHandler, aligning
// the saga messaging DLQ capabilities with pkg/resilience DLQManager.
type DLQStorage interface {
	// Store persists a DLQ message. Storing a message with an existing ID
	// replaces the previous entry.
	Store(ctx context.Context, msg *DLQMessage) error

	// Get returns the DLQ message with the given ID, or an error if not found.
	Get(ctx context.Context, id string) (*DLQMessage, error)

	// List returns a snapshot of all DLQ messages in FIFO order.
	List(ctx context.Context) ([]*DLQMessage, error)

	// Remove deletes the DLQ message with the given ID.
	Remove(ctx context.Context, id string) error

	// Size returns the number of messages currently stored.
	Size(ctx context.Context) (int, error)
}

// InMemoryDLQStorage is the default in-memory implementation of DLQStorage.
// It keeps messages in FIFO order and optionally evicts the oldest message
// when a maximum size is configured (mirroring resilience.DLQManager behavior).
type InMemoryDLQStorage struct {
	mu       sync.RWMutex
	messages map[string]*DLQMessage
	queue    []string // message IDs in FIFO order
	maxSize  int      // 0 means unlimited
}

// NewInMemoryDLQStorage creates a new in-memory DLQ storage.
// maxSize limits the number of stored messages; 0 means unlimited.
func NewInMemoryDLQStorage(maxSize int) *InMemoryDLQStorage {
	return &InMemoryDLQStorage{
		messages: make(map[string]*DLQMessage),
		queue:    make([]string, 0),
		maxSize:  maxSize,
	}
}

// Store implements DLQStorage interface.
func (s *InMemoryDLQStorage) Store(ctx context.Context, msg *DLQMessage) error {
	if msg == nil {
		return fmt.Errorf("DLQ message cannot be nil")
	}
	if msg.ID == "" {
		return fmt.Errorf("DLQ message ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.messages[msg.ID]; exists {
		s.messages[msg.ID] = msg
		return nil
	}

	// Evict the oldest message when the capacity limit is reached.
	if s.maxSize > 0 && len(s.queue) >= s.maxSize {
		oldestID := s.queue[0]
		s.queue = s.queue[1:]
		delete(s.messages, oldestID)
	}

	s.messages[msg.ID] = msg
	s.queue = append(s.queue, msg.ID)
	return nil
}

// Get implements DLQStorage interface.
func (s *InMemoryDLQStorage) Get(ctx context.Context, id string) (*DLQMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg, ok := s.messages[id]
	if !ok {
		return nil, fmt.Errorf("DLQ message not found: %s", id)
	}
	return msg, nil
}

// List implements DLQStorage interface.
func (s *InMemoryDLQStorage) List(ctx context.Context) ([]*DLQMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*DLQMessage, 0, len(s.queue))
	for _, id := range s.queue {
		if msg, ok := s.messages[id]; ok {
			result = append(result, msg)
		}
	}
	return result, nil
}

// Remove implements DLQStorage interface.
func (s *InMemoryDLQStorage) Remove(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.messages[id]; !ok {
		return fmt.Errorf("DLQ message not found: %s", id)
	}

	delete(s.messages, id)
	for i, queuedID := range s.queue {
		if queuedID == id {
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
			break
		}
	}
	return nil
}

// Size implements DLQStorage interface.
func (s *InMemoryDLQStorage) Size(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.queue), nil
}
