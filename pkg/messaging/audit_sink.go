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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/syslog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditSink defines the interface for audit log output destinations
type AuditSink interface {
	// Write writes an audit event to the sink
	Write(ctx context.Context, event *AuditEvent) error

	// Flush flushes any buffered events
	Flush() error

	// Close closes the sink and releases resources
	Close() error

	// Name returns the name of the sink
	Name() string
}

// SinkType defines the type of audit sink
type SinkType string

const (
	// SinkTypeFile writes audit logs to a file
	SinkTypeFile SinkType = "file"
	// SinkTypeJSON writes audit logs as JSON to a file
	SinkTypeJSON SinkType = "json"
	// SinkTypeSyslog writes audit logs to syslog
	SinkTypeSyslog SinkType = "syslog"
	// SinkTypeWebhook sends audit logs to an HTTP endpoint
	SinkTypeWebhook SinkType = "webhook"
	// SinkTypeMulti combines multiple sinks
	SinkTypeMulti SinkType = "multi"
)

// SinkConfig defines configuration for an audit sink
type SinkConfig struct {
	// Type of sink
	Type SinkType `json:"type" yaml:"type"`

	// File path for file-based sinks
	FilePath string `json:"file_path,omitempty" yaml:"file_path,omitempty"`

	// MaxFileSize for file rotation (in bytes)
	MaxFileSize int64 `json:"max_file_size,omitempty" yaml:"max_file_size,omitempty" default:"104857600"` // 100MB

	// MaxBackups for file rotation
	MaxBackups int `json:"max_backups,omitempty" yaml:"max_backups,omitempty" default:"10"`

	// Syslog network and address (e.g., "tcp", "localhost:514")
	SyslogNetwork string `json:"syslog_network,omitempty" yaml:"syslog_network,omitempty"`
	SyslogAddress string `json:"syslog_address,omitempty" yaml:"syslog_address,omitempty"`
	SyslogTag     string `json:"syslog_tag,omitempty" yaml:"syslog_tag,omitempty" default:"swit-audit"`

	// Webhook URL for webhook sink
	WebhookURL     string            `json:"webhook_url,omitempty" yaml:"webhook_url,omitempty"`
	WebhookHeaders map[string]string `json:"webhook_headers,omitempty" yaml:"webhook_headers,omitempty"`
	WebhookTimeout time.Duration     `json:"webhook_timeout,omitempty" yaml:"webhook_timeout,omitempty" default:"10s"`

	// BufferSize for async writes
	BufferSize int `json:"buffer_size,omitempty" yaml:"buffer_size,omitempty" default:"100"`
}

// FileSink writes audit events to a file
type FileSink struct {
	config   *SinkConfig
	mu       sync.Mutex
	file     *os.File
	written  int64
	rotation bool
}

// NewFileSink creates a new file-based audit sink
func NewFileSink(config *SinkConfig) (*FileSink, error) {
	if config.FilePath == "" {
		return nil, fmt.Errorf("file path is required for file sink")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Open or create the file
	file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", config.FilePath, err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	sink := &FileSink{
		config:   config,
		file:     file,
		written:  info.Size(),
		rotation: config.MaxFileSize > 0,
	}

	return sink, nil
}

// Write writes an audit event to the file
func (s *FileSink) Write(ctx context.Context, event *AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Format the event as a log line
	line := s.formatEvent(event)
	n, err := s.file.WriteString(line + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	s.written += int64(n)

	// Check if rotation is needed
	if s.rotation && s.written >= s.config.MaxFileSize {
		if err := s.rotate(); err != nil {
			return fmt.Errorf("failed to rotate file: %w", err)
		}
	}

	return nil
}

// formatEvent formats an audit event as a log line
func (s *FileSink) formatEvent(event *AuditEvent) string {
	return fmt.Sprintf("[%s] type=%s operation=%s success=%t message_id=%s topic=%s duration_ms=%.2f",
		event.Timestamp.Format(time.RFC3339),
		event.Type,
		event.Operation,
		event.Success,
		event.MessageID,
		event.Topic,
		event.Duration,
	)
}

// rotate rotates the log file
func (s *FileSink) rotate() error {
	// Close current file
	if err := s.file.Close(); err != nil {
		return err
	}

	// Rotate existing backups
	for i := s.config.MaxBackups - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", s.config.FilePath, i)
		newPath := fmt.Sprintf("%s.%d", s.config.FilePath, i+1)
		os.Rename(oldPath, newPath) // Ignore errors for non-existent files
	}

	// Move current file to .1
	backupPath := fmt.Sprintf("%s.1", s.config.FilePath)
	if err := os.Rename(s.config.FilePath, backupPath); err != nil {
		return err
	}

	// Create new file
	file, err := os.OpenFile(s.config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	s.file = file
	s.written = 0

	return nil
}

// Flush flushes buffered data to disk
func (s *FileSink) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		return s.file.Sync()
	}
	return nil
}

// Close closes the file sink
func (s *FileSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

// Name returns the name of the sink
func (s *FileSink) Name() string {
	return fmt.Sprintf("file(%s)", s.config.FilePath)
}

// JSONSink writes audit events as JSON to a file
type JSONSink struct {
	config  *SinkConfig
	mu      sync.Mutex
	file    *os.File
	encoder *json.Encoder
}

// NewJSONSink creates a new JSON file sink
func NewJSONSink(config *SinkConfig) (*JSONSink, error) {
	if config.FilePath == "" {
		return nil, fmt.Errorf("file path is required for JSON sink")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Open or create the file
	file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", config.FilePath, err)
	}

	sink := &JSONSink{
		config:  config,
		file:    file,
		encoder: json.NewEncoder(file),
	}

	return sink, nil
}

// Write writes an audit event as JSON
func (s *JSONSink) Write(ctx context.Context, event *AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.encoder.Encode(event); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// Flush flushes buffered data to disk
func (s *JSONSink) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		return s.file.Sync()
	}
	return nil
}

// Close closes the JSON sink
func (s *JSONSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

// Name returns the name of the sink
func (s *JSONSink) Name() string {
	return fmt.Sprintf("json(%s)", s.config.FilePath)
}

// SyslogSink writes audit events to syslog
type SyslogSink struct {
	config *SinkConfig
	writer *syslog.Writer
	mu     sync.Mutex
}

// NewSyslogSink creates a new syslog sink
func NewSyslogSink(config *SinkConfig) (*SyslogSink, error) {
	tag := config.SyslogTag
	if tag == "" {
		tag = "swit-audit"
	}

	var writer *syslog.Writer
	var err error

	if config.SyslogNetwork != "" && config.SyslogAddress != "" {
		writer, err = syslog.Dial(config.SyslogNetwork, config.SyslogAddress, syslog.LOG_INFO|syslog.LOG_USER, tag)
	} else {
		writer, err = syslog.New(syslog.LOG_INFO|syslog.LOG_USER, tag)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to syslog: %w", err)
	}

	return &SyslogSink{
		config: config,
		writer: writer,
	}, nil
}

// Write writes an audit event to syslog
func (s *SyslogSink) Write(ctx context.Context, event *AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	message := fmt.Sprintf("type=%s operation=%s success=%t message_id=%s topic=%s",
		event.Type, event.Operation, event.Success, event.MessageID, event.Topic)

	if event.Success {
		return s.writer.Info(message)
	}
	return s.writer.Err(message)
}

// Flush is a no-op for syslog
func (s *SyslogSink) Flush() error {
	return nil
}

// Close closes the syslog connection
func (s *SyslogSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writer != nil {
		err := s.writer.Close()
		s.writer = nil
		return err
	}
	return nil
}

// Name returns the name of the sink
func (s *SyslogSink) Name() string {
	return fmt.Sprintf("syslog(%s/%s)", s.config.SyslogNetwork, s.config.SyslogAddress)
}

// WebhookSink sends audit events to an HTTP endpoint
type WebhookSink struct {
	config *SinkConfig
	client *http.Client
	mu     sync.Mutex
}

// NewWebhookSink creates a new webhook sink
func NewWebhookSink(config *SinkConfig) (*WebhookSink, error) {
	if config.WebhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required for webhook sink")
	}

	timeout := config.WebhookTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &WebhookSink{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Write sends an audit event to the webhook endpoint
func (s *WebhookSink) Write(ctx context.Context, event *AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.WebhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range s.config.WebhookHeaders {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Flush is a no-op for webhook
func (s *WebhookSink) Flush() error {
	return nil
}

// Close closes the webhook sink
func (s *WebhookSink) Close() error {
	// Close idle connections
	s.client.CloseIdleConnections()
	return nil
}

// Name returns the name of the sink
func (s *WebhookSink) Name() string {
	return fmt.Sprintf("webhook(%s)", s.config.WebhookURL)
}

// MultiSink combines multiple sinks
type MultiSink struct {
	sinks []AuditSink
	mu    sync.RWMutex
}

// NewMultiSink creates a new multi-sink
func NewMultiSink(sinks ...AuditSink) *MultiSink {
	return &MultiSink{
		sinks: sinks,
	}
}

// Write writes to all sinks
func (s *MultiSink) Write(ctx context.Context, event *AuditEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var errs []error
	for _, sink := range s.sinks {
		if err := sink.Write(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", sink.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multi-sink errors: %v", errs)
	}

	return nil
}

// Flush flushes all sinks
func (s *MultiSink) Flush() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var errs []error
	for _, sink := range s.sinks {
		if err := sink.Flush(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", sink.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multi-sink flush errors: %v", errs)
	}

	return nil
}

// Close closes all sinks
func (s *MultiSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error
	for _, sink := range s.sinks {
		if err := sink.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", sink.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multi-sink close errors: %v", errs)
	}

	return nil
}

// Name returns the name of the sink
func (s *MultiSink) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, len(s.sinks))
	for i, sink := range s.sinks {
		names[i] = sink.Name()
	}
	return fmt.Sprintf("multi(%d sinks)", len(names))
}

// AddSink adds a sink to the multi-sink
func (s *MultiSink) AddSink(sink AuditSink) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sinks = append(s.sinks, sink)
}

// CreateSink creates a sink based on configuration
func CreateSink(config *SinkConfig) (AuditSink, error) {
	switch config.Type {
	case SinkTypeFile:
		return NewFileSink(config)
	case SinkTypeJSON:
		return NewJSONSink(config)
	case SinkTypeSyslog:
		return NewSyslogSink(config)
	case SinkTypeWebhook:
		return NewWebhookSink(config)
	default:
		return nil, fmt.Errorf("unknown sink type: %s", config.Type)
	}
}
