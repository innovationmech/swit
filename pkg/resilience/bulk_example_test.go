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

package resilience_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/resilience"
)

// Example_bulkExecutor_basicUsage 演示批量操作的基本用法
func Example_bulkExecutor_basicUsage() {
	// 创建批量执行器配置
	config := resilience.BulkConfig{
		Concurrency:     5,    // 并发度为 5
		ContinueOnError: true, // 遇到错误继续执行
	}

	// 创建批量执行器
	executor, err := resilience.NewBulkExecutor[int, int](config)
	if err != nil {
		panic(err)
	}

	// 准备批量操作
	operations := []resilience.BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
		{Index: 2, Data: 3},
		{Index: 3, Data: 4},
		{Index: 4, Data: 5},
	}

	// 定义操作函数
	operationFn := func(ctx context.Context, data int) (int, error) {
		// 模拟业务操作
		result := data * 2
		return result, nil
	}

	// 执行批量操作
	report := executor.Execute(context.Background(), operations, operationFn)

	// 输出结果
	fmt.Printf("Total: %d, Success: %d, Failure: %d\n",
		report.TotalCount, report.SuccessCount, report.FailureCount)

	// Output:
	// Total: 5, Success: 5, Failure: 0
}

// Example_bulkExecutor_partialFailure 演示部分失败的处理
func Example_bulkExecutor_partialFailure() {
	config := resilience.BulkConfig{
		Concurrency:     3,
		ContinueOnError: true, // 尽力而为，即使部分失败也继续
	}

	executor, err := resilience.NewBulkExecutor[string, string](config)
	if err != nil {
		panic(err)
	}

	operations := []resilience.BulkOperation[string]{
		{Index: 0, Data: "user1"},
		{Index: 1, Data: "user2"},
		{Index: 2, Data: ""}, // 空字符串会导致失败
		{Index: 3, Data: "user3"},
		{Index: 4, Data: "user4"},
	}

	operationFn := func(ctx context.Context, data string) (string, error) {
		if data == "" {
			return "", errors.New("empty user ID")
		}
		return fmt.Sprintf("processed_%s", data), nil
	}

	report := executor.Execute(context.Background(), operations, operationFn)

	// 检查是否有部分成功
	if report.HasPartialSuccess() {
		fmt.Println("Partial success detected")
		fmt.Printf("Successful: %d, Failed: %d\n",
			report.SuccessCount, report.FailureCount)

		// 获取失败的结果
		failedResults := report.GetFailedResults()
		for _, result := range failedResults {
			fmt.Printf("Failed at index %d: %v\n", result.Index, result.Error)
		}
	}

	// Output:
	// Partial success detected
	// Successful: 4, Failed: 1
	// Failed at index 2: empty user ID
}

// Example_bulkExecutor_withRetry 演示带重试的批量操作
func Example_bulkExecutor_withRetry() {
	// 配置重试策略
	retryConfig := resilience.Config{
		Strategy:     resilience.StrategyExponential,
		MaxRetries:   2,
		InitialDelay: 100 * time.Millisecond,
		Multiplier:   2.0,
	}

	config := resilience.BulkConfig{
		Concurrency:     0, // 串行执行以避免并发问题
		ContinueOnError: true,
		RetryEnabled:    true,
		RetryConfig:     &retryConfig,
	}

	executor, err := resilience.NewBulkExecutor[int, string](config)
	if err != nil {
		panic(err)
	}

	operations := []resilience.BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
		{Index: 2, Data: 3},
	}

	attemptCount := make(map[int]int)
	operationFn := func(ctx context.Context, data int) (string, error) {
		attemptCount[data]++
		// 前两次尝试失败，第三次成功
		if attemptCount[data] <= 2 {
			return "", fmt.Errorf("temporary failure for %d", data)
		}
		return fmt.Sprintf("success_%d", data), nil
	}

	report := executor.Execute(context.Background(), operations, operationFn)

	if report.IsFullSuccess() {
		fmt.Println("All operations succeeded after retries")
		fmt.Printf("Total attempts: %d\n", attemptCount[1]+attemptCount[2]+attemptCount[3])
	}

	// Output:
	// All operations succeeded after retries
	// Total attempts: 9
}

// Example_bulkExecutor_progressTracking 演示进度追踪
func Example_bulkExecutor_progressTracking() {
	config := resilience.BulkConfig{
		Concurrency:     0, // 串行执行以保证进度顺序
		ContinueOnError: true,
		OnProgress: func(completed, total int) {
			percentage := (completed * 100) / total
			fmt.Printf("Progress: %d%% (%d/%d)\n", percentage, completed, total)
		},
	}

	executor, err := resilience.NewBulkExecutor[int, int](config)
	if err != nil {
		panic(err)
	}

	operations := []resilience.BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
		{Index: 2, Data: 3},
		{Index: 3, Data: 4},
	}

	operationFn := func(ctx context.Context, data int) (int, error) {
		time.Sleep(10 * time.Millisecond) // 模拟耗时操作
		return data * 2, nil
	}

	executor.Execute(context.Background(), operations, operationFn)

	// Output:
	// Progress: 25% (1/4)
	// Progress: 50% (2/4)
	// Progress: 75% (3/4)
	// Progress: 100% (4/4)
}

// Example_bulkExecutor_failFast 演示快速失败模式
func Example_bulkExecutor_failFast() {
	config := resilience.BulkConfig{
		Concurrency:     0,     // 串行执行
		ContinueOnError: false, // 遇到错误立即停止
	}

	executor, err := resilience.NewBulkExecutor[int, int](config)
	if err != nil {
		panic(err)
	}

	operations := []resilience.BulkOperation[int]{
		{Index: 0, Data: 1},
		{Index: 1, Data: 2},
		{Index: 2, Data: 0}, // 会导致失败
		{Index: 3, Data: 3},
		{Index: 4, Data: 4},
	}

	operationFn := func(ctx context.Context, data int) (int, error) {
		if data == 0 {
			return 0, errors.New("zero not allowed")
		}
		return data * 2, nil
	}

	report := executor.Execute(context.Background(), operations, operationFn)

	fmt.Printf("Processed: %d, Skipped: %d\n",
		report.SuccessCount,
		report.TotalCount-report.SuccessCount-1) // -1 for the failed one

	// Output:
	// Processed: 2, Skipped: 2
}

// Example_bulkExecutor_complexData 演示处理复杂数据类型
func Example_bulkExecutor_complexData() {
	type User struct {
		ID   int
		Name string
	}

	type UserResult struct {
		UserID  int
		Status  string
		Message string
	}

	config := resilience.DefaultBulkConfig()
	config.Concurrency = 3

	executor, err := resilience.NewBulkExecutor[User, UserResult](config)
	if err != nil {
		panic(err)
	}

	operations := []resilience.BulkOperation[User]{
		{Index: 0, Data: User{ID: 1, Name: "Alice"}},
		{Index: 1, Data: User{ID: 2, Name: "Bob"}},
		{Index: 2, Data: User{ID: 3, Name: "Charlie"}},
	}

	operationFn := func(ctx context.Context, user User) (UserResult, error) {
		// 模拟用户处理逻辑
		return UserResult{
			UserID:  user.ID,
			Status:  "processed",
			Message: fmt.Sprintf("User %s processed successfully", user.Name),
		}, nil
	}

	report := executor.Execute(context.Background(), operations, operationFn)

	for _, result := range report.GetSuccessfulResults() {
		fmt.Printf("User %d: %s\n", result.Data.UserID, result.Data.Status)
	}

	// Output:
	// User 1: processed
	// User 2: processed
	// User 3: processed
}

// Example_bulkReport_analysis 演示如何分析批量操作报告
func Example_bulkReport_analysis() {
	// 模拟一个包含部分失败的报告
	report := &resilience.BulkReport[string]{
		TotalCount:    10,
		SuccessCount:  7,
		FailureCount:  3,
		TotalDuration: 500 * time.Millisecond,
	}

	fmt.Printf("Report: %s\n", report.String())

	if report.IsFullSuccess() {
		fmt.Println("Status: All operations successful")
	} else if report.IsFullFailure() {
		fmt.Println("Status: All operations failed")
	} else if report.HasPartialSuccess() {
		fmt.Println("Status: Partial success")
		successRate := (float64(report.SuccessCount) / float64(report.TotalCount)) * 100
		fmt.Printf("Success rate: %.1f%%\n", successRate)
	}

	avgDuration := report.TotalDuration / time.Duration(report.TotalCount)
	fmt.Printf("Average duration per operation: %s\n", avgDuration)

	// Output:
	// Report: BulkReport{Total: 10, Success: 7, Failure: 3, Duration: 500ms}
	// Status: Partial success
	// Success rate: 70.0%
	// Average duration per operation: 50ms
}
