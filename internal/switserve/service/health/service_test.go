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
)

func TestNewService(t *testing.T) {
	service := NewService()

	assert.NotNil(t, service)
}

func TestService_CheckHealth(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "healthy", status.Status)
	assert.NotZero(t, status.Timestamp)
	assert.NotNil(t, status.Details)
}

func TestService_CheckHealth_StatusStructure(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)
	assert.Equal(t, "healthy", status.Status)
	assert.Contains(t, status.Details, "server")
	assert.Contains(t, status.Details, "version")
	assert.Equal(t, "swit-serve", status.Details["server"])
	assert.Equal(t, "1.0.0", status.Details["version"])
}

func TestService_CheckHealth_Timestamp(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	beforeTime := time.Now().Unix()
	status, err := service.CheckHealth(ctx)
	afterTime := time.Now().Unix()

	require.NoError(t, err)
	assert.GreaterOrEqual(t, status.Timestamp, beforeTime)
	assert.LessOrEqual(t, status.Timestamp, afterTime)
}

func TestService_CheckHealth_WithContext(t *testing.T) {
	service := NewService()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "healthy", status.Status)
}

func TestService_CheckHealth_WithCancelledContext(t *testing.T) {
	service := NewService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "healthy", status.Status)
}

func TestHealthStatus_JSONStructure(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	status, err := service.CheckHealth(ctx)

	require.NoError(t, err)

	assert.IsType(t, "", status.Status)
	assert.IsType(t, int64(0), status.Timestamp)
	assert.IsType(t, map[string]string{}, status.Details)
}
