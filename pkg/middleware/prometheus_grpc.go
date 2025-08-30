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
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/types"
)

// PrometheusGRPCConfig configures the gRPC interceptor for Prometheus metrics
type PrometheusGRPCConfig struct {
	// Enabled controls whether metrics collection is active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// ExcludeMethods contains gRPC methods to exclude from metrics
	ExcludeMethods []string `yaml:"exclude_methods" json:"exclude_methods"`

	// CollectMessageSize enables message size metrics collection
	CollectMessageSize bool `yaml:"collect_message_size" json:"collect_message_size"`

	// CollectStreamMessages enables per-message metrics for streaming RPCs
	CollectStreamMessages bool `yaml:"collect_stream_messages" json:"collect_stream_messages"`

	// ServiceName is the service name label for metrics
	ServiceName string `yaml:"service_name" json:"service_name"`

	// HandleStreamingErrors controls whether to track streaming errors separately
	HandleStreamingErrors bool `yaml:"handle_streaming_errors" json:"handle_streaming_errors"`
}

// DefaultPrometheusGRPCConfig returns sensible defaults for gRPC metrics interceptor
func DefaultPrometheusGRPCConfig() *PrometheusGRPCConfig {
	return &PrometheusGRPCConfig{
		Enabled: true,
		ExcludeMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
		},
		CollectMessageSize:    true,
		CollectStreamMessages: true,
		ServiceName:           "unknown",
		HandleStreamingErrors: true,
	}
}

// PrometheusGRPCInterceptor provides gRPC metrics collection interceptors
type PrometheusGRPCInterceptor struct {
	config           *PrometheusGRPCConfig
	metricsCollector types.MetricsCollector
}

// NewPrometheusGRPCInterceptor creates a new gRPC metrics interceptor
func NewPrometheusGRPCInterceptor(collector types.MetricsCollector, config *PrometheusGRPCConfig) *PrometheusGRPCInterceptor {
	if config == nil {
		config = DefaultPrometheusGRPCConfig()
	}

	return &PrometheusGRPCInterceptor{
		config:           config,
		metricsCollector: collector,
	}
}

// UnaryServerInterceptor returns a grpc.UnaryServerInterceptor for metrics collection
func (i *PrometheusGRPCInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip if metrics collection is disabled
		if !i.config.Enabled {
			return handler(ctx, req)
		}

		// Check if this method should be excluded
		if i.shouldExcludeMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Start timing and record start
		start := time.Now()

		// Record server started
		i.recordServerStarted(info.FullMethod)

		// Execute the RPC
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Collect metrics
		i.collectUnaryMetrics(info.FullMethod, duration, req, resp, err)

		return resp, err
	}
}

// StreamServerInterceptor returns a grpc.StreamServerInterceptor for metrics collection
func (i *PrometheusGRPCInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip if metrics collection is disabled
		if !i.config.Enabled {
			return handler(srv, stream)
		}

		// Check if this method should be excluded
		if i.shouldExcludeMethod(info.FullMethod) {
			return handler(srv, stream)
		}

		// Start timing and record start
		start := time.Now()

		// Record server started
		i.recordServerStarted(info.FullMethod)

		// Wrap the stream for message counting
		wrappedStream := &metricsServerStream{
			ServerStream: stream,
			method:       info.FullMethod,
			interceptor:  i,
		}

		// Execute the RPC
		err := handler(srv, wrappedStream)

		// Calculate duration
		duration := time.Since(start)

		// Collect metrics
		i.collectStreamMetrics(info.FullMethod, duration, wrappedStream.sentCount, wrappedStream.receivedCount, err)

		return err
	}
}

// metricsServerStream wraps grpc.ServerStream to collect message metrics
type metricsServerStream struct {
	grpc.ServerStream
	method        string
	interceptor   *PrometheusGRPCInterceptor
	sentCount     int
	receivedCount int
}

func (s *metricsServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil && s.interceptor.config.CollectStreamMessages {
		s.sentCount++
		s.interceptor.recordMessageSent(s.method)
	}
	return err
}

func (s *metricsServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil && s.interceptor.config.CollectStreamMessages {
		s.receivedCount++
		s.interceptor.recordMessageReceived(s.method)
	}
	return err
}

// collectUnaryMetrics collects metrics for unary RPCs
func (i *PrometheusGRPCInterceptor) collectUnaryMetrics(method string, duration time.Duration, req, resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Error("gRPC unary metrics collection panic",
				zap.String("method", method),
				zap.Any("panic", r))
		}
	}()

	// Get status code
	code := codes.OK
	if err != nil {
		if st, ok := status.FromError(err); ok {
			code = st.Code()
		} else {
			code = codes.Unknown
		}
	}

	labels := map[string]string{
		"service": i.config.ServiceName,
		"method":  method,
	}

	// Record completion
	completionLabels := make(map[string]string)
	for k, v := range labels {
		completionLabels[k] = v
	}
	completionLabels["code"] = code.String()

	i.safeIncrementCounter("grpc_server_handled_total", completionLabels)

	// Record handling duration
	i.safeObserveHistogram("grpc_server_handling_seconds", duration.Seconds(), labels)

	// Record message counts for unary (1 each)
	if i.config.CollectMessageSize {
		i.safeIncrementCounter("grpc_server_msg_received_total", labels)
		if err == nil {
			i.safeIncrementCounter("grpc_server_msg_sent_total", labels)
		}
	}
}

// collectStreamMetrics collects metrics for streaming RPCs
func (i *PrometheusGRPCInterceptor) collectStreamMetrics(method string, duration time.Duration, sentCount, receivedCount int, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Error("gRPC stream metrics collection panic",
				zap.String("method", method),
				zap.Any("panic", r))
		}
	}()

	// Get status code
	code := codes.OK
	if err != nil {
		if st, ok := status.FromError(err); ok {
			code = st.Code()
		} else {
			code = codes.Unknown
		}
	}

	labels := map[string]string{
		"service": i.config.ServiceName,
		"method":  method,
	}

	// Record completion
	completionLabels := make(map[string]string)
	for k, v := range labels {
		completionLabels[k] = v
	}
	completionLabels["code"] = code.String()

	i.safeIncrementCounter("grpc_server_handled_total", completionLabels)

	// Record handling duration
	i.safeObserveHistogram("grpc_server_handling_seconds", duration.Seconds(), labels)
}

// recordServerStarted records when an RPC starts
func (i *PrometheusGRPCInterceptor) recordServerStarted(method string) {
	labels := map[string]string{
		"service": i.config.ServiceName,
		"method":  method,
	}
	i.safeIncrementCounter("grpc_server_started_total", labels)
}

// recordMessageSent records when a message is sent
func (i *PrometheusGRPCInterceptor) recordMessageSent(method string) {
	labels := map[string]string{
		"service": i.config.ServiceName,
		"method":  method,
	}
	i.safeIncrementCounter("grpc_server_msg_sent_total", labels)
}

// recordMessageReceived records when a message is received
func (i *PrometheusGRPCInterceptor) recordMessageReceived(method string) {
	labels := map[string]string{
		"service": i.config.ServiceName,
		"method":  method,
	}
	i.safeIncrementCounter("grpc_server_msg_received_total", labels)
}

// shouldExcludeMethod checks if a method should be excluded from metrics
func (i *PrometheusGRPCInterceptor) shouldExcludeMethod(method string) bool {
	for _, excludeMethod := range i.config.ExcludeMethods {
		if method == excludeMethod {
			return true
		}
	}
	return false
}

// Safe metric collection methods with error handling

func (i *PrometheusGRPCInterceptor) safeIncrementCounter(name string, labels map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Debug("gRPC counter increment panic",
				zap.String("metric", name),
				zap.Any("panic", r))
		}
	}()

	if i.metricsCollector != nil {
		i.metricsCollector.IncrementCounter(name, labels)
	}
}

func (i *PrometheusGRPCInterceptor) safeObserveHistogram(name string, value float64, labels map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Debug("gRPC histogram observation panic",
				zap.String("metric", name),
				zap.Float64("value", value),
				zap.Any("panic", r))
		}
	}()

	if i.metricsCollector != nil {
		i.metricsCollector.ObserveHistogram(name, value, labels)
	}
}

func (i *PrometheusGRPCInterceptor) safeIncrementGauge(name string, labels map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Debug("gRPC gauge increment panic",
				zap.String("metric", name),
				zap.Any("panic", r))
		}
	}()

	if i.metricsCollector != nil {
		i.metricsCollector.IncrementGauge(name, labels)
	}
}

// GetConfig returns the interceptor configuration
func (i *PrometheusGRPCInterceptor) GetConfig() *PrometheusGRPCConfig {
	return i.config
}

// UpdateConfig updates the interceptor configuration (thread-safe for basic fields)
func (i *PrometheusGRPCInterceptor) UpdateConfig(config *PrometheusGRPCConfig) {
	if config == nil {
		return
	}

	// Only update fields that are safe to change at runtime
	i.config.Enabled = config.Enabled
	i.config.ExcludeMethods = config.ExcludeMethods
	i.config.CollectMessageSize = config.CollectMessageSize
	i.config.CollectStreamMessages = config.CollectStreamMessages
}

// GetMetricsCollector returns the underlying metrics collector
func (i *PrometheusGRPCInterceptor) GetMetricsCollector() types.MetricsCollector {
	return i.metricsCollector
}
