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

package middleware

import (
	"sync"
	"testing"

	dto "github.com/prometheus/client_model/go"
)

// gatherMetric returns the metric family with the given name from the collector's registry.
func gatherMetric(t *testing.T, pmc *PrometheusMetricsCollector, name string) *dto.MetricFamily {
	t.Helper()

	families, err := pmc.GetRegistry().Gather()
	if err != nil {
		t.Fatalf("Gather() unexpected error: %v", err)
	}
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	return nil
}

func TestPrometheusMetricsCollector_Counter(t *testing.T) {
	pmc := NewPrometheusMetricsCollector()
	labels := map[string]string{"topic": "orders", "status": "ok"}

	pmc.IncrementCounter("messages_processed_total", labels, 1)
	pmc.AddToCounter("messages_processed_total", labels, 4)

	family := gatherMetric(t, pmc, "messages_processed_total")
	if family == nil {
		t.Fatal("counter metric not found in registry")
	}
	if got := family.GetMetric()[0].GetCounter().GetValue(); got != 5 {
		t.Errorf("counter value = %v, want 5", got)
	}
}

func TestPrometheusMetricsCollector_CounterRejectsNegative(t *testing.T) {
	pmc := NewPrometheusMetricsCollector()
	labels := map[string]string{"topic": "orders"}

	pmc.AddToCounter("messages_total", labels, 2)
	pmc.AddToCounter("messages_total", labels, -1) // must be ignored

	family := gatherMetric(t, pmc, "messages_total")
	if family == nil {
		t.Fatal("counter metric not found in registry")
	}
	if got := family.GetMetric()[0].GetCounter().GetValue(); got != 2 {
		t.Errorf("counter value = %v, want 2 (negative add ignored)", got)
	}
}

func TestPrometheusMetricsCollector_Gauge(t *testing.T) {
	pmc := NewPrometheusMetricsCollector()
	labels := map[string]string{"queue": "orders"}

	pmc.SetGauge("queue_depth", labels, 42)
	pmc.SetGauge("queue_depth", labels, 7)

	family := gatherMetric(t, pmc, "queue_depth")
	if family == nil {
		t.Fatal("gauge metric not found in registry")
	}
	if got := family.GetMetric()[0].GetGauge().GetValue(); got != 7 {
		t.Errorf("gauge value = %v, want 7", got)
	}
}

func TestPrometheusMetricsCollector_Histogram(t *testing.T) {
	pmc := NewPrometheusMetricsCollector()
	labels := map[string]string{"topic": "orders"}

	pmc.ObserveHistogram("processing_duration_seconds", labels, 0.25)
	pmc.ObserveHistogram("processing_duration_seconds", labels, 0.75)

	family := gatherMetric(t, pmc, "processing_duration_seconds")
	if family == nil {
		t.Fatal("histogram metric not found in registry")
	}
	histogram := family.GetMetric()[0].GetHistogram()
	if got := histogram.GetSampleCount(); got != 2 {
		t.Errorf("histogram sample count = %v, want 2", got)
	}
	if got := histogram.GetSampleSum(); got != 1.0 {
		t.Errorf("histogram sample sum = %v, want 1.0", got)
	}
}

func TestPrometheusMetricsCollector_LabelNormalization(t *testing.T) {
	pmc := NewPrometheusMetricsCollector()

	// First observation fixes the label schema to {"topic"}
	pmc.IncrementCounter("normalized_total", map[string]string{"topic": "a"}, 1)

	// Missing label becomes empty value; extra labels are dropped
	pmc.IncrementCounter("normalized_total", nil, 1)
	pmc.IncrementCounter("normalized_total", map[string]string{"topic": "a", "extra": "x"}, 1)

	family := gatherMetric(t, pmc, "normalized_total")
	if family == nil {
		t.Fatal("counter metric not found in registry")
	}

	total := 0.0
	for _, metric := range family.GetMetric() {
		total += metric.GetCounter().GetValue()
	}
	if total != 3 {
		t.Errorf("total across label sets = %v, want 3", total)
	}
}

func TestPrometheusMetricsCollector_SanitizesMetricNames(t *testing.T) {
	pmc := NewPrometheusMetricsCollector()

	pmc.IncrementCounter("messages.received-total", map[string]string{}, 1)

	family := gatherMetric(t, pmc, "messages_received_total")
	if family == nil {
		t.Fatal("sanitized metric name not found in registry")
	}
}

func TestPrometheusMetricsCollector_ConcurrentAccess(t *testing.T) {
	pmc := NewPrometheusMetricsCollector()
	labels := map[string]string{"topic": "orders"}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				pmc.IncrementCounter("concurrent_total", labels, 1)
				pmc.SetGauge("concurrent_gauge", labels, float64(j))
				pmc.ObserveHistogram("concurrent_duration", labels, 0.1)
			}
		}()
	}
	wg.Wait()

	family := gatherMetric(t, pmc, "concurrent_total")
	if family == nil {
		t.Fatal("counter metric not found in registry")
	}
	if got := family.GetMetric()[0].GetCounter().GetValue(); got != 1000 {
		t.Errorf("counter value = %v, want 1000", got)
	}
}
