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

package version

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/internal/switctl/testutil"
)

// setupMockConfigManager sets up standard mock expectations for config manager
func setupMockConfigManager(configManager *testutil.MockConfigManager) {
	configManager.On("GetString", "framework.version").Return("")
}

func TestNewCompatibilityManager(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	assert.NotNil(t, cm)
	assert.Equal(t, workDir, cm.workDir)
	assert.Equal(t, logger, cm.logger)
	assert.Equal(t, configManager, cm.configManager)
	assert.NotNil(t, cm.versionCache) // Cache is initialized but may be empty
	assert.NotEmpty(t, cm.compatibilityMatrix.Versions)
	assert.NotEmpty(t, cm.migrationRules)
	assert.NotEmpty(t, cm.deprecationWarnings)
	assert.Equal(t, 24*time.Hour, cm.updateCheckInterval)
}

func TestCompatibilityManager_CheckCompatibility(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Test compatibility check
	result := cm.CheckCompatibility()

	assert.NotNil(t, result)
	assert.NotEmpty(t, result.CurrentVersion)
	assert.NotEmpty(t, result.LatestVersion)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestCompatibilityManager_CheckCompatibility_WithEOLVersion(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Add an EOL version
	eolDate := time.Now().Add(-24 * time.Hour) // Yesterday
	eolVersion := &VersionInfo{
		Version:     "0.0.0",
		Major:       0,
		Minor:       0,
		Patch:       0,
		ReleaseDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndOfLife:   &eolDate,
		Stability:   StabilityEOL,
	}
	cm.compatibilityMatrix.Versions["0.0.0"] = eolVersion
	cm.currentVersion = "0.0.0"

	result := cm.CheckCompatibility()

	assert.False(t, result.Compatible)
	assert.True(t, result.MigrationNeeded)
	assert.NotEmpty(t, result.Issues)

	// Check for EOL issue
	found := false
	for _, issue := range result.Issues {
		if issue.Type == "version" && issue.Breaking {
			found = true
			break
		}
	}
	assert.True(t, found, "Should detect EOL version")
}

func TestCompatibilityManager_CheckCompatibility_WithBreakingChanges(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Add breaking changes
	breakingChange := BreakingChange{
		Version:     "0.0.2",
		Component:   "test-component",
		Change:      "API signature changed",
		Impact:      "High",
		Mitigation:  "Update code",
		AutoMigrate: false,
		Required:    true,
	}
	cm.compatibilityMatrix.Breakingchanges["0.0.2"] = []BreakingChange{breakingChange}
	cm.currentVersion = "0.0.1"

	result := cm.CheckCompatibility()

	assert.False(t, result.Compatible)
	assert.True(t, result.MigrationNeeded)

	// Check for breaking change issue
	found := false
	for _, issue := range result.Issues {
		if issue.Type == "breaking_change" && issue.Breaking {
			found = true
			break
		}
	}
	assert.True(t, found, "Should detect breaking changes")
}

func TestCompatibilityManager_GetCurrentVersion(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	version, err := cm.GetCurrentVersion()
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Equal(t, cm.currentVersion, version)
}

func TestCompatibilityManager_GetLatestVersion(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	version, err := cm.GetLatestVersion()
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
}

func TestCompatibilityManager_MigrateProject_SameVersion(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)
	currentVersion := cm.currentVersion

	// Migrate to same version should succeed without doing anything
	err := cm.MigrateProject(currentVersion)
	assert.NoError(t, err)
}

func TestCompatibilityManager_MigrateProject_NoCurrentVersion(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)
	cm.currentVersion = "" // Clear current version

	err := cm.MigrateProject("0.0.2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "current version not detected")
}

func TestCompatibilityManager_MigrateProject_NoMigrationPath(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)
	cm.currentVersion = "0.0.1"

	// Try to migrate to a version with no migration path
	err := cm.MigrateProject("999.999.999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find migration path")
}

func TestCompatibilityManager_GetMigrationPath(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Test getting migration path that doesn't exist
	path, err := cm.GetMigrationPath("0.0.1", "999.999.999")
	assert.Error(t, err)
	assert.Nil(t, path)
	assert.Contains(t, err.Error(), "no migration path found")
}

func TestCompatibilityManager_GetVersionInfo(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Test getting existing version info
	info, err := cm.GetVersionInfo("0.0.1")
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "0.0.1", info.Version)

	// Test getting non-existent version info
	info, err = cm.GetVersionInfo("999.999.999")
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "version 999.999.999 not found")
}

func TestCompatibilityManager_ListAvailableVersions(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	versions := cm.ListAvailableVersions()
	assert.NotEmpty(t, versions)
	assert.Contains(t, versions, "0.0.1")
	assert.Contains(t, versions, "0.0.2")

	// Versions should be sorted
	for i := 1; i < len(versions); i++ {
		assert.True(t, cm.compareVersions(versions[i-1], versions[i]) <= 0)
	}
}

func TestCompatibilityManager_GetDeprecationWarnings(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	warnings := cm.GetDeprecationWarnings()
	assert.NotEmpty(t, warnings)

	// All warnings should be applicable to current version
	for _, warning := range warnings {
		assert.True(t, cm.isComponentInUse(warning.Component))
	}
}

func TestCompatibilityManager_DetectCurrentVersion_FromGoMod(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()

	// Create go.mod file with version
	goModContent := `module test-project

go 1.19

require (
	github.com/innovationmech/swit v0.0.1
	github.com/gin-gonic/gin v1.9.1
)
`
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "go.mod"), []byte(goModContent), 0644))

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Should detect version from go.mod
	assert.Equal(t, "0.0.1", cm.currentVersion)
}

func TestCompatibilityManager_DetectCurrentVersion_FromConfig(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()

	// Mock config manager to return version
	configManager.On("GetString", "framework.version").Return("0.0.2")

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Should detect version from config
	assert.Equal(t, "0.0.2", cm.currentVersion)
}

func TestCompatibilityManager_ExtractVersionFromGoMod(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	tests := []struct {
		name     string
		content  string
		expected string
		hasError bool
	}{
		{
			name: "standard version",
			content: `module test
require github.com/innovationmech/swit v1.2.3`,
			expected: "1.2.3",
			hasError: false,
		},
		{
			name: "version with v prefix",
			content: `module test
require github.com/innovationmech/swit v1.2.3`,
			expected: "1.2.3",
			hasError: false,
		},
		{
			name: "prerelease version",
			content: `module test
require github.com/innovationmech/swit v1.2.3-beta`,
			expected: "1.2.3-beta",
			hasError: false,
		},
		{
			name: "no framework dependency",
			content: `module test
require github.com/gin-gonic/gin v1.9.1`,
			expected: "",
			hasError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			goModPath := filepath.Join(workDir, "go.mod.test")
			require.NoError(t, os.WriteFile(goModPath, []byte(test.content), 0644))

			version, err := cm.extractVersionFromGoMod(goModPath)
			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, version)
			}
		})
	}
}

func TestCompatibilityManager_ParseVersion(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	tests := []struct {
		version  string
		expected VersionInfo
	}{
		{
			version: "1.2.3",
			expected: VersionInfo{
				Version: "1.2.3",
				Major:   1,
				Minor:   2,
				Patch:   3,
			},
		},
		{
			version: "v2.0.0",
			expected: VersionInfo{
				Version: "2.0.0",
				Major:   2,
				Minor:   0,
				Patch:   0,
			},
		},
		{
			version: "invalid",
			expected: VersionInfo{
				Version: "invalid",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			result := cm.parseVersion(test.version)
			assert.Equal(t, test.expected.Version, result.Version)
			assert.Equal(t, test.expected.Major, result.Major)
			assert.Equal(t, test.expected.Minor, result.Minor)
			assert.Equal(t, test.expected.Patch, result.Patch)
		})
	}
}

func TestCompatibilityManager_CompareVersions(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s vs %s", test.v1, test.v2), func(t *testing.T) {
			result := cm.compareVersions(test.v1, test.v2)
			if test.expected < 0 {
				assert.Less(t, result, 0)
			} else if test.expected > 0 {
				assert.Greater(t, result, 0)
			} else {
				assert.Equal(t, 0, result)
			}
		})
	}
}

func TestCompatibilityManager_IsVersionAffectedByBreakingChange(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	change := BreakingChange{
		Version: "1.0.0",
	}

	// Versions before breaking change should be affected
	assert.True(t, cm.isVersionAffectedByBreakingChange("0.9.0", change))
	assert.True(t, cm.isVersionAffectedByBreakingChange("0.9.9", change))

	// Versions at or after breaking change should not be affected
	assert.False(t, cm.isVersionAffectedByBreakingChange("1.0.0", change))
	assert.False(t, cm.isVersionAffectedByBreakingChange("1.0.1", change))
	assert.False(t, cm.isVersionAffectedByBreakingChange("2.0.0", change))
}

func TestCompatibilityManager_IsComponentInUse(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// This is a simplified implementation that always returns true
	// In a real implementation, it would check the project files
	assert.True(t, cm.isComponentInUse("any-component"))
}

func TestStabilityLevel_String(t *testing.T) {
	tests := []struct {
		level    StabilityLevel
		expected string
	}{
		{StabilityAlpha, "alpha"},
		{StabilityBeta, "beta"},
		{StabilityStable, "stable"},
		{StabilityDeprecated, "deprecated"},
		{StabilityEOL, "eol"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, string(test.level))
	}
}

func TestIssueType_Values(t *testing.T) {
	types := []IssueType{
		IssueTypeAPI,
		IssueTypeConfig,
		IssueTypeDependency,
		IssueTypeFeature,
		IssueTypePerformance,
		IssueTypeSecurity,
	}

	for _, issueType := range types {
		assert.NotEmpty(t, string(issueType))
	}
}

func TestImpactLevel_Values(t *testing.T) {
	levels := []ImpactLevel{
		ImpactLow,
		ImpactMedium,
		ImpactHigh,
		ImpactCritical,
	}

	for _, level := range levels {
		assert.NotEmpty(t, string(level))
	}
}

func TestMigrationStepType_Values(t *testing.T) {
	types := []MigrationStepType{
		MigrationStepCommand,
		MigrationStepScript,
		MigrationStepFile,
		MigrationStepConfig,
		MigrationStepValidation,
		MigrationStepManual,
	}

	for _, stepType := range types {
		assert.NotEmpty(t, string(stepType))
	}
}

func TestCompatibilityManager_GetLatestVersion_NoStableVersion(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Remove all stable versions
	for version, info := range cm.compatibilityMatrix.Versions {
		info.Stability = StabilityAlpha
		cm.compatibilityMatrix.Versions[version] = info
	}

	_, err := cm.getLatestVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no stable version found")
}

func TestCompatibilityManager_CompatibilityRules(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)
	cm.currentVersion = "0.0.1"

	// Should have compatibility rules for 0.0.1 -> 0.0.2
	rules, exists := cm.compatibilityMatrix.Compatibility["0.0.1"]
	assert.True(t, exists)
	assert.NotEmpty(t, rules)

	// Find rule for 0.0.2
	var rule *CompatibilityRule
	for i, r := range rules {
		if r.ToVersion == "0.0.2" {
			rule = &rules[i]
			break
		}
	}
	assert.NotNil(t, rule)
	assert.Equal(t, "0.0.1", rule.FromVersion)
	assert.Equal(t, "0.0.2", rule.ToVersion)
	assert.True(t, rule.Compatible)
	assert.True(t, rule.Automatic)
}

func TestCompatibilityManager_DeprecationWarnings(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	warnings := cm.GetDeprecationWarnings()
	assert.NotEmpty(t, warnings)

	// Should have at least one deprecation warning
	found := false
	for _, warning := range warnings {
		if warning.Component == "old_config_format" {
			found = true
			assert.Equal(t, "0.0.2", warning.Version)
			assert.Equal(t, SeverityWarning, warning.Severity)
			assert.True(t, warning.AutoMigrate)
			break
		}
	}
	assert.True(t, found)
}

func TestCompatibilityManager_VersionCache(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Version cache should be initialized (but may be empty initially)
	assert.NotNil(t, cm.versionCache)
}

func TestCompatibilityManager_UpdateCheckInterval(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Should have default update check interval
	assert.Equal(t, 24*time.Hour, cm.updateCheckInterval)
}

func TestCompatibilityManager_RemoteVersionSource(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	// Should have default remote version source
	assert.Equal(t, "https://api.github.com/repos/innovationmech/swit/releases", cm.remoteVersionSource)
}

// Benchmark tests
func BenchmarkCompatibilityManager_CheckCompatibility(b *testing.B) {
	workDir := b.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.CheckCompatibility()
	}
}

func BenchmarkCompatibilityManager_CompareVersions(b *testing.B) {
	workDir := b.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.compareVersions("1.0.0", "2.0.0")
	}
}

func BenchmarkCompatibilityManager_ParseVersion(b *testing.B) {
	workDir := b.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	setupMockConfigManager(configManager)

	cm := NewCompatibilityManager(workDir, logger, configManager)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.parseVersion("1.2.3")
	}
}
