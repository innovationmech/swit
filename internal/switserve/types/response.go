// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package types

import (
	"net/http"
)

// ResponseStatus represents API response status
type ResponseStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// NewResponseStatus creates a new ResponseStatus
func NewResponseStatus(code int, message string) *ResponseStatus {
	return &ResponseStatus{
		Code:    code,
		Message: message,
		Success: code >= 200 && code < 300,
	}
}

// NewSuccessStatus creates a success response status
func NewSuccessStatus(message string) *ResponseStatus {
	return NewResponseStatus(http.StatusOK, message)
}

// NewErrorStatus creates an error response status
func NewErrorStatus(code int, message string) *ResponseStatus {
	return NewResponseStatus(code, message)
}

// APIResponse represents a standard API response
type APIResponse struct {
	Status *ResponseStatus `json:"status"`
	Data   interface{}     `json:"data,omitempty"`
	Error  *ErrorDetail    `json:"error,omitempty"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewSuccessResponse creates a successful API response
func NewSuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Status: NewSuccessStatus("Success"),
		Data:   data,
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(code int, errorCode, message, details string) *APIResponse {
	return &APIResponse{
		Status: NewErrorStatus(code, message),
		Error: &ErrorDetail{
			Code:    errorCode,
			Message: message,
			Details: details,
		},
	}
}
