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
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// SagaQueryAPI provides API endpoints for querying Saga instances.
type SagaQueryAPI struct {
	coordinator saga.SagaCoordinator
}

// NewSagaQueryAPI creates a new Saga query API handler.
func NewSagaQueryAPI(coordinator saga.SagaCoordinator) *SagaQueryAPI {
	return &SagaQueryAPI{
		coordinator: coordinator,
	}
}

// ListSagas handles GET /api/sagas - retrieves a paginated list of Saga instances.
//
// Query parameters:
//   - page: Page number (1-based), defaults to 1
//   - pageSize: Number of items per page (1-100), defaults to 20
//   - states: Filter by states (can be repeated or comma-separated)
//   - definitionIds: Filter by definition IDs (can be repeated or comma-separated)
//   - createdAfter: Filter by creation time (ISO 8601 format)
//   - createdBefore: Filter by creation time (ISO 8601 format)
//   - sortBy: Sort field (created_at, updated_at, state), defaults to created_at
//   - sortOrder: Sort order (asc, desc), defaults to desc
//
// Example:
//
//	GET /api/sagas?page=1&pageSize=20&states=Running&states=Completed&sortBy=created_at&sortOrder=desc
//
// Response:
//
//	{
//	  "data": [...],
//	  "pagination": {
//	    "page": 1,
//	    "pageSize": 20,
//	    "totalItems": 100,
//	    "totalPages": 5
//	  }
//	}
func (api *SagaQueryAPI) ListSagas(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse and validate request parameters
	var req SagaListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Invalid query parameters",
				zap.Error(err),
				zap.Any("query", c.Request.URL.Query()))
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Apply defaults
	req.ApplyDefaults()

	// Convert to saga filter
	filter := req.ToSagaFilter()

	// Log the query for debugging
	if logger.Logger != nil {
		logger.Logger.Debug("Listing Sagas",
			zap.Int("page", req.Page),
			zap.Int("pageSize", req.PageSize),
			zap.Strings("states", req.States),
			zap.String("sortBy", req.SortBy),
			zap.String("sortOrder", req.SortOrder))
	}

	// Query active Sagas from coordinator
	instances, err := api.coordinator.GetActiveSagas(filter)
	if err != nil {
		if logger.Logger != nil {
			logger.Logger.Error("Failed to get active Sagas",
				zap.Error(err),
				zap.Any("filter", filter))
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve Sagas: " + err.Error(),
		})
		return
	}

	// Apply additional filtering and sorting
	// Note: The coordinator may not fully support all filter options,
	// so we do post-processing here
	instances = api.applyFilters(instances, &req)

	// Calculate pagination
	totalItems := len(instances)
	totalPages := (totalItems + req.PageSize - 1) / req.PageSize
	if totalPages < 1 {
		totalPages = 1
	}

	// Apply pagination to results
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start >= totalItems {
		start = totalItems
	}
	if end > totalItems {
		end = totalItems
	}

	var pagedInstances []saga.SagaInstance
	if start < end {
		pagedInstances = instances[start:end]
	} else {
		pagedInstances = []saga.SagaInstance{}
	}

	// Convert to DTOs
	dtos := FromSagaInstances(pagedInstances)

	// Build response
	response := SagaListResponse{
		Data: dtos,
		Pagination: PaginationDTO{
			Page:       req.Page,
			PageSize:   req.PageSize,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}

	// Propagate trace context
	if traceID := c.GetHeader("X-Trace-ID"); traceID != "" {
		c.Header("X-Trace-ID", traceID)
	}

	// Log success
	if logger.Logger != nil {
		logger.Logger.Info("Successfully retrieved Sagas",
			zap.Int("count", len(dtos)),
			zap.Int("totalItems", totalItems),
			zap.Int("page", req.Page),
			zap.String("trace_id", c.GetHeader("X-Trace-ID")))
	}

	c.JSON(http.StatusOK, response)

	// Mark context as done
	_ = ctx
}

// GetSagaDetails handles GET /api/sagas/:id - retrieves detailed information about a specific Saga.
//
// Path parameters:
//   - id: The unique identifier of the Saga instance
//
// Example:
//
//	GET /api/sagas/saga-123456
//
// Response:
//
//	{
//	  "id": "saga-123456",
//	  "definitionId": "order-saga",
//	  "state": "Running",
//	  "currentStep": 2,
//	  "totalSteps": 5,
//	  ...
//	}
func (api *SagaQueryAPI) GetSagaDetails(c *gin.Context) {
	ctx := c.Request.Context()

	// Get Saga ID from path
	sagaID := c.Param("id")
	if sagaID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Saga ID is required",
		})
		return
	}

	// Log the query
	if logger.Logger != nil {
		logger.Logger.Debug("Getting Saga details",
			zap.String("sagaID", sagaID),
			zap.String("trace_id", c.GetHeader("X-Trace-ID")))
	}

	// Get Saga instance from coordinator
	instance, err := api.coordinator.GetSagaInstance(sagaID)
	if err != nil {
		// Check if it's a not found error
		if saga.IsSagaNotFound(err) {
			if logger.Logger != nil {
				logger.Logger.Warn("Saga not found",
					zap.String("sagaID", sagaID),
					zap.Error(err))
			}
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Saga not found: " + sagaID,
			})
			return
		}

		// Other errors
		if logger.Logger != nil {
			logger.Logger.Error("Failed to get Saga instance",
				zap.String("sagaID", sagaID),
				zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: "Failed to retrieve Saga details: " + err.Error(),
		})
		return
	}

	// Convert to detailed DTO
	dto := FromSagaInstanceDetailed(instance)

	// Propagate trace context
	if traceID := c.GetHeader("X-Trace-ID"); traceID != "" {
		c.Header("X-Trace-ID", traceID)
	}

	// Log success
	if logger.Logger != nil {
		logger.Logger.Info("Successfully retrieved Saga details",
			zap.String("sagaID", sagaID),
			zap.String("state", dto.State),
			zap.String("trace_id", c.GetHeader("X-Trace-ID")))
	}

	c.JSON(http.StatusOK, dto)

	// Mark context as done
	_ = ctx
}

// applyFilters applies additional client-side filtering that the coordinator might not support.
func (api *SagaQueryAPI) applyFilters(instances []saga.SagaInstance, req *SagaListRequest) []saga.SagaInstance {
	// If no additional filtering is needed, return as-is
	if len(req.States) == 0 && len(req.DefinitionIDs) == 0 &&
		req.CreatedAfter == nil && req.CreatedBefore == nil {
		return instances
	}

	filtered := make([]saga.SagaInstance, 0, len(instances))

	for _, instance := range instances {
		// Filter by states
		if len(req.States) > 0 {
			stateMatch := false
			instanceState := instance.GetState().String()
			for _, state := range req.States {
				if instanceState == state {
					stateMatch = true
					break
				}
			}
			if !stateMatch {
				continue
			}
		}

		// Filter by definition IDs
		if len(req.DefinitionIDs) > 0 {
			defMatch := false
			instanceDefID := instance.GetDefinitionID()
			for _, defID := range req.DefinitionIDs {
				if instanceDefID == defID {
					defMatch = true
					break
				}
			}
			if !defMatch {
				continue
			}
		}

		// Filter by created after
		if req.CreatedAfter != nil {
			if instance.GetCreatedAt().Before(*req.CreatedAfter) {
				continue
			}
		}

		// Filter by created before
		if req.CreatedBefore != nil {
			if instance.GetCreatedAt().After(*req.CreatedBefore) {
				continue
			}
		}

		filtered = append(filtered, instance)
	}

	return filtered
}
