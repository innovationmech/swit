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
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/types"
)

// fakeCollector is an in-memory types.MetricsCollector recording counters and gauges.
type fakeCollector struct {
	mu       sync.Mutex
	counters map[string]float64
	gauges   map[string]float64
}

var _ types.MetricsCollector = (*fakeCollector)(nil)

func newFakeCollector() *fakeCollector {
	return &fakeCollector{
		counters: make(map[string]float64),
		gauges:   make(map[string]float64),
	}
}

func metricKey(name string, labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	key := name
	for _, k := range keys {
		key += "|" + k + "=" + labels[k]
	}
	return key
}

func (f *fakeCollector) IncrementCounter(name string, labels map[string]string) {
	f.AddToCounter(name, 1, labels)
}

func (f *fakeCollector) AddToCounter(name string, value float64, labels map[string]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counters[metricKey(name, labels)] += value
}

func (f *fakeCollector) SetGauge(name string, value float64, labels map[string]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gauges[metricKey(name, labels)] = value
}

func (f *fakeCollector) IncrementGauge(name string, labels map[string]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gauges[metricKey(name, labels)]++
}

func (f *fakeCollector) DecrementGauge(name string, labels map[string]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gauges[metricKey(name, labels)]--
}

func (f *fakeCollector) ObserveHistogram(name string, value float64, labels map[string]string) {}

func (f *fakeCollector) GetMetrics() []types.Metric { return nil }

func (f *fakeCollector) GetMetric(name string) (*types.Metric, bool) { return nil, false }

func (f *fakeCollector) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counters = make(map[string]float64)
	f.gauges = make(map[string]float64)
}

func (f *fakeCollector) counter(name string, labels map[string]string) float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.counters[metricKey(name, labels)]
}

func (f *fakeCollector) gauge(name string, labels map[string]string) (float64, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.gauges[metricKey(name, labels)]
	return v, ok
}

// newTestManager creates a Manager backed by a temp dir with the given base config content.
func newTestManager(t *testing.T, baseYAML string) (*Manager, string) {
	t.Helper()
	logger.InitLogger()

	dir := t.TempDir()
	base := filepath.Join(dir, "swit.yaml")
	require.NoError(t, os.WriteFile(base, []byte(baseYAML), 0o644))

	opts := DefaultOptions()
	opts.WorkDir = dir
	opts.EnableAutomaticEnv = false
	m := NewManager(opts)
	require.NoError(t, m.Load())
	return m, dir
}

func TestConfigMonitor_StartSetsBaselineWithoutDrift(t *testing.T) {
	m, _ := newTestManager(t, "server:\n  port: 8080\n")
	collector := newFakeCollector()
	monitor := NewConfigMonitor(m, nil, collector, ConfigMonitorOptions{})

	require.NoError(t, monitor.Start(context.Background()))
	defer monitor.Stop()

	baseline, ok := collector.gauge("swit_config_baseline_timestamp", map[string]string{})
	assert.True(t, ok, "baseline timestamp gauge must be set")
	assert.Greater(t, baseline, float64(0))

	drift, ok := collector.gauge("swit_config_drift_status", map[string]string{})
	assert.True(t, ok)
	assert.Equal(t, float64(0), drift, "no drift expected when desired equals baseline")

	items, _ := collector.gauge("swit_config_drift_items", map[string]string{})
	assert.Equal(t, float64(0), items)
}

func TestConfigMonitor_StartIsIdempotent(t *testing.T) {
	m, _ := newTestManager(t, "server:\n  port: 8080\n")
	collector := newFakeCollector()
	monitor := NewConfigMonitor(m, nil, collector, ConfigMonitorOptions{})

	require.NoError(t, monitor.Start(context.Background()))
	require.NoError(t, monitor.Start(context.Background()), "second Start must be a no-op")
	monitor.Stop()
}

func TestConfigMonitor_DesiredConfigDriftDetection(t *testing.T) {
	m, dir := newTestManager(t, "server:\n  port: 9090\n  name: actual\n")

	// Desired state differs from the actual configuration
	desired := filepath.Join(dir, "desired.yaml")
	require.NoError(t, os.WriteFile(desired, []byte("server:\n  port: 8080\n  name: desired\n"), 0o644))

	collector := newFakeCollector()
	monitor := NewConfigMonitor(m, nil, collector, ConfigMonitorOptions{
		DesiredConfigPath:    desired,
		EnableSectionMetrics: true,
	})

	require.NoError(t, monitor.Start(context.Background()))
	defer monitor.Stop()

	drift, ok := collector.gauge("swit_config_drift_status", map[string]string{})
	assert.True(t, ok)
	assert.Equal(t, float64(1), drift, "drift expected between desired and actual")

	items, _ := collector.gauge("swit_config_drift_items", map[string]string{})
	assert.Equal(t, float64(2), items, "port and name differ")

	section, ok := collector.gauge("swit_config_drift_sections", map[string]string{"section": "server"})
	assert.True(t, ok, "server section drift gauge must be present")
	assert.Equal(t, float64(1), section)
}

func TestConfigMonitor_InvalidDesiredPathFallsBackToBaseline(t *testing.T) {
	m, _ := newTestManager(t, "server:\n  port: 8080\n")
	collector := newFakeCollector()
	monitor := NewConfigMonitor(m, nil, collector, ConfigMonitorOptions{
		DesiredConfigPath: "/nonexistent/desired.yaml",
	})

	require.NoError(t, monitor.Start(context.Background()))
	defer monitor.Stop()

	drift, _ := collector.gauge("swit_config_drift_status", map[string]string{})
	assert.Equal(t, float64(0), drift, "fallback to baseline must produce no drift")
}

func TestConfigMonitor_HandleChangeError(t *testing.T) {
	m, _ := newTestManager(t, "server:\n  port: 8080\n")
	collector := newFakeCollector()
	monitor := NewConfigMonitor(m, nil, collector, ConfigMonitorOptions{})
	require.NoError(t, monitor.Start(context.Background()))
	defer monitor.Stop()

	monitor.HandleChange(Change{Err: errors.New("reload boom")})

	errCount := collector.counter("swit_config_reload_errors_total", map[string]string{"reason": "reload_failed"})
	assert.Equal(t, float64(1), errCount)
}

func TestConfigMonitor_HandleChangeUpdatesMetricsAndDrift(t *testing.T) {
	m, dir := newTestManager(t, "server:\n  port: 8080\n")
	collector := newFakeCollector()
	monitor := NewConfigMonitor(m, nil, collector, ConfigMonitorOptions{})
	require.NoError(t, monitor.Start(context.Background()))
	defer monitor.Stop()

	// Simulate a change on the base layer file with a new port value
	changed := map[string]interface{}{
		"server": map[string]interface{}{"port": 9090},
	}
	monitor.HandleChange(Change{
		File:     filepath.Join(dir, "swit.yaml"),
		Settings: changed,
	})

	changes := collector.counter("swit_config_changes_total", map[string]string{"source": "file", "layer": "base"})
	assert.Equal(t, float64(1), changes, "base layer change must be counted")

	lastChange, ok := collector.gauge("swit_config_last_change_timestamp", map[string]string{})
	assert.True(t, ok)
	assert.Greater(t, lastChange, float64(0))

	changedItems, _ := collector.gauge("swit_config_changed_items", map[string]string{})
	assert.Equal(t, float64(1), changedItems, "one leaf (port) changed")

	drift, _ := collector.gauge("swit_config_drift_status", map[string]string{})
	assert.Equal(t, float64(1), drift, "change away from desired baseline must be flagged as drift")
}

func TestConfigMonitor_HotReloadEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hot reload end-to-end test in short mode")
	}

	m, dir := newTestManager(t, "server:\n  port: 8080\n")
	reloader := NewHotReloader(m, 50*time.Millisecond)
	require.NoError(t, reloader.Start())
	defer reloader.Stop()

	collector := newFakeCollector()
	monitor := NewConfigMonitor(m, reloader, collector, ConfigMonitorOptions{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, monitor.Start(ctx))
	defer monitor.Stop()

	// Modify the watched base file to trigger a reload event. The watcher is
	// registered asynchronously, so keep rewriting until the event is observed.
	base := filepath.Join(dir, "swit.yaml")
	assert.Eventually(t, func() bool {
		require.NoError(t, os.WriteFile(base, []byte("server:\n  port: 9090\n"), 0o644))
		return collector.counter("swit_config_changes_total", map[string]string{"source": "file", "layer": "base"}) >= 1
	}, 5*time.Second, 100*time.Millisecond, "hot reload change event must reach the monitor")

	assert.Eventually(t, func() bool {
		drift, _ := collector.gauge("swit_config_drift_status", map[string]string{})
		return drift == 1
	}, 5*time.Second, 50*time.Millisecond, "drift must be detected after hot reload")
}

func TestConfigMonitor_ClassifyLayer(t *testing.T) {
	m, dir := newTestManager(t, "server:\n  port: 8080\n")
	monitor := NewConfigMonitor(m, nil, newFakeCollector(), ConfigMonitorOptions{})

	assert.Equal(t, "base", monitor.classifyLayer(filepath.Join(dir, "swit.yaml")))
	assert.Equal(t, "override", monitor.classifyLayer(filepath.Join(dir, "swit.override.yaml")))
	assert.Equal(t, "other", monitor.classifyLayer(filepath.Join(dir, "random.yaml")))
	assert.Equal(t, "unknown", monitor.classifyLayer(""))
}

func TestConfigMonitor_SectionLabelLimit(t *testing.T) {
	m, dir := newTestManager(t, "a:\n  v: 1\nb:\n  v: 1\nc:\n  v: 1\n")

	desired := filepath.Join(dir, "desired.yaml")
	require.NoError(t, os.WriteFile(desired, []byte("a:\n  v: 2\nb:\n  v: 2\nc:\n  v: 2\n"), 0o644))

	collector := newFakeCollector()
	monitor := NewConfigMonitor(m, nil, collector, ConfigMonitorOptions{
		DesiredConfigPath:    desired,
		EnableSectionMetrics: true,
		SectionLabelLimit:    2,
	})

	require.NoError(t, monitor.Start(context.Background()))
	defer monitor.Stop()

	// At most SectionLabelLimit distinct section labels plus "other" aggregation
	distinct := 0
	for _, sec := range []string{"a", "b", "c"} {
		if _, ok := collector.gauge("swit_config_drift_sections", map[string]string{"section": sec}); ok {
			distinct++
		}
	}
	other, hasOther := collector.gauge("swit_config_drift_sections", map[string]string{"section": "other"})
	assert.LessOrEqual(t, distinct, 2)
	assert.True(t, hasOther, "sections above the limit must aggregate into 'other'")
	assert.Equal(t, float64(1), other)
}

func TestDiffCount(t *testing.T) {
	tests := []struct {
		name string
		a, b map[string]interface{}
		want int
	}{
		{
			name: "identical maps",
			a:    map[string]interface{}{"k": "v"},
			b:    map[string]interface{}{"k": "v"},
			want: 0,
		},
		{
			name: "one leaf changed",
			a:    map[string]interface{}{"k": "v1"},
			b:    map[string]interface{}{"k": "v2"},
			want: 1,
		},
		{
			name: "added key",
			a:    map[string]interface{}{"k": "v"},
			b:    map[string]interface{}{"k": "v", "new": 1},
			want: 1,
		},
		{
			name: "nested difference",
			a:    map[string]interface{}{"s": map[string]interface{}{"x": 1, "y": 2}},
			b:    map[string]interface{}{"s": map[string]interface{}{"x": 1, "y": 3}},
			want: 1,
		},
		{
			name: "slice length difference counts once",
			a:    map[string]interface{}{"l": []interface{}{1, 2}},
			b:    map[string]interface{}{"l": []interface{}{1}},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, diffCount(tt.a, tt.b))
		})
	}
}

func TestDiffKeys(t *testing.T) {
	desired := map[string]interface{}{
		"server":   map[string]interface{}{"port": 8080},
		"database": map[string]interface{}{"host": "localhost"},
	}
	current := map[string]interface{}{
		"server":   map[string]interface{}{"port": 9090},
		"database": map[string]interface{}{"host": "localhost"},
	}

	count, sections := diffKeys(desired, current)
	assert.Equal(t, 1, count)
	assert.Equal(t, []string{"server"}, sections)

	// nil maps are tolerated
	count, _ = diffKeys(nil, nil)
	assert.Equal(t, 0, count)
}

func TestCloneSettingsDeepCopy(t *testing.T) {
	original := map[string]interface{}{
		"nested": map[string]interface{}{"k": "v"},
		"list":   []interface{}{1, 2},
	}

	cloned := cloneSettings(original)
	require.NotNil(t, cloned)

	// Mutating the clone must not affect the original
	cloned["nested"].(map[string]interface{})["k"] = "changed"
	cloned["list"].([]interface{})[0] = 99

	assert.Equal(t, "v", original["nested"].(map[string]interface{})["k"])
	assert.Equal(t, 1, original["list"].([]interface{})[0])

	assert.Nil(t, cloneSettings(nil))
}

func TestLoadSettingsFromFile(t *testing.T) {
	m, dir := newTestManager(t, "server:\n  port: 8080\n")

	t.Run("valid yaml file", func(t *testing.T) {
		path := filepath.Join(dir, "extra.yaml")
		require.NoError(t, os.WriteFile(path, []byte("key: value\n"), 0o644))

		settings, err := loadSettingsFromFile(m, path)
		require.NoError(t, err)
		assert.Equal(t, "value", settings["key"])
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := loadSettingsFromFile(m, filepath.Join(dir, "missing.yaml"))
		assert.Error(t, err)
	})

	t.Run("directory path", func(t *testing.T) {
		_, err := loadSettingsFromFile(m, dir)
		assert.Error(t, err)
	})
}
