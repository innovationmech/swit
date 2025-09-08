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

package messaging

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBaseMessagingError_Error(t *testing.T) {
	tests := []struct {
		name     string
		error    *BaseMessagingError
		expected string
	}{
		{
			name: "basic error",
			error: &BaseMessagingError{
				Type:    ErrorTypeConnection,
				Code:    ErrCodeConnectionFailed,
				Message: "connection failed",
			},
			expected: "[CONNECTION:CONN_001] connection failed",
		},
		{
			name: "error with context",
			error: &BaseMessagingError{
				Type:    ErrorTypePublishing,
				Code:    ErrCodePublishFailed,
				Message: "publish failed",
				Context: ErrorContext{
					Component: "kafka-publisher",
					Operation: "publish",
					Topic:     "test-topic",
				},
			},
			expected: "[PUBLISHING:PUB_001] publish failed component=kafka-publisher operation=publish topic=test-topic",
		},
		{
			name: "error with cause",
			error: &BaseMessagingError{
				Type:    ErrorTypeConnection,
				Code:    ErrCodeConnectionTimeout,
				Message: "connection timeout",
				Cause:   fmt.Errorf("network unreachable"),
			},
			expected: "[CONNECTION:CONN_002] connection timeout (caused by: network unreachable)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.error.Error()
			if actual != tt.expected {
				t.Errorf("Error() = %q, expected %q", actual, tt.expected)
			}
		})
	}
}

func TestBaseMessagingError_Is(t *testing.T) {
	err1 := &BaseMessagingError{
		Type: ErrorTypeConnection,
		Code: ErrCodeConnectionFailed,
	}

	err2 := &BaseMessagingError{
		Type: ErrorTypeConnection,
		Code: ErrCodeConnectionFailed,
	}

	err3 := &BaseMessagingError{
		Type: ErrorTypeConnection,
		Code: ErrCodeConnectionTimeout,
	}

	if !errors.Is(err1, err2) {
		t.Error("Expected err1.Is(err2) to be true")
	}

	if errors.Is(err1, err3) {
		t.Error("Expected err1.Is(err3) to be false")
	}
}

func TestBaseMessagingError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := &BaseMessagingError{
		Type:  ErrorTypeConnection,
		Code:  ErrCodeConnectionFailed,
		Cause: cause,
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, expected %v", unwrapped, cause)
	}
}

func TestBaseMessagingError_WithContext(t *testing.T) {
	originalErr := &BaseMessagingError{
		Type:    ErrorTypeConnection,
		Code:    ErrCodeConnectionFailed,
		Message: "connection failed",
		Context: ErrorContext{
			Component: "original-component",
		},
	}

	newContext := ErrorContext{
		Operation: "new-operation",
		Topic:     "test-topic",
		MessageHeaders: map[string]string{
			"correlation-id": "test-123",
		},
	}

	newErr := originalErr.WithContext(newContext)

	// Should preserve original component
	if newErr.Context.Component != "original-component" {
		t.Errorf("Expected component to be preserved, got %q", newErr.Context.Component)
	}

	// Should add new fields
	if newErr.Context.Operation != "new-operation" {
		t.Errorf("Expected operation to be %q, got %q", "new-operation", newErr.Context.Operation)
	}

	if newErr.Context.Topic != "test-topic" {
		t.Errorf("Expected topic to be %q, got %q", "test-topic", newErr.Context.Topic)
	}

	if newErr.Context.MessageHeaders["correlation-id"] != "test-123" {
		t.Errorf("Expected message header to be preserved")
	}
}

func TestBaseMessagingError_WithCause(t *testing.T) {
	originalErr := &BaseMessagingError{
		Type:    ErrorTypeConnection,
		Code:    ErrCodeConnectionFailed,
		Message: "connection failed",
	}

	cause := fmt.Errorf("network error")
	newErr := originalErr.WithCause(cause)

	if newErr.Cause != cause {
		t.Errorf("Expected cause to be %v, got %v", cause, newErr.Cause)
	}
}

func TestBaseMessagingError_WithSuggestedAction(t *testing.T) {
	originalErr := &BaseMessagingError{
		Type:    ErrorTypeConnection,
		Code:    ErrCodeConnectionFailed,
		Message: "connection failed",
	}

	action := "Check network connectivity"
	newErr := originalErr.WithSuggestedAction(action)

	if newErr.SuggestedAction != action {
		t.Errorf("Expected suggested action to be %q, got %q", action, newErr.SuggestedAction)
	}
}

func TestBaseMessagingError_WithDetails(t *testing.T) {
	originalErr := &BaseMessagingError{
		Type:    ErrorTypeConnection,
		Code:    ErrCodeConnectionFailed,
		Message: "connection failed",
	}

	details := map[string]interface{}{
		"retry_count": 3,
		"timeout":     "30s",
	}

	newErr := originalErr.WithDetails(details)

	if newErr.AdditionalDetails["retry_count"] != 3 {
		t.Errorf("Expected retry_count to be 3")
	}

	if newErr.AdditionalDetails["timeout"] != "30s" {
		t.Errorf("Expected timeout to be 30s")
	}
}

func TestBaseMessagingError_GetUserFacingMessage(t *testing.T) {
	err := &BaseMessagingError{
		Message:         "Connection failed",
		SuggestedAction: "Check network connectivity",
	}

	expected := "Connection failed. Check network connectivity"
	actual := err.GetUserFacingMessage()

	if actual != expected {
		t.Errorf("GetUserFacingMessage() = %q, expected %q", actual, expected)
	}
}

func TestBaseMessagingError_GetDiagnosticInfo(t *testing.T) {
	err := &BaseMessagingError{
		Type:            ErrorTypeConnection,
		Code:            ErrCodeConnectionFailed,
		Message:         "connection failed",
		Severity:        SeverityCritical,
		Retryable:       true,
		Timestamp:       time.Now(),
		Stack:           "stack trace",
		SuggestedAction: "check network",
		AdditionalDetails: map[string]interface{}{
			"timeout": "30s",
		},
		Cause: fmt.Errorf("network error"),
		Context: ErrorContext{
			Component: "test-component",
		},
	}

	info := err.GetDiagnosticInfo()

	if info["type"] != ErrorTypeConnection {
		t.Errorf("Expected type to be %v", ErrorTypeConnection)
	}

	if info["code"] != ErrCodeConnectionFailed {
		t.Errorf("Expected code to be %v", ErrCodeConnectionFailed)
	}

	if info["message"] != "connection failed" {
		t.Errorf("Expected message to be 'connection failed'")
	}

	if info["severity"] != SeverityCritical {
		t.Errorf("Expected severity to be %v", SeverityCritical)
	}

	if info["retryable"] != true {
		t.Errorf("Expected retryable to be true")
	}

	if info["stack"] != "stack trace" {
		t.Errorf("Expected stack to be 'stack trace'")
	}

	if info["suggested_action"] != "check network" {
		t.Errorf("Expected suggested_action to be 'check network'")
	}

	if info["cause"] != "network error" {
		t.Errorf("Expected cause to be 'network error'")
	}

	context, ok := info["context"].(ErrorContext)
	if !ok {
		t.Errorf("Expected context to be ErrorContext type")
	}

	if context.Component != "test-component" {
		t.Errorf("Expected component to be 'test-component'")
	}

	additionalDetails, ok := info["additional_details"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected additional_details to be map type")
	}

	if additionalDetails["timeout"] != "30s" {
		t.Errorf("Expected timeout to be '30s'")
	}
}

func TestErrorBuilder_FluentInterface(t *testing.T) {
	err := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("Connection failed to broker").
		Severity(SeverityCritical).
		Retryable(true).
		Context(ErrorContext{
			Component: "kafka-client",
			Operation: "connect",
		}).
		Cause(fmt.Errorf("network timeout")).
		SuggestedAction("Check network connectivity").
		Details(map[string]interface{}{
			"timeout": "30s",
			"retries": 3,
		}).
		Component("updated-component").
		Operation("updated-operation").
		Topic("test-topic").
		Queue("test-queue").
		Build()

	if err.Type != ErrorTypeConnection {
		t.Errorf("Expected type to be %v, got %v", ErrorTypeConnection, err.Type)
	}

	if err.Code != ErrCodeConnectionFailed {
		t.Errorf("Expected code to be %v, got %v", ErrCodeConnectionFailed, err.Code)
	}

	if err.Message != "Connection failed to broker" {
		t.Errorf("Expected message to be 'Connection failed to broker', got %q", err.Message)
	}

	if err.Severity != SeverityCritical {
		t.Errorf("Expected severity to be %v, got %v", SeverityCritical, err.Severity)
	}

	if !err.Retryable {
		t.Errorf("Expected retryable to be true")
	}

	if err.Context.Component != "updated-component" {
		t.Errorf("Expected component to be 'updated-component', got %q", err.Context.Component)
	}

	if err.Context.Operation != "updated-operation" {
		t.Errorf("Expected operation to be 'updated-operation', got %q", err.Context.Operation)
	}

	if err.Context.Topic != "test-topic" {
		t.Errorf("Expected topic to be 'test-topic', got %q", err.Context.Topic)
	}

	if err.Context.Queue != "test-queue" {
		t.Errorf("Expected queue to be 'test-queue', got %q", err.Context.Queue)
	}

	if err.Cause == nil {
		t.Errorf("Expected cause to be set")
	}

	if err.SuggestedAction != "Check network connectivity" {
		t.Errorf("Expected suggested action to be 'Check network connectivity', got %q", err.SuggestedAction)
	}

	if err.AdditionalDetails["timeout"] != "30s" {
		t.Errorf("Expected timeout detail to be '30s'")
	}

	if err.AdditionalDetails["retries"] != 3 {
		t.Errorf("Expected retries detail to be 3")
	}
}

func TestErrorBuilder_Messagef(t *testing.T) {
	err := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Messagef("Connection failed to broker %s on port %d", "localhost", 9092).
		Build()

	expected := "Connection failed to broker localhost on port 9092"
	if err.Message != expected {
		t.Errorf("Expected message to be %q, got %q", expected, err.Message)
	}
}

func TestErrorBuilder_Error(t *testing.T) {
	err := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("Connection failed").
		Error()

	if err == nil {
		t.Error("Expected error to be non-nil")
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "CONNECTION") {
		t.Errorf("Expected error string to contain 'CONNECTION', got %q", errorStr)
	}

	if !strings.Contains(errorStr, "CONN_001") {
		t.Errorf("Expected error string to contain 'CONN_001', got %q", errorStr)
	}

	if !strings.Contains(errorStr, "Connection failed") {
		t.Errorf("Expected error string to contain 'Connection failed', got %q", errorStr)
	}
}

func TestErrorGroup(t *testing.T) {
	ctx := context.Background()
	errorGroup := NewErrorGroup(ctx)

	if errorGroup.HasErrors() {
		t.Error("Expected new error group to have no errors")
	}

	if errorGroup.FirstError() != nil {
		t.Error("Expected first error to be nil")
	}

	if len(errorGroup.Errors()) != 0 {
		t.Error("Expected errors slice to be empty")
	}

	// Add some errors
	err1 := fmt.Errorf("first error")
	err2 := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("connection error").
		Error()

	errorGroup.Add(err1)
	errorGroup.Add(err2)

	if !errorGroup.HasErrors() {
		t.Error("Expected error group to have errors")
	}

	if errorGroup.FirstError() != err1 {
		t.Error("Expected first error to be err1")
	}

	if len(errorGroup.Errors()) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errorGroup.Errors()))
	}

	// Test error string
	errorStr := errorGroup.Error()
	if !strings.Contains(errorStr, "first error") {
		t.Errorf("Expected error string to contain 'first error', got %q", errorStr)
	}

	if !strings.Contains(errorStr, "connection error") {
		t.Errorf("Expected error string to contain 'connection error', got %q", errorStr)
	}

	if !strings.Contains(errorStr, "multiple errors occurred") {
		t.Errorf("Expected error string to contain 'multiple errors occurred', got %q", errorStr)
	}

	// Add nil error (should be ignored)
	errorGroup.Add(nil)
	if len(errorGroup.Errors()) != 2 {
		t.Errorf("Expected 2 errors after adding nil, got %d", len(errorGroup.Errors()))
	}
}

func TestErrorGroup_SingleError(t *testing.T) {
	ctx := context.Background()
	errorGroup := NewErrorGroup(ctx)

	err := fmt.Errorf("single error")
	errorGroup.Add(err)

	errorStr := errorGroup.Error()
	if errorStr != "single error" {
		t.Errorf("Expected single error string to be 'single error', got %q", errorStr)
	}
}

func TestErrorGroup_ToMessagingError(t *testing.T) {
	ctx := context.Background()
	errorGroup := NewErrorGroup(ctx)

	// Empty group
	msgErr := errorGroup.ToMessagingError()
	if msgErr != nil {
		t.Error("Expected nil messaging error for empty group")
	}

	// Single messaging error
	baseErr := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("connection error").
		Build()
	errorGroup.Add(baseErr)

	msgErr = errorGroup.ToMessagingError()
	if msgErr != baseErr {
		t.Error("Expected same messaging error for single error")
	}

	// Multiple errors
	errorGroup.Add(fmt.Errorf("another error"))
	msgErr = errorGroup.ToMessagingError()

	if msgErr.Type != ErrorTypeInternal {
		t.Errorf("Expected type to be %v, got %v", ErrorTypeInternal, msgErr.Type)
	}

	if msgErr.Code != ErrCodeInternal {
		t.Errorf("Expected code to be %v, got %v", ErrCodeInternal, msgErr.Code)
	}

	if !strings.Contains(msgErr.Message, "multiple errors occurred") {
		t.Errorf("Expected message to contain 'multiple errors occurred', got %q", msgErr.Message)
	}

	if !strings.Contains(msgErr.Message, "2 errors") {
		t.Errorf("Expected message to contain '2 errors', got %q", msgErr.Message)
	}

	if msgErr.AdditionalDetails["error_count"] != 2 {
		t.Errorf("Expected error_count to be 2")
	}
}

// Benchmark tests for performance
func BenchmarkErrorBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
			Message("Connection failed").
			Severity(SeverityCritical).
			Retryable(true).
			Context(ErrorContext{
				Component: "test-component",
				Operation: "connect",
			}).
			Build()
	}
}

func BenchmarkErrorString(b *testing.B) {
	err := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("Connection failed").
		Context(ErrorContext{
			Component: "test-component",
			Operation: "connect",
			Topic:     "test-topic",
		}).
		Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkErrorGroup(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		errorGroup := NewErrorGroup(ctx)
		for j := 0; j < 10; j++ {
			errorGroup.Add(fmt.Errorf("error %d", j))
		}
		_ = errorGroup.Error()
	}
}

// Integration tests with existing error types
func TestIntegrationWithExistingErrorTypes(t *testing.T) {
	// Test that new error system works with existing MessagingError
	oldErr := &MessagingError{
		Code:      ErrConnectionFailed,
		Message:   "old error",
		Retryable: true,
		Timestamp: time.Now(),
	}

	// Should be able to wrap old error as cause
	newErr := NewError(ErrorTypeConnection, ErrCodeConnectionFailed).
		Message("New error wrapping old error").
		Cause(oldErr).
		Build()

	if newErr.Cause != oldErr {
		t.Error("Expected old error to be wrapped as cause")
	}

	// Test error unwrapping
	if !errors.Is(newErr, oldErr) {
		t.Error("Expected error.Is to work with wrapped error")
	}

	unwrapped := errors.Unwrap(newErr)
	if unwrapped != oldErr {
		t.Error("Expected error.Unwrap to return old error")
	}
}
