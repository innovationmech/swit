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

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// SagaVisualizationAPI provides API endpoints for Saga flow visualization.
type SagaVisualizationAPI struct {
	coordinator saga.SagaCoordinator
	visualizer  *SagaVisualizer
}

// NewSagaVisualizationAPI creates a new Saga visualization API handler.
func NewSagaVisualizationAPI(coordinator saga.SagaCoordinator) *SagaVisualizationAPI {
	return &SagaVisualizationAPI{
		coordinator: coordinator,
		visualizer:  NewSagaVisualizer(coordinator),
	}
}

// GetVisualization handles GET /api/sagas/:id/visualization - retrieves visualization data for a Saga.
//
// Path parameters:
//   - id: The unique identifier of the Saga instance
//
// Example:
//
//	GET /api/sagas/saga-123456/visualization
//
// Response:
//
//	{
//	  "sagaId": "saga-123456",
//	  "definitionId": "order-saga",
//	  "flowType": "orchestration",
//	  "state": "Running",
//	  "nodes": [
//	    {
//	      "id": "start",
//	      "type": "start",
//	      "label": "Start",
//	      "state": "active"
//	    },
//	    {
//	      "id": "step-0",
//	      "type": "step",
//	      "label": "Reserve Inventory",
//	      "state": "completed",
//	      "stepIndex": 0,
//	      "attempts": 1,
//	      "startTime": "2025-10-23T10:00:00Z",
//	      "endTime": "2025-10-23T10:00:02Z",
//	      "duration": 2000
//	    },
//	    ...
//	  ],
//	  "edges": [
//	    {
//	      "id": "start-to-step-0",
//	      "type": "sequential",
//	      "source": "start",
//	      "target": "step-0",
//	      "active": true
//	    },
//	    ...
//	  ],
//	  "currentStep": 2,
//	  "totalSteps": 5,
//	  "completedSteps": 2,
//	  "compensationActive": false,
//	  "progress": 40.0,
//	  "generatedAt": "2025-10-23T10:00:10Z"
//	}
func (api *SagaVisualizationAPI) GetVisualization(c *gin.Context) {
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

	// Log the request
	if logger.Logger != nil {
		logger.Logger.Debug("Getting Saga visualization",
			zap.String("sagaID", sanitizeForLog(sagaID)),
			zap.String("trace_id", sanitizeForLog(c.GetHeader("X-Trace-ID"))))
	}

	// Generate visualization data
	vizData, err := api.visualizer.GenerateVisualization(ctx, sagaID)
	if err != nil {
		// Check if it's a not found error - check both with errors.As for wrapped errors
		// and with IsSagaNotFound for direct SagaError
		var sagaErr *saga.SagaError
		isNotFound := saga.IsSagaNotFound(err) ||
			(errors.As(err, &sagaErr) && sagaErr.Code == saga.ErrCodeSagaNotFound)

		if isNotFound {
			if logger.Logger != nil {
				logger.Logger.Warn("Saga not found for visualization",
					zap.String("sagaID", sanitizeForLog(sagaID)),
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
			logger.Logger.Error("Failed to generate visualization data",
				zap.String("sagaID", sanitizeForLog(sagaID)),
				zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "visualization_failed",
			Message: "Failed to generate visualization: " + err.Error(),
		})
		return
	}

	// Propagate trace context
	if traceID := c.GetHeader("X-Trace-ID"); traceID != "" {
		c.Header("X-Trace-ID", traceID)
	}

	// Log success
	if logger.Logger != nil {
		logger.Logger.Info("Successfully generated visualization",
			zap.String("sagaID", sanitizeForLog(sagaID)),
			zap.String("flowType", string(vizData.FlowType)),
			zap.Int("nodeCount", len(vizData.Nodes)),
			zap.Int("edgeCount", len(vizData.Edges)),
			zap.String("trace_id", sanitizeForLog(c.GetHeader("X-Trace-ID"))))
	}

	c.JSON(http.StatusOK, vizData)
}
