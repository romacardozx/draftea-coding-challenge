package types

import (
	"time"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "PENDING"
	PaymentStatusProcessing PaymentStatus = "PROCESSING"
	PaymentStatusCompleted  PaymentStatus = "COMPLETED"
	PaymentStatusFailed     PaymentStatus = "FAILED"
	PaymentStatusRefunded   PaymentStatus = "REFUNDED"
)

type Payment struct {
	ID            string            `json:"id" dynamodbav:"ID"`
	UserID        string            `json:"userId" dynamodbav:"UserID"`
	Amount        float64           `json:"amount" dynamodbav:"Amount"`
	Currency      string            `json:"currency" dynamodbav:"Currency"`
	Status        PaymentStatus     `json:"status" dynamodbav:"Status"`
	ExternalID    string            `json:"externalId,omitempty" dynamodbav:"ExternalID,omitempty"`
	CorrelationID string            `json:"correlationId" dynamodbav:"CorrelationID"`
	Metadata      map[string]string `json:"metadata,omitempty" dynamodbav:"Metadata,omitempty"`
	RefundReason  string            `json:"refundReason,omitempty" dynamodbav:"RefundReason,omitempty"`
	CreatedAt     time.Time         `json:"createdAt" dynamodbav:"CreatedAt"`
	UpdatedAt     time.Time         `json:"updatedAt" dynamodbav:"UpdatedAt"`
}

type PaymentRequest struct {
	UserID        string            `json:"userId"`
	Amount        float64           `json:"amount"`
	Currency      string            `json:"currency"`
	IdempotencyKey string           `json:"idempotencyKey"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type StepFunctionInput struct {
	Action        string            `json:"action"`
	PaymentID     string            `json:"paymentId,omitempty"`
	UserID        string            `json:"userId,omitempty"`
	Amount        float64           `json:"amount,omitempty"`
	Currency      string            `json:"currency,omitempty"`
	Status        string            `json:"status,omitempty"`
	ExternalID    string            `json:"externalId,omitempty"`
	CorrelationID string            `json:"correlationId,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type LambdaResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type PaymentResponse struct {
	ID      string        `json:"id"`
	Status  PaymentStatus `json:"status"`
	Message string        `json:"message,omitempty"`
}

type Wallet struct {
	UserID    string    `json:"userId" dynamodbav:"UserID"`
	Balance   float64   `json:"balance" dynamodbav:"Balance"`
	Currency  string    `json:"currency" dynamodbav:"Currency"`
	Version   int       `json:"version" dynamodbav:"Version"`
	UpdatedAt time.Time `json:"updatedAt" dynamodbav:"UpdatedAt"`
	CreatedAt time.Time `json:"createdAt" dynamodbav:"CreatedAt"`
}

type WalletTransaction struct {
	ID            string    `json:"id" dynamodbav:"ID"`
	UserID        string    `json:"userId" dynamodbav:"UserID"`
	PaymentID     string    `json:"paymentId" dynamodbav:"PaymentID"`
	Type          string    `json:"type" dynamodbav:"Type"` // DEBIT, CREDIT
	Amount        float64   `json:"amount" dynamodbav:"Amount"`
	BalanceBefore float64   `json:"balanceBefore" dynamodbav:"BalanceBefore"`
	BalanceAfter  float64   `json:"balanceAfter" dynamodbav:"BalanceAfter"`
	Timestamp     time.Time `json:"timestamp" dynamodbav:"Timestamp"`
}