package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/draftea-coding-challenge/lambdas/refund-service/internal/handler"
	"github.com/draftea-coding-challenge/lambdas/refund-service/internal/repository"
	"github.com/draftea-coding-challenge/lambdas/refund-service/internal/service"
	"github.com/draftea-coding-challenge/shared/observability"
)

func main() {
	// Initialize AWS session
	sess := session.Must(session.NewSession())
	db := dynamodb.New(sess)

	// Get table names from environment
	paymentsTable := os.Getenv("PAYMENTS_TABLE")
	if paymentsTable == "" {
		paymentsTable = "Payments"
	}
	walletsTable := os.Getenv("WALLETS_TABLE")
	if walletsTable == "" {
		walletsTable = "Wallets"
	}
	eventsTable := os.Getenv("EVENTS_TABLE")
	if eventsTable == "" {
		eventsTable = "PaymentEvents"
	}

	// Initialize logger
	logger := &observability.Logger{
		ServiceName: "refund-service",
		RequestID:   os.Getenv("AWS_REQUEST_ID"),
	}

	// Create repository
	refundRepo := repository.NewRefundRepository(db, paymentsTable, walletsTable, eventsTable)

	// Create service
	refundService := service.NewRefundService(refundRepo, logger)

	// Create handler
	refundHandler := handler.NewRefundHandler(refundService, logger)

	// Start Lambda
	lambda.Start(refundHandler.HandleRequest)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}