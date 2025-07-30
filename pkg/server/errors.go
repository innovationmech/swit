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
	"fmt"
)

// ErrorCode represents different types of server errors
type ErrorCode string

const (
	// Configuration errors
	ErrCodeConfigInvalid    ErrorCode = "CONFIG_INVALID"
	ErrCodeConfigMissing    ErrorCode = "CONFIG_MISSING"
	ErrCodeConfigValidation ErrorCode = "CONFIG_VALIDATION"

	// Transport errors
	ErrCodeTransportStart    ErrorCode = "TRANSPORT_START"
	ErrCodeTransportStop     ErrorCode = "TRANSPORT_STOP"
	ErrCodeTransportRegister ErrorCode = "TRANSPORT_REGISTER"
	ErrCodeTransportBind     ErrorCode = "TRANSPORT_BIND"

	// Discovery errors
	ErrCodeDiscoveryConnect    ErrorCode = "DISCOVERY_CONNECT"
	ErrCodeDiscoveryRegister   ErrorCode = "DISCOVERY_REGISTER"
	ErrCodeDiscoveryDeregister ErrorCode = "DISCOVERY_DEREGISTER"
	ErrCodeDiscoveryHealth     ErrorCode = "DISCOVERY_HEALTH"

	// Lifecycle errors
	ErrCodeStartup    ErrorCode = "STARTUP"
	ErrCodeShutdown   ErrorCode = "SHUTDOWN"
	ErrCodeTimeout    ErrorCode = "TIMEOUT"
	ErrCodeDependency ErrorCode = "DEPENDENCY"

	// Service errors
	ErrCodeServiceRegister ErrorCode = "SERVICE_REGISTER"
	ErrCodeServiceInit     ErrorCode = "SERVICE_INIT"
	ErrCodeServiceHealth   ErrorCode = "SERVICE_HEALTH"

	// Factory errors
	ErrCodeFactoryCreate   ErrorCode = "FACTORY_CREATE"
	ErrCodeFactoryRegister ErrorCode = "FACTORY_REGISTER"
	ErrCodeFactoryNotFound ErrorCode = "FACTORY_NOT_FOUND"
)

// ServerError represents a structured server error with context
type ServerError struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	Cause     error                  `json:"-"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Operation string                 `json:"operation,omitempty"`
}

// Error implements the error interface
func (e *ServerError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s - %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *ServerError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e *ServerError) WithContext(key string, value interface{}) *ServerError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithOperation sets the operation that caused the error
func (e *ServerError) WithOperation(operation string) *ServerError {
	e.Operation = operation
	return e
}

// NewServerError creates a new ServerError
func NewServerError(code ErrorCode, message string) *ServerError {
	return &ServerError{
		Code:    code,
		Message: message,
	}
}

// NewServerErrorWithCause creates a new ServerError with an underlying cause
func NewServerErrorWithCause(code ErrorCode, message string, cause error) *ServerError {
	return &ServerError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewServerErrorWithDetails creates a new ServerError with details
func NewServerErrorWithDetails(code ErrorCode, message, details string) *ServerError {
	return &ServerError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WrapError wraps an existing error as a ServerError
func WrapError(code ErrorCode, message string, err error) *ServerError {
	if err == nil {
		return nil
	}
	return &ServerError{
		Code:    code,
		Message: message,
		Cause:   err,
		Details: err.Error(),
	}
}

// IsServerError checks if an error is a ServerError
func IsServerError(err error) bool {
	_, ok := err.(*ServerError)
	return ok
}

// GetServerError extracts ServerError from an error
func GetServerError(err error) (*ServerError, bool) {
	if serverErr, ok := err.(*ServerError); ok {
		return serverErr, true
	}
	return nil, false
}

// HasErrorCode checks if an error has a specific error code
func HasErrorCode(err error, code ErrorCode) bool {
	if serverErr, ok := GetServerError(err); ok {
		return serverErr.Code == code
	}
	return false
}

// Predefined error constructors for common scenarios

// Configuration errors
func ErrConfigInvalid(details string) *ServerError {
	return NewServerErrorWithDetails(ErrCodeConfigInvalid, "Invalid configuration", details)
}

func ErrConfigMissing(field string) *ServerError {
	return NewServerErrorWithDetails(ErrCodeConfigMissing, "Missing configuration", fmt.Sprintf("Required field '%s' is missing", field))
}

func ErrConfigValidation(err error) *ServerError {
	return WrapError(ErrCodeConfigValidation, "Configuration validation failed", err)
}

// Transport errors
func ErrTransportStart(transport string, err error) *ServerError {
	return WrapError(ErrCodeTransportStart, fmt.Sprintf("Failed to start %s transport", transport), err)
}

func ErrTransportStop(transport string, err error) *ServerError {
	return WrapError(ErrCodeTransportStop, fmt.Sprintf("Failed to stop %s transport", transport), err)
}

func ErrTransportRegister(transport string, err error) *ServerError {
	return WrapError(ErrCodeTransportRegister, fmt.Sprintf("Failed to register %s transport", transport), err)
}

func ErrTransportBind(address string, err error) *ServerError {
	return WrapError(ErrCodeTransportBind, fmt.Sprintf("Failed to bind to address %s", address), err)
}

// Discovery errors
func ErrDiscoveryConnect(address string, err error) *ServerError {
	return WrapError(ErrCodeDiscoveryConnect, fmt.Sprintf("Failed to connect to discovery service at %s", address), err)
}

func ErrDiscoveryRegister(service string, err error) *ServerError {
	return WrapError(ErrCodeDiscoveryRegister, fmt.Sprintf("Failed to register service %s", service), err)
}

func ErrDiscoveryDeregister(service string, err error) *ServerError {
	return WrapError(ErrCodeDiscoveryDeregister, fmt.Sprintf("Failed to deregister service %s", service), err)
}

func ErrDiscoveryHealth(service string, err error) *ServerError {
	return WrapError(ErrCodeDiscoveryHealth, fmt.Sprintf("Health check failed for service %s", service), err)
}

// Lifecycle errors
func ErrStartup(component string, err error) *ServerError {
	return WrapError(ErrCodeStartup, fmt.Sprintf("Failed to start %s", component), err)
}

func ErrShutdown(component string, err error) *ServerError {
	return WrapError(ErrCodeShutdown, fmt.Sprintf("Failed to shutdown %s", component), err)
}

func ErrTimeout(operation string, timeout string) *ServerError {
	return NewServerErrorWithDetails(ErrCodeTimeout, fmt.Sprintf("Operation %s timed out", operation), fmt.Sprintf("Timeout: %s", timeout))
}

func ErrDependency(dependency string, err error) *ServerError {
	return WrapError(ErrCodeDependency, fmt.Sprintf("Dependency %s failed", dependency), err)
}

// Service errors
func ErrServiceRegister(service string, err error) *ServerError {
	return WrapError(ErrCodeServiceRegister, fmt.Sprintf("Failed to register service %s", service), err)
}

func ErrServiceInit(service string, err error) *ServerError {
	return WrapError(ErrCodeServiceInit, fmt.Sprintf("Failed to initialize service %s", service), err)
}

func ErrServiceHealth(service string, err error) *ServerError {
	return WrapError(ErrCodeServiceHealth, fmt.Sprintf("Service %s health check failed", service), err)
}

// Factory errors
func ErrFactoryCreate(factoryType string, err error) *ServerError {
	return WrapError(ErrCodeFactoryCreate, fmt.Sprintf("Failed to create %s factory", factoryType), err)
}

func ErrFactoryRegister(factoryName string, err error) *ServerError {
	return WrapError(ErrCodeFactoryRegister, fmt.Sprintf("Failed to register factory %s", factoryName), err)
}

func ErrFactoryNotFound(factoryName string) *ServerError {
	return NewServerErrorWithDetails(ErrCodeFactoryNotFound, "Factory not found", fmt.Sprintf("Factory '%s' is not registered", factoryName))
}
