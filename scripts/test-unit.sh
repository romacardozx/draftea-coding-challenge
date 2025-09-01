#!/bin/bash

echo "🧪 Running Unit Tests for All Services"
echo "======================================"

# Set AWS credentials for local testing
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_SESSION_TOKEN=test
export AWS_REGION=us-east-1

# Test wallet-service
echo "Testing wallet-service..."
cd lambdas/wallet-service
go test ./... -v
cd ../..

# Test invoice-processor
echo "Testing invoice-processor..."
cd lambdas/invoice-processor
go test ./... -v
cd ../..

# Test payments-adapter
echo "Testing payments-adapter..."
cd lambdas/payments-adapter
go test ./... -v
cd ../..

# Test refund-service
echo "Testing refund-service..."
cd lambdas/refund-service
go test ./... -v
cd ../..

echo "✅ Unit tests completed"
