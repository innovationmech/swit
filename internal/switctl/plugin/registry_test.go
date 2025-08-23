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
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type RegistryTestSuite struct {
	suite.Suite
	registry *PluginRegistry
}

func (s *RegistryTestSuite) SetupTest() {
	s.registry = NewPluginRegistry()
}

func (s *RegistryTestSuite) TestRegisterPlugin() {
	info := &PluginInfo{
		Metadata: PluginMetadata{
			Name:         "test-plugin",
			Version:      "1.0.0",
			Description:  "Test plugin",
			Author:       "Test Author",
			LoadedAt:     time.Now(),
			Capabilities: []string{"test"},
		},
		Status: PluginStatusLoaded,
	}

	err := s.registry.RegisterPlugin("test-plugin", info)
	s.NoError(err)

	retrievedInfo, exists := s.registry.GetPluginInfo("test-plugin")
	s.True(exists)
	s.NotNil(retrievedInfo)
	s.Equal("test-plugin", retrievedInfo.Metadata.Name)
	s.Equal("1.0.0", retrievedInfo.Metadata.Version)
	s.Equal("Test Author", retrievedInfo.Metadata.Author)
}

func (s *RegistryTestSuite) TestRegisterPluginOverwrite() {
	info1 := &PluginInfo{
		Metadata: PluginMetadata{
			Name:    "test-plugin",
			Version: "1.0.0",
			Author:  "Author 1",
		},
		Status: PluginStatusLoaded,
	}

	info2 := &PluginInfo{
		Metadata: PluginMetadata{
			Name:    "test-plugin",
			Version: "2.0.0",
			Author:  "Author 2",
		},
		Status: PluginStatusActive,
	}

	err := s.registry.RegisterPlugin("test-plugin", info1)
	s.NoError(err)

	err = s.registry.RegisterPlugin("test-plugin", info2)
	s.NoError(err)

	retrievedInfo, exists := s.registry.GetPluginInfo("test-plugin")
	s.True(exists)
	s.NotNil(retrievedInfo)
	s.Equal("2.0.0", retrievedInfo.Metadata.Version)
	s.Equal("Author 2", retrievedInfo.Metadata.Author)
	s.Equal(PluginStatusActive, retrievedInfo.Status)
}

func (s *RegistryTestSuite) TestUnregisterPlugin() {
	info := &PluginInfo{
		Metadata: PluginMetadata{
			Name: "test-plugin",
		},
		Status: PluginStatusLoaded,
	}

	err := s.registry.RegisterPlugin("test-plugin", info)
	s.NoError(err)

	err = s.registry.UnregisterPlugin("test-plugin")
	s.NoError(err)

	retrievedInfo, exists := s.registry.GetPluginInfo("test-plugin")
	s.False(exists)
	s.Nil(retrievedInfo)
}

func (s *RegistryTestSuite) TestUnregisterNonExistentPlugin() {
	err := s.registry.UnregisterPlugin("non-existent")
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *RegistryTestSuite) TestGetAllPluginInfo() {
	info1 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin1"}, Status: PluginStatusActive}
	info2 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin2"}, Status: PluginStatusLoaded}
	info3 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin3"}, Status: PluginStatusDisabled}

	s.registry.RegisterPlugin("plugin1", info1)
	s.registry.RegisterPlugin("plugin2", info2)
	s.registry.RegisterPlugin("plugin3", info3)

	allInfo := s.registry.GetAllPluginInfo()
	s.Len(allInfo, 3)

	names := make(map[string]bool)
	for _, info := range allInfo {
		names[info.Metadata.Name] = true
	}

	s.True(names["plugin1"])
	s.True(names["plugin2"])
	s.True(names["plugin3"])
}

func (s *RegistryTestSuite) TestListPlugins() {
	info1 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin1"}, Status: PluginStatusActive}
	info2 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin2"}, Status: PluginStatusLoaded}

	s.registry.RegisterPlugin("plugin1", info1)
	s.registry.RegisterPlugin("plugin2", info2)

	plugins := s.registry.ListPlugins()
	s.Len(plugins, 2)
	s.Contains(plugins, "plugin1")
	s.Contains(plugins, "plugin2")
}

func (s *RegistryTestSuite) TestFindPluginsByCapability() {
	info1 := &PluginInfo{
		Metadata: PluginMetadata{
			Name:         "plugin1",
			Capabilities: []string{"auth", "logging"},
		},
		Status: PluginStatusActive,
	}
	info2 := &PluginInfo{
		Metadata: PluginMetadata{
			Name:         "plugin2",
			Capabilities: []string{"database", "auth"},
		},
		Status: PluginStatusActive,
	}
	info3 := &PluginInfo{
		Metadata: PluginMetadata{
			Name:         "plugin3",
			Capabilities: []string{"metrics"},
		},
		Status: PluginStatusActive,
	}

	s.registry.RegisterPlugin("plugin1", info1)
	s.registry.RegisterPlugin("plugin2", info2)
	s.registry.RegisterPlugin("plugin3", info3)

	authPlugins := s.registry.FindPluginsByCapability("auth")
	s.Len(authPlugins, 2)

	names := make(map[string]bool)
	for _, info := range authPlugins {
		names[info.Metadata.Name] = true
	}
	s.True(names["plugin1"])
	s.True(names["plugin2"])

	dbPlugins := s.registry.FindPluginsByCapability("database")
	s.Len(dbPlugins, 1)
	s.Equal("plugin2", dbPlugins[0].Metadata.Name)

	nonExistentPlugins := s.registry.FindPluginsByCapability("nonexistent")
	s.Len(nonExistentPlugins, 0)
}

func (s *RegistryTestSuite) TestFindPluginsByAuthor() {
	info1 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin1", Author: "John Doe"}, Status: PluginStatusActive}
	info2 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin2", Author: "John Doe"}, Status: PluginStatusActive}
	info3 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin3", Author: "Jane Smith"}, Status: PluginStatusActive}

	s.registry.RegisterPlugin("plugin1", info1)
	s.registry.RegisterPlugin("plugin2", info2)
	s.registry.RegisterPlugin("plugin3", info3)

	johnPlugins := s.registry.FindPluginsByAuthor("John Doe")
	s.Len(johnPlugins, 2)

	names := make(map[string]bool)
	for _, info := range johnPlugins {
		names[info.Metadata.Name] = true
	}
	s.True(names["plugin1"])
	s.True(names["plugin2"])

	janePlugins := s.registry.FindPluginsByAuthor("Jane Smith")
	s.Len(janePlugins, 1)
	s.Equal("plugin3", janePlugins[0].Metadata.Name)

	nonExistentPlugins := s.registry.FindPluginsByAuthor("Non Existent")
	s.Len(nonExistentPlugins, 0)
}

func (s *RegistryTestSuite) TestGetPluginsByStatus() {
	info1 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin1"}, Status: PluginStatusActive}
	info2 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin2"}, Status: PluginStatusActive}
	info3 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin3"}, Status: PluginStatusDisabled}
	info4 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin4"}, Status: PluginStatusLoaded}

	s.registry.RegisterPlugin("plugin1", info1)
	s.registry.RegisterPlugin("plugin2", info2)
	s.registry.RegisterPlugin("plugin3", info3)
	s.registry.RegisterPlugin("plugin4", info4)

	activePlugins := s.registry.GetPluginsByStatus(PluginStatusActive)
	s.Len(activePlugins, 2)

	names := make(map[string]bool)
	for _, info := range activePlugins {
		names[info.Metadata.Name] = true
	}
	s.True(names["plugin1"])
	s.True(names["plugin2"])

	disabledPlugins := s.registry.GetPluginsByStatus(PluginStatusDisabled)
	s.Len(disabledPlugins, 1)
	s.Equal("plugin3", disabledPlugins[0].Metadata.Name)

	errorPlugins := s.registry.GetPluginsByStatus(PluginStatusError)
	s.Len(errorPlugins, 0)
}

func (s *RegistryTestSuite) TestUpdatePluginStatus() {
	info := &PluginInfo{
		Metadata: PluginMetadata{
			Name: "test-plugin",
		},
		Status: PluginStatusLoaded,
	}

	s.registry.RegisterPlugin("test-plugin", info)

	err := s.registry.UpdatePluginStatus("test-plugin", PluginStatusActive)
	s.NoError(err)

	retrievedInfo, exists := s.registry.GetPluginInfo("test-plugin")
	s.True(exists)
	s.NotNil(retrievedInfo)
	s.Equal(PluginStatusActive, retrievedInfo.Status)
}

func (s *RegistryTestSuite) TestUpdatePluginStatusNonExistent() {
	err := s.registry.UpdatePluginStatus("non-existent", PluginStatusActive)
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *RegistryTestSuite) TestGetStatistics() {
	info1 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin1"}, Status: PluginStatusActive}
	info2 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin2"}, Status: PluginStatusActive}
	info3 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin3"}, Status: PluginStatusDisabled}
	info4 := &PluginInfo{Metadata: PluginMetadata{Name: "plugin4"}, Status: PluginStatusError}

	s.registry.RegisterPlugin("plugin1", info1)
	s.registry.RegisterPlugin("plugin2", info2)
	s.registry.RegisterPlugin("plugin3", info3)
	s.registry.RegisterPlugin("plugin4", info4)

	stats := s.registry.GetStatistics()
	s.Equal(4, stats.TotalPlugins)
	s.Equal(2, stats.StatusCounts[PluginStatusActive])
	s.Equal(1, stats.StatusCounts[PluginStatusDisabled])
	s.Equal(1, stats.StatusCounts[PluginStatusError])
}

func (s *RegistryTestSuite) TestGetStatisticsEmpty() {
	stats := s.registry.GetStatistics()
	s.Equal(0, stats.TotalPlugins)
	s.Equal(0, stats.StatusCounts[PluginStatusActive])
	s.Equal(0, stats.StatusCounts[PluginStatusDisabled])
	s.Equal(0, stats.StatusCounts[PluginStatusError])
}

func (s *RegistryTestSuite) TestConcurrentAccess() {
	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	// Start multiple goroutines to test concurrent access
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				pluginName := fmt.Sprintf("plugin-%d-%d", id, j)
				info := &PluginInfo{
					Metadata: PluginMetadata{
						Name: pluginName,
					},
					Status: PluginStatusLoaded,
				}

				// Register plugin
				err := s.registry.RegisterPlugin(pluginName, info)
				s.NoError(err)

				// Get plugin info
				retrievedInfo, exists := s.registry.GetPluginInfo(pluginName)
				s.True(exists)
				s.NotNil(retrievedInfo)

				// Update status
				err = s.registry.UpdatePluginStatus(pluginName, PluginStatusActive)
				s.NoError(err)

				// Unregister plugin
				err = s.registry.UnregisterPlugin(pluginName)
				s.NoError(err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify registry is empty
	stats := s.registry.GetStatistics()
	s.Equal(0, stats.TotalPlugins)
}

func TestRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}
