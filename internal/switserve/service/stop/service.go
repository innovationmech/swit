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

package stop

import (
	"context"
	"time"
)

// ShutdownStatus represents shutdown response
type ShutdownStatus struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// Service implements server shutdown business logic
type Service struct {
	shutdownFunc func()
}

// NewService creates a new stop service instance
func NewService(shutdownFunc func()) *Service {
	return &Service{
		shutdownFunc: shutdownFunc,
	}
}

// InitiateShutdown initiates graceful server shutdown
func (s *Service) InitiateShutdown(ctx context.Context) (*ShutdownStatus, error) {
	// Initiate graceful shutdown
	if s.shutdownFunc != nil {
		go s.shutdownFunc() // Non-blocking call
	}

	status := &ShutdownStatus{
		Status:    "shutdown_initiated",
		Message:   "Server shutdown initiated successfully",
		Timestamp: time.Now().Unix(),
	}

	return status, nil
}
