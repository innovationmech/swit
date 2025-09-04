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

package model

import (
	"time"
)

// Payment represents a payment transaction
type Payment struct {
	TransactionID     string                   `json:"transaction_id"`
	CustomerID        string                   `json:"customer_id"`
	OrderID           string                   `json:"order_id"`
	Amount            float64                  `json:"amount"`
	Currency          string                   `json:"currency"`
	Status            PaymentStatus            `json:"status"`
	PaymentMethod     PaymentMethod            `json:"payment_method"`
	ProcessingDetails PaymentProcessingDetails `json:"processing_details"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
}

// PaymentStatus represents the possible payment statuses
type PaymentStatus string

const (
	PaymentStatusPending           PaymentStatus = "pending"
	PaymentStatusProcessing        PaymentStatus = "processing"
	PaymentStatusCompleted         PaymentStatus = "completed"
	PaymentStatusFailed            PaymentStatus = "failed"
	PaymentStatusCancelled         PaymentStatus = "cancelled"
	PaymentStatusRefunded          PaymentStatus = "refunded"
	PaymentStatusPartiallyRefunded PaymentStatus = "partially_refunded"
)

// IsValidStatus checks if the status is valid
func (s PaymentStatus) IsValidStatus() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusProcessing, PaymentStatusCompleted,
		PaymentStatusFailed, PaymentStatusCancelled, PaymentStatusRefunded,
		PaymentStatusPartiallyRefunded:
		return true
	}
	return false
}

// String returns string representation of the status
func (s PaymentStatus) String() string {
	return string(s)
}

// PaymentMethod represents payment method information
type PaymentMethod struct {
	Type              PaymentMethodType  `json:"type"`
	CardInfo          *CardInfo          `json:"card_info,omitempty"`
	BankAccountInfo   *BankAccountInfo   `json:"bank_account_info,omitempty"`
	DigitalWalletInfo *DigitalWalletInfo `json:"digital_wallet_info,omitempty"`
}

// PaymentMethodType represents the type of payment method
type PaymentMethodType string

const (
	PaymentMethodTypeCreditCard     PaymentMethodType = "credit_card"
	PaymentMethodTypeDebitCard      PaymentMethodType = "debit_card"
	PaymentMethodTypeBankTransfer   PaymentMethodType = "bank_transfer"
	PaymentMethodTypeDigitalWallet  PaymentMethodType = "digital_wallet"
	PaymentMethodTypeCashOnDelivery PaymentMethodType = "cash_on_delivery"
)

// CardInfo represents credit/debit card information
type CardInfo struct {
	LastFourDigits string `json:"last_four_digits"`
	Brand          string `json:"brand"`
	ExpiryMonth    int32  `json:"expiry_month"`
	ExpiryYear     int32  `json:"expiry_year"`
	CardholderName string `json:"cardholder_name"`
}

// BankAccountInfo represents bank account information
type BankAccountInfo struct {
	BankName          string `json:"bank_name"`
	AccountHolderName string `json:"account_holder_name"`
	LastFourDigits    string `json:"last_four_digits"`
	RoutingNumber     string `json:"routing_number"`
}

// DigitalWalletInfo represents digital wallet information
type DigitalWalletInfo struct {
	Provider  string `json:"provider"`
	AccountID string `json:"account_id"`
}

// PaymentProcessingDetails contains processing-specific information
type PaymentProcessingDetails struct {
	ProcessorID            string        `json:"processor_id"`
	ProcessorTransactionID string        `json:"processor_transaction_id"`
	ProcessingFee          float64       `json:"processing_fee"`
	GatewayResponseCode    string        `json:"gateway_response_code"`
	GatewayResponseMessage string        `json:"gateway_response_message"`
	RiskScore              int32         `json:"risk_score"`
	ProcessingTime         time.Duration `json:"processing_time"`
}

// PaymentRefund represents refund information
type PaymentRefund struct {
	RefundID              string       `json:"refund_id"`
	OriginalTransactionID string       `json:"original_transaction_id"`
	Amount                float64      `json:"amount"`
	Currency              string       `json:"currency"`
	Reason                string       `json:"reason"`
	Status                RefundStatus `json:"status"`
	CreatedAt             time.Time    `json:"created_at"`
}

// RefundStatus represents possible refund statuses
type RefundStatus string

const (
	RefundStatusPending    RefundStatus = "pending"
	RefundStatusProcessing RefundStatus = "processing"
	RefundStatusCompleted  RefundStatus = "completed"
	RefundStatusFailed     RefundStatus = "failed"
	RefundStatusCancelled  RefundStatus = "cancelled"
)

// ProcessPaymentRequest represents a request to process a payment
type ProcessPaymentRequest struct {
	CustomerID    string        `json:"customer_id"`
	OrderID       string        `json:"order_id"`
	Amount        float64       `json:"amount"`
	Currency      string        `json:"currency"`
	PaymentMethod PaymentMethod `json:"payment_method"`
}

// ProcessPaymentResponse represents the response to payment processing
type ProcessPaymentResponse struct {
	Payment *Payment `json:"payment"`
}

// ValidatePaymentRequest represents a request to validate payment info
type ValidatePaymentRequest struct {
	CustomerID    string        `json:"customer_id"`
	Amount        float64       `json:"amount"`
	Currency      string        `json:"currency"`
	PaymentMethod PaymentMethod `json:"payment_method"`
}

// ValidatePaymentResponse represents the response to payment validation
type ValidatePaymentResponse struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors,omitempty"`
}

// GetPaymentStatusRequest represents a request to get payment status
type GetPaymentStatusRequest struct {
	TransactionID string `json:"transaction_id"`
}

// GetPaymentStatusResponse represents the response to payment status request
type GetPaymentStatusResponse struct {
	Payment *Payment `json:"payment"`
}

// RefundPaymentRequest represents a request to refund a payment
type RefundPaymentRequest struct {
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
}

// RefundPaymentResponse represents the response to refund request
type RefundPaymentResponse struct {
	Refund *PaymentRefund `json:"refund"`
}
