#!/bin/bash

echo "🧪 Running End-to-End Payment Flow Tests"
echo "========================================="

# Set AWS credentials for local testing
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_SESSION_TOKEN=test
export AWS_REGION=us-east-1

# Check if DynamoDB is running
echo "Checking DynamoDB..."
if ! nc -z localhost 8000 2>/dev/null; then
    echo "❌ DynamoDB is not running. Checking Docker..."
    if ! docker info > /dev/null 2>&1; then
        echo "❌ Docker is not running. Please start Docker Desktop and try again."
        exit 1
    fi
    echo "Starting DynamoDB Local..."
    docker-compose up -d dynamodb
    sleep 5
fi

# Create tables
echo "Setting up database tables..."
./scripts/create-tables.sh

# Start mock payment gateway
echo "Starting mock payment gateway..."
cd mock-gateway
go run main.go &
GATEWAY_PID=$!
cd ..
sleep 2

# Check if gateway is running
curl -s http://localhost:3000/health > /dev/null
if [ $? -ne 0 ]; then
    echo "❌ Mock gateway failed to start"
    exit 1
fi
echo "✅ Mock gateway running on port 3000"

# Run integration tests
echo "Running integration tests..."
cd tests/integration
go test -v -timeout 30s

TEST_RESULT=$?

# Cleanup
echo "Cleaning up..."
kill $GATEWAY_PID 2>/dev/null

if [ $TEST_RESULT -eq 0 ]; then
    echo "✅ All E2E tests passed!"
else
    echo "❌ Some tests failed"
    exit $TEST_RESULT
fi