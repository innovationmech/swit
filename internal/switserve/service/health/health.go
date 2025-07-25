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

package health

import (
	"context"
	"time"

	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/types"
)

// HealthSrv is an alias for the HealthService interface
type HealthSrv = interfaces.HealthService

// HealthServiceConfig holds configuration for the health service
type HealthServiceConfig struct {
	ServiceName string
	Version     string
	StartTime   time.Time
}

// HealthServiceOption is a function type for configuring the health service
type HealthServiceOption func(*HealthServiceConfig)

// Service implements health check business logic
type Service struct {
	config    *HealthServiceConfig
	startTime time.Time
}

// NewHealthSrv creates a new health service instance
func NewHealthSrv(opts ...HealthServiceOption) interfaces.HealthService {
	config := &HealthServiceConfig{
		ServiceName: "swit-serve",
		Version:     "1.0.0",
		StartTime:   time.Now(),
	}

	for _, opt := range opts {
		opt(config)
	}

	return &Service{
		config:    config,
		startTime: config.StartTime,
	}
}

// NewHealthSrvWithConfig creates a new health service instance with configuration
func NewHealthSrvWithConfig(config *HealthServiceConfig) interfaces.HealthService {
	if config == nil {
		config = &HealthServiceConfig{
			ServiceName: "swit-serve",
			Version:     "1.0.0",
			StartTime:   time.Now(),
		}
	}

	return &Service{
		config:    config,
		startTime: config.StartTime,
	}
}

// CheckHealth performs health checks and returns status
func (s *Service) CheckHealth(ctx context.Context) (*interfaces.HealthStatus, error) {
	// Validate context
	if ctx == nil {
		return nil, types.ErrValidation("context cannot be nil")
	}

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		// Context is cancelled, but we still return health status
		// This allows graceful handling of cancelled requests
	default:
	}

	// Create health status using shared types
	status := types.NewHealthStatus(
		types.HealthStatusHealthy,
		s.config.Version,
		time.Since(s.startTime),
	)

	// Add basic service dependencies check
	// In a real implementation, you would check:
	// - Database connectivity
	// - External service availability
	// - Resource usage checks

	// Example: Add a mock database dependency
	dbStatus := types.NewDependencyStatus(types.DependencyStatusUp, 5*time.Millisecond)
	status.AddDependency("database", dbStatus)

	// Convert to interface type
	interfaceStatus := &interfaces.HealthStatus{
		Status:    status.Status,
		Timestamp: status.Timestamp.Unix(),
		Details: map[string]string{
			"server":  s.config.ServiceName,
			"version": s.config.Version,
			"uptime":  status.Uptime.String(),
		},
	}

	return interfaceStatus, nil
}

// WithServiceName sets the service name
func WithServiceName(name string) HealthServiceOption {
	return func(config *HealthServiceConfig) {
		config.ServiceName = name
	}
}

// WithVersion sets the service version
func WithVersion(version string) HealthServiceOption {
	return func(config *HealthServiceConfig) {
		config.Version = version
	}
}

// WithStartTime sets the service start time
func WithStartTime(startTime time.Time) HealthServiceOption {
	return func(config *HealthServiceConfig) {
		config.StartTime = startTime
	}
}
