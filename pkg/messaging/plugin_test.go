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

package messaging

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockPlugin is a test plugin implementation.
type mockPlugin struct {
	*BasePlugin
	initCalled     bool
	startCalled    bool
	stopCalled     bool
	shutdownCalled bool
	shouldFailInit bool
}

func newMockPlugin(name string) *mockPlugin {
	return &mockPlugin{
		BasePlugin: NewBasePlugin(PluginMetadata{
			Name:        name,
			Version:     "1.0.0",
			Description: "Mock plugin for testing",
			Author:      "Test",
		}),
	}
}

func (mp *mockPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	mp.initCalled = true
	if mp.shouldFailInit {
		mp.SetState(PluginStateFailed)
		return fmt.Errorf("init failed")
	}
	return mp.BasePlugin.Initialize(ctx, config)
}

func (mp *mockPlugin) Start(ctx context.Context) error {
	mp.startCalled = true
	return mp.BasePlugin.Start(ctx)
}

func (mp *mockPlugin) Stop(ctx context.Context) error {
	mp.stopCalled = true
	return mp.BasePlugin.Stop(ctx)
}

func (mp *mockPlugin) Shutdown(ctx context.Context) error {
	mp.shutdownCalled = true
	return mp.BasePlugin.Shutdown(ctx)
}

func (mp *mockPlugin) CreateMiddleware() (Middleware, error) {
	if mp.State() != PluginStateStarted {
		return nil, fmt.Errorf("plugin not started")
	}
	return &mockMiddleware{name: mp.Metadata().Name}, nil
}

// mockMiddleware is a test middleware implementation.
type mockMiddleware struct {
	name   string
	called bool
}

func (mm *mockMiddleware) Name() string {
	return mm.name
}

func (mm *mockMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		mm.called = true
		return next.Handle(ctx, message)
	})
}

func TestPluginState_String(t *testing.T) {
	tests := []struct {
		state    PluginState
		expected string
	}{
		{PluginStateUninitialized, "uninitialized"},
		{PluginStateInitialized, "initialized"},
		{PluginStateStarted, "started"},
		{PluginStateStopped, "stopped"},
		{PluginStateFailed, "failed"},
		{PluginState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestBasePlugin_Lifecycle(t *testing.T) {
	ctx := context.Background()
	plugin := newMockPlugin("test-plugin")

	// Initial state should be uninitialized
	if plugin.State() != PluginStateUninitialized {
		t.Errorf("expected uninitialized state, got %s", plugin.State())
	}

	// Initialize
	config := map[string]interface{}{"key": "value"}
	if err := plugin.Initialize(ctx, config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !plugin.initCalled {
		t.Error("Initialize was not called")
	}

	if plugin.State() != PluginStateInitialized {
		t.Errorf("expected initialized state, got %s", plugin.State())
	}

	// Start
	if err := plugin.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !plugin.startCalled {
		t.Error("Start was not called")
	}

	if plugin.State() != PluginStateStarted {
		t.Errorf("expected started state, got %s", plugin.State())
	}

	// Stop
	if err := plugin.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if !plugin.stopCalled {
		t.Error("Stop was not called")
	}

	if plugin.State() != PluginStateStopped {
		t.Errorf("expected stopped state, got %s", plugin.State())
	}

	// Shutdown
	if err := plugin.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if !plugin.shutdownCalled {
		t.Error("Shutdown was not called")
	}
}

func TestBasePlugin_CreateMiddleware(t *testing.T) {
	ctx := context.Background()
	plugin := newMockPlugin("test-plugin")

	// Should fail when not started
	_, err := plugin.CreateMiddleware()
	if err == nil {
		t.Error("expected error when creating middleware before start")
	}

	// Initialize and start
	plugin.Initialize(ctx, nil)
	plugin.Start(ctx)

	// Should succeed when started
	middleware, err := plugin.CreateMiddleware()
	if err != nil {
		t.Fatalf("CreateMiddleware failed: %v", err)
	}

	if middleware == nil {
		t.Error("expected middleware, got nil")
	}

	if middleware.Name() != "test-plugin" {
		t.Errorf("expected middleware name 'test-plugin', got %s", middleware.Name())
	}
}

func TestBasePlugin_HealthCheck(t *testing.T) {
	ctx := context.Background()
	plugin := newMockPlugin("test-plugin")

	// Should fail when not started
	if err := plugin.HealthCheck(ctx); err == nil {
		t.Error("expected health check to fail when not started")
	}

	// Initialize and start
	plugin.Initialize(ctx, nil)
	plugin.Start(ctx)

	// Should succeed when started
	if err := plugin.HealthCheck(ctx); err != nil {
		t.Errorf("health check failed: %v", err)
	}

	// Should fail when in failed state
	plugin.SetState(PluginStateFailed)
	if err := plugin.HealthCheck(ctx); err == nil {
		t.Error("expected health check to fail in failed state")
	}
}

func TestPluginRegistry_RegisterPlugin(t *testing.T) {
	registry := NewPluginRegistry()
	plugin := newMockPlugin("test-plugin")

	// Register plugin
	if err := registry.RegisterPlugin(plugin); err != nil {
		t.Fatalf("RegisterPlugin failed: %v", err)
	}

	// Try to register again (should fail)
	if err := registry.RegisterPlugin(plugin); err == nil {
		t.Error("expected error when registering duplicate plugin")
	}

	// Retrieve plugin
	retrieved, err := registry.GetPlugin("test-plugin")
	if err != nil {
		t.Fatalf("GetPlugin failed: %v", err)
	}

	if retrieved != plugin {
		t.Error("retrieved plugin does not match registered plugin")
	}
}

func TestPluginRegistry_RegisterFactory(t *testing.T) {
	registry := NewPluginRegistry()

	factory := func() (Plugin, error) {
		return newMockPlugin("factory-plugin"), nil
	}

	// Register factory
	if err := registry.RegisterFactory("factory-plugin", factory); err != nil {
		t.Fatalf("RegisterFactory failed: %v", err)
	}

	// Try to register again (should fail)
	if err := registry.RegisterFactory("factory-plugin", factory); err == nil {
		t.Error("expected error when registering duplicate factory")
	}

	// Create plugin from factory
	plugin, err := registry.CreatePlugin("factory-plugin")
	if err != nil {
		t.Fatalf("CreatePlugin failed: %v", err)
	}

	if plugin == nil {
		t.Error("expected plugin, got nil")
	}
}

func TestPluginRegistry_ListPlugins(t *testing.T) {
	registry := NewPluginRegistry()

	// List should be empty initially
	if len(registry.ListPlugins()) != 0 {
		t.Error("expected empty plugin list")
	}

	// Register some plugins
	plugin1 := newMockPlugin("plugin1")
	plugin2 := newMockPlugin("plugin2")

	registry.RegisterPlugin(plugin1)
	registry.RegisterPlugin(plugin2)

	// List should contain both plugins
	names := registry.ListPlugins()
	if len(names) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(names))
	}

	// Check that both names are present
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	if !nameMap["plugin1"] || !nameMap["plugin2"] {
		t.Error("plugin names not found in list")
	}
}

func TestPluginRegistry_UnregisterPlugin(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()
	plugin := newMockPlugin("test-plugin")

	// Register and start plugin
	registry.RegisterPlugin(plugin)
	plugin.Initialize(ctx, nil)
	plugin.Start(ctx)

	// Unregister should stop and shutdown the plugin
	if err := registry.UnregisterPlugin(ctx, "test-plugin"); err != nil {
		t.Fatalf("UnregisterPlugin failed: %v", err)
	}

	if !plugin.stopCalled {
		t.Error("Stop was not called during unregister")
	}

	if !plugin.shutdownCalled {
		t.Error("Shutdown was not called during unregister")
	}

	// Plugin should no longer be in registry
	if _, err := registry.GetPlugin("test-plugin"); err == nil {
		t.Error("expected error when getting unregistered plugin")
	}
}

func TestPluginRegistry_InitializeAll(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()

	plugin1 := newMockPlugin("plugin1")
	plugin2 := newMockPlugin("plugin2")

	registry.RegisterPlugin(plugin1)
	registry.RegisterPlugin(plugin2)

	configs := map[string]map[string]interface{}{
		"plugin1": {"key1": "value1"},
		"plugin2": {"key2": "value2"},
	}

	if err := registry.InitializeAll(ctx, configs); err != nil {
		t.Fatalf("InitializeAll failed: %v", err)
	}

	if !plugin1.initCalled || !plugin2.initCalled {
		t.Error("not all plugins were initialized")
	}

	if plugin1.State() != PluginStateInitialized {
		t.Error("plugin1 not in initialized state")
	}

	if plugin2.State() != PluginStateInitialized {
		t.Error("plugin2 not in initialized state")
	}
}

func TestPluginRegistry_StartAll(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()

	plugin1 := newMockPlugin("plugin1")
	plugin2 := newMockPlugin("plugin2")

	registry.RegisterPlugin(plugin1)
	registry.RegisterPlugin(plugin2)

	// Initialize first
	registry.InitializeAll(ctx, nil)

	// Start all
	if err := registry.StartAll(ctx); err != nil {
		t.Fatalf("StartAll failed: %v", err)
	}

	if !plugin1.startCalled || !plugin2.startCalled {
		t.Error("not all plugins were started")
	}

	if plugin1.State() != PluginStateStarted {
		t.Error("plugin1 not in started state")
	}

	if plugin2.State() != PluginStateStarted {
		t.Error("plugin2 not in started state")
	}
}

func TestPluginRegistry_StopAll(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()

	plugin1 := newMockPlugin("plugin1")
	plugin2 := newMockPlugin("plugin2")

	registry.RegisterPlugin(plugin1)
	registry.RegisterPlugin(plugin2)

	// Initialize and start
	registry.InitializeAll(ctx, nil)
	registry.StartAll(ctx)

	// Stop all
	if err := registry.StopAll(ctx); err != nil {
		t.Fatalf("StopAll failed: %v", err)
	}

	if !plugin1.stopCalled || !plugin2.stopCalled {
		t.Error("not all plugins were stopped")
	}
}

func TestPluginRegistry_HealthCheckAll(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()

	plugin1 := newMockPlugin("plugin1")
	plugin2 := newMockPlugin("plugin2")

	registry.RegisterPlugin(plugin1)
	registry.RegisterPlugin(plugin2)

	// Initialize and start plugin1
	plugin1.Initialize(ctx, nil)
	plugin1.Start(ctx)

	// Don't start plugin2

	results := registry.HealthCheckAll(ctx)

	if len(results) != 2 {
		t.Errorf("expected 2 health check results, got %d", len(results))
	}

	// plugin1 should be healthy
	if err := results["plugin1"]; err != nil {
		t.Errorf("plugin1 should be healthy, got error: %v", err)
	}

	// plugin2 should be unhealthy
	if err := results["plugin2"]; err == nil {
		t.Error("plugin2 should be unhealthy")
	}
}

func TestPluginMetadata(t *testing.T) {
	metadata := PluginMetadata{
		Name:         "test-plugin",
		Version:      "1.0.0",
		Description:  "Test plugin",
		Author:       "Test Author",
		Dependencies: []string{"dep1:1.0", "dep2:2.0"},
		Tags:         []string{"test", "example"},
		CreatedAt:    time.Now(),
	}

	plugin := NewBasePlugin(metadata)
	retrieved := plugin.Metadata()

	if retrieved.Name != metadata.Name {
		t.Errorf("expected name %s, got %s", metadata.Name, retrieved.Name)
	}

	if retrieved.Version != metadata.Version {
		t.Errorf("expected version %s, got %s", metadata.Version, retrieved.Version)
	}

	if len(retrieved.Dependencies) != len(metadata.Dependencies) {
		t.Error("dependencies length mismatch")
	}

	if len(retrieved.Tags) != len(metadata.Tags) {
		t.Error("tags length mismatch")
	}
}

func TestPluginRegistry_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()

	// Register multiple plugins concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			plugin := newMockPlugin(fmt.Sprintf("plugin-%d", id))
			registry.RegisterPlugin(plugin)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all plugins are registered
	if len(registry.ListPlugins()) != 10 {
		t.Errorf("expected 10 plugins, got %d", len(registry.ListPlugins()))
	}

	// Concurrent access to plugins
	for i := 0; i < 10; i++ {
		go func(id int) {
			name := fmt.Sprintf("plugin-%d", id)
			if _, err := registry.GetPlugin(name); err != nil {
				t.Errorf("failed to get plugin %s: %v", name, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent health checks
	registry.InitializeAll(ctx, nil)
	registry.StartAll(ctx)

	for i := 0; i < 10; i++ {
		go func() {
			registry.HealthCheckAll(ctx)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
