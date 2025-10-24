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

// Package dsl provides a declarative domain-specific language (DSL) for defining Saga workflows.
//
// # Overview
//
// The dsl package enables users to define complex distributed transaction workflows using
// YAML configuration files. It provides type-safe data structures that map to YAML syntax,
// making it easy to create, validate, and process Saga definitions.
//
// # Key Features
//
//   - Declarative YAML-based syntax
//   - Type-safe Go structures with validation
//   - Support for multiple execution modes (orchestration, choreography, hybrid)
//   - Flexible step types (service, function, http, grpc, message, custom)
//   - Comprehensive retry and compensation policies
//   - Conditional execution and dependencies
//   - Hooks for success and failure events
//   - Variable templating support
//
// # Basic Usage
//
// Define a Saga in YAML:
//
//	saga:
//	  id: order-processing
//	  name: Order Processing Saga
//	  timeout: 5m
//	  mode: orchestration
//
//	steps:
//	  - id: validate-order
//	    name: Validate Order
//	    type: service
//	    action:
//	      service:
//	        name: order-service
//	        method: POST
//	        path: /validate
//	    timeout: 30s
//
//	  - id: process-payment
//	    name: Process Payment
//	    type: service
//	    action:
//	      service:
//	        name: payment-service
//	        method: POST
//	        path: /charge
//	    compensation:
//	      type: custom
//	      action:
//	        service:
//	          name: payment-service
//	          method: POST
//	          path: /refund
//	    dependencies:
//	      - validate-order
//
// Parse and use in Go:
//
//	import (
//	    "gopkg.in/yaml.v3"
//	    "github.com/innovationmech/swit/pkg/saga/dsl"
//	)
//
//	func loadSagaDefinition(yamlData []byte) (*dsl.SagaDefinition, error) {
//	    var def dsl.SagaDefinition
//	    if err := yaml.Unmarshal(yamlData, &def); err != nil {
//	        return nil, err
//	    }
//	    return &def, nil
//	}
//
// # Data Structures
//
// The package provides the following main structures:
//
//   - SagaDefinition: Top-level Saga definition
//   - SagaConfig: Saga configuration and metadata
//   - StepConfig: Individual step configuration
//   - ActionConfig: Step action definition (service, function, message, etc.)
//   - RetryPolicy: Retry strategy configuration
//   - CompensationAction: Compensation action definition
//   - Condition: Conditional execution logic
//
// # Validation
//
// All structures use the validate package for input validation.
// Required fields are enforced, and enum values are validated automatically
// during YAML unmarshaling.
//
// # Duration Format
//
// Durations in the DSL use Go's time.Duration format:
//
//   - ns: nanoseconds
//   - us/µs: microseconds
//   - ms: milliseconds
//   - s: seconds
//   - m: minutes
//   - h: hours
//
// Examples: "30s", "5m", "1h30m", "500ms"
//
// # Step Types
//
// The DSL supports various step types:
//
//   - service: Generic service invocation
//   - function: Local function execution
//   - http: HTTP request
//   - grpc: gRPC call
//   - message: Message broker publication
//   - custom: Custom implementations
//
// # Execution Modes
//
// Three execution modes are supported:
//
//   - orchestration: Centralized coordination (default)
//   - choreography: Event-driven coordination
//   - hybrid: Combination of both modes
//
// # Retry Policies
//
// Multiple retry strategies are available:
//
//   - fixed_delay: Fixed delay between retries
//   - exponential_backoff: Exponential backoff with jitter
//   - linear_backoff: Linear increase in delay
//   - no_retry: Disable retries
//
// # Compensation Strategies
//
// Compensation can be configured with different strategies:
//
//   - sequential: Compensate in reverse order (default)
//   - parallel: Compensate all steps in parallel
//   - best_effort: Continue compensation even if some fail
//
// # Variable Templating
//
// The DSL supports Go template syntax for dynamic values:
//
//   - {{.input.field}}: Access input data
//   - {{.output.step-id.field}}: Access step output
//   - {{.context.field}}: Access execution context
//   - {{.timestamp}}: Current timestamp
//   - {{.saga_id}}: Saga instance ID
//
// # Further Documentation
//
// For complete syntax specification and examples, see the spec.md file
// in this package directory.
package dsl
