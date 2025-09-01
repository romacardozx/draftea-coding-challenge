package service

import (
	"context"
	"testing"

	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/stretchr/testify/assert"
)

func TestProcessRefund_ValidationError(t *testing.T) {
	// Test with invalid payment ID
	req := &RefundRequest{
		PaymentID: "",
		Amount:    50.00,
		Reason:    "Customer request",
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewRefundService(nil, logger)
	
	_, err := service.ProcessRefund(context.Background(), req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment_id is required")
}

func TestProcessRefund_InvalidAmount(t *testing.T) {
	// Test with invalid refund amount
	req := &RefundRequest{
		PaymentID: "pay123",
		Amount:    -50.00,
		Reason:    "Customer request",
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewRefundService(nil, logger)
	
	_, err := service.ProcessRefund(context.Background(), req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
}

func TestProcessRefund_MissingReason(t *testing.T) {
	// Test with missing reason
	req := &RefundRequest{
		PaymentID: "pay123",
		Amount:    50.00,
		Reason:    "",
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewRefundService(nil, logger)
	
	_, err := service.ProcessRefund(context.Background(), req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reason is required")
}
