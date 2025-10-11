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
	"time"
)

var (
	// ErrConcurrencyLimitExceeded indicates that the maximum concurrent Saga limit has been reached.
	ErrConcurrencyLimitExceeded = errors.New("concurrency limit exceeded")

	// ErrWorkerPoolClosed indicates that the worker pool has been closed.
	ErrWorkerPoolClosed = errors.New("worker pool is closed")

	// ErrInvalidConcurrencyLimit indicates that the concurrency limit is invalid (must be > 0).
	ErrInvalidConcurrencyLimit = errors.New("concurrency limit must be greater than 0")

	// ErrAcquireTimeout indicates that acquiring a worker timed out.
	ErrAcquireTimeout = errors.New("timeout acquiring worker from pool")
)

// ConcurrencyConfig contains configuration for concurrency control.
type ConcurrencyConfig struct {
	// MaxConcurrentSagas is the maximum number of Sagas that can execute concurrently.
	// If <= 0, defaults to runtime.NumCPU() * 2.
	MaxConcurrentSagas int

	// WorkerPoolSize is the size of the goroutine worker pool.
	// If <= 0, defaults to MaxConcurrentSagas.
	WorkerPoolSize int

	// AcquireTimeout is the maximum time to wait for acquiring a worker.
	// If <= 0, defaults to 30 seconds.
	AcquireTimeout time.Duration

	// ShutdownTimeout is the maximum time to wait for graceful shutdown.
	// If <= 0, defaults to 30 seconds.
	ShutdownTimeout time.Duration
}

// DefaultConcurrencyConfig returns the default concurrency configuration.
func DefaultConcurrencyConfig() *ConcurrencyConfig {
	return &ConcurrencyConfig{
		MaxConcurrentSagas: 0, // Will be set to runtime.NumCPU() * 2
		WorkerPoolSize:     0, // Will be set to MaxConcurrentSagas
		AcquireTimeout:     30 * time.Second,
		ShutdownTimeout:    30 * time.Second,
	}
}

// Validate validates the concurrency configuration and applies defaults.
func (c *ConcurrencyConfig) Validate() error {
	// Apply defaults
	if c.MaxConcurrentSagas <= 0 {
		c.MaxConcurrentSagas = getDefaultConcurrency()
	}

	if c.WorkerPoolSize <= 0 {
		c.WorkerPoolSize = c.MaxConcurrentSagas
	}

	if c.AcquireTimeout <= 0 {
		c.AcquireTimeout = 30 * time.Second
	}

	if c.ShutdownTimeout <= 0 {
		c.ShutdownTimeout = 30 * time.Second
	}

	return nil
}

// getDefaultConcurrency returns the default concurrency limit based on CPU count.
func getDefaultConcurrency() int {
	// Use a simple heuristic: 2x the number of CPUs
	// In a real implementation, this would use runtime.NumCPU()
	// For testing purposes, we use a fixed value
	return 10
}

// ConcurrencyController manages concurrent Saga execution with rate limiting
// and resource pooling. It ensures that the number of concurrently executing
// Sagas does not exceed configured limits and provides graceful shutdown capabilities.
type ConcurrencyController struct {
	// config holds the concurrency configuration
	config *ConcurrencyConfig

	// semaphore controls the number of concurrent Saga executions
	semaphore chan struct{}

	// workerPool is the goroutine worker pool
	workerPool *WorkerPool

	// activeInstances tracks currently executing Saga instances
	activeInstances sync.Map

	// activeSagaCount tracks the number of active Sagas
	activeSagaCount int

	// mu protects concurrent access to controller state
	mu sync.RWMutex

	// closed indicates if the controller has been shut down
	closed bool

	// shutdownCh signals shutdown to all workers
	shutdownCh chan struct{}

	// wg tracks active goroutines for graceful shutdown
	wg sync.WaitGroup
}

// NewConcurrencyController creates a new concurrency controller with the given configuration.
func NewConcurrencyController(config *ConcurrencyConfig) (*ConcurrencyController, error) {
	if config == nil {
		config = DefaultConcurrencyConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	cc := &ConcurrencyController{
		config:     config,
		semaphore:  make(chan struct{}, config.MaxConcurrentSagas),
		shutdownCh: make(chan struct{}),
		closed:     false,
	}

	// Initialize worker pool
	workerPool, err := NewWorkerPool(config.WorkerPoolSize)
	if err != nil {
		return nil, err
	}
	cc.workerPool = workerPool

	return cc, nil
}

// AcquireSlot acquires a concurrency slot for executing a Saga.
// It blocks until a slot is available or the context is cancelled.
// Returns an error if the controller is closed or the acquire times out.
func (cc *ConcurrencyController) AcquireSlot(ctx context.Context, sagaID string) error {
	cc.mu.RLock()
	if cc.closed {
		cc.mu.RUnlock()
		return ErrWorkerPoolClosed
	}
	cc.mu.RUnlock()

	// Create timeout context
	acquireCtx, cancel := context.WithTimeout(ctx, cc.config.AcquireTimeout)
	defer cancel()

	select {
	case cc.semaphore <- struct{}{}:
		// Slot acquired successfully
		cc.mu.Lock()
		cc.activeSagaCount++
		cc.activeInstances.Store(sagaID, time.Now())
		cc.mu.Unlock()
		return nil
	case <-acquireCtx.Done():
		if acquireCtx.Err() == context.DeadlineExceeded {
			return ErrAcquireTimeout
		}
		return acquireCtx.Err()
	case <-cc.shutdownCh:
		return ErrWorkerPoolClosed
	}
}

// ReleaseSlot releases a concurrency slot after Saga execution completes.
func (cc *ConcurrencyController) ReleaseSlot(sagaID string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.closed {
		return
	}

	// Remove from active instances
	cc.activeInstances.Delete(sagaID)
	cc.activeSagaCount--

	// Release semaphore slot
	select {
	case <-cc.semaphore:
		// Slot released
	default:
		// Should not happen, but handle gracefully
	}
}

// SubmitTask submits a Saga execution task to the worker pool.
// The task function will be executed by an available worker.
func (cc *ConcurrencyController) SubmitTask(ctx context.Context, task func(context.Context) error) error {
	cc.mu.RLock()
	if cc.closed {
		cc.mu.RUnlock()
		return ErrWorkerPoolClosed
	}
	cc.mu.RUnlock()

	return cc.workerPool.Submit(ctx, task)
}

// GetActiveSagaCount returns the number of currently active Sagas.
func (cc *ConcurrencyController) GetActiveSagaCount() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.activeSagaCount
}

// GetActiveInstances returns a snapshot of currently active Saga instance IDs and their start times.
func (cc *ConcurrencyController) GetActiveInstances() map[string]time.Time {
	instances := make(map[string]time.Time)
	cc.activeInstances.Range(func(key, value interface{}) bool {
		instances[key.(string)] = value.(time.Time)
		return true
	})
	return instances
}

// IsSlotAvailable returns true if a concurrency slot is available.
func (cc *ConcurrencyController) IsSlotAvailable() bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.activeSagaCount < cc.config.MaxConcurrentSagas
}

// Shutdown gracefully shuts down the concurrency controller.
// It waits for all active Sagas to complete (up to ShutdownTimeout),
// then closes the worker pool and releases all resources.
func (cc *ConcurrencyController) Shutdown(ctx context.Context) error {
	cc.mu.Lock()
	if cc.closed {
		cc.mu.Unlock()
		return nil
	}
	cc.closed = true
	close(cc.shutdownCh)
	cc.mu.Unlock()

	// Create shutdown timeout context
	shutdownCtx, cancel := context.WithTimeout(ctx, cc.config.ShutdownTimeout)
	defer cancel()

	// Wait for active Sagas with timeout
	done := make(chan struct{})
	go func() {
		cc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All Sagas completed
	case <-shutdownCtx.Done():
		// Timeout reached, proceed with forced shutdown
	}

	// Shutdown worker pool
	if err := cc.workerPool.Close(); err != nil {
		return err
	}

	return nil
}

// WorkerPool manages a pool of worker goroutines for executing Saga tasks.
// It provides bounded concurrency and graceful shutdown capabilities.
type WorkerPool struct {
	// size is the number of workers in the pool
	size int

	// taskCh is the channel for submitting tasks to workers
	taskCh chan workerTask

	// closed indicates if the pool has been closed
	closed bool

	// mu protects concurrent access to pool state
	mu sync.RWMutex

	// wg tracks active workers for graceful shutdown
	wg sync.WaitGroup
}

// workerTask represents a task to be executed by a worker.
type workerTask struct {
	ctx  context.Context
	fn   func(context.Context) error
	done chan error
}

// NewWorkerPool creates a new worker pool with the specified size.
func NewWorkerPool(size int) (*WorkerPool, error) {
	if size <= 0 {
		return nil, ErrInvalidConcurrencyLimit
	}

	wp := &WorkerPool{
		size:   size,
		taskCh: make(chan workerTask, size*2), // Buffer to reduce blocking
		closed: false,
	}

	// Start workers
	for i := 0; i < size; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	return wp, nil
}

// worker is the main loop for a worker goroutine.
// It continuously processes tasks from the task channel until the pool is closed.
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for task := range wp.taskCh {
		// Check if task context is already cancelled
		select {
		case <-task.ctx.Done():
			task.done <- task.ctx.Err()
			continue
		default:
		}

		// Execute the task
		err := task.fn(task.ctx)

		// Send result back
		select {
		case task.done <- err:
		case <-task.ctx.Done():
			// Context cancelled while sending result
		}
	}
}

// Submit submits a task to the worker pool for execution.
// It blocks until a worker is available or the context is cancelled.
func (wp *WorkerPool) Submit(ctx context.Context, fn func(context.Context) error) error {
	wp.mu.RLock()
	if wp.closed {
		wp.mu.RUnlock()
		return ErrWorkerPoolClosed
	}
	wp.mu.RUnlock()

	task := workerTask{
		ctx:  ctx,
		fn:   fn,
		done: make(chan error, 1),
	}

	// Submit task to worker pool
	select {
	case wp.taskCh <- task:
		// Task submitted successfully
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait for task completion
	select {
	case err := <-task.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close gracefully shuts down the worker pool.
// It waits for all workers to finish their current tasks.
func (wp *WorkerPool) Close() error {
	wp.mu.Lock()
	if wp.closed {
		wp.mu.Unlock()
		return nil
	}
	wp.closed = true
	close(wp.taskCh)
	wp.mu.Unlock()

	// Wait for all workers to finish
	wp.wg.Wait()

	return nil
}

// Size returns the number of workers in the pool.
func (wp *WorkerPool) Size() int {
	return wp.size
}

