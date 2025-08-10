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
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CommonServicePatterns provides reusable patterns for service implementations
type CommonServicePatterns struct{}

// NewCommonServicePatterns creates a new instance of common service patterns
func NewCommonServicePatterns() *CommonServicePatterns {
	return &CommonServicePatterns{}
}

// CreateDatabaseHealthCheck creates a standard database health check function
func (p *CommonServicePatterns) CreateDatabaseHealthCheck(db *gorm.DB) HealthCheckFunc {
	return func(ctx context.Context) error {
		if db == nil {
			return fmt.Errorf("database connection is nil")
		}

		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying sql.DB: %w", err)
		}

		// Use a timeout context for the ping
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(pingCtx); err != nil {
			return fmt.Errorf("database ping failed: %w", err)
		}

		return nil
	}
}

// CreateServiceAvailabilityCheck creates a standard service availability check
func (p *CommonServicePatterns) CreateServiceAvailabilityCheck(service interface{}, serviceName string) HealthCheckFunc {
	return func(ctx context.Context) error {
		if service == nil {
			return fmt.Errorf("%s is not available", serviceName)
		}

		// If the service implements a health check interface, use it
		if healthChecker, ok := service.(interface{ HealthCheck(context.Context) error }); ok {
			return healthChecker.HealthCheck(ctx)
		}

		// Otherwise, just check if the service is not nil
		return nil
	}
}

// SafeCloseDatabase safely closes a database connection with error logging
func (p *CommonServicePatterns) SafeCloseDatabase(db *gorm.DB, serviceName string) error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Logger.Error("Failed to get underlying sql.DB for closing",
			zap.String("service", serviceName),
			zap.Error(err))
		return err
	}

	if err := sqlDB.Close(); err != nil {
		logger.Logger.Error("Failed to close database connection",
			zap.String("service", serviceName),
			zap.Error(err))
		return err
	}

	logger.Logger.Info("Database connection closed successfully",
		zap.String("service", serviceName))
	return nil
}

// CreateGenericDependencyContainer creates a dependency container with common services
func (p *CommonServicePatterns) CreateGenericDependencyContainer(
	serviceName string,
	services map[string]interface{},
	db *gorm.DB,
) *GenericDependencyContainer {
	container := NewGenericDependencyContainer(func() error {
		logger.Logger.Info("Closing dependencies", zap.String("service", serviceName))

		// Close database connection if available
		if db != nil {
			return p.SafeCloseDatabase(db, serviceName)
		}

		return nil
	})

	// Register all provided services
	for name, service := range services {
		container.RegisterService(name, service)
	}

	// Always register database if provided
	if db != nil {
		container.RegisterService("database", db)
	}

	return container
}

// StandardConfigDefaults provides standard configuration defaults for services
type StandardConfigDefaults struct {
	ServiceName          string
	HTTPPort             string
	GRPCPort             string
	EnableHTTPReady      bool
	EnableGRPCKeepalive  bool
	EnableGRPCReflection bool
	EnableGRPCHealth     bool
	DiscoveryTags        []string
	EnableCORS           bool
	EnableAuth           bool
	EnableRateLimit      bool
	EnableLogging        bool
}

// ApplyStandardDefaults applies standard configuration defaults to a server config
func (p *CommonServicePatterns) ApplyStandardDefaults(config *ServerConfig, defaults StandardConfigDefaults) {
	mapper := NewConfigMapperBase()

	// Set basic server configuration
	mapper.SetDefaultServerConfig(config, defaults.ServiceName)

	// Set HTTP configuration
	mapper.SetDefaultHTTPConfig(config, "", defaults.HTTPPort, defaults.EnableHTTPReady)

	// Set gRPC configuration
	mapper.SetDefaultGRPCConfig(config, "", defaults.GRPCPort,
		defaults.EnableGRPCKeepalive, defaults.EnableGRPCReflection, defaults.EnableGRPCHealth)

	// Set discovery configuration
	mapper.SetDefaultDiscoveryConfig(config, "", defaults.ServiceName, defaults.DiscoveryTags)

	// Set middleware configuration
	mapper.SetDefaultMiddlewareConfig(config,
		defaults.EnableCORS, defaults.EnableAuth, defaults.EnableRateLimit, defaults.EnableLogging)
}

// ServiceRegistrationBatch provides batch registration utilities
type ServiceRegistrationBatch struct {
	helper *ServiceRegistrationHelper
}

// NewServiceRegistrationBatch creates a new service registration batch
func NewServiceRegistrationBatch(registry BusinessServiceRegistry) *ServiceRegistrationBatch {
	return &ServiceRegistrationBatch{
		helper: NewServiceRegistrationHelper(registry),
	}
}

// HTTPServiceDefinition defines an HTTP service for batch registration
type HTTPServiceDefinition struct {
	ServiceName     string
	Registrar       HTTPHandlerRegistrar
	HealthCheckFunc HealthCheckFunc
}

// GRPCServiceDefinition defines a gRPC service for batch registration
type GRPCServiceDefinition struct {
	ServiceName     string
	Registrar       GRPCServiceRegistrar
	HealthCheckFunc HealthCheckFunc
}

// RegisterHTTPServices registers multiple HTTP services in batch
func (b *ServiceRegistrationBatch) RegisterHTTPServices(services []HTTPServiceDefinition) error {
	for _, service := range services {
		if err := b.helper.RegisterHTTPHandlerWithHealthCheck(
			service.ServiceName,
			service.Registrar,
			service.HealthCheckFunc,
		); err != nil {
			return fmt.Errorf("failed to register HTTP service %s: %w", service.ServiceName, err)
		}
	}
	return nil
}

// RegisterGRPCServices registers multiple gRPC services in batch
func (b *ServiceRegistrationBatch) RegisterGRPCServices(services []GRPCServiceDefinition) error {
	for _, service := range services {
		if err := b.helper.RegisterGRPCServiceWithHealthCheck(
			service.ServiceName,
			service.Registrar,
			service.HealthCheckFunc,
		); err != nil {
			return fmt.Errorf("failed to register gRPC service %s: %w", service.ServiceName, err)
		}
	}
	return nil
}

// PerformanceOptimizations provides performance optimization utilities
type PerformanceOptimizations struct{}

// NewPerformanceOptimizations creates a new performance optimizations instance
func NewPerformanceOptimizations() *PerformanceOptimizations {
	return &PerformanceOptimizations{}
}

// OptimizeGORMConnection optimizes GORM database connection settings
func (p *PerformanceOptimizations) OptimizeGORMConnection(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings for better performance
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return nil
}

// PreallocateSlices provides utilities for preallocating slices to reduce allocations
func (p *PerformanceOptimizations) PreallocateSlices() map[string]interface{} {
	return map[string]interface{}{
		"small_slice":  make([]interface{}, 0, 10),
		"medium_slice": make([]interface{}, 0, 50),
		"large_slice":  make([]interface{}, 0, 100),
	}
}

// ResourceCleanup provides utilities for resource cleanup
type ResourceCleanup struct{}

// NewResourceCleanup creates a new resource cleanup instance
func NewResourceCleanup() *ResourceCleanup {
	return &ResourceCleanup{}
}

// CleanupDatabase performs database cleanup operations
func (r *ResourceCleanup) CleanupDatabase(db *gorm.DB, serviceName string) error {
	if db == nil {
		return nil
	}

	logger.Logger.Info("Performing database cleanup", zap.String("service", serviceName))

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Get connection stats before cleanup
	stats := sqlDB.Stats()
	logger.Logger.Info("Database connection stats before cleanup",
		zap.String("service", serviceName),
		zap.Int("open_connections", stats.OpenConnections),
		zap.Int("in_use", stats.InUse),
		zap.Int("idle", stats.Idle))

	// Close the database connection
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	logger.Logger.Info("Database cleanup completed", zap.String("service", serviceName))
	return nil
}

// CleanupResources performs general resource cleanup
func (r *ResourceCleanup) CleanupResources(resources map[string]interface{}, serviceName string) error {
	logger.Logger.Info("Performing resource cleanup",
		zap.String("service", serviceName),
		zap.Int("resource_count", len(resources)))

	var cleanupErrors []error

	for name, resource := range resources {
		if closer, ok := resource.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to close %s: %w", name, err))
			}
		}
	}

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("cleanup errors: %v", cleanupErrors)
	}

	logger.Logger.Info("Resource cleanup completed", zap.String("service", serviceName))
	return nil
}
