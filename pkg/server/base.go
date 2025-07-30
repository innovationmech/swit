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
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/transport"
	"go.uber.org/zap"
)

// Server represents the base server implementation
type Server struct {
	// Configuration
	config *ServerConfig
	logger *zap.Logger

	// Core components
	transportManager     *transport.Manager
	serviceDiscovery     *discovery.ServiceDiscovery
	discoveryAbstraction *ServiceDiscoveryAbstraction
	dependencies         DependencyContainer
	registry             *serviceRegistry

	// Lifecycle management
	lifecycleHooks []LifecycleHook
	mu             sync.RWMutex
	started        bool
	stopped        bool

	// Shutdown management
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
}

// NewServer creates a new base server instance with the given configuration
func NewServer(config *ServerConfig, registrar ServiceRegistrar, deps DependencyContainer) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("server configuration is required")
	}

	if registrar == nil {
		return nil, fmt.Errorf("service registrar is required")
	}

	// Initialize logger if not already initialized
	if logger.Logger == nil {
		logger.InitLogger()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create transport manager
	transportManager := transport.NewManager()

	// Create service registry
	serviceReg := newServiceRegistry(transportManager)

	// Register services with the registry
	if err := registrar.RegisterServices(serviceReg); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	// Create service discovery if enabled
	var sd *discovery.ServiceDiscovery
	var discoveryAbstraction *ServiceDiscoveryAbstraction
	if config.Discovery.Enabled {
		var err error
		sd, err = discovery.NewServiceDiscovery(config.Discovery.Consul.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to create service discovery: %w", err)
		}
		discoveryAbstraction = NewServiceDiscoveryAbstraction(sd, &config.Discovery)
	}

	server := &Server{
		config:               config,
		logger:               logger.Logger,
		transportManager:     transportManager,
		serviceDiscovery:     sd,
		discoveryAbstraction: discoveryAbstraction,
		dependencies:         deps,
		registry:             serviceReg,
		shutdownCh:           make(chan struct{}),
	}

	// Setup transports based on configuration
	if err := server.setupTransports(); err != nil {
		return nil, fmt.Errorf("failed to setup transports: %w", err)
	}

	return server, nil
}

// Start initializes and starts the server with the given context
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server is already started")
	}

	if s.stopped {
		return fmt.Errorf("server has been stopped and cannot be restarted")
	}

	if logger.Logger != nil {
		logger.Logger.Info("Starting server", zap.String("service", s.config.ServiceName))
	}

	// Create startup context with timeout
	startupCtx, cancel := context.WithTimeout(ctx, s.config.StartupTimeout)
	defer cancel()

	// Execute startup sequence
	if err := s.executeStartupSequence(startupCtx); err != nil {
		return fmt.Errorf("startup failed: %w", err)
	}

	s.started = true
	if logger.Logger != nil {
		logger.Logger.Info("Server started successfully",
			zap.String("service", s.config.ServiceName),
			zap.String("http_address", s.GetHTTPAddress()),
			zap.String("grpc_address", s.GetGRPCAddress()),
		)
	}

	return nil
}

// Stop gracefully stops the server with the given context
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return fmt.Errorf("server is not started")
	}

	if s.stopped {
		return nil // Already stopped
	}

	if logger.Logger != nil {
		logger.Logger.Info("Stopping server", zap.String("service", s.config.ServiceName))
	}

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	// Execute shutdown sequence
	if err := s.executeShutdownSequence(shutdownCtx); err != nil {
		if logger.Logger != nil {
			logger.Logger.Error("Shutdown completed with errors", zap.Error(err))
		}
		return err
	}

	s.stopped = true
	if logger.Logger != nil {
		logger.Logger.Info("Server stopped successfully", zap.String("service", s.config.ServiceName))
	}

	return nil
}

// Shutdown performs immediate shutdown of the server
func (s *Server) Shutdown() error {
	s.shutdownOnce.Do(func() {
		close(s.shutdownCh)
	})

	// Use a background context for immediate shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	return s.Stop(ctx)
}

// GetHTTPAddress returns the HTTP server address
func (s *Server) GetHTTPAddress() string {
	if !s.config.EnableHTTP {
		return ""
	}

	// Try to get actual address from running transport first
	for _, transport := range s.transportManager.GetTransports() {
		if transport.GetName() == "http" {
			if addr := transport.GetAddress(); addr != "" {
				// Convert IPv6 [::]:port to localhost:port for testing
				if strings.HasPrefix(addr, "[::]:") {
					return "localhost" + addr[4:]
				}
				return addr
			}
			break
		}
	}

	// Fallback to configured address
	return s.config.GetHTTPAddress()
}

// GetGRPCAddress returns the gRPC server address
func (s *Server) GetGRPCAddress() string {
	if !s.config.EnableGRPC {
		return ""
	}

	// Try to get actual address from running transport first
	for _, transport := range s.transportManager.GetTransports() {
		if transport.GetName() == "grpc" {
			if addr := transport.GetAddress(); addr != "" {
				return addr
			}
			break
		}
	}

	// Fallback to configured address
	return s.config.GetGRPCAddress()
}

// GetTransports returns all registered transports
func (s *Server) GetTransports() []transport.Transport {
	return s.transportManager.GetTransports()
}

// GetServiceName returns the service name
func (s *Server) GetServiceName() string {
	return s.config.ServiceName
}

// AddLifecycleHook adds a lifecycle hook to the server
func (s *Server) AddLifecycleHook(hook LifecycleHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lifecycleHooks = append(s.lifecycleHooks, hook)
}

// setupTransports configures and registers the required transports
func (s *Server) setupTransports() error {
	if s.config.EnableHTTP {
		httpTransport, err := s.createHTTPTransport()
		if err != nil {
			return fmt.Errorf("failed to create HTTP transport: %w", err)
		}
		s.transportManager.Register(httpTransport)
	}

	if s.config.EnableGRPC {
		grpcTransport, err := s.createGRPCTransport()
		if err != nil {
			return fmt.Errorf("failed to create gRPC transport: %w", err)
		}
		s.transportManager.Register(grpcTransport)
	}

	return nil
}

// createHTTPTransport creates and configures the HTTP transport
func (s *Server) createHTTPTransport() (transport.Transport, error) {
	httpConfig := &transport.HTTPTransportConfig{
		Address:     s.config.GetHTTPAddress(),
		EnableReady: s.config.EnableReady,
	}
	httpTransport := transport.NewHTTPTransportWithConfig(httpConfig)

	// Set the address
	httpTransport.SetAddress(s.config.GetHTTPAddress())

	// Register HTTP routes
	router := httpTransport.GetRouter()
	if err := s.registry.RegisterAllHTTP(router); err != nil {
		return nil, fmt.Errorf("failed to register HTTP routes: %w", err)
	}

	return httpTransport, nil
}

// createGRPCTransport creates and configures the gRPC transport
func (s *Server) createGRPCTransport() (transport.Transport, error) {
	grpcConfig := &transport.GRPCTransportConfig{
		Address:             s.config.GetGRPCAddress(),
		MaxRecvMsgSize:      s.config.GRPCConfig.MaxRecvMsgSize,
		MaxSendMsgSize:      s.config.GRPCConfig.MaxSendMsgSize,
		EnableReflection:    s.config.GRPCConfig.EnableReflection,
		EnableHealthService: true,
		EnableKeepalive:     true,
	}
	grpcTransport := transport.NewGRPCTransportWithConfig(grpcConfig)

	// Set the address
	grpcTransport.SetAddress(s.config.GetGRPCAddress())

	return grpcTransport, nil
}

// executeStartupSequence executes the server startup sequence
func (s *Server) executeStartupSequence(ctx context.Context) error {
	// 1. Execute starting hooks
	if err := s.executeLifecycleHooks(ctx, "starting"); err != nil {
		return fmt.Errorf("starting hooks failed: %w", err)
	}

	// 2. Initialize dependencies
	if s.dependencies != nil {
		if err := s.dependencies.Initialize(ctx); err != nil {
			return fmt.Errorf("dependency initialization failed: %w", err)
		}
	}

	// 3. Initialize services
	if err := s.transportManager.InitializeAllServices(ctx); err != nil {
		return fmt.Errorf("service initialization failed: %w", err)
	}

	// 4. Start transports
	if err := s.transportManager.Start(ctx); err != nil {
		return fmt.Errorf("transport startup failed: %w", err)
	}

	// 5. Register with service discovery
	if err := s.registerWithDiscovery(ctx); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Service discovery registration failed", zap.Error(err))
		}
		// Don't fail startup if discovery registration fails
	}

	// 6. Execute started hooks
	if err := s.executeLifecycleHooks(ctx, "started"); err != nil {
		return fmt.Errorf("started hooks failed: %w", err)
	}

	return nil
}

// executeShutdownSequence executes the server shutdown sequence
func (s *Server) executeShutdownSequence(ctx context.Context) error {
	var errors []error

	// 1. Execute stopping hooks
	if err := s.executeLifecycleHooks(ctx, "stopping"); err != nil {
		errors = append(errors, fmt.Errorf("stopping hooks failed: %w", err))
	}

	// 2. Deregister from service discovery
	if err := s.deregisterFromDiscovery(ctx); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Service discovery deregistration failed", zap.Error(err))
		}
		// Don't add to errors as this is not critical
	}

	// 3. Stop transports
	if err := s.transportManager.Stop(s.config.ShutdownTimeout); err != nil {
		errors = append(errors, fmt.Errorf("transport shutdown failed: %w", err))
	}

	// 4. Shutdown services
	if err := s.transportManager.ShutdownAllServices(ctx); err != nil {
		errors = append(errors, fmt.Errorf("service shutdown failed: %w", err))
	}

	// 5. Cleanup dependencies
	if s.dependencies != nil {
		if err := s.dependencies.Cleanup(ctx); err != nil {
			errors = append(errors, fmt.Errorf("dependency cleanup failed: %w", err))
		}
	}

	// 6. Execute stopped hooks
	if err := s.executeLifecycleHooks(ctx, "stopped"); err != nil {
		errors = append(errors, fmt.Errorf("stopped hooks failed: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown completed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// executeLifecycleHooks executes lifecycle hooks for the given phase
func (s *Server) executeLifecycleHooks(ctx context.Context, phase string) error {
	for _, hook := range s.lifecycleHooks {
		var err error
		switch phase {
		case "starting":
			err = hook.OnStarting(ctx)
		case "started":
			err = hook.OnStarted(ctx)
		case "stopping":
			err = hook.OnStopping(ctx)
		case "stopped":
			err = hook.OnStopped(ctx)
		}

		if err != nil {
			return fmt.Errorf("lifecycle hook failed in %s phase: %w", phase, err)
		}
	}
	return nil
}

// registerWithDiscovery registers the server with service discovery using the abstraction layer
func (s *Server) registerWithDiscovery(ctx context.Context) error {
	if s.discoveryAbstraction == nil || !s.config.Discovery.Enabled {
		return nil
	}

	var endpoints []ServiceEndpoint

	// Prepare HTTP endpoint if enabled
	if s.config.EnableHTTP {
		if endpoint, err := s.createHTTPEndpoint(); err == nil {
			endpoints = append(endpoints, endpoint)
		} else {
			if logger.Logger != nil {
				logger.Logger.Warn("Failed to create HTTP endpoint for registration", zap.Error(err))
			}
		}
	}

	// Prepare gRPC endpoint if enabled
	if s.config.EnableGRPC {
		if endpoint, err := s.createGRPCEndpoint(); err == nil {
			endpoints = append(endpoints, endpoint)
		} else {
			if logger.Logger != nil {
				logger.Logger.Warn("Failed to create gRPC endpoint for registration", zap.Error(err))
			}
		}
	}

	// Register all endpoints with retry logic
	if len(endpoints) > 0 {
		if err := s.discoveryAbstraction.RegisterMultipleEndpoints(ctx, endpoints); err != nil {
			// Log warning but don't fail startup
			if logger.Logger != nil {
				logger.Logger.Warn("Service discovery registration failed", zap.Error(err))
			}
			return err
		}
	}

	return nil
}

// createHTTPEndpoint creates a ServiceEndpoint for HTTP registration
func (s *Server) createHTTPEndpoint() (ServiceEndpoint, error) {
	address := s.GetHTTPAddress()
	if address == "" {
		return ServiceEndpoint{}, fmt.Errorf("HTTP address not available")
	}

	// Extract port from address
	port, err := extractPortFromAddress(address)
	if err != nil {
		return ServiceEndpoint{}, fmt.Errorf("failed to extract HTTP port: %w", err)
	}

	serviceName := s.config.Discovery.ServiceName + "-http"
	return ServiceEndpoint{
		ServiceName: serviceName,
		Address:     "localhost",
		Port:        port,
		Protocol:    "http",
		Tags:        []string{"http", "api"},
	}, nil
}

// createGRPCEndpoint creates a ServiceEndpoint for gRPC registration
func (s *Server) createGRPCEndpoint() (ServiceEndpoint, error) {
	address := s.GetGRPCAddress()
	if address == "" {
		return ServiceEndpoint{}, fmt.Errorf("gRPC address not available")
	}

	// Extract port from address
	port, err := extractPortFromAddress(address)
	if err != nil {
		return ServiceEndpoint{}, fmt.Errorf("failed to extract gRPC port: %w", err)
	}

	serviceName := s.config.Discovery.ServiceName + "-grpc"
	return ServiceEndpoint{
		ServiceName: serviceName,
		Address:     "localhost",
		Port:        port,
		Protocol:    "grpc",
		Tags:        []string{"grpc", "rpc"},
	}, nil
}

// deregisterFromDiscovery deregisters the server from service discovery using the abstraction layer
func (s *Server) deregisterFromDiscovery(ctx context.Context) error {
	if s.discoveryAbstraction == nil || !s.config.Discovery.Enabled {
		return nil
	}

	// Deregister all endpoints with retry logic
	if err := s.discoveryAbstraction.DeregisterAllEndpoints(ctx); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Service discovery deregistration failed", zap.Error(err))
		}
		return err
	}

	return nil
}
