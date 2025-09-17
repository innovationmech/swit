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
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"gopkg.in/yaml.v3"
)

type exampleConfig struct {
	Current         string                             `yaml:"current"`
	MigrationTarget string                             `yaml:"migration_target"`
	Brokers         map[string]*messaging.BrokerConfig `yaml:"brokers"`
}

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	targetName := os.Getenv("SWIT_TARGET_BROKER")
	if targetName == "" {
		targetName = cfg.MigrationTarget
	}
	if targetName == "" {
		targetName = cfg.Current
	}

	currentCfg, ok := cfg.Brokers[cfg.Current]
	if !ok {
		log.Fatalf("config missing current broker definition %q", cfg.Current)
	}
	targetCfg, ok := cfg.Brokers[targetName]
	if !ok {
		log.Fatalf("config missing target broker definition %q", targetName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	plan, err := messaging.PlanBrokerSwitch(ctx, currentCfg, targetCfg)
	if err != nil {
		log.Fatalf("plan broker switch: %v", err)
	}

	fmt.Printf("Current adapter: %s\n", plan.CurrentType)
	fmt.Printf("Target adapter:  %s\n", plan.TargetType)
	fmt.Printf("Compatibility score: %d (difficulty: %s)\n\n", plan.CompatibilityScore, plan.MigrationDifficulty)

	if len(plan.FeatureDeltas) > 0 {
		fmt.Println("Feature deltas:")
		for _, delta := range plan.FeatureDeltas {
			fmt.Printf("  - [%s] %s -> %s (%s)\n", delta.Delta, delta.CurrentSupport, delta.TargetSupport, delta.Impact)
			if delta.Recommendation != "" {
				fmt.Printf("      Recommendation: %s\n", delta.Recommendation)
			}
		}
		fmt.Println()
	}

	fmt.Println("Migration checklist:")
	for _, item := range plan.Checklist {
		fmt.Printf("  - %s\n", item)
	}

	if len(plan.Recommendations) > 0 {
		fmt.Println()
		fmt.Println("Additional recommendations:")
		for _, rec := range plan.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}

	fmt.Println()
	fmt.Println("To switch adapters, update config.yaml so that `current` points to the desired broker or set the SWIT_TARGET_BROKER environment variable before deploying.")
}

func loadConfig(path string) (*exampleConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg exampleConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("no brokers defined in config")
	}
	return &cfg, nil
}
