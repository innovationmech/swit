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

// Package benchmarks 提供 Saga 系统的性能基准测试套件
//
// 本包包含以下基准测试类别：
//  1. Orchestrator 执行性能测试
//  2. 并发 Saga 执行测试
//  3. 状态持久化性能测试
//  4. DSL 解析性能测试
//  5. 消息发布性能测试
//
// 使用方法：
//
//	go test -bench=. -benchmem -benchtime=5s
//
// 查看 README.md 获取完整的基准测试文档
package benchmarks

import (
	"testing"
)

// BenchmarkExample 示例基准测试
//
// 此函数演示了如何编写 Saga 系统的基准测试。
// 实际的基准测试需要根据具体的 Saga 组件接口实现。
//
// 基准测试应该：
//  1. 使用 b.ResetTimer() 重置计时器
//  2. 使用 b.ReportAllocs() 报告内存分配
//  3. 使用 b.N 控制迭代次数
//  4. 避免在基准测试中执行昂贵的初始化操作
func BenchmarkExample(b *testing.B) {
	// 初始化测试数据
	data := make([]byte, 1024)

	// 重置计时器，排除初始化时间
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 执行被测试的操作
		_ = append([]byte{}, data...)
	}
}

// BenchmarkExampleParallel 示例并发基准测试
//
// 演示如何编写并发场景的基准测试。
func BenchmarkExampleParallel(b *testing.B) {
	// 初始化测试数据
	data := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	// 使用 RunParallel 进行并发测试
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = append([]byte{}, data...)
		}
	})
}

// TODO: 实现以下基准测试
//
// 1. Orchestrator 执行性能测试 (orchestrator_bench_test.go)
//    - BenchmarkOrchestratorSimpleSaga
//    - BenchmarkOrchestratorComplexSaga
//    - BenchmarkOrchestratorWithCompensation
//    - BenchmarkOrchestratorStateTransitions
//    - BenchmarkOrchestratorMemoryAllocation
//
// 2. 并发 Saga 执行测试 (concurrent_bench_test.go)
//    - BenchmarkConcurrentSagaExecution
//    - BenchmarkConcurrentSagaWithSharedState
//    - BenchmarkConcurrentSagaThroughput
//    - BenchmarkConcurrentSagaLatency
//
// 3. 状态持久化性能测试 (state_bench_test.go)
//    - BenchmarkStateStorageSave
//    - BenchmarkStateStorageLoad
//    - BenchmarkStateStorageUpdate
//    - BenchmarkStateStorageBatchSave
//    - BenchmarkStateStorageConcurrent
//
// 4. DSL 解析性能测试 (dsl_bench_test.go)
//    - BenchmarkDSLParserSimple
//    - BenchmarkDSLParserComplex
//    - BenchmarkDSLValidation
//    - BenchmarkDSLParserConcurrent
//
// 5. 消息发布性能测试 (messaging_bench_test.go)
//    - BenchmarkMessagePublishSingle
//    - BenchmarkMessagePublishBatch
//    - BenchmarkMessagePublishAsync
//    - BenchmarkMessageSerialization
//
// 实现指南：
//  - 参考 pkg/saga/retry/benchmark_test.go 的实现模式
//  - 参考 pkg/saga/messaging/publisher_bench_test.go 的实现模式
//  - 使用 pkg/saga/examples 中的示例作为测试数据
//  - 确保基准测试可以独立运行，不依赖外部服务
//  - 使用 mock 对象模拟外部依赖
//  - 记录关键性能指标（吞吐量、延迟、内存使用）
