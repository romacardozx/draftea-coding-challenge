#!/bin/bash

echo "Setting up LocalStack resources..."

# Create Payments table
aws dynamodb create-table \
  --table-name Payments \
  --attribute-definitions \
    AttributeName=ID,AttributeType=S \
    AttributeName=UserID,AttributeType=S \
  --key-schema \
    AttributeName=ID,KeyType=HASH \
  --global-secondary-indexes \
    'IndexName=UserIDIndex,KeySchema=[{AttributeName=UserID,KeyType=HASH}],Projection={ProjectionType=ALL}' \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url http://localhost:4566 \
  2>/dev/null && echo "✓ Payments table created" || echo "✗ Payments table already exists"

# Create Invoices table
aws dynamodb create-table \
  --table-name Invoices \
  --attribute-definitions \
    AttributeName=ID,AttributeType=S \
    AttributeName=PaymentID,AttributeType=S \
  --key-schema \
    AttributeName=ID,KeyType=HASH \
  --global-secondary-indexes \
    'IndexName=PaymentIDIndex,KeySchema=[{AttributeName=PaymentID,KeyType=HASH}],Projection={ProjectionType=ALL}' \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url http://localhost:4566 \
  2>/dev/null && echo "✓ Invoices table created" || echo "✗ Invoices table already exists"

# Create Refunds table
aws dynamodb create-table \
  --table-name Refunds \
  --attribute-definitions \
    AttributeName=ID,AttributeType=S \
    AttributeName=PaymentID,AttributeType=S \
  --key-schema \
    AttributeName=ID,KeyType=HASH \
  --global-secondary-indexes \
    'IndexName=PaymentIDIndex,KeySchema=[{AttributeName=PaymentID,KeyType=HASH}],Projection={ProjectionType=ALL}' \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url http://localhost:4566 \
  2>/dev/null && echo "✓ Refunds table created" || echo "✗ Refunds table already exists"

# Create SQS queues
aws sqs create-queue --queue-name payment-requests --endpoint-url http://localhost:4566 2>/dev/null && echo "✓ payment-requests queue created" || echo "✗ payment-requests queue already exists"
aws sqs create-queue --queue-name payment-responses --endpoint-url http://localhost:4566 2>/dev/null && echo "✓ payment-responses queue created" || echo "✗ payment-responses queue already exists"
aws sqs create-queue --queue-name refund-requests --endpoint-url http://localhost:4566 2>/dev/null && echo "✓ refund-requests queue created" || echo "✗ refund-requests queue already exists"
aws sqs create-queue --queue-name wallet-debits --endpoint-url http://localhost:4566 2>/dev/null && echo "✓ wallet-debits queue created" || echo "✗ wallet-debits queue already exists"
aws sqs create-queue --queue-name payment-dlq --endpoint-url http://localhost:4566 2>/dev/null && echo "✓ payment-dlq queue created" || echo "✗ payment-dlq queue already exists"

# Create SNS topics
aws sns create-topic --name payment-notifications --endpoint-url http://localhost:4566 2>/dev/null && echo "✓ payment-notifications topic created" || echo "✗ payment-notifications topic already exists"
aws sns create-topic --name refund-notifications --endpoint-url http://localhost:4566 2>/dev/null && echo "✓ refund-notifications topic created" || echo "✗ refund-notifications topic already exists"

echo ""
echo "Listing created resources:"
echo "------------------------"
echo "DynamoDB Tables:"
aws dynamodb list-tables --endpoint-url http://localhost:4566 --output table

echo ""
echo "SQS Queues:"
aws sqs list-queues --endpoint-url http://localhost:4566 --output table

echo ""
echo "SNS Topics:"
aws sns list-topics --endpoint-url http://localhost:4566 --output table
