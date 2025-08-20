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

package template

import (
	"crypto/md5"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Store provides template storage and management capabilities.
type Store struct {
	baseDir     string
	embeddedFS  embed.FS
	cache       map[string]*CachedTemplate
	versions    map[string]string
	watchers    map[string]*FileWatcher
	mu          sync.RWMutex
	hotReload   bool
	useEmbedded bool
}

// CachedTemplate represents a cached template with metadata.
type CachedTemplate struct {
	Content     string
	Path        string
	Hash        string
	ModTime     time.Time
	AccessTime  time.Time
	AccessCount int64
	Size        int64
}

// FileWatcher watches for file changes to enable hot reload.
type FileWatcher struct {
	Path     string
	ModTime  time.Time
	Size     int64
	Callback func(string)
}

// TemplateInfo provides information about a template.
type TemplateInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	Hash        string    `json:"hash"`
	AccessCount int64     `json:"access_count"`
	AccessTime  time.Time `json:"access_time"`
	Version     string    `json:"version,omitempty"`
}

// NewStore creates a new template store for the specified directory.
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:     baseDir,
		cache:       make(map[string]*CachedTemplate),
		versions:    make(map[string]string),
		watchers:    make(map[string]*FileWatcher),
		hotReload:   false,
		useEmbedded: false,
	}
}

// NewStoreWithEmbedded creates a new template store using embedded filesystem.
func NewStoreWithEmbedded(embeddedFS embed.FS) *Store {
	return &Store{
		embeddedFS:  embeddedFS,
		cache:       make(map[string]*CachedTemplate),
		versions:    make(map[string]string),
		watchers:    make(map[string]*FileWatcher),
		hotReload:   false,
		useEmbedded: true,
	}
}

// GetTemplate retrieves a template by name, loading from cache or filesystem.
func (s *Store) GetTemplate(name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check cache first
	if cached, exists := s.cache[name]; exists {
		// Update access information
		cached.AccessTime = time.Now()
		cached.AccessCount++

		// Check for hot reload if enabled
		if s.hotReload && !s.useEmbedded {
			if s.shouldReload(name, cached) {
				return s.loadFromFilesystem(name)
			}
		}

		return cached.Content, nil
	}

	// Load from filesystem or embedded
	if s.useEmbedded {
		return s.loadFromEmbedded(name)
	}
	return s.loadFromFilesystem(name)
}

// SetTemplate stores a template with the given name and content.
func (s *Store) SetTemplate(name, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := s.calculateHash(content)
	now := time.Now()

	cached := &CachedTemplate{
		Content:     content,
		Path:        s.getTemplatePath(name),
		Hash:        hash,
		ModTime:     now,
		AccessTime:  now,
		AccessCount: 0,
		Size:        int64(len(content)),
	}

	s.cache[name] = cached

	// Write to filesystem if not using embedded
	if !s.useEmbedded && s.baseDir != "" {
		return s.writeToFilesystem(name, content)
	}

	return nil
}

// RemoveTemplate removes a template from the cache and optionally from filesystem.
func (s *Store) RemoveTemplate(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.cache, name)
	delete(s.versions, name)

	// Remove watcher if exists
	if watcher, exists := s.watchers[name]; exists {
		delete(s.watchers, name)
		_ = watcher // Placeholder for cleanup
	}

	// Remove from filesystem if not using embedded
	if !s.useEmbedded && s.baseDir != "" {
		path := s.getTemplatePath(name)
		if _, err := os.Stat(path); err == nil {
			return os.Remove(path)
		}
	}

	return nil
}

// ListTemplates returns a list of all available templates.
func (s *Store) ListTemplates() ([]TemplateInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var templates []TemplateInfo

	// Add cached templates
	for name, cached := range s.cache {
		info := TemplateInfo{
			Name:        name,
			Path:        cached.Path,
			Size:        cached.Size,
			ModTime:     cached.ModTime,
			Hash:        cached.Hash,
			AccessCount: cached.AccessCount,
			AccessTime:  cached.AccessTime,
			Version:     s.versions[name],
		}
		templates = append(templates, info)
	}

	// If using embedded filesystem, scan embedded templates
	if s.useEmbedded {
		err := fs.WalkDir(s.embeddedFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
				return nil
			}

			// Convert path to template name
			name := strings.TrimSuffix(path, ".tmpl")
			name = strings.TrimPrefix(name, "templates/")

			// Skip if already in cache
			if _, exists := s.cache[name]; exists {
				return nil
			}

			// Get file info
			fileInfo, err := d.Info()
			if err != nil {
				return err
			}

			info := TemplateInfo{
				Name:    name,
				Path:    path,
				Size:    fileInfo.Size(),
				ModTime: fileInfo.ModTime(),
			}
			templates = append(templates, info)

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scan embedded templates: %w", err)
		}
	}

	// If using filesystem, scan template directory
	if !s.useEmbedded && s.baseDir != "" {
		err := filepath.WalkDir(s.baseDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
				return nil
			}

			// Convert path to template name
			relPath, err := filepath.Rel(s.baseDir, path)
			if err != nil {
				return err
			}
			name := strings.TrimSuffix(relPath, ".tmpl")

			// Skip if already in cache
			if _, exists := s.cache[name]; exists {
				return nil
			}

			// Get file info
			fileInfo, err := d.Info()
			if err != nil {
				return err
			}

			info := TemplateInfo{
				Name:    name,
				Path:    path,
				Size:    fileInfo.Size(),
				ModTime: fileInfo.ModTime(),
			}
			templates = append(templates, info)

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scan template directory: %w", err)
		}
	}

	return templates, nil
}

// GetTemplateInfo returns information about a specific template.
func (s *Store) GetTemplateInfo(name string) (*TemplateInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if cached, exists := s.cache[name]; exists {
		return &TemplateInfo{
			Name:        name,
			Path:        cached.Path,
			Size:        cached.Size,
			ModTime:     cached.ModTime,
			Hash:        cached.Hash,
			AccessCount: cached.AccessCount,
			AccessTime:  cached.AccessTime,
			Version:     s.versions[name],
		}, nil
	}

	// Try to get info from filesystem
	path := s.getTemplatePath(name)
	if s.useEmbedded {
		// Check embedded filesystem
		if fileInfo, err := fs.Stat(s.embeddedFS, path); err == nil {
			return &TemplateInfo{
				Name:    name,
				Path:    path,
				Size:    fileInfo.Size(),
				ModTime: fileInfo.ModTime(),
			}, nil
		}
	} else if s.baseDir != "" {
		if fileInfo, err := os.Stat(path); err == nil {
			return &TemplateInfo{
				Name:    name,
				Path:    path,
				Size:    fileInfo.Size(),
				ModTime: fileInfo.ModTime(),
			}, nil
		}
	}

	return nil, fmt.Errorf("template %s not found", name)
}

// SetVersion sets the version for a template.
func (s *Store) SetVersion(name, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.versions[name] = version
}

// GetVersion returns the version of a template.
func (s *Store) GetVersion(name string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.versions[name]
}

// EnableHotReload enables hot reload functionality for filesystem-based templates.
func (s *Store) EnableHotReload() error {
	if s.useEmbedded {
		return fmt.Errorf("hot reload not supported for embedded templates")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.hotReload = true

	return nil
}

// DisableHotReload disables hot reload functionality.
func (s *Store) DisableHotReload() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hotReload = false

	// Clear all watchers
	for name := range s.watchers {
		delete(s.watchers, name)
	}
}

// ClearCache clears all cached templates.
func (s *Store) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]*CachedTemplate)
}

// GetCacheStats returns cache statistics.
func (s *Store) GetCacheStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalSize int64
	var totalAccess int64
	for _, cached := range s.cache {
		totalSize += cached.Size
		totalAccess += cached.AccessCount
	}

	return map[string]interface{}{
		"cached_templates": len(s.cache),
		"total_size":       totalSize,
		"total_access":     totalAccess,
		"hot_reload":       s.hotReload,
		"use_embedded":     s.useEmbedded,
	}
}

// loadFromEmbedded loads a template from the embedded filesystem.
func (s *Store) loadFromEmbedded(name string) (string, error) {
	path := s.getEmbeddedTemplatePath(name)

	content, err := fs.ReadFile(s.embeddedFS, path)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded template %s: %w", name, err)
	}

	// Cache the template
	hash := s.calculateHash(string(content))
	now := time.Now()

	cached := &CachedTemplate{
		Content:     string(content),
		Path:        path,
		Hash:        hash,
		ModTime:     now,
		AccessTime:  now,
		AccessCount: 1,
		Size:        int64(len(content)),
	}

	s.cache[name] = cached

	return string(content), nil
}

// loadFromFilesystem loads a template from the filesystem.
func (s *Store) loadFromFilesystem(name string) (string, error) {
	path := s.getTemplatePath(name)

	// Check if file exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("template %s not found: %w", name, err)
	}

	// Read file content
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open template %s: %w", name, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", name, err)
	}

	// Cache the template
	hash := s.calculateHash(string(content))

	cached := &CachedTemplate{
		Content:     string(content),
		Path:        path,
		Hash:        hash,
		ModTime:     fileInfo.ModTime(),
		AccessTime:  time.Now(),
		AccessCount: 1,
		Size:        fileInfo.Size(),
	}

	s.cache[name] = cached

	// Set up file watcher if hot reload is enabled
	if s.hotReload {
		s.watchers[name] = &FileWatcher{
			Path:    path,
			ModTime: fileInfo.ModTime(),
			Size:    fileInfo.Size(),
		}
	}

	return string(content), nil
}

// writeToFilesystem writes a template to the filesystem.
func (s *Store) writeToFilesystem(name, content string) error {
	path := s.getTemplatePath(name)

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write template %s: %w", name, err)
	}

	return nil
}

// shouldReload checks if a template should be reloaded based on filesystem changes.
func (s *Store) shouldReload(name string, cached *CachedTemplate) bool {
	watcher, exists := s.watchers[name]
	if !exists {
		return false
	}

	// Check if file has been modified
	fileInfo, err := os.Stat(watcher.Path)
	if err != nil {
		return false
	}

	return fileInfo.ModTime().After(watcher.ModTime) || fileInfo.Size() != watcher.Size
}

// getTemplatePath returns the full path for a template name.
func (s *Store) getTemplatePath(name string) string {
	// If name already has .tmpl extension, use it as-is
	filename := name
	if !strings.HasSuffix(name, ".tmpl") {
		filename = name + ".tmpl"
	}

	if s.baseDir == "" {
		return filename
	}
	return filepath.Join(s.baseDir, filename)
}

// getEmbeddedTemplatePath returns the embedded path for a template name.
func (s *Store) getEmbeddedTemplatePath(name string) string {
	// If name already has .tmpl extension, use it as-is
	filename := name
	if !strings.HasSuffix(name, ".tmpl") {
		filename = name + ".tmpl"
	}
	return "templates/" + filename
}

// calculateHash calculates MD5 hash of the content.
func (s *Store) calculateHash(content string) string {
	h := md5.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// PreloadTemplates preloads all templates from the template directory.
func (s *Store) PreloadTemplates() error {
	if s.useEmbedded {
		return s.preloadEmbeddedTemplates()
	}
	return s.preloadFilesystemTemplates()
}

// preloadEmbeddedTemplates preloads all embedded templates.
func (s *Store) preloadEmbeddedTemplates() error {
	return fs.WalkDir(s.embeddedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Convert path to template name
		name := strings.TrimSuffix(path, ".tmpl")
		name = strings.TrimPrefix(name, "templates/")

		// Load template
		_, err = s.GetTemplate(name)
		return err
	})
}

// preloadFilesystemTemplates preloads all filesystem templates.
func (s *Store) preloadFilesystemTemplates() error {
	if s.baseDir == "" {
		return nil
	}

	return filepath.WalkDir(s.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Convert path to template name
		relPath, err := filepath.Rel(s.baseDir, path)
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(relPath, ".tmpl")

		// Load template
		_, err = s.GetTemplate(name)
		return err
	})
}

// ExportTemplates exports all templates to a directory.
func (s *Store) ExportTemplates(exportDir string) error {
	templates, err := s.ListTemplates()
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	for _, tmpl := range templates {
		content, err := s.GetTemplate(tmpl.Name)
		if err != nil {
			return fmt.Errorf("failed to get template %s: %w", tmpl.Name, err)
		}

		exportPath := filepath.Join(exportDir, tmpl.Name+".tmpl")
		exportDir := filepath.Dir(exportPath)

		if err := os.MkdirAll(exportDir, 0755); err != nil {
			return fmt.Errorf("failed to create export directory %s: %w", exportDir, err)
		}

		if err := os.WriteFile(exportPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to export template %s: %w", tmpl.Name, err)
		}
	}

	return nil
}

// ImportTemplates imports templates from a directory.
func (s *Store) ImportTemplates(importDir string) error {
	return filepath.WalkDir(importDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Read template content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		// Convert path to template name
		relPath, err := filepath.Rel(importDir, path)
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(relPath, ".tmpl")

		// Store template
		return s.SetTemplate(name, string(content))
	})
}
