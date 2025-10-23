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
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// SagaListRequest represents the request parameters for listing Sagas.
type SagaListRequest struct {
	// Page number (1-based), defaults to 1
	Page int `form:"page" json:"page" binding:"omitempty,min=1"`

	// Page size, defaults to 20
	PageSize int `form:"pageSize" json:"pageSize" binding:"omitempty,min=1,max=100"`

	// Filter by states (comma-separated or multiple params)
	States []string `form:"states" json:"states" binding:"omitempty"`

	// Filter by definition IDs (comma-separated or multiple params)
	DefinitionIDs []string `form:"definitionIds" json:"definitionIds" binding:"omitempty"`

	// Filter by time range
	CreatedAfter  *time.Time `form:"createdAfter" json:"createdAfter" time_format:"2006-01-02T15:04:05Z07:00" binding:"omitempty"`
	CreatedBefore *time.Time `form:"createdBefore" json:"createdBefore" time_format:"2006-01-02T15:04:05Z07:00" binding:"omitempty"`

	// Sort field: created_at, updated_at, state
	SortBy string `form:"sortBy" json:"sortBy" binding:"omitempty,oneof=created_at updated_at state"`

	// Sort order: asc, desc
	SortOrder string `form:"sortOrder" json:"sortOrder" binding:"omitempty,oneof=asc desc"`
}

// ApplyDefaults applies default values to the request.
func (r *SagaListRequest) ApplyDefaults() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 20
	}
	if r.SortBy == "" {
		r.SortBy = "created_at"
	}
	if r.SortOrder == "" {
		r.SortOrder = "desc"
	}
}

// ToSagaFilter converts the request to a saga.SagaFilter.
func (r *SagaListRequest) ToSagaFilter() *saga.SagaFilter {
	filter := &saga.SagaFilter{
		Limit:         r.PageSize,
		Offset:        (r.Page - 1) * r.PageSize,
		SortBy:        r.SortBy,
		SortOrder:     r.SortOrder,
		CreatedAfter:  r.CreatedAfter,
		CreatedBefore: r.CreatedBefore,
	}

	// Convert state strings to SagaState
	if len(r.States) > 0 {
		filter.States = make([]saga.SagaState, 0, len(r.States))
		for _, stateStr := range r.States {
			// Parse state string to SagaState
			state := parseSagaState(stateStr)
			if state != -1 { // Only add valid states
				filter.States = append(filter.States, state)
			}
		}
	}

	// Set definition IDs
	if len(r.DefinitionIDs) > 0 {
		filter.DefinitionIDs = r.DefinitionIDs
	}

	return filter
}

// parseSagaState converts a state string to saga.SagaState.
func parseSagaState(stateStr string) saga.SagaState {
	switch stateStr {
	case "Pending", "pending":
		return saga.StatePending
	case "Running", "running":
		return saga.StateRunning
	case "StepCompleted", "step_completed":
		return saga.StateStepCompleted
	case "Completed", "completed":
		return saga.StateCompleted
	case "Compensating", "compensating":
		return saga.StateCompensating
	case "Compensated", "compensated":
		return saga.StateCompensated
	case "Failed", "failed":
		return saga.StateFailed
	case "Cancelled", "cancelled":
		return saga.StateCancelled
	case "TimedOut", "timed_out":
		return saga.StateTimedOut
	default:
		return -1 // Invalid state
	}
}

// SagaDTO represents the data transfer object for a Saga instance.
type SagaDTO struct {
	// Basic information
	ID           string `json:"id"`
	DefinitionID string `json:"definitionId"`
	State        string `json:"state"`
	TraceID      string `json:"traceId,omitempty"`

	// Progress information
	CurrentStep    int `json:"currentStep"`
	TotalSteps     int `json:"totalSteps"`
	CompletedSteps int `json:"completedSteps"`

	// Time information
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	StartTime time.Time  `json:"startTime"`
	EndTime   *time.Time `json:"endTime,omitempty"`

	// Timeout information
	Timeout string `json:"timeout,omitempty"`

	// Status flags
	IsActive   bool `json:"isActive"`
	IsTerminal bool `json:"isTerminal"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Error information (only for failed Sagas)
	Error *SagaErrorDTO `json:"error,omitempty"`
}

// SagaErrorDTO represents error information for a Saga.
type SagaErrorDTO struct {
	Message   string                 `json:"message"`
	Code      string                 `json:"code,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Retryable bool                   `json:"retryable"`
}

// SagaDetailDTO represents detailed information about a Saga instance.
type SagaDetailDTO struct {
	*SagaDTO

	// Additional details
	Result interface{} `json:"result,omitempty"`

	// Execution context (if available)
	ExecutionDetails *ExecutionDetailsDTO `json:"executionDetails,omitempty"`
}

// ExecutionDetailsDTO provides additional execution context.
type ExecutionDetailsDTO struct {
	// Duration in milliseconds
	Duration int64 `json:"duration,omitempty"`

	// Whether compensation has been triggered
	CompensationTriggered bool `json:"compensationTriggered"`

	// Number of retries for current step
	Retries int `json:"retries,omitempty"`
}

// SagaListResponse represents the paginated response for Saga list.
type SagaListResponse struct {
	// List of Sagas
	Data []SagaDTO `json:"data"`

	// Pagination information
	Pagination PaginationDTO `json:"pagination"`
}

// PaginationDTO represents pagination metadata.
type PaginationDTO struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// FromSagaInstance converts a saga.SagaInstance to SagaDTO.
func FromSagaInstance(instance saga.SagaInstance) *SagaDTO {
	dto := &SagaDTO{
		ID:             instance.GetID(),
		DefinitionID:   instance.GetDefinitionID(),
		State:          instance.GetState().String(),
		TraceID:        instance.GetTraceID(),
		CurrentStep:    instance.GetCurrentStep(),
		TotalSteps:     instance.GetTotalSteps(),
		CompletedSteps: instance.GetCompletedSteps(),
		CreatedAt:      instance.GetCreatedAt(),
		UpdatedAt:      instance.GetUpdatedAt(),
		StartTime:      instance.GetStartTime(),
		IsActive:       instance.IsActive(),
		IsTerminal:     instance.IsTerminal(),
		Metadata:       instance.GetMetadata(),
	}

	// Set timeout as readable duration string
	if timeout := instance.GetTimeout(); timeout > 0 {
		dto.Timeout = timeout.String()
	}

	// Set end time if available
	if endTime := instance.GetEndTime(); !endTime.IsZero() {
		dto.EndTime = &endTime
	}

	// Set error if available
	if sagaErr := instance.GetError(); sagaErr != nil {
		dto.Error = &SagaErrorDTO{
			Message:   sagaErr.Message,
			Code:      sagaErr.Code,
			Details:   sagaErr.Details,
			Retryable: sagaErr.Retryable,
		}
	}

	return dto
}

// FromSagaInstanceDetailed converts a saga.SagaInstance to SagaDetailDTO.
func FromSagaInstanceDetailed(instance saga.SagaInstance) *SagaDetailDTO {
	dto := &SagaDetailDTO{
		SagaDTO: FromSagaInstance(instance),
		Result:  instance.GetResult(),
	}

	// Calculate execution details
	startTime := instance.GetStartTime()
	endTime := instance.GetEndTime()

	details := &ExecutionDetailsDTO{
		CompensationTriggered: instance.GetState() == saga.StateCompensating ||
			instance.GetState() == saga.StateCompensated,
	}

	// Calculate duration if Saga has ended
	if !endTime.IsZero() {
		duration := endTime.Sub(startTime)
		details.Duration = duration.Milliseconds()
	} else if !startTime.IsZero() {
		// For running Sagas, calculate current duration
		duration := time.Since(startTime)
		details.Duration = duration.Milliseconds()
	}

	dto.ExecutionDetails = details

	return dto
}

// FromSagaInstances converts a slice of saga.SagaInstance to []SagaDTO.
func FromSagaInstances(instances []saga.SagaInstance) []SagaDTO {
	dtos := make([]SagaDTO, 0, len(instances))
	for _, instance := range instances {
		if dto := FromSagaInstance(instance); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}

