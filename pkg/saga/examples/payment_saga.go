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

// Package examples provides example implementations of Saga patterns for common business scenarios.
// This file demonstrates a payment processing Saga for cross-account fund transfers,
// showcasing how to handle financial distributed transactions with strict consistency guarantees.
package examples

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// ==========================
// 数据结构定义
// ==========================

// TransferData 包含资金转账所需的所有输入数据
type TransferData struct {
	// 转账基本信息
	TransferID  string  // 转账ID（如果已生成）
	FromAccount string  // 源账户ID
	ToAccount   string  // 目标账户ID
	Amount      float64 // 转账金额
	Currency    string  // 货币类型（如 USD、CNY 等）

	// 转账说明
	Purpose     string // 转账用途
	Description string // 转账描述

	// 客户信息
	CustomerID   string // 客户ID（发起转账的用户）
	CustomerName string // 客户姓名

	// 风控信息
	RiskLevel     string   // 风险等级（low、medium、high）
	RiskFactors   []string // 风险因素列表
	RequireReview bool     // 是否需要人工审核

	// 元数据
	Metadata map[string]interface{} // 额外元数据
}

// AccountBalance 表示账户余额信息
type AccountBalance struct {
	AccountID      string    // 账户ID
	Available      float64   // 可用余额
	Frozen         float64   // 冻结金额
	Total          float64   // 总余额
	Currency       string    // 货币类型
	LastUpdated    time.Time // 最后更新时间
	MinimumBalance float64   // 最低余额要求
	OverdraftLimit float64   // 透支额度
}

// ValidateAccountsResult 存储账户验证步骤的结果
type ValidateAccountsResult struct {
	TransferID      string                 // 转账ID
	FromAccountInfo *AccountBalance        // 源账户信息
	ToAccountInfo   *AccountBalance        // 目标账户信息
	ValidationTime  time.Time              // 验证时间
	ValidationCode  string                 // 验证码（用于后续步骤验证）
	Metadata        map[string]interface{} // 元数据
}

// FreezeResult 存储资金冻结步骤的结果
type FreezeResult struct {
	FreezeID   string                 // 冻结ID
	AccountID  string                 // 账户ID
	Amount     float64                // 冻结金额
	Currency   string                 // 货币类型
	FrozenAt   time.Time              // 冻结时间
	ExpiresAt  time.Time              // 过期时间（自动解冻）
	FreezeType string                 // 冻结类型（transfer、payment等）
	Metadata   map[string]interface{} // 元数据
}

// TransferResult 存储转账执行步骤的结果
type TransferResult struct {
	TransactionID   string                 // 交易ID
	TransferID      string                 // 转账ID
	FromAccount     string                 // 源账户ID
	ToAccount       string                 // 目标账户ID
	Amount          float64                // 转账金额
	Currency        string                 // 货币类型
	Status          string                 // 转账状态（pending、completed、failed）
	CompletedAt     time.Time              // 完成时间
	TransactionHash string                 // 交易哈希（用于防篡改）
	Metadata        map[string]interface{} // 元数据
}

// UnfreezeResult 存储解冻步骤的结果
type UnfreezeResult struct {
	UnfreezeID string                 // 解冻ID
	FreezeID   string                 // 原冻结ID
	AccountID  string                 // 账户ID
	Amount     float64                // 解冻金额
	UnfrozenAt time.Time              // 解冻时间
	Reason     string                 // 解冻原因
	Metadata   map[string]interface{} // 元数据
}

// AuditRecordResult 存储审计记录步骤的结果
type AuditRecordResult struct {
	AuditID        string                 // 审计ID
	TransferID     string                 // 转账ID
	RecordType     string                 // 记录类型
	CreatedAt      time.Time              // 创建时间
	Status         string                 // 审计状态
	ReviewRequired bool                   // 是否需要审核
	Metadata       map[string]interface{} // 元数据
}

// NotificationResult 存储通知步骤的结果
type NotificationResult struct {
	NotificationID string                 // 通知ID
	TransferID     string                 // 转账ID
	Recipients     []string               // 接收者列表
	Channel        string                 // 通知渠道（email、sms、push等）
	SentAt         time.Time              // 发送时间
	Status         string                 // 发送状态
	Metadata       map[string]interface{} // 元数据
}

// ==========================
// 服务接口定义（用于模拟外部服务）
// ==========================

// AccountServiceClient 账户服务客户端接口
type AccountServiceClient interface {
	// ValidateAccount 验证账户有效性和状态
	ValidateAccount(ctx context.Context, accountID string) (*AccountBalance, error)
	// GetBalance 获取账户余额
	GetBalance(ctx context.Context, accountID string) (*AccountBalance, error)
	// FreezeAmount 冻结指定金额
	FreezeAmount(ctx context.Context, accountID string, amount float64, reason string) (*FreezeResult, error)
	// UnfreezeAmount 解冻指定金额
	UnfreezeAmount(ctx context.Context, freezeID string, reason string) error
	// DeductFrozenAmount 扣除已冻结的金额（实际转出）
	DeductFrozenAmount(ctx context.Context, freezeID string) error
	// AddAmount 增加账户金额（实际转入）
	AddAmount(ctx context.Context, accountID string, amount float64, transactionID string) error
}

// TransferServiceClient 转账服务客户端接口
type TransferServiceClient interface {
	// CreateTransfer 创建转账记录
	CreateTransfer(ctx context.Context, data *TransferData) (*TransferResult, error)
	// ExecuteTransfer 执行转账
	ExecuteTransfer(ctx context.Context, transferID string, freezeID string) (*TransferResult, error)
	// CancelTransfer 取消转账
	CancelTransfer(ctx context.Context, transferID string, reason string) error
	// GetTransferStatus 获取转账状态
	GetTransferStatus(ctx context.Context, transferID string) (string, error)
}

// AuditServiceClient 审计服务客户端接口
type AuditServiceClient interface {
	// CreateAuditRecord 创建审计记录
	CreateAuditRecord(ctx context.Context, transferID string, data *TransferData, result *TransferResult) (*AuditRecordResult, error)
	// UpdateAuditRecord 更新审计记录
	UpdateAuditRecord(ctx context.Context, auditID string, status string, details map[string]interface{}) error
}

// NotificationServiceClient 通知服务客户端接口
type NotificationServiceClient interface {
	// SendTransferNotification 发送转账通知
	SendTransferNotification(ctx context.Context, transferID string, recipients []string, data *TransferData) (*NotificationResult, error)
	// SendFailureNotification 发送失败通知
	SendFailureNotification(ctx context.Context, transferID string, reason string) error
}

// ==========================
// Saga 步骤实现
// ==========================

// ValidateAccountsStep 验证账户步骤
// 此步骤验证源账户和目标账户的有效性，检查账户状态和余额
type ValidateAccountsStep struct {
	service AccountServiceClient
}

// GetID 返回步骤ID
func (s *ValidateAccountsStep) GetID() string {
	return "validate-accounts"
}

// GetName 返回步骤名称
func (s *ValidateAccountsStep) GetName() string {
	return "验证账户"
}

// GetDescription 返回步骤描述
func (s *ValidateAccountsStep) GetDescription() string {
	return "验证源账户和目标账户的有效性，检查账户状态、余额等信息"
}

// Execute 执行账户验证操作
// 输入：TransferData
// 输出：ValidateAccountsResult
func (s *ValidateAccountsStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	transferData, ok := data.(*TransferData)
	if !ok {
		return nil, errors.New("invalid data type: expected *TransferData")
	}

	// 验证转账数据
	if err := s.validateTransferData(transferData); err != nil {
		return nil, fmt.Errorf("转账数据验证失败: %w", err)
	}

	// 验证源账户
	fromAccountInfo, err := s.service.ValidateAccount(ctx, transferData.FromAccount)
	if err != nil {
		return nil, fmt.Errorf("验证源账户失败: %w", err)
	}

	// 验证目标账户
	toAccountInfo, err := s.service.ValidateAccount(ctx, transferData.ToAccount)
	if err != nil {
		return nil, fmt.Errorf("验证目标账户失败: %w", err)
	}

	// 检查余额是否充足（考虑最低余额要求）
	requiredBalance := transferData.Amount + fromAccountInfo.MinimumBalance
	if fromAccountInfo.Available < requiredBalance {
		return nil, fmt.Errorf("账户余额不足: 可用余额 %.2f, 需要 %.2f (包含最低余额 %.2f)",
			fromAccountInfo.Available, requiredBalance, fromAccountInfo.MinimumBalance)
	}

	// 检查货币类型是否匹配
	if fromAccountInfo.Currency != toAccountInfo.Currency {
		return nil, fmt.Errorf("货币类型不匹配: 源账户 %s, 目标账户 %s",
			fromAccountInfo.Currency, toAccountInfo.Currency)
	}

	// 生成验证码（用于后续步骤验证数据一致性）
	validationCode := generateValidationCode(transferData)

	result := &ValidateAccountsResult{
		TransferID:      transferData.TransferID,
		FromAccountInfo: fromAccountInfo,
		ToAccountInfo:   toAccountInfo,
		ValidationTime:  time.Now(),
		ValidationCode:  validationCode,
		Metadata: map[string]interface{}{
			"from_account": transferData.FromAccount,
			"to_account":   transferData.ToAccount,
			"amount":       transferData.Amount,
			"currency":     transferData.Currency,
			"customer_id":  transferData.CustomerID,
			"risk_level":   transferData.RiskLevel,
		},
	}

	return result, nil
}

// Compensate 补偿操作：账户验证不需要补偿
// 因为这一步骤只是读取操作，没有修改任何状态
func (s *ValidateAccountsStep) Compensate(ctx context.Context, data interface{}) error {
	// 账户验证步骤不需要补偿，因为没有修改任何状态
	return nil
}

// GetTimeout 返回步骤超时时间
func (s *ValidateAccountsStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *ValidateAccountsStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *ValidateAccountsStep) IsRetryable(err error) bool {
	// 网络错误和临时服务不可用错误可重试
	// 账户不存在、余额不足等业务错误不可重试
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// 可以根据错误消息判断是否为业务错误
	return true
}

// GetMetadata 返回步骤元数据
func (s *ValidateAccountsStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "account_validation",
		"service":   "account-service",
		"critical":  true, // 标记为关键步骤
	}
}

// validateTransferData 验证转账数据的有效性
func (s *ValidateAccountsStep) validateTransferData(data *TransferData) error {
	if data.FromAccount == "" {
		return errors.New("源账户ID不能为空")
	}
	if data.ToAccount == "" {
		return errors.New("目标账户ID不能为空")
	}
	if data.FromAccount == data.ToAccount {
		return errors.New("源账户和目标账户不能相同")
	}
	if data.Amount <= 0 {
		return errors.New("转账金额必须大于0")
	}
	if data.Currency == "" {
		return errors.New("货币类型不能为空")
	}
	if data.CustomerID == "" {
		return errors.New("客户ID不能为空")
	}
	return nil
}

// FreezeAmountStep 冻结资金步骤
// 此步骤在源账户中冻结转账金额，防止重复使用
type FreezeAmountStep struct {
	service AccountServiceClient
}

// GetID 返回步骤ID
func (s *FreezeAmountStep) GetID() string {
	return "freeze-amount"
}

// GetName 返回步骤名称
func (s *FreezeAmountStep) GetName() string {
	return "冻结资金"
}

// GetDescription 返回步骤描述
func (s *FreezeAmountStep) GetDescription() string {
	return "在源账户中冻结转账金额，确保资金在转账完成前不会被使用"
}

// Execute 执行资金冻结操作
// 输入：ValidateAccountsResult（从上一步传递）
// 输出：FreezeResult
func (s *FreezeAmountStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	validateResult, ok := data.(*ValidateAccountsResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *ValidateAccountsResult")
	}

	// 从元数据中获取转账信息
	fromAccount, _ := validateResult.Metadata["from_account"].(string)
	amount, _ := validateResult.Metadata["amount"].(float64)

	// 冻结资金
	freezeReason := fmt.Sprintf("资金转账: %s -> %s",
		validateResult.Metadata["from_account"],
		validateResult.Metadata["to_account"])

	freezeResult, err := s.service.FreezeAmount(ctx, fromAccount, amount, freezeReason)
	if err != nil {
		return nil, fmt.Errorf("冻结资金失败: %w", err)
	}

	// 设置自动解冻时间（10分钟后）
	freezeResult.ExpiresAt = time.Now().Add(10 * time.Minute)
	freezeResult.Metadata = validateResult.Metadata

	return freezeResult, nil
}

// Compensate 补偿操作：解冻资金
func (s *FreezeAmountStep) Compensate(ctx context.Context, data interface{}) error {
	freezeResult, ok := data.(*FreezeResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *FreezeResult")
	}

	// 解冻资金
	err := s.service.UnfreezeAmount(ctx, freezeResult.FreezeID, "saga_compensation")
	if err != nil {
		// 资金解冻失败是严重问题，需要记录并可能需要人工介入
		return fmt.Errorf("解冻资金失败，需要人工处理: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *FreezeAmountStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *FreezeAmountStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *FreezeAmountStep) IsRetryable(err error) bool {
	return !errors.Is(err, context.DeadlineExceeded)
}

// GetMetadata 返回步骤元数据
func (s *FreezeAmountStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "fund_freeze",
		"service":   "account-service",
		"critical":  true, // 资金操作是关键步骤
	}
}

// ExecuteTransferStep 执行转账步骤
// 此步骤执行实际的资金转移操作
type ExecuteTransferStep struct {
	accountService  AccountServiceClient
	transferService TransferServiceClient
}

// GetID 返回步骤ID
func (s *ExecuteTransferStep) GetID() string {
	return "execute-transfer"
}

// GetName 返回步骤名称
func (s *ExecuteTransferStep) GetName() string {
	return "执行转账"
}

// GetDescription 返回步骤描述
func (s *ExecuteTransferStep) GetDescription() string {
	return "执行实际的资金转移，从源账户扣除并增加到目标账户"
}

// Execute 执行转账操作
// 输入：FreezeResult（从上一步传递）
// 输出：TransferResult
func (s *ExecuteTransferStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	freezeResult, ok := data.(*FreezeResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *FreezeResult")
	}

	// 从元数据中获取转账信息
	toAccount, _ := freezeResult.Metadata["to_account"].(string)
	amount := freezeResult.Amount

	// 1. 从源账户扣除冻结金额
	err := s.accountService.DeductFrozenAmount(ctx, freezeResult.FreezeID)
	if err != nil {
		return nil, fmt.Errorf("扣除冻结金额失败: %w", err)
	}

	// 2. 获取转账ID
	transferID, _ := freezeResult.Metadata["transfer_id"].(string)
	if transferID == "" {
		transferID = generateTransferID()
	}

	// 3. 执行转账
	transferResult, err := s.transferService.ExecuteTransfer(ctx, transferID, freezeResult.FreezeID)
	if err != nil {
		// 转账失败，需要回退已扣除的金额
		// 这将在补偿阶段处理
		return nil, fmt.Errorf("执行转账失败: %w", err)
	}

	// 4. 向目标账户增加金额
	err = s.accountService.AddAmount(ctx, toAccount, amount, transferResult.TransactionID)
	if err != nil {
		// 目标账户增加金额失败，需要回退源账户扣除
		return nil, fmt.Errorf("目标账户增加金额失败: %w", err)
	}

	// 将元数据传递到下一步
	transferResult.Metadata = freezeResult.Metadata

	return transferResult, nil
}

// Compensate 补偿操作：取消转账并退款
// 此补偿操作至关重要，必须确保资金完全回退以保持账户余额一致性
func (s *ExecuteTransferStep) Compensate(ctx context.Context, data interface{}) error {
	transferResult, ok := data.(*TransferResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *TransferResult")
	}

	// 检查转账是否已完成（只有完成的转账才需要回退资金）
	status, err := s.transferService.GetTransferStatus(ctx, transferResult.TransferID)
	if err != nil {
		return fmt.Errorf("查询转账状态失败: %w", err)
	}

	// 如果转账未完成，只需取消记录即可
	if status != "completed" {
		err := s.transferService.CancelTransfer(ctx, transferResult.TransferID, "saga_compensation")
		if err != nil {
			return fmt.Errorf("取消转账失败: %w", err)
		}
		return nil
	}

	// 转账已完成，需要回退资金
	// 注意：这些操作的顺序很重要，应该按照与正向操作相反的顺序执行

	// 1. 从目标账户扣除金额（回退增加的金额）
	// 使用负数金额来表示扣除操作，或者调用专门的扣除接口
	err = s.accountService.AddAmount(ctx, transferResult.ToAccount, -transferResult.Amount,
		fmt.Sprintf("compensation-%s", transferResult.TransactionID))
	if err != nil {
		// 目标账户扣除失败是严重问题，必须人工处理
		return fmt.Errorf("从目标账户 %s 扣除金额 %.2f 失败，需要人工处理: %w",
			transferResult.ToAccount, transferResult.Amount, err)
	}

	// 2. 退款到源账户（恢复被扣除的金额）
	// 注意：源账户的资金是通过 DeductFrozenAmount 扣除的，现在需要退回
	err = s.accountService.AddAmount(ctx, transferResult.FromAccount, transferResult.Amount,
		fmt.Sprintf("refund-%s", transferResult.TransactionID))
	if err != nil {
		// 退款失败但目标账户已扣除，这是最严重的情况
		// 此时资金既不在源账户也不在目标账户，必须立即处理
		return fmt.Errorf("退款到源账户 %s 失败 (金额 %.2f 已从目标账户扣除)，需要紧急人工处理: %w",
			transferResult.FromAccount, transferResult.Amount, err)
	}

	// 3. 取消转账记录（标记为已回退）
	err = s.transferService.CancelTransfer(ctx, transferResult.TransferID, "saga_compensation")
	if err != nil {
		// 资金已回退但记录更新失败，记录日志但不返回错误
		// 因为资金一致性比记录更新更重要
		return fmt.Errorf("取消转账记录失败（资金已回退）: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *ExecuteTransferStep) GetTimeout() time.Duration {
	return 30 * time.Second // 转账操作可能需要更长时间
}

// GetRetryPolicy 返回步骤的重试策略
func (s *ExecuteTransferStep) GetRetryPolicy() saga.RetryPolicy {
	// 转账操作重试次数较少，避免重复转账
	return saga.NewExponentialBackoffRetryPolicy(2, 2*time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *ExecuteTransferStep) IsRetryable(err error) bool {
	// 转账操作需要谨慎重试，只有明确的网络临时错误才重试
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// 余额不足、账户冻结等业务错误不重试
	return false
}

// GetMetadata 返回步骤元数据
func (s *ExecuteTransferStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "transfer_execution",
		"service":   "transfer-service",
		"critical":  true, // 最关键的步骤
	}
}

// ReleaseFreeze 释放冻结步骤
// 此步骤在转账成功后释放冻结（实际上已经扣除，所以这一步主要是清理冻结记录）
type ReleaseFreezeStep struct {
	service AccountServiceClient
}

// GetID 返回步骤ID
func (s *ReleaseFreezeStep) GetID() string {
	return "release-freeze"
}

// GetName 返回步骤名称
func (s *ReleaseFreezeStep) GetName() string {
	return "释放冻结"
}

// GetDescription 返回步骤描述
func (s *ReleaseFreezeStep) GetDescription() string {
	return "转账成功后清理冻结记录，释放冻结状态"
}

// Execute 执行释放冻结操作
// 输入：TransferResult（从上一步传递）
// 输出：UnfreezeResult
func (s *ReleaseFreezeStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	transferResult, ok := data.(*TransferResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *TransferResult")
	}

	// 从元数据中获取冻结ID
	// 注意：实际实现中可能需要从 transferResult 中获取冻结ID
	// 这里简化处理

	result := &UnfreezeResult{
		UnfreezeID: generateUnfreezeID(),
		AccountID:  transferResult.FromAccount,
		Amount:     transferResult.Amount,
		UnfrozenAt: time.Now(),
		Reason:     "transfer_completed",
		Metadata:   transferResult.Metadata,
	}

	return result, nil
}

// Compensate 补偿操作：释放冻结步骤通常不需要补偿
func (s *ReleaseFreezeStep) Compensate(ctx context.Context, data interface{}) error {
	// 释放冻结步骤不需要补偿
	// 如果转账失败，前面的步骤会处理资金回退
	return nil
}

// GetTimeout 返回步骤超时时间
func (s *ReleaseFreezeStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *ReleaseFreezeStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *ReleaseFreezeStep) IsRetryable(err error) bool {
	return !errors.Is(err, context.DeadlineExceeded)
}

// GetMetadata 返回步骤元数据
func (s *ReleaseFreezeStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "freeze_release",
		"service":   "account-service",
	}
}

// CreateAuditRecordStep 创建审计记录步骤
// 此步骤创建转账的审计记录，用于合规和追踪
type CreateAuditRecordStep struct {
	service AuditServiceClient
}

// GetID 返回步骤ID
func (s *CreateAuditRecordStep) GetID() string {
	return "create-audit-record"
}

// GetName 返回步骤名称
func (s *CreateAuditRecordStep) GetName() string {
	return "创建审计记录"
}

// GetDescription 返回步骤描述
func (s *CreateAuditRecordStep) GetDescription() string {
	return "创建转账的审计记录，记录完整的转账过程用于合规审查"
}

// Execute 执行创建审计记录操作
// 输入：UnfreezeResult（从上一步传递）
// 输出：AuditRecordResult
func (s *CreateAuditRecordStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	unfreezeResult, ok := data.(*UnfreezeResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *UnfreezeResult")
	}

	// 从元数据中获取转账信息
	transferID, _ := unfreezeResult.Metadata["transfer_id"].(string)

	// 构造转账数据和结果（用于审计）
	auditTransferData := &TransferData{
		TransferID:  transferID,
		FromAccount: unfreezeResult.Metadata["from_account"].(string),
		ToAccount:   unfreezeResult.Metadata["to_account"].(string),
		Amount:      unfreezeResult.Amount,
		Currency:    unfreezeResult.Metadata["currency"].(string),
		CustomerID:  unfreezeResult.Metadata["customer_id"].(string),
		RiskLevel:   unfreezeResult.Metadata["risk_level"].(string),
	}

	auditTransferResult := &TransferResult{
		TransferID: transferID,
		Status:     "completed",
	}

	// 创建审计记录
	auditResult, err := s.service.CreateAuditRecord(ctx, transferID, auditTransferData, auditTransferResult)
	if err != nil {
		return nil, fmt.Errorf("创建审计记录失败: %w", err)
	}

	auditResult.Metadata = unfreezeResult.Metadata

	return auditResult, nil
}

// Compensate 补偿操作：更新审计记录状态为已取消
func (s *CreateAuditRecordStep) Compensate(ctx context.Context, data interface{}) error {
	auditResult, ok := data.(*AuditRecordResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *AuditRecordResult")
	}

	// 更新审计记录状态
	err := s.service.UpdateAuditRecord(ctx, auditResult.AuditID, "cancelled", map[string]interface{}{
		"cancelled_reason": "saga_compensation",
		"cancelled_at":     time.Now(),
	})
	if err != nil {
		// 审计记录更新失败不应阻止补偿流程
		return fmt.Errorf("更新审计记录失败: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *CreateAuditRecordStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *CreateAuditRecordStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *CreateAuditRecordStep) IsRetryable(err error) bool {
	return !errors.Is(err, context.DeadlineExceeded)
}

// GetMetadata 返回步骤元数据
func (s *CreateAuditRecordStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "audit_record",
		"service":   "audit-service",
		"critical":  true, // 审计记录对于合规很重要
	}
}

// SendTransferNotificationStep 发送转账通知步骤
// 此步骤向客户发送转账完成通知
type SendTransferNotificationStep struct {
	service NotificationServiceClient
}

// GetID 返回步骤ID
func (s *SendTransferNotificationStep) GetID() string {
	return "send-notification"
}

// GetName 返回步骤名称
func (s *SendTransferNotificationStep) GetName() string {
	return "发送通知"
}

// GetDescription 返回步骤描述
func (s *SendTransferNotificationStep) GetDescription() string {
	return "向客户发送转账完成通知，提供交易详情"
}

// Execute 执行发送通知操作
// 输入：AuditRecordResult（从上一步传递）
// 输出：NotificationResult
func (s *SendTransferNotificationStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	auditResult, ok := data.(*AuditRecordResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *AuditRecordResult")
	}

	// 从元数据中获取转账信息
	transferID, _ := auditResult.Metadata["transfer_id"].(string)
	customerID, _ := auditResult.Metadata["customer_id"].(string)

	// 构造转账数据用于通知
	transferData := &TransferData{
		TransferID:  transferID,
		FromAccount: auditResult.Metadata["from_account"].(string),
		ToAccount:   auditResult.Metadata["to_account"].(string),
		Amount:      auditResult.Metadata["amount"].(float64),
		Currency:    auditResult.Metadata["currency"].(string),
		CustomerID:  customerID,
	}

	// 发送通知
	recipients := []string{customerID}
	notificationResult, err := s.service.SendTransferNotification(ctx, transferID, recipients, transferData)
	if err != nil {
		// 通知发送失败不应导致整个 Saga 失败
		// 可以记录日志并稍后重试
		return nil, fmt.Errorf("发送通知失败: %w", err)
	}

	notificationResult.Metadata = auditResult.Metadata

	return notificationResult, nil
}

// Compensate 补偿操作：发送取消通知
func (s *SendTransferNotificationStep) Compensate(ctx context.Context, data interface{}) error {
	notificationResult, ok := data.(*NotificationResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *NotificationResult")
	}

	// 发送失败通知
	err := s.service.SendFailureNotification(ctx, notificationResult.TransferID, "transfer_cancelled")
	if err != nil {
		// 失败通知发送失败不应阻止补偿流程
		return fmt.Errorf("发送失败通知失败: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *SendTransferNotificationStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *SendTransferNotificationStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *SendTransferNotificationStep) IsRetryable(err error) bool {
	return !errors.Is(err, context.DeadlineExceeded)
}

// GetMetadata 返回步骤元数据
func (s *SendTransferNotificationStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "notification",
		"service":   "notification-service",
		"critical":  false, // 通知不是关键步骤
	}
}

// ==========================
// Saga 定义
// ==========================

// PaymentProcessingSagaDefinition 支付处理 Saga 定义
type PaymentProcessingSagaDefinition struct {
	id                   string
	name                 string
	description          string
	steps                []saga.SagaStep
	timeout              time.Duration
	retryPolicy          saga.RetryPolicy
	compensationStrategy saga.CompensationStrategy
	metadata             map[string]interface{}
}

// GetID 返回 Saga 定义ID
func (d *PaymentProcessingSagaDefinition) GetID() string {
	return d.id
}

// GetName 返回 Saga 名称
func (d *PaymentProcessingSagaDefinition) GetName() string {
	return d.name
}

// GetDescription 返回 Saga 描述
func (d *PaymentProcessingSagaDefinition) GetDescription() string {
	return d.description
}

// GetSteps 返回所有步骤
func (d *PaymentProcessingSagaDefinition) GetSteps() []saga.SagaStep {
	return d.steps
}

// GetTimeout 返回超时时间
func (d *PaymentProcessingSagaDefinition) GetTimeout() time.Duration {
	return d.timeout
}

// GetRetryPolicy 返回重试策略
func (d *PaymentProcessingSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
	return d.retryPolicy
}

// GetCompensationStrategy 返回补偿策略
func (d *PaymentProcessingSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy {
	return d.compensationStrategy
}

// GetMetadata 返回元数据
func (d *PaymentProcessingSagaDefinition) GetMetadata() map[string]interface{} {
	return d.metadata
}

// Validate 验证 Saga 定义的有效性
func (d *PaymentProcessingSagaDefinition) Validate() error {
	if d.id == "" {
		return errors.New("saga ID 不能为空")
	}
	if d.name == "" {
		return errors.New("saga 名称不能为空")
	}
	if len(d.steps) == 0 {
		return errors.New("至少需要一个步骤")
	}
	if d.timeout <= 0 {
		return errors.New("超时时间必须大于0")
	}
	return nil
}

// ==========================
// 工厂函数
// ==========================

// NewPaymentProcessingSaga 创建支付处理 Saga 定义
func NewPaymentProcessingSaga(
	accountService AccountServiceClient,
	transferService TransferServiceClient,
	auditService AuditServiceClient,
	notificationService NotificationServiceClient,
) *PaymentProcessingSagaDefinition {
	// 创建步骤
	steps := []saga.SagaStep{
		&ValidateAccountsStep{service: accountService},
		&FreezeAmountStep{service: accountService},
		&ExecuteTransferStep{
			accountService:  accountService,
			transferService: transferService,
		},
		&ReleaseFreezeStep{service: accountService},
		&CreateAuditRecordStep{service: auditService},
		&SendTransferNotificationStep{service: notificationService},
	}

	return &PaymentProcessingSagaDefinition{
		id:          "payment-processing-saga",
		name:        "支付处理Saga",
		description: "处理跨账户资金转账的完整流程：验证账户→冻结资金→执行转账→释放冻结→审计记录→发送通知",
		steps:       steps,
		timeout:     10 * time.Minute, // 整个流程10分钟超时（金融操作需要较短超时）
		retryPolicy: saga.NewExponentialBackoffRetryPolicy(
			3,                    // 最多重试3次
			500*time.Millisecond, // 初始延迟500ms（金融操作需要快速响应）
			5*time.Second,        // 最大延迟5秒
		),
		compensationStrategy: saga.NewSequentialCompensationStrategy(
			5 * time.Minute, // 补偿超时5分钟
		),
		metadata: map[string]interface{}{
			"saga_type":       "payment_processing",
			"business_domain": "finance",
			"version":         "1.0.0",
			"critical":        true, // 标记为关键业务流程
			"compliance":      true, // 需要合规审计
		},
	}
}

// ==========================
// 辅助函数
// ==========================

// generateValidationCode 生成验证码
func generateValidationCode(data *TransferData) string {
	// 实际实现中应该使用加密哈希
	return fmt.Sprintf("%s-%s-%.2f-%d",
		data.FromAccount, data.ToAccount, data.Amount, time.Now().Unix())
}

// generateTransferID 生成转账ID
func generateTransferID() string {
	return fmt.Sprintf("TXN-%d", time.Now().UnixNano())
}

// generateUnfreezeID 生成解冻ID
func generateUnfreezeID() string {
	return fmt.Sprintf("UNF-%d", time.Now().UnixNano())
}
