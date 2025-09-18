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

package benchmark

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

const (
	defaultMessageSize   = 1024 // 1 KiB messages as baseline
	defaultMessageCount  = 1000
	defaultPublisherPool = 4
)

// Workload describes a benchmark profile that the harness executes against each broker target.
type Workload struct {
	Name        string
	Topic       string
	Messages    int
	MessageSize int
	Publishers  int
	BatchSize   int
}

// Validate ensures the workload configuration is usable.
func (w Workload) Validate() error {
	if strings.TrimSpace(w.Name) == "" {
		return errors.New("workload name cannot be empty")
	}
	if w.Messages <= 0 {
		return fmt.Errorf("workload %q must publish at least one message", w.Name)
	}
	if w.MessageSize <= 0 {
		return fmt.Errorf("workload %q must use a positive message size", w.Name)
	}
	if w.Publishers <= 0 {
		return fmt.Errorf("workload %q must have at least one publisher", w.Name)
	}
	if w.BatchSize < 0 {
		return fmt.Errorf("workload %q has invalid batch size", w.Name)
	}
	return nil
}

// WithDefaults returns a copy of the workload filled with sensible defaults and sanitized fields.
func (w Workload) WithDefaults() Workload {
	if w.Messages == 0 {
		w.Messages = defaultMessageCount
	}
	if w.MessageSize == 0 {
		w.MessageSize = defaultMessageSize
	}
	if w.Publishers == 0 {
		w.Publishers = defaultPublisherPool
	}
	if w.BatchSize == 0 {
		w.BatchSize = 1
	}
	if strings.TrimSpace(w.Topic) == "" {
		w.Topic = fmt.Sprintf("bench-%s", sanitizeName(w.Name))
	}
	return w
}

// Target identifies a broker instance (adapter + configuration) that should be benchmarked.
type Target struct {
	Name        string
	Description string
	Config      *messaging.BrokerConfig
	TopicPrefix string
}

// Validate ensures the target has a usable configuration.
func (t Target) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return errors.New("target name cannot be empty")
	}
	if t.Config == nil {
		return fmt.Errorf("target %q requires a broker configuration", t.Name)
	}
	if !t.Config.Type.IsValid() {
		return fmt.Errorf("target %q specifies invalid broker type %q", t.Name, t.Config.Type)
	}
	return nil
}

// Suite coordinates execution of workloads across a set of broker targets.
type Suite struct {
	Factory   messaging.MessageBrokerFactory
	Targets   []Target
	Workloads []Workload
	Timeout   time.Duration
}

// Report captures all benchmark results from a suite execution.
type Report struct {
	GeneratedAt time.Time
	Results     []Result
}

// Result describes the outcome of executing a single workload against a broker target.
type Result struct {
	TargetName   string
	BrokerType   messaging.BrokerType
	WorkloadName string
	Workload     Workload
	Metrics      Metrics
	Error        error
}

// Metrics aggregates throughput and latency information gathered from a workload run.
type Metrics struct {
	TotalMessages  int
	Successful     int
	Failed         int
	Duration       time.Duration
	Throughput     float64
	AverageLatency time.Duration
	P50Latency     time.Duration
	P95Latency     time.Duration
	P99Latency     time.Duration
	MaxLatency     time.Duration
	ErrorSummaries []ErrorSummary
}

// ErrorSummary groups errors by canonical message for reporting.
type ErrorSummary struct {
	Message string
	Count   int
}

// DefaultWorkloads returns a representative mix of light, balanced, and peak benchmark profiles.
func DefaultWorkloads() []Workload {
	return []Workload{
		{
			Name:        "light",
			Messages:    500,
			MessageSize: 512,
			Publishers:  1,
			BatchSize:   1,
		},
		{
			Name:        "balanced",
			Messages:    2000,
			MessageSize: 1024,
			Publishers:  4,
			BatchSize:   1,
		},
		{
			Name:        "burst",
			Messages:    5000,
			MessageSize: 4096,
			Publishers:  8,
			BatchSize:   10,
		},
	}
}

// Run executes all workloads for every target and returns a consolidated report.
func (s *Suite) Run(ctx context.Context) (*Report, error) {
	if s.Factory == nil {
		return nil, errors.New("benchmark suite requires a messaging factory")
	}
	if len(s.Targets) == 0 {
		return nil, errors.New("benchmark suite requires at least one target")
	}
	if len(s.Workloads) == 0 {
		return nil, errors.New("benchmark suite requires at least one workload")
	}

	var report Report
	report.GeneratedAt = time.Now()

	for _, target := range s.Targets {
		if err := target.Validate(); err != nil {
			return nil, err
		}
		if err := s.runTarget(ctx, &report, target); err != nil {
			return nil, err
		}
	}

	return &report, nil
}

func (s *Suite) runTarget(parentCtx context.Context, report *Report, target Target) error {
	broker, err := s.Factory.CreateBroker(target.Config)
	if err != nil {
		return fmt.Errorf("failed creating broker for target %q: %w", target.Name, err)
	}

	brokerCtx := parentCtx
	var cancel context.CancelFunc
	if s.Timeout > 0 {
		brokerCtx, cancel = context.WithTimeout(parentCtx, s.Timeout)
	}
	if cancel != nil {
		defer cancel()
	}

	if err := broker.Connect(brokerCtx); err != nil {
		report.Results = append(report.Results, failureResult(target, err))
		_ = broker.Close()
		return nil
	}

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_ = broker.Disconnect(cleanupCtx)
		_ = broker.Close()
	}()

	for _, workload := range s.Workloads {
		wl := workload.WithDefaults()
		if err := wl.Validate(); err != nil {
			report.Results = append(report.Results, Result{
				TargetName:   target.Name,
				BrokerType:   target.Config.Type,
				WorkloadName: workload.Name,
				Workload:     wl,
				Error:        err,
			})
			continue
		}
		if prefix := strings.TrimSpace(target.TopicPrefix); prefix != "" {
			wl.Topic = fmt.Sprintf("%s.%s", prefix, wl.Topic)
		}

		res := s.executeWorkload(brokerCtx, broker, target, wl)
		report.Results = append(report.Results, res)
	}

	return nil
}

func failureResult(target Target, err error) Result {
	return Result{
		TargetName:   target.Name,
		BrokerType:   target.Config.Type,
		WorkloadName: "connect",
		Workload: Workload{
			Name:       "connect",
			Messages:   0,
			Publishers: 0,
			BatchSize:  1,
		},
		Error: err,
	}
}

func (s *Suite) executeWorkload(ctx context.Context, broker messaging.MessageBroker, target Target, workload Workload) Result {
	metrics := Metrics{TotalMessages: workload.Messages}

	start := time.Now()
	latencies := make([]time.Duration, 0, workload.Messages)
	var latenciesMu sync.Mutex

	errorCounts := make(map[string]int)
	var errorMu sync.Mutex

	var successCount int64
	var failureCount int64

	jobs := make(chan []*messaging.Message)
	var wg sync.WaitGroup

	for worker := 0; worker < workload.Publishers; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			publisherConfig := messaging.PublisherConfig{
				Topic: workload.Topic,
				Async: false,
			}

			publisher, err := broker.CreatePublisher(publisherConfig)
			if err != nil {
				incrementError(&errorMu, errorCounts, fmt.Sprintf("publisher[%d] create: %v", workerID, err))
				return
			}
			defer publisher.Close()

			for batch := range jobs {
				select {
				case <-ctx.Done():
					incrementError(&errorMu, errorCounts, ctx.Err().Error())
					atomic.AddInt64(&failureCount, int64(len(batch)))
					return
				default:
				}

				batchStart := time.Now()
				var publishErr error
				if len(batch) == 1 && workload.BatchSize <= 1 {
					publishErr = publisher.Publish(ctx, batch[0])
				} else if len(batch) == 1 {
					publishErr = publisher.PublishBatch(ctx, batch)
				} else {
					publishErr = publisher.PublishBatch(ctx, batch)
				}

				elapsed := time.Since(batchStart)
				if publishErr != nil {
					incrementError(&errorMu, errorCounts, publishErr.Error())
					atomic.AddInt64(&failureCount, int64(len(batch)))
					continue
				}

				latenciesMu.Lock()
				for range batch {
					latencies = append(latencies, elapsed)
				}
				latenciesMu.Unlock()
				atomic.AddInt64(&successCount, int64(len(batch)))
			}
		}(worker)
	}

	go func() {
		defer close(jobs)
		payload := makePayload(workload.MessageSize)
		remaining := workload.Messages
		produced := 0
		for remaining > 0 {
			batchSize := workload.BatchSize
			if batchSize <= 1 {
				batchSize = 1
			}
			if batchSize > remaining {
				batchSize = remaining
			}

			batch := make([]*messaging.Message, batchSize)
			for i := 0; i < batchSize; i++ {
				seq := produced + i
				batch[i] = createMessage(workload, target, seq, payload)
			}

			select {
			case <-ctx.Done():
				return
			case jobs <- batch:
				produced += batchSize
				remaining -= batchSize
			}
		}
	}()

	wg.Wait()

	successful := int(atomic.LoadInt64(&successCount))
	recordedFailures := int(atomic.LoadInt64(&failureCount))
	pending := metrics.TotalMessages - successful - recordedFailures
	if pending < 0 {
		pending = 0
	}
	metrics.Successful = successful
	metrics.Failed = recordedFailures + pending
	metrics.Duration = time.Since(start)
	if metrics.Duration <= 0 {
		metrics.Duration = time.Nanosecond
	}
	metrics.Throughput = float64(metrics.Successful) / metrics.Duration.Seconds()

	metrics.ErrorSummaries = collapseErrors(errorCounts)
	metrics.AverageLatency, metrics.P50Latency, metrics.P95Latency, metrics.P99Latency, metrics.MaxLatency = computeLatencyMetrics(latencies)

	return Result{
		TargetName:   target.Name,
		BrokerType:   target.Config.Type,
		WorkloadName: workload.Name,
		Workload:     workload,
		Metrics:      metrics,
	}
}

func incrementError(mu *sync.Mutex, counts map[string]int, message string) {
	mu.Lock()
	defer mu.Unlock()
	counts[message]++
}

func collapseErrors(counts map[string]int) []ErrorSummary {
	if len(counts) == 0 {
		return nil
	}
	summaries := make([]ErrorSummary, 0, len(counts))
	for msg, count := range counts {
		summaries = append(summaries, ErrorSummary{Message: msg, Count: count})
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Count == summaries[j].Count {
			return summaries[i].Message < summaries[j].Message
		}
		return summaries[i].Count > summaries[j].Count
	})
	return summaries
}

func computeLatencyMetrics(latencies []time.Duration) (avg, p50, p95, p99, max time.Duration) {
	if len(latencies) == 0 {
		return 0, 0, 0, 0, 0
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	var sum time.Duration
	for _, v := range latencies {
		sum += v
	}

	avg = sum / time.Duration(len(latencies))
	p50 = percentile(latencies, 0.50)
	p95 = percentile(latencies, 0.95)
	p99 = percentile(latencies, 0.99)
	max = latencies[len(latencies)-1]
	return
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(math.Ceil(p*float64(len(sorted))) - 1)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func makePayload(size int) []byte {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte('A' + (i % 26))
	}
	return buf
}

func createMessage(workload Workload, target Target, seq int, payloadTemplate []byte) *messaging.Message {
	payload := make([]byte, len(payloadTemplate))
	copy(payload, payloadTemplate)

	headers := map[string]string{
		"benchmark": "true",
		"workload":  workload.Name,
		"sequence":  strconv.Itoa(seq),
	}

	if target.Description != "" {
		headers["target"] = target.Description
	}

	return &messaging.Message{
		ID:        fmt.Sprintf("%s-%d", sanitizeName(workload.Name), seq),
		Topic:     workload.Topic,
		Payload:   payload,
		Timestamp: time.Now(),
		Headers:   headers,
		Key:       []byte(fmt.Sprintf("key-%d", seq%16)),
	}
}

func sanitizeName(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return "workload"
	}

	var b strings.Builder
	b.Grow(len(input))
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune('-')
		case r == '.' || r == ':' || r == '/':
			b.WriteRune('-')
		case r == ' ':
			b.WriteRune('-')
		}
	}

	sanitized := strings.Trim(b.String(), "-")
	if sanitized == "" {
		return "workload"
	}
	return sanitized
}

// ErrorRate returns the percentage of failed messages relative to the total messages attempted.
func (m Metrics) ErrorRate() float64 {
	if m.TotalMessages == 0 {
		return 0
	}
	return float64(m.Failed) / float64(m.TotalMessages)
}

// LatencyHistogram returns a human-readable summary of high-percentile latencies.
func (m Metrics) LatencyHistogram() string {
	if m.Successful == 0 {
		return "no successful publishes"
	}
	parts := []string{
		fmt.Sprintf("avg=%s", m.AverageLatency),
		fmt.Sprintf("p50=%s", m.P50Latency),
		fmt.Sprintf("p95=%s", m.P95Latency),
		fmt.Sprintf("p99=%s", m.P99Latency),
		fmt.Sprintf("max=%s", m.MaxLatency),
	}
	return strings.Join(parts, ", ")
}

// ErrorDetails returns the error summaries formatted as a single string.
func (m Metrics) ErrorDetails() string {
	if len(m.ErrorSummaries) == 0 {
		return ""
	}
	parts := make([]string, len(m.ErrorSummaries))
	for i, summary := range m.ErrorSummaries {
		parts[i] = fmt.Sprintf("%s (%d)", summary.Message, summary.Count)
	}
	return strings.Join(parts, "; ")
}

// ToCSV renders the report in CSV format suitable for spreadsheet import.
func (r *Report) ToCSV() string {
	var b strings.Builder
	b.WriteString("broker,broker_type,workload,messages,publishers,batch_size,duration_seconds,throughput_msgs_per_sec,avg_latency_ms,p95_latency_ms,error_rate,failures,error_details\n")
	for _, result := range r.Results {
		durationSeconds := result.Metrics.Duration.Seconds()
		avgLatencyMs := float64(result.Metrics.AverageLatency) / float64(time.Millisecond)
		p95LatencyMs := float64(result.Metrics.P95Latency) / float64(time.Millisecond)

		b.WriteString(fmt.Sprintf(
			"%s,%s,%s,%d,%d,%d,%.4f,%.2f,%.2f,%.2f,%.4f,%d,%q\n",
			sanitizeForCSV(result.TargetName),
			result.BrokerType.String(),
			sanitizeForCSV(result.WorkloadName),
			result.Workload.Messages,
			result.Workload.Publishers,
			result.Workload.BatchSize,
			durationSeconds,
			result.Metrics.Throughput,
			avgLatencyMs,
			p95LatencyMs,
			result.Metrics.ErrorRate(),
			result.Metrics.Failed,
			result.Metrics.ErrorDetails(),
		))
	}
	return b.String()
}

// ToMarkdown renders the report as a Markdown table.
func (r *Report) ToMarkdown() string {
	var b strings.Builder
	b.WriteString("| Broker | Type | Workload | Messages | Publishers | Batch | Duration (s) | Throughput (msg/s) | Avg Latency (ms) | P95 Latency (ms) | Failures |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")

	for _, result := range r.Results {
		avgLatencyMs := float64(result.Metrics.AverageLatency) / float64(time.Millisecond)
		p95LatencyMs := float64(result.Metrics.P95Latency) / float64(time.Millisecond)
		b.WriteString(fmt.Sprintf(
			"| %s | %s | %s | %d | %d | %d | %.2f | %.2f | %.2f | %.2f | %d |\n",
			escapeMarkdown(result.TargetName),
			result.BrokerType.String(),
			escapeMarkdown(result.WorkloadName),
			result.Workload.Messages,
			result.Workload.Publishers,
			result.Workload.BatchSize,
			result.Metrics.Duration.Seconds(),
			result.Metrics.Throughput,
			avgLatencyMs,
			p95LatencyMs,
			result.Metrics.Failed,
		))
	}

	return b.String()
}

func sanitizeForCSV(input string) string {
	input = strings.ReplaceAll(input, "\n", " ")
	input = strings.ReplaceAll(input, ",", " ")
	return input
}

func escapeMarkdown(input string) string {
	replacer := strings.NewReplacer("|", "\\|", "\n", " ", "`", "'", "*", "\\*", "_", "\\_")
	return replacer.Replace(input)
}
