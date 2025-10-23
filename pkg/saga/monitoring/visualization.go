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
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// NodeType represents the type of a visualization node.
type NodeType string

const (
	// NodeTypeStart represents the start node of a Saga flow.
	NodeTypeStart NodeType = "start"

	// NodeTypeStep represents a normal execution step.
	NodeTypeStep NodeType = "step"

	// NodeTypeCompensation represents a compensation step.
	NodeTypeCompensation NodeType = "compensation"

	// NodeTypeEnd represents the end node of a Saga flow.
	NodeTypeEnd NodeType = "end"

	// NodeTypeDecision represents a decision/branching point.
	NodeTypeDecision NodeType = "decision"
)

// NodeState represents the execution state of a node.
type NodeState string

const (
	// NodeStatePending indicates the node is not yet executed.
	NodeStatePending NodeState = "pending"

	// NodeStateRunning indicates the node is currently executing.
	NodeStateRunning NodeState = "running"

	// NodeStateCompleted indicates the node has completed successfully.
	NodeStateCompleted NodeState = "completed"

	// NodeStateFailed indicates the node has failed.
	NodeStateFailed NodeState = "failed"

	// NodeStateCompensating indicates compensation is in progress.
	NodeStateCompensating NodeState = "compensating"

	// NodeStateCompensated indicates compensation has completed.
	NodeStateCompensated NodeState = "compensated"

	// NodeStateSkipped indicates the node was skipped.
	NodeStateSkipped NodeState = "skipped"

	// NodeStateActive indicates the node is currently active (for start/end nodes).
	NodeStateActive NodeState = "active"
)

// EdgeType represents the type of a connection between nodes.
type EdgeType string

const (
	// EdgeTypeSequential represents a sequential flow from one step to the next.
	EdgeTypeSequential EdgeType = "sequential"

	// EdgeTypeCompensation represents a compensation flow.
	EdgeTypeCompensation EdgeType = "compensation"

	// EdgeTypeConditional represents a conditional branch.
	EdgeTypeConditional EdgeType = "conditional"
)

// FlowType represents the type of Saga flow pattern.
type FlowType string

const (
	// FlowTypeOrchestration represents centralized orchestration pattern.
	FlowTypeOrchestration FlowType = "orchestration"

	// FlowTypeChoreography represents distributed choreography pattern.
	FlowTypeChoreography FlowType = "choreography"
)

// VisualizationNode represents a node in the Saga flow visualization.
type VisualizationNode struct {
	// ID is the unique identifier of the node.
	ID string `json:"id"`

	// Type is the type of the node.
	Type NodeType `json:"type"`

	// Label is the human-readable label for the node.
	Label string `json:"label"`

	// Description provides additional details about the node.
	Description string `json:"description,omitempty"`

	// State is the current execution state of the node.
	State NodeState `json:"state"`

	// StepIndex is the index of the step in the definition (for step nodes).
	StepIndex *int `json:"stepIndex,omitempty"`

	// Attempts is the number of execution attempts for this node.
	Attempts int `json:"attempts,omitempty"`

	// MaxAttempts is the maximum number of retry attempts allowed.
	MaxAttempts int `json:"maxAttempts,omitempty"`

	// StartTime is when the node started execution.
	StartTime *time.Time `json:"startTime,omitempty"`

	// EndTime is when the node completed execution.
	EndTime *time.Time `json:"endTime,omitempty"`

	// Duration is the execution duration in milliseconds.
	Duration *int64 `json:"duration,omitempty"`

	// Error contains error information if the node failed.
	Error *NodeError `json:"error,omitempty"`

	// Metadata contains additional node-specific metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Position provides hints for visual layout (optional).
	Position *NodePosition `json:"position,omitempty"`
}

// NodeError represents error information for a failed node.
type NodeError struct {
	// Message is the error message.
	Message string `json:"message"`

	// Code is the error code.
	Code string `json:"code,omitempty"`

	// Retryable indicates if the error is retryable.
	Retryable bool `json:"retryable"`
}

// NodePosition provides layout hints for visualization.
type NodePosition struct {
	// X coordinate (optional, can be computed by frontend).
	X float64 `json:"x,omitempty"`

	// Y coordinate (optional, can be computed by frontend).
	Y float64 `json:"y,omitempty"`

	// Layer is the logical layer/rank for hierarchical layout.
	Layer int `json:"layer"`
}

// VisualizationEdge represents a connection between nodes in the flow.
type VisualizationEdge struct {
	// ID is the unique identifier of the edge.
	ID string `json:"id"`

	// Type is the type of the edge.
	Type EdgeType `json:"type"`

	// Source is the ID of the source node.
	Source string `json:"source"`

	// Target is the ID of the target node.
	Target string `json:"target"`

	// Label provides additional context for the edge (optional).
	Label string `json:"label,omitempty"`

	// Condition describes the condition for conditional edges (optional).
	Condition string `json:"condition,omitempty"`

	// Active indicates if this edge is part of the current execution path.
	Active bool `json:"active"`

	// Metadata contains additional edge-specific metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// VisualizationData represents the complete visualization data for a Saga flow.
type VisualizationData struct {
	// SagaID is the ID of the Saga instance.
	SagaID string `json:"sagaId"`

	// DefinitionID is the ID of the Saga definition.
	DefinitionID string `json:"definitionId"`

	// FlowType indicates the pattern type (orchestration/choreography).
	FlowType FlowType `json:"flowType"`

	// State is the current state of the Saga.
	State string `json:"state"`

	// Nodes contains all nodes in the flow.
	Nodes []VisualizationNode `json:"nodes"`

	// Edges contains all connections between nodes.
	Edges []VisualizationEdge `json:"edges"`

	// CurrentStep is the index of the currently executing step.
	CurrentStep int `json:"currentStep"`

	// TotalSteps is the total number of steps in the definition.
	TotalSteps int `json:"totalSteps"`

	// CompletedSteps is the number of completed steps.
	CompletedSteps int `json:"completedSteps"`

	// CompensationActive indicates if compensation is currently active.
	CompensationActive bool `json:"compensationActive"`

	// Progress is the completion percentage (0-100).
	Progress float64 `json:"progress"`

	// Metadata contains additional Saga-level metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// GeneratedAt is when this visualization data was generated.
	GeneratedAt time.Time `json:"generatedAt"`
}

// SagaVisualizer generates visualization data for Saga instances.
type SagaVisualizer struct {
	coordinator saga.SagaCoordinator
}

// NewSagaVisualizer creates a new Saga visualizer.
func NewSagaVisualizer(coordinator saga.SagaCoordinator) *SagaVisualizer {
	return &SagaVisualizer{
		coordinator: coordinator,
	}
}

// GenerateVisualization generates visualization data for a Saga instance.
//
// Parameters:
//   - ctx: Context for the operation.
//   - sagaID: The unique identifier of the Saga instance.
//
// Returns:
//   - Visualization data ready for rendering.
//   - An error if the Saga is not found or generation fails.
func (v *SagaVisualizer) GenerateVisualization(ctx context.Context, sagaID string) (*VisualizationData, error) {
	// Get Saga instance
	instance, err := v.coordinator.GetSagaInstance(sagaID)
	if err != nil {
		if logger.Logger != nil {
			logger.Logger.Error("Failed to get Saga instance for visualization",
				zap.String("sagaID", sanitizeForLog(sagaID)),
				zap.Error(err))
		}
		return nil, fmt.Errorf("failed to get Saga instance: %w", err)
	}

	// Get Saga definition
	definition, err := v.getSagaDefinition(instance)
	if err != nil {
		if logger.Logger != nil {
			logger.Logger.Error("Failed to get Saga definition",
				zap.String("sagaID", sanitizeForLog(sagaID)),
				zap.String("definitionID", instance.GetDefinitionID()),
				zap.Error(err))
		}
		return nil, fmt.Errorf("failed to get Saga definition: %w", err)
	}

	// Determine flow type
	flowType := v.determineFlowType(definition)

	// Get step states
	stepStates := v.getStepStates(ctx, sagaID, definition)

	// Generate nodes
	nodes := v.generateNodes(instance, definition, stepStates)

	// Generate edges
	edges := v.generateEdges(instance, definition, stepStates)

	// Calculate progress
	progress := v.calculateProgress(instance)

	// Check if compensation is active
	compensationActive := v.isCompensationActive(instance, stepStates)

	// Build visualization data
	vizData := &VisualizationData{
		SagaID:             sagaID,
		DefinitionID:       instance.GetDefinitionID(),
		FlowType:           flowType,
		State:              instance.GetState().String(),
		Nodes:              nodes,
		Edges:              edges,
		CurrentStep:        instance.GetCurrentStep(),
		TotalSteps:         len(definition.GetSteps()),
		CompletedSteps:     v.countCompletedSteps(stepStates),
		CompensationActive: compensationActive,
		Progress:           progress,
		Metadata:           instance.GetMetadata(),
		GeneratedAt:        time.Now(),
	}

	if logger.Logger != nil {
		logger.Logger.Debug("Generated visualization data",
			zap.String("sagaID", sanitizeForLog(sagaID)),
			zap.Int("nodeCount", len(nodes)),
			zap.Int("edgeCount", len(edges)),
			zap.String("flowType", string(flowType)))
	}

	return vizData, nil
}

// getSagaDefinition retrieves the Saga definition for an instance.
func (v *SagaVisualizer) getSagaDefinition(instance saga.SagaInstance) (saga.SagaDefinition, error) {
	// Try to get definition from coordinator if it implements the interface
	if defProvider, ok := v.coordinator.(interface {
		GetDefinition(string) (saga.SagaDefinition, error)
	}); ok {
		return defProvider.GetDefinition(instance.GetDefinitionID())
	}

	// Fallback: try to extract from instance metadata
	// This is a workaround for coordinators that don't expose definitions
	return nil, fmt.Errorf("coordinator does not provide definition access")
}

// determineFlowType determines whether the Saga uses orchestration or choreography pattern.
func (v *SagaVisualizer) determineFlowType(definition saga.SagaDefinition) FlowType {
	// Check metadata for flow type hint
	if definition != nil {
		metadata := definition.GetMetadata()
		if flowType, ok := metadata["flowType"].(string); ok {
			if flowType == "choreography" {
				return FlowTypeChoreography
			}
		}
	}

	// Default to orchestration (most common pattern in this codebase)
	return FlowTypeOrchestration
}

// getStepStates retrieves the state information for all steps.
func (v *SagaVisualizer) getStepStates(ctx context.Context, sagaID string, definition saga.SagaDefinition) []*saga.StepState {
	if definition == nil {
		return nil
	}

	steps := definition.GetSteps()
	stepStates := make([]*saga.StepState, 0, len(steps))

	// Try to get step states from storage if coordinator supports it
	if stateProvider, ok := v.coordinator.(interface {
		GetStepStates(context.Context, string) ([]*saga.StepState, error)
	}); ok {
		states, err := stateProvider.GetStepStates(ctx, sagaID)
		if err == nil && states != nil {
			return states
		}
	}

	// Fallback: return empty states (will use instance-level info only)
	return stepStates
}

// generateNodes creates visualization nodes for the Saga flow.
func (v *SagaVisualizer) generateNodes(
	instance saga.SagaInstance,
	definition saga.SagaDefinition,
	stepStates []*saga.StepState,
) []VisualizationNode {
	nodes := make([]VisualizationNode, 0)

	// Add start node
	startNode := VisualizationNode{
		ID:    "start",
		Type:  NodeTypeStart,
		Label: "Start",
		State: v.getStartNodeState(instance),
		Position: &NodePosition{
			Layer: 0,
		},
	}
	nodes = append(nodes, startNode)

	// Add step nodes
	if definition != nil {
		steps := definition.GetSteps()
		currentStep := instance.GetCurrentStep()

		for i, step := range steps {
			stepNode := v.createStepNode(instance, step, i, currentStep, stepStates)
			nodes = append(nodes, stepNode)

			// Add compensation node if applicable
			if v.shouldShowCompensation(instance, i, stepStates) {
				compNode := v.createCompensationNode(instance, step, i, stepStates)
				nodes = append(nodes, compNode)
			}
		}
	}

	// Add end node
	endNode := VisualizationNode{
		ID:    "end",
		Type:  NodeTypeEnd,
		Label: "End",
		State: v.getEndNodeState(instance),
		Position: &NodePosition{
			Layer: len(nodes),
		},
	}
	nodes = append(nodes, endNode)

	return nodes
}

// createStepNode creates a visualization node for a step.
func (v *SagaVisualizer) createStepNode(
	instance saga.SagaInstance,
	step saga.SagaStep,
	stepIndex int,
	currentStep int,
	stepStates []*saga.StepState,
) VisualizationNode {
	node := VisualizationNode{
		ID:          fmt.Sprintf("step-%d", stepIndex),
		Type:        NodeTypeStep,
		Label:       step.GetName(),
		Description: step.GetDescription(),
		StepIndex:   &stepIndex,
		State:       v.getStepNodeState(instance, stepIndex, currentStep, stepStates),
		Metadata:    step.GetMetadata(),
		Position: &NodePosition{
			Layer: stepIndex + 1,
		},
	}

	// Add state-specific information
	if stepIndex < len(stepStates) && stepStates[stepIndex] != nil {
		stepState := stepStates[stepIndex]
		node.Attempts = stepState.Attempts
		node.MaxAttempts = stepState.MaxAttempts
		node.StartTime = stepState.StartedAt
		node.EndTime = stepState.CompletedAt

		// Calculate duration
		if stepState.StartedAt != nil && stepState.CompletedAt != nil {
			duration := stepState.CompletedAt.Sub(*stepState.StartedAt).Milliseconds()
			node.Duration = &duration
		}

		// Add error information
		if stepState.Error != nil {
			node.Error = &NodeError{
				Message:   stepState.Error.Message,
				Code:      stepState.Error.Code,
				Retryable: stepState.Error.Retryable,
			}
		}
	}

	return node
}

// createCompensationNode creates a visualization node for a compensation step.
func (v *SagaVisualizer) createCompensationNode(
	instance saga.SagaInstance,
	step saga.SagaStep,
	stepIndex int,
	stepStates []*saga.StepState,
) VisualizationNode {
	node := VisualizationNode{
		ID:          fmt.Sprintf("compensation-%d", stepIndex),
		Type:        NodeTypeCompensation,
		Label:       fmt.Sprintf("Compensate: %s", step.GetName()),
		Description: fmt.Sprintf("Compensation for %s", step.GetDescription()),
		StepIndex:   &stepIndex,
		State:       v.getCompensationNodeState(stepIndex, stepStates),
		Position: &NodePosition{
			Layer: stepIndex + 1,
		},
	}

	// Add compensation-specific information
	if stepIndex < len(stepStates) && stepStates[stepIndex] != nil {
		stepState := stepStates[stepIndex]
		if stepState.CompensationState != nil {
			compState := stepState.CompensationState
			node.Attempts = compState.Attempts
			node.MaxAttempts = compState.MaxAttempts
			node.StartTime = compState.StartedAt
			node.EndTime = compState.CompletedAt

			// Calculate duration
			if compState.StartedAt != nil && compState.CompletedAt != nil {
				duration := compState.CompletedAt.Sub(*compState.StartedAt).Milliseconds()
				node.Duration = &duration
			}

			// Add error information
			if compState.Error != nil {
				node.Error = &NodeError{
					Message:   compState.Error.Message,
					Code:      compState.Error.Code,
					Retryable: compState.Error.Retryable,
				}
			}
		}
	}

	return node
}

// generateEdges creates visualization edges connecting the nodes.
func (v *SagaVisualizer) generateEdges(
	instance saga.SagaInstance,
	definition saga.SagaDefinition,
	stepStates []*saga.StepState,
) []VisualizationEdge {
	edges := make([]VisualizationEdge, 0)

	if definition == nil {
		return edges
	}

	steps := definition.GetSteps()
	currentStep := instance.GetCurrentStep()

	// Edge from start to first step
	if len(steps) > 0 {
		edge := VisualizationEdge{
			ID:     "start-to-step-0",
			Type:   EdgeTypeSequential,
			Source: "start",
			Target: "step-0",
			Active: currentStep >= 0,
		}
		edges = append(edges, edge)
	}

	// Sequential edges between steps
	for i := 0; i < len(steps)-1; i++ {
		edge := VisualizationEdge{
			ID:     fmt.Sprintf("step-%d-to-step-%d", i, i+1),
			Type:   EdgeTypeSequential,
			Source: fmt.Sprintf("step-%d", i),
			Target: fmt.Sprintf("step-%d", i+1),
			Active: currentStep > i,
		}
		edges = append(edges, edge)
	}

	// Edge from last step to end
	if len(steps) > 0 {
		lastIndex := len(steps) - 1
		edge := VisualizationEdge{
			ID:     fmt.Sprintf("step-%d-to-end", lastIndex),
			Type:   EdgeTypeSequential,
			Source: fmt.Sprintf("step-%d", lastIndex),
			Target: "end",
			Active: v.isSagaCompleted(instance),
		}
		edges = append(edges, edge)
	}

	// Add compensation edges
	for i := 0; i < len(steps); i++ {
		if v.shouldShowCompensation(instance, i, stepStates) {
			// Edge from step to compensation
			edge := VisualizationEdge{
				ID:     fmt.Sprintf("step-%d-to-compensation-%d", i, i),
				Type:   EdgeTypeCompensation,
				Source: fmt.Sprintf("step-%d", i),
				Target: fmt.Sprintf("compensation-%d", i),
				Label:  "Compensate",
				Active: v.isCompensationActive(instance, stepStates) && i <= currentStep,
			}
			edges = append(edges, edge)
		}
	}

	return edges
}

// Helper methods for state determination

func (v *SagaVisualizer) getStartNodeState(instance saga.SagaInstance) NodeState {
	state := instance.GetState()
	if state == saga.StatePending {
		return NodeStatePending
	}
	return NodeStateActive
}

func (v *SagaVisualizer) getEndNodeState(instance saga.SagaInstance) NodeState {
	state := instance.GetState()
	switch state {
	case saga.StateCompleted:
		return NodeStateCompleted
	case saga.StateFailed:
		return NodeStateFailed
	case saga.StateCancelled:
		return NodeStateFailed
	default:
		return NodeStatePending
	}
}

func (v *SagaVisualizer) getStepNodeState(
	instance saga.SagaInstance,
	stepIndex int,
	currentStep int,
	stepStates []*saga.StepState,
) NodeState {
	// Use step state if available
	if stepIndex < len(stepStates) && stepStates[stepIndex] != nil {
		stepState := stepStates[stepIndex]
		switch stepState.State {
		case saga.StepStatePending:
			return NodeStatePending
		case saga.StepStateRunning:
			return NodeStateRunning
		case saga.StepStateCompleted:
			return NodeStateCompleted
		case saga.StepStateFailed:
			return NodeStateFailed
		case saga.StepStateSkipped:
			return NodeStateSkipped
		case saga.StepStateCompensating:
			return NodeStateCompensating
		case saga.StepStateCompensated:
			return NodeStateCompensated
		}
	}

	// Fallback: infer from current step
	if stepIndex < currentStep {
		return NodeStateCompleted
	} else if stepIndex == currentStep {
		return NodeStateRunning
	}
	return NodeStatePending
}

func (v *SagaVisualizer) getCompensationNodeState(
	stepIndex int,
	stepStates []*saga.StepState,
) NodeState {
	if stepIndex < len(stepStates) && stepStates[stepIndex] != nil {
		stepState := stepStates[stepIndex]
		if stepState.CompensationState != nil {
			switch stepState.CompensationState.State {
			case saga.CompensationStatePending:
				return NodeStatePending
			case saga.CompensationStateRunning:
				return NodeStateRunning
			case saga.CompensationStateCompleted:
				return NodeStateCompensated
			case saga.CompensationStateFailed:
				return NodeStateFailed
			case saga.CompensationStateSkipped:
				return NodeStateSkipped
			}
		}
	}
	return NodeStatePending
}

func (v *SagaVisualizer) shouldShowCompensation(
	instance saga.SagaInstance,
	stepIndex int,
	stepStates []*saga.StepState,
) bool {
	// Show compensation nodes if:
	// 1. Saga is in a compensating state, or
	// 2. Step has been compensated

	state := instance.GetState()
	if state == saga.StateCompensating || state == saga.StateCompensated {
		return true
	}

	// Check if this specific step has compensation state
	if stepIndex < len(stepStates) && stepStates[stepIndex] != nil {
		stepState := stepStates[stepIndex]
		if stepState.CompensationState != nil {
			return stepState.CompensationState.State != saga.CompensationStatePending
		}
	}

	return false
}

func (v *SagaVisualizer) isCompensationActive(instance saga.SagaInstance, stepStates []*saga.StepState) bool {
	state := instance.GetState()
	if state == saga.StateCompensating {
		return true
	}

	// Check if any step is actively compensating
	for _, stepState := range stepStates {
		if stepState != nil && stepState.CompensationState != nil {
			if stepState.CompensationState.State == saga.CompensationStateRunning {
				return true
			}
		}
	}

	return false
}

func (v *SagaVisualizer) isSagaCompleted(instance saga.SagaInstance) bool {
	state := instance.GetState()
	return state == saga.StateCompleted || state == saga.StateCompensated
}

func (v *SagaVisualizer) calculateProgress(instance saga.SagaInstance) float64 {
	// Get step information
	currentStep := instance.GetCurrentStep()

	// Try to get definition to get total steps
	definition, err := v.getSagaDefinition(instance)
	if err != nil || definition == nil {
		// Fallback to simple calculation
		state := instance.GetState()
		if state == saga.StateCompleted || state == saga.StateCompensated {
			return 100.0
		} else if state == saga.StateFailed || state == saga.StateCancelled {
			return float64(currentStep) * 100.0 / float64(currentStep+1)
		}
		return 0.0
	}

	totalSteps := len(definition.GetSteps())
	if totalSteps == 0 {
		return 0.0
	}

	// Calculate percentage
	progress := float64(currentStep) / float64(totalSteps) * 100.0

	// Ensure progress is in [0, 100] range
	if progress < 0 {
		progress = 0
	} else if progress > 100 {
		progress = 100
	}

	return progress
}

func (v *SagaVisualizer) countCompletedSteps(stepStates []*saga.StepState) int {
	count := 0
	for _, stepState := range stepStates {
		if stepState != nil && stepState.State == saga.StepStateCompleted {
			count++
		}
	}
	return count
}
