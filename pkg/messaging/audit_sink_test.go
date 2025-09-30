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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSink(t *testing.T) {
	t.Run("Create and write to file sink", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "audit.log")

		config := &SinkConfig{
			Type:     SinkTypeFile,
			FilePath: filePath,
		}

		sink, err := NewFileSink(config)
		require.NoError(t, err)
		require.NotNil(t, sink)
		defer sink.Close()

		event := &AuditEvent{
			Type:      AuditEventMessagePublished,
			Timestamp: time.Now(),
			MessageID: "test-msg-123",
			Topic:     "test-topic",
			Operation: "publish",
			Success:   true,
		}

		err = sink.Write(context.Background(), event)
		require.NoError(t, err)

		err = sink.Flush()
		require.NoError(t, err)

		// Verify file was created and contains data
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "test-msg-123")
		assert.Contains(t, string(data), "test-topic")
	})

	t.Run("File rotation", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "audit.log")

		config := &SinkConfig{
			Type:        SinkTypeFile,
			FilePath:    filePath,
			MaxFileSize: 100, // Small size to trigger rotation
			MaxBackups:  3,
		}

		sink, err := NewFileSink(config)
		require.NoError(t, err)
		require.NotNil(t, sink)
		defer sink.Close()

		// Write enough events to trigger rotation
		for i := 0; i < 10; i++ {
			event := &AuditEvent{
				Type:      AuditEventMessagePublished,
				Timestamp: time.Now(),
				MessageID: "test-msg-" + strings.Repeat("X", 20),
				Topic:     "test-topic",
				Operation: "publish",
				Success:   true,
			}
			err = sink.Write(context.Background(), event)
			require.NoError(t, err)
		}

		// Check that backup files were created
		backupPath := filePath + ".1"
		_, err = os.Stat(backupPath)
		assert.NoError(t, err, "Backup file should exist")
	})
}

func TestJSONSink(t *testing.T) {
	t.Run("Write JSON events", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "audit.json")

		config := &SinkConfig{
			Type:     SinkTypeJSON,
			FilePath: filePath,
		}

		sink, err := NewJSONSink(config)
		require.NoError(t, err)
		require.NotNil(t, sink)
		defer sink.Close()

		event := &AuditEvent{
			Type:      AuditEventMessagePublished,
			Timestamp: time.Now(),
			MessageID: "test-msg-123",
			Topic:     "test-topic",
			Operation: "publish",
			Success:   true,
			Duration:  10.5,
		}

		err = sink.Write(context.Background(), event)
		require.NoError(t, err)

		err = sink.Flush()
		require.NoError(t, err)

		// Read and parse JSON
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		var parsedEvent AuditEvent
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		err = json.Unmarshal([]byte(lines[0]), &parsedEvent)
		require.NoError(t, err)

		assert.Equal(t, "test-msg-123", parsedEvent.MessageID)
		assert.Equal(t, "test-topic", parsedEvent.Topic)
		assert.Equal(t, AuditEventMessagePublished, parsedEvent.Type)
	})
}

func TestWebhookSink(t *testing.T) {
	t.Run("Send to webhook", func(t *testing.T) {
		// Create test server
		received := false
		var receivedEvent AuditEvent

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received = true
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			err := json.NewDecoder(r.Body).Decode(&receivedEvent)
			require.NoError(t, err)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := &SinkConfig{
			Type:       SinkTypeWebhook,
			WebhookURL: server.URL,
		}

		sink, err := NewWebhookSink(config)
		require.NoError(t, err)
		require.NotNil(t, sink)
		defer sink.Close()

		event := &AuditEvent{
			Type:      AuditEventMessagePublished,
			Timestamp: time.Now(),
			MessageID: "test-msg-123",
			Topic:     "test-topic",
			Operation: "publish",
			Success:   true,
		}

		err = sink.Write(context.Background(), event)
		require.NoError(t, err)

		assert.True(t, received, "Webhook should receive the event")
		assert.Equal(t, "test-msg-123", receivedEvent.MessageID)
	})

	t.Run("Webhook with custom headers", func(t *testing.T) {
		receivedHeaders := make(http.Header)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := &SinkConfig{
			Type:       SinkTypeWebhook,
			WebhookURL: server.URL,
			WebhookHeaders: map[string]string{
				"X-Custom-Header": "test-value",
				"Authorization":   "Bearer token",
			},
		}

		sink, err := NewWebhookSink(config)
		require.NoError(t, err)
		defer sink.Close()

		event := &AuditEvent{
			Type:      AuditEventMessagePublished,
			Timestamp: time.Now(),
			MessageID: "test-msg-123",
			Operation: "publish",
			Success:   true,
		}

		err = sink.Write(context.Background(), event)
		require.NoError(t, err)

		assert.Equal(t, "test-value", receivedHeaders.Get("X-Custom-Header"))
		assert.Equal(t, "Bearer token", receivedHeaders.Get("Authorization"))
	})

	t.Run("Webhook error handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := &SinkConfig{
			Type:       SinkTypeWebhook,
			WebhookURL: server.URL,
		}

		sink, err := NewWebhookSink(config)
		require.NoError(t, err)
		defer sink.Close()

		event := &AuditEvent{
			Type:      AuditEventMessagePublished,
			Timestamp: time.Now(),
			MessageID: "test-msg-123",
			Operation: "publish",
			Success:   true,
		}

		err = sink.Write(context.Background(), event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func TestMultiSink(t *testing.T) {
	t.Run("Write to multiple sinks", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath1 := filepath.Join(tmpDir, "audit1.log")
		filePath2 := filepath.Join(tmpDir, "audit2.json")

		sink1, err := NewFileSink(&SinkConfig{
			Type:     SinkTypeFile,
			FilePath: filePath1,
		})
		require.NoError(t, err)

		sink2, err := NewJSONSink(&SinkConfig{
			Type:     SinkTypeJSON,
			FilePath: filePath2,
		})
		require.NoError(t, err)

		multiSink := NewMultiSink(sink1, sink2)
		defer multiSink.Close()

		event := &AuditEvent{
			Type:      AuditEventMessagePublished,
			Timestamp: time.Now(),
			MessageID: "test-msg-123",
			Topic:     "test-topic",
			Operation: "publish",
			Success:   true,
		}

		err = multiSink.Write(context.Background(), event)
		require.NoError(t, err)

		err = multiSink.Flush()
		require.NoError(t, err)

		// Verify both files were written
		data1, err := os.ReadFile(filePath1)
		require.NoError(t, err)
		assert.Contains(t, string(data1), "test-msg-123")

		data2, err := os.ReadFile(filePath2)
		require.NoError(t, err)
		assert.Contains(t, string(data2), "test-msg-123")
	})

	t.Run("Add sink dynamically", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "audit.log")

		multiSink := NewMultiSink()

		sink, err := NewFileSink(&SinkConfig{
			Type:     SinkTypeFile,
			FilePath: filePath,
		})
		require.NoError(t, err)

		multiSink.AddSink(sink)
		defer multiSink.Close()

		event := &AuditEvent{
			Type:      AuditEventMessagePublished,
			Timestamp: time.Now(),
			MessageID: "test-msg-123",
			Operation: "publish",
			Success:   true,
		}

		err = multiSink.Write(context.Background(), event)
		require.NoError(t, err)

		err = multiSink.Flush()
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "test-msg-123")
	})
}

func TestCreateSink(t *testing.T) {
	tests := []struct {
		name        string
		config      *SinkConfig
		expectError bool
	}{
		{
			name: "File sink",
			config: &SinkConfig{
				Type:     SinkTypeFile,
				FilePath: filepath.Join(t.TempDir(), "audit.log"),
			},
			expectError: false,
		},
		{
			name: "JSON sink",
			config: &SinkConfig{
				Type:     SinkTypeJSON,
				FilePath: filepath.Join(t.TempDir(), "audit.json"),
			},
			expectError: false,
		},
		{
			name: "Unknown sink type",
			config: &SinkConfig{
				Type: "unknown",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink, err := CreateSink(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, sink)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sink)
				if sink != nil {
					sink.Close()
				}
			}
		})
	}
}
