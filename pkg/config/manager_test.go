package config

import (
	"os"
	"path/filepath"
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
