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
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

type testConfig struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
	Messaging struct {
		Broker struct {
			Type string `mapstructure:"type"`
		} `mapstructure:"broker"`
		Publisher struct {
			BatchSize int `mapstructure:"batch_size"`
			Timeout   int `mapstructure:"timeout"`
		} `mapstructure:"publisher"`
	} `mapstructure:"messaging"`
}

func TestHierarchicalPrecedence(t *testing.T) {
	t.Setenv("SWIT_SERVER_PORT", "9300")
	t.Setenv("SWIT_MESSAGING_PUBLISHER_BATCH_SIZE", "200")

	tempDir := t.TempDir()

	// Base: lowest precedence (over defaults)
	writeFile(t, tempDir, "swit.yaml", `
server:
  port: "9000"
messaging:
  broker:
    type: rabbitmq
  publisher:
    batch_size: 10
    timeout: 5
`)

	// Env file: overrides base
	writeFile(t, tempDir, "swit.dev.yaml", `
server:
  port: "9100"
messaging:
  broker:
    type: kafka
  publisher:
    timeout: 8
`)

	// Override: overrides env file
	writeFile(t, tempDir, "swit.override.yaml", `
server:
  port: "9200"
messaging:
  publisher:
    batch_size: 100
`)

	m := NewManager(Options{
		WorkDir:            tempDir,
		ConfigBaseName:     "swit",
		ConfigType:         "yaml",
		EnvironmentName:    "dev",
		OverrideFilename:   "swit.override.yaml",
		EnvPrefix:          "SWIT",
		EnableAutomaticEnv: true,
	})

	// Defaults: should be overridden by base
	m.SetDefault("server.port", "8080")
	m.SetDefault("messaging.publisher.batch_size", 1)
	m.SetDefault("messaging.publisher.timeout", 2)

	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	var cfg testConfig
	if err := m.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Precedence check:
	// defaults(8080) < base(9000) < env(9100) < override(9200) < envvars(9300)
	if got, want := cfg.Server.Port, "9300"; got != want {
		t.Fatalf("server.port precedence = %s, want %s", got, want)
	}

	// Nested map merge: base sets batch_size=10, env overrides timeout to 8,
	// override sets batch_size=100, envvar sets batch_size=200 (highest)
	if got, want := cfg.Messaging.Publisher.Timeout, 8; got != want {
		t.Fatalf("publisher.timeout = %d, want %d", got, want)
	}
	if got, want := cfg.Messaging.Publisher.BatchSize, 200; got != want {
		t.Fatalf("publisher.batch_size = %d, want %d", got, want)
	}

	// Env file changed broker type from rabbitmq→kafka
	if got, want := cfg.Messaging.Broker.Type, "kafka"; got != want {
		t.Fatalf("broker.type = %s, want %s", got, want)
	}
}

func TestMissingFilesAreIgnored(t *testing.T) {
	tempDir := t.TempDir()

	// Only base exists
	writeFile(t, tempDir, "swit.yaml", `server: { port: "8000" }`)

	m := NewManager(DefaultOptions())
	m.options.WorkDir = tempDir
	m.options.EnvironmentName = "prod" // swit.prod.yaml not present

	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	var cfg testConfig
	if err := m.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := cfg.Server.Port, "8000"; got != want {
		t.Fatalf("server.port = %s, want %s", got, want)
	}
}

func TestUnmarshalNilTarget(t *testing.T) {
	m := NewManager(DefaultOptions())
	if err := m.Unmarshal(nil); err == nil {
		t.Fatalf("expected error for nil target")
	}
}

func TestAutoDetect_JSONAndYAML_MergePrecedence(t *testing.T) {
	t.Setenv("SWIT_SERVER_PORT", "9301")
	t.Setenv("SWIT_MESSAGING_PUBLISHER_BATCH_SIZE", "201")

	tempDir := t.TempDir()

	// Base: JSON
	writeFile(t, tempDir, "swit.json", `{
        "server": {"port": "9001"},
        "messaging": {
            "broker": {"type": "rabbitmq"},
            "publisher": {"batch_size": 11, "timeout": 6}
        }
    }`)

	// Env: YAML
	writeFile(t, tempDir, "swit.dev.yaml", `server: { port: "9101" }
messaging:
  publisher:
    timeout: 7
`)

	// Override: JSON
	writeFile(t, tempDir, "swit.override.json", `{"messaging": {"publisher": {"batch_size": 101}}}`)

	m := NewManager(Options{
		WorkDir:            tempDir,
		ConfigBaseName:     "swit",
		ConfigType:         "auto",
		EnvironmentName:    "dev",
		OverrideFilename:   "swit.override", // extension auto-detected
		EnvPrefix:          "SWIT",
		EnableAutomaticEnv: true,
	})

	m.SetDefault("server.port", "8080")
	m.SetDefault("messaging.publisher.batch_size", 1)
	m.SetDefault("messaging.publisher.timeout", 2)

	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	var cfg testConfig
	if err := m.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got, want := cfg.Server.Port, "9301"; got != want {
		t.Fatalf("server.port precedence = %s, want %s", got, want)
	}
	if got, want := cfg.Messaging.Publisher.Timeout, 7; got != want {
		t.Fatalf("publisher.timeout = %d, want %d", got, want)
	}
	if got, want := cfg.Messaging.Publisher.BatchSize, 201; got != want {
		t.Fatalf("publisher.batch_size = %d, want %d", got, want)
	}
}

func TestUnmarshalStrict_UnknownFields(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(t, tempDir, "swit.yaml", `server:
  port: "9000"
messaging:
  publisher:
    timeout: 8
  unknown_section: true
`)

	m := NewManager(Options{WorkDir: tempDir})
	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	var cfg testConfig
	if err := m.UnmarshalStrict(&cfg); err == nil {
		t.Fatalf("expected strict unmarshal to fail on unknown fields")
	}
}

func TestRequireKeys(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(t, tempDir, "swit.yaml", `server: { port: "8000" }`)

	m := NewManager(Options{WorkDir: tempDir})
	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := m.RequireKeys("server.port", "messaging.publisher.timeout", "nonexistent.key"); err == nil {
		t.Fatalf("expected missing keys error")
	}
}

func TestOverrideFilename_ExtensionAutoDetection(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(t, tempDir, "swit.yaml", `server: { port: "9800" }`)
	// provide custom override without extension in options; create JSON file
	writeFile(t, tempDir, "custom.override.json", `{"server": {"port": "9900"}}`)

	m := NewManager(Options{
		WorkDir:          tempDir,
		OverrideFilename: "custom.override",
		ConfigType:       "auto",
	})
	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	var cfg testConfig
	if err := m.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := cfg.Server.Port, "9900"; got != want {
		t.Fatalf("server.port = %s, want %s", got, want)
	}
}

func TestParseErrorActionable(t *testing.T) {
	tempDir := t.TempDir()
	// invalid YAML (missing colon)
	writeFile(t, tempDir, "swit.yaml", `server\n  port "9800"`)

	m := NewManager(Options{WorkDir: tempDir, ConfigType: "auto"})
	err := m.Load()
	if err == nil {
		t.Fatalf("expected parse error, got nil")
	}
	if !strings.Contains(err.Error(), "parse ") {
		t.Fatalf("error should include 'parse', got: %v", err)
	}
	if !strings.Contains(err.Error(), "swit.yaml") {
		t.Fatalf("error should include filename, got: %v", err)
	}
}

func TestInterpolationInConfigValues(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("MY_PORT", "9350")
	t.Setenv("MY_TIMEOUT", "12")

	writeFile(t, tempDir, "swit.yaml", `
server:
  port: "${MY_PORT:-9100}"
messaging:
  publisher:
    timeout: ${MY_TIMEOUT:-8}
`)

	m := NewManager(Options{WorkDir: tempDir, EnableInterpolation: true})
	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	var cfg testConfig
	if err := m.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := cfg.Server.Port, "9350"; got != want {
		t.Fatalf("interpolated server.port = %s, want %s", got, want)
	}
	if got, want := cfg.Messaging.Publisher.Timeout, 12; got != want {
		t.Fatalf("interpolated publisher.timeout = %d, want %d", got, want)
	}
}

func TestInterpolationDefaultFallback(t *testing.T) {
	tempDir := t.TempDir()

	writeFile(t, tempDir, "swit.yaml", `server: { port: "${UNSET_VAR:-9100}" }`)

	m := NewManager(Options{WorkDir: tempDir, EnableInterpolation: true})
	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	var cfg testConfig
	if err := m.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := cfg.Server.Port, "9100"; got != want {
		t.Fatalf("defaulted server.port = %s, want %s", got, want)
	}
}

func TestSecretFileSetsEnvAndOverrides(t *testing.T) {
	tempDir := t.TempDir()
	// Prepare secret file with desired port value
	secretPath := writeFile(t, tempDir, "port.txt", "9400\n")

	// Only set *_FILE, do not set the main var
	t.Setenv("SWIT_SERVER_PORT_FILE", secretPath)

	// Base file
	writeFile(t, tempDir, "swit.yaml", `server: { port: "9000" }`)

	m := NewManager(Options{WorkDir: tempDir, EnvPrefix: "SWIT", EnableAutomaticEnv: true})
	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	// After load, the env var should be populated from file
	if val := os.Getenv("SWIT_SERVER_PORT"); val != "9400" {
		t.Fatalf("SWIT_SERVER_PORT env = %q, want %q", val, "9400")
	}

	var cfg testConfig
	if err := m.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := cfg.Server.Port, "9400"; got != want {
		t.Fatalf("server.port with secret file = %s, want %s", got, want)
	}
}

func TestSecretFileConflictReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	secretPath := writeFile(t, tempDir, "token.txt", "s3cr3t\n")

	t.Setenv("SWIT_API_KEY", "inline")
	t.Setenv("SWIT_API_KEY_FILE", secretPath)

	writeFile(t, tempDir, "swit.yaml", `{}
`)

	m := NewManager(Options{WorkDir: tempDir, EnvPrefix: "SWIT"})
	if err := m.Load(); err == nil {
		t.Fatalf("expected conflict error when both VAR and VAR_FILE are set")
	}
}

func TestExtends_SingleChainAndPrecedence(t *testing.T) {
	tempDir := t.TempDir()

	// common base referenced by swit.yaml
	writeFile(t, tempDir, "base.yaml", `
server:
  port: "8000"
messaging:
  broker:
    type: rabbitmq
  publisher:
    batch_size: 5
    timeout: 3
`)

	// swit.yaml extends base.yaml and overrides some fields
	writeFile(t, tempDir, "swit.yaml", `
extends: base.yaml
server:
  port: "9000"
messaging:
  publisher:
    timeout: 6
`)

	m := NewManager(Options{WorkDir: tempDir})
	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	var cfg testConfig
	if err := m.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got, want := cfg.Server.Port, "9000"; got != want {
		t.Fatalf("server.port = %s, want %s", got, want)
	}
	if got, want := cfg.Messaging.Publisher.BatchSize, 5; got != want {
		t.Fatalf("publisher.batch_size = %d, want %d", got, want)
	}
	if got, want := cfg.Messaging.Publisher.Timeout, 6; got != want {
		t.Fatalf("publisher.timeout = %d, want %d", got, want)
	}
	if v := m.Get("extends"); v != nil {
		t.Fatalf("extends key leaked into settings: %v", v)
	}
}

func TestExtends_MultipleFiles_ListOrder(t *testing.T) {
	tempDir := t.TempDir()

	writeFile(t, tempDir, "one.yaml", `
messaging:
  publisher:
    batch_size: 10
`)
	writeFile(t, tempDir, "two.yaml", `
messaging:
  publisher:
    timeout: 7
`)
	writeFile(t, tempDir, "swit.yaml", `
extends: [one.yaml, two.yaml]
messaging:
  publisher:
    batch_size: 20
`)

	m := NewManager(Options{WorkDir: tempDir})
	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	var cfg testConfig
	if err := m.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := cfg.Messaging.Publisher.BatchSize, 20; got != want {
		t.Fatalf("publisher.batch_size = %d, want %d", got, want)
	}
	if got, want := cfg.Messaging.Publisher.Timeout, 7; got != want {
		t.Fatalf("publisher.timeout = %d, want %d", got, want)
	}
}

func TestExtends_RelativePathWithoutExtension_AutoDetection(t *testing.T) {
	tempDir := t.TempDir()
	nested := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, nested, "base.yml", `server: { port: "9700" }`)

	writeFile(t, tempDir, "swit.yaml", `
extends: configs/base
messaging:
  publisher:
    timeout: 4
`)

	m := NewManager(Options{WorkDir: tempDir, ConfigType: "auto"})
	if err := m.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	var cfg testConfig
	if err := m.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := cfg.Server.Port, "9700"; got != want {
		t.Fatalf("server.port = %s, want %s", got, want)
	}
}

func TestExtends_CycleDetection(t *testing.T) {
	tempDir := t.TempDir()

	writeFile(t, tempDir, "a.yaml", `extends: b.yaml
server: { port: "9800" }
`)
	writeFile(t, tempDir, "b.yaml", `extends: a.yaml
`)
	writeFile(t, tempDir, "swit.yaml", `extends: a.yaml
`)

	m := NewManager(Options{WorkDir: tempDir})
	if err := m.Load(); err == nil {
		t.Fatalf("expected cycle detection error, got nil")
	}
}

func TestExtends_MissingTargetReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(t, tempDir, "swit.yaml", `extends: not-exists.yaml
`)
	m := NewManager(Options{WorkDir: tempDir})
	if err := m.Load(); err == nil {
		t.Fatalf("expected error for missing extends target")
	}
}
