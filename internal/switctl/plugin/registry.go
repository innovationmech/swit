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

package plugin

import (
	"fmt"
	"sort"
	"sync"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// PluginRegistry manages plugin metadata and discovery.
type PluginRegistry struct {
	plugins  map[string]*PluginInfo
	metadata map[string]*PluginMetadata
	mu       sync.RWMutex
}

// NewPluginRegistry creates a new plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins:  make(map[string]*PluginInfo),
		metadata: make(map[string]*PluginMetadata),
	}
}

// RegisterPlugin registers a plugin with the registry.
func (r *PluginRegistry) RegisterPlugin(name string, info *PluginInfo) error {
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if info == nil {
		return fmt.Errorf("plugin info cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins[name] = info
	r.metadata[name] = &info.Metadata
	return nil
}

// UnregisterPlugin removes a plugin from the registry.
func (r *PluginRegistry) UnregisterPlugin(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	delete(r.plugins, name)
	delete(r.metadata, name)
	return nil
}

// GetPluginInfo retrieves plugin information by name.
func (r *PluginRegistry) GetPluginInfo(name string) (*PluginInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.plugins[name]
	return info, exists
}

// GetAllPluginInfo returns all plugin information.
func (r *PluginRegistry) GetAllPluginInfo() []*PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allInfo []*PluginInfo
	for _, info := range r.plugins {
		allInfo = append(allInfo, info)
	}

	return allInfo
}

// ListPlugins returns a sorted list of plugin names.
func (r *PluginRegistry) ListPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// FindPluginsByCapability finds plugins that have a specific capability.
func (r *PluginRegistry) FindPluginsByCapability(capability string) []*PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*PluginInfo
	for _, info := range r.plugins {
		for _, cap := range info.Metadata.Capabilities {
			if cap == capability {
				matches = append(matches, info)
				break
			}
		}
	}

	return matches
}

// FindPluginsByAuthor finds plugins by a specific author.
func (r *PluginRegistry) FindPluginsByAuthor(author string) []*PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*PluginInfo
	for _, info := range r.plugins {
		if info.Metadata.Author == author {
			matches = append(matches, info)
		}
	}

	return matches
}

// GetPluginsByStatus returns plugins filtered by status.
func (r *PluginRegistry) GetPluginsByStatus(status PluginStatus) []*PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*PluginInfo
	for _, info := range r.plugins {
		if info.Status == status {
			matches = append(matches, info)
		}
	}

	return matches
}

// UpdatePluginStatus updates the status of a plugin.
func (r *PluginRegistry) UpdatePluginStatus(name string, status PluginStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.plugins[name]
	if !exists {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginNotFound,
			"Plugin not found: "+name,
			nil,
		)
	}

	info.Status = status
	return nil
}

// GetStatistics returns registry statistics.
func (r *PluginRegistry) GetStatistics() RegistryStatistics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RegistryStatistics{
		TotalPlugins: len(r.plugins),
		StatusCounts: make(map[PluginStatus]int),
	}

	for _, info := range r.plugins {
		stats.StatusCounts[info.Status]++
	}

	return stats
}
