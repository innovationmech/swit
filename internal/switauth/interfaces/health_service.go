// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package interfaces

import (
	"context"

	"github.com/innovationmech/swit/internal/switauth/types"
)

// HealthService defines the interface for health check operations
type HealthService interface {
	// CheckHealth returns the current health status
	CheckHealth(ctx context.Context) *types.HealthStatus

	// GetHealthDetails returns detailed health information
	GetHealthDetails(ctx context.Context) map[string]interface{}

	// GetServiceInfo returns basic service information
	GetServiceInfo(ctx context.Context) *types.ServiceInfo

	// IsHealthy returns true if the service is healthy
	IsHealthy(ctx context.Context) bool
}
