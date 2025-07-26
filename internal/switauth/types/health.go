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

package types

import (
	sharedTypes "github.com/innovationmech/swit/pkg/types"
)

// Re-export health-related types from shared package for backward compatibility
type (
	HealthStatus     = sharedTypes.HealthStatus
	DependencyStatus = sharedTypes.DependencyStatus
	ServiceInfo      = sharedTypes.ServiceInfo
	BuildInfo        = sharedTypes.BuildInfo
)

// Re-export constants
const (
	HealthStatusHealthy   = sharedTypes.HealthStatusHealthy
	HealthStatusUnhealthy = sharedTypes.HealthStatusUnhealthy
	HealthStatusDegraded  = sharedTypes.HealthStatusDegraded

	DependencyStatusUp   = sharedTypes.DependencyStatusUp
	DependencyStatusDown = sharedTypes.DependencyStatusDown
	DependencyStatusSlow = sharedTypes.DependencyStatusSlow
)

// Re-export constructor functions
var (
	NewHealthStatus              = sharedTypes.NewHealthStatus
	NewDependencyStatus          = sharedTypes.NewDependencyStatus
	NewDependencyStatusWithError = sharedTypes.NewDependencyStatusWithError
	NewServiceInfo               = sharedTypes.NewServiceInfo
)
