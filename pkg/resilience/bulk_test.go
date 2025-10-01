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

package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBulkExecutor_Execute_Sequential(t *testing.T) {
	config := BulkConfig{
		Concurrency:     0, // 串行
		ContinueOnError: true,
	}

	executor, err := NewBulkExecutor[int, int](config)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	operations := []BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
		{Index: 2, Data: 3},
		{Index: 3, Data: 4},
		{Index: 4, Data: 5},
	}

	operationFn := func(ctx context.Context, data int) (int, error) {
		return data * 2, nil
	}

	report := executor.Execute(context.Background(), operations, operationFn)

	if !report.IsFullSuccess() {
		t.Errorf("Expected full success, got: %s", report.String())
	}

	if report.SuccessCount != 5 {
		t.Errorf("Expected 5 successes, got %d", report.SuccessCount)
	}

	for i, result := range report.Results {
		expected := operations[i].Data * 2
		if result.Data != expected {
			t.Errorf("Result[%d]: expected %d, got %d", i, expected, result.Data)
		}
	}
}

func TestBulkExecutor_Execute_Concurrent(t *testing.T) {
	config := BulkConfig{
		Concurrency:     3, // 并发度为 3
		ContinueOnError: true,
	}

	executor, err := NewBulkExecutor[int, int](config)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	operations := []BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
		{Index: 2, Data: 3},
		{Index: 3, Data: 4},
		{Index: 4, Data: 5},
	}

	var counter atomic.Int32
	operationFn := func(ctx context.Context, data int) (int, error) {
		counter.Add(1)
		time.Sleep(10 * time.Millisecond) // 模拟耗时操作
		return data * 2, nil
	}

	report := executor.Execute(context.Background(), operations, operationFn)

	if !report.IsFullSuccess() {
		t.Errorf("Expected full success, got: %s", report.String())
	}

	if report.SuccessCount != 5 {
		t.Errorf("Expected 5 successes, got %d", report.SuccessCount)
	}
}

func TestBulkExecutor_PartialFailure_ContinueOnError(t *testing.T) {
	config := BulkConfig{
		Concurrency:     2,
		ContinueOnError: true, // 继续执行
	}

	executor, err := NewBulkExecutor[int, int](config)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	operations := []BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
		{Index: 2, Data: 0}, // 会失败
		{Index: 3, Data: 4},
		{Index: 4, Data: 0}, // 会失败
	}

	operationFn := func(ctx context.Context, data int) (int, error) {
		if data == 0 {
			return 0, errors.New("zero value not allowed")
		}
		return data * 2, nil
	}

	report := executor.Execute(context.Background(), operations, operationFn)

	if !report.HasPartialSuccess() {
		t.Errorf("Expected partial success, got: %s", report.String())
	}

	if report.SuccessCount != 3 {
		t.Errorf("Expected 3 successes, got %d", report.SuccessCount)
	}

	if report.FailureCount != 2 {
		t.Errorf("Expected 2 failures, got %d", report.FailureCount)
	}

	successResults := report.GetSuccessfulResults()
	if len(successResults) != 3 {
		t.Errorf("Expected 3 successful results, got %d", len(successResults))
	}

	failedResults := report.GetFailedResults()
	if len(failedResults) != 2 {
		t.Errorf("Expected 2 failed results, got %d", len(failedResults))
	}
}

func TestBulkExecutor_PartialFailure_FailFast(t *testing.T) {
	config := BulkConfig{
		Concurrency:     0,     // 串行
		ContinueOnError: false, // 快速失败
	}

	executor, err := NewBulkExecutor[int, int](config)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	operations := []BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
		{Index: 2, Data: 0}, // 会失败
		{Index: 3, Data: 4},
		{Index: 4, Data: 5},
	}

	operationFn := func(ctx context.Context, data int) (int, error) {
		if data == 0 {
			return 0, errors.New("zero value not allowed")
		}
		return data * 2, nil
	}

	report := executor.Execute(context.Background(), operations, operationFn)

	// 前两个应该成功，第三个失败，后面两个被跳过
	if report.SuccessCount != 2 {
		t.Errorf("Expected 2 successes, got %d", report.SuccessCount)
	}

	if report.FailureCount != 3 {
		t.Errorf("Expected 3 failures (1 real + 2 skipped), got %d", report.FailureCount)
	}
}

func TestBulkExecutor_WithRetry(t *testing.T) {
	retryConfig := Config{
		Strategy:     StrategyExponential,
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
	}

	config := BulkConfig{
		Concurrency:     2,
		ContinueOnError: true,
		RetryEnabled:    true,
		RetryConfig:     &retryConfig,
	}

	executor, err := NewBulkExecutor[int, int](config)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	operations := []BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
	}

	attemptCount := make(map[int]*atomic.Int32)
	var mu sync.Mutex
	operationFn := func(ctx context.Context, data int) (int, error) {
		mu.Lock()
		if attemptCount[data] == nil {
			attemptCount[data] = &atomic.Int32{}
		}
		counter := attemptCount[data]
		mu.Unlock()

		count := counter.Add(1)
		if count <= 2 {
			return 0, fmt.Errorf("temporary failure for %d", data)
		}
		return data * 2, nil
	}

	report := executor.Execute(context.Background(), operations, operationFn)

	if !report.IsFullSuccess() {
		t.Errorf("Expected full success after retries, got: %s", report.String())
	}

	// 每个操作应该被尝试 3 次（首次 + 2 次重试）
	for _, op := range operations {
		if attemptCount[op.Data].Load() != 3 {
			t.Errorf("Expected 3 attempts for data %d, got %d", op.Data, attemptCount[op.Data].Load())
		}
	}
}

func TestBulkExecutor_ProgressCallback(t *testing.T) {
	var progressUpdates []int

	config := BulkConfig{
		Concurrency:     0,
		ContinueOnError: true,
		OnProgress: func(completed, total int) {
			progressUpdates = append(progressUpdates, completed)
		},
	}

	executor, err := NewBulkExecutor[int, int](config)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	operations := []BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
		{Index: 2, Data: 3},
	}

	operationFn := func(ctx context.Context, data int) (int, error) {
		return data * 2, nil
	}

	report := executor.Execute(context.Background(), operations, operationFn)

	if !report.IsFullSuccess() {
		t.Errorf("Expected full success, got: %s", report.String())
	}

	if len(progressUpdates) != 3 {
		t.Errorf("Expected 3 progress updates, got %d", len(progressUpdates))
	}

	expectedProgress := []int{1, 2, 3}
	for i, expected := range expectedProgress {
		if progressUpdates[i] != expected {
			t.Errorf("Progress[%d]: expected %d, got %d", i, expected, progressUpdates[i])
		}
	}
}

func TestBulkExecutor_ContextCancellation(t *testing.T) {
	config := BulkConfig{
		Concurrency:     2,
		ContinueOnError: true,
	}

	executor, err := NewBulkExecutor[int, int](config)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	operations := []BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
		{Index: 2, Data: 3},
		{Index: 3, Data: 4},
		{Index: 4, Data: 5},
	}

	ctx, cancel := context.WithCancel(context.Background())

	operationFn := func(ctx context.Context, data int) (int, error) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return data * 2, nil
		}
	}

	// 在开始后立即取消
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	report := executor.Execute(ctx, operations, operationFn)

	// 应该有一些操作被取消
	if report.HasFailures() {
		canceledCount := 0
		for _, result := range report.Results {
			if result.Error == context.Canceled {
				canceledCount++
			}
		}
		t.Logf("Canceled operations: %d out of %d", canceledCount, report.TotalCount)
	}
}

func TestBulkReport_Methods(t *testing.T) {
	// 测试全部成功
	allSuccess := &BulkReport[int]{
		TotalCount:   5,
		SuccessCount: 5,
		FailureCount: 0,
		Results: []BulkResult[int]{
			{Index: 0, Data: 1, Error: nil},
			{Index: 1, Data: 2, Error: nil},
			{Index: 2, Data: 3, Error: nil},
			{Index: 3, Data: 4, Error: nil},
			{Index: 4, Data: 5, Error: nil},
		},
	}

	if !allSuccess.IsFullSuccess() {
		t.Error("Expected IsFullSuccess to be true")
	}
	if allSuccess.HasFailures() {
		t.Error("Expected HasFailures to be false")
	}
	if allSuccess.HasPartialSuccess() {
		t.Error("Expected HasPartialSuccess to be false")
	}
	if allSuccess.IsFullFailure() {
		t.Error("Expected IsFullFailure to be false")
	}

	// 测试全部失败
	allFailure := &BulkReport[int]{
		TotalCount:   3,
		SuccessCount: 0,
		FailureCount: 3,
		Results: []BulkResult[int]{
			{Index: 0, Data: 0, Error: errors.New("error 1")},
			{Index: 1, Data: 0, Error: errors.New("error 2")},
			{Index: 2, Data: 0, Error: errors.New("error 3")},
		},
	}

	if !allFailure.IsFullFailure() {
		t.Error("Expected IsFullFailure to be true")
	}
	if !allFailure.HasFailures() {
		t.Error("Expected HasFailures to be true")
	}
	if allFailure.HasPartialSuccess() {
		t.Error("Expected HasPartialSuccess to be false")
	}
	if allFailure.IsFullSuccess() {
		t.Error("Expected IsFullSuccess to be false")
	}

	// 测试部分成功
	partialSuccess := &BulkReport[int]{
		TotalCount:   5,
		SuccessCount: 3,
		FailureCount: 2,
		Results: []BulkResult[int]{
			{Index: 0, Data: 1, Error: nil},
			{Index: 1, Data: 2, Error: nil},
			{Index: 2, Data: 0, Error: errors.New("error")},
			{Index: 3, Data: 3, Error: nil},
			{Index: 4, Data: 0, Error: errors.New("error")},
		},
	}

	if !partialSuccess.HasPartialSuccess() {
		t.Error("Expected HasPartialSuccess to be true")
	}
	if !partialSuccess.HasFailures() {
		t.Error("Expected HasFailures to be true")
	}
	if partialSuccess.IsFullSuccess() {
		t.Error("Expected IsFullSuccess to be false")
	}
	if partialSuccess.IsFullFailure() {
		t.Error("Expected IsFullFailure to be false")
	}

	successResults := partialSuccess.GetSuccessfulResults()
	if len(successResults) != 3 {
		t.Errorf("Expected 3 successful results, got %d", len(successResults))
	}

	failedResults := partialSuccess.GetFailedResults()
	if len(failedResults) != 2 {
		t.Errorf("Expected 2 failed results, got %d", len(failedResults))
	}
}

func TestBulkConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  BulkConfig
		wantErr bool
	}{
		{
			name: "valid config without retry",
			config: BulkConfig{
				Concurrency:     5,
				ContinueOnError: true,
				RetryEnabled:    false,
			},
			wantErr: false,
		},
		{
			name: "valid config with retry",
			config: BulkConfig{
				Concurrency:     5,
				ContinueOnError: true,
				RetryEnabled:    true,
				RetryConfig: &Config{
					Strategy:     StrategyExponential,
					MaxRetries:   3,
					InitialDelay: time.Second,
					Multiplier:   2.0,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid retry config",
			config: BulkConfig{
				Concurrency:     5,
				ContinueOnError: true,
				RetryEnabled:    true,
				RetryConfig: &Config{
					Strategy:     StrategyExponential,
					MaxRetries:   -1, // 无效
					InitialDelay: time.Second,
					Multiplier:   2.0,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkBulkExecutor_Sequential(b *testing.B) {
	config := BulkConfig{
		Concurrency:     0,
		ContinueOnError: true,
	}

	executor, _ := NewBulkExecutor[int, int](config)

	operations := make([]BulkOperation[int], 100)
	for i := 0; i < 100; i++ {
		operations[i] = BulkOperation[int]{Index: i, Data: i}
	}

	operationFn := func(ctx context.Context, data int) (int, error) {
		return data * 2, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute(context.Background(), operations, operationFn)
	}
}

func BenchmarkBulkExecutor_Concurrent(b *testing.B) {
	config := BulkConfig{
		Concurrency:     10,
		ContinueOnError: true,
	}

	executor, _ := NewBulkExecutor[int, int](config)

	operations := make([]BulkOperation[int], 100)
	for i := 0; i < 100; i++ {
		operations[i] = BulkOperation[int]{Index: i, Data: i}
	}

	operationFn := func(ctx context.Context, data int) (int, error) {
		return data * 2, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute(context.Background(), operations, operationFn)
	}
}
