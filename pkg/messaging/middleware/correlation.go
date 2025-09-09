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

package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// CorrelationContext holds correlation information in the context.
type CorrelationContext struct {
	ID        string
	TraceID   string
	SpanID    string
	RequestID string
	SessionID string
	UserID    string
	TenantID  string
}

// CorrelationIDGenerator defines the interface for generating correlation IDs.
type CorrelationIDGenerator interface {
	Generate() string
	GenerateWithPrefix(prefix string) string
	Validate(id string) bool
}

// UUIDGenerator generates UUID-style correlation IDs.
type UUIDGenerator struct {
	prefix string
}

// NewUUIDGenerator creates a new UUID-style correlation ID generator.
func NewUUIDGenerator(prefix string) *UUIDGenerator {
	return &UUIDGenerator{prefix: prefix}
}

// Generate generates a new correlation ID.
func (u *UUIDGenerator) Generate() string {
	return u.GenerateWithPrefix(u.prefix)
}

// GenerateWithPrefix generates a new correlation ID with a specific prefix.
func (u *UUIDGenerator) GenerateWithPrefix(prefix string) string {
	// Generate 16 random bytes
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based generation
		timestamp := time.Now().UnixNano()
		return fmt.Sprintf("%s%016x", prefix, timestamp)
	}

	// Format as UUID-like string
	id := fmt.Sprintf("%s%08x-%04x-%04x-%04x-%012x",
		prefix,
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16])

	return id
}

// Validate validates a correlation ID format.
func (u *UUIDGenerator) Validate(id string) bool {
	if len(u.prefix) > 0 && !strings.HasPrefix(id, u.prefix) {
		return false
	}

	// Remove prefix for validation
	cleanID := strings.TrimPrefix(id, u.prefix)

	// Simple UUID format validation (8-4-4-4-12)
	parts := strings.Split(cleanID, "-")
	if len(parts) != 5 {
		return false
	}

	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			return false
		}
		// Check if it's valid hex
		if _, err := hex.DecodeString(part); err != nil {
			return false
		}
	}

	return true
}

// TimestampGenerator generates timestamp-based correlation IDs.
type TimestampGenerator struct {
	prefix string
}

// NewTimestampGenerator creates a new timestamp-based correlation ID generator.
func NewTimestampGenerator(prefix string) *TimestampGenerator {
	return &TimestampGenerator{prefix: prefix}
}

// Generate generates a new timestamp-based correlation ID.
func (t *TimestampGenerator) Generate() string {
	return t.GenerateWithPrefix(t.prefix)
}

// GenerateWithPrefix generates a new correlation ID with a specific prefix.
func (t *TimestampGenerator) GenerateWithPrefix(prefix string) string {
	timestamp := time.Now().UnixNano()

	// Add some randomness to avoid collisions
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		randomBytes = []byte{0, 0, 0, 0}
	}

	return fmt.Sprintf("%s%016x%08x", prefix, timestamp, randomBytes)
}

// Validate validates a timestamp-based correlation ID.
func (t *TimestampGenerator) Validate(id string) bool {
	if len(t.prefix) > 0 && !strings.HasPrefix(id, t.prefix) {
		return false
	}

	cleanID := strings.TrimPrefix(id, t.prefix)

	// Should be 24 hex characters (16 for timestamp + 8 for random)
	if len(cleanID) != 24 {
		return false
	}

	// Check if it's valid hex
	if _, err := hex.DecodeString(cleanID); err != nil {
		return false
	}

	return true
}

// CorrelationContextKey is the context key for correlation information.
type CorrelationContextKey struct{}

// CorrelationExtractor defines how to extract correlation information from messages.
type CorrelationExtractor interface {
	Extract(message *messaging.Message) *CorrelationContext
	Inject(message *messaging.Message, correlation *CorrelationContext)
}

// HeaderCorrelationExtractor extracts correlation information from message headers.
type HeaderCorrelationExtractor struct {
	CorrelationIDHeader string
	TraceIDHeader       string
	SpanIDHeader        string
	RequestIDHeader     string
	SessionIDHeader     string
	UserIDHeader        string
	TenantIDHeader      string
}

// NewHeaderCorrelationExtractor creates a new header-based correlation extractor.
func NewHeaderCorrelationExtractor() *HeaderCorrelationExtractor {
	return &HeaderCorrelationExtractor{
		CorrelationIDHeader: "x-correlation-id",
		TraceIDHeader:       "x-trace-id",
		SpanIDHeader:        "x-span-id",
		RequestIDHeader:     "x-request-id",
		SessionIDHeader:     "x-session-id",
		UserIDHeader:        "x-user-id",
		TenantIDHeader:      "x-tenant-id",
	}
}

// Extract extracts correlation information from message headers.
func (hce *HeaderCorrelationExtractor) Extract(message *messaging.Message) *CorrelationContext {
	return &CorrelationContext{
		ID:        message.Headers[hce.CorrelationIDHeader],
		TraceID:   message.Headers[hce.TraceIDHeader],
		SpanID:    message.Headers[hce.SpanIDHeader],
		RequestID: message.Headers[hce.RequestIDHeader],
		SessionID: message.Headers[hce.SessionIDHeader],
		UserID:    message.Headers[hce.UserIDHeader],
		TenantID:  message.Headers[hce.TenantIDHeader],
	}
}

// Inject injects correlation information into message headers.
func (hce *HeaderCorrelationExtractor) Inject(message *messaging.Message, correlation *CorrelationContext) {
	if message.Headers == nil {
		message.Headers = make(map[string]string)
	}

	if correlation.ID != "" {
		message.Headers[hce.CorrelationIDHeader] = correlation.ID
	}
	if correlation.TraceID != "" {
		message.Headers[hce.TraceIDHeader] = correlation.TraceID
	}
	if correlation.SpanID != "" {
		message.Headers[hce.SpanIDHeader] = correlation.SpanID
	}
	if correlation.RequestID != "" {
		message.Headers[hce.RequestIDHeader] = correlation.RequestID
	}
	if correlation.SessionID != "" {
		message.Headers[hce.SessionIDHeader] = correlation.SessionID
	}
	if correlation.UserID != "" {
		message.Headers[hce.UserIDHeader] = correlation.UserID
	}
	if correlation.TenantID != "" {
		message.Headers[hce.TenantIDHeader] = correlation.TenantID
	}
}

// CorrelationConfig holds configuration for the correlation middleware.
type CorrelationConfig struct {
	Generator          CorrelationIDGenerator
	Extractor          CorrelationExtractor
	GenerateIfMissing  bool
	PropagateToContext bool
	ValidateExisting   bool
	OverrideExisting   bool
	IncludeInMessage   bool
}

// DefaultCorrelationConfig returns a default correlation configuration.
func DefaultCorrelationConfig() *CorrelationConfig {
	return &CorrelationConfig{
		Generator:          NewUUIDGenerator("corr-"),
		Extractor:          NewHeaderCorrelationExtractor(),
		GenerateIfMissing:  true,
		PropagateToContext: true,
		ValidateExisting:   false,
		OverrideExisting:   false,
		IncludeInMessage:   true,
	}
}

// CorrelationMiddleware provides correlation ID management for message processing.
type CorrelationMiddleware struct {
	config *CorrelationConfig
}

// NewCorrelationMiddleware creates a new correlation middleware.
func NewCorrelationMiddleware(config *CorrelationConfig) *CorrelationMiddleware {
	if config == nil {
		config = DefaultCorrelationConfig()
	}
	if config.Generator == nil {
		config.Generator = NewUUIDGenerator("corr-")
	}
	if config.Extractor == nil {
		config.Extractor = NewHeaderCorrelationExtractor()
	}
	return &CorrelationMiddleware{
		config: config,
	}
}

// Name returns the middleware name.
func (cm *CorrelationMiddleware) Name() string {
	return "correlation"
}

// Wrap wraps a handler with correlation ID functionality.
func (cm *CorrelationMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	return messaging.MessageHandlerFunc(func(ctx context.Context, message *messaging.Message) error {
		// Extract existing correlation information
		correlation := cm.config.Extractor.Extract(message)

		// Generate correlation ID if missing or override existing
		if (correlation.ID == "" && cm.config.GenerateIfMissing) || cm.config.OverrideExisting {
			correlation.ID = cm.config.Generator.Generate()
		}

		// Validate existing correlation ID if configured
		if cm.config.ValidateExisting && correlation.ID != "" {
			if !cm.config.Generator.Validate(correlation.ID) {
				// Invalid correlation ID, generate a new one
				correlation.ID = cm.config.Generator.Generate()
			}
		}

		// Ensure message has correlation ID if required
		if cm.config.IncludeInMessage && correlation.ID != "" {
			if message.CorrelationID == "" || cm.config.OverrideExisting {
				message.CorrelationID = correlation.ID
			}
		}

		// Inject correlation information back into message
		cm.config.Extractor.Inject(message, correlation)

		// Propagate correlation to context if configured
		if cm.config.PropagateToContext {
			ctx = context.WithValue(ctx, CorrelationContextKey{}, correlation)
		}

		return next.Handle(ctx, message)
	})
}

// GetCorrelationFromContext extracts correlation information from context.
func GetCorrelationFromContext(ctx context.Context) *CorrelationContext {
	if correlation, ok := ctx.Value(CorrelationContextKey{}).(*CorrelationContext); ok {
		return correlation
	}
	return nil
}

// WithCorrelation adds correlation information to context.
func WithCorrelation(ctx context.Context, correlation *CorrelationContext) context.Context {
	return context.WithValue(ctx, CorrelationContextKey{}, correlation)
}

// RequestTrackingMiddleware provides enhanced request tracking with correlation chains.
type RequestTrackingMiddleware struct {
	generator       CorrelationIDGenerator
	extractor       CorrelationExtractor
	trackingEnabled bool
	chainTracking   bool
}

// NewRequestTrackingMiddleware creates a new request tracking middleware.
func NewRequestTrackingMiddleware(generator CorrelationIDGenerator, extractor CorrelationExtractor) *RequestTrackingMiddleware {
	if generator == nil {
		generator = NewUUIDGenerator("req-")
	}
	if extractor == nil {
		extractor = NewHeaderCorrelationExtractor()
	}

	return &RequestTrackingMiddleware{
		generator:       generator,
		extractor:       extractor,
		trackingEnabled: true,
		chainTracking:   true,
	}
}

// Name returns the middleware name.
func (rtm *RequestTrackingMiddleware) Name() string {
	return "request-tracking"
}

// Wrap wraps a handler with request tracking functionality.
func (rtm *RequestTrackingMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	return messaging.MessageHandlerFunc(func(ctx context.Context, message *messaging.Message) error {
		if !rtm.trackingEnabled {
			return next.Handle(ctx, message)
		}

		// Extract existing correlation information
		correlation := rtm.extractor.Extract(message)

		// Create a new request ID for this processing
		requestID := rtm.generator.GenerateWithPrefix("req-")

		// If we have an existing correlation ID, create a chain
		if rtm.chainTracking && correlation.ID != "" {
			correlation.RequestID = requestID
		} else {
			// Use request ID as correlation ID if none exists
			if correlation.ID == "" {
				correlation.ID = requestID
			}
			correlation.RequestID = requestID
		}

		// Add processing timestamp
		if message.Headers == nil {
			message.Headers = make(map[string]string)
		}
		message.Headers["x-processing-timestamp"] = time.Now().Format(time.RFC3339Nano)
		message.Headers["x-processing-request-id"] = requestID

		// Inject updated correlation information
		rtm.extractor.Inject(message, correlation)

		// Propagate to context
		ctx = WithCorrelation(ctx, correlation)

		return next.Handle(ctx, message)
	})
}

// TenantIsolationMiddleware provides tenant-aware correlation tracking.
type TenantIsolationMiddleware struct {
	extractor       CorrelationExtractor
	defaultTenant   string
	enforceTenantID bool
}

// NewTenantIsolationMiddleware creates a new tenant isolation middleware.
func NewTenantIsolationMiddleware(defaultTenant string, enforceTenantID bool) *TenantIsolationMiddleware {
	return &TenantIsolationMiddleware{
		extractor:       NewHeaderCorrelationExtractor(),
		defaultTenant:   defaultTenant,
		enforceTenantID: enforceTenantID,
	}
}

// Name returns the middleware name.
func (tim *TenantIsolationMiddleware) Name() string {
	return "tenant-isolation"
}

// Wrap wraps a handler with tenant isolation functionality.
func (tim *TenantIsolationMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	return messaging.MessageHandlerFunc(func(ctx context.Context, message *messaging.Message) error {
		// Extract correlation information
		correlation := tim.extractor.Extract(message)

		// Ensure tenant ID is present
		if correlation.TenantID == "" {
			if tim.defaultTenant != "" {
				correlation.TenantID = tim.defaultTenant
			} else if tim.enforceTenantID {
				return messaging.NewProcessingError(
					"tenant ID is required but not provided",
					fmt.Errorf("missing tenant ID in message %s", message.ID),
				)
			}
		}

		// Inject tenant information back
		tim.extractor.Inject(message, correlation)

		// Add tenant context
		ctx = WithCorrelation(ctx, correlation)

		return next.Handle(ctx, message)
	})
}

// CreateCorrelationMiddleware is a factory function to create correlation middleware from configuration.
func CreateCorrelationMiddleware(config map[string]interface{}) (messaging.Middleware, error) {
	correlationConfig := DefaultCorrelationConfig()

	// Configure generator type
	if generatorType, ok := config["generator_type"]; ok {
		if generatorTypeStr, ok := generatorType.(string); ok {
			switch generatorTypeStr {
			case "uuid":
				if prefix, ok := config["prefix"].(string); ok {
					correlationConfig.Generator = NewUUIDGenerator(prefix)
				} else {
					correlationConfig.Generator = NewUUIDGenerator("corr-")
				}
			case "timestamp":
				if prefix, ok := config["prefix"].(string); ok {
					correlationConfig.Generator = NewTimestampGenerator(prefix)
				} else {
					correlationConfig.Generator = NewTimestampGenerator("corr-")
				}
			default:
				correlationConfig.Generator = NewUUIDGenerator("corr-")
			}
		}
	}

	// Configure generation options
	if generateIfMissing, ok := config["generate_if_missing"]; ok {
		if generateIfMissingBool, ok := generateIfMissing.(bool); ok {
			correlationConfig.GenerateIfMissing = generateIfMissingBool
		}
	}

	if propagateToContext, ok := config["propagate_to_context"]; ok {
		if propagateToContextBool, ok := propagateToContext.(bool); ok {
			correlationConfig.PropagateToContext = propagateToContextBool
		}
	}

	if overrideExisting, ok := config["override_existing"]; ok {
		if overrideExistingBool, ok := overrideExisting.(bool); ok {
			correlationConfig.OverrideExisting = overrideExistingBool
		}
	}

	return NewCorrelationMiddleware(correlationConfig), nil
}
