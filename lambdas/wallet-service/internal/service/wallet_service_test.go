package service

import (
	"context"
	"testing"

	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/stretchr/testify/assert"
)

func TestDebitWallet_ValidationError(t *testing.T) {
	// Test with invalid user ID
	req := DebitRequest{
		UserID: "",
		Amount: 100.00,
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewWalletService(nil, logger)
	
	_, err := service.DebitWallet(context.Background(), req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "userID is required")
}

func TestDebitWallet_InvalidAmount(t *testing.T) {
	// Test with invalid amount
	req := DebitRequest{
		UserID: "user123",
		Amount: -100.00,
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewWalletService(nil, logger)
	
	_, err := service.DebitWallet(context.Background(), req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
}

func TestCreditWallet_ValidationError(t *testing.T) {
	// Test with invalid user ID
	req := CreditRequest{
		UserID: "",
		Amount: 100.00,
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewWalletService(nil, logger)
	
	_, err := service.CreditWallet(context.Background(), req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "userID is required")
}

func TestCreditWallet_InvalidAmount(t *testing.T) {
	// Test with invalid amount
	req := CreditRequest{
		UserID: "user123",
		Amount: -100.00,
	}

	logger := observability.NewLogger(context.Background(), "test")
	service := NewWalletService(nil, logger)
	
	_, err := service.CreditWallet(context.Background(), req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
}
