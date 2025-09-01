package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/draftea-coding-challenge/lambdas/wallet-service/internal/handler"
	"github.com/draftea-coding-challenge/lambdas/wallet-service/internal/repository"
	"github.com/draftea-coding-challenge/lambdas/wallet-service/internal/service"
	"github.com/draftea-coding-challenge/shared/observability"
)

func main() {
	// Initialize logger
	logger := observability.NewLogger(context.Background(), "wallet-service")

	// Initialize AWS session
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(getEnv("AWS_REGION", "us-east-1")),
	}))

	// Initialize DynamoDB client
	var dynamoClient *dynamodb.DynamoDB

	// Set endpoint for local development
	if endpoint := os.Getenv("DYNAMODB_ENDPOINT"); endpoint != "" {
		dynamoClient = dynamodb.New(sess, &aws.Config{
			Endpoint: aws.String(endpoint),
		})
		logger.Info("Using custom DynamoDB endpoint", map[string]interface{}{
			"endpoint": endpoint,
		})
	} else {
		dynamoClient = dynamodb.New(sess)
	}

	// Initialize repository
	walletsTable := getEnv("WALLETS_TABLE", "Wallets")
	eventsTable := getEnv("PAYMENT_EVENTS_TABLE", "PaymentEvents")

	repo := repository.NewWalletRepository(dynamoClient, walletsTable, eventsTable)

	// Initialize service
	walletService := service.NewWalletService(repo, logger)

	// Initialize handler
	h := handler.NewWalletHandler(walletService, logger)

	// Start Lambda handler
	lambda.Start(h.HandleRequest)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
