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
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

// SentryManager manages Sentry initialization, configuration, and lifecycle
type SentryManager struct {
	config      *SentryConfig
	isEnabled   bool
	initialized bool
	mu          sync.RWMutex
}

// NewSentryManager creates a new SentryManager with the provided configuration
func NewSentryManager(config *SentryConfig) *SentryManager {
	if config == nil {
		config = &SentryConfig{Enabled: false}
	}

	return &SentryManager{
		config:    config,
		isEnabled: config.Enabled,
	}
}

// Initialize initializes Sentry with the configured options
func (sm *SentryManager) Initialize(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isEnabled {
		return nil
	}

	if sm.initialized {
		return fmt.Errorf("sentry manager already initialized")
	}

	sentryConfig := sentry.ClientOptions{
		Dsn:              sm.config.DSN,
		Environment:      sm.config.Environment,
		Release:          sm.config.Release,
		SampleRate:       sm.config.SampleRate,
		TracesSampleRate: sm.config.TracesSampleRate,
		AttachStacktrace: sm.config.AttachStacktrace,
		EnableTracing:    sm.config.EnableTracing,
		Debug:            sm.config.Debug,
		ServerName:       sm.config.ServerName,
		MaxBreadcrumbs:   sm.config.MaxBreadcrumbs,
		BeforeSend:       sm.createBeforeSendHook(),
	}

	// Add tags
	if sm.config.Tags != nil && len(sm.config.Tags) > 0 {
		sentryConfig.Tags = make(map[string]string)
		for k, v := range sm.config.Tags {
			sentryConfig.Tags[k] = v
		}
	}

	// Initialize Sentry client
	err := sentry.Init(sentryConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	sm.initialized = true
	return nil
}

// Close closes the Sentry client and flushes any pending events
func (sm *SentryManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.initialized {
		return nil
	}

	// Flush all pending events with timeout
	sentry.Flush(5 * time.Second)
	sm.initialized = false
	return nil
}

// IsEnabled returns true if Sentry is enabled
func (sm *SentryManager) IsEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.isEnabled
}

// IsInitialized returns true if Sentry has been initialized
func (sm *SentryManager) IsInitialized() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.initialized
}

// CaptureException captures an exception and sends it to Sentry
func (sm *SentryManager) CaptureException(exception error) *sentry.EventID {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return nil
	}

	eventID := sentry.CaptureException(exception)
	return eventID
}

// CaptureMessage captures a message and sends it to Sentry
func (sm *SentryManager) CaptureMessage(message string, level sentry.Level) *sentry.EventID {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return nil
	}

	eventID := sentry.CaptureMessage(message)
	return eventID
}

// CaptureEvent captures a custom event and sends it to Sentry
func (sm *SentryManager) CaptureEvent(event *sentry.Event) *sentry.EventID {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return nil
	}

	eventID := sentry.CaptureEvent(event)
	return eventID
}

// AddBreadcrumb adds a breadcrumb to the current scope
func (sm *SentryManager) AddBreadcrumb(breadcrumb *sentry.Breadcrumb) {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return
	}

	sentry.AddBreadcrumb(breadcrumb)
}

// ConfigureScope configures the global Sentry scope with additional context
func (sm *SentryManager) ConfigureScope(configureFunc func(scope *sentry.Scope)) {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return
	}

	sentry.ConfigureScope(configureFunc)
}

// WithScope executes a function with a new Sentry scope
func (sm *SentryManager) WithScope(scopeFunc func(scope *sentry.Scope)) {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return
	}

	sentry.WithScope(scopeFunc)
}

// GetHubFromContext gets the Sentry hub from context
func (sm *SentryManager) GetHubFromContext(ctx context.Context) *sentry.Hub {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return nil
	}

	return sentry.GetHubFromContext(ctx)
}

// ShouldCaptureHTTPError determines if an HTTP error should be captured based on configuration
func (sm *SentryManager) ShouldCaptureHTTPError(statusCode int) bool {
	if !sm.IsEnabled() {
		return false
	}

	// Check if status code should be ignored
	if slices.Contains(sm.config.HTTPIgnoreStatusCode, statusCode) {
		return false
	}

	// Only capture server errors (5xx) and some client errors
	return statusCode >= 500 || (statusCode >= 400 && statusCode < 500 && !slices.Contains(sm.config.HTTPIgnoreStatusCode, statusCode))
}

// ShouldCaptureHTTPPath determines if an HTTP path should be captured
func (sm *SentryManager) ShouldCaptureHTTPPath(path string) bool {
	if !sm.IsEnabled() {
		return true
	}

	// Check if path should be ignored
	for _, ignorePath := range sm.config.HTTPIgnorePaths {
		if strings.Contains(path, ignorePath) {
			return false
		}
	}

	return true
}

// ShouldCaptureError determines if an error should be captured based on configuration
func (sm *SentryManager) ShouldCaptureError(err error) bool {
	if !sm.IsEnabled() || err == nil {
		return false
	}

	errorMsg := err.Error()
	for _, ignorePattern := range sm.config.IgnoreErrors {
		if strings.Contains(errorMsg, ignorePattern) {
			return false
		}
	}

	return true
}

// createBeforeSendHook creates a before send hook if configured
func (sm *SentryManager) createBeforeSendHook() sentry.EventProcessor {
	if !sm.config.BeforeSend {
		return nil
	}

	return func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		// Add server metadata
		if event.ServerName == "" && sm.config.ServerName != "" {
			event.ServerName = sm.config.ServerName
		}

		// Add environment tags if not present
		if event.Environment == "" && sm.config.Environment != "" {
			event.Environment = sm.config.Environment
		}

		// Filter out ignored errors
		if event.Exception != nil && len(event.Exception) > 0 {
			for _, ex := range event.Exception {
				if ex.Value != "" && !sm.ShouldCaptureError(fmt.Errorf("%s", ex.Value)) {
					return nil
				}
			}
		}

		return event
	}
}

// RecoverWithSentry recovers from panics and sends them to Sentry
func (sm *SentryManager) RecoverWithSentry() {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return
	}

	if !sm.config.CapturePanics {
		return
	}

	if err := recover(); err != nil {
		sentry.CurrentHub().Recover(err)
		sentry.Flush(2 * time.Second)
		panic(err) // Re-panic to maintain normal behavior
	}
}

// AddContextTags adds common context tags to Sentry scope
func (sm *SentryManager) AddContextTags(serviceName, version, environment string) {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return
	}

	sm.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("service", serviceName)
		if version != "" {
			scope.SetTag("version", version)
		}
		if environment != "" {
			scope.SetTag("environment", environment)
		}
	})
}

// SetUser sets user information in the current Sentry scope
func (sm *SentryManager) SetUser(user *sentry.User) {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(*user)
	})
}

// SetContext sets custom context in the current Sentry scope
func (sm *SentryManager) SetContext(key string, context map[string]interface{}) {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext(key, context)
	})
}

// Flush flushes any pending Sentry events
func (sm *SentryManager) Flush(timeout time.Duration) bool {
	if !sm.IsEnabled() || !sm.IsInitialized() {
		return true
	}

	return sentry.Flush(timeout)
}
