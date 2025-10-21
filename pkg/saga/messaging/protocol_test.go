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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSagaMessageProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol *SagaMessageProtocol
	}{
		{
			name: "complete protocol message",
			protocol: &SagaMessageProtocol{
				Version:     ProtocolVersion1_1,
				MessageType: MessageTypeSagaEvent,
				SagaID:      "saga-123",
				EventID:     "event-456",
				EventType:   string(saga.EventSagaStarted),
				Timestamp:   time.Now(),
				Headers: map[string]string{
					HeaderSagaID:    "saga-123",
					HeaderEventID:   "event-456",
					HeaderEventType: string(saga.EventSagaStarted),
				},
				Payload:       map[string]interface{}{"key": "value"},
				CorrelationID: "corr-789",
				Source:        "order-service",
				TraceID:       "trace-001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.protocol)
			assert.Equal(t, "saga-123", tt.protocol.SagaID)
			assert.Equal(t, "event-456", tt.protocol.EventID)
		})
	}
}

func TestStandardProtocolValidator_Validate(t *testing.T) {
	validator := NewStandardProtocolValidator()

	tests := []struct {
		name      string
		protocol  *SagaMessageProtocol
		wantError bool
	}{
		{
			name: "valid protocol message",
			protocol: &SagaMessageProtocol{
				Version:     ProtocolVersion1_0,
				MessageType: MessageTypeSagaEvent,
				SagaID:      "saga-123",
				EventID:     "event-456",
				EventType:   string(saga.EventSagaStarted),
				Timestamp:   time.Now(),
			},
			wantError: false,
		},
		{
			name:      "nil protocol message",
			protocol:  nil,
			wantError: true,
		},
		{
			name: "missing saga ID",
			protocol: &SagaMessageProtocol{
				Version:     ProtocolVersion1_0,
				MessageType: MessageTypeSagaEvent,
				EventID:     "event-456",
				EventType:   string(saga.EventSagaStarted),
				Timestamp:   time.Now(),
			},
			wantError: true,
		},
		{
			name: "missing event ID",
			protocol: &SagaMessageProtocol{
				Version:     ProtocolVersion1_0,
				MessageType: MessageTypeSagaEvent,
				SagaID:      "saga-123",
				EventType:   string(saga.EventSagaStarted),
				Timestamp:   time.Now(),
			},
			wantError: true,
		},
		{
			name: "missing event type",
			protocol: &SagaMessageProtocol{
				Version:     ProtocolVersion1_0,
				MessageType: MessageTypeSagaEvent,
				SagaID:      "saga-123",
				EventID:     "event-456",
				Timestamp:   time.Now(),
			},
			wantError: true,
		},
		{
			name: "missing message type",
			protocol: &SagaMessageProtocol{
				Version:   ProtocolVersion1_0,
				SagaID:    "saga-123",
				EventID:   "event-456",
				EventType: string(saga.EventSagaStarted),
				Timestamp: time.Now(),
			},
			wantError: true,
		},
		{
			name: "zero timestamp",
			protocol: &SagaMessageProtocol{
				Version:     ProtocolVersion1_0,
				MessageType: MessageTypeSagaEvent,
				SagaID:      "saga-123",
				EventID:     "event-456",
				EventType:   string(saga.EventSagaStarted),
			},
			wantError: true,
		},
		{
			name: "unsupported version",
			protocol: &SagaMessageProtocol{
				Version:     "9.9",
				MessageType: MessageTypeSagaEvent,
				SagaID:      "saga-123",
				EventID:     "event-456",
				EventType:   string(saga.EventSagaStarted),
				Timestamp:   time.Now(),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.protocol)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardProtocolValidator_ValidateVersion(t *testing.T) {
	validator := NewStandardProtocolValidator()

	tests := []struct {
		name      string
		version   string
		wantError bool
	}{
		{
			name:      "valid version 1.0",
			version:   ProtocolVersion1_0,
			wantError: false,
		},
		{
			name:      "valid version 1.1",
			version:   ProtocolVersion1_1,
			wantError: false,
		},
		{
			name:      "empty version",
			version:   "",
			wantError: true,
		},
		{
			name:      "unsupported version",
			version:   "2.0",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateVersion(tt.version)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardProtocolValidator_ValidateHeaders(t *testing.T) {
	validator := NewStandardProtocolValidator()

	tests := []struct {
		name      string
		headers   map[string]string
		wantError bool
	}{
		{
			name: "valid headers",
			headers: map[string]string{
				HeaderSagaID:    "saga-123",
				HeaderEventID:   "event-456",
				HeaderEventType: string(saga.EventSagaStarted),
				HeaderTimestamp: time.Now().Format(time.RFC3339),
			},
			wantError: false,
		},
		{
			name:      "nil headers",
			headers:   nil,
			wantError: true,
		},
		{
			name: "missing required header",
			headers: map[string]string{
				HeaderSagaID:  "saga-123",
				HeaderEventID: "event-456",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateHeaders(tt.headers)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardTopicRouter_GetTopicForEvent(t *testing.T) {
	router := NewStandardTopicRouter()

	tests := []struct {
		name      string
		eventType saga.SagaEventType
		want      string
	}{
		{
			name:      "saga started event",
			eventType: saga.EventSagaStarted,
			want:      "saga.events.saga.started",
		},
		{
			name:      "saga completed event",
			eventType: saga.EventSagaCompleted,
			want:      "saga.events.saga.completed",
		},
		{
			name:      "step started event",
			eventType: saga.EventSagaStepStarted,
			want:      "saga.events.saga.step.started",
		},
		{
			name:      "compensation started event",
			eventType: saga.EventCompensationStarted,
			want:      "saga.events.compensation.started",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.GetTopicForEvent(tt.eventType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStandardTopicRouter_GetTopicForCommand(t *testing.T) {
	router := NewStandardTopicRouter()

	tests := []struct {
		name        string
		commandType string
		want        string
	}{
		{
			name:        "start command",
			commandType: "start",
			want:        "saga.commands.start",
		},
		{
			name:        "cancel command",
			commandType: "cancel",
			want:        "saga.commands.cancel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.GetTopicForCommand(tt.commandType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStandardTopicRouter_GetTopicForCompensation(t *testing.T) {
	router := NewStandardTopicRouter()

	tests := []struct {
		name      string
		eventType saga.SagaEventType
		want      string
	}{
		{
			name:      "compensation started",
			eventType: saga.EventCompensationStarted,
			want:      "saga.compensation.compensation.started",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.GetTopicForCompensation(tt.eventType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStandardTopicRouter_ParseTopic(t *testing.T) {
	router := NewStandardTopicRouter()

	tests := []struct {
		name       string
		topic      string
		wantPrefix string
		wantCat    string
		wantError  bool
	}{
		{
			name:       "valid event topic",
			topic:      "saga.events.started",
			wantPrefix: "saga.events",
			wantCat:    "started",
			wantError:  false,
		},
		{
			name:       "valid command topic",
			topic:      "saga.commands.cancel",
			wantPrefix: "saga.commands",
			wantCat:    "cancel",
			wantError:  false,
		},
		{
			name:      "empty topic",
			topic:     "",
			wantError: true,
		},
		{
			name:      "invalid topic format",
			topic:     "invalid",
			wantError: true,
		},
		{
			name:       "wildcard topic",
			topic:      "saga.events.*",
			wantPrefix: "saga.events",
			wantCat:    "*",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := router.ParseTopic(tt.topic)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPrefix, info.Prefix)
			assert.Equal(t, tt.wantCat, info.Category)
		})
	}
}

func TestStandardTopicRouter_ValidateTopic(t *testing.T) {
	router := NewStandardTopicRouter()

	tests := []struct {
		name      string
		topic     string
		wantError bool
	}{
		{
			name:      "valid topic",
			topic:     "saga.events.started",
			wantError: false,
		},
		{
			name:      "valid topic with wildcard",
			topic:     "saga.events.*",
			wantError: false,
		},
		{
			name:      "empty topic",
			topic:     "",
			wantError: true,
		},
		{
			name:      "too short",
			topic:     "ab",
			wantError: true,
		},
		{
			name:      "invalid characters",
			topic:     "saga@events#started",
			wantError: true,
		},
		{
			name:      "consecutive separators",
			topic:     "saga..events",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := router.ValidateTopic(tt.topic)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardVersionComparer_IsCompatible(t *testing.T) {
	comparer := NewStandardVersionComparer()

	tests := []struct {
		name     string
		version1 string
		version2 string
		want     bool
	}{
		{
			name:     "same version",
			version1: "1.0",
			version2: "1.0",
			want:     true,
		},
		{
			name:     "same major version",
			version1: "1.0",
			version2: "1.1",
			want:     true,
		},
		{
			name:     "different major version",
			version1: "1.0",
			version2: "2.0",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := comparer.IsCompatible(tt.version1, tt.version2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStandardVersionComparer_Compare(t *testing.T) {
	comparer := NewStandardVersionComparer()

	tests := []struct {
		name     string
		version1 string
		version2 string
		want     int
	}{
		{
			name:     "equal versions",
			version1: "1.0",
			version2: "1.0",
			want:     0,
		},
		{
			name:     "version1 less than version2",
			version1: "1.0",
			version2: "1.1",
			want:     -1,
		},
		{
			name:     "version1 greater than version2",
			version1: "1.1",
			version2: "1.0",
			want:     1,
		},
		{
			name:     "major version difference",
			version1: "1.0",
			version2: "2.0",
			want:     -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := comparer.Compare(tt.version1, tt.version2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStandardVersionComparer_IsSupported(t *testing.T) {
	comparer := NewStandardVersionComparer()

	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "supported version 1.0",
			version: ProtocolVersion1_0,
			want:    true,
		},
		{
			name:    "supported version 1.1",
			version: ProtocolVersion1_1,
			want:    true,
		},
		{
			name:    "unsupported old version",
			version: "0.9",
			want:    false,
		},
		{
			name:    "unsupported new version",
			version: "2.0",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := comparer.IsSupported(tt.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHeaderBuilder(t *testing.T) {
	t.Run("build complete headers", func(t *testing.T) {
		now := time.Now()
		builder := NewHeaderBuilder()

		headers := builder.
			WithSagaID("saga-123").
			WithEventID("event-456").
			WithEventType(string(saga.EventSagaStarted)).
			WithTimestamp(now).
			WithVersion(ProtocolVersion1_1).
			WithCorrelation("corr-789").
			WithSource("order-service").
			WithService("payment-service").
			WithServiceVersion("1.0.0").
			WithStepID("step-001").
			WithTraceID("trace-001").
			WithSpanID("span-001").
			WithParentSpanID("parent-span-001").
			WithContentType("application/json").
			WithAttempt(2).
			WithMaxAttempts(3).
			WithDuration(100*time.Millisecond).
			WithCustomHeader("custom-key", "custom-value").
			Build()

		assert.Equal(t, "saga-123", headers[HeaderSagaID])
		assert.Equal(t, "event-456", headers[HeaderEventID])
		assert.Equal(t, string(saga.EventSagaStarted), headers[HeaderEventType])
		assert.Equal(t, ProtocolVersion1_1, headers[HeaderVersion])
		assert.Equal(t, "corr-789", headers[HeaderCorrelation])
		assert.Equal(t, "order-service", headers[HeaderSource])
		assert.Equal(t, "payment-service", headers[HeaderService])
		assert.Equal(t, "1.0.0", headers[HeaderServiceVersion])
		assert.Equal(t, "step-001", headers[HeaderStepID])
		assert.Equal(t, "trace-001", headers[HeaderTraceID])
		assert.Equal(t, "span-001", headers[HeaderSpanID])
		assert.Equal(t, "parent-span-001", headers[HeaderParentSpanID])
		assert.Equal(t, "application/json", headers[HeaderContentType])
		assert.Equal(t, "2", headers[HeaderAttempt])
		assert.Equal(t, "3", headers[HeaderMaxAttempts])
		assert.Equal(t, "100", headers[HeaderDuration])
		assert.Equal(t, "custom-value", headers["custom-key"])
	})

	t.Run("build minimal headers", func(t *testing.T) {
		builder := NewHeaderBuilder()
		headers := builder.
			WithSagaID("saga-123").
			Build()

		assert.Equal(t, "saga-123", headers[HeaderSagaID])
		assert.Len(t, headers, 1)
	})
}

func TestBuildFromSagaEvent(t *testing.T) {
	now := time.Now()
	event := &saga.SagaEvent{
		ID:             "event-123",
		SagaID:         "saga-456",
		StepID:         "step-789",
		Type:           saga.EventSagaStarted,
		Version:        ProtocolVersion1_1,
		Timestamp:      now,
		CorrelationID:  "corr-001",
		Source:         "order-service",
		Service:        "payment-service",
		ServiceVersion: "1.0.0",
		TraceID:        "trace-001",
		SpanID:         "span-001",
		ParentSpanID:   "parent-span-001",
		Attempt:        2,
		MaxAttempts:    3,
		Duration:       100 * time.Millisecond,
	}

	headers := BuildFromSagaEvent(event)

	assert.Equal(t, "saga-456", headers[HeaderSagaID])
	assert.Equal(t, "event-123", headers[HeaderEventID])
	assert.Equal(t, string(saga.EventSagaStarted), headers[HeaderEventType])
	assert.Equal(t, ProtocolVersion1_1, headers[HeaderVersion])
	assert.Equal(t, "corr-001", headers[HeaderCorrelation])
	assert.Equal(t, "step-789", headers[HeaderStepID])
	assert.Equal(t, "order-service", headers[HeaderSource])
	assert.Equal(t, "payment-service", headers[HeaderService])
	assert.Equal(t, "1.0.0", headers[HeaderServiceVersion])
	assert.Equal(t, "trace-001", headers[HeaderTraceID])
	assert.Equal(t, "span-001", headers[HeaderSpanID])
	assert.Equal(t, "parent-span-001", headers[HeaderParentSpanID])
	assert.Equal(t, "2", headers[HeaderAttempt])
	assert.Equal(t, "3", headers[HeaderMaxAttempts])
	assert.Equal(t, "100", headers[HeaderDuration])
}

func TestBuildFromSagaEvent_MinimalEvent(t *testing.T) {
	event := &saga.SagaEvent{
		ID:        "event-123",
		SagaID:    "saga-456",
		Type:      saga.EventSagaStarted,
		Version:   ProtocolVersion1_0,
		Timestamp: time.Now(),
	}

	headers := BuildFromSagaEvent(event)

	assert.Equal(t, "saga-456", headers[HeaderSagaID])
	assert.Equal(t, "event-123", headers[HeaderEventID])
	assert.Equal(t, string(saga.EventSagaStarted), headers[HeaderEventType])
	assert.Equal(t, ProtocolVersion1_0, headers[HeaderVersion])

	// Optional fields should not be present
	_, hasCorrelation := headers[HeaderCorrelation]
	assert.False(t, hasCorrelation)
	_, hasStepID := headers[HeaderStepID]
	assert.False(t, hasStepID)
}
