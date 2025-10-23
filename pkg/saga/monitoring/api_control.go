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

package monitoring

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
	"go.uber.org/zap"
)

// refCountedMutex is a mutex wrapper that tracks how many goroutines are using it.
// This prevents the race condition where a mutex is deleted from the map while still in use.
type refCountedMutex struct {
	mu    sync.Mutex
	count int64 // Reference count using atomic operations
}

// acquire increments the reference count and returns the underlying mutex.
func (rcm *refCountedMutex) acquire() *sync.Mutex {
	atomic.AddInt64(&rcm.count, 1)
	return &rcm.mu
}

// release decrements the reference count and returns the new count.
func (rcm *refCountedMutex) release() int64 {
	return atomic.AddInt64(&rcm.count, -1)
}

// canDelete returns true if no goroutines are currently referencing this mutex.
func (rcm *refCountedMutex) canDelete() bool {
	return atomic.LoadInt64(&rcm.count) == 0
}

// SagaControlAPI provides API endpoints for controlling Saga instances.
// It supports operations like cancel and retry, with proper validation and logging.
type SagaControlAPI struct {
	coordinator saga.SagaCoordinator
	validator   *OperationValidator

	// recoveryManager provides retry functionality
	// This is optional - if not provided, we use coordinator's CancelSaga directly
	recoveryManager *state.RecoveryManager

	// operationLock prevents concurrent operations on the same Saga
	operationLock sync.Map // sagaID -> *refCountedMutex
}

// NewSagaControlAPI creates a new Saga control API handler.
//
// Parameters:
//   - coordinator: The Saga coordinator for control operations.
//   - recoveryManager: Optional recovery manager for advanced retry operations.
//
// Returns:
//   - A configured SagaControlAPI ready to handle control requests.
func NewSagaControlAPI(coordinator saga.SagaCoordinator, recoveryManager *state.RecoveryManager) *SagaControlAPI {
	return &SagaControlAPI{
		coordinator:     coordinator,
		validator:       NewOperationValidator(coordinator),
		recoveryManager: recoveryManager,
	}
}

// CancelSagaRequest represents the request body for cancelling a Saga.
type CancelSagaRequest struct {
	// Reason for cancellation (required for audit purposes)
	Reason string `json:"reason" binding:"required,min=1,max=500"`

	// Force cancellation even if the Saga is in an unusual state
	Force bool `json:"force,omitempty"`
}

// RetrySagaRequest represents the request body for retrying a Saga.
type RetrySagaRequest struct {
	// FromStep specifies which step to retry from (0-based index)
	// If not specified or -1, retry from the current/last failed step
	FromStep int `json:"fromStep" binding:"omitempty,min=-1"`

	// Reason for retry (optional, for audit purposes)
	Reason string `json:"reason,omitempty" binding:"omitempty,max=500"`
}

// OperationResponse represents the response for control operations.
type OperationResponse struct {
	// Success indicates if the operation succeeded
	Success bool `json:"success"`

	// Message provides a human-readable result message
	Message string `json:"message"`

	// OperationID is a unique identifier for this operation (for audit/tracking)
	OperationID string `json:"operationId,omitempty"`

	// Timestamp when the operation was executed
	Timestamp time.Time `json:"timestamp"`

	// SagaState is the new state of the Saga after the operation
	SagaState string `json:"sagaState,omitempty"`

	// Details provides additional operation details
	Details map[string]interface{} `json:"details,omitempty"`
}

// CancelSaga handles POST /api/sagas/:id/cancel - cancels a running Saga.
//
// Request body:
//
//	{
//	  "reason": "User requested cancellation",
//	  "force": false
//	}
//
// Response:
//
//	{
//	  "success": true,
//	  "message": "Saga cancelled successfully",
//	  "operationId": "cancel-saga123-1234567890",
//	  "timestamp": "2025-10-23T10:30:00Z",
//	  "sagaState": "Compensating"
//	}
func (api *SagaControlAPI) CancelSaga(c *gin.Context) {
	ctx := c.Request.Context()
	sagaID := c.Param("id")
	startTime := time.Now()

	if logger.Logger != nil {
		logger.Logger.Info("Cancel saga request received",
			zap.String("saga_id", sagaID))
	}

	// Parse request body
	var req CancelSagaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Invalid cancel request body",
				zap.String("saga_id", sagaID),
				zap.Error(err))
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Acquire operation lock for this Saga
	if err := api.acquireOperationLock(sagaID); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Concurrent operation detected",
				zap.String("saga_id", sagaID))
		}
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "concurrent_operation",
			Message: "Another operation is in progress for this Saga",
		})
		return
	}
	defer api.releaseOperationLock(sagaID)

	// Validate permission
	if err := api.validator.ValidatePermission(ctx, "cancel", sagaID); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Permission denied",
				zap.String("saga_id", sagaID),
				zap.Error(err))
		}
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "permission_denied",
			Message: "You do not have permission to cancel this Saga",
		})
		return
	}

	// Validate cancel operation
	instance, err := api.validator.ValidateCancelOperation(ctx, sagaID, req.Reason)
	if err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Cancel operation validation failed",
				zap.String("saga_id", sagaID),
				zap.Error(err))
		}

		statusCode := http.StatusBadRequest
		errorCode := "validation_failed"

		if errors.Is(err, ErrSagaNotFound) {
			statusCode = http.StatusNotFound
			errorCode = "saga_not_found"
		} else if errors.Is(err, ErrSagaAlreadyTerminal) {
			statusCode = http.StatusConflict
			errorCode = "saga_already_terminal"
		} else if errors.Is(err, ErrInvalidSagaState) {
			statusCode = http.StatusConflict
			errorCode = "invalid_state"
		}

		c.JSON(statusCode, ErrorResponse{
			Error:   errorCode,
			Message: err.Error(),
		})
		return
	}

	// Perform cancel operation
	if err := api.coordinator.CancelSaga(ctx, sagaID, req.Reason); err != nil {
		if logger.Logger != nil {
			logger.Logger.Error("Failed to cancel saga",
				zap.String("saga_id", sagaID),
				zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "cancel_failed",
			Message: "Failed to cancel Saga: " + err.Error(),
		})
		return
	}

	// Get updated Saga state
	updatedInstance, _ := api.coordinator.GetSagaInstance(sagaID)
	newState := "Unknown"
	if updatedInstance != nil {
		newState = updatedInstance.GetState().String()
	}

	duration := time.Since(startTime)
	operationID := generateOperationID("cancel", sagaID)

	if logger.Logger != nil {
		logger.Logger.Info("Saga cancelled successfully",
			zap.String("saga_id", sagaID),
			zap.String("reason", sanitizeForLog(req.Reason)),
			zap.String("new_state", newState),
			zap.Duration("duration", duration),
			zap.String("operation_id", operationID))
	}

	response := OperationResponse{
		Success:     true,
		Message:     "Saga cancelled successfully",
		OperationID: operationID,
		Timestamp:   time.Now(),
		SagaState:   newState,
		Details: map[string]interface{}{
			"reason":            req.Reason,
			"previous_state":    instance.GetState().String(),
			"operation_time_ms": duration.Milliseconds(),
		},
	}

	c.JSON(http.StatusOK, response)
}

// RetrySaga handles POST /api/sagas/:id/retry - retries a failed Saga.
//
// Request body:
//
//	{
//	  "fromStep": 2,
//	  "reason": "Transient error resolved"
//	}
//
// Response:
//
//	{
//	  "success": true,
//	  "message": "Saga retry initiated successfully",
//	  "operationId": "retry-saga123-1234567890",
//	  "timestamp": "2025-10-23T10:30:00Z",
//	  "sagaState": "Running"
//	}
func (api *SagaControlAPI) RetrySaga(c *gin.Context) {
	ctx := c.Request.Context()
	sagaID := c.Param("id")
	startTime := time.Now()

	if logger.Logger != nil {
		logger.Logger.Info("Retry saga request received",
			zap.String("saga_id", sagaID))
	}

	// Parse request body
	var req RetrySagaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Invalid retry request body",
				zap.String("saga_id", sagaID),
				zap.Error(err))
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Default to -1 (retry from current/last failed step) if not specified
	if req.FromStep < -1 {
		req.FromStep = -1
	}

	// Acquire operation lock for this Saga
	if err := api.acquireOperationLock(sagaID); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Concurrent operation detected",
				zap.String("saga_id", sagaID))
		}
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "concurrent_operation",
			Message: "Another operation is in progress for this Saga",
		})
		return
	}
	defer api.releaseOperationLock(sagaID)

	// Validate permission
	if err := api.validator.ValidatePermission(ctx, "retry", sagaID); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Permission denied",
				zap.String("saga_id", sagaID),
				zap.Error(err))
		}
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "permission_denied",
			Message: "You do not have permission to retry this Saga",
		})
		return
	}

	// Validate retry operation
	instance, err := api.validator.ValidateRetryOperation(ctx, sagaID, req.FromStep)
	if err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Retry operation validation failed",
				zap.String("saga_id", sagaID),
				zap.Int("from_step", req.FromStep),
				zap.Error(err))
		}

		statusCode := http.StatusBadRequest
		errorCode := "validation_failed"

		if errors.Is(err, ErrSagaNotFound) {
			statusCode = http.StatusNotFound
			errorCode = "saga_not_found"
		} else if errors.Is(err, ErrInvalidSagaState) {
			statusCode = http.StatusConflict
			errorCode = "invalid_state"
		}

		c.JSON(statusCode, ErrorResponse{
			Error:   errorCode,
			Message: err.Error(),
		})
		return
	}

	// Perform retry operation
	var retryErr error
	if api.recoveryManager != nil && req.FromStep >= 0 {
		// Use recovery manager for advanced retry with specific step
		retryErr = api.recoveryManager.RetrySaga(ctx, sagaID, req.FromStep)
	} else if api.recoveryManager != nil {
		// Use recovery manager for standard recovery
		retryErr = api.recoveryManager.RecoverSaga(ctx, sagaID)
	} else {
		// Fallback: If no recovery manager, we can't directly retry
		// This is a limitation - ideally coordinator should have a Retry method
		retryErr = errors.New("retry operation requires recovery manager")
	}

	if retryErr != nil {
		if logger.Logger != nil {
			logger.Logger.Error("Failed to retry saga",
				zap.String("saga_id", sagaID),
				zap.Int("from_step", req.FromStep),
				zap.Error(retryErr))
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "retry_failed",
			Message: "Failed to retry Saga: " + retryErr.Error(),
		})
		return
	}

	// Get updated Saga state
	updatedInstance, _ := api.coordinator.GetSagaInstance(sagaID)
	newState := "Unknown"
	if updatedInstance != nil {
		newState = updatedInstance.GetState().String()
	}

	duration := time.Since(startTime)
	operationID := generateOperationID("retry", sagaID)

	if logger.Logger != nil {
		logger.Logger.Info("Saga retry initiated successfully",
			zap.String("saga_id", sagaID),
			zap.Int("from_step", req.FromStep),
			zap.String("reason", sanitizeForLog(req.Reason)),
			zap.String("new_state", newState),
			zap.Duration("duration", duration),
			zap.String("operation_id", operationID))
	}

	response := OperationResponse{
		Success:     true,
		Message:     "Saga retry initiated successfully",
		OperationID: operationID,
		Timestamp:   time.Now(),
		SagaState:   newState,
		Details: map[string]interface{}{
			"from_step":         req.FromStep,
			"reason":            req.Reason,
			"previous_state":    instance.GetState().String(),
			"operation_time_ms": duration.Milliseconds(),
		},
	}

	c.JSON(http.StatusOK, response)
}

// acquireOperationLock acquires a lock for the given Saga to prevent concurrent operations.
func (api *SagaControlAPI) acquireOperationLock(sagaID string) error {
	// Get or create ref-counted mutex for this Saga
	rcmInterface, _ := api.operationLock.LoadOrStore(sagaID, &refCountedMutex{})
	rcm := rcmInterface.(*refCountedMutex)

	// Acquire reference to the underlying mutex
	mutex := rcm.acquire()

	// Try to acquire lock (non-blocking)
	if !mutex.TryLock() {
		// Release our reference since we failed to acquire the lock
		rcm.release()
		return ErrConcurrentOperation
	}

	return nil
}

// releaseOperationLock releases the lock for the given Saga.
func (api *SagaControlAPI) releaseOperationLock(sagaID string) {
	rcmInterface, ok := api.operationLock.Load(sagaID)
	if !ok {
		return
	}

	rcm := rcmInterface.(*refCountedMutex)

	// Get the underlying mutex and unlock it
	mutex := &rcm.mu
	mutex.Unlock()

	// Release our reference to the mutex
	newCount := rcm.release()

	// If this was the last reference, schedule cleanup
	if newCount == 0 {
		// Schedule cleanup in a goroutine to avoid blocking the current request
		go func() {
			// Add a small delay to allow any pending operations to acquire the mutex
			// This prevents immediate deletion while other operations might be starting
			time.Sleep(10 * time.Millisecond)

			// Double-check that no new references have been added during the delay
			if rcm.canDelete() {
				api.operationLock.Delete(sagaID)
			}
		}()
	}
}

// generateOperationID generates a unique operation ID for audit/tracking purposes.
func generateOperationID(operation, sagaID string) string {
	return operation + "-" + sagaID + "-" + time.Now().Format("20060102150405")
}
