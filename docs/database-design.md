# Database Design - Payment Saga System

## DynamoDB Tables

### 1. Wallets Table
```json
{
  "TableName": "Wallets",
  "PartitionKey": "userId",
  "Attributes": {
    "userId": "user-123",
    "balance": 1000.00,
    "currency": "USD",
    "version": 1,
    "updatedAt": "2024-01-01T10:00:00Z",
    "createdAt": "2024-01-01T09:00:00Z"
  }
}
```

### 2. PaymentEvents Table (Event Sourcing)
```json
{
  "TableName": "PaymentEvents",
  "PartitionKey": "paymentId",
  "SortKey": "timestamp",
  "GSI1": {
    "PartitionKey": "userId",
    "SortKey": "timestamp"
  },
  "Attributes": {
    "paymentId": "pay-789",
    "timestamp": "2024-01-01T10:00:00.123Z",
    "userId": "user-123",
    "eventType": "PAYMENT_INITIATED|WALLET_DEBITED|PAYMENT_PROCESSED|REFUND_INITIATED",
    "amount": 100.50,
    "status": "SUCCESS|FAILED|PENDING",
    "metadata": {},
    "correlationId": "exec-456"
  }
}
```

### 3. CircuitBreakerState Table
```json
{
  "TableName": "CircuitBreakerState",
  "PartitionKey": "serviceName",
  "TTL": "resetTime",
  "Attributes": {
    "serviceName": "payment-gateway",
    "state": "CLOSED|OPEN|HALF_OPEN",
    "failureCount": 0,
    "successCount": 0,
    "lastFailureTime": "2024-01-01T10:00:00Z",
    "lastSuccessTime": "2024-01-01T09:50:00Z",
    "resetTime": 1704110400
  }
}
```

### 4. IdempotencyKeys Table
```json
{
  "TableName": "IdempotencyKeys",
  "PartitionKey": "idempotencyKey",
  "TTL": "expirationTime",
  "Attributes": {
    "idempotencyKey": "user-123:pay-789",
    "paymentId": "pay-789",
    "status": "PROCESSING|COMPLETED",
    "result": {},
    "createdAt": "2024-01-01T10:00:00Z",
    "expirationTime": 1704196800
  }
}
```

### 5. Metrics Table (Time Series)
```json
{
  "TableName": "Metrics",
  "PartitionKey": "metricType#date",
  "SortKey": "timestamp",
  "Attributes": {
    "metricType#date": "PAYMENT_LATENCY#2024-01-01",
    "timestamp": "2024-01-01T10:00:00.123Z",
    "value": 234.56,
    "dimensions": {
      "service": "payments-adapter",
      "operation": "process_payment",
      "status": "success"
    }
  }
}
```

## Access Patterns

1. **Get Wallet Balance**: Query by userId
2. **Update Wallet Balance**: Conditional update with version check
3. **Get Payment History**: Query PaymentEvents GSI1 by userId
4. **Get Payment Status**: Query PaymentEvents by paymentId
5. **Check Circuit Breaker**: Get item by serviceName
6. **Idempotency Check**: Get item by idempotencyKey
7. **Metrics Aggregation**: Query by metricType#date range

## Consistency Guarantees

- **Optimistic Locking**: Version field in Wallets
- **Conditional Writes**: Prevent negative balances
- **Event Sourcing**: Complete audit trail
- **TTL**: Auto-cleanup of old data
