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

package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// TestPlugin is a mock plugin implementation for testing.
type TestPlugin struct {
	mu           sync.RWMutex
	name         string
	version      string
	description  string
	initialized  bool
	executeCount int
	lastArgs     []string
	executeError error
	cleanupError error
}

// NewTestPlugin creates a new test plugin.
func NewTestPlugin(name, version string) *TestPlugin {
	return &TestPlugin{
		name:        name,
		version:     version,
		description: fmt.Sprintf("Test plugin %s v%s", name, version),
		initialized: false,
	}
}

// NewTestPluginWithError creates a new test plugin that will return errors.
func NewTestPluginWithError(name, version string, executeErr, cleanupErr error) *TestPlugin {
	plugin := NewTestPlugin(name, version)
	plugin.executeError = executeErr
	plugin.cleanupError = cleanupErr
	return plugin
}

// Name returns the plugin name.
func (p *TestPlugin) Name() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.name
}

// Version returns the plugin version.
func (p *TestPlugin) Version() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.version
}

// Description returns the plugin description.
func (p *TestPlugin) Description() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.description
}

// Initialize initializes the plugin.
func (p *TestPlugin) Initialize(config interfaces.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.name == "" {
		return fmt.Errorf("invalid plugin configuration: name cannot be empty")
	}
	p.initialized = true
	return nil
}

// Execute executes the plugin.
func (p *TestPlugin) Execute(ctx context.Context, args []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("plugin %s not initialized", p.name)
	}

	if p.executeError != nil {
		return p.executeError
	}

	p.executeCount++
	p.lastArgs = make([]string, len(args))
	copy(p.lastArgs, args)

	// Release lock before the sleep to avoid blocking other operations
	p.mu.Unlock()
	defer p.mu.Lock()

	// Simulate some work
	select {
	case <-time.After(10 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Cleanup cleans up the plugin.
func (p *TestPlugin) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cleanupError != nil {
		return p.cleanupError
	}
	p.initialized = false
	return nil
}

// GetExecuteCount returns the number of times Execute was called.
func (p *TestPlugin) GetExecuteCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.executeCount
}

// GetLastArgs returns the last arguments passed to Execute.
func (p *TestPlugin) GetLastArgs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]string, len(p.lastArgs))
	copy(result, p.lastArgs)
	return result
}

// IsInitialized returns whether the plugin is initialized.
func (p *TestPlugin) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.initialized
}

// SetDescription sets the plugin description.
func (p *TestPlugin) SetDescription(desc string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.description = desc
}
