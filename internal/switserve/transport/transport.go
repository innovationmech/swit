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

package transport

import (
	"context"
	"sync"
	"time"
)

// Transport defines the interface for different transport mechanisms
type Transport interface {
	// Start starts the transport server
	Start(ctx context.Context) error
	// Stop gracefully stops the transport server
	Stop(ctx context.Context) error
	// Name returns the transport name
	Name() string
	// Address returns the listening address
	Address() string
}

// Manager manages multiple transport instances
type Manager struct {
	transports []Transport
	mu         sync.RWMutex
}

// NewManager creates a new transport manager
func NewManager() *Manager {
	return &Manager{
		transports: make([]Transport, 0),
	}
}

// Register adds a transport to the manager
func (m *Manager) Register(transport Transport) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transports = append(m.transports, transport)
}

// Start starts all registered transports
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(m.transports))

	for _, transport := range m.transports {
		wg.Add(1)
		go func(t Transport) {
			defer wg.Done()
			if err := t.Start(ctx); err != nil {
				errChan <- err
			}
		}(transport)
	}

	wg.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop gracefully stops all transports
func (m *Manager) Stop(timeout time.Duration) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var wg sync.WaitGroup
	for _, transport := range m.transports {
		wg.Add(1)
		go func(t Transport) {
			defer wg.Done()
			_ = t.Stop(ctx)
		}(transport)
	}

	wg.Wait()
	return nil
}

// GetTransports returns all registered transports
func (m *Manager) GetTransports() []Transport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Transport, len(m.transports))
	copy(result, m.transports)
	return result
}
