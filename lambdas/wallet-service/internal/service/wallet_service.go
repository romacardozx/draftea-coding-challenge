package service

import (
	"context"
	"fmt"

	"github.com/draftea-coding-challenge/lambdas/wallet-service/internal/repository"
	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/draftea-coding-challenge/shared/types"
)

// WalletService handles business logic for wallets
type WalletService struct {
	repo   *repository.WalletRepository
	logger *observability.Logger
}

// NewWalletService creates a new wallet service
func NewWalletService(repo *repository.WalletRepository, logger *observability.Logger) *WalletService {
	return &WalletService{
		repo:   repo,
		logger: logger,
	}
}

// DebitRequest represents a wallet debit request
type DebitRequest struct {
	UserID        string `json:"userId"`
	Amount        float64 `json:"amount"`
	PaymentID     string `json:"paymentId"`
	CorrelationID string `json:"correlationId"`
}

// CreditRequest represents a wallet credit request
type CreditRequest struct {
	UserID        string `json:"userId"`
	Amount        float64 `json:"amount"`
	PaymentID     string `json:"paymentId"`
	CorrelationID string `json:"correlationId"`
	RefundReason  string `json:"refundReason,omitempty"`
}

// DebitWallet debits amount from user's wallet
func (s *WalletService) DebitWallet(ctx context.Context, req DebitRequest) (*types.Wallet, error) {
	// Validate request
	if err := s.validateDebitRequest(req); err != nil {
		return nil, err
	}

	// Get or create wallet
	wallet, err := s.repo.GetWallet(ctx, req.UserID)
	if err != nil {
		// Create wallet if not exists
		wallet = &types.Wallet{
			UserID:  req.UserID,
			Balance: 0,
			Version: 0,
		}
		if err := s.repo.CreateWallet(ctx, wallet); err != nil {
			s.logger.Error("Failed to create wallet", err, map[string]interface{}{
				"userId": req.UserID,
			})
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
	}

	// Check balance
	if wallet.Balance < req.Amount {
		return nil, fmt.Errorf("insufficient balance: available %f, required %f", wallet.Balance, req.Amount)
	}

	// Perform debit
	_, err = s.repo.DebitWallet(ctx, req.UserID, req.Amount, req.PaymentID)
	if err != nil {
		s.logger.Error("Failed to debit wallet", err, map[string]interface{}{
			"userId":    req.UserID,
			"amount":    req.Amount,
			"paymentId": req.PaymentID,
		})
		return nil, err
	}

	// Get updated wallet
	updatedWallet, err := s.repo.GetWallet(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated wallet: %w", err)
	}

	s.logger.Info("Wallet debited successfully", map[string]interface{}{
		"userId":     req.UserID,
		"amount":     req.Amount,
		"newBalance": updatedWallet.Balance,
		"paymentId":  req.PaymentID,
	})

	return updatedWallet, nil
}

// CreditWallet credits amount to user's wallet
func (s *WalletService) CreditWallet(ctx context.Context, req CreditRequest) (*types.Wallet, error) {
	// Validate request
	if err := s.validateCreditRequest(req); err != nil {
		return nil, err
	}

	// Get or create wallet
	wallet, err := s.repo.GetWallet(ctx, req.UserID)
	if err != nil {
		// Create wallet if not exists
		wallet = &types.Wallet{
			UserID:  req.UserID,
			Balance: 0,
			Version: 0,
		}
		if err := s.repo.CreateWallet(ctx, wallet); err != nil {
			s.logger.Error("Failed to create wallet", err, map[string]interface{}{
				"userId": req.UserID,
			})
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
	}

	// Perform credit
	_, err = s.repo.CreditWallet(ctx, req.UserID, req.Amount, req.PaymentID)
	if err != nil {
		s.logger.Error("Failed to credit wallet", err, map[string]interface{}{
			"userId":    req.UserID,
			"amount":    req.Amount,
			"paymentId": req.PaymentID,
		})
		return nil, err
	}

	// Get updated wallet
	updatedWallet, err := s.repo.GetWallet(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated wallet: %w", err)
	}

	s.logger.Info("Wallet credited successfully", map[string]interface{}{
		"userId":     req.UserID,
		"amount":     req.Amount,
		"newBalance": updatedWallet.Balance,
		"paymentId":  req.PaymentID,
		"reason":     req.RefundReason,
	})

	return updatedWallet, nil
}

// GetBalance retrieves wallet balance for a user
func (s *WalletService) GetBalance(ctx context.Context, userID string) (*types.Wallet, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	wallet, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		// Create wallet if not exists
		wallet = &types.Wallet{
			UserID:  userID,
			Balance: 0,
			Version: 0,
		}
		if err := s.repo.CreateWallet(ctx, wallet); err != nil {
			s.logger.Error("Failed to create wallet", err, map[string]interface{}{
				"userId": userID,
			})
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
	}

	return wallet, nil
}

// validateDebitRequest validates debit request
func (s *WalletService) validateDebitRequest(req DebitRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("userID is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if req.PaymentID == "" {
		return fmt.Errorf("paymentID is required")
	}
	return nil
}

// validateCreditRequest validates credit request
func (s *WalletService) validateCreditRequest(req CreditRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("userID is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if req.PaymentID == "" {
		return fmt.Errorf("paymentID is required")
	}
	return nil
}
