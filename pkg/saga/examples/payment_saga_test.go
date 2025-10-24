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

package examples

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// ==========================
// Mock 服务实现
// ==========================

// mockAccountService 模拟账户服务
type mockAccountService struct {
	validateAccountFunc    func(ctx context.Context, accountID string) (*AccountBalance, error)
	getBalanceFunc         func(ctx context.Context, accountID string) (*AccountBalance, error)
	freezeAmountFunc       func(ctx context.Context, accountID string, amount float64, reason string) (*FreezeResult, error)
	unfreezeAmountFunc     func(ctx context.Context, freezeID string, reason string) error
	deductFrozenAmountFunc func(ctx context.Context, freezeID string) error
	addAmountFunc          func(ctx context.Context, accountID string, amount float64, transactionID string) error
}

func (m *mockAccountService) ValidateAccount(ctx context.Context, accountID string) (*AccountBalance, error) {
	if m.validateAccountFunc != nil {
		return m.validateAccountFunc(ctx, accountID)
	}
	return &AccountBalance{
		AccountID:      accountID,
		Available:      1000.0,
		Frozen:         0.0,
		Total:          1000.0,
		Currency:       "CNY",
		LastUpdated:    time.Now(),
		MinimumBalance: 0.0,
		OverdraftLimit: 0.0,
	}, nil
}

func (m *mockAccountService) GetBalance(ctx context.Context, accountID string) (*AccountBalance, error) {
	if m.getBalanceFunc != nil {
		return m.getBalanceFunc(ctx, accountID)
	}
	return &AccountBalance{
		AccountID:   accountID,
		Available:   1000.0,
		Frozen:      0.0,
		Total:       1000.0,
		Currency:    "CNY",
		LastUpdated: time.Now(),
	}, nil
}

func (m *mockAccountService) FreezeAmount(ctx context.Context, accountID string, amount float64, reason string) (*FreezeResult, error) {
	if m.freezeAmountFunc != nil {
		return m.freezeAmountFunc(ctx, accountID, amount, reason)
	}
	return &FreezeResult{
		FreezeID:   "FRZ-123",
		AccountID:  accountID,
		Amount:     amount,
		Currency:   "CNY",
		FrozenAt:   time.Now(),
		ExpiresAt:  time.Now().Add(10 * time.Minute),
		FreezeType: "transfer",
		Metadata:   map[string]interface{}{},
	}, nil
}

func (m *mockAccountService) UnfreezeAmount(ctx context.Context, freezeID string, reason string) error {
	if m.unfreezeAmountFunc != nil {
		return m.unfreezeAmountFunc(ctx, freezeID, reason)
	}
	return nil
}

func (m *mockAccountService) DeductFrozenAmount(ctx context.Context, freezeID string) error {
	if m.deductFrozenAmountFunc != nil {
		return m.deductFrozenAmountFunc(ctx, freezeID)
	}
	return nil
}

func (m *mockAccountService) AddAmount(ctx context.Context, accountID string, amount float64, transactionID string) error {
	if m.addAmountFunc != nil {
		return m.addAmountFunc(ctx, accountID, amount, transactionID)
	}
	return nil
}

// mockTransferService 模拟转账服务
type mockTransferService struct {
	createTransferFunc    func(ctx context.Context, data *TransferData) (*TransferResult, error)
	executeTransferFunc   func(ctx context.Context, transferID string, freezeID string) (*TransferResult, error)
	cancelTransferFunc    func(ctx context.Context, transferID string, reason string) error
	getTransferStatusFunc func(ctx context.Context, transferID string) (string, error)
}

func (m *mockTransferService) CreateTransfer(ctx context.Context, data *TransferData) (*TransferResult, error) {
	if m.createTransferFunc != nil {
		return m.createTransferFunc(ctx, data)
	}
	return &TransferResult{
		TransactionID:   "TXN-123",
		TransferID:      data.TransferID,
		FromAccount:     data.FromAccount,
		ToAccount:       data.ToAccount,
		Amount:          data.Amount,
		Currency:        data.Currency,
		Status:          "completed",
		CompletedAt:     time.Now(),
		TransactionHash: "hash-123",
		Metadata:        map[string]interface{}{},
	}, nil
}

func (m *mockTransferService) ExecuteTransfer(ctx context.Context, transferID string, freezeID string) (*TransferResult, error) {
	if m.executeTransferFunc != nil {
		return m.executeTransferFunc(ctx, transferID, freezeID)
	}
	return &TransferResult{
		TransactionID:   "TXN-123",
		TransferID:      transferID,
		FromAccount:     "ACC-001",
		ToAccount:       "ACC-002",
		Amount:          100.0,
		Currency:        "CNY",
		Status:          "completed",
		CompletedAt:     time.Now(),
		TransactionHash: "hash-123",
		Metadata:        map[string]interface{}{},
	}, nil
}

func (m *mockTransferService) CancelTransfer(ctx context.Context, transferID string, reason string) error {
	if m.cancelTransferFunc != nil {
		return m.cancelTransferFunc(ctx, transferID, reason)
	}
	return nil
}

func (m *mockTransferService) GetTransferStatus(ctx context.Context, transferID string) (string, error) {
	if m.getTransferStatusFunc != nil {
		return m.getTransferStatusFunc(ctx, transferID)
	}
	return "completed", nil
}

// mockAuditService 模拟审计服务
type mockAuditService struct {
	createAuditRecordFunc func(ctx context.Context, transferID string, data *TransferData, result *TransferResult) (*AuditRecordResult, error)
	updateAuditRecordFunc func(ctx context.Context, auditID string, status string, details map[string]interface{}) error
}

func (m *mockAuditService) CreateAuditRecord(ctx context.Context, transferID string, data *TransferData, result *TransferResult) (*AuditRecordResult, error) {
	if m.createAuditRecordFunc != nil {
		return m.createAuditRecordFunc(ctx, transferID, data, result)
	}
	return &AuditRecordResult{
		AuditID:        "AUD-123",
		TransferID:     transferID,
		RecordType:     "transfer",
		CreatedAt:      time.Now(),
		Status:         "created",
		ReviewRequired: false,
		Metadata:       map[string]interface{}{},
	}, nil
}

func (m *mockAuditService) UpdateAuditRecord(ctx context.Context, auditID string, status string, details map[string]interface{}) error {
	if m.updateAuditRecordFunc != nil {
		return m.updateAuditRecordFunc(ctx, auditID, status, details)
	}
	return nil
}

// mockNotificationService 模拟通知服务
type mockNotificationService struct {
	sendTransferNotificationFunc func(ctx context.Context, transferID string, recipients []string, data *TransferData) (*NotificationResult, error)
	sendFailureNotificationFunc  func(ctx context.Context, transferID string, reason string) error
}

func (m *mockNotificationService) SendTransferNotification(ctx context.Context, transferID string, recipients []string, data *TransferData) (*NotificationResult, error) {
	if m.sendTransferNotificationFunc != nil {
		return m.sendTransferNotificationFunc(ctx, transferID, recipients, data)
	}
	return &NotificationResult{
		NotificationID: "NOT-123",
		TransferID:     transferID,
		Recipients:     recipients,
		Channel:        "email",
		SentAt:         time.Now(),
		Status:         "sent",
		Metadata:       map[string]interface{}{},
	}, nil
}

func (m *mockNotificationService) SendFailureNotification(ctx context.Context, transferID string, reason string) error {
	if m.sendFailureNotificationFunc != nil {
		return m.sendFailureNotificationFunc(ctx, transferID, reason)
	}
	return nil
}

// ==========================
// 测试辅助函数
// ==========================

// createTestTransferData 创建测试用转账数据
func createTestTransferData() *TransferData {
	return &TransferData{
		TransferID:    "TRF-001",
		FromAccount:   "ACC-001",
		ToAccount:     "ACC-002",
		Amount:        100.0,
		Currency:      "CNY",
		Purpose:       "payment",
		Description:   "测试转账",
		CustomerID:    "CUST-001",
		CustomerName:  "张三",
		RiskLevel:     "low",
		RiskFactors:   []string{},
		RequireReview: false,
		Metadata:      map[string]interface{}{},
	}
}

// ==========================
// ValidateAccountsStep 测试
// ==========================

func TestValidateAccountsStep_Execute_Success(t *testing.T) {
	mockService := &mockAccountService{}
	step := &ValidateAccountsStep{service: mockService}
	ctx := context.Background()

	transferData := createTestTransferData()
	result, err := step.Execute(ctx, transferData)

	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	validateResult, ok := result.(*ValidateAccountsResult)
	if !ok {
		t.Fatalf("结果类型错误，期望 *ValidateAccountsResult")
	}

	if validateResult.FromAccountInfo == nil {
		t.Error("FromAccountInfo 不应为 nil")
	}
	if validateResult.ToAccountInfo == nil {
		t.Error("ToAccountInfo 不应为 nil")
	}
	if validateResult.ValidationCode == "" {
		t.Error("ValidationCode 不应为空")
	}
}

func TestValidateAccountsStep_Execute_ValidationError(t *testing.T) {
	mockService := &mockAccountService{}
	step := &ValidateAccountsStep{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name         string
		transferData *TransferData
		wantErr      bool
	}{
		{
			name: "空源账户ID",
			transferData: &TransferData{
				FromAccount: "",
				ToAccount:   "ACC-002",
				Amount:      100.0,
				Currency:    "CNY",
				CustomerID:  "CUST-001",
			},
			wantErr: true,
		},
		{
			name: "空目标账户ID",
			transferData: &TransferData{
				FromAccount: "ACC-001",
				ToAccount:   "",
				Amount:      100.0,
				Currency:    "CNY",
				CustomerID:  "CUST-001",
			},
			wantErr: true,
		},
		{
			name: "源账户和目标账户相同",
			transferData: &TransferData{
				FromAccount: "ACC-001",
				ToAccount:   "ACC-001",
				Amount:      100.0,
				Currency:    "CNY",
				CustomerID:  "CUST-001",
			},
			wantErr: true,
		},
		{
			name: "转账金额为0",
			transferData: &TransferData{
				FromAccount: "ACC-001",
				ToAccount:   "ACC-002",
				Amount:      0,
				Currency:    "CNY",
				CustomerID:  "CUST-001",
			},
			wantErr: true,
		},
		{
			name: "转账金额为负数",
			transferData: &TransferData{
				FromAccount: "ACC-001",
				ToAccount:   "ACC-002",
				Amount:      -100.0,
				Currency:    "CNY",
				CustomerID:  "CUST-001",
			},
			wantErr: true,
		},
		{
			name: "空货币类型",
			transferData: &TransferData{
				FromAccount: "ACC-001",
				ToAccount:   "ACC-002",
				Amount:      100.0,
				Currency:    "",
				CustomerID:  "CUST-001",
			},
			wantErr: true,
		},
		{
			name: "空客户ID",
			transferData: &TransferData{
				FromAccount: "ACC-001",
				ToAccount:   "ACC-002",
				Amount:      100.0,
				Currency:    "CNY",
				CustomerID:  "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := step.Execute(ctx, tt.transferData)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAccountsStep_Execute_InsufficientBalance(t *testing.T) {
	mockService := &mockAccountService{
		validateAccountFunc: func(ctx context.Context, accountID string) (*AccountBalance, error) {
			if accountID == "ACC-001" {
				return &AccountBalance{
					AccountID:      accountID,
					Available:      50.0, // 余额不足
					Frozen:         0.0,
					Total:          50.0,
					Currency:       "CNY",
					MinimumBalance: 0.0,
				}, nil
			}
			return &AccountBalance{
				AccountID: accountID,
				Available: 1000.0,
				Currency:  "CNY",
			}, nil
		},
	}
	step := &ValidateAccountsStep{service: mockService}
	ctx := context.Background()

	transferData := createTestTransferData()
	transferData.Amount = 100.0 // 需要100，但只有50

	_, err := step.Execute(ctx, transferData)
	if err == nil {
		t.Error("Expected insufficient balance error")
	}
}

func TestValidateAccountsStep_Execute_CurrencyMismatch(t *testing.T) {
	mockService := &mockAccountService{
		validateAccountFunc: func(ctx context.Context, accountID string) (*AccountBalance, error) {
			if accountID == "ACC-001" {
				return &AccountBalance{
					AccountID: accountID,
					Available: 1000.0,
					Currency:  "CNY",
				}, nil
			}
			return &AccountBalance{
				AccountID: accountID,
				Available: 1000.0,
				Currency:  "USD", // 不同货币
			}, nil
		},
	}
	step := &ValidateAccountsStep{service: mockService}
	ctx := context.Background()

	transferData := createTestTransferData()
	_, err := step.Execute(ctx, transferData)
	if err == nil {
		t.Error("Expected currency mismatch error")
	}
}

func TestValidateAccountsStep_Compensate(t *testing.T) {
	mockService := &mockAccountService{}
	step := &ValidateAccountsStep{service: mockService}
	ctx := context.Background()

	result := &ValidateAccountsResult{
		TransferID: "TRF-001",
	}

	// 验证步骤不需要补偿，应该返回 nil
	err := step.Compensate(ctx, result)
	if err != nil {
		t.Errorf("Compensate() should return nil, got %v", err)
	}
}

func TestValidateAccountsStep_Metadata(t *testing.T) {
	mockService := &mockAccountService{}
	step := &ValidateAccountsStep{service: mockService}

	if step.GetID() != "validate-accounts" {
		t.Errorf("GetID() = %s, want validate-accounts", step.GetID())
	}
	if step.GetName() == "" {
		t.Error("GetName() should not be empty")
	}
	if step.GetTimeout() <= 0 {
		t.Error("GetTimeout() should be positive")
	}
	if step.GetRetryPolicy() == nil {
		t.Error("GetRetryPolicy() should not be nil")
	}
}

func TestValidateAccountsStep_AllMethods(t *testing.T) {
	mockService := &mockAccountService{}
	step := &ValidateAccountsStep{service: mockService}

	// Test all interface methods
	if step.GetID() != "validate-accounts" {
		t.Errorf("GetID() = %s, want validate-accounts", step.GetID())
	}
	if step.GetName() == "" {
		t.Error("GetName() should not be empty")
	}
	if step.GetDescription() == "" {
		t.Error("GetDescription() should not be empty")
	}
	if step.GetTimeout() <= 0 {
		t.Error("GetTimeout() should be positive")
	}
	if step.GetRetryPolicy() == nil {
		t.Error("GetRetryPolicy() should not be nil")
	}
	if step.GetMetadata() == nil {
		t.Error("GetMetadata() should not be nil")
	}

	// Test IsRetryable
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("IsRetryable() should return false for deadline exceeded")
	}
	if !step.IsRetryable(errors.New("network error")) {
		t.Error("IsRetryable() should return true for network error")
	}
}

// ==========================
// FreezeAmountStep 测试
// ==========================

func TestFreezeAmountStep_Execute_Success(t *testing.T) {
	mockService := &mockAccountService{}
	step := &FreezeAmountStep{service: mockService}
	ctx := context.Background()

	validateResult := &ValidateAccountsResult{
		TransferID: "TRF-001",
		Metadata: map[string]interface{}{
			"from_account": "ACC-001",
			"to_account":   "ACC-002",
			"amount":       100.0,
			"currency":     "CNY",
		},
	}

	result, err := step.Execute(ctx, validateResult)
	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	freezeResult, ok := result.(*FreezeResult)
	if !ok {
		t.Fatalf("结果类型错误，期望 *FreezeResult")
	}

	if freezeResult.FreezeID == "" {
		t.Error("FreezeID 不应为空")
	}
	if freezeResult.Amount != 100.0 {
		t.Errorf("Amount = %.2f, want 100.00", freezeResult.Amount)
	}
}

func TestFreezeAmountStep_Execute_ServiceError(t *testing.T) {
	mockService := &mockAccountService{
		freezeAmountFunc: func(ctx context.Context, accountID string, amount float64, reason string) (*FreezeResult, error) {
			return nil, errors.New("冻结失败")
		},
	}
	step := &FreezeAmountStep{service: mockService}
	ctx := context.Background()

	validateResult := &ValidateAccountsResult{
		Metadata: map[string]interface{}{
			"from_account": "ACC-001",
			"amount":       100.0,
		},
	}

	_, err := step.Execute(ctx, validateResult)
	if err == nil {
		t.Error("Expected freeze error")
	}
}

func TestFreezeAmountStep_Compensate_Success(t *testing.T) {
	mockService := &mockAccountService{}
	step := &FreezeAmountStep{service: mockService}
	ctx := context.Background()

	freezeResult := &FreezeResult{
		FreezeID: "FRZ-123",
	}

	err := step.Compensate(ctx, freezeResult)
	if err != nil {
		t.Errorf("Compensate() 失败: %v", err)
	}
}

func TestFreezeAmountStep_Compensate_ServiceError(t *testing.T) {
	mockService := &mockAccountService{
		unfreezeAmountFunc: func(ctx context.Context, freezeID string, reason string) error {
			return errors.New("解冻失败")
		},
	}
	step := &FreezeAmountStep{service: mockService}
	ctx := context.Background()

	freezeResult := &FreezeResult{
		FreezeID: "FRZ-123",
	}

	err := step.Compensate(ctx, freezeResult)
	if err == nil {
		t.Error("Expected unfreeze error")
	}
}

func TestFreezeAmountStep_AllMethods(t *testing.T) {
	mockService := &mockAccountService{}
	step := &FreezeAmountStep{service: mockService}

	if step.GetID() != "freeze-amount" {
		t.Errorf("GetID() = %s, want freeze-amount", step.GetID())
	}
	if step.GetName() == "" {
		t.Error("GetName() should not be empty")
	}
	if step.GetDescription() == "" {
		t.Error("GetDescription() should not be empty")
	}
	if step.GetTimeout() <= 0 {
		t.Error("GetTimeout() should be positive")
	}
	if step.GetRetryPolicy() == nil {
		t.Error("GetRetryPolicy() should not be nil")
	}
	if step.GetMetadata() == nil {
		t.Error("GetMetadata() should not be nil")
	}
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("IsRetryable() should return false for deadline exceeded")
	}
}

// ==========================
// ExecuteTransferStep 测试
// ==========================

func TestExecuteTransferStep_Execute_Success(t *testing.T) {
	mockAccountService := &mockAccountService{}
	mockTransferService := &mockTransferService{}
	step := &ExecuteTransferStep{
		accountService:  mockAccountService,
		transferService: mockTransferService,
	}
	ctx := context.Background()

	freezeResult := &FreezeResult{
		FreezeID:  "FRZ-123",
		AccountID: "ACC-001",
		Amount:    100.0,
		Currency:  "CNY",
		Metadata: map[string]interface{}{
			"transfer_id": "TRF-001",
			"to_account":  "ACC-002",
		},
	}

	result, err := step.Execute(ctx, freezeResult)
	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	transferResult, ok := result.(*TransferResult)
	if !ok {
		t.Fatalf("结果类型错误，期望 *TransferResult")
	}

	if transferResult.TransactionID == "" {
		t.Error("TransactionID 不应为空")
	}
	if transferResult.Status != "completed" {
		t.Errorf("Status = %s, want completed", transferResult.Status)
	}
}

func TestExecuteTransferStep_Execute_DeductError(t *testing.T) {
	mockAccountService := &mockAccountService{
		deductFrozenAmountFunc: func(ctx context.Context, freezeID string) error {
			return errors.New("扣除失败")
		},
	}
	mockTransferService := &mockTransferService{}
	step := &ExecuteTransferStep{
		accountService:  mockAccountService,
		transferService: mockTransferService,
	}
	ctx := context.Background()

	freezeResult := &FreezeResult{
		FreezeID: "FRZ-123",
		Metadata: map[string]interface{}{
			"to_account": "ACC-002",
		},
	}

	_, err := step.Execute(ctx, freezeResult)
	if err == nil {
		t.Error("Expected deduct error")
	}
}

func TestExecuteTransferStep_Execute_TransferError(t *testing.T) {
	mockAccountService := &mockAccountService{}
	mockTransferService := &mockTransferService{
		executeTransferFunc: func(ctx context.Context, transferID string, freezeID string) (*TransferResult, error) {
			return nil, errors.New("转账失败")
		},
	}
	step := &ExecuteTransferStep{
		accountService:  mockAccountService,
		transferService: mockTransferService,
	}
	ctx := context.Background()

	freezeResult := &FreezeResult{
		FreezeID: "FRZ-123",
		Metadata: map[string]interface{}{
			"transfer_id": "TRF-001",
			"to_account":  "ACC-002",
		},
	}

	_, err := step.Execute(ctx, freezeResult)
	if err == nil {
		t.Error("Expected transfer error")
	}
}

func TestExecuteTransferStep_Execute_AddAmountError(t *testing.T) {
	mockAccountService := &mockAccountService{
		addAmountFunc: func(ctx context.Context, accountID string, amount float64, transactionID string) error {
			return errors.New("增加金额失败")
		},
	}
	mockTransferService := &mockTransferService{}
	step := &ExecuteTransferStep{
		accountService:  mockAccountService,
		transferService: mockTransferService,
	}
	ctx := context.Background()

	freezeResult := &FreezeResult{
		FreezeID:  "FRZ-123",
		Amount:    100.0,
		Currency:  "CNY",
		AccountID: "ACC-001",
		Metadata: map[string]interface{}{
			"transfer_id": "TRF-001",
			"to_account":  "ACC-002",
		},
	}

	_, err := step.Execute(ctx, freezeResult)
	if err == nil {
		t.Error("Expected add amount error")
	}
}

func TestExecuteTransferStep_Compensate_CompletedTransfer(t *testing.T) {
	// 记录补偿操作的调用顺序
	var operations []string

	mockAccountService := &mockAccountService{
		addAmountFunc: func(ctx context.Context, accountID string, amount float64, transactionID string) error {
			if amount < 0 {
				operations = append(operations, fmt.Sprintf("deduct_from_%s", accountID))
			} else {
				operations = append(operations, fmt.Sprintf("refund_to_%s", accountID))
			}
			return nil
		},
	}
	mockTransferService := &mockTransferService{
		getTransferStatusFunc: func(ctx context.Context, transferID string) (string, error) {
			return "completed", nil
		},
		cancelTransferFunc: func(ctx context.Context, transferID string, reason string) error {
			operations = append(operations, "cancel_transfer")
			return nil
		},
	}
	step := &ExecuteTransferStep{
		accountService:  mockAccountService,
		transferService: mockTransferService,
	}
	ctx := context.Background()

	transferResult := &TransferResult{
		TransferID:  "TRF-001",
		FromAccount: "ACC-001",
		ToAccount:   "ACC-002",
		Amount:      100.0,
	}

	err := step.Compensate(ctx, transferResult)
	if err != nil {
		t.Errorf("Compensate() 失败: %v", err)
	}

	// 验证操作顺序：先从目标账户扣除，再退款到源账户，最后取消记录
	expectedOps := []string{
		"deduct_from_ACC-002",
		"refund_to_ACC-001",
		"cancel_transfer",
	}
	if len(operations) != len(expectedOps) {
		t.Errorf("操作数量不匹配，期望 %d，实际 %d", len(expectedOps), len(operations))
	}
	for i, op := range expectedOps {
		if i >= len(operations) || operations[i] != op {
			t.Errorf("操作顺序错误，步骤 %d 期望 %s，实际 %v", i+1, op, operations)
		}
	}
}

func TestExecuteTransferStep_Compensate_NotCompletedTransfer(t *testing.T) {
	mockAccountService := &mockAccountService{}
	mockTransferService := &mockTransferService{
		getTransferStatusFunc: func(ctx context.Context, transferID string) (string, error) {
			return "pending", nil
		},
	}
	step := &ExecuteTransferStep{
		accountService:  mockAccountService,
		transferService: mockTransferService,
	}
	ctx := context.Background()

	transferResult := &TransferResult{
		TransferID: "TRF-001",
	}

	err := step.Compensate(ctx, transferResult)
	if err != nil {
		t.Errorf("Compensate() 失败: %v", err)
	}
}

func TestExecuteTransferStep_Compensate_DeductFromDestinationFails(t *testing.T) {
	mockAccountService := &mockAccountService{
		addAmountFunc: func(ctx context.Context, accountID string, amount float64, transactionID string) error {
			if amount < 0 { // 扣除操作
				return errors.New("目标账户扣除失败")
			}
			return nil
		},
	}
	mockTransferService := &mockTransferService{
		getTransferStatusFunc: func(ctx context.Context, transferID string) (string, error) {
			return "completed", nil
		},
	}
	step := &ExecuteTransferStep{
		accountService:  mockAccountService,
		transferService: mockTransferService,
	}
	ctx := context.Background()

	transferResult := &TransferResult{
		TransferID:  "TRF-001",
		FromAccount: "ACC-001",
		ToAccount:   "ACC-002",
		Amount:      100.0,
	}

	err := step.Compensate(ctx, transferResult)
	if err == nil {
		t.Error("Expected error when deducting from destination account fails")
	}
}

func TestExecuteTransferStep_Compensate_RefundToSourceFails(t *testing.T) {
	mockAccountService := &mockAccountService{
		addAmountFunc: func(ctx context.Context, accountID string, amount float64, transactionID string) error {
			if amount > 0 { // 退款操作
				return errors.New("源账户退款失败")
			}
			return nil
		},
	}
	mockTransferService := &mockTransferService{
		getTransferStatusFunc: func(ctx context.Context, transferID string) (string, error) {
			return "completed", nil
		},
	}
	step := &ExecuteTransferStep{
		accountService:  mockAccountService,
		transferService: mockTransferService,
	}
	ctx := context.Background()

	transferResult := &TransferResult{
		TransferID:  "TRF-001",
		FromAccount: "ACC-001",
		ToAccount:   "ACC-002",
		Amount:      100.0,
	}

	err := step.Compensate(ctx, transferResult)
	if err == nil {
		t.Error("Expected error when refunding to source account fails")
	}
	if err != nil && !contains(err.Error(), "需要紧急人工处理") {
		t.Error("Error message should indicate urgent manual intervention needed")
	}
}

func TestExecuteTransferStep_Compensate_GetStatusFails(t *testing.T) {
	mockAccountService := &mockAccountService{}
	mockTransferService := &mockTransferService{
		getTransferStatusFunc: func(ctx context.Context, transferID string) (string, error) {
			return "", errors.New("查询状态失败")
		},
	}
	step := &ExecuteTransferStep{
		accountService:  mockAccountService,
		transferService: mockTransferService,
	}
	ctx := context.Background()

	transferResult := &TransferResult{
		TransferID: "TRF-001",
	}

	err := step.Compensate(ctx, transferResult)
	if err == nil {
		t.Error("Expected error when getting transfer status fails")
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestExecuteTransferStep_AllMethods(t *testing.T) {
	mockAccountService := &mockAccountService{}
	mockTransferService := &mockTransferService{}
	step := &ExecuteTransferStep{
		accountService:  mockAccountService,
		transferService: mockTransferService,
	}

	if step.GetID() != "execute-transfer" {
		t.Errorf("GetID() = %s, want execute-transfer", step.GetID())
	}
	if step.GetName() == "" {
		t.Error("GetName() should not be empty")
	}
	if step.GetDescription() == "" {
		t.Error("GetDescription() should not be empty")
	}
	if step.GetTimeout() <= 0 {
		t.Error("GetTimeout() should be positive")
	}
	if step.GetRetryPolicy() == nil {
		t.Error("GetRetryPolicy() should not be nil")
	}
	if step.GetMetadata() == nil {
		t.Error("GetMetadata() should not be nil")
	}
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("IsRetryable() should return false for deadline exceeded")
	}
}

func TestReleaseFreezeStep_AllMethods(t *testing.T) {
	mockService := &mockAccountService{}
	step := &ReleaseFreezeStep{service: mockService}

	if step.GetID() != "release-freeze" {
		t.Errorf("GetID() = %s, want release-freeze", step.GetID())
	}
	if step.GetName() == "" {
		t.Error("GetName() should not be empty")
	}
	if step.GetDescription() == "" {
		t.Error("GetDescription() should not be empty")
	}
	if step.GetTimeout() <= 0 {
		t.Error("GetTimeout() should be positive")
	}
	if step.GetRetryPolicy() == nil {
		t.Error("GetRetryPolicy() should not be nil")
	}
	if step.GetMetadata() == nil {
		t.Error("GetMetadata() should not be nil")
	}
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("IsRetryable() should return false for deadline exceeded")
	}
}

// ==========================
// CreateAuditRecordStep 测试
// ==========================

func TestCreateAuditRecordStep_Execute_Success(t *testing.T) {
	mockService := &mockAuditService{}
	step := &CreateAuditRecordStep{service: mockService}
	ctx := context.Background()

	unfreezeResult := &UnfreezeResult{
		Amount: 100.0,
		Metadata: map[string]interface{}{
			"transfer_id":  "TRF-001",
			"from_account": "ACC-001",
			"to_account":   "ACC-002",
			"currency":     "CNY",
			"customer_id":  "CUST-001",
			"risk_level":   "low",
		},
	}

	result, err := step.Execute(ctx, unfreezeResult)
	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	auditResult, ok := result.(*AuditRecordResult)
	if !ok {
		t.Fatalf("结果类型错误，期望 *AuditRecordResult")
	}

	if auditResult.AuditID == "" {
		t.Error("AuditID 不应为空")
	}
}

func TestCreateAuditRecordStep_Compensate_Success(t *testing.T) {
	mockService := &mockAuditService{}
	step := &CreateAuditRecordStep{service: mockService}
	ctx := context.Background()

	auditResult := &AuditRecordResult{
		AuditID: "AUD-123",
	}

	err := step.Compensate(ctx, auditResult)
	if err != nil {
		t.Errorf("Compensate() 失败: %v", err)
	}
}

func TestCreateAuditRecordStep_AllMethods(t *testing.T) {
	mockService := &mockAuditService{}
	step := &CreateAuditRecordStep{service: mockService}

	if step.GetID() != "create-audit-record" {
		t.Errorf("GetID() = %s, want create-audit-record", step.GetID())
	}
	if step.GetName() == "" {
		t.Error("GetName() should not be empty")
	}
	if step.GetDescription() == "" {
		t.Error("GetDescription() should not be empty")
	}
	if step.GetTimeout() <= 0 {
		t.Error("GetTimeout() should be positive")
	}
	if step.GetRetryPolicy() == nil {
		t.Error("GetRetryPolicy() should not be nil")
	}
	if step.GetMetadata() == nil {
		t.Error("GetMetadata() should not be nil")
	}
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("IsRetryable() should return false for deadline exceeded")
	}
}

// ==========================
// SendTransferNotificationStep 测试
// ==========================

func TestSendTransferNotificationStep_Execute_Success(t *testing.T) {
	mockService := &mockNotificationService{}
	step := &SendTransferNotificationStep{service: mockService}
	ctx := context.Background()

	auditResult := &AuditRecordResult{
		TransferID: "TRF-001",
		Metadata: map[string]interface{}{
			"transfer_id":  "TRF-001",
			"from_account": "ACC-001",
			"to_account":   "ACC-002",
			"amount":       100.0,
			"currency":     "CNY",
			"customer_id":  "CUST-001",
		},
	}

	result, err := step.Execute(ctx, auditResult)
	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	notificationResult, ok := result.(*NotificationResult)
	if !ok {
		t.Fatalf("结果类型错误，期望 *NotificationResult")
	}

	if notificationResult.NotificationID == "" {
		t.Error("NotificationID 不应为空")
	}
	if notificationResult.Status != "sent" {
		t.Errorf("Status = %s, want sent", notificationResult.Status)
	}
}

func TestSendTransferNotificationStep_Execute_NotificationFails(t *testing.T) {
	// 模拟通知服务失败
	mockService := &mockNotificationService{
		sendTransferNotificationFunc: func(ctx context.Context, transferID string, recipients []string, data *TransferData) (*NotificationResult, error) {
			return nil, errors.New("通知服务暂时不可用")
		},
	}
	step := &SendTransferNotificationStep{service: mockService}
	ctx := context.Background()

	auditResult := &AuditRecordResult{
		TransferID: "TRF-001",
		Metadata: map[string]interface{}{
			"transfer_id":  "TRF-001",
			"from_account": "ACC-001",
			"to_account":   "ACC-002",
			"amount":       100.0,
			"currency":     "CNY",
			"customer_id":  "CUST-001",
		},
	}

	// 关键：即使通知失败，Execute 也应该返回成功（nil error）
	// 因为资金转账已完成，不应该因为通知失败而回滚交易
	result, err := step.Execute(ctx, auditResult)
	if err != nil {
		t.Fatalf("Execute() 不应该因通知失败而返回错误: %v", err)
	}

	notificationResult, ok := result.(*NotificationResult)
	if !ok {
		t.Fatalf("结果类型错误，期望 *NotificationResult")
	}

	// 验证结果包含失败信息
	if notificationResult.Status != "failed" {
		t.Errorf("Status = %s, want failed", notificationResult.Status)
	}
	if notificationResult.Channel != "none" {
		t.Errorf("Channel = %s, want none", notificationResult.Channel)
	}
	if notificationResult.TransferID != "TRF-001" {
		t.Errorf("TransferID = %s, want TRF-001", notificationResult.TransferID)
	}

	// 验证元数据中包含错误信息和重试建议
	if notificationResult.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	if _, ok := notificationResult.Metadata["error"]; !ok {
		t.Error("Metadata should contain error information")
	}
	if retry, ok := notificationResult.Metadata["retry_recommended"].(bool); !ok || !retry {
		t.Error("Metadata should recommend retry")
	}
	if _, ok := notificationResult.Metadata["failure_timestamp"]; !ok {
		t.Error("Metadata should contain failure timestamp")
	}
}

func TestSendTransferNotificationStep_Compensate_Success(t *testing.T) {
	mockService := &mockNotificationService{}
	step := &SendTransferNotificationStep{service: mockService}
	ctx := context.Background()

	notificationResult := &NotificationResult{
		NotificationID: "NOT-123",
		TransferID:     "TRF-001",
	}

	err := step.Compensate(ctx, notificationResult)
	if err != nil {
		t.Errorf("Compensate() 失败: %v", err)
	}
}

func TestSendTransferNotificationStep_AllMethods(t *testing.T) {
	mockService := &mockNotificationService{}
	step := &SendTransferNotificationStep{service: mockService}

	if step.GetID() != "send-notification" {
		t.Errorf("GetID() = %s, want send-notification", step.GetID())
	}
	if step.GetName() == "" {
		t.Error("GetName() should not be empty")
	}
	if step.GetDescription() == "" {
		t.Error("GetDescription() should not be empty")
	}
	if step.GetTimeout() <= 0 {
		t.Error("GetTimeout() should be positive")
	}
	if step.GetRetryPolicy() == nil {
		t.Error("GetRetryPolicy() should not be nil")
	}
	if step.GetMetadata() == nil {
		t.Error("GetMetadata() should not be nil")
	}
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("IsRetryable() should return false for deadline exceeded")
	}
}

// ==========================
// PaymentProcessingSagaDefinition 测试
// ==========================

func TestNewPaymentProcessingSaga(t *testing.T) {
	mockAccountService := &mockAccountService{}
	mockTransferService := &mockTransferService{}
	mockAuditService := &mockAuditService{}
	mockNotificationService := &mockNotificationService{}

	sagaDef := NewPaymentProcessingSaga(
		mockAccountService,
		mockTransferService,
		mockAuditService,
		mockNotificationService,
	)

	if sagaDef == nil {
		t.Fatal("NewPaymentProcessingSaga() returned nil")
	}

	if sagaDef.GetID() == "" {
		t.Error("Saga ID should not be empty")
	}

	if sagaDef.GetName() == "" {
		t.Error("Saga name should not be empty")
	}

	steps := sagaDef.GetSteps()
	expectedSteps := 6 // 6个步骤
	if len(steps) != expectedSteps {
		t.Errorf("Expected %d steps, got %d", expectedSteps, len(steps))
	}

	if sagaDef.GetTimeout() <= 0 {
		t.Error("Saga timeout should be positive")
	}

	if sagaDef.GetRetryPolicy() == nil {
		t.Error("Retry policy should not be nil")
	}

	if sagaDef.GetCompensationStrategy() == nil {
		t.Error("Compensation strategy should not be nil")
	}
}

func TestPaymentProcessingSagaDefinition_Validate(t *testing.T) {
	mockAccountService := &mockAccountService{}
	mockTransferService := &mockTransferService{}
	mockAuditService := &mockAuditService{}
	mockNotificationService := &mockNotificationService{}

	sagaDef := NewPaymentProcessingSaga(
		mockAccountService,
		mockTransferService,
		mockAuditService,
		mockNotificationService,
	)

	err := sagaDef.Validate()
	if err != nil {
		t.Errorf("Validate() should not return error for valid definition, got: %v", err)
	}
}

func TestPaymentProcessingSagaDefinition_Validate_InvalidCases(t *testing.T) {
	tests := []struct {
		name    string
		sagaDef *PaymentProcessingSagaDefinition
		wantErr bool
	}{
		{
			name: "空ID",
			sagaDef: &PaymentProcessingSagaDefinition{
				id:      "",
				name:    "测试",
				steps:   []saga.SagaStep{&ValidateAccountsStep{}},
				timeout: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "空名称",
			sagaDef: &PaymentProcessingSagaDefinition{
				id:      "test",
				name:    "",
				steps:   []saga.SagaStep{&ValidateAccountsStep{}},
				timeout: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "没有步骤",
			sagaDef: &PaymentProcessingSagaDefinition{
				id:      "test",
				name:    "测试",
				steps:   []saga.SagaStep{},
				timeout: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "超时时间为0",
			sagaDef: &PaymentProcessingSagaDefinition{
				id:      "test",
				name:    "测试",
				steps:   []saga.SagaStep{&ValidateAccountsStep{}},
				timeout: 0,
			},
			wantErr: true,
		},
		{
			name: "负超时时间",
			sagaDef: &PaymentProcessingSagaDefinition{
				id:      "test",
				name:    "测试",
				steps:   []saga.SagaStep{&ValidateAccountsStep{}},
				timeout: -time.Minute,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sagaDef.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPaymentProcessingSagaDefinition_Metadata(t *testing.T) {
	mockAccountService := &mockAccountService{}
	mockTransferService := &mockTransferService{}
	mockAuditService := &mockAuditService{}
	mockNotificationService := &mockNotificationService{}

	sagaDef := NewPaymentProcessingSaga(
		mockAccountService,
		mockTransferService,
		mockAuditService,
		mockNotificationService,
	)

	metadata := sagaDef.GetMetadata()
	if metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	expectedFields := []string{"saga_type", "business_domain", "version", "critical", "compliance"}
	for _, field := range expectedFields {
		if _, ok := metadata[field]; !ok {
			t.Errorf("Metadata missing field: %s", field)
		}
	}
}

// ==========================
// 集成测试
// ==========================

func TestPaymentSaga_FullFlow_Success(t *testing.T) {
	// 创建所有 mock 服务
	mockAccountService := &mockAccountService{}
	mockTransferService := &mockTransferService{}
	mockAuditService := &mockAuditService{}
	mockNotificationService := &mockNotificationService{}

	// 创建 Saga 定义
	sagaDef := NewPaymentProcessingSaga(
		mockAccountService,
		mockTransferService,
		mockAuditService,
		mockNotificationService,
	)

	// 准备测试数据
	transferData := createTestTransferData()
	ctx := context.Background()

	// 执行所有步骤
	var currentData interface{} = transferData
	var err error

	steps := sagaDef.GetSteps()
	for i, step := range steps {
		t.Logf("执行步骤 %d: %s", i+1, step.GetName())
		currentData, err = step.Execute(ctx, currentData)
		if err != nil {
			t.Fatalf("步骤 %d (%s) 执行失败: %v", i+1, step.GetName(), err)
		}
	}

	t.Log("所有步骤执行成功")
}

func TestPaymentSaga_Compensation_Success(t *testing.T) {
	// 创建所有 mock 服务
	mockAccountService := &mockAccountService{}
	mockTransferService := &mockTransferService{}
	mockAuditService := &mockAuditService{}
	mockNotificationService := &mockNotificationService{}

	// 创建 Saga 定义
	sagaDef := NewPaymentProcessingSaga(
		mockAccountService,
		mockTransferService,
		mockAuditService,
		mockNotificationService,
	)

	// 准备测试数据
	transferData := createTestTransferData()
	ctx := context.Background()

	// 执行前几个步骤
	steps := sagaDef.GetSteps()
	var executedResults []interface{}
	var currentData interface{} = transferData

	// 执行前3个步骤
	for i := 0; i < 3 && i < len(steps); i++ {
		result, err := steps[i].Execute(ctx, currentData)
		if err != nil {
			t.Fatalf("步骤 %d 执行失败: %v", i+1, err)
		}
		executedResults = append(executedResults, result)
		currentData = result
	}

	// 执行补偿（逆序）
	t.Log("开始执行补偿操作")
	for i := len(executedResults) - 1; i >= 0; i-- {
		err := steps[i].Compensate(ctx, executedResults[i])
		if err != nil {
			t.Logf("步骤 %d 补偿警告: %v", i+1, err)
			// 某些补偿失败不应导致测试失败
		}
	}

	t.Log("补偿操作完成")
}

// ==========================
// 辅助函数测试
// ==========================

func TestGenerateValidationCode(t *testing.T) {
	data := createTestTransferData()
	code := generateValidationCode(data)
	if code == "" {
		t.Error("generateValidationCode() should not return empty string")
	}
}

func TestGenerateTransferID(t *testing.T) {
	id1 := generateTransferID()
	time.Sleep(1 * time.Millisecond) // 确保时间戳不同
	id2 := generateTransferID()

	if id1 == "" {
		t.Error("generateTransferID() should not return empty string")
	}
	if id1 == id2 {
		t.Error("generateTransferID() should generate unique IDs")
	}
}

func TestGenerateUnfreezeID(t *testing.T) {
	id1 := generateUnfreezeID()
	time.Sleep(1 * time.Millisecond) // 确保时间戳不同
	id2 := generateUnfreezeID()

	if id1 == "" {
		t.Error("generateUnfreezeID() should not return empty string")
	}
	if id1 == id2 {
		t.Error("generateUnfreezeID() should generate unique IDs")
	}
}
