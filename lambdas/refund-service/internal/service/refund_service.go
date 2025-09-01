package service

import (
	"context"
	"fmt"
	"time"

	"github.com/draftea-coding-challenge/lambdas/refund-service/internal/repository"
	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/draftea-coding-challenge/shared/types"
	"github.com/google/uuid"
)

// RefundService handles refund business logic
type RefundService struct {
	repo   *repository.RefundRepository
	logger *observability.Logger
}

// NewRefundService creates a new refund service
func NewRefundService(repo *repository.RefundRepository, logger *observability.Logger) *RefundService {
	return &RefundService{
		repo:   repo,
		logger: logger,
	}
}

// RefundRequest represents a refund request
type RefundRequest struct {
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Reason    string  `json:"reason"`
}

// RefundResponse represents a refund response
type RefundResponse struct {
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	Reason    string  `json:"reason,omitempty"`
	RefundID  string  `json:"refund_id,omitempty"`
}

// ProcessRefund processes a refund request
func (s *RefundService) ProcessRefund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	// Validate request
	if err := s.validateRefundRequest(req); err != nil {
		s.logger.Warn("Invalid refund request", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	// Get original payment
	payment, err := s.repo.GetPayment(req.PaymentID)
	if err != nil {
		s.logger.Error("Failed to get payment", err, map[string]interface{}{
			"payment_id": req.PaymentID,
		})
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	// Check payment status
	if payment.Status == types.PaymentStatusRefunded {
		return nil, fmt.Errorf("payment already refunded")
	}

	if payment.Status != types.PaymentStatusCompleted {
		return nil, fmt.Errorf("only completed payments can be refunded")
	}

	// Validate refund amount
	if req.Amount > payment.Amount {
		return nil, fmt.Errorf("refund amount exceeds payment amount")
	}

	// Credit wallet
	if err := s.repo.CreditWallet(payment.UserID, req.Amount); err != nil {
		s.logger.Error("Failed to credit wallet", err, map[string]interface{}{
			"user_id": payment.UserID,
			"amount":  req.Amount,
		})
		return nil, fmt.Errorf("failed to process refund: %w", err)
	}

	// Update payment status
	payment.Status = types.PaymentStatusRefunded
	payment.UpdatedAt = time.Now()
	
	if err := s.repo.UpdatePayment(payment); err != nil {
		s.logger.Error("Failed to update payment status", err, map[string]interface{}{
			"payment_id": payment.ID,
		})
		// Log error but don't fail the refund since wallet was already credited
	}

	// Log refund event
	refundID := uuid.New().String()
	event := &types.PaymentEvent{
		ID:            refundID,
		PaymentID:     payment.ID,
		UserID:        payment.UserID,
		EventType:     "REFUNDED",
		Amount:        req.Amount,
		Status:        "completed",
		CorrelationID: payment.CorrelationID,
		Timestamp:     time.Now(),
		Metadata: map[string]interface{}{
			"reason": req.Reason,
		},
	}
	
	if err := s.repo.LogPaymentEvent(event); err != nil {
		s.logger.Error("Failed to log refund event", err, map[string]interface{}{
			"payment_id": payment.ID,
			"refund_id":  refundID,
		})
	}

	s.logger.Info("Refund processed successfully", map[string]interface{}{
		"payment_id": payment.ID,
		"refund_id":  refundID,
		"amount":     req.Amount,
		"user_id":    payment.UserID,
	})

	return &RefundResponse{
		PaymentID: payment.ID,
		Amount:    req.Amount,
		Status:    "refunded",
		Reason:    req.Reason,
		RefundID:  refundID,
	}, nil
}

// ProcessFullRefund processes a full refund for a payment
func (s *RefundService) ProcessFullRefund(ctx context.Context, paymentID string) (*RefundResponse, error) {
	// Get original payment
	payment, err := s.repo.GetPayment(paymentID)
	if err != nil {
		s.logger.Error("Failed to get payment", err, map[string]interface{}{
			"payment_id": paymentID,
		})
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	// Process full refund
	return s.ProcessRefund(ctx, &RefundRequest{
		PaymentID: paymentID,
		Amount:    payment.Amount,
		Reason:    "Full refund",
	})
}

// GetRefundStatus gets the refund status of a payment
func (s *RefundService) GetRefundStatus(ctx context.Context, paymentID string) (*RefundStatusResponse, error) {
	if paymentID == "" {
		return nil, fmt.Errorf("payment_id is required")
	}

	payment, err := s.repo.GetPayment(paymentID)
	if err != nil {
		s.logger.Error("Failed to get payment", err, map[string]interface{}{
			"payment_id": paymentID,
		})
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	return &RefundStatusResponse{
		PaymentID: payment.ID,
		Status:    string(payment.Status),
		Refunded:  payment.Status == types.PaymentStatusRefunded,
		Amount:    payment.Amount,
	}, nil
}

// ProcessStepFunctionRefund processes a refund from Step Function input
func (s *RefundService) ProcessStepFunctionRefund(ctx context.Context, input *types.StepFunctionInput) (*types.LambdaResponse, error) {
	payment, err := s.repo.GetPayment(input.PaymentID)
	if err != nil {
		return &types.LambdaResponse{
			Success: false,
			Error:   "Payment not found",
		}, nil
	}

	// Credit wallet with full payment amount
	if err := s.repo.CreditWallet(payment.UserID, payment.Amount); err != nil {
		return &types.LambdaResponse{
			Success: false,
			Error:   "Failed to credit wallet",
		}, nil
	}

	// Update payment status
	payment.Status = types.PaymentStatusRefunded
	payment.UpdatedAt = time.Now()
	s.repo.UpdatePayment(payment)

	// Log refund event
	refundID := uuid.New().String()
	event := &types.PaymentEvent{
		ID:            refundID,
		PaymentID:     payment.ID,
		UserID:        payment.UserID,
		EventType:     "REFUNDED",
		Amount:        payment.Amount,
		Status:        "completed",
		CorrelationID: payment.CorrelationID,
		Timestamp:     time.Now(),
		Metadata: map[string]interface{}{
			"reason": "Step Function refund",
		},
	}
	s.repo.LogPaymentEvent(event)

	return &types.LambdaResponse{
		Success: true,
		Data: map[string]interface{}{
			"payment_id": payment.ID,
			"amount":     payment.Amount,
			"status":     "refunded",
			"refund_id":  refundID,
		},
	}, nil
}

// CheckStepFunctionRefundStatus checks refund status from Step Function input
func (s *RefundService) CheckStepFunctionRefundStatus(ctx context.Context, input *types.StepFunctionInput) (*types.LambdaResponse, error) {
	payment, err := s.repo.GetPayment(input.PaymentID)
	if err != nil {
		return &types.LambdaResponse{
			Success: false,
			Error:   "Payment not found",
		}, nil
	}

	return &types.LambdaResponse{
		Success: true,
		Data: map[string]interface{}{
			"payment_id": payment.ID,
			"status":     payment.Status,
			"refunded":   payment.Status == types.PaymentStatusRefunded,
		},
	}, nil
}

// RefundStatusResponse represents a refund status response
type RefundStatusResponse struct {
	PaymentID string  `json:"payment_id"`
	Status    string  `json:"status"`
	Refunded  bool    `json:"refunded"`
	Amount    float64 `json:"amount"`
}

// validateRefundRequest validates a refund request
func (s *RefundService) validateRefundRequest(req *RefundRequest) error {
	if req.PaymentID == "" {
		return fmt.Errorf("payment_id is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if req.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}
