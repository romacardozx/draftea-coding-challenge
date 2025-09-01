#!/bin/bash

echo "Configuring event source mappings for Lambda functions..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Get SQS queue ARNs
PAYMENT_QUEUE_ARN=$(aws sqs get-queue-attributes \
  --queue-url http://localhost:4566/000000000000/payment-requests \
  --attribute-names QueueArn \
  --endpoint-url http://localhost:4566 \
  --query 'Attributes.QueueArn' \
  --output text 2>/dev/null)

WALLET_QUEUE_ARN=$(aws sqs get-queue-attributes \
  --queue-url http://localhost:4566/000000000000/wallet-debits \
  --attribute-names QueueArn \
  --endpoint-url http://localhost:4566 \
  --query 'Attributes.QueueArn' \
  --output text 2>/dev/null)

REFUND_QUEUE_ARN=$(aws sqs get-queue-attributes \
  --queue-url http://localhost:4566/000000000000/refund-requests \
  --attribute-names QueueArn \
  --endpoint-url http://localhost:4566 \
  --query 'Attributes.QueueArn' \
  --output text 2>/dev/null)

# Create event source mappings
echo -e "${GREEN}Connecting payment-requests queue to payments-adapter Lambda...${NC}"
aws lambda create-event-source-mapping \
  --function-name payments-adapter \
  --event-source-arn "$PAYMENT_QUEUE_ARN" \
  --batch-size 1 \
  --endpoint-url http://localhost:4566 \
  2>/dev/null && echo "✓ payments-adapter connected" || echo "✗ payments-adapter mapping already exists"

echo -e "${GREEN}Connecting wallet-debits queue to wallet-service Lambda...${NC}"
aws lambda create-event-source-mapping \
  --function-name wallet-service \
  --event-source-arn "$WALLET_QUEUE_ARN" \
  --batch-size 1 \
  --endpoint-url http://localhost:4566 \
  2>/dev/null && echo "✓ wallet-service connected" || echo "✗ wallet-service mapping already exists"

echo -e "${GREEN}Connecting refund-requests queue to refund-service Lambda...${NC}"
aws lambda create-event-source-mapping \
  --function-name refund-service \
  --event-source-arn "$REFUND_QUEUE_ARN" \
  --batch-size 1 \
  --endpoint-url http://localhost:4566 \
  2>/dev/null && echo "✓ refund-service connected" || echo "✗ refund-service mapping already exists"

# List event source mappings
echo ""
echo "Active event source mappings:"
aws lambda list-event-source-mappings --endpoint-url http://localhost:4566 --output table

echo ""
echo -e "${GREEN}Event source configuration complete!${NC}"
