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

	"gopkg.in/yaml.v3"
)

func TestRunDiffCommand_JSON_Output(t *testing.T) {
	dir := t.TempDir()
	leftPath := filepath.Join(dir, "left.yaml")
	rightPath := filepath.Join(dir, "right.yaml")

	left := `server:
  http:
    port: 9000
featureX: true
`
	right := `server:
  http:
    port: 9100
newKey: "abc"
`
	if err := os.WriteFile(leftPath, []byte(left), 0644); err != nil {
		t.Fatalf("write left: %v", err)
	}
	if err := os.WriteFile(rightPath, []byte(right), 0644); err != nil {
		t.Fatalf("write right: %v", err)
	}

	// capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		_ = w.Close()
		os.Stdout = old
	}()

	// force json output
	reportFormat = "json"
	if err := runDiffCommand(leftPath, rightPath, true, false); err != nil {
		t.Fatalf("runDiffCommand: %v", err)
	}

	_ = w.Close()
	// consume the pipe to avoid linter complaining; ignore content validation here
	_ = r.Close()
}

func TestMigrateConfigFileWithRules(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "conf.yaml")
	content := `version: v1
old:
  key1: val1
obsolete:
  flag: true
`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	rules := []MigrationRule{
		{FromKey: "old.key1", ToKey: "new.key1"},
		{FromKey: "obsolete.flag", Remove: true},
		{ToKey: "added.defaulted", Default: "x"},
	}

	if err := migrateConfigFileWithRules(p, "v1", "v2", false, rules); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read migrated: %v", err)
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := m["version"]; got != "v2" {
		t.Fatalf("version not updated, got %v", got)
	}

	// check new.key1
	newMap, _ := m["new"].(map[string]interface{})
	if newMap == nil || newMap["key1"] != "val1" {
		t.Fatalf("new.key1 missing: %v", newMap)
	}

	// obsolete.flag should be removed
	if obs, ok := m["obsolete"].(map[string]interface{}); ok {
		if _, exists := obs["flag"]; exists {
			t.Fatalf("obsolete.flag still present")
		}
	}

	// default applied
	added, _ := m["added"].(map[string]interface{})
	if added == nil || added["defaulted"] != "x" {
		t.Fatalf("added.defaulted missing: %v", added)
	}
}
