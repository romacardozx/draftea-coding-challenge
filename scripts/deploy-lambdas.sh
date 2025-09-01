#!/bin/bash

echo "Deploying Lambda functions to LocalStack..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Deploy each Lambda function
echo -e "${GREEN}Deploying payments-adapter...${NC}"
AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws lambda create-function \
  --function-name payments-adapter \
  --runtime provided.al2 \
  --role arn:aws:iam::000000000000:role/lambda-role \
  --handler bootstrap \
  --zip-file fileb://lambdas/payments-adapter/payments-adapter.zip \
  --environment Variables="{DYNAMODB_ENDPOINT=http://host.docker.internal:4566,SQS_ENDPOINT=http://host.docker.internal:4566,GATEWAY_URL=http://payment-mock-gateway:3000}" \
  --endpoint-url http://localhost:4566 \
  --region us-east-1 \
  2>/dev/null && echo "✓ payments-adapter deployed" || echo "✗ payments-adapter already exists"

echo -e "${GREEN}Deploying wallet-service...${NC}"
AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws lambda create-function \
  --function-name wallet-service \
  --runtime provided.al2 \
  --role arn:aws:iam::000000000000:role/lambda-role \
  --handler bootstrap \
  --zip-file fileb://lambdas/wallet-service/wallet-service.zip \
  --environment Variables="{DYNAMODB_ENDPOINT=http://host.docker.internal:4566,SQS_ENDPOINT=http://host.docker.internal:4566}" \
  --endpoint-url http://localhost:4566 \
  --region us-east-1 \
  2>/dev/null && echo "✓ wallet-service deployed" || echo "✗ wallet-service already exists"

echo -e "${GREEN}Deploying invoice-processor...${NC}"
AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws lambda create-function \
  --function-name invoice-processor \
  --runtime provided.al2 \
  --role arn:aws:iam::000000000000:role/lambda-role \
  --handler bootstrap \
  --zip-file fileb://lambdas/invoice-processor/invoice-processor.zip \
  --environment Variables="{DYNAMODB_ENDPOINT=http://host.docker.internal:4566,SQS_ENDPOINT=http://host.docker.internal:4566}" \
  --endpoint-url http://localhost:4566 \
  --region us-east-1 \
  2>/dev/null && echo "✓ invoice-processor deployed" || echo "✗ invoice-processor already exists"

echo -e "${GREEN}Deploying refund-service...${NC}"
AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws lambda create-function \
  --function-name refund-service \
  --runtime provided.al2 \
  --role arn:aws:iam::000000000000:role/lambda-role \
  --handler bootstrap \
  --zip-file fileb://lambdas/refund-service/refund-service.zip \
  --environment Variables="{DYNAMODB_ENDPOINT=http://host.docker.internal:4566,SQS_ENDPOINT=http://host.docker.internal:4566}" \
  --endpoint-url http://localhost:4566 \
  --region us-east-1 \
  2>/dev/null && echo "✓ refund-service deployed" || echo "✗ refund-service already exists"

# Create Step Functions state machine
echo -e "${GREEN}Creating Step Functions state machine...${NC}"
aws stepfunctions create-state-machine \
  --name PaymentProcessingStateMachine \
  --definition file://state-machine/stateMachine.json \
  --role-arn arn:aws:iam::000000000000:role/stepfunctions-role \
  --endpoint-url http://localhost:4566 \
  2>/dev/null && echo "✓ State machine created" || echo "✗ State machine already exists"

# List deployed functions
echo -e "\nDeployed Lambda functions:"
AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws lambda list-functions --endpoint-url http://localhost:4566 --region us-east-1 --query 'Functions[].FunctionName' --output table

echo ""
echo -e "${GREEN}Lambda deployment complete!${NC}"
