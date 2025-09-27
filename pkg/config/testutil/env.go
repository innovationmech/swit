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

package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// EnvSandbox provides a temporary directory and scoped environment variable sandbox for config testing.
// It creates an isolated working directory for writing config fixtures and offers helpers to set/unset
// environment variables with automatic restoration on cleanup.
type EnvSandbox struct {
	T          *testing.T
	Dir        string
	originalWD string
	savedEnv   map[string]*string
}

// NewEnvSandbox creates a new environment sandbox rooted at a temporary directory.
// The caller should defer sandbox.Cleanup().
func NewEnvSandbox(t *testing.T) *EnvSandbox {
	t.Helper()
	dir, err := os.MkdirTemp("", "swit-config-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	wd, _ := os.Getwd()
	return &EnvSandbox{T: t, Dir: dir, originalWD: wd, savedEnv: make(map[string]*string)}
}

// Chdir changes process working directory to the sandbox directory.
func (s *EnvSandbox) Chdir() {
	s.T.Helper()
	if err := os.Chdir(s.Dir); err != nil {
		s.T.Fatalf("chdir: %v", err)
	}
}

// WriteFile writes content to a relative path under the sandbox directory, creating parent directories.
func (s *EnvSandbox) WriteFile(rel string, content []byte) string {
	s.T.Helper()
	p := filepath.Join(s.Dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		s.T.Fatalf("mkdirs for %s: %v", p, err)
	}
	if err := os.WriteFile(p, content, 0o644); err != nil {
		s.T.Fatalf("write %s: %v", p, err)
	}
	return p
}

// WriteYAML marshals v as YAML and writes to a relative path.
func (s *EnvSandbox) WriteYAML(rel string, v interface{}) string {
	s.T.Helper()
	data, err := yaml.Marshal(v)
	if err != nil {
		s.T.Fatalf("yaml marshal %s: %v", rel, err)
	}
	return s.WriteFile(rel, data)
}

// SetEnv sets an environment variable and records its previous value for later restoration.
func (s *EnvSandbox) SetEnv(key, value string) {
	s.T.Helper()
	if _, ok := s.savedEnv[key]; !ok {
		if prev, exists := os.LookupEnv(key); exists {
			v := prev
			s.savedEnv[key] = &v
		} else {
			s.savedEnv[key] = nil
		}
	}
	if err := os.Setenv(key, value); err != nil {
		s.T.Fatalf("setenv %s: %v", key, err)
	}
}

// UnsetEnv unsets an environment variable and records its previous value.
func (s *EnvSandbox) UnsetEnv(key string) {
	s.T.Helper()
	if _, ok := s.savedEnv[key]; !ok {
		if prev, exists := os.LookupEnv(key); exists {
			v := prev
			s.savedEnv[key] = &v
		} else {
			s.savedEnv[key] = nil
		}
	}
	if err := os.Unsetenv(key); err != nil {
		s.T.Fatalf("unsetenv %s: %v", key, err)
	}
}

// Cleanup restores environment variables, resets working directory, and removes the sandbox directory.
func (s *EnvSandbox) Cleanup() {
	s.T.Helper()
	// Restore env
	for k, v := range s.savedEnv {
		if v == nil {
			_ = os.Unsetenv(k)
		} else {
			_ = os.Setenv(k, *v)
		}
	}
	// Restore wd
	if s.originalWD != "" {
		_ = os.Chdir(s.originalWD)
	}
	// Remove dir
	if s.Dir != "" {
		_ = os.RemoveAll(s.Dir)
	}
}
