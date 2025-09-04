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

package v1

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/innovationmech/swit/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// MockTracingManager is a mock implementation of TracingManager for testing
type MockTracingManager struct {
	mock.Mock
}

func (m *MockTracingManager) Initialize(ctx context.Context, config *tracing.TracingConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockTracingManager) StartSpan(ctx context.Context, operationName string, opts ...tracing.SpanOption) (context.Context, tracing.Span) {
	args := m.Called(ctx, operationName, opts)
	return args.Get(0).(context.Context), args.Get(1).(tracing.Span)
}

func (m *MockTracingManager) SpanFromContext(ctx context.Context) tracing.Span {
	args := m.Called(ctx)
	return args.Get(0).(tracing.Span)
}

func (m *MockTracingManager) InjectHTTPHeaders(ctx context.Context, headers http.Header) {
	m.Called(ctx, headers)
}

func (m *MockTracingManager) ExtractHTTPHeaders(headers http.Header) context.Context {
	args := m.Called(headers)
	return args.Get(0).(context.Context)
}

func (m *MockTracingManager) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockSpan is a mock implementation of Span for testing
type MockSpan struct {
	mock.Mock
}

func (m *MockSpan) SetAttribute(key string, value interface{}) {
	m.Called(key, value)
}

func (m *MockSpan) SetAttributes(attrs ...attribute.KeyValue) {
	m.Called(attrs)
}

func (m *MockSpan) AddEvent(name string, opts ...oteltrace.EventOption) {
	m.Called(name, opts)
}

func (m *MockSpan) SetStatus(code codes.Code, description string) {
	m.Called(code, description)
}

func (m *MockSpan) End(opts ...oteltrace.SpanEndOption) {
	m.Called(opts)
}

func (m *MockSpan) RecordError(err error, opts ...oteltrace.EventOption) {
	m.Called(err, opts)
}

func (m *MockSpan) SpanContext() oteltrace.SpanContext {
	args := m.Called()
	return args.Get(0).(oteltrace.SpanContext)
}

func TestNotificationService_CreateNotification_WithTracing(t *testing.T) {
	// Setup mocks
	mockTracingManager := &MockTracingManager{}
	mockMainSpan := &MockSpan{}
	mockValidationSpan := &MockSpan{}
	mockStorageSpan := &MockSpan{}

	ctx := context.Background()

	// Setup expectations for main span
	mockTracingManager.On("StartSpan", ctx, "NotificationService.CreateNotification", mock.Anything).
		Return(ctx, mockMainSpan)
	mockMainSpan.On("End", mock.Anything).Return()
	mockMainSpan.On("SetAttribute", mock.AnythingOfType("string"), mock.Anything).Return()

	// Setup expectations for validation span
	mockTracingManager.On("StartSpan", ctx, "validate_notification_input", mock.Anything).
		Return(ctx, mockValidationSpan)
	mockValidationSpan.On("End", mock.Anything).Return()

	// Setup expectations for storage span
	mockTracingManager.On("StartSpan", ctx, "store_notification", mock.Anything).
		Return(ctx, mockStorageSpan)
	mockStorageSpan.On("End", mock.Anything).Return()
	mockStorageSpan.On("SetAttribute", mock.AnythingOfType("string"), mock.Anything).Return()

	// Create notification service with tracing
	service := NewServiceWithTracing(mockTracingManager)

	// Execute the method
	notification, err := service.CreateNotification(ctx, "user123", "Test Title", "Test Content")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, "user123", notification.UserID)
	assert.Equal(t, "Test Title", notification.Title)
	assert.Equal(t, "Test Content", notification.Content)

	mockTracingManager.AssertExpectations(t)
	mockMainSpan.AssertExpectations(t)
	mockValidationSpan.AssertExpectations(t)
	mockStorageSpan.AssertExpectations(t)
}

func TestNotificationService_CreateNotification_WithoutTracing(t *testing.T) {
	// Create notification service without tracing
	service := NewService()

	// Execute the method
	notification, err := service.CreateNotification(context.Background(), "user123", "Test Title", "Test Content")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, "user123", notification.UserID)
	assert.Equal(t, "Test Title", notification.Title)
	assert.Equal(t, "Test Content", notification.Content)
}

func TestNotificationService_CreateNotification_ValidationError_WithTracing(t *testing.T) {
	// Setup mocks
	mockTracingManager := &MockTracingManager{}
	mockMainSpan := &MockSpan{}
	mockValidationSpan := &MockSpan{}

	ctx := context.Background()

	// Setup expectations for main span
	mockTracingManager.On("StartSpan", ctx, "NotificationService.CreateNotification", mock.Anything).
		Return(ctx, mockMainSpan)
	mockMainSpan.On("End", mock.Anything).Return()
	mockMainSpan.On("SetStatus", mock.Anything, "validation failed").Return()

	// Setup expectations for validation span
	mockTracingManager.On("StartSpan", ctx, "validate_notification_input", mock.Anything).
		Return(ctx, mockValidationSpan)
	mockValidationSpan.On("End", mock.Anything).Return()
	mockValidationSpan.On("SetStatus", mock.Anything, "userID cannot be empty").Return()

	// Create notification service with tracing
	service := NewServiceWithTracing(mockTracingManager)

	// Execute the method with empty userID
	notification, err := service.CreateNotification(ctx, "", "Test Title", "Test Content")

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, notification)
	assert.Contains(t, err.Error(), "userID cannot be empty")

	mockTracingManager.AssertExpectations(t)
	mockMainSpan.AssertExpectations(t)
	mockValidationSpan.AssertExpectations(t)
}
