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
