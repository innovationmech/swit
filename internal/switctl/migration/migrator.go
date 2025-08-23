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

// Package migration provides project structure migration and code update capabilities for switctl.
package migration

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// ProjectMigrator handles project structure migration and code updates.
type ProjectMigrator struct {
	mu                 sync.RWMutex
	workDir            string
	logger             interfaces.Logger
	configManager      interfaces.ConfigManager
	fileSystem         interfaces.FileSystem
	templateEngine     interfaces.TemplateEngine
	migrations         []Migration
	bestPractices      []BestPractice
	executedMigrations map[string]time.Time
	rollbackStack      []RollbackEntry
	dryRun             bool
	backupEnabled      bool
	backupDir          string
}

// Migration represents a single migration operation.
type Migration struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Version      string               `json:"version"`
	Type         MigrationType        `json:"type"`
	Priority     int                  `json:"priority"`
	Dependencies []string             `json:"dependencies"`
	Conditions   []MigrationCondition `json:"conditions"`
	Operations   []MigrationOperation `json:"operations"`
	Rollback     []MigrationOperation `json:"rollback,omitempty"`
	Validation   []ValidationCheck    `json:"validation,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	ExecutedAt   *time.Time           `json:"executed_at,omitempty"`
	Reversible   bool                 `json:"reversible"`
	Metadata     map[string]string    `json:"metadata"`
}

// MigrationOperation represents a single operation within a migration.
type MigrationOperation struct {
	ID          string                 `json:"id"`
	Type        OperationType          `json:"type"`
	Target      string                 `json:"target"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
	Condition   string                 `json:"condition,omitempty"`
	OnFailure   FailureAction          `json:"on_failure"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	Retry       RetryConfig            `json:"retry,omitempty"`
}

// MigrationCondition represents a condition that must be met for migration.
type MigrationCondition struct {
	Type     ConditionType `json:"type"`
	Target   string        `json:"target"`
	Operator string        `json:"operator"`
	Value    interface{}   `json:"value"`
	Required bool          `json:"required"`
	Message  string        `json:"message,omitempty"`
}

// ValidationCheck represents a validation check for migration.
type ValidationCheck struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       ValidationType    `json:"type"`
	Target     string            `json:"target"`
	Expected   interface{}       `json:"expected"`
	Critical   bool              `json:"critical"`
	Message    string            `json:"message,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// BestPractice represents a best practice update or rule.
type BestPractice struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Category    BestPracticeCategory `json:"category"`
	Impact      ImpactLevel          `json:"impact"`
	Actions     []BestPracticeAction `json:"actions"`
	Applies     []ApplicabilityRule  `json:"applies"`
	Version     string               `json:"version"`
	Tags        []string             `json:"tags"`
	AutoApply   bool                 `json:"auto_apply"`
}

// BestPracticeAction represents an action to apply a best practice.
type BestPracticeAction struct {
	Type        ActionType             `json:"type"`
	Target      string                 `json:"target"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
	Optional    bool                   `json:"optional"`
}

// ApplicabilityRule determines when a best practice applies.
type ApplicabilityRule struct {
	Condition string      `json:"condition"`
	Value     interface{} `json:"value"`
	Operator  string      `json:"operator"`
}

// RollbackEntry represents an entry in the rollback stack.
type RollbackEntry struct {
	MigrationID string                 `json:"migration_id"`
	Operations  []MigrationOperation   `json:"operations"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     map[string]interface{} `json:"context"`
}

// RetryConfig represents retry configuration for operations.
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	Backoff     BackoffType   `json:"backoff"`
}

// Enums and constants

type MigrationType string

const (
	MigrationTypeStructural   MigrationType = "structural"
	MigrationTypeCode         MigrationType = "code"
	MigrationTypeConfig       MigrationType = "config"
	MigrationTypeDependency   MigrationType = "dependency"
	MigrationTypeBestPractice MigrationType = "best_practice"
)

type OperationType string

const (
	OperationTypeFileCreate      OperationType = "file_create"
	OperationTypeFileUpdate      OperationType = "file_update"
	OperationTypeFileDelete      OperationType = "file_delete"
	OperationTypeFileMove        OperationType = "file_move"
	OperationTypeFileCopy        OperationType = "file_copy"
	OperationTypeDirectoryCreate OperationType = "directory_create"
	OperationTypeDirectoryDelete OperationType = "directory_delete"
	OperationTypeConfigUpdate    OperationType = "config_update"
	OperationTypeCommand         OperationType = "command"
	OperationTypeTemplate        OperationType = "template"
	OperationTypeReplace         OperationType = "replace"
	OperationTypeInsert          OperationType = "insert"
	OperationTypeRemove          OperationType = "remove"
)

type ConditionType string

const (
	ConditionTypeFileExists      ConditionType = "file_exists"
	ConditionTypeDirectoryExists ConditionType = "directory_exists"
	ConditionTypeConfigValue     ConditionType = "config_value"
	ConditionTypeVersion         ConditionType = "version"
	ConditionTypeContent         ConditionType = "content"
	ConditionTypePattern         ConditionType = "pattern"
)

type ValidationType string

const (
	ValidationTypeFileExists  ValidationType = "file_exists"
	ValidationTypeFileContent ValidationType = "file_content"
	ValidationTypeConfigValue ValidationType = "config_value"
	ValidationTypeCommand     ValidationType = "command"
	ValidationTypePattern     ValidationType = "pattern"
	ValidationTypeStructure   ValidationType = "structure"
)

type BestPracticeCategory string

const (
	CategorySecurity        BestPracticeCategory = "security"
	CategoryPerformance     BestPracticeCategory = "performance"
	CategoryMaintainability BestPracticeCategory = "maintainability"
	CategoryTesting         BestPracticeCategory = "testing"
	CategoryDocumentation   BestPracticeCategory = "documentation"
	CategoryStructure       BestPracticeCategory = "structure"
	CategoryDependencies    BestPracticeCategory = "dependencies"
)

type ImpactLevel string

const (
	ImpactLevelLow      ImpactLevel = "low"
	ImpactLevelMedium   ImpactLevel = "medium"
	ImpactLevelHigh     ImpactLevel = "high"
	ImpactLevelCritical ImpactLevel = "critical"
)

type ActionType string

const (
	ActionTypeFileOperation    ActionType = "file_operation"
	ActionTypeConfigChange     ActionType = "config_change"
	ActionTypeCodeTransform    ActionType = "code_transform"
	ActionTypeDependencyUpdate ActionType = "dependency_update"
	ActionTypeDocGeneration    ActionType = "doc_generation"
)

type FailureAction string

const (
	FailureActionStop     FailureAction = "stop"
	FailureActionContinue FailureAction = "continue"
	FailureActionRollback FailureAction = "rollback"
	FailureActionRetry    FailureAction = "retry"
)

type BackoffType string

const (
	BackoffTypeLinear      BackoffType = "linear"
	BackoffTypeExponential BackoffType = "exponential"
	BackoffTypeFixed       BackoffType = "fixed"
)

// NewProjectMigrator creates a new project migrator.
func NewProjectMigrator(workDir string, logger interfaces.Logger, configManager interfaces.ConfigManager, fileSystem interfaces.FileSystem) *ProjectMigrator {
	migrator := &ProjectMigrator{
		workDir:            workDir,
		logger:             logger,
		configManager:      configManager,
		fileSystem:         fileSystem,
		migrations:         make([]Migration, 0),
		bestPractices:      make([]BestPractice, 0),
		executedMigrations: make(map[string]time.Time),
		rollbackStack:      make([]RollbackEntry, 0),
		dryRun:             false,
		backupEnabled:      true,
		backupDir:          filepath.Join(workDir, ".switctl", "backups"),
	}

	// Initialize built-in migrations and best practices
	migrator.initializeBuiltinMigrations()
	migrator.initializeBuiltinBestPractices()

	return migrator
}

// ExecuteMigration executes a specific migration by ID.
func (pm *ProjectMigrator) ExecuteMigration(migrationID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	migration, err := pm.findMigration(migrationID)
	if err != nil {
		return fmt.Errorf("migration not found: %w", err)
	}

	return pm.executeMigration(migration)
}

// ExecuteMigrations executes all pending migrations.
func (pm *ProjectMigrator) ExecuteMigrations() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Get pending migrations sorted by priority
	pendingMigrations := pm.getPendingMigrations()

	if len(pendingMigrations) == 0 {
		if pm.logger != nil {
			pm.logger.Info("No pending migrations found")
		}
		return nil
	}

	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Executing %d pending migrations", len(pendingMigrations)))
	}

	// Execute migrations in order
	for _, migration := range pendingMigrations {
		if err := pm.executeMigration(&migration); err != nil {
			return fmt.Errorf("migration %s failed: %w", migration.ID, err)
		}
	}

	if pm.logger != nil {
		pm.logger.Info("All migrations executed successfully")
	}

	return nil
}

// RollbackMigration rolls back a specific migration.
func (pm *ProjectMigrator) RollbackMigration(migrationID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	migration, err := pm.findMigration(migrationID)
	if err != nil {
		return fmt.Errorf("migration not found: %w", err)
	}

	if !migration.Reversible {
		return fmt.Errorf("migration %s is not reversible", migrationID)
	}

	return pm.rollbackMigration(migration)
}

// RollbackLastMigration rolls back the last executed migration.
func (pm *ProjectMigrator) RollbackLastMigration() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.rollbackStack) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	lastEntry := pm.rollbackStack[len(pm.rollbackStack)-1]

	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Rolling back migration: %s", lastEntry.MigrationID))
	}

	// Execute rollback operations
	for i := len(lastEntry.Operations) - 1; i >= 0; i-- {
		operation := lastEntry.Operations[i]
		if err := pm.executeOperation(&operation); err != nil {
			return fmt.Errorf("rollback operation failed: %w", err)
		}
	}

	// Remove from rollback stack and executed migrations
	pm.rollbackStack = pm.rollbackStack[:len(pm.rollbackStack)-1]
	delete(pm.executedMigrations, lastEntry.MigrationID)

	if pm.logger != nil {
		pm.logger.Info("Migration rolled back successfully")
	}

	return nil
}

// ApplyBestPractices applies all applicable best practices.
func (pm *ProjectMigrator) ApplyBestPractices() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	applicablePractices := pm.getApplicableBestPractices()

	if len(applicablePractices) == 0 {
		if pm.logger != nil {
			pm.logger.Info("No applicable best practices found")
		}
		return nil
	}

	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Applying %d best practices", len(applicablePractices)))
	}

	for _, practice := range applicablePractices {
		if err := pm.applyBestPractice(&practice); err != nil {
			if pm.logger != nil {
				pm.logger.Error(fmt.Sprintf("Failed to apply best practice %s: %v", practice.ID, err))
			}
			// Continue with other practices unless critical
			if practice.Impact == ImpactLevelCritical {
				return fmt.Errorf("critical best practice %s failed: %w", practice.ID, err)
			}
		}
	}

	if pm.logger != nil {
		pm.logger.Info("Best practices applied successfully")
	}

	return nil
}

// ValidateProject validates the project structure and configuration.
func (pm *ProjectMigrator) ValidateProject() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.logger != nil {
		pm.logger.Info("Validating project structure and configuration")
	}

	// Run validation checks from all migrations
	var validationErrors []string

	for _, migration := range pm.migrations {
		for _, validation := range migration.Validation {
			if err := pm.executeValidation(&validation); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("%s: %s", validation.Name, err.Error()))

				if validation.Critical {
					return fmt.Errorf("critical validation failed - %s: %w", validation.Name, err)
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("validation failed with %d errors: %s", len(validationErrors), strings.Join(validationErrors, "; "))
	}

	if pm.logger != nil {
		pm.logger.Info("Project validation completed successfully")
	}

	return nil
}

// GetPendingMigrations returns a list of pending migrations.
func (pm *ProjectMigrator) GetPendingMigrations() []Migration {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.getPendingMigrations()
}

// GetExecutedMigrations returns a list of executed migrations.
func (pm *ProjectMigrator) GetExecutedMigrations() map[string]time.Time {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	executed := make(map[string]time.Time)
	for id, timestamp := range pm.executedMigrations {
		executed[id] = timestamp
	}
	return executed
}

// GetApplicableBestPractices returns a list of applicable best practices.
func (pm *ProjectMigrator) GetApplicableBestPractices() []BestPractice {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.getApplicableBestPractices()
}

// SetDryRun enables or disables dry run mode.
func (pm *ProjectMigrator) SetDryRun(enabled bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.dryRun = enabled
}

// SetBackupEnabled enables or disables backup creation.
func (pm *ProjectMigrator) SetBackupEnabled(enabled bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.backupEnabled = enabled
}

// AddMigration adds a custom migration.
func (pm *ProjectMigrator) AddMigration(migration Migration) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Validate migration
	if err := pm.validateMigration(&migration); err != nil {
		return fmt.Errorf("invalid migration: %w", err)
	}

	pm.migrations = append(pm.migrations, migration)
	return nil
}

// AddBestPractice adds a custom best practice.
func (pm *ProjectMigrator) AddBestPractice(practice BestPractice) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.bestPractices = append(pm.bestPractices, practice)
	return nil
}

// Private methods

func (pm *ProjectMigrator) findMigration(id string) (*Migration, error) {
	for _, migration := range pm.migrations {
		if migration.ID == id {
			return &migration, nil
		}
	}
	return nil, fmt.Errorf("migration %s not found", id)
}

func (pm *ProjectMigrator) getPendingMigrations() []Migration {
	pending := make([]Migration, 0)

	for _, migration := range pm.migrations {
		if _, executed := pm.executedMigrations[migration.ID]; !executed {
			// Check if dependencies are satisfied
			if pm.areDependenciesSatisfied(&migration) {
				pending = append(pending, migration)
			}
		}
	}

	// Sort by priority (higher priority first)
	for i := 0; i < len(pending)-1; i++ {
		for j := i + 1; j < len(pending); j++ {
			if pending[i].Priority < pending[j].Priority {
				pending[i], pending[j] = pending[j], pending[i]
			}
		}
	}

	return pending
}

func (pm *ProjectMigrator) getApplicableBestPractices() []BestPractice {
	applicable := make([]BestPractice, 0)

	for _, practice := range pm.bestPractices {
		if pm.isBestPracticeApplicable(&practice) {
			applicable = append(applicable, practice)
		}
	}

	return applicable
}

func (pm *ProjectMigrator) areDependenciesSatisfied(migration *Migration) bool {
	for _, depID := range migration.Dependencies {
		if _, executed := pm.executedMigrations[depID]; !executed {
			return false
		}
	}
	return true
}

func (pm *ProjectMigrator) isBestPracticeApplicable(practice *BestPractice) bool {
	// Check applicability rules
	for _, rule := range practice.Applies {
		if !pm.evaluateApplicabilityRule(&rule) {
			return false
		}
	}
	return true
}

func (pm *ProjectMigrator) evaluateApplicabilityRule(rule *ApplicabilityRule) bool {
	// Simplified rule evaluation - in a real implementation this would be more comprehensive
	switch rule.Condition {
	case "file_exists":
		return pm.fileSystem.Exists(rule.Value.(string))
	case "config_value":
		if pm.configManager != nil {
			value := pm.configManager.Get(rule.Value.(string))
			return value != nil
		}
	}
	return true
}

func (pm *ProjectMigrator) executeMigration(migration *Migration) error {
	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Executing migration: %s", migration.Name))
	}

	// Check conditions
	for _, condition := range migration.Conditions {
		if !pm.evaluateCondition(&condition) {
			if condition.Required {
				return fmt.Errorf("required condition not met: %s", condition.Message)
			}
			if pm.logger != nil {
				pm.logger.Warn(fmt.Sprintf("Optional condition not met: %s", condition.Message))
			}
		}
	}

	// Create backup if enabled
	if pm.backupEnabled && !pm.dryRun {
		if err := pm.createBackup(migration.ID); err != nil {
			if pm.logger != nil {
				pm.logger.Warn(fmt.Sprintf("Failed to create backup: %v", err))
			}
		}
	}

	// Execute operations
	rollbackOps := make([]MigrationOperation, 0)

	for _, operation := range migration.Operations {
		if pm.dryRun {
			if pm.logger != nil {
				pm.logger.Info(fmt.Sprintf("DRY RUN: Would execute operation %s on %s", operation.Type, operation.Target))
			}
			continue
		}

		// Generate rollback operation before executing
		if migration.Reversible {
			rollbackOp := pm.generateRollbackOperation(&operation)
			if rollbackOp != nil {
				rollbackOps = append(rollbackOps, *rollbackOp)
			}
		}

		if err := pm.executeOperation(&operation); err != nil {
			// Handle failure based on operation configuration
			switch operation.OnFailure {
			case FailureActionStop:
				return fmt.Errorf("operation failed: %w", err)
			case FailureActionRollback:
				pm.rollbackOperations(rollbackOps)
				return fmt.Errorf("operation failed and rolled back: %w", err)
			case FailureActionContinue:
				if pm.logger != nil {
					pm.logger.Warn(fmt.Sprintf("Operation failed but continuing: %v", err))
				}
			case FailureActionRetry:
				if err := pm.retryOperation(&operation); err != nil {
					return fmt.Errorf("operation failed after retries: %w", err)
				}
			}
		}
	}

	// Run validation
	for _, validation := range migration.Validation {
		if err := pm.executeValidation(&validation); err != nil {
			if validation.Critical {
				// Rollback on critical validation failure
				if migration.Reversible {
					pm.rollbackOperations(rollbackOps)
				}
				return fmt.Errorf("critical validation failed: %w", err)
			}
			if pm.logger != nil {
				pm.logger.Warn(fmt.Sprintf("Validation warning: %v", err))
			}
		}
	}

	// Mark as executed
	if !pm.dryRun {
		now := time.Now()
		pm.executedMigrations[migration.ID] = now
		migration.ExecutedAt = &now

		// Add to rollback stack
		if migration.Reversible && len(rollbackOps) > 0 {
			pm.rollbackStack = append(pm.rollbackStack, RollbackEntry{
				MigrationID: migration.ID,
				Operations:  rollbackOps,
				Timestamp:   now,
				Context:     make(map[string]interface{}),
			})
		}
	}

	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Migration %s executed successfully", migration.Name))
	}

	return nil
}

func (pm *ProjectMigrator) rollbackMigration(migration *Migration) error {
	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Rolling back migration: %s", migration.Name))
	}

	// Find rollback entry
	var rollbackEntry *RollbackEntry
	for i := len(pm.rollbackStack) - 1; i >= 0; i-- {
		if pm.rollbackStack[i].MigrationID == migration.ID {
			rollbackEntry = &pm.rollbackStack[i]
			break
		}
	}

	if rollbackEntry == nil {
		// Use migration's rollback operations
		for i := len(migration.Rollback) - 1; i >= 0; i-- {
			operation := migration.Rollback[i]
			if err := pm.executeOperation(&operation); err != nil {
				return fmt.Errorf("rollback operation failed: %w", err)
			}
		}
	} else {
		// Use generated rollback operations
		for i := len(rollbackEntry.Operations) - 1; i >= 0; i-- {
			operation := rollbackEntry.Operations[i]
			if err := pm.executeOperation(&operation); err != nil {
				return fmt.Errorf("rollback operation failed: %w", err)
			}
		}
	}

	// Remove from executed migrations
	delete(pm.executedMigrations, migration.ID)
	migration.ExecutedAt = nil

	if pm.logger != nil {
		pm.logger.Info("Migration rolled back successfully")
	}

	return nil
}

func (pm *ProjectMigrator) applyBestPractice(practice *BestPractice) error {
	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Applying best practice: %s", practice.Name))
	}

	for _, action := range practice.Actions {
		if err := pm.executeBestPracticeAction(&action); err != nil {
			if action.Optional {
				if pm.logger != nil {
					pm.logger.Warn(fmt.Sprintf("Optional best practice action failed: %v", err))
				}
				continue
			}
			return fmt.Errorf("best practice action failed: %w", err)
		}
	}

	return nil
}

func (pm *ProjectMigrator) executeOperation(operation *MigrationOperation) error {
	switch operation.Type {
	case OperationTypeFileCreate:
		return pm.executeFileCreateOperation(operation)
	case OperationTypeFileUpdate:
		return pm.executeFileUpdateOperation(operation)
	case OperationTypeFileDelete:
		return pm.executeFileDeleteOperation(operation)
	case OperationTypeFileMove:
		return pm.executeFileMoveOperation(operation)
	case OperationTypeFileCopy:
		return pm.executeFileCopyOperation(operation)
	case OperationTypeDirectoryCreate:
		return pm.executeDirectoryCreateOperation(operation)
	case OperationTypeDirectoryDelete:
		return pm.executeDirectoryDeleteOperation(operation)
	case OperationTypeConfigUpdate:
		return pm.executeConfigUpdateOperation(operation)
	case OperationTypeTemplate:
		return pm.executeTemplateOperation(operation)
	case OperationTypeReplace:
		return pm.executeReplaceOperation(operation)
	case OperationTypeInsert:
		return pm.executeInsertOperation(operation)
	case OperationTypeRemove:
		return pm.executeRemoveOperation(operation)
	default:
		return fmt.Errorf("unknown operation type: %s", operation.Type)
	}
}

func (pm *ProjectMigrator) executeFileCreateOperation(operation *MigrationOperation) error {
	target := operation.Target
	content := ""

	if contentParam, exists := operation.Parameters["content"]; exists {
		content = contentParam.(string)
	}

	if pm.fileSystem.Exists(target) {
		return fmt.Errorf("file already exists: %s", target)
	}

	return pm.fileSystem.WriteFile(target, []byte(content), 0644)
}

func (pm *ProjectMigrator) executeFileUpdateOperation(operation *MigrationOperation) error {
	target := operation.Target

	if !pm.fileSystem.Exists(target) {
		return fmt.Errorf("file does not exist: %s", target)
	}

	// Read current content
	currentContent, err := pm.fileSystem.ReadFile(target)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Apply updates based on parameters
	newContent := string(currentContent)

	if search, exists := operation.Parameters["search"]; exists {
		if replace, exists := operation.Parameters["replace"]; exists {
			newContent = strings.ReplaceAll(newContent, search.(string), replace.(string))
		}
	}

	return pm.fileSystem.WriteFile(target, []byte(newContent), 0644)
}

func (pm *ProjectMigrator) executeFileDeleteOperation(operation *MigrationOperation) error {
	target := operation.Target

	if !pm.fileSystem.Exists(target) {
		if pm.logger != nil {
			pm.logger.Warn(fmt.Sprintf("File does not exist, skipping delete: %s", target))
		}
		return nil
	}

	return pm.fileSystem.Remove(target)
}

func (pm *ProjectMigrator) executeFileMoveOperation(operation *MigrationOperation) error {
	source := operation.Target
	destination := operation.Parameters["destination"].(string)

	if !pm.fileSystem.Exists(source) {
		return fmt.Errorf("source file does not exist: %s", source)
	}

	// Read source file
	content, err := pm.fileSystem.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Create destination directory if needed
	destDir := filepath.Dir(destination)
	if !pm.fileSystem.Exists(destDir) {
		if err := pm.fileSystem.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}

	// Write to destination
	if err := pm.fileSystem.WriteFile(destination, content, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	// Remove source
	return pm.fileSystem.Remove(source)
}

func (pm *ProjectMigrator) executeFileCopyOperation(operation *MigrationOperation) error {
	source := operation.Target
	destination := operation.Parameters["destination"].(string)

	return pm.fileSystem.Copy(source, destination)
}

func (pm *ProjectMigrator) executeDirectoryCreateOperation(operation *MigrationOperation) error {
	target := operation.Target

	return pm.fileSystem.MkdirAll(target, 0755)
}

func (pm *ProjectMigrator) executeDirectoryDeleteOperation(operation *MigrationOperation) error {
	target := operation.Target

	if !pm.fileSystem.Exists(target) {
		if pm.logger != nil {
			pm.logger.Warn(fmt.Sprintf("Directory does not exist, skipping delete: %s", target))
		}
		return nil
	}

	return pm.fileSystem.Remove(target)
}

func (pm *ProjectMigrator) executeConfigUpdateOperation(operation *MigrationOperation) error {
	if pm.configManager == nil {
		return fmt.Errorf("config manager not available")
	}

	key := operation.Target

	if value, exists := operation.Parameters["value"]; exists {
		pm.configManager.Set(key, value)
	} else if operation.Parameters["delete"] == true {
		pm.configManager.Set(key, nil)
	}

	return nil
}

func (pm *ProjectMigrator) executeTemplateOperation(operation *MigrationOperation) error {
	if pm.templateEngine == nil {
		return fmt.Errorf("template engine not available")
	}

	templateName := operation.Parameters["template"].(string)
	outputPath := operation.Target
	data := operation.Parameters["data"]

	template, err := pm.templateEngine.LoadTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	content, err := pm.templateEngine.RenderTemplate(template, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return pm.fileSystem.WriteFile(outputPath, content, 0644)
}

func (pm *ProjectMigrator) executeReplaceOperation(operation *MigrationOperation) error {
	target := operation.Target
	search := operation.Parameters["search"].(string)
	replace := operation.Parameters["replace"].(string)

	content, err := pm.fileSystem.ReadFile(target)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Handle regex replacement if specified
	if useRegex, exists := operation.Parameters["regex"]; exists && useRegex.(bool) {
		re, err := regexp.Compile(search)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		newContent := re.ReplaceAllString(string(content), replace)
		return pm.fileSystem.WriteFile(target, []byte(newContent), 0644)
	}

	// Simple string replacement
	newContent := strings.ReplaceAll(string(content), search, replace)
	return pm.fileSystem.WriteFile(target, []byte(newContent), 0644)
}

func (pm *ProjectMigrator) executeInsertOperation(operation *MigrationOperation) error {
	target := operation.Target
	content := operation.Parameters["content"].(string)
	position := "end" // default

	if pos, exists := operation.Parameters["position"]; exists {
		position = pos.(string)
	}

	existingContent, err := pm.fileSystem.ReadFile(target)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var newContent string
	switch position {
	case "start":
		newContent = content + string(existingContent)
	case "end":
		newContent = string(existingContent) + content
	default:
		// Insert at specific line
		lines := strings.Split(string(existingContent), "\n")
		lineNum := operation.Parameters["line"].(int)
		if lineNum >= 0 && lineNum <= len(lines) {
			before := strings.Join(lines[:lineNum], "\n")
			after := strings.Join(lines[lineNum:], "\n")
			newContent = before + "\n" + content + "\n" + after
		} else {
			return fmt.Errorf("invalid line number: %d", lineNum)
		}
	}

	return pm.fileSystem.WriteFile(target, []byte(newContent), 0644)
}

func (pm *ProjectMigrator) executeRemoveOperation(operation *MigrationOperation) error {
	target := operation.Target
	pattern := operation.Parameters["pattern"].(string)

	content, err := pm.fileSystem.ReadFile(target)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Handle regex removal if specified
	if useRegex, exists := operation.Parameters["regex"]; exists && useRegex.(bool) {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		newContent := re.ReplaceAllString(string(content), "")
		return pm.fileSystem.WriteFile(target, []byte(newContent), 0644)
	}

	// Simple string removal
	newContent := strings.ReplaceAll(string(content), pattern, "")
	return pm.fileSystem.WriteFile(target, []byte(newContent), 0644)
}

func (pm *ProjectMigrator) executeBestPracticeAction(action *BestPracticeAction) error {
	switch action.Type {
	case ActionTypeFileOperation:
		return pm.executeBestPracticeFileOperation(action)
	case ActionTypeConfigChange:
		return pm.executeBestPracticeConfigChange(action)
	case ActionTypeCodeTransform:
		return pm.executeBestPracticeCodeTransform(action)
	case ActionTypeDependencyUpdate:
		return pm.executeBestPracticeDependencyUpdate(action)
	case ActionTypeDocGeneration:
		return pm.executeBestPracticeDocGeneration(action)
	default:
		return fmt.Errorf("unknown best practice action type: %s", action.Type)
	}
}

func (pm *ProjectMigrator) executeBestPracticeFileOperation(action *BestPracticeAction) error {
	// Implementation would depend on specific file operation
	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Executing file operation: %s", action.Description))
	}
	return nil
}

func (pm *ProjectMigrator) executeBestPracticeConfigChange(action *BestPracticeAction) error {
	if pm.configManager == nil {
		return fmt.Errorf("config manager not available")
	}

	key := action.Target
	if value, exists := action.Parameters["value"]; exists {
		pm.configManager.Set(key, value)
	}

	return nil
}

func (pm *ProjectMigrator) executeBestPracticeCodeTransform(action *BestPracticeAction) error {
	// Implementation would involve code analysis and transformation
	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Executing code transform: %s", action.Description))
	}
	return nil
}

func (pm *ProjectMigrator) executeBestPracticeDependencyUpdate(action *BestPracticeAction) error {
	// Implementation would involve updating go.mod or package files
	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Executing dependency update: %s", action.Description))
	}
	return nil
}

func (pm *ProjectMigrator) executeBestPracticeDocGeneration(action *BestPracticeAction) error {
	// Implementation would involve generating documentation
	if pm.logger != nil {
		pm.logger.Info(fmt.Sprintf("Executing doc generation: %s", action.Description))
	}
	return nil
}

func (pm *ProjectMigrator) executeValidation(validation *ValidationCheck) error {
	switch validation.Type {
	case ValidationTypeFileExists:
		if !pm.fileSystem.Exists(validation.Target) {
			return fmt.Errorf("file does not exist: %s", validation.Target)
		}
	case ValidationTypeFileContent:
		content, err := pm.fileSystem.ReadFile(validation.Target)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		expected := validation.Expected.(string)
		if !strings.Contains(string(content), expected) {
			return fmt.Errorf("file content validation failed: expected content not found")
		}
	case ValidationTypeConfigValue:
		if pm.configManager == nil {
			return fmt.Errorf("config manager not available")
		}
		value := pm.configManager.Get(validation.Target)
		if value != validation.Expected {
			return fmt.Errorf("config value mismatch: expected %v, got %v", validation.Expected, value)
		}
	case ValidationTypePattern:
		content, err := pm.fileSystem.ReadFile(validation.Target)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		pattern := validation.Expected.(string)
		matched, err := regexp.Match(pattern, content)
		if err != nil {
			return fmt.Errorf("regex error: %w", err)
		}
		if !matched {
			return fmt.Errorf("pattern validation failed: pattern not found")
		}
	}
	return nil
}

func (pm *ProjectMigrator) evaluateCondition(condition *MigrationCondition) bool {
	switch condition.Type {
	case ConditionTypeFileExists:
		return pm.fileSystem.Exists(condition.Target)
	case ConditionTypeDirectoryExists:
		return pm.fileSystem.Exists(condition.Target)
	case ConditionTypeConfigValue:
		if pm.configManager == nil {
			return false
		}
		value := pm.configManager.Get(condition.Target)
		return pm.compareValues(value, condition.Value, condition.Operator)
	case ConditionTypeContent:
		content, err := pm.fileSystem.ReadFile(condition.Target)
		if err != nil {
			return false
		}
		return strings.Contains(string(content), condition.Value.(string))
	}
	return true
}

func (pm *ProjectMigrator) compareValues(actual, expected interface{}, operator string) bool {
	switch operator {
	case "==":
		return actual == expected
	case "!=":
		return actual != expected
	case "contains":
		if actualStr, ok := actual.(string); ok {
			return strings.Contains(actualStr, expected.(string))
		}
	}
	return false
}

func (pm *ProjectMigrator) generateRollbackOperation(operation *MigrationOperation) *MigrationOperation {
	switch operation.Type {
	case OperationTypeFileCreate:
		return &MigrationOperation{
			Type:   OperationTypeFileDelete,
			Target: operation.Target,
		}
	case OperationTypeFileDelete:
		// Would need to store file content for restoration
		return nil
	case OperationTypeFileMove:
		return &MigrationOperation{
			Type:   OperationTypeFileMove,
			Target: operation.Parameters["destination"].(string),
			Parameters: map[string]interface{}{
				"destination": operation.Target,
			},
		}
	}
	return nil
}

func (pm *ProjectMigrator) rollbackOperations(operations []MigrationOperation) {
	for i := len(operations) - 1; i >= 0; i-- {
		operation := operations[i]
		if err := pm.executeOperation(&operation); err != nil && pm.logger != nil {
			pm.logger.Error(fmt.Sprintf("Rollback operation failed: %v", err))
		}
	}
}

func (pm *ProjectMigrator) retryOperation(operation *MigrationOperation) error {
	if operation.Retry.MaxAttempts <= 1 {
		return fmt.Errorf("no retry configuration")
	}

	var lastErr error
	for attempt := 1; attempt < operation.Retry.MaxAttempts; attempt++ {
		time.Sleep(pm.calculateRetryDelay(attempt, &operation.Retry))

		if err := pm.executeOperation(operation); err != nil {
			lastErr = err
			if pm.logger != nil {
				pm.logger.Warn(fmt.Sprintf("Retry attempt %d failed: %v", attempt, err))
			}
			continue
		}

		return nil // Success
	}

	return fmt.Errorf("operation failed after %d attempts: %w", operation.Retry.MaxAttempts, lastErr)
}

func (pm *ProjectMigrator) calculateRetryDelay(attempt int, config *RetryConfig) time.Duration {
	switch config.Backoff {
	case BackoffTypeLinear:
		return config.Delay * time.Duration(attempt)
	case BackoffTypeExponential:
		return config.Delay * time.Duration(1<<attempt)
	case BackoffTypeFixed:
		return config.Delay
	default:
		return config.Delay
	}
}

func (pm *ProjectMigrator) createBackup(migrationID string) error {
	backupPath := filepath.Join(pm.backupDir, fmt.Sprintf("%s_%d", migrationID, time.Now().Unix()))

	if err := pm.fileSystem.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy important files to backup
	// This is a simplified implementation
	return pm.fileSystem.Copy(pm.workDir, backupPath)
}

func (pm *ProjectMigrator) validateMigration(migration *Migration) error {
	if migration.ID == "" {
		return fmt.Errorf("migration ID is required")
	}
	if migration.Name == "" {
		return fmt.Errorf("migration name is required")
	}
	if len(migration.Operations) == 0 {
		return fmt.Errorf("migration must have at least one operation")
	}
	return nil
}

func (pm *ProjectMigrator) initializeBuiltinMigrations() {
	// Initialize built-in migrations
	pm.migrations = []Migration{
		{
			ID:          "001_update_go_mod",
			Name:        "Update go.mod framework version",
			Description: "Updates the framework version in go.mod file",
			Version:     "0.0.2",
			Type:        MigrationTypeDependency,
			Priority:    100,
			Conditions: []MigrationCondition{
				{
					Type:     ConditionTypeFileExists,
					Target:   "go.mod",
					Required: true,
					Message:  "go.mod file must exist",
				},
			},
			Operations: []MigrationOperation{
				{
					Type:   OperationTypeReplace,
					Target: "go.mod",
					Parameters: map[string]interface{}{
						"search":  "github.com/innovationmech/swit v0.0.1",
						"replace": "github.com/innovationmech/swit v0.0.2",
						"regex":   false,
					},
					OnFailure: FailureActionStop,
				},
			},
			Validation: []ValidationCheck{
				{
					Name:     "verify_version_update",
					Type:     ValidationTypeFileContent,
					Target:   "go.mod",
					Expected: "github.com/innovationmech/swit v0.0.2",
					Critical: true,
				},
			},
			Reversible: true,
			CreatedAt:  time.Now(),
		},
		{
			ID:          "002_create_config_dir",
			Name:        "Create .switctl configuration directory",
			Description: "Creates the .switctl directory for configuration files",
			Version:     "0.0.2",
			Type:        MigrationTypeStructural,
			Priority:    90,
			Operations: []MigrationOperation{
				{
					Type:      OperationTypeDirectoryCreate,
					Target:    ".switctl",
					OnFailure: FailureActionContinue,
				},
			},
			Validation: []ValidationCheck{
				{
					Name:     "verify_config_dir",
					Type:     ValidationTypeFileExists,
					Target:   ".switctl",
					Critical: false,
				},
			},
			Reversible: true,
			CreatedAt:  time.Now(),
		},
	}
}

func (pm *ProjectMigrator) initializeBuiltinBestPractices() {
	// Initialize built-in best practices
	pm.bestPractices = []BestPractice{
		{
			ID:          "bp_001_add_copyright_headers",
			Name:        "Add copyright headers to Go files",
			Description: "Ensures all Go files have proper copyright headers",
			Category:    CategoryMaintainability,
			Impact:      ImpactLevelLow,
			Actions: []BestPracticeAction{
				{
					Type:        ActionTypeCodeTransform,
					Target:      "*.go",
					Description: "Add copyright header to Go files",
					Parameters: map[string]interface{}{
						"header_template": "copyright_header.tmpl",
					},
					Optional: true,
				},
			},
			Applies: []ApplicabilityRule{
				{
					Condition: "file_exists",
					Value:     "*.go",
					Operator:  "pattern",
				},
			},
			Version:   "0.0.2",
			Tags:      []string{"legal", "maintainability"},
			AutoApply: false,
		},
		{
			ID:          "bp_002_add_test_files",
			Name:        "Ensure test files exist for Go packages",
			Description: "Creates test files for packages that don't have them",
			Category:    CategoryTesting,
			Impact:      ImpactLevelMedium,
			Actions: []BestPracticeAction{
				{
					Type:        ActionTypeFileOperation,
					Target:      "*_test.go",
					Description: "Create test files for packages without tests",
					Parameters: map[string]interface{}{
						"template": "test_file.tmpl",
					},
					Optional: true,
				},
			},
			Applies: []ApplicabilityRule{
				{
					Condition: "file_exists",
					Value:     "*.go",
					Operator:  "pattern",
				},
			},
			Version:   "0.0.2",
			Tags:      []string{"testing", "quality"},
			AutoApply: false,
		},
	}
}

// CopyFile copies a file from source to destination.
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
