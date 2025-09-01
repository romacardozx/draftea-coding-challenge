package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/draftea-coding-challenge/lambdas/payments-adapter/internal/gateway"
	"github.com/draftea-coding-challenge/lambdas/payments-adapter/internal/handler"
	"github.com/draftea-coding-challenge/lambdas/payments-adapter/internal/service"
	"github.com/draftea-coding-challenge/shared/observability"
)

func main() {
	// Initialize logger
	logger := observability.NewLogger(context.Background(), "payments-adapter")
	
	// Get gateway configuration from environment
	gatewayURL := getEnv("GATEWAY_URL", "http://localhost:3000")
	gatewayAPIKey := getEnv("GATEWAY_API_KEY", "test-api-key")
	
	// Create gateway client
	gatewayClient := gateway.NewClient(gatewayURL, gatewayAPIKey)
	
	// Create service layer
	paymentService := service.NewPaymentAdapterService(gatewayClient, logger)
	
	// Create handler
	h := handler.NewPaymentAdapterHandler(paymentService, logger)
	
	// Start Lambda handler
	lambda.Start(h.HandleRequest)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}