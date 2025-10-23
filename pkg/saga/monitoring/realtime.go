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

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// RealtimePusher handles real-time updates via Server-Sent Events (SSE).
// It broadcasts metrics updates to connected clients at regular intervals.
type RealtimePusher struct {
	collector MetricsCollector

	// Configuration
	updateInterval time.Duration
	bufferSize     int

	// Client management
	mu      sync.RWMutex
	clients map[string]*sseClient
	nextID  int

	// Control channels
	stopCh chan struct{}
	doneCh chan struct{}
}

// sseClient represents a connected SSE client.
type sseClient struct {
	id       string
	channel  chan *MetricsUpdate
	context  context.Context
	cancelFn context.CancelFunc
}

// MetricsUpdate represents a real-time metrics update.
type MetricsUpdate struct {
	Type      string         `json:"type"`
	Data      MetricsSummary `json:"data"`
	Timestamp time.Time      `json:"timestamp"`
}

// RealtimePusherConfig contains configuration for the real-time pusher.
type RealtimePusherConfig struct {
	// UpdateInterval is how often to broadcast metrics updates
	// Default: 2 seconds
	UpdateInterval time.Duration

	// BufferSize is the channel buffer size for each client
	// Default: 10
	BufferSize int
}

// DefaultRealtimePusherConfig returns default configuration.
func DefaultRealtimePusherConfig() *RealtimePusherConfig {
	return &RealtimePusherConfig{
		UpdateInterval: 2 * time.Second,
		BufferSize:     10,
	}
}

// NewRealtimePusher creates a new real-time pusher.
//
// Parameters:
//   - collector: The metrics collector to read from.
//   - config: Configuration for the pusher. If nil, defaults are used.
//
// Returns:
//   - A configured RealtimePusher ready to accept connections.
func NewRealtimePusher(collector MetricsCollector, config *RealtimePusherConfig) *RealtimePusher {
	if config == nil {
		config = DefaultRealtimePusherConfig()
	}

	pusher := &RealtimePusher{
		collector:      collector,
		updateInterval: config.UpdateInterval,
		bufferSize:     config.BufferSize,
		clients:        make(map[string]*sseClient),
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
	}

	return pusher
}

// Start begins broadcasting metrics updates to connected clients.
func (rp *RealtimePusher) Start() {
	if logger.Logger != nil {
		logger.Logger.Info("Starting real-time metrics pusher",
			zap.Duration("update_interval", rp.updateInterval))
	}

	go rp.broadcastLoop()
}

// Stop halts the real-time pusher and disconnects all clients.
func (rp *RealtimePusher) Stop() {
	if logger.Logger != nil {
		logger.Logger.Info("Stopping real-time metrics pusher")
	}

	close(rp.stopCh)

	// Wait for broadcast loop to finish
	<-rp.doneCh

	// Disconnect all clients
	rp.mu.Lock()
	for _, client := range rp.clients {
		client.cancelFn()
		close(client.channel)
	}
	rp.clients = make(map[string]*sseClient)
	rp.mu.Unlock()

	if logger.Logger != nil {
		logger.Logger.Info("Real-time metrics pusher stopped")
	}
}

// HandleSSE handles Server-Sent Events connections for real-time metrics.
//
// This endpoint streams metrics updates to clients in real-time using SSE.
// Clients can connect and receive periodic updates without polling.
//
// Usage:
//
//	curl -N http://localhost:8090/api/metrics/stream
//
// Response format:
//
//	data: {"type":"metrics","data":{...},"timestamp":"2025-10-23T10:30:00Z"}
//	data: {"type":"metrics","data":{...},"timestamp":"2025-10-23T10:30:02Z"}
func (rp *RealtimePusher) HandleSSE(c *gin.Context) {
	// Set headers for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// Create client
	clientCtx, cancel := context.WithCancel(c.Request.Context())
	client := &sseClient{
		id:       rp.generateClientID(),
		channel:  make(chan *MetricsUpdate, rp.bufferSize),
		context:  clientCtx,
		cancelFn: cancel,
	}

	// Register client
	rp.mu.Lock()
	rp.clients[client.id] = client
	rp.mu.Unlock()

	if logger.Logger != nil {
		logger.Logger.Info("New SSE client connected",
			zap.String("client_id", client.id),
			zap.Int("total_clients", len(rp.clients)))
	}

	// Cleanup on disconnect
	defer func() {
		rp.mu.Lock()
		delete(rp.clients, client.id)
		clientCount := len(rp.clients)
		rp.mu.Unlock()

		cancel()
		close(client.channel)

		if logger.Logger != nil {
			logger.Logger.Info("SSE client disconnected",
				zap.String("client_id", client.id),
				zap.Int("remaining_clients", clientCount))
		}
	}()

	// Stream updates to client
	for {
		select {
		case update, ok := <-client.channel:
			if !ok {
				return
			}

			// Marshal update to JSON
			data, err := json.Marshal(update)
			if err != nil {
				if logger.Logger != nil {
					logger.Logger.Error("Failed to marshal metrics update",
						zap.String("client_id", client.id),
						zap.Error(err))
				}
				continue
			}

			// Send SSE message
			_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			if err != nil {
				if logger.Logger != nil {
					logger.Logger.Debug("Failed to write to client",
						zap.String("client_id", client.id),
						zap.Error(err))
				}
				return
			}

			// Flush the response writer
			c.Writer.Flush()

		case <-client.context.Done():
			return

		case <-c.Request.Context().Done():
			return
		}
	}
}

// broadcastLoop periodically broadcasts metrics updates to all connected clients.
func (rp *RealtimePusher) broadcastLoop() {
	ticker := time.NewTicker(rp.updateInterval)
	defer ticker.Stop()
	defer close(rp.doneCh)

	for {
		select {
		case <-ticker.C:
			// Get current metrics
			metrics := rp.collector.GetMetrics()
			summary := calculateMetricsSummary(metrics)

			// Create update
			update := &MetricsUpdate{
				Type:      "metrics",
				Data:      summary,
				Timestamp: time.Now(),
			}

			// Broadcast to all clients
			rp.broadcast(update)

		case <-rp.stopCh:
			return
		}
	}
}

// broadcast sends a metrics update to all connected clients.
func (rp *RealtimePusher) broadcast(update *MetricsUpdate) {
	rp.mu.RLock()
	clientCount := len(rp.clients)
	clients := make([]*sseClient, 0, clientCount)
	for _, client := range rp.clients {
		clients = append(clients, client)
	}
	rp.mu.RUnlock()

	if clientCount == 0 {
		return
	}

	if logger.Logger != nil {
		logger.Logger.Debug("Broadcasting metrics update",
			zap.Int("client_count", clientCount),
			zap.Int64("active_sagas", update.Data.Active))
	}

	// Send to all clients (non-blocking)
	for _, client := range clients {
		select {
		case client.channel <- update:
			// Successfully sent
		default:
			// Channel is full, skip this update for this client
			if logger.Logger != nil {
				logger.Logger.Warn("Client channel full, skipping update",
					zap.String("client_id", client.id))
			}
		}
	}
}

// generateClientID generates a unique ID for a client.
func (rp *RealtimePusher) generateClientID() string {
	rp.nextID++
	return fmt.Sprintf("client-%d-%d", time.Now().Unix(), rp.nextID)
}

// GetConnectedClientsCount returns the number of currently connected SSE clients.
func (rp *RealtimePusher) GetConnectedClientsCount() int {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return len(rp.clients)
}
