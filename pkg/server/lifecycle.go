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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// LifecycleManager manages server lifecycle hooks and phases
type LifecycleManager struct {
	hooks []LifecycleHook
	mu    sync.RWMutex
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		hooks: make([]LifecycleHook, 0),
	}
}

// AddHook adds a lifecycle hook
func (lm *LifecycleManager) AddHook(hook LifecycleHook) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.hooks = append(lm.hooks, hook)
}

// RemoveHook removes a lifecycle hook by comparing function pointers
func (lm *LifecycleManager) RemoveHook(hook LifecycleHook) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	for i, h := range lm.hooks {
		// Note: This is a simple comparison and may not work for all cases
		// In practice, you might want to use a unique identifier for hooks
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", hook) {
			lm.hooks = append(lm.hooks[:i], lm.hooks[i+1:]...)
			return true
		}
	}
	return false
}

// ExecuteStartingHooks executes all starting phase hooks
func (lm *LifecycleManager) ExecuteStartingHooks(ctx context.Context) error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if logger.Logger != nil {
		logger.Logger.Debug("Executing starting hooks", zap.Int("count", len(lm.hooks)))
	}

	for i, hook := range lm.hooks {
		if err := lm.executeHookWithTimeout(ctx, "starting", i, hook.OnStarting); err != nil {
			return fmt.Errorf("starting hook %d failed: %w", i, err)
		}
	}

	return nil
}

// ExecuteStartedHooks executes all started phase hooks
func (lm *LifecycleManager) ExecuteStartedHooks(ctx context.Context) error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if logger.Logger != nil {
		logger.Logger.Debug("Executing started hooks", zap.Int("count", len(lm.hooks)))
	}

	for i, hook := range lm.hooks {
		if err := lm.executeHookWithTimeout(ctx, "started", i, hook.OnStarted); err != nil {
			return fmt.Errorf("started hook %d failed: %w", i, err)
		}
	}

	return nil
}

// ExecuteStoppingHooks executes all stopping phase hooks
func (lm *LifecycleManager) ExecuteStoppingHooks(ctx context.Context) error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if logger.Logger != nil {
		logger.Logger.Debug("Executing stopping hooks", zap.Int("count", len(lm.hooks)))
	}

	// Execute stopping hooks in reverse order
	for i := len(lm.hooks) - 1; i >= 0; i-- {
		hook := lm.hooks[i]
		if err := lm.executeHookWithTimeout(ctx, "stopping", i, hook.OnStopping); err != nil {
			if logger.Logger != nil {
				logger.Logger.Warn("Stopping hook failed", zap.Int("hook_index", i), zap.Error(err))
			}
			// Continue with other hooks even if one fails
		}
	}

	return nil
}

// ExecuteStoppedHooks executes all stopped phase hooks
func (lm *LifecycleManager) ExecuteStoppedHooks(ctx context.Context) error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if logger.Logger != nil {
		logger.Logger.Debug("Executing stopped hooks", zap.Int("count", len(lm.hooks)))
	}

	// Execute stopped hooks in reverse order
	for i := len(lm.hooks) - 1; i >= 0; i-- {
		hook := lm.hooks[i]
		if err := lm.executeHookWithTimeout(ctx, "stopped", i, hook.OnStopped); err != nil {
			if logger.Logger != nil {
				logger.Logger.Warn("Stopped hook failed", zap.Int("hook_index", i), zap.Error(err))
			}
			// Continue with other hooks even if one fails
		}
	}

	return nil
}

// executeHookWithTimeout executes a hook function with timeout protection
func (lm *LifecycleManager) executeHookWithTimeout(ctx context.Context, phase string, index int, hookFunc func(context.Context) error) error {
	// Create a timeout context for the hook execution
	hookCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Channel to receive the result
	resultCh := make(chan error, 1)

	// Execute the hook in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultCh <- fmt.Errorf("hook panicked: %v", r)
			}
		}()
		resultCh <- hookFunc(hookCtx)
	}()

	// Wait for either completion or timeout
	select {
	case err := <-resultCh:
		if err != nil {
			if logger.Logger != nil {
				logger.Logger.Error("Lifecycle hook failed",
					zap.String("phase", phase),
					zap.Int("index", index),
					zap.Error(err),
				)
			}
		}
		return err
	case <-hookCtx.Done():
		err := fmt.Errorf("hook execution timeout after 30 seconds")
		if logger.Logger != nil {
			logger.Logger.Error("Lifecycle hook timeout",
				zap.String("phase", phase),
				zap.Int("index", index),
				zap.Error(err),
			)
		}
		return err
	}
}

// GetHookCount returns the number of registered hooks
func (lm *LifecycleManager) GetHookCount() int {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return len(lm.hooks)
}

// Clear removes all registered hooks
func (lm *LifecycleManager) Clear() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.hooks = lm.hooks[:0]
}

// BasicLifecycleHook provides a basic implementation of LifecycleHook
type BasicLifecycleHook struct {
	name       string
	onStarting func(context.Context) error
	onStarted  func(context.Context) error
	onStopping func(context.Context) error
	onStopped  func(context.Context) error
}

// NewBasicLifecycleHook creates a new basic lifecycle hook
func NewBasicLifecycleHook(name string) *BasicLifecycleHook {
	return &BasicLifecycleHook{
		name:       name,
		onStarting: func(context.Context) error { return nil },
		onStarted:  func(context.Context) error { return nil },
		onStopping: func(context.Context) error { return nil },
		onStopped:  func(context.Context) error { return nil },
	}
}

// SetOnStarting sets the starting phase handler
func (h *BasicLifecycleHook) SetOnStarting(fn func(context.Context) error) *BasicLifecycleHook {
	h.onStarting = fn
	return h
}

// SetOnStarted sets the started phase handler
func (h *BasicLifecycleHook) SetOnStarted(fn func(context.Context) error) *BasicLifecycleHook {
	h.onStarted = fn
	return h
}

// SetOnStopping sets the stopping phase handler
func (h *BasicLifecycleHook) SetOnStopping(fn func(context.Context) error) *BasicLifecycleHook {
	h.onStopping = fn
	return h
}

// SetOnStopped sets the stopped phase handler
func (h *BasicLifecycleHook) SetOnStopped(fn func(context.Context) error) *BasicLifecycleHook {
	h.onStopped = fn
	return h
}

// OnStarting implements LifecycleHook
func (h *BasicLifecycleHook) OnStarting(ctx context.Context) error {
	if logger.Logger != nil {
		logger.Logger.Debug("Executing starting hook", zap.String("name", h.name))
	}
	return h.onStarting(ctx)
}

// OnStarted implements LifecycleHook
func (h *BasicLifecycleHook) OnStarted(ctx context.Context) error {
	if logger.Logger != nil {
		logger.Logger.Debug("Executing started hook", zap.String("name", h.name))
	}
	return h.onStarted(ctx)
}

// OnStopping implements LifecycleHook
func (h *BasicLifecycleHook) OnStopping(ctx context.Context) error {
	if logger.Logger != nil {
		logger.Logger.Debug("Executing stopping hook", zap.String("name", h.name))
	}
	return h.onStopping(ctx)
}

// OnStopped implements LifecycleHook
func (h *BasicLifecycleHook) OnStopped(ctx context.Context) error {
	if logger.Logger != nil {
		logger.Logger.Debug("Executing stopped hook", zap.String("name", h.name))
	}
	return h.onStopped(ctx)
}

// GetName returns the hook name
func (h *BasicLifecycleHook) GetName() string {
	return h.name
}
