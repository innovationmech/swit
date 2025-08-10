// Copyright 2024 Swit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

// DeepCopy creates a deep copy of the ServerConfig to prevent shared references
// between different factory instances. This ensures that modifications to one
// configuration instance don't affect others.
func (c *ServerConfig) DeepCopy() *ServerConfig {
	if c == nil {
		return nil
	}

	copy := &ServerConfig{
		ServiceName:     c.ServiceName,
		ShutdownTimeout: c.ShutdownTimeout,
	}

	// Deep copy HTTP configuration
	copy.HTTP = c.deepCopyHTTPConfig()

	// Deep copy GRPC configuration
	copy.GRPC = c.deepCopyGRPCConfig()

	// Deep copy Discovery configuration
	copy.Discovery = c.deepCopyDiscoveryConfig()

	// Deep copy Middleware configuration
	copy.Middleware = c.Middleware // Simple struct, no nested pointers

	return copy
}

// deepCopyHTTPConfig creates a deep copy of HTTPConfig
func (c *ServerConfig) deepCopyHTTPConfig() HTTPConfig {
	httpCopy := HTTPConfig{
		Port:         c.HTTP.Port,
		Address:      c.HTTP.Address,
		EnableReady:  c.HTTP.EnableReady,
		Enabled:      c.HTTP.Enabled,
		TestMode:     c.HTTP.TestMode,
		TestPort:     c.HTTP.TestPort,
		ReadTimeout:  c.HTTP.ReadTimeout,
		WriteTimeout: c.HTTP.WriteTimeout,
		IdleTimeout:  c.HTTP.IdleTimeout,
	}

	// Deep copy Headers map
	if c.HTTP.Headers != nil {
		httpCopy.Headers = make(map[string]string, len(c.HTTP.Headers))
		for k, v := range c.HTTP.Headers {
			httpCopy.Headers[k] = v
		}
	}

	// Deep copy HTTPMiddleware
	httpCopy.Middleware = c.deepCopyHTTPMiddleware()

	return httpCopy
}

// deepCopyHTTPMiddleware creates a deep copy of HTTPMiddleware
func (c *ServerConfig) deepCopyHTTPMiddleware() HTTPMiddleware {
	middlewareCopy := HTTPMiddleware{
		EnableCORS:      c.HTTP.Middleware.EnableCORS,
		EnableAuth:      c.HTTP.Middleware.EnableAuth,
		EnableRateLimit: c.HTTP.Middleware.EnableRateLimit,
		EnableLogging:   c.HTTP.Middleware.EnableLogging,
		EnableTimeout:   c.HTTP.Middleware.EnableTimeout,
	}

	// Deep copy CORSConfig
	middlewareCopy.CORSConfig = c.deepCopyCORSConfig()

	// Deep copy RateLimitConfig
	middlewareCopy.RateLimitConfig = c.HTTP.Middleware.RateLimitConfig // Simple struct, no nested pointers

	// Deep copy TimeoutConfig
	middlewareCopy.TimeoutConfig = c.HTTP.Middleware.TimeoutConfig // Simple struct, no nested pointers

	// Deep copy CustomHeaders map
	if c.HTTP.Middleware.CustomHeaders != nil {
		middlewareCopy.CustomHeaders = make(map[string]string, len(c.HTTP.Middleware.CustomHeaders))
		for k, v := range c.HTTP.Middleware.CustomHeaders {
			middlewareCopy.CustomHeaders[k] = v
		}
	}

	return middlewareCopy
}

// deepCopyCORSConfig creates a deep copy of CORSConfig
func (c *ServerConfig) deepCopyCORSConfig() CORSConfig {
	corsCopy := CORSConfig{
		AllowCredentials: c.HTTP.Middleware.CORSConfig.AllowCredentials,
		MaxAge:           c.HTTP.Middleware.CORSConfig.MaxAge,
	}

	// Deep copy slices
	if c.HTTP.Middleware.CORSConfig.AllowOrigins != nil {
		corsCopy.AllowOrigins = make([]string, len(c.HTTP.Middleware.CORSConfig.AllowOrigins))
		copy(corsCopy.AllowOrigins, c.HTTP.Middleware.CORSConfig.AllowOrigins)
	}

	if c.HTTP.Middleware.CORSConfig.AllowMethods != nil {
		corsCopy.AllowMethods = make([]string, len(c.HTTP.Middleware.CORSConfig.AllowMethods))
		copy(corsCopy.AllowMethods, c.HTTP.Middleware.CORSConfig.AllowMethods)
	}

	if c.HTTP.Middleware.CORSConfig.AllowHeaders != nil {
		corsCopy.AllowHeaders = make([]string, len(c.HTTP.Middleware.CORSConfig.AllowHeaders))
		copy(corsCopy.AllowHeaders, c.HTTP.Middleware.CORSConfig.AllowHeaders)
	}

	if c.HTTP.Middleware.CORSConfig.ExposeHeaders != nil {
		corsCopy.ExposeHeaders = make([]string, len(c.HTTP.Middleware.CORSConfig.ExposeHeaders))
		copy(corsCopy.ExposeHeaders, c.HTTP.Middleware.CORSConfig.ExposeHeaders)
	}

	return corsCopy
}

// deepCopyGRPCConfig creates a deep copy of GRPCConfig
func (c *ServerConfig) deepCopyGRPCConfig() GRPCConfig {
	grpcCopy := GRPCConfig{
		Port:                c.GRPC.Port,
		Address:             c.GRPC.Address,
		EnableKeepalive:     c.GRPC.EnableKeepalive,
		EnableReflection:    c.GRPC.EnableReflection,
		EnableHealthService: c.GRPC.EnableHealthService,
		Enabled:             c.GRPC.Enabled,
		TestMode:            c.GRPC.TestMode,
		TestPort:            c.GRPC.TestPort,
		MaxRecvMsgSize:      c.GRPC.MaxRecvMsgSize,
		MaxSendMsgSize:      c.GRPC.MaxSendMsgSize,
		KeepaliveParams:     c.GRPC.KeepaliveParams, // Simple struct, no nested pointers
		KeepalivePolicy:     c.GRPC.KeepalivePolicy, // Simple struct, no nested pointers
		Interceptors:        c.GRPC.Interceptors,    // Simple struct, no nested pointers
		TLS:                 c.GRPC.TLS,             // Simple struct, no nested pointers
	}

	return grpcCopy
}

// deepCopyDiscoveryConfig creates a deep copy of DiscoveryConfig
func (c *ServerConfig) deepCopyDiscoveryConfig() DiscoveryConfig {
	discoveryCopy := DiscoveryConfig{
		Address:             c.Discovery.Address,
		ServiceName:         c.Discovery.ServiceName,
		Enabled:             c.Discovery.Enabled,
		FailureMode:         c.Discovery.FailureMode,
		HealthCheckRequired: c.Discovery.HealthCheckRequired,
		RegistrationTimeout: c.Discovery.RegistrationTimeout,
	}

	// Deep copy Tags slice
	if c.Discovery.Tags != nil {
		discoveryCopy.Tags = make([]string, len(c.Discovery.Tags))
		copy(discoveryCopy.Tags, c.Discovery.Tags)
	}

	return discoveryCopy
}
