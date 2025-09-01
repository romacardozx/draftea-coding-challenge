#!/bin/bash

echo "🚀 Deploying Payment Processing System Locally"
echo "=============================================="

# Set environment variables
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_SESSION_TOKEN=test
export AWS_REGION=us-east-1

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker Desktop and try again."
    exit 1
fi

# Start DynamoDB Local
echo "Starting DynamoDB Local..."
docker-compose up -d dynamodb
sleep 3

# Create tables
echo "Creating DynamoDB tables..."
./scripts/create-tables.sh

# Build all Lambda functions
echo "Building Lambda functions..."
for service in wallet-service invoice-processor payments-adapter refund-service; do
    echo "  Building $service..."
    cd lambdas/$service
    GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/main.go
    cd ../..
done

# Start mock payment gateway
echo "Starting mock payment gateway..."
cd mock-gateway
go run main.go &
GATEWAY_PID=$!
cd ..
echo "Mock gateway running with PID: $GATEWAY_PID"

# Start SAM local API
echo "Starting SAM Local API..."
sam local start-api \
    --env-vars env.json \
    --docker-network bridge \
    --host 0.0.0.0 \
    --port 3001 &
SAM_PID=$!

echo ""
echo "✅ Services are starting up..."
echo "   - DynamoDB Local: http://localhost:8000"
echo "   - Mock Payment Gateway: http://localhost:3000"
echo "   - SAM Local API: http://localhost:3001"
echo ""
echo "Press Ctrl+C to stop all services"

# Wait for user interrupt
trap "echo 'Shutting down...'; kill $GATEWAY_PID $SAM_PID; docker-compose down; exit" INT
wait
