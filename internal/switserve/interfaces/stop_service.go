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

	"github.com/innovationmech/swit/internal/switserve/types"
)

// ShutdownStatus represents the status of a shutdown operation
type ShutdownStatus struct {
	// Status indicates the current shutdown status
	Status types.Status `json:"status"`
	// Message provides additional information about the shutdown
	Message string `json:"message"`
	// InitiatedAt is the timestamp when shutdown was initiated
	InitiatedAt int64 `json:"initiated_at"`
	// CompletedAt is the timestamp when shutdown completed (if applicable)
	CompletedAt *int64 `json:"completed_at,omitempty"`
}

// StopService defines the interface for graceful service shutdown operations.
// This interface provides methods for initiating and monitoring the shutdown
// process of the application and its components.
//
// Future implementations should support:
// - Graceful shutdown with configurable timeouts
// - Component-specific shutdown ordering
// - Shutdown status monitoring
// - Emergency shutdown capabilities
// - Shutdown hooks for cleanup operations
type StopService interface {
	// InitiateShutdown begins the graceful shutdown process.
	// It coordinates the shutdown of all application components
	// in the proper order and returns the shutdown status.
	// Returns shutdown status information or an error if shutdown fails.
	InitiateShutdown(ctx context.Context) (*ShutdownStatus, error)
}
