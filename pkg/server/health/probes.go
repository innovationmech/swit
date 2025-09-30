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
	"fmt"
	"sync/atomic"
	"time"
)

// BasicHealthChecker implements a basic health checker
type BasicHealthChecker struct {
	name        string
	checkFunc   func(context.Context) error
	description string
}

// NewBasicHealthChecker creates a new basic health checker
func NewBasicHealthChecker(name string, checkFunc func(context.Context) error) *BasicHealthChecker {
	return &BasicHealthChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

// Check performs the health check
func (c *BasicHealthChecker) Check(ctx context.Context) error {
	if c.checkFunc == nil {
		return nil // No-op check always passes
	}
	return c.checkFunc(ctx)
}

// GetName returns the health checker name
func (c *BasicHealthChecker) GetName() string {
	return c.name
}

// BasicReadinessChecker implements a basic readiness checker
type BasicReadinessChecker struct {
	name      string
	checkFunc func(context.Context) error
}

// NewBasicReadinessChecker creates a new basic readiness checker
func NewBasicReadinessChecker(name string, checkFunc func(context.Context) error) *BasicReadinessChecker {
	return &BasicReadinessChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

// CheckReadiness performs the readiness check
func (c *BasicReadinessChecker) CheckReadiness(ctx context.Context) error {
	if c.checkFunc == nil {
		return nil // No-op check always passes
	}
	return c.checkFunc(ctx)
}

// GetName returns the readiness checker name
func (c *BasicReadinessChecker) GetName() string {
	return c.name
}

// BasicLivenessChecker implements a basic liveness checker
type BasicLivenessChecker struct {
	name      string
	checkFunc func(context.Context) error
}

// NewBasicLivenessChecker creates a new basic liveness checker
func NewBasicLivenessChecker(name string, checkFunc func(context.Context) error) *BasicLivenessChecker {
	return &BasicLivenessChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

// CheckLiveness performs the liveness check
func (c *BasicLivenessChecker) CheckLiveness(ctx context.Context) error {
	if c.checkFunc == nil {
		return nil // No-op check always passes
	}
	return c.checkFunc(ctx)
}

// GetName returns the liveness checker name
func (c *BasicLivenessChecker) GetName() string {
	return c.name
}

// ServiceReadinessProbe checks if a service is ready to handle requests
type ServiceReadinessProbe struct {
	name         string
	startTime    time.Time
	minStartTime time.Duration
	isReady      atomic.Bool
}

// NewServiceReadinessProbe creates a new service readiness probe
func NewServiceReadinessProbe(name string, minStartTime time.Duration) *ServiceReadinessProbe {
	probe := &ServiceReadinessProbe{
		name:         name,
		startTime:    time.Now(),
		minStartTime: minStartTime,
	}
	probe.isReady.Store(false)
	return probe
}

// CheckReadiness checks if the service is ready
func (p *ServiceReadinessProbe) CheckReadiness(ctx context.Context) error {
	// Check minimum start time has elapsed
	if time.Since(p.startTime) < p.minStartTime {
		return fmt.Errorf("service %s still starting up", p.name)
	}

	// Check if manually marked as ready
	if !p.isReady.Load() {
		return fmt.Errorf("service %s not ready", p.name)
	}

	return nil
}

// GetName returns the probe name
func (p *ServiceReadinessProbe) GetName() string {
	return p.name
}

// SetReady marks the service as ready
func (p *ServiceReadinessProbe) SetReady(ready bool) {
	p.isReady.Store(ready)
}

// IsReady returns the current readiness state
func (p *ServiceReadinessProbe) IsReady() bool {
	return p.isReady.Load()
}

// ServiceLivenessProbe checks if a service is alive
type ServiceLivenessProbe struct {
	name      string
	lastPing  atomic.Int64
	timeout   time.Duration
	isRunning atomic.Bool
}

// NewServiceLivenessProbe creates a new service liveness probe
func NewServiceLivenessProbe(name string, timeout time.Duration) *ServiceLivenessProbe {
	probe := &ServiceLivenessProbe{
		name:    name,
		timeout: timeout,
	}
	probe.lastPing.Store(time.Now().UnixNano())
	probe.isRunning.Store(true)
	return probe
}

// CheckLiveness checks if the service is alive
func (p *ServiceLivenessProbe) CheckLiveness(ctx context.Context) error {
	if !p.isRunning.Load() {
		return fmt.Errorf("service %s is not running", p.name)
	}

	lastPingTime := time.Unix(0, p.lastPing.Load())
	if time.Since(lastPingTime) > p.timeout {
		return fmt.Errorf("service %s has not responded within %v", p.name, p.timeout)
	}

	return nil
}

// GetName returns the probe name
func (p *ServiceLivenessProbe) GetName() string {
	return p.name
}

// Ping updates the last ping time
func (p *ServiceLivenessProbe) Ping() {
	p.lastPing.Store(time.Now().UnixNano())
}

// SetRunning sets the running state
func (p *ServiceLivenessProbe) SetRunning(running bool) {
	p.isRunning.Store(running)
	if running {
		p.Ping()
	}
}

// IsRunning returns the current running state
func (p *ServiceLivenessProbe) IsRunning() bool {
	return p.isRunning.Load()
}

// StartHeartbeat starts a heartbeat goroutine
func (p *ServiceLivenessProbe) StartHeartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if p.isRunning.Load() {
					p.Ping()
				}
			}
		}
	}()
}

// DependencyHealthChecker checks the health of a dependency
type DependencyHealthChecker struct {
	name       string
	dependency HealthChecker
}

// NewDependencyHealthChecker creates a new dependency health checker
func NewDependencyHealthChecker(name string, dependency HealthChecker) *DependencyHealthChecker {
	return &DependencyHealthChecker{
		name:       name,
		dependency: dependency,
	}
}

// Check performs the health check
func (c *DependencyHealthChecker) Check(ctx context.Context) error {
	return c.dependency.Check(ctx)
}

// GetName returns the health checker name
func (c *DependencyHealthChecker) GetName() string {
	return c.name
}

// CompositeHealthChecker aggregates multiple health checkers
type CompositeHealthChecker struct {
	name     string
	checkers []HealthChecker
}

// NewCompositeHealthChecker creates a new composite health checker
func NewCompositeHealthChecker(name string, checkers ...HealthChecker) *CompositeHealthChecker {
	return &CompositeHealthChecker{
		name:     name,
		checkers: checkers,
	}
}

// Check performs all health checks and returns error if any fails
func (c *CompositeHealthChecker) Check(ctx context.Context) error {
	for _, checker := range c.checkers {
		if err := checker.Check(ctx); err != nil {
			return fmt.Errorf("%s check failed: %w", checker.GetName(), err)
		}
	}
	return nil
}

// GetName returns the health checker name
func (c *CompositeHealthChecker) GetName() string {
	return c.name
}

// AddChecker adds a health checker to the composite
func (c *CompositeHealthChecker) AddChecker(checker HealthChecker) {
	c.checkers = append(c.checkers, checker)
}
