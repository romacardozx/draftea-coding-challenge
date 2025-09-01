package service

import (
	"context"
	"testing"

	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/stretchr/testify/assert"
)

func TestCreatePayment_ValidationError(t *testing.T) {
	// Test with invalid user ID
	req := CreatePaymentRequest{
		UserID:   "",
		Amount:   100.00,
		Currency: "USD",
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewPaymentService(nil, logger)
	
	_, err := service.CreatePayment(context.Background(), req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is required")
}

func TestCreatePayment_InvalidAmount(t *testing.T) {
	// Test with invalid amount
	req := CreatePaymentRequest{
		UserID:   "user123",
		Amount:   -100.00,
		Currency: "USD",
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewPaymentService(nil, logger)
	
	_, err := service.CreatePayment(context.Background(), req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
}

func TestCreatePayment_MissingCurrency(t *testing.T) {
	// Test with missing currency
	req := CreatePaymentRequest{
		UserID:   "user123",
		Amount:   100.00,
		Currency: "",
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewPaymentService(nil, logger)
	
	_, err := service.CreatePayment(context.Background(), req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid currency format")
}

func TestGetPayment_InvalidID(t *testing.T) {
	logger := observability.NewLogger(context.Background(), "test")
	service := NewPaymentService(nil, logger)
	
	result, err := service.GetPayment(context.Background(), "")
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "payment ID is required")
}

func TestUpdatePaymentStatus_InvalidPaymentID(t *testing.T) {
	// Skip this test as it requires a valid repository
	t.Skip("Requires repository implementation")
}
