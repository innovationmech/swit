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
	"os"
	"path/filepath"
	"testing"
)

func TestPluginLoader_LoadFromFactory(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()
	loader := NewPluginLoader(registry)

	factory := func() (Plugin, error) {
		return newMockPlugin("factory-test"), nil
	}

	config := map[string]interface{}{
		"key": "value",
	}

	// Load plugin from factory
	if err := loader.LoadFromFactory(ctx, "factory-test", factory, config); err != nil {
		t.Fatalf("LoadFromFactory failed: %v", err)
	}

	// Verify plugin is registered
	plugin, err := registry.GetPlugin("factory-test")
	if err != nil {
		t.Fatalf("plugin not found in registry: %v", err)
	}

	// Verify plugin is initialized
	if plugin.State() != PluginStateInitialized {
		t.Errorf("expected initialized state, got %s", plugin.State())
	}

	// Verify plugin is in loaded plugins list
	loadedPlugins := loader.GetLoadedPlugins()
	if len(loadedPlugins) != 1 {
		t.Errorf("expected 1 loaded plugin, got %d", len(loadedPlugins))
	}
}

func TestPluginLoader_LoadMultiple(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()
	loader := NewPluginLoader(registry)

	factories := map[string]PluginFactory{
		"plugin1": func() (Plugin, error) {
			return newMockPlugin("plugin1"), nil
		},
		"plugin2": func() (Plugin, error) {
			return newMockPlugin("plugin2"), nil
		},
		"plugin3": func() (Plugin, error) {
			return newMockPlugin("plugin3"), nil
		},
	}

	configs := map[string]map[string]interface{}{
		"plugin1": {"setting": "value1"},
		"plugin2": {"setting": "value2"},
	}

	// Load multiple plugins
	if err := loader.LoadMultiple(ctx, factories, configs); err != nil {
		t.Fatalf("LoadMultiple failed: %v", err)
	}

	// Verify all plugins are loaded
	loadedPlugins := loader.GetLoadedPlugins()
	if len(loadedPlugins) != 3 {
		t.Errorf("expected 3 loaded plugins, got %d", len(loadedPlugins))
	}

	// Verify each plugin
	for name := range factories {
		plugin, err := registry.GetPlugin(name)
		if err != nil {
			t.Errorf("plugin %s not found: %v", name, err)
			continue
		}

		if plugin.State() != PluginStateInitialized {
			t.Errorf("plugin %s not initialized", name)
		}
	}
}

func TestPluginLoader_Unload(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()
	loader := NewPluginLoader(registry)

	factory := func() (Plugin, error) {
		return newMockPlugin("unload-test"), nil
	}

	// Load plugin
	if err := loader.LoadFromFactory(ctx, "unload-test", factory, nil); err != nil {
		t.Fatalf("LoadFromFactory failed: %v", err)
	}

	// Start the plugin
	plugin, _ := registry.GetPlugin("unload-test")
	plugin.Start(ctx)

	// Unload plugin
	if err := loader.Unload(ctx, "unload-test"); err != nil {
		t.Fatalf("Unload failed: %v", err)
	}

	// Verify plugin is unloaded
	if _, err := registry.GetPlugin("unload-test"); err == nil {
		t.Error("expected error when getting unloaded plugin")
	}

	// Verify plugin is not in loaded plugins list
	loadedPlugins := loader.GetLoadedPlugins()
	if len(loadedPlugins) != 0 {
		t.Errorf("expected 0 loaded plugins, got %d", len(loadedPlugins))
	}
}

func TestPluginLoader_UnloadAll(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()
	loader := NewPluginLoader(registry)

	// Load multiple plugins
	factories := map[string]PluginFactory{
		"plugin1": func() (Plugin, error) { return newMockPlugin("plugin1"), nil },
		"plugin2": func() (Plugin, error) { return newMockPlugin("plugin2"), nil },
		"plugin3": func() (Plugin, error) { return newMockPlugin("plugin3"), nil },
	}

	loader.LoadMultiple(ctx, factories, nil)

	// Unload all
	if err := loader.UnloadAll(ctx); err != nil {
		t.Fatalf("UnloadAll failed: %v", err)
	}

	// Verify no plugins are loaded
	if len(loader.GetLoadedPlugins()) != 0 {
		t.Error("expected no loaded plugins after UnloadAll")
	}

	// Verify no plugins in registry
	if len(registry.ListPlugins()) != 0 {
		t.Error("expected no plugins in registry after UnloadAll")
	}
}

func TestPluginLoader_Reload(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()
	loader := NewPluginLoader(registry)

	factory := func() (Plugin, error) {
		return newMockPlugin("reload-test"), nil
	}

	// Load and register factory
	if err := loader.LoadFromFactory(ctx, "reload-test", factory, nil); err != nil {
		t.Fatalf("LoadFromFactory failed: %v", err)
	}

	// Start the plugin
	plugin, _ := registry.GetPlugin("reload-test")
	plugin.Start(ctx)

	// Reload with new config
	newConfig := map[string]interface{}{
		"new_setting": "new_value",
	}

	if err := loader.Reload(ctx, "reload-test", newConfig); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	// Verify new plugin instance
	reloadedPlugin, err := registry.GetPlugin("reload-test")
	if err != nil {
		t.Fatalf("reloaded plugin not found: %v", err)
	}

	// Plugin should be initialized but not started
	if reloadedPlugin.State() != PluginStateInitialized {
		t.Errorf("expected initialized state after reload, got %s", reloadedPlugin.State())
	}
}

func TestPluginDiscovery_AddSearchPath(t *testing.T) {
	discovery := NewPluginDiscovery()

	// Initially should have no search paths
	discovery.AddSearchPath("/path/to/plugins1")
	discovery.AddSearchPath("/path/to/plugins2")

	// Note: We can't directly test the search paths as they're private,
	// but we can test the behavior through Discover()
}

func TestPluginDiscovery_Discover(t *testing.T) {
	// Create a temporary directory for test plugins
	tmpDir, err := os.MkdirTemp("", "plugin-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test .so files
	testFiles := []string{"plugin1.so", "plugin2.so", "notaplugin.txt"}
	for _, filename := range testFiles {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte("fake plugin"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Create discovery with the temp directory
	discovery := NewPluginDiscovery(tmpDir)

	// Discover plugins
	pluginPaths, err := discovery.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should find only .so files
	if len(pluginPaths) != 2 {
		t.Errorf("expected 2 plugin paths, got %d", len(pluginPaths))
	}

	// Verify paths are correct
	for _, path := range pluginPaths {
		if filepath.Ext(path) != ".so" {
			t.Errorf("expected .so extension, got %s", path)
		}
	}
}

func TestPluginDiscovery_DiscoverNonexistentPath(t *testing.T) {
	discovery := NewPluginDiscovery("/nonexistent/path")

	// Should return error for nonexistent path
	_, err := discovery.Discover()
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestPluginLoader_GetLoadedPlugins(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()
	loader := NewPluginLoader(registry)

	// Initially should be empty
	if len(loader.GetLoadedPlugins()) != 0 {
		t.Error("expected no loaded plugins initially")
	}

	// Load some plugins
	factories := map[string]PluginFactory{
		"plugin1": func() (Plugin, error) { return newMockPlugin("plugin1"), nil },
		"plugin2": func() (Plugin, error) { return newMockPlugin("plugin2"), nil },
	}

	loader.LoadMultiple(ctx, factories, nil)

	// Should return loaded plugin names
	loadedPlugins := loader.GetLoadedPlugins()
	if len(loadedPlugins) != 2 {
		t.Errorf("expected 2 loaded plugins, got %d", len(loadedPlugins))
	}

	// Verify names are present
	nameMap := make(map[string]bool)
	for _, name := range loadedPlugins {
		nameMap[name] = true
	}

	if !nameMap["plugin1"] || !nameMap["plugin2"] {
		t.Error("expected plugin names not found")
	}
}

func TestPluginLoader_FailedInitialization(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()
	loader := NewPluginLoader(registry)

	// Create a factory that returns a plugin that fails initialization
	factory := func() (Plugin, error) {
		plugin := newMockPlugin("fail-init")
		plugin.shouldFailInit = true
		return plugin, nil
	}

	// Should fail to load
	if err := loader.LoadFromFactory(ctx, "fail-init", factory, nil); err == nil {
		t.Error("expected error when plugin initialization fails")
	}

	// Plugin should not be in loaded plugins
	if len(loader.GetLoadedPlugins()) != 0 {
		t.Error("expected no loaded plugins after failed initialization")
	}
}

func TestPluginLoader_ConcurrentLoading(t *testing.T) {
	ctx := context.Background()
	registry := NewPluginRegistry()
	loader := NewPluginLoader(registry)

	// Load plugins concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			factory := func() (Plugin, error) {
				return newMockPlugin("concurrent-plugin"), nil
			}

			// Note: This will fail for duplicates, which is expected
			loader.LoadFromFactory(ctx, "concurrent-plugin", factory, nil)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Only one plugin should be loaded (the first one wins)
	loadedPlugins := loader.GetLoadedPlugins()
	if len(loadedPlugins) != 1 {
		t.Errorf("expected 1 loaded plugin, got %d", len(loadedPlugins))
	}
}

func BenchmarkPluginLoader_LoadFromFactory(b *testing.B) {
	ctx := context.Background()

	factory := func() (Plugin, error) {
		return newMockPlugin("benchmark-plugin"), nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry := NewPluginRegistry()
		loader := NewPluginLoader(registry)
		loader.LoadFromFactory(ctx, "benchmark-plugin", factory, nil)
	}
}

func BenchmarkPluginLoader_LoadMultiple(b *testing.B) {
	ctx := context.Background()

	factories := map[string]PluginFactory{
		"plugin1": func() (Plugin, error) { return newMockPlugin("plugin1"), nil },
		"plugin2": func() (Plugin, error) { return newMockPlugin("plugin2"), nil },
		"plugin3": func() (Plugin, error) { return newMockPlugin("plugin3"), nil },
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry := NewPluginRegistry()
		loader := NewPluginLoader(registry)
		loader.LoadMultiple(ctx, factories, nil)
	}
}
