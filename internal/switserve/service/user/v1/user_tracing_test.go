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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/innovationmech/swit/internal/switserve/model"
	"github.com/innovationmech/swit/pkg/tracing"
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

func (m *MockTracingManager) InjectHTTPHeaders(ctx context.Context, headers interface{}) {
	m.Called(ctx, headers)
}

func (m *MockTracingManager) ExtractHTTPHeaders(headers interface{}) context.Context {
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

func (m *MockSpan) SetAttributes(attrs ...interface{}) {
	m.Called(attrs)
}

func (m *MockSpan) AddEvent(name string, opts ...interface{}) {
	m.Called(name, opts)
}

func (m *MockSpan) SetStatus(code interface{}, description string) {
	m.Called(code, description)
}

func (m *MockSpan) End(opts ...interface{}) {
	m.Called(opts)
}

func (m *MockSpan) RecordError(err error, opts ...interface{}) {
	m.Called(err, opts)
}

func (m *MockSpan) SpanContext() interface{} {
	args := m.Called()
	return args.Get(0)
}

func TestUserService_CreateUser_WithTracing(t *testing.T) {
	// Setup mocks
	mockRepo := &MockUserRepository{}
	mockTracingManager := &MockTracingManager{}
	mockSpan := &MockSpan{}

	// Setup expectations
	ctx := context.Background()
	mockTracingManager.On("StartSpan", ctx, "UserService.CreateUser", mock.Anything).
		Return(ctx, mockSpan)
	mockSpan.On("End", mock.Anything).Return()
	mockSpan.On("SetAttribute", mock.AnythingOfType("string"), mock.Anything).Return()

	// Create validation span
	mockTracingManager.On("StartSpan", ctx, "validate_user_input").
		Return(ctx, mockSpan)

	// Create hash password span
	mockTracingManager.On("StartSpan", ctx, "hash_password").
		Return(ctx, mockSpan)

	// Setup repository mock
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*model.User")).Return(nil)

	// Create user service with tracing
	userService, err := NewUserSrv(
		WithUserRepository(mockRepo),
		WithTracingManager(mockTracingManager),
	)
	assert.NoError(t, err)

	// Execute the method
	err = userService.CreateUser(ctx, user)

	// Assertions
	assert.NoError(t, err)
	mockTracingManager.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestUserService_CreateUser_WithoutTracing(t *testing.T) {
	// Setup mocks
	mockRepo := &MockUserRepository{}

	// Setup repository mock
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

	// Create user service without tracing
	userService, err := NewUserSrv(
		WithUserRepository(mockRepo),
	)
	assert.NoError(t, err)

	// Execute the method
	err = userService.CreateUser(context.Background(), user)

	// Assertions
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetUserByEmail_WithTracing(t *testing.T) {
	// Setup mocks
	mockRepo := &MockUserRepository{}
	mockTracingManager := &MockTracingManager{}
	mockSpan := &MockSpan{}

	// Setup expectations
	ctx := context.Background()
	mockTracingManager.On("StartSpan", ctx, "UserService.GetUserByEmail", mock.Anything).
		Return(ctx, mockSpan)
	mockSpan.On("End", mock.Anything).Return()
	mockSpan.On("SetAttribute", mock.AnythingOfType("string"), mock.Anything).Return()

	// Setup repository mock
	expectedUser := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	mockRepo.On("GetUserByEmail", ctx, "test@example.com").Return(expectedUser, nil)

	// Create user service with tracing
	userService, err := NewUserSrv(
		WithUserRepository(mockRepo),
		WithTracingManager(mockTracingManager),
	)
	assert.NoError(t, err)

	// Execute the method
	user, err := userService.GetUserByEmail(ctx, "test@example.com")

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockTracingManager.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}