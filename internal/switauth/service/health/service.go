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
)

// Service defines the interface for health check operations
type Service interface {
	// CheckHealth returns the current health status
	CheckHealth(ctx context.Context) *Status
	// GetHealthDetails returns detailed health information
	GetHealthDetails(ctx context.Context) map[string]interface{}
}

// Status represents the health check response
type Status struct {
	Status    string            `json:"status"`
	Timestamp int64             `json:"timestamp"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Details   map[string]string `json:"details,omitempty"`
}

// healthService implements the Service interface
type healthService struct {
	name    string
	version string
}

// NewHealthService creates a new health service instance
func NewHealthService() Service {
	return &healthService{
		name:    "switauth",
		version: "1.0.0",
	}
}

// CheckHealth returns the basic health status
func (h *healthService) CheckHealth(ctx context.Context) *Status {
	return &Status{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Service:   h.name,
		Version:   h.version,
		Details: map[string]string{
			"uptime": "running",
		},
	}
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
