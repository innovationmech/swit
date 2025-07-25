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

package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/types"
)

func TestNewHealthSrv(t *testing.T) {
	service := NewHealthSrv()

	assert.NotNil(t, service)
	// Verify that the service implements the HealthService interface
	var _ interfaces.HealthService = service
	assert.IsType(t, &Service{}, service)
}

func TestNewHealthSrvWithConfig(t *testing.T) {
	config := &HealthServiceConfig{
		ServiceName: "test-service",
		Version:     "2.0.0",
		StartTime:   time.Now(),
	}
	service := NewHealthSrvWithConfig(config)

	assert.NotNil(t, service)
}

func TestHealthService_CheckHealth(t *testing.T) {
	service := NewHealthSrv()
	ctx := context.Background()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "healthy", status.Status)
	assert.NotZero(t, status.Timestamp)
	assert.NotNil(t, status.Details)
}

func TestHealthService_CheckHealth_StatusStructure(t *testing.T) {
	service := NewHealthSrv()
	ctx := context.Background()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)
	assert.Equal(t, "healthy", status.Status)
	assert.Contains(t, status.Details, "server")
	assert.Contains(t, status.Details, "version")
	assert.Contains(t, status.Details, "uptime")
	assert.Equal(t, "swit-serve", status.Details["server"])
	assert.Equal(t, "1.0.0", status.Details["version"])
}

func TestHealthService_CheckHealth_Timestamp(t *testing.T) {
	service := NewHealthSrv()
	ctx := context.Background()

	beforeTime := time.Now().Unix()
	status, err := service.CheckHealth(ctx)
	afterTime := time.Now().Unix()

	require.NoError(t, err)
	assert.GreaterOrEqual(t, status.Timestamp, beforeTime)
	assert.LessOrEqual(t, status.Timestamp, afterTime)
}

func TestHealthService_CheckHealth_WithContext(t *testing.T) {
	service := NewHealthSrv()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "healthy", status.Status)
}

func TestHealthService_CheckHealth_WithCancelledContext(t *testing.T) {
	service := NewHealthSrv()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "healthy", status.Status)
}

func TestHealthService_CheckHealth_NilContext(t *testing.T) {
	service := NewHealthSrv()

	status, err := service.CheckHealth(nil)

	assert.Nil(t, status)
	assert.Error(t, err)
	assert.True(t, types.IsServiceError(err))
	serviceErr := err.(*types.ServiceError)
	assert.Equal(t, types.ErrCodeValidation, serviceErr.Code)
}

func TestHealthService_CheckHealth_WithOptions(t *testing.T) {
	service := NewHealthSrv(
		WithServiceName("test-service"),
		WithVersion("2.0.0"),
	)
	ctx := context.Background()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)
	assert.Equal(t, "test-service", status.Details["server"])
	assert.Equal(t, "2.0.0", status.Details["version"])
}

func TestHealthStatus_JSONStructure(t *testing.T) {
	service := NewHealthSrv()
	ctx := context.Background()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)

	assert.IsType(t, "", status.Status)
	assert.IsType(t, int64(0), status.Timestamp)
	assert.IsType(t, map[string]string{}, status.Details)
}
