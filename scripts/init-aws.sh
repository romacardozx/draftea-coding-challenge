#!/bin/bash

echo "Initializing LocalStack AWS resources..."

# Wait for LocalStack to be ready
sleep 5

# Create DynamoDB tables
echo "Creating DynamoDB tables..."

# Payments table
awslocal dynamodb create-table \
  --table-name Payments \
  --attribute-definitions \
    AttributeName=ID,AttributeType=S \
    AttributeName=UserID,AttributeType=S \
  --key-schema AttributeName=ID,KeyType=HASH \
  --global-secondary-indexes \
    IndexName=UserIDIndex,Keys=["{AttributeName=UserID,KeyType=HASH}"],Projection="{ProjectionType=ALL}" \
  --billing-mode PAY_PER_REQUEST \
  2>/dev/null || echo "Payments table already exists"

# PaymentEvents table
awslocal dynamodb create-table \
  --table-name PaymentEvents \
  --attribute-definitions \
    AttributeName=PaymentID,AttributeType=S \
    AttributeName=Timestamp,AttributeType=N \
  --key-schema \
    AttributeName=PaymentID,KeyType=HASH \
    AttributeName=Timestamp,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  2>/dev/null || echo "PaymentEvents table already exists"

# Wallets table
awslocal dynamodb create-table \
  --table-name Wallets \
  --attribute-definitions \
    AttributeName=UserID,AttributeType=S \
  --key-schema AttributeName=UserID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  2>/dev/null || echo "Wallets table already exists"

# Invoices table
awslocal dynamodb create-table \
  --table-name Invoices \
  --attribute-definitions \
    AttributeName=ID,AttributeType=S \
    AttributeName=PaymentID,AttributeType=S \
  --key-schema AttributeName=ID,KeyType=HASH \
  --global-secondary-indexes \
    IndexName=PaymentIDIndex,Keys=["{AttributeName=PaymentID,KeyType=HASH}"],Projection="{ProjectionType=ALL}" \
  --billing-mode PAY_PER_REQUEST \
  2>/dev/null || echo "Invoices table already exists"

# Refunds table
awslocal dynamodb create-table \
  --table-name Refunds \
  --attribute-definitions \
    AttributeName=ID,AttributeType=S \
    AttributeName=PaymentID,AttributeType=S \
  --key-schema AttributeName=ID,KeyType=HASH \
  --global-secondary-indexes \
    IndexName=PaymentIDIndex,Keys=["{AttributeName=PaymentID,KeyType=HASH}"],Projection="{ProjectionType=ALL}" \
  --billing-mode PAY_PER_REQUEST \
  2>/dev/null || echo "Refunds table already exists"

# Create SQS queues
echo "Creating SQS queues..."

awslocal sqs create-queue --queue-name payment-requests 2>/dev/null || echo "payment-requests queue already exists"
awslocal sqs create-queue --queue-name payment-responses 2>/dev/null || echo "payment-responses queue already exists"
awslocal sqs create-queue --queue-name refund-requests 2>/dev/null || echo "refund-requests queue already exists"
awslocal sqs create-queue --queue-name wallet-debits 2>/dev/null || echo "wallet-debits queue already exists"
awslocal sqs create-queue --queue-name payment-dlq 2>/dev/null || echo "payment-dlq queue already exists"

# Create SNS topics
echo "Creating SNS topics..."

awslocal sns create-topic --name payment-notifications 2>/dev/null || echo "payment-notifications topic already exists"
awslocal sns create-topic --name refund-notifications 2>/dev/null || echo "refund-notifications topic already exists"

echo "LocalStack initialization complete!"
