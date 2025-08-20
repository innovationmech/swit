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

// Package filesystem provides filesystem operations abstraction for switctl.
package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// OSFileSystem implements the FileSystem interface using the operating system's filesystem.
type OSFileSystem struct {
	basePath string
}

// NewOSFileSystem creates a new OS-based filesystem implementation.
func NewOSFileSystem(basePath string) *OSFileSystem {
	return &OSFileSystem{
		basePath: basePath,
	}
}

// WriteFile writes content to a file.
func (fs *OSFileSystem) WriteFile(path string, content []byte, perm int) error {
	fullPath := fs.resolvePath(path)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return os.WriteFile(fullPath, content, os.FileMode(perm))
}

// ReadFile reads content from a file.
func (fs *OSFileSystem) ReadFile(path string) ([]byte, error) {
	fullPath := fs.resolvePath(path)
	return os.ReadFile(fullPath)
}

// MkdirAll creates directories recursively.
func (fs *OSFileSystem) MkdirAll(path string, perm int) error {
	fullPath := fs.resolvePath(path)
	return os.MkdirAll(fullPath, os.FileMode(perm))
}

// Exists checks if a path exists.
func (fs *OSFileSystem) Exists(path string) bool {
	fullPath := fs.resolvePath(path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// Remove removes a file or directory.
func (fs *OSFileSystem) Remove(path string) error {
	fullPath := fs.resolvePath(path)
	return os.RemoveAll(fullPath)
}

// Copy copies a file or directory.
func (fs *OSFileSystem) Copy(src, dst string) error {
	srcPath := fs.resolvePath(src)
	dstPath := fs.resolvePath(dst)

	return fs.copyRecursive(srcPath, dstPath)
}

// resolvePath resolves a relative path against the base path.
func (fs *OSFileSystem) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if fs.basePath == "" {
		return path
	}
	return filepath.Join(fs.basePath, path)
}

// copyRecursive recursively copies files and directories.
func (fs *OSFileSystem) copyRecursive(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source %s: %w", src, err)
	}

	if srcInfo.IsDir() {
		return fs.copyDir(src, dst, srcInfo)
	}
	return fs.copyFile(src, dst, srcInfo)
}

// copyDir copies a directory and its contents.
func (fs *OSFileSystem) copyDir(src, dst string, srcInfo os.FileInfo) error {
	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dst, err)
	}

	// Read directory contents
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", src, err)
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if err := fs.copyRecursive(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// copyFile copies a single file.
func (fs *OSFileSystem) copyFile(src, dst string, srcInfo os.FileInfo) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	// Copy file contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Set file permissions
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", dst, err)
	}

	return nil
}

// InMemoryFileSystem implements the FileSystem interface using in-memory storage.
// Useful for testing and temporary operations.
type InMemoryFileSystem struct {
	files map[string][]byte
	dirs  map[string]bool
}

// NewInMemoryFileSystem creates a new in-memory filesystem implementation.
func NewInMemoryFileSystem() *InMemoryFileSystem {
	return &InMemoryFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

// WriteFile writes content to a file in memory.
func (fs *InMemoryFileSystem) WriteFile(path string, content []byte, perm int) error {
	normalizedPath := fs.normalizePath(path)

	// Ensure parent directories exist
	dir := filepath.Dir(normalizedPath)
	if err := fs.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Copy content to avoid external modifications
	fileCopy := make([]byte, len(content))
	copy(fileCopy, content)

	fs.files[normalizedPath] = fileCopy
	return nil
}

// ReadFile reads content from a file in memory.
func (fs *InMemoryFileSystem) ReadFile(path string) ([]byte, error) {
	normalizedPath := fs.normalizePath(path)

	content, exists := fs.files[normalizedPath]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	// Return a copy to avoid external modifications
	contentCopy := make([]byte, len(content))
	copy(contentCopy, content)

	return contentCopy, nil
}

// MkdirAll creates directories recursively in memory.
func (fs *InMemoryFileSystem) MkdirAll(path string, perm int) error {
	normalizedPath := fs.normalizePath(path)

	// Create all parent directories
	parts := strings.Split(normalizedPath, "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}
		fs.dirs[currentPath] = true
	}

	return nil
}

// Exists checks if a path exists in memory.
func (fs *InMemoryFileSystem) Exists(path string) bool {
	normalizedPath := fs.normalizePath(path)

	// Check if it's a file
	if _, exists := fs.files[normalizedPath]; exists {
		return true
	}

	// Check if it's a directory
	if _, exists := fs.dirs[normalizedPath]; exists {
		return true
	}

	return false
}

// Remove removes a file or directory from memory.
func (fs *InMemoryFileSystem) Remove(path string) error {
	normalizedPath := fs.normalizePath(path)

	// Remove file if it exists
	delete(fs.files, normalizedPath)

	// Remove directory and all subdirectories/files
	delete(fs.dirs, normalizedPath)

	// Remove all files and directories under this path
	prefix := normalizedPath + "/"
	for filePath := range fs.files {
		if strings.HasPrefix(filePath, prefix) {
			delete(fs.files, filePath)
		}
	}

	for dirPath := range fs.dirs {
		if strings.HasPrefix(dirPath, prefix) {
			delete(fs.dirs, dirPath)
		}
	}

	return nil
}

// Copy copies a file or directory in memory.
func (fs *InMemoryFileSystem) Copy(src, dst string) error {
	srcPath := fs.normalizePath(src)
	dstPath := fs.normalizePath(dst)

	// Check if source is a file
	if content, exists := fs.files[srcPath]; exists {
		return fs.WriteFile(dstPath, content, 0644)
	}

	// Check if source is a directory
	if _, exists := fs.dirs[srcPath]; exists {
		// Create destination directory
		if err := fs.MkdirAll(dstPath, 0755); err != nil {
			return err
		}

		// Copy all files under source directory
		prefix := srcPath + "/"
		for filePath, content := range fs.files {
			if strings.HasPrefix(filePath, prefix) {
				relPath := strings.TrimPrefix(filePath, prefix)
				dstFilePath := dstPath + "/" + relPath
				if err := fs.WriteFile(dstFilePath, content, 0644); err != nil {
					return err
				}
			}
		}

		// Copy all subdirectories
		for dirPath := range fs.dirs {
			if strings.HasPrefix(dirPath, prefix) {
				relPath := strings.TrimPrefix(dirPath, prefix)
				dstDirPath := dstPath + "/" + relPath
				if err := fs.MkdirAll(dstDirPath, 0755); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return fmt.Errorf("source path not found: %s", src)
}

// normalizePath normalizes a path for consistent storage.
func (fs *InMemoryFileSystem) normalizePath(path string) string {
	// Convert to forward slashes and remove leading slash
	normalized := filepath.ToSlash(path)
	normalized = strings.TrimPrefix(normalized, "/")
	return normalized
}

// GetFiles returns all files in the in-memory filesystem.
func (fs *InMemoryFileSystem) GetFiles() map[string][]byte {
	result := make(map[string][]byte)
	for path, content := range fs.files {
		contentCopy := make([]byte, len(content))
		copy(contentCopy, content)
		result[path] = contentCopy
	}
	return result
}

// GetDirectories returns all directories in the in-memory filesystem.
func (fs *InMemoryFileSystem) GetDirectories() []string {
	var dirs []string
	for dir := range fs.dirs {
		dirs = append(dirs, dir)
	}
	return dirs
}

// Clear clears all files and directories from the in-memory filesystem.
func (fs *InMemoryFileSystem) Clear() {
	fs.files = make(map[string][]byte)
	fs.dirs = make(map[string]bool)
}

// Ensure both implementations satisfy the FileSystem interface
var (
	_ interfaces.FileSystem = (*OSFileSystem)(nil)
	_ interfaces.FileSystem = (*InMemoryFileSystem)(nil)
)
