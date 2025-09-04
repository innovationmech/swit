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

package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExporterFactory(t *testing.T) {
	factory := NewExporterFactory()
	assert.NotNil(t, factory)
}

func TestExporterFactory_CreateSpanExporter_Console(t *testing.T) {
	factory := NewExporterFactory()
	ctx := context.Background()

	config := ExporterConfig{
		Type: "console",
	}

	exporter, err := factory.CreateSpanExporter(ctx, config)
	require.NoError(t, err)
	assert.NotNil(t, exporter)

	// Clean up
	err = exporter.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestExporterFactory_CreateSpanExporter_Jaeger(t *testing.T) {
	factory := NewExporterFactory()
	ctx := context.Background()

	t.Run("with endpoint", func(t *testing.T) {
		config := ExporterConfig{
			Type:     "jaeger",
			Endpoint: "http://localhost:14268/api/traces",
		}

		exporter, err := factory.CreateSpanExporter(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, exporter)

		// Clean up
		err = exporter.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("with collector endpoint", func(t *testing.T) {
		config := ExporterConfig{
			Type: "jaeger",
			Jaeger: JaegerConfig{
				CollectorEndpoint: "http://localhost:14268/api/traces",
			},
		}

		exporter, err := factory.CreateSpanExporter(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, exporter)

		// Clean up
		err = exporter.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("with authentication", func(t *testing.T) {
		config := ExporterConfig{
			Type:     "jaeger",
			Endpoint: "http://localhost:14268/api/traces",
			Jaeger: JaegerConfig{
				Username: "admin",
				Password: "secret",
			},
		}

		exporter, err := factory.CreateSpanExporter(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, exporter)

		// Clean up
		err = exporter.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("with agent endpoint should fail", func(t *testing.T) {
		config := ExporterConfig{
			Type: "jaeger",
			Jaeger: JaegerConfig{
				AgentEndpoint: "localhost:6831",
			},
		}

		_, err := factory.CreateSpanExporter(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent endpoint is deprecated")
	})

	t.Run("default endpoint", func(t *testing.T) {
		config := ExporterConfig{
			Type: "jaeger",
		}

		exporter, err := factory.CreateSpanExporter(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, exporter)

		// Clean up
		err = exporter.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestExporterFactory_CreateSpanExporter_OTLP(t *testing.T) {
	factory := NewExporterFactory()
	ctx := context.Background()

	t.Run("HTTP endpoint", func(t *testing.T) {
		config := ExporterConfig{
			Type:     "otlp",
			Endpoint: "http://localhost:4318/v1/traces",
		}

		exporter, err := factory.CreateSpanExporter(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, exporter)

		// Clean up
		err = exporter.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("gRPC endpoint", func(t *testing.T) {
		config := ExporterConfig{
			Type:     "otlp",
			Endpoint: "localhost:4317",
		}

		exporter, err := factory.CreateSpanExporter(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, exporter)

		// Clean up
		err = exporter.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("with OTLP config endpoint", func(t *testing.T) {
		config := ExporterConfig{
			Type: "otlp",
			OTLP: OTLPConfig{
				Endpoint: "http://localhost:4318/v1/traces",
			},
		}

		exporter, err := factory.CreateSpanExporter(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, exporter)

		// Clean up
		err = exporter.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("with headers", func(t *testing.T) {
		config := ExporterConfig{
			Type:     "otlp",
			Endpoint: "http://localhost:4318/v1/traces",
			Headers: map[string]string{
				"Authorization": "Bearer token",
			},
			OTLP: OTLPConfig{
				Headers: map[string]string{
					"X-Custom": "value",
				},
			},
		}

		exporter, err := factory.CreateSpanExporter(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, exporter)

		// Clean up
		err = exporter.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("with insecure", func(t *testing.T) {
		config := ExporterConfig{
			Type:     "otlp",
			Endpoint: "http://localhost:4318/v1/traces",
			OTLP: OTLPConfig{
				Insecure: true,
			},
		}

		exporter, err := factory.CreateSpanExporter(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, exporter)

		// Clean up
		err = exporter.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("with compression", func(t *testing.T) {
		config := ExporterConfig{
			Type:     "otlp",
			Endpoint: "http://localhost:4318/v1/traces",
			OTLP: OTLPConfig{
				Compression: "gzip",
			},
		}

		exporter, err := factory.CreateSpanExporter(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, exporter)

		// Clean up
		err = exporter.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("without endpoint should fail", func(t *testing.T) {
		config := ExporterConfig{
			Type: "otlp",
		}

		_, err := factory.CreateSpanExporter(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "otlp exporter requires endpoint")
	})
}

func TestExporterFactory_CreateSpanExporter_Invalid(t *testing.T) {
	factory := NewExporterFactory()
	ctx := context.Background()

	config := ExporterConfig{
		Type: "invalid",
	}

	_, err := factory.CreateSpanExporter(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported exporter type")
}

func TestShouldUseHTTPProtocol(t *testing.T) {
	factory := &exporterFactory{}

	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "HTTP endpoint with /v1/traces",
			endpoint: "http://localhost:4318/v1/traces",
			expected: true,
		},
		{
			name:     "HTTPS endpoint with /v1/traces",
			endpoint: "https://api.example.com/v1/traces",
			expected: true,
		},
		{
			name:     "gRPC endpoint",
			endpoint: "localhost:4317",
			expected: false,
		},
		{
			name:     "HTTP endpoint without /v1/traces",
			endpoint: "http://localhost:4318",
			expected: false,
		},
		{
			name:     "empty endpoint",
			endpoint: "",
			expected: false,
		},
		{
			name:     "short endpoint",
			endpoint: "short",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := factory.shouldUseHTTPProtocol(tt.endpoint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateBatchSpanProcessor(t *testing.T) {
	factory := NewExporterFactory()
	ctx := context.Background()

	config := ExporterConfig{
		Type: "console",
	}

	exporter, err := factory.CreateSpanExporter(ctx, config)
	require.NoError(t, err)

	processor := CreateBatchSpanProcessor(exporter)
	assert.NotNil(t, processor)

	// Clean up
	err = processor.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestCreateSimpleSpanProcessor(t *testing.T) {
	factory := NewExporterFactory()
	ctx := context.Background()

	config := ExporterConfig{
		Type: "console",
	}

	exporter, err := factory.CreateSpanExporter(ctx, config)
	require.NoError(t, err)

	processor := CreateSimpleSpanProcessor(exporter)
	assert.NotNil(t, processor)

	// Clean up
	err = processor.Shutdown(ctx)
	assert.NoError(t, err)
}
