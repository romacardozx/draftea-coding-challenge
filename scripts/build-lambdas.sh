#!/bin/bash

echo "Building Lambda functions..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Build each Lambda function
LAMBDAS=("payments-adapter" "wallet-service" "invoice-processor" "refund-service")

for lambda in "${LAMBDAS[@]}"; do
    echo -e "${GREEN}Building $lambda...${NC}"
    
    cd lambdas/$lambda
    
    # Build for Linux/AMD64 (Lambda runtime) with CGO disabled
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -ldflags="-s -w" -o bootstrap cmd/main.go
    
    # Create deployment package
    zip -j $lambda.zip bootstrap
    
    cd ../..
    
    echo "✓ $lambda built and packaged"
done

echo ""
echo -e "${GREEN}All Lambda functions built successfully!${NC}"
echo "Deployment packages created:"
ls -la lambdas/*/*.zip
