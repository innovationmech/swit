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

package interfaces

import (
	"context"
)

// HealthStatus represents health check response
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp int64             `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// HealthService defines the interface for health check and monitoring operations.
// This interface provides methods for checking the health status of the service
// and its dependencies, enabling proper monitoring and alerting.
//
// Implementations should check various aspects of system health including:
// - Database connectivity
// - External service availability
// - Resource usage (memory, CPU, disk)
// - Application-specific health indicators
type HealthService interface {
	// CheckHealth performs a comprehensive health check of the service.
	// It should verify the status of all critical dependencies and return
	// a detailed health status including any issues found.
	//
	// The context can be used to set timeouts for health checks to prevent
	// hanging operations from affecting the overall service availability.
	//
	// Returns a HealthStatus with detailed information about the service state,
	// or an error if the health check itself fails to execute.
	CheckHealth(ctx context.Context) (*HealthStatus, error)
}
