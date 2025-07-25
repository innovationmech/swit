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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	shutdownFunc := func() {}

	service := NewService(shutdownFunc)

	assert.NotNil(t, service)
	// Note: Cannot directly test private fields due to interface abstraction
}

func TestNewService_NilShutdownFunc(t *testing.T) {
	service := NewService(nil)

	assert.NotNil(t, service)
	// Note: Cannot directly test private fields due to interface abstraction
}

func TestService_InitiateShutdown(t *testing.T) {
	var shutdownCalled int32
	shutdownFunc := func() {
		atomic.StoreInt32(&shutdownCalled, 1)
	}

	service := NewService(shutdownFunc)
	ctx := context.Background()

	status, err := service.InitiateShutdown(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "shutdown_initiated", status.Status.String())
	assert.Equal(t, "Server shutdown initiated successfully", status.Message)
	assert.NotZero(t, status.InitiatedAt)

	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&shutdownCalled) == 1
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func TestService_InitiateShutdown_NilFunction(t *testing.T) {
	service := NewService(nil)
	ctx := context.Background()

	status, err := service.InitiateShutdown(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "shutdown_initiated", status.Status.String())
	assert.Equal(t, "Server shutdown initiated successfully", status.Message)
	assert.NotZero(t, status.InitiatedAt)
}

func TestService_InitiateShutdown_StatusStructure(t *testing.T) {
	service := NewService(func() {})
	ctx := context.Background()

	status, err := service.InitiateShutdown(ctx)

	require.NoError(t, err)
	assert.Equal(t, "shutdown_initiated", status.Status.String())
	assert.Equal(t, "Server shutdown initiated successfully", status.Message)
	assert.IsType(t, int64(0), status.InitiatedAt)
}

func TestService_InitiateShutdown_Timestamp(t *testing.T) {
	service := NewService(func() {})
	ctx := context.Background()

	beforeTime := time.Now().Unix()
	status, err := service.InitiateShutdown(ctx)
	afterTime := time.Now().Unix()

	require.NoError(t, err)
	assert.GreaterOrEqual(t, status.InitiatedAt, beforeTime)
	assert.LessOrEqual(t, status.InitiatedAt, afterTime)
}

func TestService_InitiateShutdown_WithContext(t *testing.T) {
	service := NewService(func() {})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := service.InitiateShutdown(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "shutdown_initiated", status.Status.String())
}

func TestService_InitiateShutdown_WithCancelledContext(t *testing.T) {
	service := NewService(func() {})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	status, err := service.InitiateShutdown(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "shutdown_initiated", status.Status.String())
}

func TestService_InitiateShutdown_Concurrent(t *testing.T) {
	var shutdownCount int32
	shutdownFunc := func() {
		atomic.AddInt32(&shutdownCount, 1)
	}

	service := NewService(shutdownFunc)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, err := service.InitiateShutdown(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, status)
			assert.Equal(t, "shutdown_initiated", status.Status.String())
		}()
	}
	wg.Wait()

	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&shutdownCount) == 5
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func TestShutdownStatus_JSONStructure(t *testing.T) {
	service := NewService(func() {})
	ctx := context.Background()

	status, err := service.InitiateShutdown(ctx)

	require.NoError(t, err)

	assert.IsType(t, types.Status(""), status.Status)
	assert.IsType(t, "", status.Message)
	assert.IsType(t, int64(0), status.InitiatedAt)
}
