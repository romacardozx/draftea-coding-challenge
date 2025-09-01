package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/draftea-coding-challenge/lambdas/invoice-processor/internal/handler"
	"github.com/draftea-coding-challenge/lambdas/invoice-processor/internal/repository"
	"github.com/draftea-coding-challenge/lambdas/invoice-processor/internal/service"
	"github.com/draftea-coding-challenge/shared/observability"
)

func main() {
	// Initialize logger
	logger := observability.NewLogger(context.Background(), "invoice-processor")

	// Initialize AWS session
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(getEnv("AWS_REGION", "us-east-1")),
	}))

	// Create DynamoDB client config
	dynamoConfig := &aws.Config{}

	// Configure local endpoint if provided
	if endpoint := os.Getenv("DYNAMODB_ENDPOINT"); endpoint != "" {
		dynamoConfig.Endpoint = aws.String(endpoint)
		logger.Info("Using custom DynamoDB endpoint", map[string]interface{}{
			"endpoint": endpoint,
		})
	}

	// Create DynamoDB client with config
	dynamoClient := dynamodb.New(sess, dynamoConfig)

	// Initialize repository
	paymentsTable := getEnv("PAYMENTS_TABLE", "Payments")
	eventsTable := getEnv("PAYMENT_EVENTS_TABLE", "PaymentEvents")

	repo := repository.NewPaymentRepository(dynamoClient, paymentsTable, eventsTable)

	// Initialize service
	paymentService := service.NewPaymentService(repo, logger)

	// Initialize handler
	h := handler.NewInvoiceHandler(paymentService, logger)

	// Start Lambda handler
	lambda.Start(h.HandleRequest)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
