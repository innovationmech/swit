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
	"math/rand/v2"
	"strconv"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// ServiceDiscoveryManager abstracts service discovery operations for the base server
type ServiceDiscoveryManager interface {
	// RegisterService registers a service with discovery
	RegisterService(ctx context.Context, registration *ServiceRegistration) error
	// DeregisterService removes a service from discovery
	DeregisterService(ctx context.Context, registration *ServiceRegistration) error
	// RegisterMultipleEndpoints registers multiple endpoints for a service
	RegisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error
	// DeregisterMultipleEndpoints removes multiple endpoints from discovery
	DeregisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error
	// IsHealthy checks if the discovery service is available
	IsHealthy(ctx context.Context) bool
}

// ServiceRegistration represents a service registration with discovery
type ServiceRegistration struct {
	ServiceName string            `json:"service_name"`
	ServiceID   string            `json:"service_id"`
	Address     string            `json:"address"`
	Port        int               `json:"port"`
	Tags        []string          `json:"tags"`
	Meta        map[string]string `json:"meta"`
	HealthCheck *HealthCheckInfo  `json:"health_check,omitempty"`
}

// HealthCheckInfo represents health check configuration for service discovery
type HealthCheckInfo struct {
	HTTP                           string        `json:"http,omitempty"`
	GRPC                           string        `json:"grpc,omitempty"`
	Interval                       time.Duration `json:"interval"`
	Timeout                        time.Duration `json:"timeout"`
	DeregisterCriticalServiceAfter time.Duration `json:"deregister_critical_service_after"`
}

// DiscoveryManagerImpl implements ServiceDiscoveryManager using Consul
type DiscoveryManagerImpl struct {
	client        *discovery.ServiceDiscovery
	config        *DiscoveryConfig
	retryConfig   *RetryConfig
	registrations map[string]*ServiceRegistration // Track registered services
	healthStatus  *DiscoveryHealthStatus
	mu            sync.RWMutex
	startTime     time.Time
}

// RetryConfig holds retry configuration for discovery operations
type RetryConfig struct {
	MaxRetries      int           `yaml:"max_retries" json:"max_retries"`
	InitialInterval time.Duration `yaml:"initial_interval" json:"initial_interval"`
	MaxInterval     time.Duration `yaml:"max_interval" json:"max_interval"`
	Multiplier      float64       `yaml:"multiplier" json:"multiplier"`
	EnableJitter    bool          `yaml:"enable_jitter" json:"enable_jitter"`
}

// DiscoveryHealthStatus represents the health status of the discovery service
type DiscoveryHealthStatus struct {
	Available         bool          `json:"available"`
	LastCheck         time.Time     `json:"last_check"`
	LastError         string        `json:"last_error,omitempty"`
	ConsecutiveErrors int           `json:"consecutive_errors"`
	TotalErrors       int           `json:"total_errors"`
	Uptime            time.Duration `json:"uptime"`
}

// NewDiscoveryManager creates a new service discovery manager
func NewDiscoveryManager(config *DiscoveryConfig) (ServiceDiscoveryManager, error) {
	if config == nil || !config.Enabled {
		return &NoOpDiscoveryManager{}, nil
	}

	client, err := discovery.GetServiceDiscoveryByAddress(config.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	retryConfig := &RetryConfig{
		MaxRetries:      3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		EnableJitter:    true,
	}

	healthStatus := &DiscoveryHealthStatus{
		Available:         true,
		LastCheck:         time.Now(),
		ConsecutiveErrors: 0,
		TotalErrors:       0,
	}

	return &DiscoveryManagerImpl{
		client:        client,
		config:        config,
		retryConfig:   retryConfig,
		registrations: make(map[string]*ServiceRegistration),
		healthStatus:  healthStatus,
		startTime:     time.Now(),
	}, nil
}

// RegisterService registers a single service with discovery
func (d *DiscoveryManagerImpl) RegisterService(ctx context.Context, registration *ServiceRegistration) error {
	if registration == nil {
		return fmt.Errorf("service registration cannot be nil")
	}

	// Generate service ID if not provided
	if registration.ServiceID == "" {
		registration.ServiceID = fmt.Sprintf("%s-%s-%d",
			registration.ServiceName, registration.Address, registration.Port)
	}

	// Perform registration with retry and graceful failure handling
	err := d.retryOperationWithGracefulFailure(ctx, func() error {
		return d.client.RegisterService(registration.ServiceName, registration.Address, registration.Port)
	}, fmt.Sprintf("register_service_%s", registration.ServiceName))

	if err != nil {
		// Error already logged in retryOperationWithGracefulFailure
		return err
	}

	// Track successful registration
	d.registrations[registration.ServiceID] = registration

	logger.Logger.Info("Service registered with discovery",
		zap.String("service", registration.ServiceName),
		zap.String("service_id", registration.ServiceID),
		zap.String("address", registration.Address),
		zap.Int("port", registration.Port),
		zap.Strings("tags", registration.Tags))

	return nil
}

// DeregisterService removes a service from discovery
func (d *DiscoveryManagerImpl) DeregisterService(ctx context.Context, registration *ServiceRegistration) error {
	if registration == nil {
		return fmt.Errorf("service registration cannot be nil")
	}

	// Generate service ID if not provided
	if registration.ServiceID == "" {
		registration.ServiceID = fmt.Sprintf("%s-%s-%d",
			registration.ServiceName, registration.Address, registration.Port)
	}

	// Perform deregistration with retry and graceful failure handling
	err := d.retryOperationWithGracefulFailure(ctx, func() error {
		return d.client.DeregisterService(registration.ServiceName, registration.Address, registration.Port)
	}, fmt.Sprintf("deregister_service_%s", registration.ServiceName))

	if err != nil {
		// Error already logged in retryOperationWithGracefulFailure
		return err
	}

	// Remove from tracking
	delete(d.registrations, registration.ServiceID)

	logger.Logger.Info("Service deregistered from discovery",
		zap.String("service", registration.ServiceName),
		zap.String("service_id", registration.ServiceID))

	return nil
}

// RegisterMultipleEndpoints registers multiple endpoints for a service
func (d *DiscoveryManagerImpl) RegisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error {
	if len(registrations) == 0 {
		return nil
	}

	var errors []error
	successCount := 0

	for _, registration := range registrations {
		if err := d.RegisterService(ctx, registration); err != nil {
			errors = append(errors, fmt.Errorf("failed to register %s: %w", registration.ServiceName, err))
		} else {
			successCount++
		}
	}

	logger.Logger.Info("Multiple endpoint registration completed",
		zap.Int("total", len(registrations)),
		zap.Int("successful", successCount),
		zap.Int("failed", len(errors)))

	// Return error if all registrations failed
	if len(errors) == len(registrations) {
		return fmt.Errorf("all service registrations failed: %v", errors)
	}

	// Log warnings for partial failures but don't fail completely
	if len(errors) > 0 {
		for _, err := range errors {
			logger.Logger.Warn("Partial registration failure", zap.Error(err))
		}
	}

	return nil
}

// DeregisterMultipleEndpoints removes multiple endpoints from discovery
func (d *DiscoveryManagerImpl) DeregisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error {
	if len(registrations) == 0 {
		return nil
	}

	var errors []error
	successCount := 0

	for _, registration := range registrations {
		if err := d.DeregisterService(ctx, registration); err != nil {
			errors = append(errors, fmt.Errorf("failed to deregister %s: %w", registration.ServiceName, err))
		} else {
			successCount++
		}
	}

	logger.Logger.Info("Multiple endpoint deregistration completed",
		zap.Int("total", len(registrations)),
		zap.Int("successful", successCount),
		zap.Int("failed", len(errors)))

	// Log warnings for failures but don't fail completely during shutdown
	if len(errors) > 0 {
		for _, err := range errors {
			logger.Logger.Warn("Partial deregistration failure", zap.Error(err))
		}
	}

	return nil
}

// IsHealthy checks if the discovery service is available
func (d *DiscoveryManagerImpl) IsHealthy(ctx context.Context) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Update last check time
	d.healthStatus.LastCheck = time.Now()
	d.healthStatus.Uptime = time.Since(d.startTime)

	// Simple health check - try to get service list
	// This is a lightweight operation that tests connectivity
	_, err := d.client.GetInstanceRandom("consul") // Try to get consul service itself

	if err != nil {
		d.healthStatus.Available = false
		d.healthStatus.LastError = err.Error()
		d.healthStatus.ConsecutiveErrors++
		d.healthStatus.TotalErrors++

		logger.Logger.Warn("Discovery service health check failed",
			zap.Error(err),
			zap.Int("consecutive_errors", d.healthStatus.ConsecutiveErrors),
			zap.Int("total_errors", d.healthStatus.TotalErrors))

		return false
	}

	// Reset error counters on successful health check
	if d.healthStatus.ConsecutiveErrors > 0 {
		logger.Logger.Info("Discovery service health restored",
			zap.Int("previous_consecutive_errors", d.healthStatus.ConsecutiveErrors))
	}

	d.healthStatus.Available = true
	d.healthStatus.LastError = ""
	d.healthStatus.ConsecutiveErrors = 0

	return true
}

// GetHealthStatus returns the current health status of the discovery service
func (d *DiscoveryManagerImpl) GetHealthStatus() *DiscoveryHealthStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Create a copy to avoid race conditions
	status := *d.healthStatus
	return &status
}

// IsDiscoveryAvailable checks if discovery is available without updating health status
func (d *DiscoveryManagerImpl) IsDiscoveryAvailable() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.healthStatus.Available
}

// retryOperation performs an operation with exponential backoff retry and jitter
func (d *DiscoveryManagerImpl) retryOperation(ctx context.Context, operation func() error) error {
	var lastErr error
	interval := d.retryConfig.InitialInterval

	for attempt := 0; attempt <= d.retryConfig.MaxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Perform the operation
		if err := operation(); err != nil {
			lastErr = err

			// Don't retry on last attempt
			if attempt == d.retryConfig.MaxRetries {
				// Log final failure with warning level
				logger.Logger.Warn("Discovery operation failed after all retries",
					zap.Int("total_attempts", attempt+1),
					zap.Error(err))
				break
			}

			// Calculate next interval with exponential backoff
			nextInterval := time.Duration(float64(interval) * d.retryConfig.Multiplier)
			if nextInterval > d.retryConfig.MaxInterval {
				nextInterval = d.retryConfig.MaxInterval
			}

			// Add jitter to prevent thundering herd
			if d.retryConfig.EnableJitter {
				jitter := time.Duration(rand.Float64() * float64(nextInterval) * 0.1) // 10% jitter
				nextInterval = nextInterval + jitter
			}

			logger.Logger.Debug("Discovery operation failed, retrying",
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", d.retryConfig.MaxRetries),
				zap.Duration("next_retry_in", nextInterval),
				zap.Error(err))

			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(nextInterval):
			}

			interval = nextInterval
		} else {
			// Operation succeeded
			if attempt > 0 {
				logger.Logger.Info("Discovery operation succeeded after retries",
					zap.Int("attempts", attempt+1))
			}
			return nil
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", d.retryConfig.MaxRetries+1, lastErr)
}

// retryOperationWithGracefulFailure performs an operation with retry but doesn't fail the server on discovery errors
func (d *DiscoveryManagerImpl) retryOperationWithGracefulFailure(ctx context.Context, operation func() error, operationName string) error {
	err := d.retryOperation(ctx, operation)
	if err != nil {
		// Update health status
		d.mu.Lock()
		d.healthStatus.Available = false
		d.healthStatus.LastError = err.Error()
		d.healthStatus.ConsecutiveErrors++
		d.healthStatus.TotalErrors++
		d.mu.Unlock()

		// Log warning but don't fail the server
		logger.Logger.Warn("Discovery operation failed gracefully",
			zap.String("operation", operationName),
			zap.Error(err),
			zap.String("action", "continuing without discovery"))

		return err
	}

	// Reset error counters on success
	d.mu.Lock()
	if d.healthStatus.ConsecutiveErrors > 0 {
		logger.Logger.Info("Discovery operation recovered",
			zap.String("operation", operationName),
			zap.Int("previous_errors", d.healthStatus.ConsecutiveErrors))
	}
	d.healthStatus.Available = true
	d.healthStatus.LastError = ""
	d.healthStatus.ConsecutiveErrors = 0
	d.mu.Unlock()

	return nil
}

// NoOpDiscoveryManager is a no-op implementation for when discovery is disabled
type NoOpDiscoveryManager struct{}

func (n *NoOpDiscoveryManager) RegisterService(ctx context.Context, registration *ServiceRegistration) error {
	logger.Logger.Debug("Service discovery disabled, skipping registration",
		zap.String("service", registration.ServiceName))
	return nil
}

func (n *NoOpDiscoveryManager) DeregisterService(ctx context.Context, registration *ServiceRegistration) error {
	logger.Logger.Debug("Service discovery disabled, skipping deregistration",
		zap.String("service", registration.ServiceName))
	return nil
}

func (n *NoOpDiscoveryManager) RegisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error {
	logger.Logger.Debug("Service discovery disabled, skipping multiple endpoint registration",
		zap.Int("count", len(registrations)))
	return nil
}

func (n *NoOpDiscoveryManager) DeregisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error {
	logger.Logger.Debug("Service discovery disabled, skipping multiple endpoint deregistration",
		zap.Int("count", len(registrations)))
	return nil
}

func (n *NoOpDiscoveryManager) IsHealthy(ctx context.Context) bool {
	return true // Always healthy when disabled
}

// GetHealthStatus returns a healthy status for no-op manager
func (n *NoOpDiscoveryManager) GetHealthStatus() *DiscoveryHealthStatus {
	return &DiscoveryHealthStatus{
		Available:         true,
		LastCheck:         time.Now(),
		ConsecutiveErrors: 0,
		TotalErrors:       0,
		Uptime:            0, // No meaningful uptime for no-op
	}
}

// IsDiscoveryAvailable always returns true for no-op manager
func (n *NoOpDiscoveryManager) IsDiscoveryAvailable() bool {
	return true
}

// Helper functions for creating service registrations

// CreateHTTPServiceRegistration creates a service registration for HTTP endpoint
func CreateHTTPServiceRegistration(serviceName, address string, port int, tags []string) *ServiceRegistration {
	portInt, _ := strconv.Atoi(fmt.Sprintf("%d", port))

	return &ServiceRegistration{
		ServiceName: serviceName,
		Address:     address,
		Port:        portInt,
		Tags:        append(tags, "http"),
		Meta: map[string]string{
			"protocol": "http",
			"version":  "v1",
		},
		HealthCheck: &HealthCheckInfo{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", address, port),
			Interval:                       10 * time.Second,
			Timeout:                        5 * time.Second,
			DeregisterCriticalServiceAfter: 1 * time.Minute,
		},
	}
}

// CreateGRPCServiceRegistration creates a service registration for gRPC endpoint
func CreateGRPCServiceRegistration(serviceName, address string, port int, tags []string) *ServiceRegistration {
	portInt, _ := strconv.Atoi(fmt.Sprintf("%d", port))

	return &ServiceRegistration{
		ServiceName: serviceName + "-grpc",
		Address:     address,
		Port:        portInt,
		Tags:        append(tags, "grpc"),
		Meta: map[string]string{
			"protocol": "grpc",
			"version":  "v1",
		},
		HealthCheck: &HealthCheckInfo{
			GRPC:                           fmt.Sprintf("%s:%d", address, port),
			Interval:                       10 * time.Second,
			Timeout:                        5 * time.Second,
			DeregisterCriticalServiceAfter: 1 * time.Minute,
		},
	}
}

// CreateServiceRegistrations creates registrations for both HTTP and gRPC endpoints
// This function implements multi-endpoint registration support with service naming patterns
// and port-based service differentiation for discovery
func CreateServiceRegistrations(config *ServerConfig) []*ServiceRegistration {
	var registrations []*ServiceRegistration
	address := "localhost" // Default address for local services

	// Create HTTP registration if enabled
	if config.IsHTTPEnabled() {
		port, _ := strconv.Atoi(config.HTTP.Port)
		httpReg := CreateHTTPServiceRegistration(
			config.Discovery.ServiceName,
			address,
			port,
			config.Discovery.Tags,
		)
		registrations = append(registrations, httpReg)

		logger.Logger.Debug("Created HTTP service registration",
			zap.String("service", httpReg.ServiceName),
			zap.String("address", httpReg.Address),
			zap.Int("port", httpReg.Port),
			zap.Strings("tags", httpReg.Tags))
	}

	// Create gRPC registration if enabled
	if config.IsGRPCEnabled() {
		grpcPort, _ := strconv.Atoi(config.GRPC.Port)
		httpPort := 0
		if config.IsHTTPEnabled() {
			httpPort, _ = strconv.Atoi(config.HTTP.Port)
		}

		// Always register gRPC service with distinct naming pattern
		// Use port-based differentiation for service discovery
		grpcReg := CreateGRPCServiceRegistration(
			config.Discovery.ServiceName,
			address,
			grpcPort,
			config.Discovery.Tags,
		)

		// Add port differentiation metadata
		if grpcReg.Meta == nil {
			grpcReg.Meta = make(map[string]string)
		}
		grpcReg.Meta["port_type"] = "grpc"
		grpcReg.Meta["http_port"] = strconv.Itoa(httpPort)
		grpcReg.Meta["grpc_port"] = strconv.Itoa(grpcPort)

		// Only register gRPC separately if it's on a different port
		// If same port, the HTTP registration covers both protocols
		if grpcPort != httpPort {
			registrations = append(registrations, grpcReg)

			logger.Logger.Debug("Created gRPC service registration",
				zap.String("service", grpcReg.ServiceName),
				zap.String("address", grpcReg.Address),
				zap.Int("port", grpcReg.Port),
				zap.Strings("tags", grpcReg.Tags))
		} else {
			// Same port - update HTTP registration to include gRPC metadata
			if len(registrations) > 0 && config.IsHTTPEnabled() {
				httpReg := registrations[0]
				if httpReg.Meta == nil {
					httpReg.Meta = make(map[string]string)
				}
				httpReg.Meta["supports_grpc"] = "true"
				httpReg.Meta["port_type"] = "dual"
				httpReg.Tags = append(httpReg.Tags, "grpc")

				logger.Logger.Debug("Updated HTTP registration to support dual protocol",
					zap.String("service", httpReg.ServiceName),
					zap.Int("port", httpReg.Port))
			}
		}
	}

	logger.Logger.Info("Service registrations created",
		zap.String("base_service", config.Discovery.ServiceName),
		zap.Int("total_registrations", len(registrations)),
		zap.Bool("http_enabled", config.IsHTTPEnabled()),
		zap.Bool("grpc_enabled", config.IsGRPCEnabled()))

	return registrations
}

// ServiceNamingPattern defines different patterns for service naming in discovery
type ServiceNamingPattern string

const (
	// ServiceNamingPatternBase uses the base service name
	ServiceNamingPatternBase ServiceNamingPattern = "base"
	// ServiceNamingPatternProtocol appends protocol suffix (e.g., service-grpc)
	ServiceNamingPatternProtocol ServiceNamingPattern = "protocol"
	// ServiceNamingPatternPort appends port suffix (e.g., service-9080)
	ServiceNamingPatternPort ServiceNamingPattern = "port"
)

// GenerateServiceName generates a service name based on the naming pattern
func GenerateServiceName(baseName string, protocol string, port int, pattern ServiceNamingPattern) string {
	switch pattern {
	case ServiceNamingPatternProtocol:
		if protocol != "http" {
			return fmt.Sprintf("%s-%s", baseName, protocol)
		}
		return baseName
	case ServiceNamingPatternPort:
		return fmt.Sprintf("%s-%d", baseName, port)
	case ServiceNamingPatternBase:
		fallthrough
	default:
		return baseName
	}
}

// GetServicesByProtocol filters service registrations by protocol
func GetServicesByProtocol(registrations []*ServiceRegistration, protocol string) []*ServiceRegistration {
	var filtered []*ServiceRegistration
	for _, reg := range registrations {
		if reg.Meta != nil && reg.Meta["protocol"] == protocol {
			filtered = append(filtered, reg)
		}
	}
	return filtered
}

// GetServicesByPort filters service registrations by port
func GetServicesByPort(registrations []*ServiceRegistration, port int) []*ServiceRegistration {
	var filtered []*ServiceRegistration
	for _, reg := range registrations {
		if reg.Port == port {
			filtered = append(filtered, reg)
		}
	}
	return filtered
}

// GetServiceEndpointInfo returns detailed endpoint information for a service
func GetServiceEndpointInfo(registrations []*ServiceRegistration) map[string]interface{} {
	info := make(map[string]interface{})

	var httpEndpoints []map[string]interface{}
	var grpcEndpoints []map[string]interface{}
	var dualEndpoints []map[string]interface{}

	for _, reg := range registrations {
		endpoint := map[string]interface{}{
			"service_name": reg.ServiceName,
			"service_id":   reg.ServiceID,
			"address":      reg.Address,
			"port":         reg.Port,
			"tags":         reg.Tags,
			"meta":         reg.Meta,
		}

		if reg.Meta != nil {
			switch reg.Meta["port_type"] {
			case "http":
				httpEndpoints = append(httpEndpoints, endpoint)
			case "grpc":
				grpcEndpoints = append(grpcEndpoints, endpoint)
			case "dual":
				dualEndpoints = append(dualEndpoints, endpoint)
			default:
				// Determine by protocol metadata
				if reg.Meta["protocol"] == "http" {
					httpEndpoints = append(httpEndpoints, endpoint)
				} else if reg.Meta["protocol"] == "grpc" {
					grpcEndpoints = append(grpcEndpoints, endpoint)
				}
			}
		}
	}

	info["http_endpoints"] = httpEndpoints
	info["grpc_endpoints"] = grpcEndpoints
	info["dual_endpoints"] = dualEndpoints
	info["total_endpoints"] = len(registrations)

	return info
}

// ValidateServiceRegistrations validates that service registrations are properly configured
func ValidateServiceRegistrations(registrations []*ServiceRegistration) error {
	if len(registrations) == 0 {
		return fmt.Errorf("no service registrations provided")
	}

	seenPorts := make(map[int]string)
	seenServiceIDs := make(map[string]bool)

	for _, reg := range registrations {
		if reg == nil {
			return fmt.Errorf("nil service registration found")
		}

		if reg.ServiceName == "" {
			return fmt.Errorf("service name cannot be empty")
		}

		if reg.Address == "" {
			return fmt.Errorf("service address cannot be empty for service %s", reg.ServiceName)
		}

		if reg.Port <= 0 || reg.Port > 65535 {
			return fmt.Errorf("invalid port %d for service %s", reg.Port, reg.ServiceName)
		}

		// Check for port conflicts between different services
		if existingService, exists := seenPorts[reg.Port]; exists && existingService != reg.ServiceName {
			return fmt.Errorf("port %d is used by multiple services: %s and %s",
				reg.Port, existingService, reg.ServiceName)
		}
		seenPorts[reg.Port] = reg.ServiceName

		// Generate service ID if not provided
		if reg.ServiceID == "" {
			reg.ServiceID = fmt.Sprintf("%s-%s-%d", reg.ServiceName, reg.Address, reg.Port)
		}

		// Check for duplicate service IDs (only if they represent different logical services)
		if seenServiceIDs[reg.ServiceID] {
			return fmt.Errorf("duplicate service ID: %s", reg.ServiceID)
		}
		seenServiceIDs[reg.ServiceID] = true
	}

	return nil
}
