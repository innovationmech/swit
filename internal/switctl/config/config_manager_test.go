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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Test hierarchical precedence: defaults < global < user < project < env
func TestHierarchicalConfigManager_Load_MergePrecedence(t *testing.T) {
	tempDir := t.TempDir()

	// Prepare file paths for all levels under tempDir
	globalPath := filepath.Join(tempDir, "global.yaml")
	userPath := filepath.Join(tempDir, "user.yaml")
	projectPath := filepath.Join(tempDir, ".switctl.yaml")

	// Global (lowest among files)
	if err := os.WriteFile(globalPath, []byte(
		"logging:\n  level: info\nserver:\n  http:\n    port: 9000\n"), 0o644); err != nil {
		t.Fatalf("write global: %v", err)
	}

	// User overrides global
	if err := os.WriteFile(userPath, []byte(
		"server:\n  http:\n    port: 9100\n"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	// Project overrides user
	if err := os.WriteFile(projectPath, []byte(
		"server:\n  http:\n    port: 9200\n"), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}

	// Env overrides all
	t.Setenv("SWITCTL_SERVER_HTTP_PORT", "9300")

	hcm := NewHierarchicalConfigManager(tempDir, nil)
	hcm.SetConfigPath(GlobalLevel, globalPath)
	hcm.SetConfigPath(UserLevel, userPath)
	hcm.SetConfigPath(ProjectLevel, projectPath)

	if err := hcm.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if got, want := hcm.GetString("server.http.port"), "9300"; got != want {
		t.Fatalf("server.http.port = %s, want %s", got, want)
	}
}

// Test validation: invalid port should fail
func TestHierarchicalConfigManager_Validation_InvalidPort(t *testing.T) {
	tempDir := t.TempDir()
	userPath := filepath.Join(tempDir, "user.yaml")

	// Invalid port -1
	if err := os.WriteFile(userPath, []byte(
		"server:\n  http:\n    port: -1\n"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	hcm := NewHierarchicalConfigManager(tempDir, nil)
	hcm.SetConfigPath(UserLevel, userPath)

	if err := hcm.Load(); err == nil {
		t.Fatalf("expected validation error for invalid port, got nil")
	}
}

// Test environment binding picks up defaults and existing keys
func TestHierarchicalConfigManager_EnvBinding_ForKnownKeys(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, ".switctl.yaml")

	if err := os.WriteFile(projectPath, []byte(
		"logging:\n  level: warn\n"), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}

	// Override a default key via env
	t.Setenv("SWITCTL_LOGGING_LEVEL", "debug")

	hcm := NewHierarchicalConfigManager(tempDir, nil)
	hcm.SetConfigPath(ProjectLevel, projectPath)

	if err := hcm.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if got, want := hcm.GetString("logging.level"), "debug"; got != want {
		t.Fatalf("logging.level = %s, want %s", got, want)
	}
}
