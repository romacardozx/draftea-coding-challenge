package service

import (
	"context"
	"testing"

	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/stretchr/testify/assert"
)

func TestProcessPayment_ValidationError(t *testing.T) {
	// Test with invalid payment ID
	req := &ProcessPaymentRequest{
		PaymentID: "",
		UserID:    "user123",
		Amount:    100.00,
		Currency:  "USD",
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewPaymentAdapterService(nil, logger)
	
	result, err := service.ProcessPayment(context.Background(), req)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "paymentID is required")
}

func TestProcessPayment_InvalidAmount(t *testing.T) {
	// Test with invalid amount
	req := &ProcessPaymentRequest{
		PaymentID: "pay123",
		UserID:    "user123",
		Amount:    -100.00,
		Currency:  "USD",
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewPaymentAdapterService(nil, logger)
	
	result, err := service.ProcessPayment(context.Background(), req)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
}

func TestProcessPayment_MissingCurrency(t *testing.T) {
	// Test with missing currency
	req := &ProcessPaymentRequest{
		PaymentID: "pay123",
		UserID:    "user123",
		Amount:    100.00,
		Currency:  "",
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewPaymentAdapterService(nil, logger)
	
	result, err := service.ProcessPayment(context.Background(), req)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid currency format")
}

func TestProcessPayment_MissingUserID(t *testing.T) {
	// Test with missing user ID
	req := &ProcessPaymentRequest{
		PaymentID: "pay123",
		UserID:    "",
		Amount:    100.00,
		Currency:  "USD",
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewPaymentAdapterService(nil, logger)
	
	result, err := service.ProcessPayment(context.Background(), req)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "userID is required")
}
