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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/types"
)

// ConfigMonitorOptions configures the configuration monitoring behavior.
type ConfigMonitorOptions struct {
	// DesiredConfigPath, when set, loads desired/baseline state from the given file path
	// for drift detection. When empty, initial merged settings are used as baseline.
	DesiredConfigPath string

	// EnableSectionMetrics enables per-section drift gauges (limited to avoid cardinality).
	EnableSectionMetrics bool

	// SectionLabelLimit limits number of unique section labels recorded.
	SectionLabelLimit int
}

// ConfigMonitor monitors configuration changes and detects configuration drift
// against a desired state. It reports Prometheus metrics through the provided
// types.MetricsCollector implementation.
type ConfigMonitor struct {
	manager   *Manager
	reloader  *HotReloader
	collector types.MetricsCollector
	options   ConfigMonitorOptions

	mu               sync.RWMutex
	baselineSettings map[string]interface{}
	desiredSettings  map[string]interface{}
	previousSettings map[string]interface{}

	started int32 // atomic guard
	stopCh  chan struct{}

	// section tracking to constrain label cardinality
	seenSections map[string]struct{}
}

// NewConfigMonitor creates a new configuration monitor.
func NewConfigMonitor(manager *Manager, reloader *HotReloader, collector types.MetricsCollector, options ConfigMonitorOptions) *ConfigMonitor {
	if options.SectionLabelLimit <= 0 {
		options.SectionLabelLimit = 10
	}
	return &ConfigMonitor{
		manager:          manager,
		reloader:         reloader,
		collector:        collector,
		options:          options,
		stopCh:           make(chan struct{}),
		seenSections:     make(map[string]struct{}),
		baselineSettings: nil,
	}
}

// Start begins monitoring configuration changes. It's safe to call once.
func (cm *ConfigMonitor) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&cm.started, 0, 1) {
		return nil
	}

	// Initialize baseline and desired state
	cm.mu.Lock()
	baseline := cm.manager.AllSettings()
	cm.baselineSettings = cloneSettings(baseline)
	cm.previousSettings = cloneSettings(baseline)

	if path := strings.TrimSpace(cm.options.DesiredConfigPath); path != "" {
		if desired, err := loadSettingsFromFile(cm.manager, path); err != nil {
			logger.Logger.Warn("failed to load desired configuration; falling back to initial baseline",
				zap.Error(err),
				zap.String("path", path))
			cm.desiredSettings = cloneSettings(baseline)
		} else {
			cm.desiredSettings = desired
		}
	} else if envPath := strings.TrimSpace(os.Getenv("SWIT_CONFIG_DESIRED_FILE")); envPath != "" {
		if desired, err := loadSettingsFromFile(cm.manager, envPath); err != nil {
			logger.Logger.Warn("failed to load desired configuration from env; falling back to initial baseline",
				zap.Error(err),
				zap.String("path", envPath))
			cm.desiredSettings = cloneSettings(baseline)
		} else {
			cm.desiredSettings = desired
		}
	} else {
		cm.desiredSettings = cloneSettings(baseline)
	}
	cm.mu.Unlock()

	// Record baseline timestamp
	cm.collector.SetGauge("swit_config_baseline_timestamp", float64(time.Now().Unix()), map[string]string{})
	cm.updateDriftMetrics(baseline)

	// If no reloader, just return after setting baseline
	if cm.reloader == nil {
		return nil
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-cm.stopCh:
				return
			case change, ok := <-cm.reloader.Events():
				if !ok {
					return
				}
				cm.HandleChange(change)
			}
		}
	}()

	return nil
}

// Stop stops the monitor if it was started.
func (cm *ConfigMonitor) Stop() {
	if atomic.LoadInt32(&cm.started) == 1 {
		close(cm.stopCh)
	}
}

// HandleChange processes a configuration change event. Safe to call from tests.
func (cm *ConfigMonitor) HandleChange(change Change) {
	if change.Err != nil {
		cm.collector.IncrementCounter("swit_config_reload_errors_total", map[string]string{
			"reason": "reload_failed",
		})
		return
	}

	// Update change counters
	layer := cm.classifyLayer(change.File)
	cm.collector.IncrementCounter("swit_config_changes_total", map[string]string{
		"source": "file",
		"layer":  layer,
	})
	cm.collector.SetGauge("swit_config_last_change_timestamp", float64(time.Now().Unix()), map[string]string{})

	// Compute change set size vs previous
	cm.mu.RLock()
	prev := cm.previousSettings
	cm.mu.RUnlock()

	changed := diffCount(prev, change.Settings)
	cm.collector.SetGauge("swit_config_changed_items", float64(changed), map[string]string{})

	// Update previous snapshot
	cm.mu.Lock()
	cm.previousSettings = cloneSettings(change.Settings)
	cm.mu.Unlock()

	// Drift detection against desired
	cm.updateDriftMetrics(change.Settings)
}

func (cm *ConfigMonitor) updateDriftMetrics(current map[string]interface{}) {
	cm.mu.RLock()
	desired := cm.desiredSettings
	cm.mu.RUnlock()

	driftItems, sections := diffKeys(desired, current)
	if driftItems > 0 {
		cm.collector.SetGauge("swit_config_drift_status", 1, map[string]string{})
	} else {
		cm.collector.SetGauge("swit_config_drift_status", 0, map[string]string{})
	}
	cm.collector.SetGauge("swit_config_drift_items", float64(driftItems), map[string]string{})

	if cm.options.EnableSectionMetrics && driftItems > 0 {
		// Constrain section labels to avoid cardinality issues
		for _, sec := range sections {
			if len(cm.seenSections) >= cm.options.SectionLabelLimit {
				// Aggregate to "other" when over the limit
				cm.collector.SetGauge("swit_config_drift_sections", 1, map[string]string{"section": "other"})
				continue
			}
			if _, ok := cm.seenSections[sec]; !ok {
				cm.seenSections[sec] = struct{}{}
			}
			cm.collector.SetGauge("swit_config_drift_sections", 1, map[string]string{"section": sec})
		}
	}
}

// classifyLayer tries to classify the changed file to a configuration layer label.
func (cm *ConfigMonitor) classifyLayer(path string) string {
	if strings.TrimSpace(path) == "" {
		return "unknown"
	}

	abs := path
	if p, err := filepath.Abs(path); err == nil {
		abs = p
	}

	// Compare against candidate lists
	baseCandidates := cm.manager.fileCandidatesFor(BaseLayer)
	envCandidates := cm.manager.fileCandidatesFor(EnvironmentFileLayer)
	overCandidates := cm.manager.fileCandidatesFor(OverrideFileLayer)

	if containsPath(baseCandidates, abs) {
		return "base"
	}
	if containsPath(envCandidates, abs) {
		return "env"
	}
	if containsPath(overCandidates, abs) {
		return "override"
	}
	return "other"
}

func containsPath(candidates []string, path string) bool {
	for _, c := range candidates {
		if p, err := filepath.Abs(c); err == nil {
			if p == path {
				return true
			}
		} else if c == path {
			return true
		}
	}
	return false
}

// loadSettingsFromFile reads a config file using viper semantics similar to Manager.mergeFile.
func loadSettingsFromFile(m *Manager, path string) (map[string]interface{}, error) {
	abs := path
	if p, err := filepath.Abs(path); err == nil {
		abs = p
	}

	if st, err := os.Stat(abs); err != nil || st.IsDir() {
		return nil, fmt.Errorf("desired config path is invalid: %s", abs)
	}

	vp := viper.New()
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(abs)), ".")
	if ext == "yml" {
		ext = "yaml"
	}
	if ext == "" {
		ext = m.normalizedConfigExt()
	}
	vp.SetConfigType(ext)

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	if err := vp.ReadConfig(strings.NewReader(string(data))); err != nil {
		return nil, err
	}
	settings := vp.AllSettings()
	// Normalize any map[interface{}]interface{} that may appear
	return normalizeSettings(settings), nil
}

func normalizeSettings(v interface{}) map[string]interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, vv := range val {
			out[k] = normalizeValue(vv)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, vv := range val {
			out[fmt.Sprintf("%v", k)] = normalizeValue(vv)
		}
		return out
	default:
		return map[string]interface{}{}
	}
}

func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, vv := range val {
			out[k] = normalizeValue(vv)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, vv := range val {
			out[fmt.Sprintf("%v", k)] = normalizeValue(vv)
		}
		return out
	case []interface{}:
		for i := range val {
			val[i] = normalizeValue(val[i])
		}
		return val
	default:
		return v
	}
}

func cloneSettings(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = deepClone(v)
	}
	return out
}

func deepClone(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, vv := range val {
			out[k] = deepClone(vv)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i := range val {
			out[i] = deepClone(val[i])
		}
		return out
	default:
		return val
	}
}

// diffCount returns number of differing leaf nodes between two settings maps.
func diffCount(a, b map[string]interface{}) int {
	return countDiffRecursive(a, b)
}

func countDiffRecursive(a, b interface{}) int {
	// Handle nils
	if a == nil && b == nil {
		return 0
	}
	if a == nil || b == nil {
		return 1
	}

	// Compare by type
	switch av := a.(type) {
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok {
			return 1
		}
		// Union of keys
		keys := make(map[string]struct{})
		for k := range av {
			keys[k] = struct{}{}
		}
		for k := range bv {
			keys[k] = struct{}{}
		}
		count := 0
		for k := range keys {
			count += countDiffRecursive(av[k], bv[k])
		}
		return count
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok {
			return 1
		}
		if len(av) != len(bv) {
			return 1
		}
		for i := range av {
			if countDiffRecursive(av[i], bv[i]) != 0 {
				return 1
			}
		}
		return 0
	default:
		if reflect.DeepEqual(a, b) {
			return 0
		}
		return 1
	}
}

// diffKeys returns total number of differing leaf nodes and a limited list of top-level sections.
func diffKeys(desired, current map[string]interface{}) (int, []string) {
	if desired == nil {
		desired = map[string]interface{}{}
	}
	if current == nil {
		current = map[string]interface{}{}
	}
	// Collect sections with differences
	sectionsSet := make(map[string]struct{})
	count := diffKeysRecursive(desired, current, "", sectionsSet)
	sections := make([]string, 0, len(sectionsSet))
	for s := range sectionsSet {
		sections = append(sections, s)
	}
	return count, sections
}

func diffKeysRecursive(a, b interface{}, path string, sections map[string]struct{}) int {
	// Terminal comparisons
	if a == nil && b == nil {
		return 0
	}
	if a == nil || b == nil {
		markSection(path, sections)
		return 1
	}

	switch av := a.(type) {
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok {
			markSection(path, sections)
			return 1
		}
		// Keys union
		keys := make(map[string]struct{})
		for k := range av {
			keys[k] = struct{}{}
		}
		for k := range bv {
			keys[k] = struct{}{}
		}
		total := 0
		for k := range keys {
			childPath := joinPath(path, k)
			total += diffKeysRecursive(av[k], bv[k], childPath, sections)
		}
		return total
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok {
			markSection(path, sections)
			return 1
		}
		if len(av) != len(bv) {
			markSection(path, sections)
			return 1
		}
		for i := range av {
			if diffKeysRecursive(av[i], bv[i], path, sections) != 0 {
				markSection(path, sections)
				return 1
			}
		}
		return 0
	default:
		if reflect.DeepEqual(a, b) {
			return 0
		}
		markSection(path, sections)
		return 1
	}
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

func markSection(path string, sections map[string]struct{}) {
	if path == "" {
		sections["root"] = struct{}{}
		return
	}
	parts := strings.Split(path, ".")
	sections[parts[0]] = struct{}{}
}

// no-op: zap imported for logging
