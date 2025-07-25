// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package types

import (
	"time"
)

// HealthStatus represents the health status of the service
type HealthStatus struct {
	Status       string                      `json:"status"`
	Timestamp    time.Time                   `json:"timestamp"`
	Version      string                      `json:"version"`
	Uptime       time.Duration               `json:"uptime"`
	Dependencies map[string]DependencyStatus `json:"dependencies,omitempty"`
}

// DependencyStatus represents the status of a dependency
type DependencyStatus struct {
	Status    string        `json:"status"`
	Latency   time.Duration `json:"latency,omitempty"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// ServiceInfo represents basic service information
type ServiceInfo struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	StartTime   time.Time     `json:"start_time"`
	Uptime      time.Duration `json:"uptime"`
	Environment string        `json:"environment"`
	BuildInfo   *BuildInfo    `json:"build_info,omitempty"`
}

// BuildInfo represents build information
type BuildInfo struct {
	Commit    string    `json:"commit"`
	Branch    string    `json:"branch"`
	BuildTime time.Time `json:"build_time"`
	GoVersion string    `json:"go_version"`
}

// Health status constants
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusDegraded  = "degraded"
)

// Dependency status constants
const (
	DependencyStatusUp   = "up"
	DependencyStatusDown = "down"
	DependencyStatusSlow = "slow"
)

// NewHealthStatus creates a new health status
func NewHealthStatus(status, version string, uptime time.Duration) *HealthStatus {
	return &HealthStatus{
		Status:       status,
		Timestamp:    time.Now(),
		Version:      version,
		Uptime:       uptime,
		Dependencies: make(map[string]DependencyStatus),
	}
}

// AddDependency adds a dependency status
func (h *HealthStatus) AddDependency(name string, status DependencyStatus) {
	if h.Dependencies == nil {
		h.Dependencies = make(map[string]DependencyStatus)
	}
	h.Dependencies[name] = status
}

// IsHealthy checks if the service is healthy
func (h *HealthStatus) IsHealthy() bool {
	return h.Status == HealthStatusHealthy
}

// NewDependencyStatus creates a new dependency status
func NewDependencyStatus(status string, latency time.Duration) DependencyStatus {
	return DependencyStatus{
		Status:    status,
		Latency:   latency,
		Timestamp: time.Now(),
	}
}

// NewDependencyStatusWithError creates a new dependency status with error
func NewDependencyStatusWithError(status, errorMsg string) DependencyStatus {
	return DependencyStatus{
		Status:    status,
		Error:     errorMsg,
		Timestamp: time.Now(),
	}
}

// NewServiceInfo creates a new service info
func NewServiceInfo(name, version, description, environment string, startTime time.Time) *ServiceInfo {
	return &ServiceInfo{
		Name:        name,
		Version:     version,
		Description: description,
		StartTime:   startTime,
		Uptime:      time.Since(startTime),
		Environment: environment,
	}
}

// SetBuildInfo sets the build information
func (s *ServiceInfo) SetBuildInfo(buildInfo *BuildInfo) {
	s.BuildInfo = buildInfo
}

// UpdateUptime updates the uptime
func (s *ServiceInfo) UpdateUptime() {
	s.Uptime = time.Since(s.StartTime)
}
