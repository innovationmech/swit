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

package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerConfig(t *testing.T) {
	config := NewServerConfig()

	// Test that defaults are set
	assert.Equal(t, "swit-service", config.ServiceName)
	assert.Equal(t, "8080", config.HTTP.Port)
	assert.True(t, config.HTTP.Enabled)
	assert.True(t, config.HTTP.EnableReady)
	assert.Equal(t, "9080", config.GRPC.Port)
	assert.True(t, config.GRPC.Enabled)
	assert.True(t, config.GRPC.EnableKeepalive)
	assert.True(t, config.GRPC.EnableReflection)
	assert.True(t, config.GRPC.EnableHealthService)
	assert.Equal(t, "127.0.0.1:8500", config.Discovery.Address)
	assert.Equal(t, "swit-service", config.Discovery.ServiceName)
	assert.True(t, config.Discovery.Enabled)
	assert.Equal(t, []string{"v1"}, config.Discovery.Tags)
	assert.Equal(t, DiscoveryFailureModeGraceful, config.Discovery.FailureMode)
	assert.False(t, config.Discovery.HealthCheckRequired)
	assert.Equal(t, 30*time.Second, config.Discovery.RegistrationTimeout)
	assert.True(t, config.Middleware.EnableCORS)
	assert.False(t, config.Middleware.EnableAuth)
	assert.False(t, config.Middleware.EnableRateLimit)
	assert.True(t, config.Middleware.EnableLogging)
	assert.Equal(t, 5*time.Second, config.ShutdownTimeout)

	// Test Prometheus defaults
	assert.True(t, config.Prometheus.Enabled)
	assert.Equal(t, "/metrics", config.Prometheus.Endpoint)
	assert.Equal(t, "swit", config.Prometheus.Namespace)
	assert.Equal(t, "server", config.Prometheus.Subsystem)
	assert.Equal(t, []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10}, config.Prometheus.Buckets.Duration)
	assert.Equal(t, []float64{100, 1000, 10000, 100000, 1000000}, config.Prometheus.Buckets.Size)
	assert.NotNil(t, config.Prometheus.Labels)
}

func TestServerConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		initial  *ServerConfig
		expected *ServerConfig
	}{
		{
			name:    "empty config",
			initial: &ServerConfig{},
			expected: &ServerConfig{
				ServiceName: "swit-service",
				HTTP: HTTPConfig{
					Port:         "8080",
					Address:      ":8080",
					Enabled:      true,
					EnableReady:  true,
					TestMode:     false,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
					Headers:      make(map[string]string),
					Middleware: HTTPMiddleware{
						EnableCORS:      true,
						EnableAuth:      false,
						EnableRateLimit: false,
						EnableLogging:   true,
						EnableTimeout:   true,
						CORSConfig: CORSConfig{
							AllowOrigins: []string{
								"http://localhost:3000",
								"http://localhost:8080",
								"http://127.0.0.1:3000",
								"http://127.0.0.1:8080",
							},
							AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
							AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
							AllowCredentials: true,
							MaxAge:           86400,
						},
						RateLimitConfig: RateLimitConfig{
							RequestsPerSecond: 100,
							BurstSize:         200,
							WindowSize:        time.Minute,
							KeyFunc:           "ip",
						},
						TimeoutConfig: TimeoutConfig{
							RequestTimeout: 30 * time.Second,
							HandlerTimeout: 25 * time.Second,
						},
						CustomHeaders: make(map[string]string),
					},
				},
				GRPC: GRPCConfig{
					Port:                "9080",
					Address:             ":9080",
					Enabled:             true,
					EnableKeepalive:     true,
					EnableReflection:    true,
					EnableHealthService: true,
					TestMode:            false,
					MaxRecvMsgSize:      4 * 1024 * 1024,
					MaxSendMsgSize:      4 * 1024 * 1024,
					KeepaliveParams: GRPCKeepaliveParams{
						MaxConnectionIdle:     15 * time.Second,
						MaxConnectionAge:      30 * time.Second,
						MaxConnectionAgeGrace: 5 * time.Second,
						Time:                  5 * time.Second,
						Timeout:               1 * time.Second,
					},
					KeepalivePolicy: GRPCKeepalivePolicy{
						MinTime:             5 * time.Second,
						PermitWithoutStream: true,
					},
					Interceptors: GRPCInterceptorConfig{
						EnableAuth:      false,
						EnableLogging:   true,
						EnableMetrics:   false,
						EnableRecovery:  true,
						EnableRateLimit: false,
					},
					TLS: GRPCTLSConfig{
						Enabled: false,
					},
				},
				Discovery: DiscoveryConfig{
					Address:             "127.0.0.1:8500",
					ServiceName:         "swit-service",
					Tags:                []string{"v1"},
					Enabled:             true,
					FailureMode:         DiscoveryFailureModeGraceful,
					HealthCheckRequired: false,
					RegistrationTimeout: 30 * time.Second,
				},
				Middleware: MiddlewareConfig{
					EnableCORS:      true,
					EnableAuth:      false,
					EnableRateLimit: false,
					EnableLogging:   true,
				},
				Sentry: SentryConfig{
					Enabled:              false,
					DSN:                  "",
					Environment:          "development",
					Release:              "",
					SampleRate:           1.0,
					TracesSampleRate:     0.0,
					AttachStacktrace:     true,
					EnableTracing:        false,
					Debug:                false,
					ServerName:           "",
					Tags:                 map[string]string{},
					BeforeSend:           false,
					IntegrateHTTP:        true,
					IntegrateGRPC:        true,
					CapturePanics:        true,
					MaxBreadcrumbs:       30,
					IgnoreErrors:         nil,
					HTTPIgnorePaths:      nil,
					HTTPIgnoreStatusCode: []int{400, 401, 403, 404},
				},
				Logging: LoggingConfig{
					Level:              "info",
					Development:        false,
					Encoding:           "json",
					OutputPaths:        []string{"stdout"},
					ErrorOutputPaths:   []string{"stderr"},
					DisableCaller:      false,
					DisableStacktrace:  false,
					SamplingEnabled:    false,
					SamplingInitial:    0,
					SamplingThereafter: 0,
				},
				Prometheus: PrometheusConfig{
					Enabled:   true,
					Endpoint:  "/metrics",
					Namespace: "swit",
					Subsystem: "server",
					Buckets: PrometheusBuckets{
						Duration: []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10},
						Size:     []float64{100, 1000, 10000, 100000, 1000000},
					},
					Labels:           make(map[string]string),
					CardinalityLimit: 10000,
				},
				ShutdownTimeout: 5 * time.Second,
			},
		},
		{
			name: "partial config with custom HTTP port",
			initial: &ServerConfig{
				ServiceName: "custom-service",
				HTTP: HTTPConfig{
					Port: "9000",
				},
			},
			expected: &ServerConfig{
				ServiceName: "custom-service",
				HTTP: HTTPConfig{
					Port:         "9000",
					Address:      ":9000",
					Enabled:      true,
					EnableReady:  true,
					TestMode:     false,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
					Headers:      make(map[string]string),
					Middleware: HTTPMiddleware{
						EnableCORS:      true,
						EnableAuth:      false,
						EnableRateLimit: false,
						EnableLogging:   true,
						EnableTimeout:   true,
						CORSConfig: CORSConfig{
							AllowOrigins: []string{
								"http://localhost:3000",
								"http://localhost:8080",
								"http://127.0.0.1:3000",
								"http://127.0.0.1:8080",
							},
							AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
							AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
							AllowCredentials: true,
							MaxAge:           86400,
						},
						RateLimitConfig: RateLimitConfig{
							RequestsPerSecond: 100,
							BurstSize:         200,
							WindowSize:        time.Minute,
							KeyFunc:           "ip",
						},
						TimeoutConfig: TimeoutConfig{
							RequestTimeout: 30 * time.Second,
							HandlerTimeout: 25 * time.Second,
						},
						CustomHeaders: make(map[string]string),
					},
				},
				GRPC: GRPCConfig{
					Port:                "10000", // 9000 + 1000
					Address:             ":10000",
					Enabled:             true,
					EnableKeepalive:     true,
					EnableReflection:    true,
					EnableHealthService: true,
					TestMode:            false,
					MaxRecvMsgSize:      4 * 1024 * 1024,
					MaxSendMsgSize:      4 * 1024 * 1024,
					KeepaliveParams: GRPCKeepaliveParams{
						MaxConnectionIdle:     15 * time.Second,
						MaxConnectionAge:      30 * time.Second,
						MaxConnectionAgeGrace: 5 * time.Second,
						Time:                  5 * time.Second,
						Timeout:               1 * time.Second,
					},
					KeepalivePolicy: GRPCKeepalivePolicy{
						MinTime:             5 * time.Second,
						PermitWithoutStream: true,
					},
					Interceptors: GRPCInterceptorConfig{
						EnableAuth:      false,
						EnableLogging:   true,
						EnableMetrics:   false,
						EnableRecovery:  true,
						EnableRateLimit: false,
					},
					TLS: GRPCTLSConfig{
						Enabled: false,
					},
				},
				Discovery: DiscoveryConfig{
					Address:             "127.0.0.1:8500",
					ServiceName:         "custom-service",
					Tags:                []string{"v1"},
					Enabled:             true,
					FailureMode:         DiscoveryFailureModeGraceful,
					HealthCheckRequired: false,
					RegistrationTimeout: 30 * time.Second,
				},
				Middleware: MiddlewareConfig{
					EnableCORS:      true,
					EnableAuth:      false,
					EnableRateLimit: false,
					EnableLogging:   true,
				},
				Sentry: SentryConfig{
					Enabled:              false,
					DSN:                  "",
					Environment:          "development",
					Release:              "",
					SampleRate:           1.0,
					TracesSampleRate:     0.0,
					AttachStacktrace:     true,
					EnableTracing:        false,
					Debug:                false,
					ServerName:           "",
					Tags:                 map[string]string{},
					BeforeSend:           false,
					IntegrateHTTP:        true,
					IntegrateGRPC:        true,
					CapturePanics:        true,
					MaxBreadcrumbs:       30,
					IgnoreErrors:         nil,
					HTTPIgnorePaths:      nil,
					HTTPIgnoreStatusCode: []int{400, 401, 403, 404},
				},
				Logging: LoggingConfig{
					Level:              "info",
					Development:        false,
					Encoding:           "json",
					OutputPaths:        []string{"stdout"},
					ErrorOutputPaths:   []string{"stderr"},
					DisableCaller:      false,
					DisableStacktrace:  false,
					SamplingEnabled:    false,
					SamplingInitial:    0,
					SamplingThereafter: 0,
				},
				Prometheus: PrometheusConfig{
					Enabled:   true,
					Endpoint:  "/metrics",
					Namespace: "swit",
					Subsystem: "server",
					Buckets: PrometheusBuckets{
						Duration: []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10},
						Size:     []float64{100, 1000, 10000, 100000, 1000000},
					},
					Labels:           make(map[string]string),
					CardinalityLimit: 10000,
				},
				ShutdownTimeout: 5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.SetDefaults()
			assert.Equal(t, tt.expected, tt.initial)
		})
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ServerConfig
		wantErr string
	}{
		{
			name:   "valid default config",
			config: NewServerConfig(),
		},
		{
			name: "missing service name",
			config: &ServerConfig{
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
			},
			wantErr: "service_name is required",
		},
		{
			name: "missing HTTP port when enabled",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP:        HTTPConfig{Enabled: true},
			},
			wantErr: "http.port is required when HTTP is enabled",
		},
		{
			name: "invalid HTTP port",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "invalid",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
			},
			wantErr: "http.port must be a valid port number",
		},
		{
			name: "HTTP port out of range",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "70000",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
			},
			wantErr: "http.port must be between 0 and 65535",
		},
		{
			name: "missing gRPC port when enabled",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				GRPC: GRPCConfig{Enabled: true},
			},
			wantErr: "grpc.port is required when gRPC is enabled",
		},
		{
			name: "invalid gRPC port",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				GRPC: GRPCConfig{
					Port:           "invalid",
					Enabled:        true,
					MaxRecvMsgSize: 4 * 1024 * 1024,
					MaxSendMsgSize: 4 * 1024 * 1024,
				},
			},
			wantErr: "grpc.port must be a valid port number",
		},
		{
			name: "both transports disabled",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP:        HTTPConfig{Enabled: false},
				GRPC:        GRPCConfig{Enabled: false},
			},
			wantErr: "at least one transport (HTTP or gRPC) must be enabled",
		},
		{
			name: "same port for both transports",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				GRPC: GRPCConfig{
					Port:           "8080",
					Enabled:        true,
					MaxRecvMsgSize: 4 * 1024 * 1024,
					MaxSendMsgSize: 4 * 1024 * 1024,
				},
			},
			wantErr: "http.port and grpc.port must be different",
		},
		{
			name: "missing discovery address when enabled",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Discovery: DiscoveryConfig{Enabled: true},
			},
			wantErr: "discovery.address is required when discovery is enabled",
		},
		{
			name: "missing discovery service name when enabled",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Discovery: DiscoveryConfig{Address: "localhost:8500", Enabled: true},
			},
			wantErr: "discovery.service_name is required when discovery is enabled",
		},
		{
			name: "invalid shutdown timeout",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				ShutdownTimeout: -1 * time.Second,
			},
			wantErr: "shutdown_timeout must be positive",
		},
		{
			name: "prometheus enabled with missing endpoint",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Prometheus: PrometheusConfig{
					Enabled:   true,
					Endpoint:  "", // Empty endpoint should fail
					Namespace: "swit",
					Subsystem: "server",
					Buckets: PrometheusBuckets{
						Duration: []float64{0.1, 1.0},
						Size:     []float64{100, 1000},
					},
				},
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr: "prometheus.endpoint cannot be empty when Prometheus is enabled",
		},
		{
			name: "prometheus endpoint without leading slash",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Prometheus: PrometheusConfig{
					Enabled:   true,
					Endpoint:  "metrics", // Missing leading slash
					Namespace: "swit",
					Subsystem: "server",
					Buckets: PrometheusBuckets{
						Duration: []float64{0.1, 1.0},
						Size:     []float64{100, 1000},
					},
				},
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr: "prometheus.endpoint must start with '/'",
		},
		{
			name: "prometheus buckets not sorted",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Prometheus: PrometheusConfig{
					Enabled:   true,
					Endpoint:  "/metrics",
					Namespace: "swit",
					Subsystem: "server",
					Buckets: PrometheusBuckets{
						Duration: []float64{1.0, 0.1}, // Not sorted
						Size:     []float64{100, 1000},
					},
				},
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr: "prometheus.buckets.duration must be sorted in ascending order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServerConfig_calculateGRPCPort(t *testing.T) {
	tests := []struct {
		name     string
		httpPort string
		expected string
	}{
		{
			name:     "standard port",
			httpPort: "8080",
			expected: "9080",
		},
		{
			name:     "custom port",
			httpPort: "9000",
			expected: "10000",
		},
		{
			name:     "empty port",
			httpPort: "",
			expected: "9080",
		},
		{
			name:     "invalid port",
			httpPort: "invalid",
			expected: "9080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ServerConfig{
				HTTP: HTTPConfig{Port: tt.httpPort},
			}
			result := config.calculateGRPCPort()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServerConfig_AddressMethods(t *testing.T) {
	config := &ServerConfig{
		HTTP: HTTPConfig{Port: "8080", Enabled: true},
		GRPC: GRPCConfig{Port: "9080", Enabled: true},
	}

	assert.Equal(t, ":8080", config.GetHTTPAddress())
	assert.Equal(t, ":9080", config.GetGRPCAddress())
	assert.True(t, config.IsHTTPEnabled())
	assert.True(t, config.IsGRPCEnabled())
	assert.False(t, config.IsDiscoveryEnabled()) // Default is false in this test

	// Test disabled transports
	config.HTTP.Enabled = false
	config.GRPC.Enabled = false

	assert.Equal(t, "", config.GetHTTPAddress())
	assert.Equal(t, "", config.GetGRPCAddress())
	assert.False(t, config.IsHTTPEnabled())
	assert.False(t, config.IsGRPCEnabled())
}

func TestServerConfig_DiscoveryEnabled(t *testing.T) {
	config := &ServerConfig{
		Discovery: DiscoveryConfig{Enabled: true},
	}

	assert.True(t, config.IsDiscoveryEnabled())

	config.Discovery.Enabled = false
	assert.False(t, config.IsDiscoveryEnabled())
}
