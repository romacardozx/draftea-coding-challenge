package repository

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/draftea-coding-challenge/shared/types"
)

// RefundRepository handles data access for refunds
type RefundRepository struct {
	db               *dynamodb.DynamoDB
	paymentsTable    string
	walletsTable     string
	eventsTable      string
}

// NewRefundRepository creates a new refund repository
func NewRefundRepository(db *dynamodb.DynamoDB, paymentsTable, walletsTable, eventsTable string) *RefundRepository {
	return &RefundRepository{
		db:            db,
		paymentsTable: paymentsTable,
		walletsTable:  walletsTable,
		eventsTable:   eventsTable,
	}
}

// GetPayment retrieves a payment by ID
func (r *RefundRepository) GetPayment(paymentID string) (*types.Payment, error) {
	result, err := r.db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(r.paymentsTable),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(paymentID),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("payment not found")
	}

	var payment types.Payment
	if err := dynamodbattribute.UnmarshalMap(result.Item, &payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment: %w", err)
	}

	return &payment, nil
}

// UpdatePayment updates a payment record
func (r *RefundRepository) UpdatePayment(payment *types.Payment) error {
	av, err := dynamodbattribute.MarshalMap(payment)
	if err != nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}

	_, err = r.db.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(r.paymentsTable),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	return nil
}

// CreditWallet adds funds to a user's wallet
func (r *RefundRepository) CreditWallet(userID string, amount float64) error {
	// First check if wallet exists, create if not
	_, err := r.db.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String(r.walletsTable),
		Key: map[string]*dynamodb.AttributeValue{
			"UserID": {
				S: aws.String(userID),
			},
		},
		UpdateExpression: aws.String("SET Balance = if_not_exists(Balance, :zero) + :amount"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":amount": {
				N: aws.String(fmt.Sprintf("%.2f", amount)),
			},
			":zero": {
				N: aws.String("0"),
			},
		},
		ReturnValues: aws.String("UPDATED_NEW"),
	})
	
	if err != nil {
		return fmt.Errorf("failed to credit wallet: %w", err)
	}

	return nil
}

// LogPaymentEvent logs a payment event
func (r *RefundRepository) LogPaymentEvent(event *types.PaymentEvent) error {
	av, err := dynamodbattribute.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal payment event: %w", err)
	}

	_, err = r.db.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(r.eventsTable),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to log payment event: %w", err)
	}

	return nil
}

// GetWalletBalance retrieves the current balance of a wallet
func (r *RefundRepository) GetWalletBalance(userID string) (float64, error) {
	result, err := r.db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(r.walletsTable),
		Key: map[string]*dynamodb.AttributeValue{
			"UserID": {
				S: aws.String(userID),
			},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get wallet: %w", err)
	}

	if result.Item == nil {
		// Wallet doesn't exist, return 0
		return 0, nil
	}

	var wallet types.Wallet
	if err := dynamodbattribute.UnmarshalMap(result.Item, &wallet); err != nil {
		return 0, fmt.Errorf("failed to unmarshal wallet: %w", err)
	}

	return wallet.Balance, nil
}
