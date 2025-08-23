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

package checker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkStyleCheckerPerformance benchmarks the style checker performance
func BenchmarkStyleCheckerPerformance(b *testing.B) {
	tempDir := createBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	checker := NewStyleChecker(tempDir, StyleCheckerOptions{
		EnableGofmt:          true,
		EnableGolangciLint:   false, // Disable for faster benchmarks
		EnableImportCheck:    true,
		EnableCopyrightCheck: true,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStyleCheckerConcurrent benchmarks concurrent style checking
func BenchmarkStyleCheckerConcurrent(b *testing.B) {
	tempDir := createBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			checker := NewStyleChecker(tempDir, StyleCheckerOptions{
				EnableGofmt:          true,
				EnableGolangciLint:   false,
				EnableImportCheck:    true,
				EnableCopyrightCheck: true,
			})

			ctx := context.Background()
			_, err := checker.Check(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkTestRunnerPerformance benchmarks the test runner performance
func BenchmarkTestRunnerPerformance(b *testing.B) {
	tempDir := createBenchmarkProjectWithTests(b)
	defer os.RemoveAll(tempDir)

	runner := NewTestRunner(tempDir, TestRunnerOptions{
		EnableCoverage:      false, // Disable coverage for faster benchmarks
		EnableRaceDetection: false,
		CoverageThreshold:   0,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		packages, err := runner.findTestPackages()
		if err != nil {
			b.Fatal(err)
		}

		ctx := context.Background()
		_, err = runner.RunTests(ctx, packages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTestRunnerWithCoverage benchmarks test runner with coverage enabled
func BenchmarkTestRunnerWithCoverage(b *testing.B) {
	tempDir := createBenchmarkProjectWithTests(b)
	defer os.RemoveAll(tempDir)

	runner := NewTestRunner(tempDir, TestRunnerOptions{
		EnableCoverage:      true,
		EnableRaceDetection: false,
		CoverageThreshold:   80.0,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		packages, err := runner.findTestPackages()
		if err != nil {
			b.Fatal(err)
		}

		ctx := context.Background()
		_, err = runner.RunTests(ctx, packages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSecurityCheckerPerformance benchmarks the security checker performance
func BenchmarkSecurityCheckerPerformance(b *testing.B) {
	tempDir := createBenchmarkProjectWithSecurityIssues(b)
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{messages: make([]LogMessage, 0)}
	checker := NewSecurityChecker(tempDir, logger)

	// Configure for faster benchmarks
	checker.SetConfig(SecurityConfig{
		GosecEnabled:      false, // Disable external tools for benchmarks
		NancyEnabled:      false,
		TrivyEnabled:      false,
		SeverityThreshold: "medium",
		MaxIssues:         20,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConfigChecker benchmarks the config checker performance
func BenchmarkConfigChecker(b *testing.B) {
	tempDir := createBenchmarkProjectWithConfigs(b)
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{messages: make([]LogMessage, 0)}
	checker := NewConfigChecker(tempDir, logger)

	// Configure for benchmarks
	checker.SetConfig(ConfigValidationConfig{
		YAMLValidation:  true,
		JSONValidation:  true,
		ProtoValidation: false, // Disable for faster benchmarks
		TOMLValidation:  false,
		StrictMode:      false,
		CheckKeys:       true,
		CheckValues:     true,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCompositeChecker benchmarks the full composite checker
func BenchmarkCompositeChecker(b *testing.B) {
	tempDir := createComprehensiveBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{messages: make([]LogMessage, 0)}
	checker := NewCompositeChecker(tempDir, logger)

	// Configure for faster benchmarks
	checker.SetStyleConfig(StyleCheckerOptions{
		EnableGofmt:          true,
		EnableGolangciLint:   false,
		EnableImportCheck:    true,
		EnableCopyrightCheck: false,
	})

	checker.SetTestConfig(TestRunnerOptions{
		EnableCoverage:      false,
		EnableRaceDetection: false,
		CoverageThreshold:   0,
	})

	checker.SetSecurityConfig(SecurityConfig{
		GosecEnabled:      false,
		NancyEnabled:      false,
		TrivyEnabled:      false,
		SeverityThreshold: "high",
		MaxIssues:         10,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark different checker methods
		b.Run("style", func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				_ = checker.CheckCodeStyle()
			}
		})

		b.Run("tests", func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				_ = checker.RunTests()
			}
		})

		b.Run("security", func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				_ = checker.CheckSecurity()
			}
		})

		break // Only run once for the overall benchmark
	}
}

// BenchmarkManager benchmarks the checker manager performance
func BenchmarkManager(b *testing.B) {
	tempDir := createBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{messages: make([]LogMessage, 0)}
	manager := NewManager(logger)
	manager.SetTimeout(30 * time.Second)
	manager.SetParallel(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewCompositeChecker(tempDir, logger)
		manager.RegisterChecker(checker)
	}
}

// BenchmarkConcurrentCheckers benchmarks concurrent checker operations
func BenchmarkConcurrentCheckers(b *testing.B) {
	tempDir := createBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{messages: make([]LogMessage, 0)}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			checker := NewCompositeChecker(tempDir, logger)

			// Run different checks concurrently
			var wg sync.WaitGroup
			wg.Add(3)

			go func() {
				defer wg.Done()
				checker.CheckCodeStyle()
			}()

			go func() {
				defer wg.Done()
				checker.CheckSecurity()
			}()

			go func() {
				defer wg.Done()
				checker.ValidateConfig()
			}()

			wg.Wait()
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	tempDir := createBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{messages: make([]LogMessage, 0)}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker := NewCompositeChecker(tempDir, logger)
		result := checker.CheckCodeStyle()
		_ = result // Prevent optimization
	}
}

// BenchmarkLargeProject benchmarks performance with a large project
func BenchmarkLargeProject(b *testing.B) {
	tempDir := createLargeBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{messages: make([]LogMessage, 0)}
	checker := NewCompositeChecker(tempDir, logger)

	// Configure for reasonable performance with large projects
	checker.SetStyleConfig(StyleCheckerOptions{
		EnableGofmt:          true,
		EnableGolangciLint:   false,
		EnableImportCheck:    false,
		EnableCopyrightCheck: false,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := checker.CheckCodeStyle()
		_ = result
	}
}

// BenchmarkDifferentProjectSizes benchmarks performance across different project sizes
func BenchmarkDifferentProjectSizes(b *testing.B) {
	sizes := []struct {
		name     string
		numFiles int
		numLines int
	}{
		{"small", 5, 50},
		{"medium", 20, 200},
		{"large", 100, 1000},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			tempDir := createProjectOfSize(b, size.numFiles, size.numLines)
			defer os.RemoveAll(tempDir)

			logger := &MockLogger{messages: make([]LogMessage, 0)}
			checker := NewCompositeChecker(tempDir, logger)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := checker.CheckCodeStyle()
				_ = result
			}
		})
	}
}

// BenchmarkContextCancellation benchmarks performance under context cancellation
func BenchmarkContextCancellation(b *testing.B) {
	tempDir := createBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewStyleChecker(tempDir, StyleCheckerOptions{})

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		_, err := checker.Check(ctx)
		cancel()

		// Expect either success or context cancellation
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			b.Fatal(err)
		}
	}
}

// Helper functions for creating benchmark projects

func createBenchmarkProject(b *testing.B) string {
	tempDir, err := os.MkdirTemp("", "bench-*")
	if err != nil {
		b.Fatal(err)
	}

	// Create go.mod
	goMod := `module benchmark/test

go 1.19

require (
    github.com/stretchr/testify v1.8.0
)
`
	writeFile(b, tempDir, "go.mod", goMod)

	// Create main.go
	mainGo := `package main

import (
    "fmt"
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/", handleRoot)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World!")
}
`
	writeFile(b, tempDir, "main.go", mainGo)

	// Create package files
	for i := 0; i < 5; i++ {
		pkgContent := fmt.Sprintf(`package pkg%d

import "fmt"

func Function%d() {
    fmt.Printf("Function %d called\n", %d)
}
`, i, i, i, i)

		pkgDir := filepath.Join(tempDir, fmt.Sprintf("pkg%d", i))
		os.MkdirAll(pkgDir, 0755)
		writeFile(b, pkgDir, fmt.Sprintf("pkg%d.go", i), pkgContent)
	}

	return tempDir
}

func createBenchmarkProjectWithTests(b *testing.B) string {
	tempDir := createBenchmarkProject(b)

	// Add test files
	for i := 0; i < 3; i++ {
		testContent := fmt.Sprintf(`package pkg%d

import "testing"

func TestFunction%d(t *testing.T) {
    Function%d()
    // Simple test
}

func BenchmarkFunction%d(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Function%d()
    }
}
`, i, i, i, i, i)

		pkgDir := filepath.Join(tempDir, fmt.Sprintf("pkg%d", i))
		writeFile(b, pkgDir, fmt.Sprintf("pkg%d_test.go", i), testContent)
	}

	return tempDir
}

func createBenchmarkProjectWithSecurityIssues(b *testing.B) string {
	tempDir := createBenchmarkProject(b)

	// Create files with potential security issues
	securityCode := `package security

import (
    "crypto/md5"
    "fmt"
    "net/http"
)

func WeakHash() {
    hasher := md5.New()
    hasher.Write([]byte("data"))
}

func PotentialIssue() {
    password := "hardcoded123"
    fmt.Println(password)
}

func HTTPUsage() {
    resp, err := http.Get("http://example.com")
    if err != nil {
        return
    }
    defer resp.Body.Close()
}
`

	securityDir := filepath.Join(tempDir, "security")
	os.MkdirAll(securityDir, 0755)
	writeFile(b, securityDir, "security.go", securityCode)

	return tempDir
}

func createBenchmarkProjectWithConfigs(b *testing.B) string {
	tempDir := createBenchmarkProject(b)

	// Create various config files
	configs := map[string]string{
		"config.yaml": `
server:
  port: 8080
  host: localhost
database:
  type: mysql
  host: localhost
  port: 3306
`,
		"app.json": `{
  "app": {
    "name": "benchmark-app",
    "version": "1.0.0"
  },
  "features": {
    "enabled": true
  }
}`,
		"docker-compose.yaml": `
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
`,
	}

	for filename, content := range configs {
		writeFile(b, tempDir, filename, content)
	}

	return tempDir
}

func createComprehensiveBenchmarkProject(b *testing.B) string {
	tempDir := createBenchmarkProject(b)

	// Add tests
	tempDir2 := createBenchmarkProjectWithTests(b)
	copyFiles(tempDir2, tempDir)
	os.RemoveAll(tempDir2)

	// Add configs
	tempDir3 := createBenchmarkProjectWithConfigs(b)
	copyFiles(tempDir3, tempDir)
	os.RemoveAll(tempDir3)

	// Add security issues
	tempDir4 := createBenchmarkProjectWithSecurityIssues(b)
	copyFiles(tempDir4, tempDir)
	os.RemoveAll(tempDir4)

	return tempDir
}

func createLargeBenchmarkProject(b *testing.B) string {
	tempDir := createBenchmarkProject(b)

	// Create many more files
	for i := 0; i < 50; i++ {
		pkgContent := fmt.Sprintf(`package large%d

import (
    "context"
    "fmt"
    "time"
)

type Service%d struct {
    ID   int
    Name string
}

func NewService%d() *Service%d {
    return &Service%d{ID: %d, Name: "service_%d"}
}

func (s *Service%d) Process(ctx context.Context) error {
    fmt.Printf("Processing service %%d\n", s.ID)
    return nil
}

func (s *Service%d) Start() error {
    time.Sleep(time.Millisecond)
    return nil
}

func (s *Service%d) Stop() error {
    return nil
}
`, i, i, i, i, i, i, i, i, i, i)

		pkgDir := filepath.Join(tempDir, fmt.Sprintf("large%d", i))
		os.MkdirAll(pkgDir, 0755)
		writeFile(b, pkgDir, fmt.Sprintf("service%d.go", i), pkgContent)
	}

	return tempDir
}

func createProjectOfSize(b *testing.B, numFiles, numLines int) string {
	tempDir, err := os.MkdirTemp("", "size-bench-*")
	if err != nil {
		b.Fatal(err)
	}

	writeFile(b, tempDir, "go.mod", "module size/test\ngo 1.19\n")

	for i := 0; i < numFiles; i++ {
		var content string
		content += fmt.Sprintf("package file%d\n\n", i)

		for j := 0; j < numLines; j++ {
			content += fmt.Sprintf("// Line %d in file %d\n", j, i)
			if j%10 == 0 {
				content += fmt.Sprintf("func Function%d_%d() {\n", i, j)
				content += fmt.Sprintf("    // Function body for %d_%d\n", i, j)
				content += "}\n\n"
			}
		}

		writeFile(b, tempDir, fmt.Sprintf("file%d.go", i), content)
	}

	return tempDir
}

func writeFile(b *testing.B, dir, filename, content string) {
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
	if err != nil {
		b.Fatal(err)
	}
}

func copyFiles(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, srcFile, info.Mode())
	})
}

// Performance monitoring benchmarks

// BenchmarkMemoryUsage measures memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	tempDir := createBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{messages: make([]LogMessage, 0)}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	initialAlloc := memStats.Alloc

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		checker := NewCompositeChecker(tempDir, logger)
		result := checker.CheckCodeStyle()
		_ = result

		// Force garbage collection every 100 iterations
		if i%100 == 0 {
			runtime.GC()
		}
	}

	runtime.ReadMemStats(&memStats)
	finalAlloc := memStats.Alloc

	b.Logf("Memory growth: %d bytes", finalAlloc-initialAlloc)
}

// BenchmarkGoroutineUsage measures goroutine usage
func BenchmarkGoroutineUsage(b *testing.B) {
	tempDir := createBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{messages: make([]LogMessage, 0)}
	initialGoroutines := runtime.NumGoroutine()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker := NewCompositeChecker(tempDir, logger)
		result := checker.CheckCodeStyle()
		_ = result
	}

	finalGoroutines := runtime.NumGoroutine()
	b.Logf("Goroutine growth: %d", finalGoroutines-initialGoroutines)
}

// BenchmarkScalability tests scalability with different numbers of concurrent operations
func BenchmarkScalability(b *testing.B) {
	tempDir := createBenchmarkProject(b)
	defer os.RemoveAll(tempDir)

	concurrencyLevels := []int{1, 2, 4, 8, 16}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency-%d", concurrency), func(b *testing.B) {
			logger := &MockLogger{messages: make([]LogMessage, 0)}

			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					checker := NewCompositeChecker(tempDir, logger)
					result := checker.CheckCodeStyle()
					_ = result
				}
			})
		})
	}
}
