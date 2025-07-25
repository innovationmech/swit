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

	"github.com/innovationmech/swit/internal/switauth/interfaces"
	"github.com/innovationmech/swit/internal/switauth/types"
)

// healthService implements the Service interface
type healthService struct {
	name    string
	version string
}

// NewHealthService creates a new health service instance
func NewHealthService() interfaces.HealthService {
	return &healthService{
		name:    "switauth",
		version: "1.0.0",
	}
}

// CheckHealth returns the basic health status
func (h *healthService) CheckHealth(ctx context.Context) *types.HealthStatus {
	return types.NewHealthStatus(types.HealthStatusHealthy, h.version, 0)
}

// GetServiceInfo returns basic service information
func (h *healthService) GetServiceInfo(ctx context.Context) *types.ServiceInfo {
	return types.NewServiceInfo(h.name, h.version, "Authentication service", "production", time.Now())
}

// IsHealthy returns true if the service is healthy
func (h *healthService) IsHealthy(ctx context.Context) bool {
	return true
}

// GetHealthDetails returns comprehensive health information
func (h *healthService) GetHealthDetails(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"service":   h.name,
		"version":   h.version,
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    "running",
		"checks": map[string]interface{}{
			"database": "ok",
			"cache":    "ok",
			"external": "ok",
		},
	}
}
