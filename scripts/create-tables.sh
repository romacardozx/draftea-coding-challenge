#!/bin/bash

echo "Creating DynamoDB tables..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# AWS credentials for LocalStack
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
ENDPOINT_URL="http://localhost:4566"

# Create Payments table
echo -e "${GREEN}Creating Payments table...${NC}"
aws dynamodb create-table \
  --table-name Payments \
  --attribute-definitions \
    AttributeName=ID,AttributeType=S \
    AttributeName=UserID,AttributeType=S \
    AttributeName=Status,AttributeType=S \
  --key-schema AttributeName=ID,KeyType=HASH \
  --global-secondary-indexes \
    '[{"IndexName":"UserIndex","KeySchema":[{"AttributeName":"UserID","KeyType":"HASH"}],"Projection":{"ProjectionType":"ALL"}},{"IndexName":"StatusIndex","KeySchema":[{"AttributeName":"Status","KeyType":"HASH"}],"Projection":{"ProjectionType":"ALL"}}]' \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url $ENDPOINT_URL \
  --region $AWS_DEFAULT_REGION \
  2>/dev/null && echo "✓ Payments table created" || echo "✗ Payments table already exists"

# Create Wallets table
echo -e "${GREEN}Creating Wallets table...${NC}"
aws dynamodb create-table \
  --table-name Wallets \
  --attribute-definitions \
    AttributeName=UserID,AttributeType=S \
  --key-schema AttributeName=UserID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url $ENDPOINT_URL \
  --region $AWS_DEFAULT_REGION \
  2>/dev/null && echo "✓ Wallets table created" || echo "✗ Wallets table already exists"

# Create PaymentEvents table
echo -e "${GREEN}Creating PaymentEvents table...${NC}"
aws dynamodb create-table \
  --table-name PaymentEvents \
  --attribute-definitions \
    AttributeName=PaymentID,AttributeType=S \
    AttributeName=Timestamp,AttributeType=S \
  --key-schema \
    AttributeName=PaymentID,KeyType=HASH \
    AttributeName=Timestamp,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url $ENDPOINT_URL \
  --region $AWS_DEFAULT_REGION \
  2>/dev/null && echo "✓ PaymentEvents table created" || echo "✗ PaymentEvents table already exists"

# Create WalletTransactions table (if needed)
echo -e "${GREEN}Creating WalletTransactions table...${NC}"
aws dynamodb create-table \
  --table-name WalletTransactions \
  --attribute-definitions \
    AttributeName=ID,AttributeType=S \
    AttributeName=UserID,AttributeType=S \
    AttributeName=Timestamp,AttributeType=S \
  --key-schema AttributeName=ID,KeyType=HASH \
  --global-secondary-indexes \
    '[{"IndexName":"UserTransactionsIndex","KeySchema":[{"AttributeName":"UserID","KeyType":"HASH"},{"AttributeName":"Timestamp","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"}}]' \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url $ENDPOINT_URL \
  --region $AWS_DEFAULT_REGION \
  2>/dev/null && echo "✓ WalletTransactions table created" || echo "✗ WalletTransactions table already exists"

# Seed initial wallet data
echo -e "${GREEN}Seeding initial wallet data...${NC}"
aws dynamodb put-item \
  --table-name Wallets \
  --item '{"UserID": {"S": "user_test_001"}, "Balance": {"N": "1000"}, "Currency": {"S": "USD"}, "Version": {"N": "1"}}' \
  --endpoint-url $ENDPOINT_URL \
  --region $AWS_DEFAULT_REGION \
  2>/dev/null && echo "✓ Initial wallet created for user_test_001" || echo "✗ Wallet already exists"

# List all tables
echo -e "\n${GREEN}DynamoDB tables created:${NC}"
aws dynamodb list-tables --endpoint-url $ENDPOINT_URL --region $AWS_DEFAULT_REGION --query 'TableNames' --output table

echo -e "${GREEN}Table creation complete!${NC}"