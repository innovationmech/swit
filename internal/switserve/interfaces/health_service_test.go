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

package interfaces

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHealthService is a mock implementation of HealthService for testing
type MockHealthService struct {
	mock.Mock
}

func (m *MockHealthService) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HealthStatus), args.Error(1)
}

func TestHealthService_Interface(t *testing.T) {
	// Test that MockHealthService implements HealthService interface
	var _ HealthService = (*MockHealthService)(nil)
}

func TestHealthStatus_Structure(t *testing.T) {
	// Test HealthStatus struct creation and field access
	timestamp := time.Now().Unix()
	status := &HealthStatus{
		Status:    "healthy",
		Timestamp: timestamp,
		Details: map[string]string{
			"server":  "test-server",
			"version": "1.0.0",
		},
	}

	assert.Equal(t, "healthy", status.Status)
	assert.Equal(t, timestamp, status.Timestamp)
	assert.Equal(t, "test-server", status.Details["server"])
	assert.Equal(t, "1.0.0", status.Details["version"])
}

func TestMockHealthService_CheckHealth_Success(t *testing.T) {
	mockService := new(MockHealthService)
	ctx := context.Background()
	timestamp := time.Now().Unix()

	expectedStatus := &HealthStatus{
		Status:    "healthy",
		Timestamp: timestamp,
		Details: map[string]string{
			"server":  "swit-serve",
			"version": "1.0.0",
		},
	}

	mockService.On("CheckHealth", ctx).Return(expectedStatus, nil)

	status, err := mockService.CheckHealth(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, expectedStatus.Status, status.Status)
	assert.Equal(t, expectedStatus.Timestamp, status.Timestamp)
	assert.Equal(t, expectedStatus.Details, status.Details)
	mockService.AssertExpectations(t)
}

func TestMockHealthService_CheckHealth_Failure(t *testing.T) {
	mockService := new(MockHealthService)
	ctx := context.Background()
	expectedErr := errors.New("database connection failed")

	mockService.On("CheckHealth", ctx).Return(nil, expectedErr)

	status, err := mockService.CheckHealth(ctx)

	assert.Error(t, err)
	assert.Nil(t, status)
	assert.Equal(t, expectedErr, err)
	mockService.AssertExpectations(t)
}

func TestMockHealthService_CheckHealth_WithTimeout(t *testing.T) {
	mockService := new(MockHealthService)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	expectedStatus := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Details: map[string]string{
			"timeout": "5s",
		},
	}

	mockService.On("CheckHealth", ctx).Return(expectedStatus, nil)

	status, err := mockService.CheckHealth(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "healthy", status.Status)
	mockService.AssertExpectations(t)
}

func TestMockHealthService_CheckHealth_UnhealthyStatus(t *testing.T) {
	mockService := new(MockHealthService)
	ctx := context.Background()

	expectedStatus := &HealthStatus{
		Status:    "unhealthy",
		Timestamp: time.Now().Unix(),
		Details: map[string]string{
			"database": "connection failed",
			"redis":    "timeout",
		},
	}

	mockService.On("CheckHealth", ctx).Return(expectedStatus, nil)

	status, err := mockService.CheckHealth(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "unhealthy", status.Status)
	assert.Contains(t, status.Details, "database")
	assert.Contains(t, status.Details, "redis")
	mockService.AssertExpectations(t)
}
