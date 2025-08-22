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

package migration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/internal/switctl/testutil"
)

func TestNewProjectMigrator(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	assert.NotNil(t, migrator)
	assert.Equal(t, workDir, migrator.workDir)
	assert.Equal(t, logger, migrator.logger)
	assert.Equal(t, configManager, migrator.configManager)
	assert.Equal(t, fileSystem, migrator.fileSystem)
	assert.NotEmpty(t, migrator.migrations)
	assert.NotEmpty(t, migrator.bestPractices)
	assert.Empty(t, migrator.executedMigrations)
	assert.Empty(t, migrator.rollbackStack)
	assert.False(t, migrator.dryRun)
	assert.True(t, migrator.backupEnabled)
	assert.Equal(t, filepath.Join(workDir, ".switctl", "backups"), migrator.backupDir)
}

func TestProjectMigrator_ExecuteMigration_NotFound(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	err := migrator.ExecuteMigration("non-existent-migration")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migration not found")
}

func TestProjectMigrator_ExecuteMigration_Success(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)
	migrator.SetBackupEnabled(false) // Disable backup for easier testing

	// Mock file system operations
	fileSystem.On("Exists", "go.mod").Return(true)
	fileSystem.On("ReadFile", "go.mod").Return([]byte("module test\nrequire github.com/innovationmech/swit v0.0.1"), nil)
	fileSystem.On("WriteFile", "go.mod", []byte("module test\nrequire github.com/innovationmech/swit v0.0.2"), 0644).Return(nil)

	// Execute built-in migration
	err := migrator.ExecuteMigration("001_update_go_mod")
	assert.NoError(t, err)

	// Verify migration was marked as executed
	executed := migrator.GetExecutedMigrations()
	assert.Contains(t, executed, "001_update_go_mod")
}

func TestProjectMigrator_ExecuteMigrations_NoPending(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	// Mark all migrations as executed
	for _, migration := range migrator.migrations {
		migrator.executedMigrations[migration.ID] = time.Now()
	}

	err := migrator.ExecuteMigrations()
	assert.NoError(t, err)
}

func TestProjectMigrator_ExecuteMigrations_WithPending(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)
	migrator.SetBackupEnabled(false) // Disable backup for easier testing

	// Mock file system operations for all migrations
	fileSystem.On("Exists", "go.mod").Return(true)
	fileSystem.On("ReadFile", "go.mod").Return([]byte("module test\nrequire github.com/innovationmech/swit v0.0.1"), nil)
	fileSystem.On("WriteFile", "go.mod", []byte("module test\nrequire github.com/innovationmech/swit v0.0.2"), 0644).Return(nil)
	fileSystem.On("MkdirAll", ".switctl", 0755).Return(nil)
	fileSystem.On("Exists", ".switctl").Return(true)

	err := migrator.ExecuteMigrations()
	assert.NoError(t, err)

	// Verify all migrations were executed
	executed := migrator.GetExecutedMigrations()
	assert.Len(t, executed, len(migrator.migrations))
}

func TestProjectMigrator_RollbackMigration_NotFound(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	err := migrator.RollbackMigration("non-existent-migration")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migration not found")
}

func TestProjectMigrator_RollbackMigration_NotReversible(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	// Add a non-reversible migration
	migration := Migration{
		ID:         "test-migration",
		Name:       "Test Migration",
		Reversible: false,
	}
	migrator.migrations = append(migrator.migrations, migration)

	err := migrator.RollbackMigration("test-migration")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not reversible")
}

func TestProjectMigrator_RollbackLastMigration_NoMigrations(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	err := migrator.RollbackLastMigration()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no migrations to rollback")
}

func TestProjectMigrator_RollbackLastMigration_Success(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	// Add a rollback entry
	rollbackEntry := RollbackEntry{
		MigrationID: "test-migration",
		Operations: []MigrationOperation{
			{
				Type:   OperationTypeFileDelete,
				Target: "test-file",
			},
		},
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
	migrator.rollbackStack = append(migrator.rollbackStack, rollbackEntry)
	migrator.executedMigrations["test-migration"] = time.Now()

	// Mock file system operations
	fileSystem.On("Exists", "test-file").Return(true)
	fileSystem.On("Remove", "test-file").Return(nil)

	err := migrator.RollbackLastMigration()
	assert.NoError(t, err)

	// Verify rollback stack is empty and migration is not executed
	assert.Empty(t, migrator.rollbackStack)
	assert.NotContains(t, migrator.executedMigrations, "test-migration")
}

func TestProjectMigrator_ApplyBestPractices_None(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	// Mock fileSystem.Exists to return false for all pattern checks
	fileSystem.On("Exists", "*.go").Return(false)

	// Override the isComponentInUse method behavior for testing
	// We'll add a simple migration that marks no components as in use
	migrator.bestPractices = []BestPractice{} // Clear best practices for this test

	err := migrator.ApplyBestPractices()
	assert.NoError(t, err)
}

func TestProjectMigrator_ValidateProject_Success(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	// Mock all validation requirements
	fileSystem.On("Exists", "go.mod").Return(true)
	fileSystem.On("ReadFile", "go.mod").Return([]byte("module test\nrequire github.com/innovationmech/swit v0.0.2"), nil)
	fileSystem.On("Exists", ".switctl").Return(true)

	err := migrator.ValidateProject()
	assert.NoError(t, err)
}

func TestProjectMigrator_ValidateProject_CriticalFailure(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	// Mock validation failure
	fileSystem.On("Exists", "go.mod").Return(true)                                  // File exists
	fileSystem.On("ReadFile", "go.mod").Return([]byte("module test\ngo 1.19"), nil) // File content without expected version

	err := migrator.ValidateProject()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "critical validation failed")
}

func TestProjectMigrator_GetPendingMigrations(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	pending := migrator.GetPendingMigrations()
	assert.NotEmpty(t, pending)

	// All built-in migrations should be pending initially
	assert.Len(t, pending, len(migrator.migrations))
}

func TestProjectMigrator_GetExecutedMigrations(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	// Initially should be empty
	executed := migrator.GetExecutedMigrations()
	assert.Empty(t, executed)

	// Add an executed migration
	migrator.executedMigrations["test-migration"] = time.Now()

	executed = migrator.GetExecutedMigrations()
	assert.Len(t, executed, 1)
	assert.Contains(t, executed, "test-migration")
}

func TestProjectMigrator_GetApplicableBestPractices(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	// Mock file system to make practices applicable
	fileSystem.On("Exists", "*.go").Return(true)

	practices := migrator.GetApplicableBestPractices()
	assert.NotEmpty(t, practices)
}

func TestProjectMigrator_SetDryRun(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	assert.False(t, migrator.dryRun)

	migrator.SetDryRun(true)
	assert.True(t, migrator.dryRun)

	migrator.SetDryRun(false)
	assert.False(t, migrator.dryRun)
}

func TestProjectMigrator_SetBackupEnabled(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	assert.True(t, migrator.backupEnabled)

	migrator.SetBackupEnabled(false)
	assert.False(t, migrator.backupEnabled)

	migrator.SetBackupEnabled(true)
	assert.True(t, migrator.backupEnabled)
}

func TestProjectMigrator_AddMigration_Valid(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	initialCount := len(migrator.migrations)

	migration := Migration{
		ID:   "custom-migration",
		Name: "Custom Migration",
		Operations: []MigrationOperation{
			{
				Type:   OperationTypeFileCreate,
				Target: "test-file",
			},
		},
	}

	err := migrator.AddMigration(migration)
	assert.NoError(t, err)
	assert.Len(t, migrator.migrations, initialCount+1)
}

func TestProjectMigrator_AddMigration_Invalid(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	// Missing ID
	migration := Migration{
		Name: "Invalid Migration",
	}

	err := migrator.AddMigration(migration)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid migration")
}

func TestProjectMigrator_AddBestPractice(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	initialCount := len(migrator.bestPractices)

	practice := BestPractice{
		ID:          "custom-practice",
		Name:        "Custom Practice",
		Description: "A custom best practice",
		Category:    CategorySecurity,
		Impact:      ImpactLevelLow,
	}

	err := migrator.AddBestPractice(practice)
	assert.NoError(t, err)
	assert.Len(t, migrator.bestPractices, initialCount+1)
}

func TestProjectMigrator_ExecuteFileOperations(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	tests := []struct {
		name      string
		operation MigrationOperation
		setupMock func()
	}{
		{
			name: "file create",
			operation: MigrationOperation{
				Type:   OperationTypeFileCreate,
				Target: "new-file.txt",
				Parameters: map[string]interface{}{
					"content": "Hello, World!",
				},
			},
			setupMock: func() {
				fileSystem.On("Exists", "new-file.txt").Return(false)
				fileSystem.On("WriteFile", "new-file.txt", []byte("Hello, World!"), 0644).Return(nil)
			},
		},
		{
			name: "file update",
			operation: MigrationOperation{
				Type:   OperationTypeFileUpdate,
				Target: "existing-file.txt",
				Parameters: map[string]interface{}{
					"search":  "old",
					"replace": "new",
				},
			},
			setupMock: func() {
				fileSystem.On("Exists", "existing-file.txt").Return(true)
				fileSystem.On("ReadFile", "existing-file.txt").Return([]byte("old content"), nil)
				fileSystem.On("WriteFile", "existing-file.txt", []byte("new content"), 0644).Return(nil)
			},
		},
		{
			name: "file delete",
			operation: MigrationOperation{
				Type:   OperationTypeFileDelete,
				Target: "old-file.txt",
			},
			setupMock: func() {
				fileSystem.On("Exists", "old-file.txt").Return(true)
				fileSystem.On("Remove", "old-file.txt").Return(nil)
			},
		},
		{
			name: "directory create",
			operation: MigrationOperation{
				Type:   OperationTypeDirectoryCreate,
				Target: "new-dir",
			},
			setupMock: func() {
				fileSystem.On("MkdirAll", "new-dir", 0755).Return(nil)
			},
		},
		{
			name: "directory delete",
			operation: MigrationOperation{
				Type:   OperationTypeDirectoryDelete,
				Target: "old-dir",
			},
			setupMock: func() {
				fileSystem.On("Exists", "old-dir").Return(true)
				fileSystem.On("Remove", "old-dir").Return(nil)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Reset mock
			fileSystem.ExpectedCalls = nil
			test.setupMock()

			err := migrator.executeOperation(&test.operation)
			assert.NoError(t, err)

			fileSystem.AssertExpectations(t)
		})
	}
}

func TestProjectMigrator_ExecuteConfigOperation(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	operation := MigrationOperation{
		Type:   OperationTypeConfigUpdate,
		Target: "test.key",
		Parameters: map[string]interface{}{
			"value": "test.value",
		},
	}

	configManager.On("Set", "test.key", "test.value").Return()

	err := migrator.executeOperation(&operation)
	assert.NoError(t, err)

	configManager.AssertExpectations(t)
}

func TestProjectMigrator_ExecuteReplaceOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation MigrationOperation
		content   string
		expected  string
	}{
		{
			name: "simple replace",
			operation: MigrationOperation{
				Type:   OperationTypeReplace,
				Target: "test.txt",
				Parameters: map[string]interface{}{
					"search":  "old",
					"replace": "new",
				},
			},
			content:  "old content",
			expected: "new content",
		},
		{
			name: "regex replace",
			operation: MigrationOperation{
				Type:   OperationTypeReplace,
				Target: "test.txt",
				Parameters: map[string]interface{}{
					"search":  "v\\d+\\.\\d+\\.\\d+",
					"replace": "v2.0.0",
					"regex":   true,
				},
			},
			content:  "version v1.2.3",
			expected: "version v2.0.0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create fresh instances for each subtest
			workDir := t.TempDir()
			logger := testutil.NewNoOpLogger()
			configManager := testutil.NewMockConfigManager()
			fileSystem := testutil.NewMockFileSystem()

			migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

			// Setup mock expectations
			fileSystem.On("ReadFile", "test.txt").Return([]byte(test.content), nil)
			fileSystem.On("WriteFile", "test.txt", []byte(test.expected), 0644).Return(nil)

			err := migrator.executeOperation(&test.operation)
			assert.NoError(t, err)

			fileSystem.AssertExpectations(t)
		})
	}
}

func TestProjectMigrator_ExecuteInsertOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation MigrationOperation
		content   string
		expected  string
	}{
		{
			name: "insert at start",
			operation: MigrationOperation{
				Type:   OperationTypeInsert,
				Target: "test.txt",
				Parameters: map[string]interface{}{
					"content":  "new line\n",
					"position": "start",
				},
			},
			content:  "existing content",
			expected: "new line\nexisting content",
		},
		{
			name: "insert at end",
			operation: MigrationOperation{
				Type:   OperationTypeInsert,
				Target: "test.txt",
				Parameters: map[string]interface{}{
					"content":  "\nnew line",
					"position": "end",
				},
			},
			content:  "existing content",
			expected: "existing content\nnew line",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create fresh instances for each subtest
			workDir := t.TempDir()
			logger := testutil.NewNoOpLogger()
			configManager := testutil.NewMockConfigManager()
			fileSystem := testutil.NewMockFileSystem()

			migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

			// Setup mock expectations
			fileSystem.On("ReadFile", "test.txt").Return([]byte(test.content), nil)
			fileSystem.On("WriteFile", "test.txt", []byte(test.expected), 0644).Return(nil)

			err := migrator.executeOperation(&test.operation)
			assert.NoError(t, err)

			fileSystem.AssertExpectations(t)
		})
	}
}

func TestProjectMigrator_ExecuteRemoveOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation MigrationOperation
		content   string
		expected  string
	}{
		{
			name: "simple remove",
			operation: MigrationOperation{
				Type:   OperationTypeRemove,
				Target: "test.txt",
				Parameters: map[string]interface{}{
					"pattern": "remove me",
				},
			},
			content:  "keep this remove me keep this",
			expected: "keep this  keep this",
		},
		{
			name: "regex remove",
			operation: MigrationOperation{
				Type:   OperationTypeRemove,
				Target: "test.txt",
				Parameters: map[string]interface{}{
					"pattern": "v\\d+\\.\\d+\\.\\d+",
					"regex":   true,
				},
			},
			content:  "version v1.2.3 and v4.5.6",
			expected: "version  and ",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create fresh instances for each subtest
			workDir := t.TempDir()
			logger := testutil.NewNoOpLogger()
			configManager := testutil.NewMockConfigManager()
			fileSystem := testutil.NewMockFileSystem()

			migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

			// Setup mock expectations
			fileSystem.On("ReadFile", "test.txt").Return([]byte(test.content), nil)
			fileSystem.On("WriteFile", "test.txt", []byte(test.expected), 0644).Return(nil)

			err := migrator.executeOperation(&test.operation)
			assert.NoError(t, err)

			fileSystem.AssertExpectations(t)
		})
	}
}

func TestProjectMigrator_ExecuteValidation(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	tests := []struct {
		name       string
		validation ValidationCheck
		setupMock  func()
		expectErr  bool
	}{
		{
			name: "file exists - success",
			validation: ValidationCheck{
				Type:   ValidationTypeFileExists,
				Target: "existing-file.txt",
			},
			setupMock: func() {
				fileSystem.On("Exists", "existing-file.txt").Return(true)
			},
			expectErr: false,
		},
		{
			name: "file exists - failure",
			validation: ValidationCheck{
				Type:   ValidationTypeFileExists,
				Target: "missing-file.txt",
			},
			setupMock: func() {
				fileSystem.On("Exists", "missing-file.txt").Return(false)
			},
			expectErr: true,
		},
		{
			name: "file content - success",
			validation: ValidationCheck{
				Type:     ValidationTypeFileContent,
				Target:   "test-file.txt",
				Expected: "expected content",
			},
			setupMock: func() {
				fileSystem.On("ReadFile", "test-file.txt").Return([]byte("file has expected content here"), nil)
			},
			expectErr: false,
		},
		{
			name: "config value - success",
			validation: ValidationCheck{
				Type:     ValidationTypeConfigValue,
				Target:   "test.key",
				Expected: "test.value",
			},
			setupMock: func() {
				configManager.On("Get", "test.key").Return("test.value")
			},
			expectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Reset mocks
			fileSystem.ExpectedCalls = nil
			configManager.ExpectedCalls = nil
			test.setupMock()

			err := migrator.executeValidation(&test.validation)
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			fileSystem.AssertExpectations(t)
			configManager.AssertExpectations(t)
		})
	}
}

func TestProjectMigrator_EvaluateCondition(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	tests := []struct {
		name      string
		condition MigrationCondition
		setupMock func()
		expected  bool
	}{
		{
			name: "file exists - true",
			condition: MigrationCondition{
				Type:   ConditionTypeFileExists,
				Target: "existing-file.txt",
			},
			setupMock: func() {
				fileSystem.On("Exists", "existing-file.txt").Return(true)
			},
			expected: true,
		},
		{
			name: "file exists - false",
			condition: MigrationCondition{
				Type:   ConditionTypeFileExists,
				Target: "missing-file.txt",
			},
			setupMock: func() {
				fileSystem.On("Exists", "missing-file.txt").Return(false)
			},
			expected: false,
		},
		{
			name: "config value equals",
			condition: MigrationCondition{
				Type:     ConditionTypeConfigValue,
				Target:   "test.key",
				Operator: "==",
				Value:    "expected",
			},
			setupMock: func() {
				configManager.On("Get", "test.key").Return("expected")
			},
			expected: true,
		},
		{
			name: "content contains",
			condition: MigrationCondition{
				Type:   ConditionTypeContent,
				Target: "test.txt",
				Value:  "search text",
			},
			setupMock: func() {
				fileSystem.On("ReadFile", "test.txt").Return([]byte("file contains search text here"), nil)
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Reset mocks
			fileSystem.ExpectedCalls = nil
			configManager.ExpectedCalls = nil
			test.setupMock()

			result := migrator.evaluateCondition(&test.condition)
			assert.Equal(t, test.expected, result)

			fileSystem.AssertExpectations(t)
			configManager.AssertExpectations(t)
		})
	}
}

func TestProjectMigrator_DryRunMode(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)
	migrator.SetDryRun(true)

	// Mock only the condition checks, not the actual operations
	fileSystem.On("Exists", "go.mod").Return(true)
	fileSystem.On("ReadFile", "go.mod").Return([]byte("module test\ngo 1.19\ngithub.com/innovationmech/swit v0.0.2"), nil)

	// Execute migration in dry run mode
	err := migrator.ExecuteMigration("001_update_go_mod")
	assert.NoError(t, err)

	// Verify migration was NOT marked as executed in dry run mode
	executed := migrator.GetExecutedMigrations()
	assert.NotContains(t, executed, "001_update_go_mod")
}

func TestProjectMigrator_FailureHandling(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	// Create a migration with failure handling
	migration := Migration{
		ID:   "test-failure-migration",
		Name: "Test Failure Migration",
		Operations: []MigrationOperation{
			{
				Type:      OperationTypeFileCreate,
				Target:    "test-file",
				OnFailure: FailureActionStop,
			},
		},
		Reversible: true,
	}

	// Add the migration
	migrator.migrations = append(migrator.migrations, migration)

	// Mock file system to fail
	fileSystem.On("MkdirAll", mock.AnythingOfType("string"), 0755).Return(nil)                      // Allow backup directory creation
	fileSystem.On("Copy", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil) // Allow backup creation
	fileSystem.On("Exists", "test-file").Return(true)                                               // File already exists, will fail create

	err := migrator.ExecuteMigration("test-failure-migration")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file already exists")
}

func TestProjectMigrator_CompareValues(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	tests := []struct {
		actual   interface{}
		expected interface{}
		operator string
		result   bool
	}{
		{"hello", "hello", "==", true},
		{"hello", "world", "==", false},
		{"hello", "world", "!=", true},
		{"hello world", "world", "contains", true},
		{"hello", "world", "contains", false},
	}

	for _, test := range tests {
		result := migrator.compareValues(test.actual, test.expected, test.operator)
		assert.Equal(t, test.result, result,
			"compareValues(%v, %v, %s) should return %v",
			test.actual, test.expected, test.operator, test.result)
	}
}

func TestProjectMigrator_GenerateRollbackOperation(t *testing.T) {
	workDir := t.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	tests := []struct {
		name      string
		operation MigrationOperation
		expected  *MigrationOperation
	}{
		{
			name: "file create rollback",
			operation: MigrationOperation{
				Type:   OperationTypeFileCreate,
				Target: "new-file.txt",
			},
			expected: &MigrationOperation{
				Type:   OperationTypeFileDelete,
				Target: "new-file.txt",
			},
		},
		{
			name: "file delete rollback",
			operation: MigrationOperation{
				Type:   OperationTypeFileDelete,
				Target: "deleted-file.txt",
			},
			expected: nil, // Cannot rollback delete without storing content
		},
		{
			name: "file move rollback",
			operation: MigrationOperation{
				Type:   OperationTypeFileMove,
				Target: "source.txt",
				Parameters: map[string]interface{}{
					"destination": "dest.txt",
				},
			},
			expected: &MigrationOperation{
				Type:   OperationTypeFileMove,
				Target: "dest.txt",
				Parameters: map[string]interface{}{
					"destination": "source.txt",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := migrator.generateRollbackOperation(&test.operation)
			if test.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, test.expected.Type, result.Type)
				assert.Equal(t, test.expected.Target, result.Target)
				if test.expected.Parameters != nil {
					assert.Equal(t, test.expected.Parameters, result.Parameters)
				}
			}
		})
	}
}

func TestMigrationType_Values(t *testing.T) {
	types := []MigrationType{
		MigrationTypeStructural,
		MigrationTypeCode,
		MigrationTypeConfig,
		MigrationTypeDependency,
		MigrationTypeBestPractice,
	}

	for _, migrationType := range types {
		assert.NotEmpty(t, string(migrationType))
	}
}

func TestOperationType_Values(t *testing.T) {
	types := []OperationType{
		OperationTypeFileCreate,
		OperationTypeFileUpdate,
		OperationTypeFileDelete,
		OperationTypeFileMove,
		OperationTypeFileCopy,
		OperationTypeDirectoryCreate,
		OperationTypeDirectoryDelete,
		OperationTypeConfigUpdate,
		OperationTypeCommand,
		OperationTypeTemplate,
		OperationTypeReplace,
		OperationTypeInsert,
		OperationTypeRemove,
	}

	for _, operationType := range types {
		assert.NotEmpty(t, string(operationType))
	}
}

func TestBestPracticeCategory_Values(t *testing.T) {
	categories := []BestPracticeCategory{
		CategorySecurity,
		CategoryPerformance,
		CategoryMaintainability,
		CategoryTesting,
		CategoryDocumentation,
		CategoryStructure,
		CategoryDependencies,
	}

	for _, category := range categories {
		assert.NotEmpty(t, string(category))
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "destination.txt")

	// Create source file
	content := "Hello, World!"
	require.NoError(t, os.WriteFile(srcPath, []byte(content), 0644))

	// Test copy
	err := CopyFile(srcPath, dstPath)
	assert.NoError(t, err)

	// Verify destination file
	dstContent, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(dstContent))
}

func TestCopyFile_SourceNotFound(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "nonexistent.txt")
	dstPath := filepath.Join(tempDir, "destination.txt")

	err := CopyFile(srcPath, dstPath)
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkProjectMigrator_ExecuteMigration(b *testing.B) {
	workDir := b.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)
	migrator.SetDryRun(true) // Use dry run to avoid actual file operations

	// Mock file system operations
	fileSystem.On("Exists", "go.mod").Return(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		migrator.ExecuteMigration("001_update_go_mod")
	}
}

func BenchmarkProjectMigrator_EvaluateCondition(b *testing.B) {
	workDir := b.TempDir()
	logger := testutil.NewNoOpLogger()
	configManager := testutil.NewMockConfigManager()
	fileSystem := testutil.NewMockFileSystem()

	migrator := NewProjectMigrator(workDir, logger, configManager, fileSystem)

	condition := MigrationCondition{
		Type:   ConditionTypeFileExists,
		Target: "test-file",
	}

	fileSystem.On("Exists", "test-file").Return(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		migrator.evaluateCondition(&condition)
	}
}
