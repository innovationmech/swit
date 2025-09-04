package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

// ExporterFactory defines the interface for creating span exporters
type ExporterFactory interface {
	CreateSpanExporter(ctx context.Context, config ExporterConfig) (trace.SpanExporter, error)
}

// exporterFactory implements the ExporterFactory interface
type exporterFactory struct{}

// NewExporterFactory creates a new exporter factory
func NewExporterFactory() ExporterFactory {
	return &exporterFactory{}
}

// CreateSpanExporter creates a span exporter based on the configuration
func (f *exporterFactory) CreateSpanExporter(ctx context.Context, config ExporterConfig) (trace.SpanExporter, error) {
	switch config.Type {
	case "console":
		return f.createConsoleExporter(config)
	case "jaeger":
		return f.createJaegerExporter(config)
	case "otlp":
		return f.createOTLPExporter(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", config.Type)
	}
}

// createConsoleExporter creates a console/stdout exporter
func (f *exporterFactory) createConsoleExporter(config ExporterConfig) (trace.SpanExporter, error) {
	options := []stdouttrace.Option{
		stdouttrace.WithPrettyPrint(),
	}

	exporter, err := stdouttrace.New(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create console exporter: %w", err)
	}

	return exporter, nil
}

// createJaegerExporter creates a Jaeger exporter using OTLP (deprecated Jaeger exporter)
func (f *exporterFactory) createJaegerExporter(config ExporterConfig) (trace.SpanExporter, error) {
	// Since the Jaeger exporter is deprecated, we'll use OTLP to send to Jaeger
	// Jaeger supports OTLP ingestion on port 4318 (HTTP) and 4317 (gRPC)

	endpoint := config.Endpoint
	if endpoint == "" && config.Jaeger.CollectorEndpoint != "" {
		endpoint = config.Jaeger.CollectorEndpoint
	}
	if endpoint == "" && config.Jaeger.AgentEndpoint != "" {
		return nil, fmt.Errorf("jaeger agent endpoint is deprecated, please use collector endpoint")
	}
	if endpoint == "" {
		// Default to local Jaeger OTLP HTTP endpoint
		endpoint = "http://localhost:4318/v1/traces"
	}

	// Convert to OTLP config for Jaeger
	otlpConfig := config
	otlpConfig.Type = "otlp"
	otlpConfig.Endpoint = endpoint

	// Set up authentication headers if provided
	if config.Jaeger.Username != "" && config.Jaeger.Password != "" {
		if otlpConfig.OTLP.Headers == nil {
			otlpConfig.OTLP.Headers = make(map[string]string)
		}
		// Use basic auth header (this is a simplified example)
		otlpConfig.OTLP.Headers["Authorization"] = fmt.Sprintf("Basic %s:%s", config.Jaeger.Username, config.Jaeger.Password)
	}

	return f.createOTLPExporter(context.Background(), otlpConfig)
}

// createOTLPExporter creates an OTLP exporter (HTTP or gRPC)
func (f *exporterFactory) createOTLPExporter(ctx context.Context, config ExporterConfig) (trace.SpanExporter, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = config.OTLP.Endpoint
	}
	if endpoint == "" {
		return nil, fmt.Errorf("otlp exporter requires endpoint")
	}

	// Determine if this should be HTTP or gRPC based on endpoint or explicit configuration
	useHTTP := f.shouldUseHTTPProtocol(endpoint)

	if useHTTP {
		return f.createOTLPHTTPExporter(ctx, config, endpoint)
	}
	return f.createOTLPGRPCExporter(ctx, config, endpoint)
}

// shouldUseHTTPProtocol determines if HTTP protocol should be used for OTLP
func (f *exporterFactory) shouldUseHTTPProtocol(endpoint string) bool {
	// Simple heuristic: if endpoint contains /v1/traces, assume HTTP
	// This is the standard OTLP HTTP endpoint path
	if len(endpoint) < 10 {
		return false
	}
	return endpoint[len(endpoint)-10:] == "/v1/traces"
}

// createOTLPHTTPExporter creates an OTLP HTTP exporter
func (f *exporterFactory) createOTLPHTTPExporter(ctx context.Context, config ExporterConfig, endpoint string) (trace.SpanExporter, error) {
	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithTimeout(config.GetTimeout()),
	}

	// Configure insecure connection if specified
	if config.OTLP.Insecure {
		options = append(options, otlptracehttp.WithInsecure())
	}

	// Configure headers
	headers := make(map[string]string)
	for k, v := range config.Headers {
		headers[k] = v
	}
	for k, v := range config.OTLP.Headers {
		headers[k] = v
	}
	if len(headers) > 0 {
		options = append(options, otlptracehttp.WithHeaders(headers))
	}

	// Configure compression
	if config.OTLP.Compression == "gzip" {
		options = append(options, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
	} else if config.OTLP.Compression == "none" {
		options = append(options, otlptracehttp.WithCompression(otlptracehttp.NoCompression))
	}

	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp http exporter: %w", err)
	}

	return exporter, nil
}

// createOTLPGRPCExporter creates an OTLP gRPC exporter
func (f *exporterFactory) createOTLPGRPCExporter(ctx context.Context, config ExporterConfig, endpoint string) (trace.SpanExporter, error) {
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithTimeout(config.GetTimeout()),
	}

	// Configure insecure connection if specified
	if config.OTLP.Insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	// Configure headers
	headers := make(map[string]string)
	for k, v := range config.Headers {
		headers[k] = v
	}
	for k, v := range config.OTLP.Headers {
		headers[k] = v
	}
	if len(headers) > 0 {
		options = append(options, otlptracegrpc.WithHeaders(headers))
	}

	// Configure compression
	if config.OTLP.Compression == "gzip" {
		options = append(options, otlptracegrpc.WithCompressor("gzip"))
	}

	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp grpc exporter: %w", err)
	}

	return exporter, nil
}

// CreateBatchSpanProcessor creates a batch span processor with reasonable defaults
func CreateBatchSpanProcessor(exporter trace.SpanExporter) trace.SpanProcessor {
	return trace.NewBatchSpanProcessor(exporter,
		trace.WithBatchTimeout(5000), // 5 seconds
		trace.WithMaxExportBatchSize(512),
		trace.WithMaxQueueSize(2048),
	)
}

// CreateSimpleSpanProcessor creates a simple span processor for debugging
func CreateSimpleSpanProcessor(exporter trace.SpanExporter) trace.SpanProcessor {
	return trace.NewSimpleSpanProcessor(exporter)
}
