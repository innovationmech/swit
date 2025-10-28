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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRealtimePusher tests the creation of RealtimePusher.
func TestNewRealtimePusher(t *testing.T) {
	tests := []struct {
		name   string
		config *RealtimePusherConfig
	}{
		{
			name:   "with nil config",
			config: nil,
		},
		{
			name: "with custom config",
			config: &RealtimePusherConfig{
				UpdateInterval: 1 * time.Second,
				BufferSize:     20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			collectorConfig := &Config{
				Registry: registry,
			}
			collector, err := NewSagaMetricsCollector(collectorConfig)
			require.NoError(t, err)

			pusher := NewRealtimePusher(collector, tt.config)
			assert.NotNil(t, pusher)
			assert.NotNil(t, pusher.collector)
			assert.NotNil(t, pusher.clients)
			assert.NotNil(t, pusher.stopCh)
			assert.NotNil(t, pusher.doneCh)

			if tt.config != nil {
				assert.Equal(t, tt.config.UpdateInterval, pusher.updateInterval)
				assert.Equal(t, tt.config.BufferSize, pusher.bufferSize)
			} else {
				// Check defaults
				assert.Equal(t, 2*time.Second, pusher.updateInterval)
				assert.Equal(t, 10, pusher.bufferSize)
			}
		})
	}
}

// TestDefaultRealtimePusherConfig tests the default configuration.
func TestDefaultRealtimePusherConfig(t *testing.T) {
	config := DefaultRealtimePusherConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 2*time.Second, config.UpdateInterval)
	assert.Equal(t, 10, config.BufferSize)
}

// TestRealtimePusher_StartStop tests starting and stopping the pusher.
func TestRealtimePusher_StartStop(t *testing.T) {
	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(t, err)

	config := &RealtimePusherConfig{
		UpdateInterval: 100 * time.Millisecond,
		BufferSize:     10,
	}
	pusher := NewRealtimePusher(collector, config)

	// Start pusher
	pusher.Start()

	// Wait a bit to ensure it's running
	time.Sleep(150 * time.Millisecond)

	// Stop pusher
	pusher.Stop()

	// Verify it stopped cleanly
	assert.Equal(t, 0, pusher.GetConnectedClientsCount())
}

// TestRealtimePusher_HandleSSE tests the SSE endpoint.
func TestRealtimePusher_HandleSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(t, err)

	config := &RealtimePusherConfig{
		UpdateInterval: 100 * time.Millisecond,
		BufferSize:     10,
	}
	pusher := NewRealtimePusher(collector, config)
	pusher.Start()
	defer pusher.Stop()

	// Create test request with cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/api/metrics/stream", nil)
	req = req.WithContext(ctx)
	c.Request = req

	// Record some metrics to get updates
	go func() {
		time.Sleep(50 * time.Millisecond)
		collector.RecordSagaStarted("saga-1")
		collector.RecordSagaCompleted("saga-1", 100*time.Millisecond)
	}()

	// Handle SSE connection in a goroutine
	done := make(chan bool)
	go func() {
		pusher.HandleSSE(c)
		done <- true
	}()

	// Wait for context timeout or handler completion
	select {
	case <-done:
		// Handler completed
	case <-time.After(1 * time.Second):
		t.Fatal("Handler did not complete in time")
	}

	// Check headers
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))

	// Check that we received some SSE data
	body := w.Body.String()
	assert.Contains(t, body, "data:")
}

// TestRealtimePusher_MultipleClients tests multiple SSE connections.
func TestRealtimePusher_MultipleClients(t *testing.T) {
	t.Skip("Flaky test due to goroutine scheduling and timing dependencies. See issue #747")
	gin.SetMode(gin.TestMode)

	// Setup
	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(t, err)

	config := &RealtimePusherConfig{
		UpdateInterval: 100 * time.Millisecond,
		BufferSize:     10,
	}
	pusher := NewRealtimePusher(collector, config)
	pusher.Start()
	defer pusher.Stop()

	// Create multiple clients
	numClients := 3
	clients := make([]*httptest.ResponseRecorder, numClients)
	contexts := make([]context.Context, numClients)
	cancels := make([]context.CancelFunc, numClients)
	doneChannels := make([]chan bool, numClients)
	startedChannels := make([]chan struct{}, numClients)

	for i := 0; i < numClients; i++ {
		clients[i] = httptest.NewRecorder()
		// Increase timeout to 2 seconds to ensure clients stay connected during test
		contexts[i], cancels[i] = context.WithTimeout(context.Background(), 2*time.Second)
		doneChannels[i] = make(chan bool)
		startedChannels[i] = make(chan struct{})

		c, _ := gin.CreateTestContext(clients[i])
		req := httptest.NewRequest("GET", "/api/metrics/stream", nil)
		req = req.WithContext(contexts[i])
		c.Request = req

		// Start each client handler
		go func(idx int, ctx *gin.Context) {
			// Signal that goroutine has started
			close(startedChannels[idx])
			pusher.HandleSSE(ctx)
			doneChannels[idx] <- true
		}(i, c)
	}

	// Wait for all goroutines to start
	for i := 0; i < numClients; i++ {
		<-startedChannels[i]
	}

	// Wait for all clients to be registered and connected
	// Use polling with generous timeout to handle scheduler variations
	var clientCount int
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		clientCount = pusher.GetConnectedClientsCount()
		if clientCount == numClients {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, numClients, clientCount, "Expected all clients to be connected")

	// Record some metrics
	collector.RecordSagaStarted("saga-1")
	collector.RecordSagaCompleted("saga-1", 100*time.Millisecond)

	// Wait a bit for metrics to be broadcasted
	time.Sleep(200 * time.Millisecond)

	// Cancel all client contexts to trigger disconnect
	for i := 0; i < numClients; i++ {
		cancels[i]()
	}

	// Wait for all clients to finish
	for i := 0; i < numClients; i++ {
		select {
		case <-doneChannels[i]:
			// Client finished
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Client %d did not complete in time", i)
		}
	}

	// Verify all clients disconnected
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, pusher.GetConnectedClientsCount(), "Expected all clients to disconnect")
}

// TestRealtimePusher_Broadcast tests the broadcast mechanism.
func TestRealtimePusher_Broadcast(t *testing.T) {
	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(t, err)

	config := &RealtimePusherConfig{
		UpdateInterval: 50 * time.Millisecond,
		BufferSize:     10,
	}
	pusher := NewRealtimePusher(collector, config)

	// Create a test client manually
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &sseClient{
		id:       "test-client",
		channel:  make(chan *MetricsUpdate, config.BufferSize),
		context:  ctx,
		cancelFn: cancel,
	}

	pusher.mu.Lock()
	pusher.clients[client.id] = client
	pusher.mu.Unlock()

	// Create test update
	update := &MetricsUpdate{
		Type: "metrics",
		Data: MetricsSummary{
			Total:     10,
			Active:    2,
			Completed: 8,
		},
		Timestamp: time.Now(),
	}

	// Broadcast update
	pusher.broadcast(update)

	// Verify client received update
	select {
	case received := <-client.channel:
		assert.Equal(t, "metrics", received.Type)
		assert.Equal(t, int64(10), received.Data.Total)
		assert.Equal(t, int64(2), received.Data.Active)
	case <-time.After(1 * time.Second):
		t.Fatal("Client did not receive update")
	}
}

// TestRealtimePusher_BroadcastWithFullChannel tests broadcast when a client channel is full.
func TestRealtimePusher_BroadcastWithFullChannel(t *testing.T) {
	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(t, err)

	config := &RealtimePusherConfig{
		UpdateInterval: 50 * time.Millisecond,
		BufferSize:     2, // Small buffer
	}
	pusher := NewRealtimePusher(collector, config)

	// Create a test client with a small channel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &sseClient{
		id:       "test-client",
		channel:  make(chan *MetricsUpdate, config.BufferSize),
		context:  ctx,
		cancelFn: cancel,
	}

	pusher.mu.Lock()
	pusher.clients[client.id] = client
	pusher.mu.Unlock()

	// Fill the channel
	for i := 0; i < config.BufferSize; i++ {
		update := &MetricsUpdate{
			Type:      "metrics",
			Data:      MetricsSummary{Total: int64(i)},
			Timestamp: time.Now(),
		}
		client.channel <- update
	}

	// Try to broadcast another update (channel is full)
	extraUpdate := &MetricsUpdate{
		Type:      "metrics",
		Data:      MetricsSummary{Total: 999},
		Timestamp: time.Now(),
	}

	// This should not block
	pusher.broadcast(extraUpdate)

	// Verify channel still has original updates
	assert.Len(t, client.channel, config.BufferSize)
}

// TestRealtimePusher_ClientIDGeneration tests unique client ID generation.
func TestRealtimePusher_ClientIDGeneration(t *testing.T) {
	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(t, err)

	pusher := NewRealtimePusher(collector, nil)

	// Generate multiple IDs
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := pusher.generateClientID()
		assert.NotEmpty(t, id)
		assert.False(t, ids[id], "Generated duplicate client ID")
		ids[id] = true
	}
}

// TestRealtimePusher_GetConnectedClientsCount tests the client count getter.
func TestRealtimePusher_GetConnectedClientsCount(t *testing.T) {
	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(t, err)

	pusher := NewRealtimePusher(collector, nil)

	// Initially no clients
	assert.Equal(t, 0, pusher.GetConnectedClientsCount())

	// Add some clients
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		client := &sseClient{
			id:       fmt.Sprintf("client-%d", i),
			channel:  make(chan *MetricsUpdate, 10),
			context:  ctx,
			cancelFn: cancel,
		}
		pusher.mu.Lock()
		pusher.clients[client.id] = client
		pusher.mu.Unlock()
	}

	assert.Equal(t, 5, pusher.GetConnectedClientsCount())

	// Remove a client
	pusher.mu.Lock()
	delete(pusher.clients, "client-0")
	pusher.mu.Unlock()

	assert.Equal(t, 4, pusher.GetConnectedClientsCount())
}

// TestRealtimePusher_SSEMessageFormat tests the SSE message format.
func TestRealtimePusher_SSEMessageFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(t, err)

	// Record some metrics
	collector.RecordSagaStarted("saga-1")
	collector.RecordSagaCompleted("saga-1", 100*time.Millisecond)

	config := &RealtimePusherConfig{
		UpdateInterval: 100 * time.Millisecond,
		BufferSize:     10,
	}
	pusher := NewRealtimePusher(collector, config)
	pusher.Start()
	defer pusher.Stop()

	// Create test request with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/api/metrics/stream", nil)
	req = req.WithContext(ctx)
	c.Request = req

	// Handle SSE connection
	done := make(chan bool)
	go func() {
		pusher.HandleSSE(c)
		done <- true
	}()

	// Wait for handler to complete
	<-done

	// Parse SSE messages
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	foundDataLine := false
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			foundDataLine = true
			// Extract JSON from "data: {...}"
			jsonData := strings.TrimPrefix(line, "data: ")

			// Parse JSON
			var update MetricsUpdate
			err := json.Unmarshal([]byte(jsonData), &update)
			require.NoError(t, err)

			// Verify structure
			assert.Equal(t, "metrics", update.Type)
			assert.NotNil(t, update.Data)
			assert.NotZero(t, update.Timestamp)
			break
		}
	}

	assert.True(t, foundDataLine, "Expected to find at least one data line in SSE stream")
}

// TestRealtimePusher_BroadcastLoop tests the broadcast loop.
func TestRealtimePusher_BroadcastLoop(t *testing.T) {
	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(t, err)

	config := &RealtimePusherConfig{
		UpdateInterval: 50 * time.Millisecond,
		BufferSize:     10,
	}
	pusher := NewRealtimePusher(collector, config)

	// Start pusher
	pusher.Start()

	// Create a test client to receive broadcasts
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &sseClient{
		id:       "test-client",
		channel:  make(chan *MetricsUpdate, config.BufferSize),
		context:  ctx,
		cancelFn: cancel,
	}

	pusher.mu.Lock()
	pusher.clients[client.id] = client
	pusher.mu.Unlock()

	// Record some metrics
	collector.RecordSagaStarted("saga-1")

	// Wait for at least one broadcast
	select {
	case update := <-client.channel:
		assert.Equal(t, "metrics", update.Type)
		assert.Equal(t, int64(1), update.Data.Total)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Did not receive broadcast in time")
	}

	// Stop pusher
	pusher.Stop()
}

// BenchmarkRealtimePusher_Broadcast benchmarks the broadcast operation.
func BenchmarkRealtimePusher_Broadcast(b *testing.B) {
	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(b, err)

	pusher := NewRealtimePusher(collector, nil)

	// Create multiple clients
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		client := &sseClient{
			id:       fmt.Sprintf("client-%d", i),
			channel:  make(chan *MetricsUpdate, 10),
			context:  ctx,
			cancelFn: cancel,
		}
		pusher.clients[client.id] = client

		// Consume updates in background to prevent channel blocking
		go func(c *sseClient) {
			for {
				select {
				case <-c.channel:
					// Consume update
				case <-c.context.Done():
					return
				}
			}
		}(client)
	}

	update := &MetricsUpdate{
		Type: "metrics",
		Data: MetricsSummary{
			Total:     100,
			Active:    10,
			Completed: 90,
		},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pusher.broadcast(update)
	}
}

// TestSSEConnectionLifecycle tests the full lifecycle of an SSE connection.
func TestSSEConnectionLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := prometheus.NewRegistry()
	collectorConfig := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(collectorConfig)
	require.NoError(t, err)

	config := &RealtimePusherConfig{
		UpdateInterval: 50 * time.Millisecond,
		BufferSize:     10,
	}
	pusher := NewRealtimePusher(collector, config)
	pusher.Start()
	defer pusher.Stop()

	// Initially no clients
	assert.Equal(t, 0, pusher.GetConnectedClientsCount())

	// Connect a client
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/api/metrics/stream", nil)
	req = req.WithContext(ctx)
	c.Request = req

	done := make(chan bool)
	go func() {
		pusher.HandleSSE(c)
		done <- true
	}()

	// Wait for client to connect
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, pusher.GetConnectedClientsCount())

	// Record metrics
	collector.RecordSagaStarted("saga-1")

	// Wait for handler to complete
	<-done

	// Verify client disconnected
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, pusher.GetConnectedClientsCount())

	// Verify we received data
	body := w.Body.String()
	assert.Contains(t, body, "data:")
}

// parseSSEMessage is a helper function to parse SSE messages from response body.
func parseSSEMessage(t *testing.T, body string) []MetricsUpdate {
	var updates []MetricsUpdate
	scanner := bufio.NewScanner(strings.NewReader(body))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")
			var update MetricsUpdate
			err := json.Unmarshal([]byte(jsonData), &update)
			if err != nil {
				t.Logf("Failed to parse SSE message: %v", err)
				continue
			}
			updates = append(updates, update)
		}
	}

	return updates
}
