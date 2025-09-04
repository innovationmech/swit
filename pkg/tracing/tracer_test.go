package tracing

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTracingManager(t *testing.T) {
	tm := NewTracingManager()
	assert.NotNil(t, tm)
}

func TestTracingManager_Initialize(t *testing.T) {
	tests := []struct {
		name    string
		config  *TracingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "tracing config cannot be nil",
		},
		{
			name: "invalid config",
			config: &TracingConfig{
				Enabled:     true,
				ServiceName: "", // missing service name
			},
			wantErr: true,
			errMsg:  "invalid tracing config",
		},
		{
			name: "disabled config",
			config: &TracingConfig{
				Enabled:     false,
				ServiceName: "test-service",
			},
			wantErr: false,
		},
		{
			name: "valid console config",
			config: &TracingConfig{
				Enabled:     true,
				ServiceName: "test-service",
				Sampling: SamplingConfig{
					Type: "always_on",
				},
				Exporter: ExporterConfig{
					Type: "console",
				},
				Propagators: []string{"tracecontext"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTracingManager()
			ctx := context.Background()

			err := tm.Initialize(ctx, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTracingManager_StartSpan(t *testing.T) {
	tm := NewTracingManager()
	config := &TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
		Sampling: SamplingConfig{
			Type: "always_on",
		},
		Exporter: ExporterConfig{
			Type: "console",
		},
	}

	ctx := context.Background()
	err := tm.Initialize(ctx, config)
	require.NoError(t, err)

	t.Run("basic span creation", func(t *testing.T) {
		newCtx, span := tm.StartSpan(ctx, "test-operation")
		assert.NotNil(t, newCtx)
		assert.NotNil(t, span)
		span.End()
	})

	t.Run("span with options", func(t *testing.T) {
		newCtx, span := tm.StartSpan(ctx, "test-operation",
			WithSpanKind(trace.SpanKindServer),
			WithAttributes(
				attribute.String("test.key", "test.value"),
			),
		)
		assert.NotNil(t, newCtx)
		assert.NotNil(t, span)
		span.End()
	})
}

func TestTracingManager_SpanFromContext(t *testing.T) {
	tm := NewTracingManager()
	config := &TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
		Sampling: SamplingConfig{
			Type: "always_on",
		},
		Exporter: ExporterConfig{
			Type: "console",
		},
	}

	ctx := context.Background()
	err := tm.Initialize(ctx, config)
	require.NoError(t, err)

	t.Run("span from context", func(t *testing.T) {
		newCtx, span1 := tm.StartSpan(ctx, "test-operation")
		span2 := tm.SpanFromContext(newCtx)
		assert.NotNil(t, span2)
		span1.End()
	})

	t.Run("no span in context", func(t *testing.T) {
		emptyCtx := context.Background()
		span := tm.SpanFromContext(emptyCtx)
		assert.NotNil(t, span)
		// Should return no-op span
	})
}

func TestTracingManager_HTTPHeaders(t *testing.T) {
	tm := NewTracingManager()
	config := &TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
		Sampling: SamplingConfig{
			Type: "always_on",
		},
		Exporter: ExporterConfig{
			Type: "console",
		},
		Propagators: []string{"tracecontext"},
	}

	ctx := context.Background()
	err := tm.Initialize(ctx, config)
	require.NoError(t, err)

	t.Run("inject and extract headers", func(t *testing.T) {
		ctx, span := tm.StartSpan(ctx, "test-operation")
		defer span.End()

		headers := make(http.Header)
		tm.InjectHTTPHeaders(ctx, headers)

		// Headers should be populated
		assert.NotEmpty(t, headers)

		// Extract context
		extractedCtx := tm.ExtractHTTPHeaders(headers)
		assert.NotNil(t, extractedCtx)
	})
}

func TestTracingManager_Shutdown(t *testing.T) {
	tm := NewTracingManager()

	t.Run("shutdown uninitialized manager", func(t *testing.T) {
		err := tm.Shutdown(context.Background())
		assert.NoError(t, err)
	})

	t.Run("shutdown initialized manager", func(t *testing.T) {
		config := &TracingConfig{
			Enabled:     true,
			ServiceName: "test-service",
			Sampling: SamplingConfig{
				Type: "always_on",
			},
			Exporter: ExporterConfig{
				Type: "console",
			},
		}

		ctx := context.Background()
		err := tm.Initialize(ctx, config)
		require.NoError(t, err)

		err = tm.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSpanWrapper(t *testing.T) {
	tm := NewTracingManager()
	config := &TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
		Sampling: SamplingConfig{
			Type: "always_on",
		},
		Exporter: ExporterConfig{
			Type: "console",
		},
	}

	ctx := context.Background()
	err := tm.Initialize(ctx, config)
	require.NoError(t, err)

	ctx, span := tm.StartSpan(ctx, "test-operation")
	defer span.End()

	t.Run("set attribute string", func(t *testing.T) {
		span.SetAttribute("test.string", "value")
	})

	t.Run("set attribute int", func(t *testing.T) {
		span.SetAttribute("test.int", 42)
	})

	t.Run("set attribute int64", func(t *testing.T) {
		span.SetAttribute("test.int64", int64(42))
	})

	t.Run("set attribute float64", func(t *testing.T) {
		span.SetAttribute("test.float64", 42.5)
	})

	t.Run("set attribute bool", func(t *testing.T) {
		span.SetAttribute("test.bool", true)
	})

	t.Run("set attribute other type", func(t *testing.T) {
		span.SetAttribute("test.other", []string{"a", "b"})
	})

	t.Run("set attributes multiple", func(t *testing.T) {
		span.SetAttributes(
			attribute.String("multi.key1", "value1"),
			attribute.String("multi.key2", "value2"),
		)
	})

	t.Run("add event", func(t *testing.T) {
		span.AddEvent("test-event")
	})

	t.Run("set status", func(t *testing.T) {
		span.SetStatus(codes.Ok, "success")
	})

	t.Run("record error", func(t *testing.T) {
		span.RecordError(assert.AnError)
	})

	t.Run("span context", func(t *testing.T) {
		spanCtx := span.SpanContext()
		assert.NotNil(t, spanCtx)
	})
}

func TestNoOpSpan(t *testing.T) {
	span := &noOpSpan{}

	t.Run("all methods should not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			span.SetAttribute("key", "value")
			span.SetAttributes(attribute.String("key", "value"))
			span.AddEvent("event")
			span.SetStatus(codes.Ok, "ok")
			span.End()
			span.RecordError(assert.AnError)
			spanCtx := span.SpanContext()
			assert.False(t, spanCtx.IsValid())
		})
	})
}

func TestSpanOptions(t *testing.T) {
	t.Run("WithSpanKind", func(t *testing.T) {
		config := &spanConfig{}
		opt := WithSpanKind(trace.SpanKindServer)
		opt(config)
		assert.Equal(t, trace.SpanKindServer, config.spanKind)
	})

	t.Run("WithAttributes", func(t *testing.T) {
		config := &spanConfig{}
		attrs := []attribute.KeyValue{
			attribute.String("key1", "value1"),
			attribute.String("key2", "value2"),
		}
		opt := WithAttributes(attrs...)
		opt(config)
		assert.Equal(t, attrs, config.attributes)
	})
}

func TestTracingManagerDisabled(t *testing.T) {
	tm := NewTracingManager()
	config := &TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	ctx := context.Background()
	err := tm.Initialize(ctx, config)
	require.NoError(t, err)

	t.Run("disabled tracing should still work", func(t *testing.T) {
		ctx, span := tm.StartSpan(ctx, "test-operation")
		assert.NotNil(t, span)

		span.SetAttribute("test", "value")
		span.AddEvent("test-event")
		span.SetStatus(codes.Ok, "ok")
		span.End()

		headers := make(http.Header)
		tm.InjectHTTPHeaders(ctx, headers)

		extractedCtx := tm.ExtractHTTPHeaders(headers)
		assert.NotNil(t, extractedCtx)

		err := tm.Shutdown(ctx)
		assert.NoError(t, err)
	})
}
