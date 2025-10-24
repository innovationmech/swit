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

// Package main provides a command-line tool for validating and testing Saga DSL files.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/innovationmech/swit/pkg/saga/dsl"
)

const (
	version = "1.0.0"
	usage   = `saga-dsl-validate - Saga DSL validation and testing tool

USAGE:
    saga-dsl-validate [command] [options] <files...>

COMMANDS:
    check       Check syntax of DSL files
    validate    Validate DSL files against all rules
    dry-run     Perform dry-run simulation
    test        Run comprehensive tests (syntax + validation + dry-run)
    format      Format DSL files
    version     Show version information
    help        Show this help message

OPTIONS:
    -format string       Output format: text, json, yaml (default: text)
    -strict             Enable strict mode (warnings treated as errors)
    -verbose            Enable verbose output
    -fail-fast          Stop on first failure
    -pattern string     File pattern for directory scanning (default: *.yaml)
    -o string           Output file (default: stdout)

EXAMPLES:
    # Check syntax of a single file
    saga-dsl-validate check order.saga.yaml

    # Validate a file
    saga-dsl-validate validate order.saga.yaml

    # Validate with JSON output
    saga-dsl-validate validate -format json order.saga.yaml

    # Perform dry-run simulation
    saga-dsl-validate dry-run order.saga.yaml

    # Test multiple files
    saga-dsl-validate test order.saga.yaml payment.saga.yaml

    # Test all YAML files in a directory
    saga-dsl-validate test ./sagas/

    # Test with strict mode and verbose output
    saga-dsl-validate test -strict -verbose ./sagas/

    # Format a DSL file
    saga-dsl-validate format order.saga.yaml
`
)

type config struct {
	command   string
	format    string
	strict    bool
	verbose   bool
	failFast  bool
	pattern   string
	outputFile string
	files     []string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := parseArgs()
	if err != nil {
		return err
	}

	switch cfg.command {
	case "check":
		return runCheck(cfg)
	case "validate":
		return runValidate(cfg)
	case "dry-run":
		return runDryRun(cfg)
	case "test":
		return runTest(cfg)
	case "format":
		return runFormat(cfg)
	case "version":
		return runVersion()
	case "help":
		fmt.Print(usage)
		return nil
	default:
		fmt.Print(usage)
		return fmt.Errorf("unknown command: %s", cfg.command)
	}
}

func parseArgs() (*config, error) {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(0)
	}

	cfg := &config{}
	cfg.command = os.Args[1]

	// Create flag set for parsing options
	fs := flag.NewFlagSet(cfg.command, flag.ExitOnError)
	fs.StringVar(&cfg.format, "format", "text", "Output format: text, json, yaml")
	fs.BoolVar(&cfg.strict, "strict", false, "Enable strict mode")
	fs.BoolVar(&cfg.verbose, "verbose", false, "Enable verbose output")
	fs.BoolVar(&cfg.failFast, "fail-fast", false, "Stop on first failure")
	fs.StringVar(&cfg.pattern, "pattern", "*.yaml", "File pattern for directory scanning")
	fs.StringVar(&cfg.outputFile, "o", "", "Output file")

	// Parse flags
	if err := fs.Parse(os.Args[2:]); err != nil {
		return nil, err
	}

	// Get remaining arguments as files
	cfg.files = fs.Args()

	// Validate format
	switch cfg.format {
	case "text", "json", "yaml":
		// Valid
	default:
		return nil, fmt.Errorf("invalid format: %s (must be text, json, or yaml)", cfg.format)
	}

	return cfg, nil
}

func runCheck(cfg *config) error {
	if len(cfg.files) == 0 {
		return fmt.Errorf("no files specified")
	}

	var hasError bool
	for _, file := range cfg.files {
		if err := dsl.CheckSyntax(file); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", file, err)
			hasError = true
			if cfg.failFast {
				break
			}
		} else {
			fmt.Printf("✓ %s: syntax OK\n", file)
		}
	}

	if hasError {
		return fmt.Errorf("syntax check failed")
	}

	return nil
}

func runValidate(cfg *config) error {
	if len(cfg.files) == 0 {
		return fmt.Errorf("no files specified")
	}

	var hasError bool
	for _, file := range cfg.files {
		if err := dsl.ValidateFile(file); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", file, err)
			hasError = true
			if cfg.failFast {
				break
			}
		} else {
			fmt.Printf("✓ %s: validation passed\n", file)
		}
	}

	if hasError {
		return fmt.Errorf("validation failed")
	}

	return nil
}

func runDryRun(cfg *config) error {
	if len(cfg.files) == 0 {
		return fmt.Errorf("no files specified")
	}

	// Setup output writer
	output := os.Stdout
	if cfg.outputFile != "" {
		f, err := os.Create(cfg.outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		output = f
	}

	// Create tester with dry-run enabled
	opts := &dsl.TesterOptions{
		StrictMode:   cfg.strict,
		DryRun:       true,
		Format:       cfg.format,
		Verbose:      cfg.verbose,
		FailFast:     cfg.failFast,
		OutputWriter: output,
	}

	tester := dsl.NewTester(opts)

	// Test files
	files, err := collectFiles(cfg.files, cfg.pattern)
	if err != nil {
		return err
	}

	if len(files) == 1 {
		// Single file: print result directly
		result := tester.TestFile(files[0])
		if err := tester.PrintResult(result); err != nil {
			return err
		}

		if result.Status == dsl.TestStatusFail {
			return fmt.Errorf("dry-run failed")
		}
	} else {
		// Multiple files: print suite
		suite := tester.TestFiles(files)
		if err := tester.PrintSuite(suite); err != nil {
			return err
		}

		if suite.FailedFiles > 0 {
			return fmt.Errorf("dry-run failed: %d/%d files failed", suite.FailedFiles, suite.TotalFiles)
		}
	}

	return nil
}

func runTest(cfg *config) error {
	if len(cfg.files) == 0 {
		return fmt.Errorf("no files specified")
	}

	// Setup output writer
	output := os.Stdout
	if cfg.outputFile != "" {
		f, err := os.Create(cfg.outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		output = f
	}

	// Create tester with full testing enabled
	opts := &dsl.TesterOptions{
		StrictMode:   cfg.strict,
		DryRun:       true, // Full test includes dry-run
		Format:       cfg.format,
		Verbose:      cfg.verbose,
		FailFast:     cfg.failFast,
		OutputWriter: output,
	}

	tester := dsl.NewTester(opts)

	// Collect files
	files, err := collectFiles(cfg.files, cfg.pattern)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no DSL files found")
	}

	// Run tests
	suite := tester.TestFiles(files)
	if err := tester.PrintSuite(suite); err != nil {
		return err
	}

	// Check results
	if suite.FailedFiles > 0 {
		return fmt.Errorf("tests failed: %d/%d files failed", suite.FailedFiles, suite.TotalFiles)
	}

	return nil
}

func runFormat(cfg *config) error {
	if len(cfg.files) == 0 {
		return fmt.Errorf("no files specified")
	}

	var hasError bool
	for _, file := range cfg.files {
		if err := dsl.FormatFile(file); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", file, err)
			hasError = true
			if cfg.failFast {
				break
			}
		} else {
			fmt.Printf("✓ %s: formatted\n", file)
		}
	}

	if hasError {
		return fmt.Errorf("formatting failed")
	}

	return nil
}

func runVersion() error {
	fmt.Printf("saga-dsl-validate version %s\n", version)
	return nil
}

// collectFiles collects all DSL files from the given paths.
// If a path is a directory, it scans recursively for files matching the pattern.
func collectFiles(paths []string, pattern string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if info.IsDir() {
			// Scan directory recursively
			err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				matched, err := filepath.Match(pattern, filepath.Base(p))
				if err != nil {
					return err
				}

				if matched && !seen[p] {
					files = append(files, p)
					seen[p] = true
				}

				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("failed to scan directory %s: %w", path, err)
			}
		} else {
			// Single file
			if !seen[path] {
				// Check if file matches pattern
				matched, err := filepath.Match(pattern, filepath.Base(path))
				if err != nil {
					return nil, err
				}

				// For explicitly specified files, add them even if they don't match pattern
				// But warn if pattern was explicitly set
				if !matched && pattern != "*.yaml" {
					fmt.Fprintf(os.Stderr, "Warning: %s does not match pattern %s\n", path, pattern)
				}

				files = append(files, path)
				seen[path] = true
			}
		}
	}

	// Filter files to only include saga definition files
	var sagaFiles []string
	for _, file := range files {
		// Include files that end with .saga.yaml or .yaml
		if strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml") {
			sagaFiles = append(sagaFiles, file)
		}
	}

	return sagaFiles, nil
}

