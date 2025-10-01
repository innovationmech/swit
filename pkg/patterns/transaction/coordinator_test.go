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

package transaction

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockParticipant 模拟事务参与者
type mockParticipant struct {
	name               string
	prepareShouldFail  bool
	commitShouldFail   bool
	rollbackShouldFail bool
	prepareDelay       time.Duration
	commitDelay        time.Duration
	rollbackDelay      time.Duration

	prepareCalls  int
	commitCalls   int
	rollbackCalls int
	mu            sync.Mutex
}

func newMockParticipant(name string) *mockParticipant {
	return &mockParticipant{
		name: name,
	}
}

func (m *mockParticipant) Prepare(ctx context.Context, txID string) error {
	m.mu.Lock()
	m.prepareCalls++
	m.mu.Unlock()

	if m.prepareDelay > 0 {
		time.Sleep(m.prepareDelay)
	}

	if m.prepareShouldFail {
		return fmt.Errorf("%s: prepare failed", m.name)
	}

	return nil
}

func (m *mockParticipant) Commit(ctx context.Context, txID string) error {
	m.mu.Lock()
	m.commitCalls++
	m.mu.Unlock()

	if m.commitDelay > 0 {
		time.Sleep(m.commitDelay)
	}

	if m.commitShouldFail {
		return fmt.Errorf("%s: commit failed", m.name)
	}

	return nil
}

func (m *mockParticipant) Rollback(ctx context.Context, txID string) error {
	m.mu.Lock()
	m.rollbackCalls++
	m.mu.Unlock()

	if m.rollbackDelay > 0 {
		time.Sleep(m.rollbackDelay)
	}

	if m.rollbackShouldFail {
		return fmt.Errorf("%s: rollback failed", m.name)
	}

	return nil
}

func (m *mockParticipant) GetName() string {
	return m.name
}

func (m *mockParticipant) getCalls() (prepare, commit, rollback int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.prepareCalls, m.commitCalls, m.rollbackCalls
}

// TestCoordinatorBasic 测试基本协调器功能
func TestCoordinatorBasic(t *testing.T) {
	config := &TransactionConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
	}

	coordinator := NewCoordinator(config)

	// 测试创建事务
	participants := []TransactionParticipant{
		newMockParticipant("participant1"),
		newMockParticipant("participant2"),
	}

	txExec, err := coordinator.Begin(context.Background(), participants)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	if txExec.ID == "" {
		t.Error("Transaction ID should not be empty")
	}

	if txExec.State != StateInitial {
		t.Errorf("Expected state %v, got %v", StateInitial, txExec.State)
	}

	// 测试获取事务
	retrieved, err := coordinator.GetTransaction(txExec.ID)
	if err != nil {
		t.Fatalf("GetTransaction failed: %v", err)
	}

	if retrieved.ID != txExec.ID {
		t.Errorf("Expected transaction ID %s, got %s", txExec.ID, retrieved.ID)
	}
}

// TestTwoPhaseCommitSuccess 测试 2PC 成功场景
func TestTwoPhaseCommitSuccess(t *testing.T) {
	config := &TransactionConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      2,
		RetryDelay:      10 * time.Millisecond,
		DefaultStrategy: "2pc",
	}

	coordinator := NewCoordinator(config)

	p1 := newMockParticipant("participant1")
	p2 := newMockParticipant("participant2")
	participants := []TransactionParticipant{p1, p2}

	err := coordinator.BeginAndExecute(context.Background(), participants, func() error {
		// 模拟业务逻辑
		return nil
	})

	if err != nil {
		t.Fatalf("Transaction should succeed, got error: %v", err)
	}

	// 验证调用次数
	prepare1, commit1, rollback1 := p1.getCalls()
	prepare2, commit2, rollback2 := p2.getCalls()

	if prepare1 != 1 {
		t.Errorf("Participant1: expected 1 prepare call, got %d", prepare1)
	}
	if commit1 != 1 {
		t.Errorf("Participant1: expected 1 commit call, got %d", commit1)
	}
	if rollback1 != 0 {
		t.Errorf("Participant1: expected 0 rollback calls, got %d", rollback1)
	}

	if prepare2 != 1 {
		t.Errorf("Participant2: expected 1 prepare call, got %d", prepare2)
	}
	if commit2 != 1 {
		t.Errorf("Participant2: expected 1 commit call, got %d", commit2)
	}
	if rollback2 != 0 {
		t.Errorf("Participant2: expected 0 rollback calls, got %d", rollback2)
	}
}

// TestTwoPhaseCommitPrepareFail 测试 2PC Prepare 失败场景
func TestTwoPhaseCommitPrepareFail(t *testing.T) {
	config := &TransactionConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      0, // 不重试
		DefaultStrategy: "2pc",
	}

	coordinator := NewCoordinator(config)

	p1 := newMockParticipant("participant1")
	p2 := newMockParticipant("participant2")
	p2.prepareShouldFail = true // Participant2 准备失败

	participants := []TransactionParticipant{p1, p2}

	err := coordinator.BeginAndExecute(context.Background(), participants, func() error {
		return nil
	})

	if err == nil {
		t.Fatal("Transaction should fail when prepare fails")
	}

	// 验证两个参与者都被回滚
	_, _, rollback1 := p1.getCalls()
	_, _, rollback2 := p2.getCalls()

	if rollback1 != 1 {
		t.Errorf("Participant1: expected 1 rollback call, got %d", rollback1)
	}
	if rollback2 != 1 {
		t.Errorf("Participant2: expected 1 rollback call, got %d", rollback2)
	}
}

// TestTwoPhaseCommitBusinessLogicFail 测试 2PC 业务逻辑失败场景
func TestTwoPhaseCommitBusinessLogicFail(t *testing.T) {
	config := &TransactionConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      0,
		DefaultStrategy: "2pc",
	}

	coordinator := NewCoordinator(config)

	p1 := newMockParticipant("participant1")
	p2 := newMockParticipant("participant2")
	participants := []TransactionParticipant{p1, p2}

	businessErr := errors.New("business logic error")

	err := coordinator.BeginAndExecute(context.Background(), participants, func() error {
		return businessErr
	})

	if err == nil {
		t.Fatal("Transaction should fail when business logic fails")
	}

	if !errors.Is(err, businessErr) {
		t.Errorf("Expected business logic error, got: %v", err)
	}

	// 验证两个参与者都被回滚
	prepare1, commit1, rollback1 := p1.getCalls()
	prepare2, commit2, rollback2 := p2.getCalls()

	if prepare1 != 1 {
		t.Errorf("Participant1: expected 1 prepare call, got %d", prepare1)
	}
	if commit1 != 0 {
		t.Errorf("Participant1: expected 0 commit calls, got %d", commit1)
	}
	if rollback1 != 1 {
		t.Errorf("Participant1: expected 1 rollback call, got %d", rollback1)
	}

	if prepare2 != 1 {
		t.Errorf("Participant2: expected 1 prepare call, got %d", prepare2)
	}
	if commit2 != 0 {
		t.Errorf("Participant2: expected 0 commit calls, got %d", commit2)
	}
	if rollback2 != 1 {
		t.Errorf("Participant2: expected 1 rollback call, got %d", rollback2)
	}
}

// TestCompensationSuccess 测试补偿策略成功场景
func TestCompensationSuccess(t *testing.T) {
	config := &TransactionConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      0,
		DefaultStrategy: "compensation",
	}

	coordinator := NewCoordinator(config)

	p1 := newMockParticipant("participant1")
	p2 := newMockParticipant("participant2")
	participants := []TransactionParticipant{p1, p2}

	err := coordinator.BeginAndExecute(context.Background(), participants, func() error {
		return nil
	})

	if err != nil {
		t.Fatalf("Transaction should succeed, got error: %v", err)
	}

	// 补偿策略不调用 Prepare，直接 Commit
	prepare1, commit1, rollback1 := p1.getCalls()
	prepare2, commit2, rollback2 := p2.getCalls()

	if prepare1 != 0 {
		t.Errorf("Participant1: expected 0 prepare calls, got %d", prepare1)
	}
	if commit1 != 1 {
		t.Errorf("Participant1: expected 1 commit call, got %d", commit1)
	}
	if rollback1 != 0 {
		t.Errorf("Participant1: expected 0 rollback calls, got %d", rollback1)
	}

	if prepare2 != 0 {
		t.Errorf("Participant2: expected 0 prepare calls, got %d", prepare2)
	}
	if commit2 != 1 {
		t.Errorf("Participant2: expected 1 commit call, got %d", commit2)
	}
	if rollback2 != 0 {
		t.Errorf("Participant2: expected 0 rollback calls, got %d", rollback2)
	}
}

// TestCompensationFail 测试补偿策略失败场景
func TestCompensationFail(t *testing.T) {
	config := &TransactionConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      0,
		DefaultStrategy: "compensation",
	}

	coordinator := NewCoordinator(config)

	p1 := newMockParticipant("participant1")
	p2 := newMockParticipant("participant2")
	p2.commitShouldFail = true // Participant2 提交失败

	participants := []TransactionParticipant{p1, p2}

	err := coordinator.BeginAndExecute(context.Background(), participants, func() error {
		return nil
	})

	if err == nil {
		t.Fatal("Transaction should fail when participant commit fails")
	}

	// P1 应该被执行然后被补偿
	// P2 尝试提交但失败，由于没有成功执行，不需要补偿
	_, commit1, rollback1 := p1.getCalls()
	_, commit2, rollback2 := p2.getCalls()

	if commit1 != 1 {
		t.Errorf("Participant1: expected 1 commit call, got %d", commit1)
	}
	if rollback1 != 1 {
		t.Errorf("Participant1: expected 1 rollback call (compensation), got %d", rollback1)
	}

	if commit2 < 1 { // 至少尝试提交一次（可能重试）
		t.Errorf("Participant2: expected at least 1 commit attempt, got %d", commit2)
	}
	// P2 的 commit 失败，没有成功执行，因此不需要补偿
	if rollback2 != 0 {
		t.Errorf("Participant2: expected 0 rollback calls (no compensation needed for failed commit), got %d", rollback2)
	}
}

// TestStrategySwitch 测试策略切换
func TestStrategySwitch(t *testing.T) {
	config := &TransactionConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      0,
		DefaultStrategy: "2pc",
	}

	coordinator := NewCoordinator(config)

	// 验证默认是 2PC
	if coordinator.strategy.GetName() != "2pc" {
		t.Errorf("Expected default strategy '2pc', got '%s'", coordinator.strategy.GetName())
	}

	// 切换到补偿策略
	compensationStrategy := NewCompensationStrategy(config)
	coordinator.SetStrategy(compensationStrategy)

	if coordinator.strategy.GetName() != "compensation" {
		t.Errorf("Expected strategy 'compensation', got '%s'", coordinator.strategy.GetName())
	}
}

// TestContextCancellation 测试上下文取消
func TestContextCancellation(t *testing.T) {
	config := &TransactionConfig{
		Timeout:         10 * time.Second,
		MaxRetries:      0,
		DefaultStrategy: "2pc",
	}

	coordinator := NewCoordinator(config)

	p1 := newMockParticipant("participant1")
	p1.prepareDelay = 500 * time.Millisecond
	participants := []TransactionParticipant{p1}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := coordinator.BeginAndExecute(ctx, participants, func() error {
		return nil
	})

	if err == nil {
		t.Fatal("Transaction should fail when context is canceled")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
}

// TestCleanupCompletedTransactions 测试清理已完成的事务
func TestCleanupCompletedTransactions(t *testing.T) {
	config := &TransactionConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}

	coordinator := NewCoordinator(config)

	// 创建一些事务
	for i := 0; i < 5; i++ {
		p := newMockParticipant(fmt.Sprintf("participant%d", i))
		participants := []TransactionParticipant{p}

		_ = coordinator.BeginAndExecute(context.Background(), participants, func() error {
			return nil
		})

		// 添加延迟以确保时间差异
		time.Sleep(10 * time.Millisecond)
	}

	// 等待一段时间
	time.Sleep(100 * time.Millisecond)

	// 清理 50ms 之前的事务
	cleaned := coordinator.CleanupCompletedTransactions(50 * time.Millisecond)

	if cleaned == 0 {
		t.Error("Should have cleaned up some transactions")
	}

	t.Logf("Cleaned up %d transactions", cleaned)
}

// BenchmarkTwoPhaseCommit 基准测试 2PC
func BenchmarkTwoPhaseCommit(b *testing.B) {
	config := &TransactionConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      0,
		DefaultStrategy: "2pc",
	}

	coordinator := NewCoordinator(config)

	participants := []TransactionParticipant{
		newMockParticipant("p1"),
		newMockParticipant("p2"),
		newMockParticipant("p3"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = coordinator.BeginAndExecute(context.Background(), participants, func() error {
			return nil
		})
	}
}

// BenchmarkCompensation 基准测试补偿策略
func BenchmarkCompensation(b *testing.B) {
	config := &TransactionConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      0,
		DefaultStrategy: "compensation",
	}

	coordinator := NewCoordinator(config)

	participants := []TransactionParticipant{
		newMockParticipant("p1"),
		newMockParticipant("p2"),
		newMockParticipant("p3"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = coordinator.BeginAndExecute(context.Background(), participants, func() error {
			return nil
		})
	}
}
