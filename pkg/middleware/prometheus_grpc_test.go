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

package middleware

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/innovationmech/swit/pkg/types"
)

// mockGRPCMetricsCollector is the same as the HTTP one but included here for clarity
type mockGRPCMetricsCollector struct {
	mu         sync.RWMutex
	counters   map[string]float64
	gauges     map[string]float64
	histograms map[string][]float64
	labels     map[string]map[string]string
	panics     bool
}

func newMockGRPCMetricsCollector() *mockGRPCMetricsCollector {
	return &mockGRPCMetricsCollector{
		counters:   make(map[string]float64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
		labels:     make(map[string]map[string]string),
	}
}

func (m *mockGRPCMetricsCollector) IncrementCounter(name string, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.counters[key]++
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockGRPCMetricsCollector) AddToCounter(name string, value float64, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.counters[key] += value
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockGRPCMetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.gauges[key] = value
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockGRPCMetricsCollector) IncrementGauge(name string, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.gauges[key]++
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockGRPCMetricsCollector) DecrementGauge(name string, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.gauges[key]--
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockGRPCMetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.histograms[key] = append(m.histograms[key], value)
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockGRPCMetricsCollector) GetMetrics() []types.Metric {
	return nil
}

func (m *mockGRPCMetricsCollector) GetMetric(name string) (*types.Metric, bool) {
	return nil, false
}

func (m *mockGRPCMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters = make(map[string]float64)
	m.gauges = make(map[string]float64)
	m.histograms = make(map[string][]float64)
	m.labels = make(map[string]map[string]string)
}

func (m *mockGRPCMetricsCollector) getKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	// Sort label keys for deterministic key generation
	var keys []string
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	parts = append(parts, name)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
	}
	return strings.Join(parts, ",")
}

func (m *mockGRPCMetricsCollector) copyLabels(labels map[string]string) map[string]string {
	copied := make(map[string]string)
	for k, v := range labels {
		copied[k] = v
	}
	return copied
}

// Helper methods for testing
func (m *mockGRPCMetricsCollector) getCounterValue(name string, labels map[string]string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.getKey(name, labels)
	return m.counters[key]
}

func (m *mockGRPCMetricsCollector) getGaugeValue(name string, labels map[string]string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.getKey(name, labels)
	return m.gauges[key]
}

func (m *mockGRPCMetricsCollector) getHistogramValues(name string, labels map[string]string) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.getKey(name, labels)
	values := make([]float64, len(m.histograms[key]))
	copy(values, m.histograms[key])
	return values
}

func (m *mockGRPCMetricsCollector) setPanics(panics bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.panics = panics
}

// Mock gRPC service implementations
type mockUnaryHandler func(ctx context.Context, req interface{}) (interface{}, error)
type mockStreamHandler func(srv interface{}, stream grpc.ServerStream) error

type mockServerStream struct {
	ctx      context.Context
	sentMsgs []interface{}
	recvMsgs []interface{}
	recvErr  error
	sendErr  error
}

func newMockServerStream(ctx context.Context) *mockServerStream {
	return &mockServerStream{
		ctx:      ctx,
		sentMsgs: make([]interface{}, 0),
		recvMsgs: make([]interface{}, 0),
	}
}

func (s *mockServerStream) Context() context.Context {
	return s.ctx
}

func (s *mockServerStream) SendMsg(m interface{}) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sentMsgs = append(s.sentMsgs, m)
	return nil
}

func (s *mockServerStream) RecvMsg(m interface{}) error {
	if s.recvErr != nil {
		return s.recvErr
	}
	if len(s.recvMsgs) == 0 {
		return io.EOF
	}
	// Simulate receiving a message
	s.recvMsgs = s.recvMsgs[1:]
	return nil
}

// Required methods for grpc.ServerStream interface
func (s *mockServerStream) SetHeader(md metadata.MD) error {
	return nil
}

func (s *mockServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (s *mockServerStream) SetTrailer(md metadata.MD) {
}

func (s *mockServerStream) setSendError(err error) {
	s.sendErr = err
}

func (s *mockServerStream) setRecvError(err error) {
	s.recvErr = err
}

func (s *mockServerStream) addRecvMsg(msg interface{}) {
	s.recvMsgs = append(s.recvMsgs, msg)
}

func TestDefaultPrometheusGRPCConfig(t *testing.T) {
	config := DefaultPrometheusGRPCConfig()

	assert.True(t, config.Enabled)
	assert.True(t, config.CollectMessageSize)
	assert.True(t, config.CollectStreamMessages)
	assert.True(t, config.HandleStreamingErrors)
	assert.Equal(t, "unknown", config.ServiceName)

	expectedExcludeMethods := []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
	}
	assert.Equal(t, expectedExcludeMethods, config.ExcludeMethods)
}

func TestNewPrometheusGRPCInterceptor(t *testing.T) {
	collector := newMockGRPCMetricsCollector()

	t.Run("WithConfig", func(t *testing.T) {
		config := &PrometheusGRPCConfig{
			Enabled:     true,
			ServiceName: "test-service",
		}

		interceptor := NewPrometheusGRPCInterceptor(collector, config)
		assert.NotNil(t, interceptor)
		assert.Equal(t, config, interceptor.config)
		assert.Equal(t, collector, interceptor.metricsCollector)
		// Registry is no longer exposed in the interceptor
	})

	t.Run("WithNilConfig", func(t *testing.T) {
		interceptor := NewPrometheusGRPCInterceptor(collector, nil)
		assert.NotNil(t, interceptor)
		assert.NotNil(t, interceptor.config)
		assert.Equal(t, DefaultPrometheusGRPCConfig(), interceptor.config)
	})
}

func TestPrometheusGRPCInterceptor_UnaryInterceptor(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		Enabled:            true,
		ServiceName:        "test-service",
		CollectMessageSize: true,
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)
	unaryInterceptor := interceptor.UnaryServerInterceptor()

	t.Run("SuccessfulUnaryCall", func(t *testing.T) {
		collector.Reset()

		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "response", nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.TestService/TestMethod",
		}

		ctx := context.Background()
		resp, err := unaryInterceptor(ctx, "request", info, handler)

		assert.NoError(t, err)
		assert.Equal(t, "response", resp)

		// Verify metrics
		expectedLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestMethod",
		}

		// Check server started
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_started_total", expectedLabels))

		// Check server handled with OK code
		handledLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestMethod",
			"code":    codes.OK.String(),
		}
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_handled_total", handledLabels))

		// Check handling duration
		durations := collector.getHistogramValues("grpc_server_handling_seconds", expectedLabels)
		assert.Len(t, durations, 1)
		assert.Greater(t, durations[0], float64(0))

		// Check message counts
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_msg_sent_total", expectedLabels))
	})

	t.Run("UnaryCallWithError", func(t *testing.T) {
		collector.Reset()

		expectedErr := status.Error(codes.InvalidArgument, "invalid request")
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, expectedErr
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.TestService/TestMethod",
		}

		ctx := context.Background()
		resp, err := unaryInterceptor(ctx, "request", info, handler)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, expectedErr, err)

		// Verify metrics with error code
		handledLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestMethod",
			"code":    codes.InvalidArgument.String(),
		}
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_handled_total", handledLabels))

		// Message received but not sent due to error
		expectedLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestMethod",
		}
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
		assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_sent_total", expectedLabels))
	})

	t.Run("UnaryCallWithNonGRPCError", func(t *testing.T) {
		collector.Reset()

		expectedErr := errors.New("regular error")
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, expectedErr
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.TestService/TestMethod",
		}

		ctx := context.Background()
		_, err := unaryInterceptor(ctx, "request", info, handler)

		assert.Error(t, err)

		// Should be classified as Unknown status
		handledLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestMethod",
			"code":    codes.Unknown.String(),
		}
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_handled_total", handledLabels))
	})
}

func TestPrometheusGRPCInterceptor_StreamInterceptor(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		Enabled:               true,
		ServiceName:           "test-service",
		CollectStreamMessages: true,
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)
	streamInterceptor := interceptor.StreamServerInterceptor()

	t.Run("SuccessfulStreamCall", func(t *testing.T) {
		collector.Reset()

		handler := func(srv interface{}, stream grpc.ServerStream) error {
			// Simulate receiving and sending messages
			mockStream := stream.(*metricsServerStream)

			// Simulate receiving 3 messages
			for i := 0; i < 3; i++ {
				err := mockStream.RecvMsg(fmt.Sprintf("msg%d", i))
				if err != nil && err != io.EOF {
					return err
				}
			}

			// Simulate sending 2 messages
			for i := 0; i < 2; i++ {
				err := mockStream.SendMsg(fmt.Sprintf("response%d", i))
				if err != nil {
					return err
				}
			}

			return nil
		}

		info := &grpc.StreamServerInfo{
			FullMethod: "/test.TestService/TestStream",
		}

		ctx := context.Background()
		mockStream := newMockServerStream(ctx)
		// Add messages to receive
		for i := 0; i < 3; i++ {
			mockStream.addRecvMsg(fmt.Sprintf("msg%d", i))
		}

		err := streamInterceptor(nil, mockStream, info, handler)
		assert.NoError(t, err)

		// Verify metrics
		expectedLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestStream",
		}

		// Check server started
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_started_total", expectedLabels))

		// Check server handled with OK code
		handledLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestStream",
			"code":    codes.OK.String(),
		}
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_handled_total", handledLabels))

		// Check handling duration
		durations := collector.getHistogramValues("grpc_server_handling_seconds", expectedLabels)
		assert.Len(t, durations, 1)
		assert.Greater(t, durations[0], float64(0))

		// Check message counts - these are tracked via the wrapped stream
		assert.Equal(t, float64(3), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
		assert.Equal(t, float64(2), collector.getCounterValue("grpc_server_msg_sent_total", expectedLabels))
	})

	t.Run("StreamCallWithError", func(t *testing.T) {
		collector.Reset()

		expectedErr := status.Error(codes.Internal, "stream error")
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			return expectedErr
		}

		info := &grpc.StreamServerInfo{
			FullMethod: "/test.TestService/TestStream",
		}

		ctx := context.Background()
		mockStream := newMockServerStream(ctx)

		err := streamInterceptor(nil, mockStream, info, handler)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		// Verify error is tracked
		handledLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestStream",
			"code":    codes.Internal.String(),
		}
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_handled_total", handledLabels))
	})
}

func TestPrometheusGRPCInterceptor_ExcludeMethods(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		Enabled:        true,
		ServiceName:    "test-service",
		ExcludeMethods: []string{"/grpc.health.v1.Health/Check", "/test.TestService/ExcludedMethod"},
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)
	unaryInterceptor := interceptor.UnaryServerInterceptor()

	t.Run("ExcludedMethod", func(t *testing.T) {
		collector.Reset()

		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "response", nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/grpc.health.v1.Health/Check",
		}

		ctx := context.Background()
		resp, err := unaryInterceptor(ctx, "request", info, handler)

		assert.NoError(t, err)
		assert.Equal(t, "response", resp)

		// No metrics should be collected for excluded methods
		expectedLabels := map[string]string{
			"service": "test-service",
			"method":  "/grpc.health.v1.Health/Check",
		}

		assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_started_total", expectedLabels))
		assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
	})

	t.Run("IncludedMethod", func(t *testing.T) {
		collector.Reset()

		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "response", nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.TestService/IncludedMethod",
		}

		ctx := context.Background()
		resp, err := unaryInterceptor(ctx, "request", info, handler)

		assert.NoError(t, err)
		assert.Equal(t, "response", resp)

		// Metrics should be collected for included methods
		expectedLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/IncludedMethod",
		}

		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_started_total", expectedLabels))
	})
}

func TestPrometheusGRPCInterceptor_Disabled(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)
	unaryInterceptor := interceptor.UnaryServerInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/TestMethod",
	}

	ctx := context.Background()
	resp, err := unaryInterceptor(ctx, "request", info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)

	// No metrics should be collected when disabled
	expectedLabels := map[string]string{
		"service": "test-service",
		"method":  "/test.TestService/TestMethod",
	}

	assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_started_total", expectedLabels))
	assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
}

func TestPrometheusGRPCInterceptor_PanicRecovery(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)
	unaryInterceptor := interceptor.UnaryServerInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/TestMethod",
	}

	// Enable panics in collector
	collector.setPanics(true)

	ctx := context.Background()

	// Should not panic even if metrics collection panics
	assert.NotPanics(t, func() {
		resp, err := unaryInterceptor(ctx, "request", info, handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})
}

func TestPrometheusGRPCInterceptor_MessageSizeCollection(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		Enabled:            true,
		ServiceName:        "test-service",
		CollectMessageSize: false, // Disabled
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)
	unaryInterceptor := interceptor.UnaryServerInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/TestMethod",
	}

	ctx := context.Background()
	_, err := unaryInterceptor(ctx, "request", info, handler)
	assert.NoError(t, err)

	// Message size metrics should not be collected when disabled
	expectedLabels := map[string]string{
		"service": "test-service",
		"method":  "/test.TestService/TestMethod",
	}

	// Server started and handled should still be tracked
	assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_started_total", expectedLabels))

	// But message counts should not be tracked
	assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
	assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_sent_total", expectedLabels))
}

func TestPrometheusGRPCInterceptor_StreamMessageCollection(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		Enabled:               true,
		ServiceName:           "test-service",
		CollectStreamMessages: false, // Disabled
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)
	streamInterceptor := interceptor.StreamServerInterceptor()

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		// Try to send and receive messages
		wrappedStream := stream.(*metricsServerStream)

		_ = wrappedStream.SendMsg("response")
		_ = wrappedStream.RecvMsg("request")

		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.TestService/TestStream",
	}

	ctx := context.Background()
	mockStream := newMockServerStream(ctx)
	mockStream.addRecvMsg("request")

	err := streamInterceptor(nil, mockStream, info, handler)
	assert.NoError(t, err)

	// Stream message metrics should not be collected when disabled
	expectedLabels := map[string]string{
		"service": "test-service",
		"method":  "/test.TestService/TestStream",
	}

	assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
	assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_sent_total", expectedLabels))
}

func TestPrometheusGRPCInterceptor_UpdateConfig(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	interceptor := NewPrometheusGRPCInterceptor(collector, &PrometheusGRPCConfig{
		Enabled:               false,
		ExcludeMethods:        []string{"/health"},
		CollectMessageSize:    false,
		CollectStreamMessages: false,
	})

	newConfig := &PrometheusGRPCConfig{
		Enabled:               true,
		ExcludeMethods:        []string{"/health", "/metrics"},
		CollectMessageSize:    true,
		CollectStreamMessages: true,
	}

	interceptor.UpdateConfig(newConfig)

	config := interceptor.GetConfig()
	assert.True(t, config.Enabled)
	assert.Equal(t, []string{"/health", "/metrics"}, config.ExcludeMethods)
	assert.True(t, config.CollectMessageSize)
	assert.True(t, config.CollectStreamMessages)
}

func TestPrometheusGRPCInterceptor_UpdateConfigNil(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	originalConfig := &PrometheusGRPCConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, originalConfig)
	interceptor.UpdateConfig(nil)

	// Config should remain unchanged
	assert.Equal(t, originalConfig, interceptor.GetConfig())
}

func TestPrometheusGRPCInterceptor_GetMethods(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)

	assert.Equal(t, collector, interceptor.GetMetricsCollector())
}

func TestPrometheusGRPCInterceptor_ShouldExcludeMethod(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		ExcludeMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
			"/test.Service/ExcludedMethod",
		},
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)

	tests := []struct {
		method   string
		excluded bool
	}{
		{"/grpc.health.v1.Health/Check", true},
		{"/grpc.health.v1.Health/Watch", true},
		{"/test.Service/ExcludedMethod", true},
		{"/test.Service/IncludedMethod", false},
		{"/grpc.health.v1.Health/Status", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.excluded, interceptor.shouldExcludeMethod(tt.method), "method: %s", tt.method)
	}
}

func TestMetricsServerStream_SendRecvMsg(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		Enabled:               true,
		ServiceName:           "test-service",
		CollectStreamMessages: true,
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)

	ctx := context.Background()
	mockStream := newMockServerStream(ctx)

	wrappedStream := &metricsServerStream{
		ServerStream: mockStream,
		method:       "/test.TestService/TestStream",
		interceptor:  interceptor,
	}

	t.Run("SendMsg", func(t *testing.T) {
		collector.Reset()

		err := wrappedStream.SendMsg("message")
		assert.NoError(t, err)

		expectedLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestStream",
		}
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_msg_sent_total", expectedLabels))
		assert.Equal(t, int64(1), wrappedStream.sentCount)
	})

	t.Run("SendMsgWithError", func(t *testing.T) {
		collector.Reset()
		atomic.StoreInt64(&wrappedStream.sentCount, 0) // Reset sentCount from previous test

		mockStream.setSendError(errors.New("send error"))

		err := wrappedStream.SendMsg("message")
		assert.Error(t, err)

		// Should not increment counter on error
		expectedLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestStream",
		}
		assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_sent_total", expectedLabels))
		assert.Equal(t, int64(0), wrappedStream.sentCount) // Should remain 0 after error
	})

	t.Run("RecvMsg", func(t *testing.T) {
		collector.Reset()
		atomic.StoreInt64(&wrappedStream.receivedCount, 0) // Reset receivedCount from previous tests
		mockStream.setSendError(nil)    // Reset error

		mockStream.addRecvMsg("message")

		var msg interface{}
		err := wrappedStream.RecvMsg(&msg)
		assert.NoError(t, err)

		expectedLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestStream",
		}
		assert.Equal(t, float64(1), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
		assert.Equal(t, int64(1), wrappedStream.receivedCount)
	})

	t.Run("RecvMsgWithError", func(t *testing.T) {
		collector.Reset()
		atomic.StoreInt64(&wrappedStream.receivedCount, 0) // Reset receivedCount from previous tests

		mockStream.setRecvError(io.EOF)

		var msg interface{}
		err := wrappedStream.RecvMsg(&msg)
		assert.Error(t, err)
		assert.Equal(t, io.EOF, err)

		// Should not increment counter on error
		expectedLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestStream",
		}
		assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
		assert.Equal(t, int64(0), wrappedStream.receivedCount) // Should remain 0 after error
	})

	t.Run("StreamMessagesDisabled", func(t *testing.T) {
		disabledConfig := &PrometheusGRPCConfig{
			Enabled:               true,
			ServiceName:           "test-service",
			CollectStreamMessages: false,
		}
		disabledInterceptor := NewPrometheusGRPCInterceptor(collector, disabledConfig)

		wrappedStreamDisabled := &metricsServerStream{
			ServerStream: mockStream,
			method:       "/test.TestService/TestStream",
			interceptor:  disabledInterceptor,
		}

		collector.Reset()
		mockStream.setRecvError(nil) // Reset error
		mockStream.addRecvMsg("message")

		// Even with successful operations, metrics should not be collected
		err := wrappedStreamDisabled.SendMsg("message")
		assert.NoError(t, err)

		var msg interface{}
		err = wrappedStreamDisabled.RecvMsg(&msg)
		assert.NoError(t, err)

		expectedLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.TestService/TestStream",
		}
		assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_sent_total", expectedLabels))
		assert.Equal(t, float64(0), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
	})
}

func TestPrometheusGRPCInterceptor_ConcurrentOperations(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	config := &PrometheusGRPCConfig{
		Enabled:            true,
		ServiceName:        "test-service",
		CollectMessageSize: true,
	}

	interceptor := NewPrometheusGRPCInterceptor(collector, config)
	unaryInterceptor := interceptor.UnaryServerInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		time.Sleep(10 * time.Millisecond) // Simulate processing
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/TestMethod",
	}

	// Run concurrent requests
	const numRequests = 20
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx := context.Background()
			resp, err := unaryInterceptor(ctx, "request", info, handler)

			assert.NoError(t, err)
			assert.Equal(t, "response", resp)
		}()
	}

	wg.Wait()

	// Verify all requests were counted
	expectedLabels := map[string]string{
		"service": "test-service",
		"method":  "/test.TestService/TestMethod",
	}

	assert.Equal(t, float64(numRequests), collector.getCounterValue("grpc_server_started_total", expectedLabels))
	assert.Equal(t, float64(numRequests), collector.getCounterValue("grpc_server_msg_received_total", expectedLabels))
	assert.Equal(t, float64(numRequests), collector.getCounterValue("grpc_server_msg_sent_total", expectedLabels))
}

// Benchmark tests to verify <5% overhead requirement
func BenchmarkPrometheusGRPCInterceptor_UnaryEnabled(b *testing.B) {
	collector := newMockGRPCMetricsCollector()
	interceptor := NewPrometheusGRPCInterceptor(collector, &PrometheusGRPCConfig{
		Enabled:     true,
		ServiceName: "test-service",
	})

	unaryInterceptor := interceptor.UnaryServerInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/BenchmarkMethod",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = unaryInterceptor(ctx, "request", info, handler)
	}
}

func BenchmarkPrometheusGRPCInterceptor_UnaryDisabled(b *testing.B) {
	collector := newMockGRPCMetricsCollector()
	interceptor := NewPrometheusGRPCInterceptor(collector, &PrometheusGRPCConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})

	unaryInterceptor := interceptor.UnaryServerInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.TestService/BenchmarkMethod",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = unaryInterceptor(ctx, "request", info, handler)
	}
}

func BenchmarkPrometheusGRPCInterceptor_UnaryNoInterceptor(b *testing.B) {
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler(ctx, "request")
	}
}

func BenchmarkPrometheusGRPCInterceptor_StreamEnabled(b *testing.B) {
	collector := newMockGRPCMetricsCollector()
	interceptor := NewPrometheusGRPCInterceptor(collector, &PrometheusGRPCConfig{
		Enabled:     true,
		ServiceName: "test-service",
	})

	streamInterceptor := interceptor.StreamServerInterceptor()

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.TestService/BenchmarkStream",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockStream := newMockServerStream(ctx)
		_ = streamInterceptor(nil, mockStream, info, handler)
	}
}

func BenchmarkPrometheusGRPCInterceptor_StreamDisabled(b *testing.B) {
	collector := newMockGRPCMetricsCollector()
	interceptor := NewPrometheusGRPCInterceptor(collector, &PrometheusGRPCConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})

	streamInterceptor := interceptor.StreamServerInterceptor()

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.TestService/BenchmarkStream",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockStream := newMockServerStream(ctx)
		_ = streamInterceptor(nil, mockStream, info, handler)
	}
}

// TestPrometheusGRPCInterceptor_RaceConditionSafety tests thread safety under high concurrency
func TestPrometheusGRPCInterceptor_RaceConditionSafety(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	interceptor := NewPrometheusGRPCInterceptor(collector, &PrometheusGRPCConfig{
		Enabled:               true,
		ServiceName:           "test-service",
		CollectStreamMessages: true,
	})

	unaryInterceptor := interceptor.UnaryServerInterceptor()
	streamInterceptor := interceptor.StreamServerInterceptor()

	// Test concurrent unary calls
	t.Run("ConcurrentUnaryRequests", func(t *testing.T) {
		const numGoroutines = 100
		const callsPerGoroutine = 50
		var wg sync.WaitGroup

		// Mock unary handler
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			time.Sleep(time.Microsecond) // Simulate small processing delay
			return "response", nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.RaceTestService/ConcurrentMethod",
		}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < callsPerGoroutine; j++ {
					ctx := context.Background()
					req := fmt.Sprintf("request_%d_%d", goroutineID, j)

					_, err := unaryInterceptor(ctx, req, info, handler)
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		// Verify metrics were collected without race conditions
		totalCalls := numGoroutines * callsPerGoroutine

		// grpc_server_started_total doesn't include code
		startedLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.RaceTestService/ConcurrentMethod",
		}

		// grpc_server_handled_total includes code
		handledLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.RaceTestService/ConcurrentMethod",
			"code":    "OK",
		}

		assert.Equal(t, float64(totalCalls), collector.getCounterValue("grpc_server_started_total", startedLabels))
		assert.Equal(t, float64(totalCalls), collector.getCounterValue("grpc_server_handled_total", handledLabels))
	})

	// Test concurrent streaming calls (basic)
	t.Run("ConcurrentStreamingRequests", func(t *testing.T) {
		const numGoroutines = 50
		var wg sync.WaitGroup

		// Simple stream handler that doesn't send/receive messages
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			// Just simulate some processing without message exchange
			time.Sleep(time.Microsecond)
			return nil
		}

		info := &grpc.StreamServerInfo{
			FullMethod: "/test.RaceTestService/ConcurrentStreamMethod",
		}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				ctx := context.Background()
				mockStream := newMockServerStream(ctx)

				err := streamInterceptor(nil, mockStream, info, handler)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		// Verify streaming metrics were collected safely
		// grpc_server_started_total doesn't include code
		startedStreamLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.RaceTestService/ConcurrentStreamMethod",
		}

		// grpc_server_handled_total includes code
		handledStreamLabels := map[string]string{
			"service": "test-service",
			"method":  "/test.RaceTestService/ConcurrentStreamMethod",
			"code":    "OK",
		}

		assert.Equal(t, float64(numGoroutines), collector.getCounterValue("grpc_server_started_total", startedStreamLabels))
		assert.Equal(t, float64(numGoroutines), collector.getCounterValue("grpc_server_handled_total", handledStreamLabels))
	})
}

// TestPrometheusGRPCInterceptor_AtomicMessageCounting specifically tests atomic counter operations
func TestPrometheusGRPCInterceptor_AtomicMessageCounting(t *testing.T) {
	collector := newMockGRPCMetricsCollector()
	interceptor := NewPrometheusGRPCInterceptor(collector, &PrometheusGRPCConfig{
		Enabled:               true,
		ServiceName:           "test-service",
		CollectStreamMessages: true,
	})

	// Test direct atomic operations on the wrapped stream
	ctx := context.Background()
	mockStream := newMockServerStream(ctx)

	// Create wrapped stream directly
	wrappedStream := &metricsServerStream{
		ServerStream:  mockStream,
		method:        "/test.TestService/AtomicTest",
		interceptor:   interceptor,
		sentCount:     0,
		receivedCount: 0,
	}

	// Test concurrent message operations
	const numGoroutines = 100
	const messagesPerGoroutine = 50
	var wg sync.WaitGroup

	// Add messages to the mock stream's buffer for RecvMsg to consume
	totalMessages := numGoroutines * messagesPerGoroutine
	for i := 0; i < totalMessages; i++ {
		mockStream.addRecvMsg(fmt.Sprintf("message-%d", i))
	}

	// Test concurrent SendMsg operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				wrappedStream.SendMsg(&struct{}{})
			}
		}()
	}

	// Test concurrent RecvMsg operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				var msg interface{}
				wrappedStream.RecvMsg(&msg)
			}
		}()
	}

	wg.Wait()

	// Verify atomic counting worked correctly
	expectedMessages := int64(numGoroutines * messagesPerGoroutine)
	sentCount := atomic.LoadInt64(&wrappedStream.sentCount)
	receivedCount := atomic.LoadInt64(&wrappedStream.receivedCount)

	assert.Equal(t, expectedMessages, sentCount, "Sent message count should be accurate")
	assert.Equal(t, expectedMessages, receivedCount, "Received message count should be accurate")

	t.Logf("Successfully counted %d sent and %d received messages with atomic operations",
		sentCount, receivedCount)

	// Also verify the metrics were collected atomically
	msgLabels := map[string]string{
		"service": "test-service",
		"method":  "/test.TestService/AtomicTest",
	}

	metricSentCount := collector.getCounterValue("grpc_server_msg_sent_total", msgLabels)
	metricRecvCount := collector.getCounterValue("grpc_server_msg_received_total", msgLabels)

	assert.Equal(t, float64(expectedMessages), metricSentCount, "Metric sent count should match atomic count")
	assert.Equal(t, float64(expectedMessages), metricRecvCount, "Metric received count should match atomic count")
}
