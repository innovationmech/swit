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

package server

import (
	"errors"
	"fmt"
)

// Server error codes as defined in the design document
const (
	ErrCodeConfigInvalid        = "CONFIG_INVALID"
	ErrCodeTransportFailed      = "TRANSPORT_FAILED"
	ErrCodeServiceRegFailed     = "SERVICE_REGISTRATION_FAILED"
	ErrCodeDiscoveryFailed      = "DISCOVERY_FAILED"
	ErrCodeShutdownTimeout      = "SHUTDOWN_TIMEOUT"
	ErrCodeDependencyFailed     = "DEPENDENCY_FAILED"
	ErrCodeLifecycleFailed      = "LIFECYCLE_FAILED"
	ErrCodeMiddlewareFailed     = "MIDDLEWARE_FAILED"
	ErrCodeServerAlreadyStarted = "SERVER_ALREADY_STARTED"
	ErrCodeServerNotStarted     = "SERVER_NOT_STARTED"
)

// ServerError represents a server-level error with categorization and context preservation
type ServerError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   string            `json:"details,omitempty"`
	Category  ErrorCategory     `json:"category"`
	Context   map[string]string `json:"context,omitempty"`
	Cause     error             `json:"-"`
	Operation string            `json:"operation,omitempty"`
}

// ErrorCategory represents different types of server failures
type ErrorCategory string

const (
	CategoryConfig     ErrorCategory = "configuration"
	CategoryTransport  ErrorCategory = "transport"
	CategoryDiscovery  ErrorCategory = "discovery"
	CategoryDependency ErrorCategory = "dependency"
	CategoryLifecycle  ErrorCategory = "lifecycle"
	CategoryMiddleware ErrorCategory = "middleware"
	CategoryService    ErrorCategory = "service"
)

// Error implements the error interface
func (e *ServerError) Error() string {
	if e.Details != "" && e.Operation != "" {
		return fmt.Sprintf("[%s] %s (%s) (%s)", e.Code, e.Message, e.Details, e.Operation)
	}
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s ()", e.Code, e.Message, e.Details)
	}
	if e.Operation != "" {
		return fmt.Sprintf("[%s] %s (%s)", e.Code, e.Message, e.Operation)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for error wrapping
func (e *ServerError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e *ServerError) WithContext(key, value string) *ServerError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// WithOperation sets the operation context for the error
func (e *ServerError) WithOperation(operation string) *ServerError {
	e.Operation = operation
	return e
}

// NewServerError creates a new server error
func NewServerError(code, message string, category ErrorCategory) *ServerError {
	return &ServerError{
		Code:     code,
		Message:  message,
		Category: category,
		Context:  make(map[string]string),
	}
}

// NewServerErrorWithDetails creates a new server error with details
func NewServerErrorWithDetails(code, message, details string, category ErrorCategory) *ServerError {
	return &ServerError{
		Code:     code,
		Message:  message,
		Details:  details,
		Category: category,
		Context:  make(map[string]string),
	}
}

// NewServerErrorWithCause creates a new server error with underlying cause
func NewServerErrorWithCause(code, message string, category ErrorCategory, cause error) *ServerError {
	return &ServerError{
		Code:     code,
		Message:  message,
		Category: category,
		Cause:    cause,
		Context:  make(map[string]string),
	}
}

// WrapError wraps an existing error with server error context
func WrapError(err error, code, message string, category ErrorCategory) *ServerError {
	return &ServerError{
		Code:     code,
		Message:  message,
		Category: category,
		Cause:    err,
		Context:  make(map[string]string),
	}
}

// Configuration error factory functions

// ErrConfigInvalid creates a configuration validation error
func ErrConfigInvalid(message string) *ServerError {
	return NewServerError(ErrCodeConfigInvalid, message, CategoryConfig)
}

// ErrConfigInvalidWithCause creates a configuration error with underlying cause
func ErrConfigInvalidWithCause(message string, cause error) *ServerError {
	return NewServerErrorWithCause(ErrCodeConfigInvalid, message, CategoryConfig, cause)
}

// Transport error factory functions

// ErrTransportFailed creates a transport initialization/operation error
func ErrTransportFailed(message string) *ServerError {
	return NewServerError(ErrCodeTransportFailed, message, CategoryTransport)
}

// ErrTransportFailedWithCause creates a transport error with underlying cause
func ErrTransportFailedWithCause(message string, cause error) *ServerError {
	return NewServerErrorWithCause(ErrCodeTransportFailed, message, CategoryTransport, cause)
}

// Service registration error factory functions

// ErrServiceRegFailed creates a service registration error
func ErrServiceRegFailed(message string) *ServerError {
	return NewServerError(ErrCodeServiceRegFailed, message, CategoryService)
}

// ErrServiceRegFailedWithCause creates a service registration error with underlying cause
func ErrServiceRegFailedWithCause(message string, cause error) *ServerError {
	return NewServerErrorWithCause(ErrCodeServiceRegFailed, message, CategoryService, cause)
}

// Discovery error factory functions

// ErrDiscoveryFailed creates a service discovery error
func ErrDiscoveryFailed(message string) *ServerError {
	return NewServerError(ErrCodeDiscoveryFailed, message, CategoryDiscovery)
}

// ErrDiscoveryFailedWithCause creates a service discovery error with underlying cause
func ErrDiscoveryFailedWithCause(message string, cause error) *ServerError {
	return NewServerErrorWithCause(ErrCodeDiscoveryFailed, message, CategoryDiscovery, cause)
}

// Dependency error factory functions

// ErrDependencyFailed creates a dependency initialization/operation error
func ErrDependencyFailed(message string) *ServerError {
	return NewServerError(ErrCodeDependencyFailed, message, CategoryDependency)
}

// ErrDependencyFailedWithCause creates a dependency error with underlying cause
func ErrDependencyFailedWithCause(message string, cause error) *ServerError {
	return NewServerErrorWithCause(ErrCodeDependencyFailed, message, CategoryDependency, cause)
}

// Lifecycle error factory functions

// ErrLifecycleFailed creates a server lifecycle error
func ErrLifecycleFailed(message string) *ServerError {
	return NewServerError(ErrCodeLifecycleFailed, message, CategoryLifecycle)
}

// ErrLifecycleFailedWithCause creates a lifecycle error with underlying cause
func ErrLifecycleFailedWithCause(message string, cause error) *ServerError {
	return NewServerErrorWithCause(ErrCodeLifecycleFailed, message, CategoryLifecycle, cause)
}

// ErrServerAlreadyStarted creates an error for attempting to start an already started server
func ErrServerAlreadyStarted() *ServerError {
	return NewServerError(ErrCodeServerAlreadyStarted, "server is already started", CategoryLifecycle)
}

// ErrServerNotStarted creates an error for operations requiring a started server
func ErrServerNotStarted() *ServerError {
	return NewServerError(ErrCodeServerNotStarted, "server is not started", CategoryLifecycle)
}

// Middleware error factory functions

// ErrMiddlewareFailed creates a middleware configuration error
func ErrMiddlewareFailed(message string) *ServerError {
	return NewServerError(ErrCodeMiddlewareFailed, message, CategoryMiddleware)
}

// ErrMiddlewareFailedWithCause creates a middleware error with underlying cause
func ErrMiddlewareFailedWithCause(message string, cause error) *ServerError {
	return NewServerErrorWithCause(ErrCodeMiddlewareFailed, message, CategoryMiddleware, cause)
}

// Shutdown error factory functions

// ErrShutdownTimeout creates a shutdown timeout error
func ErrShutdownTimeout(message string) *ServerError {
	return NewServerError(ErrCodeShutdownTimeout, message, CategoryLifecycle)
}

// ErrShutdownTimeoutWithCause creates a shutdown timeout error with underlying cause
func ErrShutdownTimeoutWithCause(message string, cause error) *ServerError {
	return NewServerErrorWithCause(ErrCodeShutdownTimeout, message, CategoryLifecycle, cause)
}

// Error checking utility functions

// IsServerError checks if an error is a ServerError
func IsServerError(err error) bool {
	var serverErr *ServerError
	return errors.As(err, &serverErr)
}

// GetServerError extracts ServerError from error
func GetServerError(err error) *ServerError {
	var serverErr *ServerError
	if errors.As(err, &serverErr) {
		return serverErr
	}
	return nil
}

// IsErrorCategory checks if an error belongs to a specific category
func IsErrorCategory(err error, category ErrorCategory) bool {
	if serverErr := GetServerError(err); serverErr != nil {
		return serverErr.Category == category
	}
	return false
}

// IsErrorCode checks if an error has a specific error code
func IsErrorCode(err error, code string) bool {
	if serverErr := GetServerError(err); serverErr != nil {
		return serverErr.Code == code
	}
	return false
}

// GetErrorContext retrieves context information from a server error
func GetErrorContext(err error) map[string]string {
	if serverErr := GetServerError(err); serverErr != nil {
		return serverErr.Context
	}
	return nil
}

// GetErrorCategory retrieves the category from a server error
func GetErrorCategory(err error) ErrorCategory {
	if serverErr := GetServerError(err); serverErr != nil {
		return serverErr.Category
	}
	return ""
}
