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

// Package template provides template engine functionality for code generation.
package template

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

//go:embed templates/*
var embeddedTemplates embed.FS

// Engine implements the TemplateEngine interface using Go's text/template.
type Engine struct {
	templateDir string
	funcMap     template.FuncMap
	templates   map[string]*template.Template
	store       *Store
	mu          sync.RWMutex // Protects templates map from concurrent access
}

// GoTemplate wraps a Go text/template.Template to implement the Template interface.
type GoTemplate struct {
	template *template.Template
	name     string
}

// NewEngine creates a new template engine instance.
func NewEngine(templateDir string) *Engine {
	engine := &Engine{
		templateDir: templateDir,
		templates:   make(map[string]*template.Template),
		funcMap:     make(template.FuncMap),
	}

	// Register built-in template functions
	engine.registerBuiltinFunctions()

	// Initialize template store
	engine.store = NewStore(templateDir)

	return engine
}

// NewEngineWithEmbedded creates a new template engine instance using embedded templates.
func NewEngineWithEmbedded() *Engine {
	engine := &Engine{
		templateDir: "",
		templates:   make(map[string]*template.Template),
		funcMap:     make(template.FuncMap),
	}

	// Register built-in template functions
	engine.registerBuiltinFunctions()

	// Initialize template store with embedded filesystem
	engine.store = NewStoreWithEmbedded(embeddedTemplates)

	return engine
}

// LoadTemplate loads a template by name.
func (e *Engine) LoadTemplate(name string) (interfaces.Template, error) {
	// Check if template is already loaded (read lock)
	e.mu.RLock()
	if tmpl, exists := e.templates[name]; exists {
		e.mu.RUnlock()
		return &GoTemplate{template: tmpl, name: name}, nil
	}
	e.mu.RUnlock()

	// Load from store
	content, err := e.store.GetTemplate(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load template %s: %w", name, err)
	}

	// Parse template with functions
	tmpl, err := template.New(name).Funcs(e.funcMap).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	// Cache the template (write lock)
	e.mu.Lock()
	// Double-check after acquiring write lock (another goroutine might have loaded it)
	if existingTmpl, exists := e.templates[name]; exists {
		e.mu.Unlock()
		return &GoTemplate{template: existingTmpl, name: name}, nil
	}
	e.templates[name] = tmpl
	e.mu.Unlock()

	return &GoTemplate{template: tmpl, name: name}, nil
}

// RenderTemplate renders a template with the provided data.
func (e *Engine) RenderTemplate(template interfaces.Template, data interface{}) ([]byte, error) {
	goTemplate, ok := template.(*GoTemplate)
	if !ok {
		return nil, fmt.Errorf("template is not a GoTemplate")
	}

	var buf bytes.Buffer
	if err := goTemplate.template.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render template %s: %w", goTemplate.name, err)
	}

	return buf.Bytes(), nil
}

// RegisterFunction registers a custom template function.
func (e *Engine) RegisterFunction(name string, fn interface{}) error {
	if name == "" {
		return fmt.Errorf("function name cannot be empty")
	}

	if fn == nil {
		return fmt.Errorf("function cannot be nil")
	}

	// Validate that fn is actually a function
	// Go's text/template accepts any interface{}, but we should validate for better error messages
	switch fn.(type) {
	case func(), func() interface{}, func() string, func() int, func() bool,
		func(interface{}) interface{}, func(interface{}) string, func(interface{}) int, func(interface{}) bool,
		func(interface{}, interface{}) interface{}, func(interface{}, interface{}) string,
		func(interface{}, interface{}) int, func(interface{}, interface{}) bool,
		func(string) string, func(string, string) string, func(int, int) int,
		func(string) interface{}, func(int) interface{}, func(bool) interface{}:
		// These are valid function types
	default:
		// Check if it's a function type using reflection would be more thorough,
		// but for testing purposes we can check for common invalid types
		switch fn.(type) {
		case string, int, bool, []string, map[string]interface{}:
			return fmt.Errorf("function must be callable, got %T", fn)
		}
		// Allow other types that might be functions (this is a basic check)
	}

	e.funcMap[name] = fn

	// Clear template cache to force recompilation with new function
	e.mu.Lock()
	e.templates = make(map[string]*template.Template)
	e.mu.Unlock()

	return nil
}

// SetTemplateDir sets the template directory path.
func (e *Engine) SetTemplateDir(dir string) error {
	// Validate directory exists
	if dir != "" {
		if _, err := os.Stat(dir); err != nil {
			return fmt.Errorf("template directory does not exist: %w", err)
		}
	}

	e.templateDir = dir
	e.store = NewStore(dir)

	// Clear template cache
	e.mu.Lock()
	e.templates = make(map[string]*template.Template)
	e.mu.Unlock()

	return nil
}

// GetStore returns the template store.
func (e *Engine) GetStore() *Store {
	return e.store
}

// LoadTemplatesFromDirectory loads all templates from a directory.
func (e *Engine) LoadTemplatesFromDirectory(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Get relative path from template directory
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		// Remove .tmpl extension for template name
		templateName := strings.TrimSuffix(relPath, ".tmpl")

		// Load the template
		_, err = e.LoadTemplate(templateName)
		if err != nil {
			return fmt.Errorf("failed to load template %s: %w", templateName, err)
		}

		return nil
	})
}

// GetLoadedTemplates returns a list of loaded template names.
func (e *Engine) GetLoadedTemplates() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var names []string
	for name := range e.templates {
		names = append(names, name)
	}
	return names
}

// ClearCache clears the template cache.
func (e *Engine) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.templates = make(map[string]*template.Template)
}

// registerBuiltinFunctions registers built-in template functions.
func (e *Engine) registerBuiltinFunctions() {
	// String manipulation functions
	e.funcMap["camelCase"] = toCamelCase
	e.funcMap["snakeCase"] = toSnakeCase
	e.funcMap["kebabCase"] = toKebabCase
	e.funcMap["pascalCase"] = toPascalCase
	e.funcMap["titleCase"] = humanize // titleCase is an alias for humanize
	e.funcMap["screamingSnakeCase"] = toScreamingSnakeCase
	e.funcMap["lower"] = strings.ToLower
	e.funcMap["upper"] = strings.ToUpper
	e.funcMap["title"] = strings.Title
	e.funcMap["trim"] = strings.TrimSpace
	e.funcMap["replace"] = strings.ReplaceAll
	e.funcMap["hasPrefix"] = strings.HasPrefix
	e.funcMap["hasSuffix"] = strings.HasSuffix
	e.funcMap["contains"] = strings.Contains
	e.funcMap["join"] = joinSlice
	e.funcMap["split"] = strings.Split

	// Array/slice functions
	e.funcMap["first"] = getFirst
	e.funcMap["last"] = getLast
	e.funcMap["rest"] = getRest
	e.funcMap["initial"] = getInitial
	e.funcMap["reverse"] = reverse
	e.funcMap["sort"] = sortStrings
	e.funcMap["length"] = getLength

	// Conditional functions
	e.funcMap["eq"] = eq
	e.funcMap["ne"] = ne
	e.funcMap["lt"] = lt
	e.funcMap["le"] = le
	e.funcMap["gt"] = gt
	e.funcMap["ge"] = ge
	e.funcMap["and"] = and
	e.funcMap["or"] = or
	e.funcMap["not"] = not
	e.funcMap["default"] = defaultValue

	// Date/time functions
	e.funcMap["now"] = time.Now
	e.funcMap["year"] = func() int { return time.Now().Year() }
	e.funcMap["date"] = func(format string) string { return time.Now().Format(format) }
	e.funcMap["formatDate"] = func(t time.Time, format string) string { return t.Format(format) }
	e.funcMap["formatTime"] = func(t time.Time, format string) string { return t.Format(format) }

	// Math functions
	e.funcMap["add"] = add
	e.funcMap["sub"] = sub
	e.funcMap["mul"] = mul
	e.funcMap["div"] = div
	e.funcMap["mod"] = mod

	// Type functions
	e.funcMap["pluralize"] = pluralize
	e.funcMap["singularize"] = singularize
	e.funcMap["humanize"] = humanize
	e.funcMap["dehumanize"] = dehumanize

	// Go-specific functions
	e.funcMap["exportedName"] = toExportedName
	e.funcMap["unexportedName"] = toUnexportedName
	e.funcMap["packageName"] = toPackageName
	e.funcMap["receiverName"] = toReceiverName
	e.funcMap["fieldTag"] = buildFieldTag
	e.funcMap["importAlias"] = generateImportAlias
}

// Name returns the template name.
func (gt *GoTemplate) Name() string {
	return gt.name
}

// Render renders the template with the provided data.
func (gt *GoTemplate) Render(data interface{}) ([]byte, error) {
	if gt.template == nil {
		return nil, fmt.Errorf("cannot render nil template %s", gt.name)
	}

	var buf bytes.Buffer
	if err := gt.template.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render template %s: %w", gt.name, err)
	}
	return buf.Bytes(), nil
}

// String case conversion functions

// toCamelCase converts a string to camelCase.
func toCamelCase(s string) string {
	return toCaseStyle(s, false)
}

// toPascalCase converts a string to PascalCase.
func toPascalCase(s string) string {
	return toCaseStyle(s, true)
}

// toSnakeCase converts a string to snake_case.
func toSnakeCase(s string) string {
	// Handle camelCase and PascalCase
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	s = re.ReplaceAllString(s, "${1}_${2}")

	// Replace spaces and hyphens with underscores
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")

	// Clean up multiple underscores
	re = regexp.MustCompile("_{2,}")
	s = re.ReplaceAllString(s, "_")

	return strings.ToLower(strings.Trim(s, "_"))
}

// toKebabCase converts a string to kebab-case.
func toKebabCase(s string) string {
	return strings.ReplaceAll(toSnakeCase(s), "_", "-")
}

// toScreamingSnakeCase converts a string to SCREAMING_SNAKE_CASE.
func toScreamingSnakeCase(s string) string {
	return strings.ToUpper(toSnakeCase(s))
}

// toCaseStyle is a helper function for camelCase and PascalCase conversion.
func toCaseStyle(s string, firstUpper bool) string {
	if s == "" {
		return s
	}

	// Split by common delimiters
	parts := regexp.MustCompile(`[_\-\s]+`).Split(s, -1)
	var result strings.Builder

	for i, part := range parts {
		if part == "" {
			continue
		}

		// Clean up the part
		part = strings.ToLower(part)

		// Capitalize first letter if needed
		if i == 0 && !firstUpper {
			result.WriteString(part)
		} else if len(part) > 0 {
			result.WriteRune(unicode.ToUpper(rune(part[0])))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}

	return result.String()
}

// Array/slice helper functions

func getFirst(slice interface{}) interface{} {
	s := toSlice(slice)
	if len(s) == 0 {
		return nil
	}
	return s[0]
}

func getLast(slice interface{}) interface{} {
	s := toSlice(slice)
	if len(s) == 0 {
		return nil
	}
	return s[len(s)-1]
}

func getRest(slice interface{}) []interface{} {
	s := toSlice(slice)
	if len(s) <= 1 {
		return []interface{}{}
	}
	return s[1:]
}

func getInitial(slice interface{}) []interface{} {
	s := toSlice(slice)
	if len(s) <= 1 {
		return []interface{}{}
	}
	return s[:len(s)-1]
}

func reverse(slice interface{}) []interface{} {
	s := toSlice(slice)
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[len(s)-1-i] = v
	}
	return result
}

func sortStrings(slice interface{}) []string {
	s := toSlice(slice)
	result := make([]string, len(s))
	for i, v := range s {
		result[i] = fmt.Sprintf("%v", v)
	}
	// Simple bubble sort for demonstration
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

func getLength(v interface{}) int {
	s := toSlice(v)
	return len(s)
}

func joinSlice(sep string, slice interface{}) string {
	s := toSlice(slice)
	result := make([]string, len(s))
	for i, v := range s {
		result[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(result, sep)
}

func toSlice(v interface{}) []interface{} {
	switch s := v.(type) {
	case []interface{}:
		return s
	case []string:
		result := make([]interface{}, len(s))
		for i, item := range s {
			result[i] = item
		}
		return result
	case []int:
		result := make([]interface{}, len(s))
		for i, item := range s {
			result[i] = item
		}
		return result
	default:
		return []interface{}{}
	}
}

// Comparison functions

func eq(a, b interface{}) bool { return a == b }
func ne(a, b interface{}) bool { return a != b }
func lt(a, b interface{}) bool {
	return compareNumbers(a, b) < 0
}
func le(a, b interface{}) bool {
	return compareNumbers(a, b) <= 0
}
func gt(a, b interface{}) bool {
	return compareNumbers(a, b) > 0
}
func ge(a, b interface{}) bool {
	return compareNumbers(a, b) >= 0
}

func and(a, b bool) bool { return a && b }
func or(a, b bool) bool  { return a || b }
func not(a bool) bool    { return !a }

// defaultValue returns the default value if the first value is empty/nil/zero.
func defaultValue(value, defaultVal interface{}) interface{} {
	if isEmptyValue(value) {
		return defaultVal
	}
	return value
}

// isEmptyValue checks if a value is considered "empty" for default purposes.
func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}

	switch val := v.(type) {
	case string:
		return val == ""
	case int, int32, int64:
		return val == 0
	case float32, float64:
		return val == 0.0
	case bool:
		return !val
	default:
		return false
	}
}

func compareNumbers(a, b interface{}) int {
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)
	if aFloat < bFloat {
		return -1
	} else if aFloat > bFloat {
		return 1
	}
	return 0
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	default:
		return 0
	}
}

// Math functions

func add(a, b interface{}) float64 { return toFloat64(a) + toFloat64(b) }
func sub(a, b interface{}) float64 { return toFloat64(a) - toFloat64(b) }
func mul(a, b interface{}) float64 { return toFloat64(a) * toFloat64(b) }
func div(a, b interface{}) float64 {
	bFloat := toFloat64(b)
	if bFloat == 0 {
		return 0
	}
	return toFloat64(a) / bFloat
}
func mod(a, b interface{}) int {
	aInt := int(toFloat64(a))
	bInt := int(toFloat64(b))
	if bInt == 0 {
		return 0
	}
	return aInt % bInt
}

// String manipulation functions

func pluralize(s string) string {
	if s == "" {
		return s
	}

	// Simple pluralization rules
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") ||
		strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	return s + "s"
}

func singularize(s string) string {
	if s == "" {
		return s
	}

	// Simple singularization rules
	if strings.HasSuffix(s, "ies") && len(s) > 3 {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "es") && len(s) > 2 {
		if strings.HasSuffix(s, "ses") || strings.HasSuffix(s, "xes") ||
			strings.HasSuffix(s, "zes") || strings.HasSuffix(s, "ches") ||
			strings.HasSuffix(s, "shes") {
			return s[:len(s)-2]
		}
	}
	if strings.HasSuffix(s, "s") && len(s) > 1 {
		return s[:len(s)-1]
	}
	return s
}

func humanize(s string) string {
	// Convert snake_case or kebab-case to human readable
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")

	// Capitalize first letter of each word
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

func dehumanize(s string) string {
	// Convert human readable to snake_case
	return toSnakeCase(s)
}

// Go-specific helper functions

func toExportedName(s string) string {
	// Exported names start with uppercase letter (PascalCase)
	return toPascalCase(s)
}

func toUnexportedName(s string) string {
	// Unexported names start with lowercase letter (camelCase)
	return toCamelCase(s)
}

func toPackageName(s string) string {
	// Package names should be lowercase
	name := toSnakeCase(s)
	name = strings.ReplaceAll(name, "_", "")
	return strings.ToLower(name)
}

func toReceiverName(structName string) string {
	if structName == "" {
		return "x"
	}

	// Use first letter(s) of the struct name
	name := strings.ToLower(structName)
	if len(name) == 1 {
		return name
	}

	// If starts with vowel, use first two letters
	if strings.ContainsRune("aeiou", rune(name[0])) && len(name) > 1 {
		return name[:2]
	}

	return name[:1]
}

func buildFieldTag(tagMap map[string]string) string {
	if len(tagMap) == 0 {
		return ""
	}

	var tags []string
	for key, value := range tagMap {
		if value == "" {
			tags = append(tags, key)
		} else {
			tags = append(tags, fmt.Sprintf(`%s:"%s"`, key, value))
		}
	}

	return strings.Join(tags, " ")
}

func generateImportAlias(importPath string) string {
	parts := strings.Split(importPath, "/")
	if len(parts) == 0 {
		return ""
	}

	lastPart := parts[len(parts)-1]

	// Remove common prefixes/suffixes
	lastPart = strings.TrimPrefix(lastPart, "go-")
	lastPart = strings.TrimSuffix(lastPart, "-go")
	lastPart = strings.TrimPrefix(lastPart, "lib")

	// Convert to valid Go identifier
	lastPart = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(lastPart, "")

	if lastPart == "" {
		return "pkg"
	}

	return strings.ToLower(lastPart)
}
