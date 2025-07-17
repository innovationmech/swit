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
	"fmt"
	"sync"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Transport defines the interface for different transport protocols
// (HTTP, gRPC, WebSocket, etc.)
type Transport interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetName() string
}

// Manager manages multiple transport protocols
// Provides unified control over HTTP, gRPC, and future transports
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
	logger.Logger.Info("Transport registered",
		zap.String("transport", transport.GetName()))
}

// Start starts all registered transports
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	transports := make([]Transport, len(m.transports))
	copy(transports, m.transports)
	m.mu.RUnlock()

	for _, transport := range transports {
		logger.Logger.Info("Starting transport",
			zap.String("transport", transport.GetName()))
		if err := transport.Start(ctx); err != nil {
			return fmt.Errorf("failed to start %s transport: %w", transport.GetName(), err)
		}
	}

	logger.Logger.Info("All transports started successfully")
	return nil
}

// Stop gracefully stops all registered transports
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.RLock()
	transports := make([]Transport, len(m.transports))
	copy(transports, m.transports)
	m.mu.RUnlock()

	for _, transport := range transports {
		logger.Logger.Info("Stopping transport",
			zap.String("transport", transport.GetName()))
		if err := transport.Stop(ctx); err != nil {
			logger.Logger.Error("Failed to stop transport",
				zap.String("transport", transport.GetName()),
				zap.Error(err))
			// Continue stopping other transports
		}
	}

	logger.Logger.Info("All transports stopped")
	return nil
}

// GetTransports returns a list of all registered transports
func (m *Manager) GetTransports() []Transport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	transports := make([]Transport, len(m.transports))
	copy(transports, m.transports)
	return transports
}
