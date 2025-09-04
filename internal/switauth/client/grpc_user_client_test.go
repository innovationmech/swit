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

package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/innovationmech/swit/pkg/tracing"
)

// MockServiceDiscovery is a mock implementation of ServiceDiscovery
type MockServiceDiscovery struct {
	mock.Mock
}

func (m *MockServiceDiscovery) GetInstanceRoundRobin(serviceName string) (string, error) {
	args := m.Called(serviceName)
	return args.String(0), args.Error(1)
}

func (m *MockServiceDiscovery) RegisterInstance(serviceName, address string, metadata map[string]string) error {
	args := m.Called(serviceName, address, metadata)
	return args.Error(0)
}

func (m *MockServiceDiscovery) DeregisterInstance(serviceName, address string) error {
	args := m.Called(serviceName, address)
	return args.Error(0)
}

func (m *MockServiceDiscovery) DiscoverInstances(serviceName string) ([]string, error) {
	args := m.Called(serviceName)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockServiceDiscovery) StartHealthCheck(serviceName, address string, healthCheckURL string) {}
func (m *MockServiceDiscovery) StopHealthCheck(serviceName, address string)                         {}
func (m *MockServiceDiscovery) Close() error                                                        { return nil }

// MockTracingManager is a mock implementation of TracingManager
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

// MockSpan is a mock implementation of Span
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

func TestNewGRPCUserClient(t *testing.T) {
	tests := []struct {
		name             string
		serviceDiscovery *MockServiceDiscovery
		tracingManager   *MockTracingManager
		expectError      bool
	}{
		{
			name:             "successful client creation",
			serviceDiscovery: &MockServiceDiscovery{},
			tracingManager:   &MockTracingManager{},
			expectError:      false,
		},
		{
			name:             "client creation with nil tracing manager",
			serviceDiscovery: &MockServiceDiscovery{},
			tracingManager:   nil,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewGRPCUserClient(tt.serviceDiscovery, tt.tracingManager)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)

				grpcClient, ok := client.(*GRPCUserClient)
				assert.True(t, ok)
				assert.Equal(t, "swit-serve", grpcClient.serviceName)
			}
		})
	}
}

func TestGRPCUserClient_ValidateUserCredentials_ServiceDiscoveryFailure(t *testing.T) {
	mockSD := &MockServiceDiscovery{}
	mockTM := &MockTracingManager{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockSD.On("GetInstanceRoundRobin", "swit-serve").Return("", assert.AnError)
	mockTM.On("StartSpan", mock.Anything, "gRPC_connect_to_service", mock.Anything).Return(context.Background(), mockSpan)
	mockSpan.On("End", mock.Anything)
	mockSpan.On("SetStatus", mock.Anything, "service discovery failed")
	mockSpan.On("SetAttribute", "error.type", "service_discovery")
	mockSpan.On("RecordError", assert.AnError, mock.Anything)

	client, err := NewGRPCUserClient(mockSD, mockTM)
	assert.NoError(t, err)

	// Test
	user, err := client.ValidateUserCredentials(context.Background(), "testuser", "password")

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "unable to discover swit-serve service")

	// Verify mocks
	mockSD.AssertExpectations(t)
	mockTM.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
}

// TestGRPCUserClient_TracingIntegration tests the tracing integration
func TestGRPCUserClient_TracingIntegration(t *testing.T) {
	t.Skip("Integration test - requires actual gRPC server")

	// This test would require a real gRPC server for complete integration testing
	// In a real test suite, you would:
	// 1. Start a test gRPC server
	// 2. Configure service discovery to point to the test server
	// 3. Make actual gRPC calls with tracing
	// 4. Verify that spans are created and propagated correctly
}

// TestGRPCUserClient_ConnectionStateMonitoring tests connection state monitoring
func TestGRPCUserClient_ConnectionStateMonitoring(t *testing.T) {
	mockSD := &MockServiceDiscovery{}
	mockTM := &MockTracingManager{}

	client, err := NewGRPCUserClient(mockSD, mockTM)
	assert.NoError(t, err)

	grpcClient, ok := client.(*GRPCUserClient)
	assert.True(t, ok)

	// Test connection state when no connection exists
	state := grpcClient.GetConnectionState()
	assert.Equal(t, "IDLE", state.String())
}

// Benchmark tests for performance impact assessment
func BenchmarkGRPCUserClient_WithTracing(b *testing.B) {
	mockSD := &MockServiceDiscovery{}
	mockTM := &MockTracingManager{}
	mockSpan := &MockSpan{}

	// Setup minimal mocks for benchmarking
	mockSD.On("GetInstanceRoundRobin", mock.Anything).Return("", assert.AnError)
	mockTM.On("StartSpan", mock.Anything, mock.Anything, mock.Anything).Return(context.Background(), mockSpan)
	mockSpan.On("End", mock.Anything)
	mockSpan.On("SetStatus", mock.Anything, mock.Anything)
	mockSpan.On("SetAttribute", mock.Anything, mock.Anything)
	mockSpan.On("RecordError", mock.Anything, mock.Anything)

	client, _ := NewGRPCUserClient(mockSD, mockTM)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.ValidateUserCredentials(context.Background(), "testuser", "password")
	}
}

func BenchmarkGRPCUserClient_WithoutTracing(b *testing.B) {
	mockSD := &MockServiceDiscovery{}
	client, _ := NewGRPCUserClient(mockSD, nil) // No tracing manager

	// Setup minimal mocks
	mockSD.On("GetInstanceRoundRobin", mock.Anything).Return("", assert.AnError)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.ValidateUserCredentials(context.Background(), "testuser", "password")
	}
}
