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

package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TemplateStoreTestSuite provides test suite for template store functionality.
type TemplateStoreTestSuite struct {
	suite.Suite
	store   *Store
	tempDir string
}

// SetupTest sets up the test environment before each test.
func (suite *TemplateStoreTestSuite) SetupTest() {
	var err error
	suite.tempDir, err = os.MkdirTemp("", "template-store-test-*")
	suite.Require().NoError(err)

	// Create test templates
	testTemplates := map[string]string{
		"simple.tmpl":     `Hello {{.Name}}!`,
		"service.tmpl":    `package {{.Package}}\n\ntype Service struct{}`,
		"handler.tmpl":    `func Handle() {}\n`,
		"config.tmpl":     `type Config struct {\n\tHost string\n}`,
		"nested/sub.tmpl": `Nested template: {{.Value}}`,
	}

	for name, content := range testTemplates {
		path := filepath.Join(suite.tempDir, name)
		dir := filepath.Dir(path)
		err := os.MkdirAll(dir, 0755)
		suite.Require().NoError(err)

		err = os.WriteFile(path, []byte(content), 0644)
		suite.Require().NoError(err)
	}

	suite.store = NewStore(suite.tempDir)
}

// TearDownTest cleans up after each test.
func (suite *TemplateStoreTestSuite) TearDownTest() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// TestNewStore tests store creation.
func (suite *TemplateStoreTestSuite) TestNewStore() {
	suite.NotNil(suite.store)
	suite.Equal(suite.tempDir, suite.store.baseDir)
	suite.NotNil(suite.store.cache)
	suite.NotNil(suite.store.versions)
	suite.NotNil(suite.store.watchers)
	suite.False(suite.store.hotReload)
	suite.False(suite.store.useEmbedded)
}

// TestNewStoreWithEmbedded tests store creation with embedded filesystem.
func (suite *TemplateStoreTestSuite) TestNewStoreWithEmbedded() {
	embeddedStore := NewStoreWithEmbedded(embeddedTemplates)
	suite.NotNil(embeddedStore)
	suite.Equal("", embeddedStore.baseDir)
	suite.NotNil(embeddedStore.cache)
	suite.NotNil(embeddedStore.versions)
	suite.NotNil(embeddedStore.watchers)
	suite.False(embeddedStore.hotReload)
	suite.True(embeddedStore.useEmbedded)
}

// TestGetTemplate tests template retrieval functionality.
func (suite *TemplateStoreTestSuite) TestGetTemplate() {
	tests := []struct {
		name         string
		templateName string
		expectError  bool
		errorMsg     string
		validateFunc func(string) bool
	}{
		{
			name:         "Get existing template",
			templateName: "simple",
			expectError:  false,
			validateFunc: func(content string) bool {
				return strings.Contains(content, "Hello {{.Name}}!")
			},
		},
		{
			name:         "Get nested template",
			templateName: "nested/sub",
			expectError:  false,
			validateFunc: func(content string) bool {
				return strings.Contains(content, "Nested template: {{.Value}}")
			},
		},
		{
			name:         "Get non-existent template",
			templateName: "non_existent",
			expectError:  true,
			errorMsg:     "template non_existent not found",
		},
		{
			name:         "Get cached template",
			templateName: "simple",
			expectError:  false,
			validateFunc: func(content string) bool {
				return strings.Contains(content, "Hello {{.Name}}!")
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			content, err := suite.store.GetTemplate(tt.templateName)

			if tt.expectError {
				suite.Error(err)
				if tt.errorMsg != "" {
					suite.Contains(err.Error(), tt.errorMsg)
				}
			} else {
				suite.NoError(err)
				suite.NotEmpty(content)

				if tt.validateFunc != nil {
					suite.True(tt.validateFunc(content), "Content validation failed: %s", content)
				}

				// Verify caching - second call should be from cache
				content2, err2 := suite.store.GetTemplate(tt.templateName)
				suite.NoError(err2)
				suite.Equal(content, content2)
			}
		})
	}
}

// TestSetTemplate tests template storage functionality.
func (suite *TemplateStoreTestSuite) TestSetTemplate() {
	tests := []struct {
		name         string
		templateName string
		content      string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "Set new template",
			templateName: "new_template",
			content:      "New template content: {{.Data}}",
			expectError:  false,
		},
		{
			name:         "Overwrite existing template",
			templateName: "simple",
			content:      "Updated content: {{.UpdatedData}}",
			expectError:  false,
		},
		{
			name:         "Set template with nested path",
			templateName: "nested/deep/new",
			content:      "Deep nested template",
			expectError:  false,
		},
		{
			name:         "Set empty template",
			templateName: "empty",
			content:      "",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := suite.store.SetTemplate(tt.templateName, tt.content)

			if tt.expectError {
				suite.Error(err)
				if tt.errorMsg != "" {
					suite.Contains(err.Error(), tt.errorMsg)
				}
			} else {
				suite.NoError(err)

				// Verify the template was stored correctly
				retrievedContent, err := suite.store.GetTemplate(tt.templateName)
				suite.NoError(err)
				suite.Equal(tt.content, retrievedContent)

				// Check cache entry
				suite.store.mu.RLock()
				cached, exists := suite.store.cache[tt.templateName]
				suite.store.mu.RUnlock()
				suite.True(exists)
				suite.Equal(tt.content, cached.Content)
				suite.NotEmpty(cached.Hash)
				suite.Equal(int64(len(tt.content)), cached.Size)
			}
		})
	}
}

// TestRemoveTemplate tests template removal functionality.
func (suite *TemplateStoreTestSuite) TestRemoveTemplate() {
	// First, set a template
	err := suite.store.SetTemplate("to_remove", "Content to remove")
	suite.NoError(err)

	// Verify it exists
	content, err := suite.store.GetTemplate("to_remove")
	suite.NoError(err)
	suite.Equal("Content to remove", content)

	// Remove it
	err = suite.store.RemoveTemplate("to_remove")
	suite.NoError(err)

	// Verify it's gone from cache
	suite.store.mu.RLock()
	_, exists := suite.store.cache["to_remove"]
	suite.store.mu.RUnlock()
	suite.False(exists)

	// Verify it's gone from filesystem
	_, err = suite.store.GetTemplate("to_remove")
	suite.Error(err)

	// Test removing non-existent template (should not error)
	err = suite.store.RemoveTemplate("non_existent")
	suite.NoError(err)
}

// TestListTemplates tests listing all available templates.
func (suite *TemplateStoreTestSuite) TestListTemplates() {
	// Add some templates to cache first
	err := suite.store.SetTemplate("cached_template", "Cached content")
	suite.NoError(err)

	templates, err := suite.store.ListTemplates()
	suite.NoError(err)
	suite.NotEmpty(templates)

	// Check that we have both cached and filesystem templates
	templateNames := make(map[string]bool)
	hashedCount := 0
	for _, tmpl := range templates {
		templateNames[tmpl.Name] = true
		suite.NotEmpty(tmpl.Name)
		suite.NotEmpty(tmpl.Path)
		suite.Greater(tmpl.Size, int64(0))
		// Hash might be empty for non-cached templates (performance optimization)
		if tmpl.Hash != "" {
			hashedCount++
		}
	}
	// At least cached templates should have hashes
	suite.Greater(hashedCount, 0, "At least some templates should have hashes calculated")

	// Check specific templates exist
	suite.True(templateNames["simple"])
	suite.True(templateNames["service"])
	suite.True(templateNames["handler"])
	suite.True(templateNames["config"])
	suite.True(templateNames["nested/sub"])
	suite.True(templateNames["cached_template"])
}

// TestGetTemplateInfo tests getting information about a specific template.
func (suite *TemplateStoreTestSuite) TestGetTemplateInfo() {
	tests := []struct {
		name         string
		templateName string
		setup        func()
		expectError  bool
		errorMsg     string
		validateFunc func(*TemplateInfo) bool
	}{
		{
			name:         "Get info for cached template",
			templateName: "cached_info",
			setup: func() {
				err := suite.store.SetTemplate("cached_info", "Cached template for info")
				suite.NoError(err)
			},
			expectError: false,
			validateFunc: func(info *TemplateInfo) bool {
				return info.Name == "cached_info" &&
					info.Size > 0 &&
					info.Hash != "" &&
					info.AccessCount >= 0
			},
		},
		{
			name:         "Get info for filesystem template",
			templateName: "simple",
			setup:        func() {},
			expectError:  false,
			validateFunc: func(info *TemplateInfo) bool {
				return info.Name == "simple" &&
					info.Size > 0 &&
					!info.ModTime.IsZero()
			},
		},
		{
			name:         "Get info for non-existent template",
			templateName: "non_existent",
			setup:        func() {},
			expectError:  true,
			errorMsg:     "template non_existent not found",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setup()

			info, err := suite.store.GetTemplateInfo(tt.templateName)

			if tt.expectError {
				suite.Error(err)
				suite.Nil(info)
				if tt.errorMsg != "" {
					suite.Contains(err.Error(), tt.errorMsg)
				}
			} else {
				suite.NoError(err)
				suite.NotNil(info)

				if tt.validateFunc != nil {
					suite.True(tt.validateFunc(info), "Template info validation failed")
				}
			}
		})
	}
}

// TestVersioning tests template versioning functionality.
func (suite *TemplateStoreTestSuite) TestVersioning() {
	templateName := "versioned_template"

	// Initially no version
	version := suite.store.GetVersion(templateName)
	suite.Empty(version)

	// Set version
	suite.store.SetVersion(templateName, "1.0.0")
	version = suite.store.GetVersion(templateName)
	suite.Equal("1.0.0", version)

	// Update version
	suite.store.SetVersion(templateName, "1.1.0")
	version = suite.store.GetVersion(templateName)
	suite.Equal("1.1.0", version)

	// Check version in template info
	err := suite.store.SetTemplate(templateName, "Versioned content")
	suite.NoError(err)

	info, err := suite.store.GetTemplateInfo(templateName)
	suite.NoError(err)
	suite.Equal("1.1.0", info.Version)
}

// TestHotReload tests hot reload functionality.
func (suite *TemplateStoreTestSuite) TestHotReload() {
	// Test enabling hot reload
	err := suite.store.EnableHotReload()
	suite.NoError(err)
	suite.True(suite.store.hotReload)

	// Test disabling hot reload
	suite.store.DisableHotReload()
	suite.False(suite.store.hotReload)
	suite.Empty(suite.store.watchers)

	// Test hot reload with embedded store (should error)
	embeddedStore := NewStoreWithEmbedded(embeddedTemplates)
	err = embeddedStore.EnableHotReload()
	suite.Error(err)
	suite.Contains(err.Error(), "hot reload not supported for embedded templates")
}

// TestCacheOperations tests cache-related operations.
func (suite *TemplateStoreTestSuite) TestCacheOperations() {
	// Add some templates to cache
	err := suite.store.SetTemplate("cache_test1", "Content 1")
	suite.NoError(err)
	err = suite.store.SetTemplate("cache_test2", "Content 2")
	suite.NoError(err)

	// Check cache stats
	stats := suite.store.GetCacheStats()
	suite.NotNil(stats)
	suite.Equal(2, stats["cached_templates"])
	suite.Greater(stats["total_size"].(int64), int64(0))
	suite.Equal(int64(0), stats["total_access"].(int64)) // No access yet
	suite.False(stats["hot_reload"].(bool))
	suite.False(stats["use_embedded"].(bool))

	// Access templates to increase access count
	_, err = suite.store.GetTemplate("cache_test1")
	suite.NoError(err)
	_, err = suite.store.GetTemplate("cache_test2")
	suite.NoError(err)

	// Check updated stats
	stats = suite.store.GetCacheStats()
	suite.Equal(int64(2), stats["total_access"].(int64))

	// Clear cache
	suite.store.ClearCache()
	stats = suite.store.GetCacheStats()
	suite.Equal(0, stats["cached_templates"])
	suite.Equal(int64(0), stats["total_size"].(int64))
	suite.Equal(int64(0), stats["total_access"].(int64))
}

// TestPreloadTemplates tests template preloading functionality.
func (suite *TemplateStoreTestSuite) TestPreloadTemplates() {
	// Initially no templates in cache
	stats := suite.store.GetCacheStats()
	suite.Equal(0, stats["cached_templates"])

	// Preload templates
	err := suite.store.PreloadTemplates()
	suite.NoError(err)

	// Check that templates are now cached
	stats = suite.store.GetCacheStats()
	suite.Greater(stats["cached_templates"], 0)

	// Verify specific templates are loaded
	templateNames := []string{"simple", "service", "handler", "config", "nested/sub"}
	for _, name := range templateNames {
		suite.store.mu.RLock()
		_, exists := suite.store.cache[name]
		suite.store.mu.RUnlock()
		suite.True(exists, "Template %s should be preloaded", name)
	}
}

// TestExportTemplates tests template export functionality.
func (suite *TemplateStoreTestSuite) TestExportTemplates() {
	exportDir, err := os.MkdirTemp("", "template-export-test-*")
	suite.NoError(err)
	defer os.RemoveAll(exportDir)

	// Add some templates to cache
	err = suite.store.SetTemplate("export_test1", "Export content 1")
	suite.NoError(err)
	err = suite.store.SetTemplate("export_test2", "Export content 2")
	suite.NoError(err)

	// Export templates
	err = suite.store.ExportTemplates(exportDir)
	suite.NoError(err)

	// Verify exported files exist
	templates, err := suite.store.ListTemplates()
	suite.NoError(err)

	for _, tmpl := range templates {
		exportPath := filepath.Join(exportDir, tmpl.Name+".tmpl")

		// Check if file exists
		_, err := os.Stat(exportPath)
		suite.NoError(err, "Exported template file should exist: %s", exportPath)

		// Verify content
		content, err := os.ReadFile(exportPath)
		suite.NoError(err)

		originalContent, err := suite.store.GetTemplate(tmpl.Name)
		suite.NoError(err)
		suite.Equal(originalContent, string(content))
	}
}

// TestImportTemplates tests template import functionality.
func (suite *TemplateStoreTestSuite) TestImportTemplates() {
	importDir, err := os.MkdirTemp("", "template-import-test-*")
	suite.NoError(err)
	defer os.RemoveAll(importDir)

	// Create templates to import
	importTemplates := map[string]string{
		"import_test1.tmpl":  "Imported content 1: {{.Value1}}",
		"import_test2.tmpl":  "Imported content 2: {{.Value2}}",
		"nested/import.tmpl": "Nested imported content: {{.Nested}}",
	}

	for name, content := range importTemplates {
		path := filepath.Join(importDir, name)
		dir := filepath.Dir(path)
		err := os.MkdirAll(dir, 0755)
		suite.NoError(err)

		err = os.WriteFile(path, []byte(content), 0644)
		suite.NoError(err)
	}

	// Import templates
	err = suite.store.ImportTemplates(importDir)
	suite.NoError(err)

	// Verify templates were imported
	for name, expectedContent := range importTemplates {
		templateName := strings.TrimSuffix(name, ".tmpl")
		content, err := suite.store.GetTemplate(templateName)
		suite.NoError(err)
		suite.Equal(expectedContent, content)
	}
}

// TestHealthCheck tests database health check functionality.
func (suite *TemplateStoreTestSuite) TestHealthCheck() {
	// Create a simple repository for health check testing
	tempDir, err := os.MkdirTemp("", "health-check-test-*")
	suite.NoError(err)
	defer os.RemoveAll(tempDir)

	// This test is conceptual since our Store doesn't have a direct database connection
	// In a real MongoDB repository, this would test actual database connectivity

	// For now, we test that the store is responsive
	_, err = suite.store.GetTemplateInfo("simple")
	suite.NoError(err) // This indicates the store is healthy
}

// TestConcurrentAccess tests concurrent access to the store.
func (suite *TemplateStoreTestSuite) TestConcurrentAccess() {
	const numGoroutines = 10
	const numOperations = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	// Start multiple goroutines that perform various store operations concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				templateName := fmt.Sprintf("concurrent_%d_%d", id, j)
				content := fmt.Sprintf("Content for goroutine %d, operation %d: {{.Value}}", id, j)

				// Set template
				if err := suite.store.SetTemplate(templateName, content); err != nil {
					errors <- err
					continue
				}

				// Get template
				retrievedContent, err := suite.store.GetTemplate(templateName)
				if err != nil {
					errors <- err
					continue
				}

				if retrievedContent != content {
					errors <- fmt.Errorf("content mismatch: expected %s, got %s", content, retrievedContent)
					continue
				}

				// Get template info
				_, err = suite.store.GetTemplateInfo(templateName)
				if err != nil {
					errors <- err
					continue
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		suite.Fail("Concurrent access error: %v", err)
	}

	// Verify that all templates were created
	templates, err := suite.store.ListTemplates()
	suite.NoError(err)
	suite.GreaterOrEqual(len(templates), numGoroutines*numOperations)
}

// TestPathHandling tests various path handling scenarios.
func (suite *TemplateStoreTestSuite) TestPathHandling() {
	tests := []struct {
		name         string
		templateName string
		expectedPath string
	}{
		{
			name:         "Simple template name",
			templateName: "simple",
			expectedPath: "simple.tmpl",
		},
		{
			name:         "Nested template name",
			templateName: "nested/template",
			expectedPath: "nested/template.tmpl",
		},
		{
			name:         "Deep nested template name",
			templateName: "deep/nested/path/template",
			expectedPath: "deep/nested/path/template.tmpl",
		},
		{
			name:         "Template name with extension",
			templateName: "template.tmpl",
			expectedPath: "template.tmpl",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			path := suite.store.getTemplatePath(tt.templateName)
			expectedFullPath := filepath.Join(suite.tempDir, tt.expectedPath)
			suite.Equal(expectedFullPath, path)
		})
	}
}

// TestEmbeddedTemplateHandling tests embedded template functionality.
func (suite *TemplateStoreTestSuite) TestEmbeddedTemplateHandling() {
	embeddedStore := NewStoreWithEmbedded(embeddedTemplates)

	// Test getting embedded template path
	path := embeddedStore.getEmbeddedTemplatePath("auth/jwt/types")
	suite.Equal("templates/auth/jwt/types.tmpl", path)

	path = embeddedStore.getEmbeddedTemplatePath("service/main.go")
	suite.Equal("templates/service/main.go.tmpl", path)

	// Test with extension already present
	path = embeddedStore.getEmbeddedTemplatePath("service.tmpl")
	suite.Equal("templates/service.tmpl", path)
}

// TestHashCalculation tests hash calculation functionality.
func (suite *TemplateStoreTestSuite) TestHashCalculation() {
	content1 := "Test content 1"
	content2 := "Test content 2"

	hash1 := suite.store.calculateHash(content1)
	hash2 := suite.store.calculateHash(content2)
	hash1Copy := suite.store.calculateHash(content1)

	// Hashes should be deterministic
	suite.Equal(hash1, hash1Copy)

	// Different content should have different hashes
	suite.NotEqual(hash1, hash2)

	// Hashes should be non-empty hex strings
	suite.NotEmpty(hash1)
	suite.NotEmpty(hash2)
	suite.Len(hash1, 32) // MD5 hash is 32 hex characters
	suite.Len(hash2, 32)
}

// TestAccessTracking tests access count and time tracking.
func (suite *TemplateStoreTestSuite) TestAccessTracking() {
	templateName := "access_test"
	content := "Content for access tracking test"

	// Set template
	err := suite.store.SetTemplate(templateName, content)
	suite.NoError(err)

	// Initial access count should be 0
	info, err := suite.store.GetTemplateInfo(templateName)
	suite.NoError(err)
	suite.Equal(int64(0), info.AccessCount)

	// Access template multiple times
	for i := 0; i < 5; i++ {
		_, err := suite.store.GetTemplate(templateName)
		suite.NoError(err)

		// Small delay to see time changes
		time.Sleep(time.Millisecond)
	}

	// Check access count increased
	info, err = suite.store.GetTemplateInfo(templateName)
	suite.NoError(err)
	suite.Equal(int64(5), info.AccessCount)
	suite.True(info.AccessTime.After(info.ModTime))
}

// Run the test suite.
func TestTemplateStoreTestSuite(t *testing.T) {
	suite.Run(t, new(TemplateStoreTestSuite))
}

// TestStoreBasics tests basic store functionality without the test suite.
func TestStoreBasics(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "store-basic-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test template file
	tmplContent := "Basic template: {{.Value}}"
	tmplPath := filepath.Join(tempDir, "basic.tmpl")
	err = os.WriteFile(tmplPath, []byte(tmplContent), 0644)
	require.NoError(t, err)

	// Test store creation
	store := NewStore(tempDir)
	assert.NotNil(t, store)

	// Test getting template
	content, err := store.GetTemplate("basic")
	assert.NoError(t, err)
	assert.Equal(t, tmplContent, content)

	// Test setting template
	newContent := "Updated template: {{.NewValue}}"
	err = store.SetTemplate("basic", newContent)
	assert.NoError(t, err)

	// Verify updated content
	retrievedContent, err := store.GetTemplate("basic")
	assert.NoError(t, err)
	assert.Equal(t, newContent, retrievedContent)

	// Test template info
	info, err := store.GetTemplateInfo("basic")
	assert.NoError(t, err)
	assert.Equal(t, "basic", info.Name)
	assert.Greater(t, info.Size, int64(0))
	assert.NotEmpty(t, info.Hash)
}

// TestStoreErrorScenarios tests various error scenarios.
func TestStoreErrorScenarios(t *testing.T) {
	// Test with non-existent directory
	store := NewStore("/non/existent/directory")

	_, err := store.GetTemplate("any_template")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template any_template not found")

	// Test removing non-existent template
	err = store.RemoveTemplate("non_existent")
	assert.NoError(t, err) // Should not error

	// Test getting info for non-existent template
	_, err = store.GetTemplateInfo("non_existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template non_existent not found")
}

// TestStoreCleanup tests cleanup operations.
func TestStoreCleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "store-cleanup-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)

	// Add templates
	for i := 0; i < 5; i++ {
		templateName := fmt.Sprintf("cleanup_test_%d", i)
		content := fmt.Sprintf("Content %d", i)
		err := store.SetTemplate(templateName, content)
		require.NoError(t, err)
	}

	// Check initial cache state
	stats := store.GetCacheStats()
	assert.Equal(t, 5, stats["cached_templates"])

	// Clear cache
	store.ClearCache()

	// Verify cache is cleared
	stats = store.GetCacheStats()
	assert.Equal(t, 0, stats["cached_templates"])
	assert.Equal(t, int64(0), stats["total_size"].(int64))
	assert.Equal(t, int64(0), stats["total_access"].(int64))
}
