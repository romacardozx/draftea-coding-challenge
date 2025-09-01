# Observability Dashboard - Payment Saga System

## 📊 Key Metrics & KPIs

### Business Metrics
- **Payment Success Rate**: % of successful payments
- **Average Payment Processing Time**: P50, P95, P99 latencies
- **Daily Transaction Volume**: Number and value
- **Refund Rate**: % of payments refunded
- **Wallet Balance Accuracy**: Reconciliation metrics

### Technical Metrics
- **Lambda Cold Start Rate**: % of cold starts per function
- **Circuit Breaker State**: Open/Closed/Half-Open states
- **Retry Success Rate**: % of successful retries
- **Error Rate by Type**: Validation, Gateway, Internal errors
- **DynamoDB Throttling**: Read/Write capacity consumption

## 🔍 CloudWatch Insights Queries

### 1. Payment Success Rate (Last Hour)
```sql
fields @timestamp, service, status
| filter metricType = "BUSINESS_METRIC"
| filter metricName = "PAYMENT_PROCESSED"
| stats count(*) by status
| sort @timestamp desc
```

### 2. P95 Latency by Service
```sql
fields @timestamp, service, latencyMs
| filter metricType = "PERFORMANCE_METRIC"
| stats pct(latencyMs, 95) as p95_latency by service
| sort p95_latency desc
```

### 3. Circuit Breaker Events
```sql
fields @timestamp, circuitBreakerState, targetService
| filter metricType = "RESILIENCE_METRIC"
| sort @timestamp desc
| limit 20
```

### 4. Error Analysis
```sql
fields @timestamp, service, error, message
| filter level = "ERROR"
| stats count(*) as error_count by service, error
| sort error_count desc
```

### 5. Distributed Trace Analysis
```sql
fields @timestamp, traceId, service, duration
| filter traceId = "1-5e1b4f87-1234567890abcdef"
| sort @timestamp asc
```

## 🎯 Alerting Rules

### Critical Alerts
1. **Payment Success Rate < 95%** (5 min window)
2. **Circuit Breaker OPEN** for > 2 minutes
3. **Wallet Balance Negative** 
4. **DynamoDB Throttling** > 10 events/min

### Warning Alerts
1. **P95 Latency > 3 seconds**
2. **Error Rate > 1%** (5 min window)
3. **Cold Start Rate > 20%**
4. **Retry Rate > 5%**

## 📈 Custom CloudWatch Dashboard JSON

```json
{
  "widgets": [
    {
      "type": "metric",
      "properties": {
        "metrics": [
          ["PaymentSaga", "PaymentSuccess", {"stat": "Sum"}],
          [".", "PaymentFailed", {"stat": "Sum"}]
        ],
        "period": 300,
        "stat": "Average",
        "region": "us-east-1",
        "title": "Payment Success Rate"
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          ["AWS/Lambda", "Duration", {"stat": "Average"}],
          [".", ".", {"stat": "p95"}],
          [".", ".", {"stat": "p99"}]
        ],
        "period": 60,
        "stat": "Average",
        "region": "us-east-1",
        "title": "Lambda Performance"
      }
    },
    {
      "type": "log",
      "properties": {
        "query": "SOURCE '/aws/lambda/payment-saga'\n| fields @timestamp, level, message\n| filter level = 'ERROR'\n| sort @timestamp desc\n| limit 20",
        "region": "us-east-1",
        "title": "Recent Errors"
      }
    }
  ]
}
```

## 🔄 X-Ray Service Map

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│ API Gateway │────▶│ Step Function│────▶│   Lambda   │
└─────────────┘     └──────────────┘     └────────────┘
                            │                    │
                            ▼                    ▼
                    ┌──────────────┐     ┌────────────┐
                    │   DynamoDB   │     │  External  │
                    └──────────────┘     │   Gateway  │
                                         └────────────┘
```

## 📝 Log Correlation

All logs include:
- **RequestID**: AWS Lambda request ID
- **TraceID**: X-Ray trace ID for correlation
- **Timestamp**: RFC3339 format
- **Service**: Lambda function name
- **Level**: DEBUG, INFO, WARN, ERROR
- **Structured Data**: JSON format for CloudWatch Insights

Example Log Entry:
```json
{
  "timestamp": "2024-01-01T10:00:00.123Z",
  "level": "INFO",
  "service": "payments-adapter",
  "requestId": "abc-123",
  "traceId": "1-5e1b4f87-1234567890abcdef",
  "message": "Payment processed successfully",
  "attributes": {
    "paymentId": "pay-789",
    "amount": 100.50,
    "latencyMs": 234
  }
}
```

## 🚀 Performance Optimization Insights

1. **Cache wallet balances** in Lambda memory (with TTL)
2. **Batch DynamoDB writes** where possible
3. **Use provisioned concurrency** for critical Lambdas
4. **Implement request coalescing** for duplicate requests
5. **Pre-warm Lambdas** during high-traffic periods
