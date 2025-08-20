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

package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOSFileSystem_BasicOperations(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "filesystem-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fs := NewOSFileSystem(tmpDir)

	// Test WriteFile and ReadFile
	testContent := []byte("Hello, World!")
	testPath := "test.txt"

	err = fs.WriteFile(testPath, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	if !fs.Exists(testPath) {
		t.Error("File should exist after writing")
	}

	content, err := fs.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Expected %q, got %q", string(testContent), string(content))
	}

	// Test MkdirAll
	dirPath := "test/nested/directory"
	err = fs.MkdirAll(dirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if !fs.Exists(dirPath) {
		t.Error("Directory should exist after creation")
	}

	// Test Remove
	err = fs.Remove(testPath)
	if err != nil {
		t.Fatalf("Failed to remove file: %v", err)
	}

	if fs.Exists(testPath) {
		t.Error("File should not exist after removal")
	}
}

func TestInMemoryFileSystem_BasicOperations(t *testing.T) {
	fs := NewInMemoryFileSystem()

	// Test WriteFile and ReadFile
	testContent := []byte("Hello, In-Memory World!")
	testPath := "memory/test.txt"

	err := fs.WriteFile(testPath, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	if !fs.Exists(testPath) {
		t.Error("File should exist after writing")
	}

	content, err := fs.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Expected %q, got %q", string(testContent), string(content))
	}

	// Test MkdirAll
	dirPath := "test/nested/directory"
	err = fs.MkdirAll(dirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if !fs.Exists(dirPath) {
		t.Error("Directory should exist after creation")
	}

	// Test GetFiles and GetDirectories
	files := fs.GetFiles()
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	dirs := fs.GetDirectories()
	if len(dirs) == 0 {
		t.Error("Expected at least one directory")
	}

	// Test Remove
	err = fs.Remove(testPath)
	if err != nil {
		t.Fatalf("Failed to remove file: %v", err)
	}

	if fs.Exists(testPath) {
		t.Error("File should not exist after removal")
	}

	// Test Clear
	fs.Clear()
	files = fs.GetFiles()
	dirs = fs.GetDirectories()
	if len(files) != 0 || len(dirs) != 0 {
		t.Error("File system should be empty after clear")
	}
}

func TestInMemoryFileSystem_Copy(t *testing.T) {
	fs := NewInMemoryFileSystem()

	// Create source file
	srcContent := []byte("Source content")
	srcPath := "source.txt"
	err := fs.WriteFile(srcPath, srcContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Test file copy
	dstPath := "destination.txt"
	err = fs.Copy(srcPath, dstPath)
	if err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify destination exists and has same content
	if !fs.Exists(dstPath) {
		t.Error("Destination file should exist after copy")
	}

	dstContent, err := fs.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != string(srcContent) {
		t.Errorf("Expected %q, got %q", string(srcContent), string(dstContent))
	}

	// Test directory copy
	err = fs.MkdirAll("src/subdir", 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	err = fs.WriteFile("src/file1.txt", []byte("file1"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	err = fs.WriteFile("src/subdir/file2.txt", []byte("file2"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	err = fs.Copy("src", "dst")
	if err != nil {
		t.Fatalf("Failed to copy directory: %v", err)
	}

	// Verify directory copy
	if !fs.Exists("dst") {
		t.Error("Destination directory should exist")
	}

	if !fs.Exists("dst/file1.txt") {
		t.Error("Copied file should exist")
	}

	if !fs.Exists("dst/subdir/file2.txt") {
		t.Error("Copied subdirectory file should exist")
	}

	content1, _ := fs.ReadFile("dst/file1.txt")
	if string(content1) != "file1" {
		t.Errorf("Expected 'file1', got %q", string(content1))
	}

	content2, _ := fs.ReadFile("dst/subdir/file2.txt")
	if string(content2) != "file2" {
		t.Errorf("Expected 'file2', got %q", string(content2))
	}
}

func TestOSFileSystem_Copy(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "filesystem-copy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fs := NewOSFileSystem(tmpDir)

	// Create source file
	srcContent := []byte("Source content for OS filesystem")
	srcPath := "source.txt"
	err = fs.WriteFile(srcPath, srcContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Test file copy
	dstPath := "destination.txt"
	err = fs.Copy(srcPath, dstPath)
	if err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify destination exists and has same content
	if !fs.Exists(dstPath) {
		t.Error("Destination file should exist after copy")
	}

	dstContent, err := fs.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != string(srcContent) {
		t.Errorf("Expected %q, got %q", string(srcContent), string(dstContent))
	}

	// Verify both files have same permissions
	srcInfo, _ := os.Stat(filepath.Join(tmpDir, srcPath))
	dstInfo, _ := os.Stat(filepath.Join(tmpDir, dstPath))
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("File permissions should match: src=%v, dst=%v", srcInfo.Mode(), dstInfo.Mode())
	}
}
