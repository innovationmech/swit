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

package server

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// SentryManager handles Sentry SDK initialization, configuration, and lifecycle management
type SentryManager struct {
	config    *SentryConfig
	logger    *zap.Logger
	hub       *sentry.Hub
	isEnabled bool
}

// NewSentryManager creates a new Sentry manager instance
func NewSentryManager(config *SentryConfig, logger *zap.Logger) *SentryManager {
	return &SentryManager{
		config:    config,
		logger:    logger,
		isEnabled: config.Enabled,
	}
}

// Initialize initializes the Sentry SDK with the provided configuration
func (sm *SentryManager) Initialize() error {
	if !sm.config.Enabled {
		sm.logger.Info("Sentry monitoring is disabled")
		return nil
	}

	// Prepare Sentry options
	options := sentry.ClientOptions{
		Dsn:              sm.config.DSN,
		Debug:            sm.config.Debug,
		Environment:      sm.config.Environment,
		Release:          sm.config.Release,
		SampleRate:       sm.config.SampleRate,
		TracesSampleRate: sm.config.TracesSampleRate,
		AttachStacktrace: sm.config.AttachStacktrace,
		ServerName:       sm.config.ServerName,
		ProfilesSampleRate: sm.config.ProfilesSampleRate,
		EnableTracing:    sm.config.EnableTracing,
	}

	// Add custom tags if provided
	if len(sm.config.Tags) > 0 {
		options.BeforeSend = func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			for key, value := range sm.config.Tags {
				event.Tags[key] = value
			}
			return event
		}
	}

	// Initialize Sentry client
	err := sentry.Init(options)
	if err != nil {
		return fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	// Create a new hub for this manager
	sm.hub = sentry.CurrentHub().Clone()

	sm.logger.Info("Sentry monitoring initialized successfully",
		zap.String("environment", sm.config.Environment),
		zap.String("release", sm.config.Release),
		zap.Float64("sample_rate", sm.config.SampleRate),
		zap.Bool("tracing_enabled", sm.config.EnableTracing),
	)

	return nil
}

// Shutdown flushes pending events and closes the Sentry client
func (sm *SentryManager) Shutdown(ctx context.Context) error {
	if !sm.isEnabled {
		return nil
	}

	flushTimeout := sm.config.FlushTimeout
	if flushTimeout == 0 {
		flushTimeout = 2 * time.Second
	}

	sm.logger.Info("Flushing Sentry events before shutdown", zap.Duration("timeout", flushTimeout))

	// Flush pending events with timeout
	if sentry.Flush(flushTimeout) {
		sm.logger.Info("Sentry events flushed successfully")
	} else {
		sm.logger.Warn("Sentry flush timed out, some events may be lost")
	}

	return nil
}

// CaptureError captures an error and sends it to Sentry with optional context
func (sm *SentryManager) CaptureError(err error, tags map[string]string, extra map[string]interface{}) *sentry.EventID {
	if !sm.isEnabled || err == nil {
		return nil
	}

	// Configure scope with additional context
	eventID := sm.hub.WithScope(func(scope *sentry.Scope) {
		// Add tags
		for key, value := range tags {
			scope.SetTag(key, value)
		}

		// Add extra data
		for key, value := range extra {
			scope.SetExtra(key, value)
		}

		// Capture the error
		sm.hub.CaptureException(err)
	})

	return eventID
}

// CaptureMessage captures a message and sends it to Sentry
func (sm *SentryManager) CaptureMessage(message string, level sentry.Level, tags map[string]string, extra map[string]interface{}) *sentry.EventID {
	if !sm.isEnabled {
		return nil
	}

	// Configure scope with additional context
	eventID := sm.hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)

		// Add tags
		for key, value := range tags {
			scope.SetTag(key, value)
		}

		// Add extra data
		for key, value := range extra {
			scope.SetExtra(key, value)
		}

		// Capture the message
		sm.hub.CaptureMessage(message)
	})

	return eventID
}

// AddBreadcrumb adds a breadcrumb to the current scope
func (sm *SentryManager) AddBreadcrumb(message string, category string, level sentry.Level, data map[string]interface{}) {
	if !sm.isEnabled {
		return
	}

	breadcrumb := &sentry.Breadcrumb{
		Message:   message,
		Category:  category,
		Level:     level,
		Data:      data,
		Timestamp: time.Now(),
	}

	sm.hub.AddBreadcrumb(breadcrumb, nil)
}

// ConfigureScope allows configuration of the current Sentry scope
func (sm *SentryManager) ConfigureScope(f func(*sentry.Scope)) {
	if !sm.isEnabled {
		return
	}

	sm.hub.ConfigureScope(f)
}

// GetHub returns the Sentry hub for advanced usage
func (sm *SentryManager) GetHub() *sentry.Hub {
	if !sm.isEnabled {
		return nil
	}
	return sm.hub
}

// IsEnabled returns whether Sentry monitoring is enabled
func (sm *SentryManager) IsEnabled() bool {
	return sm.isEnabled
}

// StartTransaction starts a new Sentry transaction for performance monitoring
func (sm *SentryManager) StartTransaction(ctx context.Context, name string, op string) *sentry.Span {
	if !sm.isEnabled || !sm.config.EnableTracing {
		return nil
	}

	transaction := sentry.StartTransaction(ctx, name)
	transaction.Op = op
	return transaction
}

// RecoverWithSentry captures panics and sends them to Sentry
func (sm *SentryManager) RecoverWithSentry() {
	if !sm.isEnabled {
		return
	}

	if r := recover(); r != nil {
		var err error
		switch x := r.(type) {
		case string:
			err = fmt.Errorf("panic: %s", x)
		case error:
			err = fmt.Errorf("panic: %w", x)
		default:
			err = fmt.Errorf("panic: %v", x)
		}

		sm.CaptureError(err, map[string]string{"type": "panic"}, nil)
		
		// Re-panic to maintain normal panic behavior
		panic(r)
	}
}