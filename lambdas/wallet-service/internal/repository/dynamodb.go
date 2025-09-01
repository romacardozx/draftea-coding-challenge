package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"github.com/draftea-coding-challenge/shared/types"
)

type WalletRepository struct {
	db              *dynamodb.DynamoDB
	walletsTable    string
	eventsTable     string
}

func NewWalletRepository(db *dynamodb.DynamoDB, walletsTable, eventsTable string) *WalletRepository {
	return &WalletRepository{
		db:           db,
		walletsTable: walletsTable,
		eventsTable:  eventsTable,
	}
}

// GetWallet retrieves wallet for a user, creates if not exists
func (r *WalletRepository) GetWallet(ctx context.Context, userID string) (*types.Wallet, error) {
	result, err := r.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.walletsTable),
		Key: map[string]*dynamodb.AttributeValue{
			"UserID": {
				S: aws.String(userID),
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Create wallet if doesn't exist
	if result.Item == nil {
		wallet := &types.Wallet{
			UserID:    userID,
			Balance:   1000.0, // Initial balance
			Currency:  "USD",
			Version:   0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := r.CreateWallet(ctx, wallet); err != nil {
			return nil, err
		}
		return wallet, nil
	}

	var wallet types.Wallet
	if err := dynamodbattribute.UnmarshalMap(result.Item, &wallet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet: %w", err)
	}

	return &wallet, nil
}

// CreateWallet creates a new wallet
func (r *WalletRepository) CreateWallet(ctx context.Context, wallet *types.Wallet) error {
	item, err := dynamodbattribute.MarshalMap(wallet)
	if err != nil {
		return fmt.Errorf("failed to marshal wallet: %w", err)
	}

	_, err = r.db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.walletsTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(UserID)"),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return fmt.Errorf("wallet already exists for user %s", wallet.UserID)
		}
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	return nil
}

// DebitWallet debits amount from wallet with optimistic locking
func (r *WalletRepository) DebitWallet(ctx context.Context, userID string, amount float64, paymentID string) (*types.WalletTransaction, error) {
	wallet, err := r.GetWallet(ctx, userID)
	if err != nil {
		return nil, err
	}

	if wallet.Balance < amount {
		return nil, fmt.Errorf("insufficient balance: available=%f, required=%f", wallet.Balance, amount)
	}

	newBalance := wallet.Balance - amount
	transaction := &types.WalletTransaction{
		ID:            uuid.New().String(),
		UserID:        userID,
		PaymentID:     paymentID,
		Type:          "DEBIT",
		Amount:        amount,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  newBalance,
		Timestamp:     time.Now(),
	}

	// Update wallet with optimistic locking
	_, err = r.db.UpdateItemWithContext(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.walletsTable),
		Key: map[string]*dynamodb.AttributeValue{
			"UserID": {
				S: aws.String(userID),
			},
		},
		UpdateExpression: aws.String("SET Balance = :balance, Version = :newVersion, UpdatedAt = :updatedAt"),
		ConditionExpression: aws.String("Version = :currentVersion AND Balance >= :amount"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":balance": {
				N: aws.String(fmt.Sprintf("%f", newBalance)),
			},
			":newVersion": {
				N: aws.String(fmt.Sprintf("%d", wallet.Version+1)),
			},
			":currentVersion": {
				N: aws.String(fmt.Sprintf("%d", wallet.Version)),
			},
			":amount": {
				N: aws.String(fmt.Sprintf("%f", amount)),
			},
			":updatedAt": {
				S: aws.String(time.Now().Format(time.RFC3339)),
			},
		},
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return nil, fmt.Errorf("concurrent update detected or insufficient balance")
		}
		return nil, fmt.Errorf("failed to debit wallet: %w", err)
	}

	// Record transaction event
	if err := r.recordEvent(ctx, transaction); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to record transaction event: %v\n", err)
	}

	return transaction, nil
}

// CreditWallet credits amount to wallet (for refunds)
func (r *WalletRepository) CreditWallet(ctx context.Context, userID string, amount float64, paymentID string) (*types.WalletTransaction, error) {
	wallet, err := r.GetWallet(ctx, userID)
	if err != nil {
		return nil, err
	}

	newBalance := wallet.Balance + amount
	transaction := &types.WalletTransaction{
		ID:            uuid.New().String(),
		UserID:        userID,
		PaymentID:     paymentID,
		Type:          "CREDIT",
		Amount:        amount,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  newBalance,
		Timestamp:     time.Now(),
	}

	// Update wallet with optimistic locking
	_, err = r.db.UpdateItemWithContext(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.walletsTable),
		Key: map[string]*dynamodb.AttributeValue{
			"UserID": {
				S: aws.String(userID),
			},
		},
		UpdateExpression: aws.String("SET Balance = :balance, Version = :newVersion, UpdatedAt = :updatedAt"),
		ConditionExpression: aws.String("Version = :currentVersion"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":balance": {
				N: aws.String(fmt.Sprintf("%f", newBalance)),
			},
			":newVersion": {
				N: aws.String(fmt.Sprintf("%d", wallet.Version+1)),
			},
			":currentVersion": {
				N: aws.String(fmt.Sprintf("%d", wallet.Version)),
			},
			":updatedAt": {
				S: aws.String(time.Now().Format(time.RFC3339)),
			},
		},
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return nil, fmt.Errorf("concurrent update detected")
		}
		return nil, fmt.Errorf("failed to credit wallet: %w", err)
	}

	// Record transaction event
	if err := r.recordEvent(ctx, transaction); err != nil {
		fmt.Printf("Failed to record transaction event: %v\n", err)
	}

	return transaction, nil
}

func (r *WalletRepository) recordEvent(ctx context.Context, transaction *types.WalletTransaction) error {
	event := &types.PaymentEvent{
		ID:        fmt.Sprintf("%s#%s", transaction.PaymentID, uuid.New().String()),
		PaymentID: transaction.PaymentID,
		UserID:    transaction.UserID,
		EventType: string(types.EventWalletDebited),
		Amount:    transaction.Amount,
		Status:    "SUCCESS",
		Metadata: map[string]interface{}{
			"transactionId": transaction.ID,
			"type":          transaction.Type,
			"balanceBefore": transaction.BalanceBefore,
			"balanceAfter":  transaction.BalanceAfter,
		},
		Timestamp: transaction.Timestamp,
	}

	if transaction.Type == "CREDIT" {
		event.EventType = string(types.EventWalletCredited)
	}

	item, err := dynamodbattribute.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = r.db.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.eventsTable),
		Item:      item,
	})

	return err
}