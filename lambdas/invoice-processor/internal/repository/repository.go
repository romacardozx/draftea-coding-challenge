package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"github.com/draftea-coding-challenge/shared/types"
)

type PaymentRepository struct {
	client        *dynamodb.DynamoDB
	paymentsTable string
	eventsTable   string
}

func NewPaymentRepository(client *dynamodb.DynamoDB, paymentsTable, eventsTable string) *PaymentRepository {
	return &PaymentRepository{
		client:        client,
		paymentsTable: paymentsTable,
		eventsTable:   eventsTable,
	}
}

// CreatePayment creates a new payment record
func (r *PaymentRepository) CreatePayment(ctx context.Context, payment *types.Payment) error {
	// Only generate ID if not provided
	if payment.ID == "" {
		payment.ID = uuid.New().String()
	}
	
	// Set status if not provided
	if payment.Status == "" {
		payment.Status = types.PaymentStatusPending
	}
	
	// Set timestamps if not provided
	if payment.CreatedAt.IsZero() {
		payment.CreatedAt = time.Now()
	}
	if payment.UpdatedAt.IsZero() {
		payment.UpdatedAt = time.Now()
	}
	
	// Create correlation ID if not provided
	if payment.CorrelationID == "" {
		payment.CorrelationID = uuid.New().String()
	}
	
	// Log payment before marshaling
	fmt.Printf("Payment before marshal: ID=%s, UserID=%s, Amount=%f\n", payment.ID, payment.UserID, payment.Amount)
	
	item, err := dynamodbattribute.MarshalMap(payment)
	if err != nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}
	
	// Log marshaled item
	fmt.Printf("Marshaled item: %+v\n", item)
	
	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.paymentsTable),
		Item:      item,
	}
	
	_, err = r.client.PutItemWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	
	// Record payment created event
	// Convert metadata from map[string]string to map[string]interface{}
	var eventMetadata map[string]interface{}
	if payment.Metadata != nil {
		eventMetadata = make(map[string]interface{})
		for k, v := range payment.Metadata {
			eventMetadata[k] = v
		}
	}
	
	event := &types.PaymentEvent{
		ID:            uuid.New().String(),
		PaymentID:     payment.ID,
		UserID:        payment.UserID,
		EventType:     "PAYMENT_CREATED",
		Amount:        payment.Amount,
		Status:        string(payment.Status),
		Metadata:      eventMetadata,
		CorrelationID: payment.CorrelationID,
		Timestamp:     time.Now(),
	}
	
	if err := r.recordEvent(ctx, event); err != nil {
		// Log error but don't fail the payment creation
		fmt.Printf("Warning: failed to record payment event: %v\n", err)
	}
	
	return nil
}

// UpdatePaymentStatus updates the status of a payment
func (r *PaymentRepository) UpdatePaymentStatus(ctx context.Context, paymentID string, status types.PaymentStatus, externalID string) error {
	updateExpr := "SET #status = :status, UpdatedAt = :updatedAt"
	exprAttrNames := map[string]*string{
		"#status": aws.String("Status"),
	}
	exprAttrValues := map[string]*dynamodb.AttributeValue{
		":status":    {S: aws.String(string(status))},
		":updatedAt": {S: aws.String(time.Now().Format(time.RFC3339))},
	}
	
	if externalID != "" {
		updateExpr += ", ExternalID = :externalID"
		exprAttrValues[":externalID"] = &dynamodb.AttributeValue{S: aws.String(externalID)}
	}
	
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.paymentsTable),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {S: aws.String(paymentID)},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
	}
	
	_, err := r.client.UpdateItemWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}
	
	return nil
}

// GetPayment retrieves a payment by ID
func (r *PaymentRepository) GetPayment(ctx context.Context, paymentID string) (*types.Payment, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.paymentsTable),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {S: aws.String(paymentID)},
		},
	}
	
	result, err := r.client.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	
	if result.Item == nil {
		return nil, fmt.Errorf("payment not found: %s", paymentID)
	}
	
	var payment types.Payment
	if err := dynamodbattribute.UnmarshalMap(result.Item, &payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment: %w", err)
	}
	
	return &payment, nil
}

// CheckIdempotency checks if a payment with the given idempotency key already exists
func (r *PaymentRepository) CheckIdempotency(ctx context.Context, idempotencyKey string) (*types.Payment, error) {
	// Query payments by idempotency key (would need a GSI in production)
	// For now, this is a simplified implementation
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(r.paymentsTable),
		FilterExpression: aws.String("Metadata.idempotencyKey = :key"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":key": {S: aws.String(idempotencyKey)},
		},
	}
	
	result, err := r.client.ScanWithContext(ctx, scanInput)
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}
	
	if len(result.Items) > 0 {
		var payment types.Payment
		if err := dynamodbattribute.UnmarshalMap(result.Items[0], &payment); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payment: %w", err)
		}
		return &payment, nil
	}
	
	return nil, nil
}

// recordEvent records a payment event
func (r *PaymentRepository) recordEvent(ctx context.Context, event *types.PaymentEvent) error {
	item, err := dynamodbattribute.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	// Add PaymentID and EventID for composite key
	item["PaymentID"] = &dynamodb.AttributeValue{S: aws.String(event.PaymentID)}
	item["EventID"] = &dynamodb.AttributeValue{S: aws.String(event.ID)}
	
	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.eventsTable),
		Item:      item,
	}
	
	_, err = r.client.PutItemWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to record event: %w", err)
	}
	
	return nil
}
