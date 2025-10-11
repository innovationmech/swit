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

package coordinator

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConcurrencyConfig_Validate tests the validation of concurrency configuration.
func TestConcurrencyConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ConcurrencyConfig
		wantErr bool
	}{
		{
			name: "valid config with all fields",
			config: &ConcurrencyConfig{
				MaxConcurrentSagas: 10,
				WorkerPoolSize:     10,
				AcquireTimeout:     5 * time.Second,
				ShutdownTimeout:    10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid config with defaults",
			config: &ConcurrencyConfig{
				MaxConcurrentSagas: 0, // Will use default
				WorkerPoolSize:     0, // Will use default
				AcquireTimeout:     0, // Will use default
				ShutdownTimeout:    0, // Will use default
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: false, // Should work with nil and apply defaults
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config == nil {
				tt.config = DefaultConcurrencyConfig()
			}
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				// Verify defaults are applied
				if tt.config.MaxConcurrentSagas <= 0 {
					t.Errorf("MaxConcurrentSagas should be > 0 after validation, got %d", tt.config.MaxConcurrentSagas)
				}
				if tt.config.WorkerPoolSize <= 0 {
					t.Errorf("WorkerPoolSize should be > 0 after validation, got %d", tt.config.WorkerPoolSize)
				}
				if tt.config.AcquireTimeout <= 0 {
					t.Errorf("AcquireTimeout should be > 0 after validation, got %v", tt.config.AcquireTimeout)
				}
				if tt.config.ShutdownTimeout <= 0 {
					t.Errorf("ShutdownTimeout should be > 0 after validation, got %v", tt.config.ShutdownTimeout)
				}
			}
		})
	}
}

// TestNewConcurrencyController tests the creation of concurrency controller.
func TestNewConcurrencyController(t *testing.T) {
	tests := []struct {
		name    string
		config  *ConcurrencyConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ConcurrencyConfig{
				MaxConcurrentSagas: 5,
				WorkerPoolSize:     5,
				AcquireTimeout:     5 * time.Second,
				ShutdownTimeout:    10 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc, err := NewConcurrencyController(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConcurrencyController() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if cc == nil {
					t.Error("NewConcurrencyController() returned nil controller")
					return
				}
				if cc.workerPool == nil {
					t.Error("Worker pool should be initialized")
				}
				if cc.semaphore == nil {
					t.Error("Semaphore should be initialized")
				}
				// Clean up
				_ = cc.Shutdown(context.Background())
			}
		})
	}
}

// TestConcurrencyController_AcquireReleaseSlot tests acquiring and releasing slots.
func TestConcurrencyController_AcquireReleaseSlot(t *testing.T) {
	config := &ConcurrencyConfig{
		MaxConcurrentSagas: 2,
		WorkerPoolSize:     2,
		AcquireTimeout:     1 * time.Second,
		ShutdownTimeout:    5 * time.Second,
	}

	cc, err := NewConcurrencyController(config)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer cc.Shutdown(context.Background())

	ctx := context.Background()

	// Acquire first slot
	if err := cc.AcquireSlot(ctx, "saga-1"); err != nil {
		t.Errorf("Failed to acquire first slot: %v", err)
	}

	// Check active count
	if count := cc.GetActiveSagaCount(); count != 1 {
		t.Errorf("Expected active count 1, got %d", count)
	}

	// Acquire second slot
	if err := cc.AcquireSlot(ctx, "saga-2"); err != nil {
		t.Errorf("Failed to acquire second slot: %v", err)
	}

	// Check active count
	if count := cc.GetActiveSagaCount(); count != 2 {
		t.Errorf("Expected active count 2, got %d", count)
	}

	// Try to acquire third slot (should timeout)
	shortCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	err = cc.AcquireSlot(shortCtx, "saga-3")
	if err == nil {
		t.Error("Expected timeout error when acquiring beyond limit")
	}

	// Release first slot
	cc.ReleaseSlot("saga-1")

	// Check active count
	if count := cc.GetActiveSagaCount(); count != 1 {
		t.Errorf("Expected active count 1 after release, got %d", count)
	}

	// Now should be able to acquire
	if err := cc.AcquireSlot(ctx, "saga-3"); err != nil {
		t.Errorf("Failed to acquire slot after release: %v", err)
	}

	// Clean up
	cc.ReleaseSlot("saga-2")
	cc.ReleaseSlot("saga-3")
}

// TestConcurrencyController_ConcurrentAcquire tests concurrent slot acquisition.
func TestConcurrencyController_ConcurrentAcquire(t *testing.T) {
	maxConcurrent := 10
	numGoroutines := 50

	config := &ConcurrencyConfig{
		MaxConcurrentSagas: maxConcurrent,
		WorkerPoolSize:     maxConcurrent,
		AcquireTimeout:     5 * time.Second,
		ShutdownTimeout:    10 * time.Second,
	}

	cc, err := NewConcurrencyController(config)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer cc.Shutdown(context.Background())

	ctx := context.Background()
	var wg sync.WaitGroup
	var acquired atomic.Int32
	var maxSeen atomic.Int32

	// Launch many goroutines trying to acquire slots
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			sagaID := string(rune('a' + id))
			if err := cc.AcquireSlot(ctx, sagaID); err != nil {
				// Some may fail due to timeout, that's ok
				return
			}

			// Track concurrent acquisitions
			current := acquired.Add(1)
			for {
				max := maxSeen.Load()
				if current <= max || maxSeen.CompareAndSwap(max, current) {
					break
				}
			}

			// Simulate work
			time.Sleep(50 * time.Millisecond)

			acquired.Add(-1)
			cc.ReleaseSlot(sagaID)
		}(i)
	}

	wg.Wait()

	// Check that we never exceeded the limit
	if max := maxSeen.Load(); max > int32(maxConcurrent) {
		t.Errorf("Concurrency limit violated: max concurrent = %d, limit = %d", max, maxConcurrent)
	}

	// Check that all slots are released
	if count := cc.GetActiveSagaCount(); count != 0 {
		t.Errorf("Expected 0 active sagas after all released, got %d", count)
	}
}

// TestConcurrencyController_IsSlotAvailable tests slot availability check.
func TestConcurrencyController_IsSlotAvailable(t *testing.T) {
	config := &ConcurrencyConfig{
		MaxConcurrentSagas: 2,
		WorkerPoolSize:     2,
		AcquireTimeout:     1 * time.Second,
		ShutdownTimeout:    5 * time.Second,
	}

	cc, err := NewConcurrencyController(config)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer cc.Shutdown(context.Background())

	ctx := context.Background()

	// Initially should be available
	if !cc.IsSlotAvailable() {
		t.Error("Slot should be available initially")
	}

	// Acquire slots
	_ = cc.AcquireSlot(ctx, "saga-1")
	if !cc.IsSlotAvailable() {
		t.Error("Slot should still be available after 1 acquisition")
	}

	_ = cc.AcquireSlot(ctx, "saga-2")
	if cc.IsSlotAvailable() {
		t.Error("Slot should not be available when limit reached")
	}

	// Release one
	cc.ReleaseSlot("saga-1")
	if !cc.IsSlotAvailable() {
		t.Error("Slot should be available after release")
	}

	// Clean up
	cc.ReleaseSlot("saga-2")
}

// TestConcurrencyController_GetActiveInstances tests retrieving active instances.
func TestConcurrencyController_GetActiveInstances(t *testing.T) {
	config := &ConcurrencyConfig{
		MaxConcurrentSagas: 5,
		WorkerPoolSize:     5,
		AcquireTimeout:     1 * time.Second,
		ShutdownTimeout:    5 * time.Second,
	}

	cc, err := NewConcurrencyController(config)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer cc.Shutdown(context.Background())

	ctx := context.Background()

	// Acquire some slots
	sagaIDs := []string{"saga-1", "saga-2", "saga-3"}
	for _, id := range sagaIDs {
		if err := cc.AcquireSlot(ctx, id); err != nil {
			t.Fatalf("Failed to acquire slot for %s: %v", id, err)
		}
	}

	// Get active instances
	instances := cc.GetActiveInstances()

	// Verify all IDs are present
	if len(instances) != len(sagaIDs) {
		t.Errorf("Expected %d active instances, got %d", len(sagaIDs), len(instances))
	}

	for _, id := range sagaIDs {
		if _, ok := instances[id]; !ok {
			t.Errorf("Expected to find saga %s in active instances", id)
		}
	}

	// Clean up
	for _, id := range sagaIDs {
		cc.ReleaseSlot(id)
	}
}

// TestConcurrencyController_Shutdown tests graceful shutdown.
func TestConcurrencyController_Shutdown(t *testing.T) {
	config := &ConcurrencyConfig{
		MaxConcurrentSagas: 5,
		WorkerPoolSize:     5,
		AcquireTimeout:     1 * time.Second,
		ShutdownTimeout:    2 * time.Second,
	}

	cc, err := NewConcurrencyController(config)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}

	ctx := context.Background()

	// Start some long-running tasks
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sagaID := string(rune('a' + id))
			if err := cc.AcquireSlot(ctx, sagaID); err != nil {
				return
			}
			defer cc.ReleaseSlot(sagaID)

			// Simulate work
			time.Sleep(500 * time.Millisecond)
		}(i)
	}

	// Wait a bit for tasks to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown
	shutdownErr := cc.Shutdown(ctx)
	if shutdownErr != nil {
		t.Errorf("Shutdown failed: %v", shutdownErr)
	}

	// Wait for goroutines
	wg.Wait()

	// After shutdown, acquiring should fail
	err = cc.AcquireSlot(ctx, "saga-after-shutdown")
	if err != ErrWorkerPoolClosed {
		t.Errorf("Expected ErrWorkerPoolClosed after shutdown, got %v", err)
	}
}

// TestWorkerPool_Submit tests submitting tasks to worker pool.
func TestWorkerPool_Submit(t *testing.T) {
	poolSize := 3
	wp, err := NewWorkerPool(poolSize)
	if err != nil {
		t.Fatalf("Failed to create worker pool: %v", err)
	}
	defer wp.Close()

	ctx := context.Background()
	var counter atomic.Int32

	// Submit multiple tasks
	numTasks := 10
	var wg sync.WaitGroup
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := wp.Submit(ctx, func(ctx context.Context) error {
				counter.Add(1)
				time.Sleep(10 * time.Millisecond)
				return nil
			})
			if err != nil {
				t.Errorf("Submit failed: %v", err)
			}
		}()
	}

	wg.Wait()

	// Verify all tasks executed
	if count := counter.Load(); count != int32(numTasks) {
		t.Errorf("Expected %d tasks to execute, got %d", numTasks, count)
	}
}

// TestWorkerPool_SubmitWithError tests error handling in worker pool.
func TestWorkerPool_SubmitWithError(t *testing.T) {
	wp, err := NewWorkerPool(2)
	if err != nil {
		t.Fatalf("Failed to create worker pool: %v", err)
	}
	defer wp.Close()

	ctx := context.Background()
	expectedErr := errors.New("task error")

	// Submit task that returns error
	err = wp.Submit(ctx, func(ctx context.Context) error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

// TestWorkerPool_SubmitWithContextCancellation tests context cancellation.
func TestWorkerPool_SubmitWithContextCancellation(t *testing.T) {
	wp, err := NewWorkerPool(2)
	if err != nil {
		t.Fatalf("Failed to create worker pool: %v", err)
	}
	defer wp.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Submit task with cancelled context
	err = wp.Submit(ctx, func(ctx context.Context) error {
		time.Sleep(1 * time.Second)
		return nil
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// TestWorkerPool_Close tests worker pool closure.
func TestWorkerPool_Close(t *testing.T) {
	wp, err := NewWorkerPool(2)
	if err != nil {
		t.Fatalf("Failed to create worker pool: %v", err)
	}

	// Close the pool
	if err := wp.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Submitting after close should fail
	ctx := context.Background()
	err = wp.Submit(ctx, func(ctx context.Context) error {
		return nil
	})

	if err != ErrWorkerPoolClosed {
		t.Errorf("Expected ErrWorkerPoolClosed after close, got %v", err)
	}

	// Closing again should be safe
	if err := wp.Close(); err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}

// TestWorkerPool_InvalidSize tests creating pool with invalid size.
func TestWorkerPool_InvalidSize(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"zero size", 0},
		{"negative size", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp, err := NewWorkerPool(tt.size)
			if err != ErrInvalidConcurrencyLimit {
				t.Errorf("Expected ErrInvalidConcurrencyLimit, got %v", err)
			}
			if wp != nil {
				t.Error("Expected nil worker pool for invalid size")
			}
		})
	}
}

// TestConcurrencyController_RaceConditions tests for race conditions.
// This test should be run with -race flag: go test -race
func TestConcurrencyController_RaceConditions(t *testing.T) {
	config := &ConcurrencyConfig{
		MaxConcurrentSagas: 20,
		WorkerPoolSize:     20,
		AcquireTimeout:     5 * time.Second,
		ShutdownTimeout:    10 * time.Second,
	}

	cc, err := NewConcurrencyController(config)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer cc.Shutdown(context.Background())

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent acquire/release operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sagaID := string(rune('a' + id))

			// Acquire
			if err := cc.AcquireSlot(ctx, sagaID); err != nil {
				return
			}

			// Concurrent reads
			_ = cc.GetActiveSagaCount()
			_ = cc.IsSlotAvailable()
			_ = cc.GetActiveInstances()

			// Release
			cc.ReleaseSlot(sagaID)
		}(i)
	}

	wg.Wait()
}

// TestConcurrencyController_SubmitTask tests submitting tasks to the controller.
func TestConcurrencyController_SubmitTask(t *testing.T) {
	config := &ConcurrencyConfig{
		MaxConcurrentSagas: 5,
		WorkerPoolSize:     5,
		AcquireTimeout:     1 * time.Second,
		ShutdownTimeout:    5 * time.Second,
	}

	cc, err := NewConcurrencyController(config)
	if err != nil {
		t.Fatalf("Failed to create controller: %v", err)
	}
	defer cc.Shutdown(context.Background())

	ctx := context.Background()
	var executed atomic.Bool

	// Submit a task
	err = cc.SubmitTask(ctx, func(ctx context.Context) error {
		executed.Store(true)
		return nil
	})

	if err != nil {
		t.Errorf("SubmitTask failed: %v", err)
	}

	if !executed.Load() {
		t.Error("Task was not executed")
	}
}
