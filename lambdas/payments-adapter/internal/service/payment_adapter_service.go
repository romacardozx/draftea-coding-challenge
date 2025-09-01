package service

import (
	"context"
	"fmt"

	"github.com/draftea-coding-challenge/lambdas/payments-adapter/internal/circuitbreaker"
	"github.com/draftea-coding-challenge/lambdas/payments-adapter/internal/gateway"
	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/draftea-coding-challenge/shared/types"
)

// PaymentAdapterService handles business logic for payment gateway interactions
type PaymentAdapterService struct {
	gateway *circuitbreaker.CircuitBreakerClient
	logger  *observability.Logger
}

// NewPaymentAdapterService creates a new payment adapter service
func NewPaymentAdapterService(gatewayClient gateway.PaymentGatewayClient, logger *observability.Logger) *PaymentAdapterService {
	// Wrap the gateway client with circuit breaker
	cbClient := circuitbreaker.NewCircuitBreakerClient(gatewayClient, "payment-gateway")
	
	return &PaymentAdapterService{
		gateway: cbClient,
		logger:  logger,
	}
}

// ProcessPaymentRequest represents a payment processing request
type ProcessPaymentRequest struct {
	PaymentID     string            `json:"paymentId"`
	UserID        string            `json:"userId"`
	Amount        float64           `json:"amount"`
	Currency      string            `json:"currency"`
	Status        string            `json:"status"`
	CorrelationID string            `json:"correlationId"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// PaymentStatusRequest represents a payment status check request
type PaymentStatusRequest struct {
	ExternalID string `json:"externalId"`
}

// ProcessPayment processes a payment through the external gateway
func (s *PaymentAdapterService) ProcessPayment(ctx context.Context, req *ProcessPaymentRequest) (*gateway.GatewayResponse, error) {
	// Validate request
	if err := s.validateProcessPaymentRequest(req); err != nil {
		return nil, err
	}

	// Convert to Payment type for gateway
	payment := &types.Payment{
		ID:            req.PaymentID,
		UserID:        req.UserID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Status:        types.PaymentStatus(req.Status),
		CorrelationID: req.CorrelationID,
		Metadata:      req.Metadata,
	}

	// Process through gateway with circuit breaker
	resp, err := s.gateway.ProcessPayment(ctx, payment)
	if err != nil {
		s.logger.Error("Failed to process payment through gateway", err, map[string]interface{}{
			"paymentId":     payment.ID,
			"userId":        payment.UserID,
			"amount":        payment.Amount,
			"correlationId": payment.CorrelationID,
		})
		return nil, fmt.Errorf("gateway processing failed: %w", err)
	}

	s.logger.Info("Payment processed successfully", map[string]interface{}{
		"paymentId":   payment.ID,
		"externalId":  resp.ExternalID,
		"status":      resp.Status,
	})

	return resp, nil
}

// ProcessPaymentFromPayment processes a payment object directly
func (s *PaymentAdapterService) ProcessPaymentFromPayment(ctx context.Context, payment *types.Payment) (*gateway.GatewayResponse, error) {
	if payment == nil {
		return nil, fmt.Errorf("payment is required")
	}

	// Process through gateway with circuit breaker
	resp, err := s.gateway.ProcessPayment(ctx, payment)
	if err != nil {
		s.logger.Error("Failed to process payment through gateway", err, map[string]interface{}{
			"paymentId":     payment.ID,
			"userId":        payment.UserID,
			"amount":        payment.Amount,
			"correlationId": payment.CorrelationID,
		})
		return nil, fmt.Errorf("gateway processing failed: %w", err)
	}

	s.logger.Info("Payment processed successfully", map[string]interface{}{
		"paymentId":   payment.ID,
		"externalId":  resp.ExternalID,
		"status":      resp.Status,
	})

	return resp, nil
}

// GetPaymentStatus retrieves payment status from the external gateway
func (s *PaymentAdapterService) GetPaymentStatus(ctx context.Context, externalID string) (*gateway.GatewayResponse, error) {
	if externalID == "" {
		return nil, fmt.Errorf("externalID is required")
	}

	resp, err := s.gateway.GetPaymentStatus(ctx, externalID)
	if err != nil {
		s.logger.Error("Failed to get payment status from gateway", err, map[string]interface{}{
			"externalId": externalID,
		})
		return nil, fmt.Errorf("failed to get payment status: %w", err)
	}

	s.logger.Info("Payment status retrieved", map[string]interface{}{
		"externalId": externalID,
		"status":     resp.Status,
	})

	return resp, nil
}

// ProcessStepFunctionPayment processes payment from Step Function input
func (s *PaymentAdapterService) ProcessStepFunctionPayment(ctx context.Context, input *types.StepFunctionInput) (*types.LambdaResponse, error) {
	payment := &types.Payment{
		ID:            input.PaymentID,
		UserID:        input.UserID,
		Amount:        input.Amount,
		Currency:      input.Currency,
		Status:        types.PaymentStatus(input.Status),
		CorrelationID: input.CorrelationID,
		Metadata:      input.Metadata,
	}

	resp, err := s.gateway.ProcessPayment(ctx, payment)
	if err != nil {
		s.logger.Error("Failed to process payment from Step Function", err, map[string]interface{}{
			"paymentId": payment.ID,
		})
		return &types.LambdaResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &types.LambdaResponse{
		Success: true,
		Data: map[string]interface{}{
			"externalId": resp.ExternalID,
			"status":     resp.Status,
			"message":    resp.Message,
		},
	}, nil
}

// CheckStepFunctionStatus checks payment status from Step Function input
func (s *PaymentAdapterService) CheckStepFunctionStatus(ctx context.Context, input *types.StepFunctionInput) (*types.LambdaResponse, error) {
	if input.ExternalID == "" {
		return &types.LambdaResponse{
			Success: false,
			Error:   "externalId is required",
		}, nil
	}

	resp, err := s.gateway.GetPaymentStatus(ctx, input.ExternalID)
	if err != nil {
		s.logger.Error("Failed to get payment status from Step Function", err, map[string]interface{}{
			"externalId": input.ExternalID,
		})
		return &types.LambdaResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &types.LambdaResponse{
		Success: true,
		Data: map[string]interface{}{
			"externalId": resp.ExternalID,
			"status":     resp.Status,
			"message":    resp.Message,
		},
	}, nil
}

// GetCircuitBreakerState returns the current state of the circuit breaker
func (s *PaymentAdapterService) GetCircuitBreakerState() string {
	return s.gateway.GetState()
}

// validateProcessPaymentRequest validates the payment processing request
func (s *PaymentAdapterService) validateProcessPaymentRequest(req *ProcessPaymentRequest) error {
	if req.PaymentID == "" {
		return fmt.Errorf("paymentID is required")
	}
	if req.UserID == "" {
		return fmt.Errorf("userID is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if req.Currency == "" || len(req.Currency) != 3 {
		return fmt.Errorf("invalid currency format")
	}
	return nil
}
