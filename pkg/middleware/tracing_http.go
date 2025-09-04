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
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/innovationmech/swit/pkg/tracing"
)

// HTTPTracingConfig holds configuration for HTTP tracing middleware
type HTTPTracingConfig struct {
	SkipPaths        []string                             // Paths to skip tracing
	RecordReqBody    bool                                 // Whether to record request body
	RecordRespBody   bool                                 // Whether to record response body
	MaxBodySize      int                                  // Maximum size of body to record
	CustomAttributes map[string]func(*gin.Context) string // Custom attribute extractors
}

// DefaultHTTPTracingConfig returns default HTTP tracing configuration
func DefaultHTTPTracingConfig() *HTTPTracingConfig {
	return &HTTPTracingConfig{
		SkipPaths:        []string{"/health", "/ready", "/metrics"},
		RecordReqBody:    false,
		RecordRespBody:   false,
		MaxBodySize:      4096,
		CustomAttributes: make(map[string]func(*gin.Context) string),
	}
}

// TracingMiddleware creates a Gin middleware that adds OpenTelemetry tracing
func TracingMiddleware(tm tracing.TracingManager) gin.HandlerFunc {
	return TracingMiddlewareWithConfig(tm, DefaultHTTPTracingConfig())
}

// TracingMiddlewareWithConfig creates a Gin middleware with custom configuration
func TracingMiddlewareWithConfig(tm tracing.TracingManager, config *HTTPTracingConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip tracing for configured paths
		for _, skipPath := range config.SkipPaths {
			if c.Request.URL.Path == skipPath {
				c.Next()
				return
			}
		}

		// Extract tracing context from headers
		ctx := tm.ExtractHTTPHeaders(c.Request.Header)
		c.Request = c.Request.WithContext(ctx)

		// Start a new span
		operationName := c.Request.Method + " " + c.FullPath()
		if c.FullPath() == "" {
			operationName = c.Request.Method + " " + c.Request.URL.Path
		}

		ctx, span := tm.StartSpan(
			ctx,
			operationName,
			tracing.WithSpanKind(trace.SpanKindServer),
			tracing.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Request.Method),
				semconv.URLFull(c.Request.URL.String()),
				semconv.URLScheme(c.Request.URL.Scheme),
				semconv.ServerAddress(c.Request.Host),
				semconv.URLPath(c.Request.URL.Path),
				semconv.UserAgentOriginal(c.Request.UserAgent()),
				semconv.HTTPRoute(c.FullPath()),
			),
		)
		defer span.End()

		// Update context in the Gin context
		c.Request = c.Request.WithContext(ctx)

		// Add additional attributes
		if c.Request.ContentLength > 0 {
			span.SetAttribute("http.request.content_length", c.Request.ContentLength)
		}

		if remoteAddr := c.ClientIP(); remoteAddr != "" {
			span.SetAttribute("http.client_ip", remoteAddr)
		}

		// Add query parameters
		if len(c.Request.URL.RawQuery) > 0 {
			span.SetAttribute("http.url.query", c.Request.URL.RawQuery)
		}

		// Record request body if configured
		if config.RecordReqBody && c.Request.ContentLength > 0 && c.Request.ContentLength <= int64(config.MaxBodySize) {
			if body := readRequestBody(c); body != "" {
				span.SetAttribute("http.request.body", body)
			}
		}

		// Add custom attributes
		for attrName, extractor := range config.CustomAttributes {
			if value := extractor(c); value != "" {
				span.SetAttribute(attrName, value)
			}
		}

		// Create a response writer wrapper to capture response data
		respWriter := &responseWriter{
			ResponseWriter: c.Writer,
			statusCode:     http.StatusOK,
			size:           0,
		}
		c.Writer = respWriter

		// Process the request
		c.Next()

		// Record response data
		statusCode := respWriter.statusCode
		span.SetAttribute(string(semconv.HTTPResponseStatusCodeKey), statusCode)
		span.SetAttribute("http.response.size", respWriter.size)

		// Set span status based on HTTP status code
		if statusCode >= 400 {
			span.SetStatus(codes.Error, http.StatusText(statusCode))

			// Add error information if available
			if len(c.Errors) > 0 {
				span.RecordError(c.Errors.Last())
				for _, err := range c.Errors {
					span.AddEvent("error", trace.WithAttributes(
						attribute.String("error.message", err.Error()),
						attribute.Int("error.type", int(err.Type)),
					))
				}
			}
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Record response body if configured
		if config.RecordRespBody && respWriter.size > 0 && respWriter.size <= int64(config.MaxBodySize) {
			if body := string(respWriter.body); body != "" {
				span.SetAttribute("http.response.body", body)
			}
		}

		// Inject tracing context into response headers
		tm.InjectHTTPHeaders(ctx, c.Writer.Header())
	}
}

// HTTPClientTracingRoundTripper returns an HTTP round tripper with tracing enabled
func HTTPClientTracingRoundTripper(tm tracing.TracingManager, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &tracingRoundTripper{
		base: base,
		tm:   tm,
	}
}

// tracingRoundTripper wraps an http.RoundTripper with tracing
type tracingRoundTripper struct {
	base http.RoundTripper
	tm   tracing.TracingManager
}

// RoundTrip implements http.RoundTripper
func (t *tracingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	operationName := req.Method + " " + req.URL.Path
	ctx, span := t.tm.StartSpan(
		ctx,
		operationName,
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			semconv.HTTPRequestMethodKey.String(req.Method),
			semconv.URLFull(req.URL.String()),
			semconv.URLScheme(req.URL.Scheme),
			semconv.ServerAddress(req.Host),
			semconv.URLPath(req.URL.Path),
		),
	)
	defer span.End()

	// Inject tracing headers
	t.tm.InjectHTTPHeaders(ctx, req.Header)

	// Make the request with traced context
	req = req.WithContext(ctx)
	resp, err := t.base.RoundTrip(req)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return resp, err
	}

	// Record response status
	span.SetAttribute(string(semconv.HTTPResponseStatusCodeKey), resp.StatusCode)
	if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return resp, nil
}

// responseWriter wraps gin.ResponseWriter to capture response data
type responseWriter struct {
	gin.ResponseWriter
	statusCode int
	size       int64
	body       []byte
}

// Write captures the response body
func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += int64(n)
	if len(w.body)+len(b) <= 4096 { // Limit body capture to 4KB
		w.body = append(w.body, b...)
	}
	return n, err
}

// WriteHeader captures the status code
func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// readRequestBody attempts to read the request body for tracing
func readRequestBody(c *gin.Context) string {
	if body, exists := c.Get(gin.BodyBytesKey); exists {
		if bodyBytes, ok := body.([]byte); ok {
			return string(bodyBytes)
		}
	}
	return ""
}
