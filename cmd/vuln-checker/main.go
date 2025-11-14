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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/innovationmech/swit/pkg/security/vulnerability"
	"go.uber.org/zap"
)

func main() {
	// Command line flags
	modulePath := flag.String("module", ".", "Path to the Go module to check")
	reportPath := flag.String("report", "_output/security/vulnerability-report.json", "Path to save the vulnerability report")
	failOnHigh := flag.Bool("fail-on-high", true, "Fail on high severity vulnerabilities")
	failOnMedium := flag.Bool("fail-on-medium", false, "Fail on medium severity vulnerabilities")
	failOnCritical := flag.Bool("fail-on-critical", true, "Fail on critical severity vulnerabilities")
	timeout := flag.Duration("timeout", 5*time.Minute, "Timeout for vulnerability check")
	flag.Parse()

	// Create logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Configure vulnerability checker
	config := &vulnerability.CheckerConfig{
		Database:                vulnerability.DefaultDatabaseConfig(),
		ModulePath:              *modulePath,
		FailOnHigh:              *failOnHigh,
		FailOnMedium:            *failOnMedium,
		FailOnCritical:          *failOnCritical,
		Timeout:                 *timeout,
		UpdateInterval:          24 * time.Hour,
		ReportPath:              *reportPath,
		NotifyOnVulnerabilities: false,
	}

	// Create vulnerability checker
	checker, err := vulnerability.NewVulnerabilityChecker(config, logger)
	if err != nil {
		logger.Fatal("Failed to create vulnerability checker", zap.Error(err))
	}

	// Run vulnerability check
	logger.Info("Starting vulnerability check",
		zap.String("module_path", *modulePath))

	ctx := context.Background()
	result, err := checker.Check(ctx)
	if err != nil {
		logger.Error("Vulnerability check failed", zap.Error(err))
		os.Exit(1)
	}

	// Print summary
	fmt.Println("\n========================================")
	fmt.Println("  Vulnerability Check Summary")
	fmt.Println("========================================")
	fmt.Printf("Total Dependencies:       %d\n", result.Summary.TotalDependencies)
	fmt.Printf("Vulnerable Dependencies:  %d\n", result.Summary.VulnerableCount)
	fmt.Printf("Total Vulnerabilities:    %d\n", result.Summary.TotalVulnerabilities)
	fmt.Println("----------------------------------------")
	fmt.Printf("Critical:                 %d\n", result.Summary.CriticalCount)
	fmt.Printf("High:                     %d\n", result.Summary.HighCount)
	fmt.Printf("Medium:                   %d\n", result.Summary.MediumCount)
	fmt.Printf("Low:                      %d\n", result.Summary.LowCount)
	fmt.Println("========================================")
	fmt.Printf("Check Duration:           %v\n", result.Duration)
	fmt.Printf("Report saved to:          %s\n", *reportPath)
	fmt.Println("========================================\n")

	// Print vulnerabilities if any
	if result.Summary.TotalVulnerabilities > 0 {
		fmt.Println("Vulnerabilities found:")
		fmt.Println()
		for _, vuln := range result.Vulnerabilities {
			fmt.Printf("📦 %s@%s\n", vuln.PackageName, vuln.InstalledVersion)
			fmt.Printf("   🔒 %s [%s]\n", vuln.VulnerabilityID, vuln.Severity)
			fmt.Printf("   📝 %s\n", vuln.Summary)
			if vuln.FixedVersion != "" {
				fmt.Printf("   ✅ Fixed in: %s\n", vuln.FixedVersion)
			} else {
				fmt.Printf("   ⚠️  No fix available yet\n")
			}
			if vuln.CVSS > 0 {
				fmt.Printf("   📊 CVSS Score: %.1f\n", vuln.CVSS)
			}
			fmt.Println()
		}
	}

	// Determine exit status
	if checker.ShouldFail(result) {
		logger.Warn("Vulnerability check failed due to severity threshold",
			zap.Bool("fail_on_critical", *failOnCritical),
			zap.Bool("fail_on_high", *failOnHigh),
			zap.Bool("fail_on_medium", *failOnMedium),
			zap.Int("critical_count", result.Summary.CriticalCount),
			zap.Int("high_count", result.Summary.HighCount),
			zap.Int("medium_count", result.Summary.MediumCount))
		os.Exit(1)
	}

	logger.Info("Vulnerability check passed")
}

