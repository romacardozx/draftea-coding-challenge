package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type MetricsCollector struct {
	logger  *Logger
	db      *dynamodb.DynamoDB
	table   string
	service string
}

type Metric struct {
	Name      string
	Value     float64
	Unit      string
	Timestamp time.Time
	Tags      map[string]string
}

func NewMetricsCollector(logger *Logger, db *dynamodb.DynamoDB, service string) *MetricsCollector {
	return &MetricsCollector{
		logger:  logger,
		db:      db,
		table:   "Metrics",
		service: service,
	}
}

// RecordPaymentMetric records payment-related metrics
func (m *MetricsCollector) RecordPaymentMetric(ctx context.Context, metricType string, value float64, status string) {
	metric := Metric{
		Name:      metricType,
		Value:     value,
		Unit:      "Count",
		Timestamp: time.Now(),
		Tags: map[string]string{
			"service": m.service,
			"status":  status,
		},
	}
	
	// Log for CloudWatch Metrics
	m.logger.Info("Metric recorded", map[string]interface{}{
		"metricName":  metric.Name,
		"metricValue": metric.Value,
		"metricUnit":  metric.Unit,
		"metricTags":  metric.Tags,
		"metricType":  "BUSINESS_METRIC",
	})
	
	// Store in DynamoDB for historical analysis
	m.storeMetric(ctx, metric)
}

// RecordLatency records operation latency
func (m *MetricsCollector) RecordLatency(ctx context.Context, operation string, latencyMs float64) {
	metric := Metric{
		Name:      "OPERATION_LATENCY",
		Value:     latencyMs,
		Unit:      "Milliseconds",
		Timestamp: time.Now(),
		Tags: map[string]string{
			"service":   m.service,
			"operation": operation,
		},
	}
	
	m.logger.Info("Latency recorded", map[string]interface{}{
		"operation":   operation,
		"latencyMs":   latencyMs,
		"metricType":  "PERFORMANCE_METRIC",
	})
	
	m.storeMetric(ctx, metric)
}

// RecordCircuitBreakerEvent records circuit breaker state changes
func (m *MetricsCollector) RecordCircuitBreakerEvent(ctx context.Context, state string, serviceName string) {
	m.logger.Warn("Circuit breaker state changed", map[string]interface{}{
		"circuitBreakerState": state,
		"targetService":       serviceName,
		"metricType":          "RESILIENCE_METRIC",
	})
	
	metric := Metric{
		Name:      "CIRCUIT_BREAKER_STATE_CHANGE",
		Value:     1,
		Unit:      "Count",
		Timestamp: time.Now(),
		Tags: map[string]string{
			"service":       m.service,
			"targetService": serviceName,
			"newState":      state,
		},
	}
	
	m.storeMetric(ctx, metric)
}

// RecordErrorRate tracks error rates
func (m *MetricsCollector) RecordErrorRate(ctx context.Context, errorType string) {
	m.logger.Error("Error occurred", nil, map[string]interface{}{
		"errorType":  errorType,
		"metricType": "ERROR_METRIC",
	})
	
	metric := Metric{
		Name:      "ERROR_RATE",
		Value:     1,
		Unit:      "Count",
		Timestamp: time.Now(),
		Tags: map[string]string{
			"service":   m.service,
			"errorType": errorType,
		},
	}
	
	m.storeMetric(ctx, metric)
}

func (m *MetricsCollector) storeMetric(ctx context.Context, metric Metric) {
	// Partition key: metricType#date for efficient querying
	pk := fmt.Sprintf("%s#%s", metric.Name, metric.Timestamp.Format("2006-01-02"))
	sk := metric.Timestamp.Format(time.RFC3339Nano)
	
	item := map[string]*dynamodb.AttributeValue{
		"PK":        {S: aws.String(pk)},
		"SK":        {S: aws.String(sk)},
		"value":     {N: aws.String(fmt.Sprintf("%f", metric.Value))},
		"unit":      {S: aws.String(metric.Unit)},
		"service":   {S: aws.String(m.service)},
		"timestamp": {S: aws.String(sk)},
	}
	
	// Add tags as map
	if len(metric.Tags) > 0 {
		tagMap := make(map[string]*dynamodb.AttributeValue)
		for k, v := range metric.Tags {
			tagMap[k] = &dynamodb.AttributeValue{S: aws.String(v)}
		}
		item["dimensions"] = &dynamodb.AttributeValue{M: tagMap}
	}
	
	_, err := m.db.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(m.table),
		Item:      item,
	})
	
	if err != nil {
		m.logger.Error("Failed to store metric", err, nil)
	}
}

// Key Metrics to Track:
// 1. Payment Success Rate
// 2. Payment Processing Time
// 3. Wallet Operation Latency
// 4. Circuit Breaker Trips
// 5. Retry Attempts
// 6. Refund Rate
// 7. Error Rate by Type
