package service

import (
	"context"
	"fmt"
	"time"

	"github.com/draftea-coding-challenge/lambdas/invoice-processor/internal/repository"
	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/draftea-coding-challenge/shared/types"
)

// PaymentService handles business logic for payments
type PaymentService struct {
	repo   *repository.PaymentRepository
	logger *observability.Logger
}

// NewPaymentService creates a new payment service
func NewPaymentService(repo *repository.PaymentRepository, logger *observability.Logger) *PaymentService {
	return &PaymentService{
		repo:   repo,
		logger: logger,
	}
}

// CreatePaymentRequest represents a payment creation request
type CreatePaymentRequest struct {
	UserID         string            `json:"userId"`
	Amount         float64           `json:"amount"`
	Currency       string            `json:"currency"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	IdempotencyKey string            `json:"idempotencyKey,omitempty"`
	CorrelationID  string            `json:"correlationId,omitempty"`
}

// CreatePayment handles payment creation with idempotency
func (s *PaymentService) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*types.Payment, error) {
	// Validate request
	if err := s.validateCreatePaymentRequest(req); err != nil {
		return nil, err
	}

	// Check idempotency
	if req.IdempotencyKey != "" {
		existingPayment, err := s.repo.CheckIdempotency(ctx, req.IdempotencyKey)
		if err != nil {
			s.logger.Error("Failed to check idempotency", err, nil)
		} else if existingPayment != nil {
			s.logger.Info("Idempotent request, returning existing payment", map[string]interface{}{
				"paymentId":      existingPayment.ID,
				"idempotencyKey": req.IdempotencyKey,
			})
			return existingPayment, nil
		}
	}

	// Create payment
	payment := &types.Payment{
		UserID:        req.UserID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Status:        types.PaymentStatusPending,
		CorrelationID: req.CorrelationID,
		Metadata:      req.Metadata,
	}

	if payment.Metadata == nil {
		payment.Metadata = make(map[string]string)
	}

	if req.IdempotencyKey != "" {
		payment.Metadata["idempotencyKey"] = req.IdempotencyKey
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		s.logger.Error("Failed to create payment", err, map[string]interface{}{
			"userId": req.UserID,
			"amount": req.Amount,
		})
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	s.logger.Info("Payment created successfully", map[string]interface{}{
		"paymentId": payment.ID,
		"userId":    payment.UserID,
		"amount":    payment.Amount,
		"status":    payment.Status,
	})

	return payment, nil
}

// GetPayment retrieves a payment by ID
func (s *PaymentService) GetPayment(ctx context.Context, paymentID string) (*types.Payment, error) {
	if paymentID == "" {
		return nil, fmt.Errorf("payment ID is required")
	}

	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		s.logger.Error("Failed to get payment", err, map[string]interface{}{
			"paymentId": paymentID,
		})
		return nil, err
	}

	return payment, nil
}

// CreatePaymentFromStepFunction creates a payment from Step Function input
func (s *PaymentService) CreatePaymentFromStepFunction(ctx context.Context, input types.StepFunctionInput) (*types.Payment, error) {
	payment := &types.Payment{
		ID:            input.PaymentID,
		UserID:        input.UserID,
		Amount:        input.Amount,
		Currency:      input.Currency,
		CorrelationID: input.CorrelationID,
		Metadata:      input.Metadata,
		Status:        types.PaymentStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	return payment, nil
}

// UpdatePaymentStatus updates the status of a payment
func (s *PaymentService) UpdatePaymentStatus(ctx context.Context, paymentID string, status types.PaymentStatus, externalID string) (*types.Payment, error) {
	if err := s.repo.UpdatePaymentStatus(ctx, paymentID, status, externalID); err != nil {
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

	// Get updated payment
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated payment: %w", err)
	}

	return payment, nil
}

// validateCreatePaymentRequest validates the payment creation request
func (s *PaymentService) validateCreatePaymentRequest(req CreatePaymentRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	if req.Currency == "" || len(req.Currency) != 3 {
		return fmt.Errorf("invalid currency format")
	}

	return nil
}
