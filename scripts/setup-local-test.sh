#!/bin/bash

# Setup script for local testing without Docker
# Uses AWS CLI with LocalStack endpoint or mocked responses

echo "Setting up local test environment..."

# Export AWS credentials for local testing
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1

# Check if LocalStack is available, otherwise use mock mode
if curl -s http://localhost:4566/health > /dev/null 2>&1; then
    echo "LocalStack is running, using it for AWS services"
    export AWS_ENDPOINT=http://localhost:4566
else
    echo "LocalStack not running, tests will use mocked AWS responses"
    export USE_MOCK_AWS=true
fi

# Check if mock-gateway is running
if ! curl -s http://localhost:3000/health > /dev/null 2>&1; then
    echo "Starting mock-gateway..."
    cd mock-gateway && go build -o mock-gateway && ./mock-gateway &
    sleep 2
    cd ..
fi

echo "Environment ready for testing!"
echo "Run tests with: cd tests && go test -v ./integration/..."
