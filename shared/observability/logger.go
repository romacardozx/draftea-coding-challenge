package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
)

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

type Logger struct {
	ServiceName string
	RequestID   string
}

type LogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	Service     string                 `json:"service"`
	RequestID   string                 `json:"requestId"`
	Message     string                 `json:"message"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    *float64               `json:"duration,omitempty"`
	TraceID     string                 `json:"traceId,omitempty"`
}

func NewLogger(ctx context.Context, serviceName string) *Logger {
	requestID := ""
	if lc, ok := lambdacontext.FromContext(ctx); ok {
		requestID = lc.AwsRequestID
	}
	
	// X-Ray trace ID for distributed tracing
	if traceID := os.Getenv("_X_AMZN_TRACE_ID"); traceID != "" {
		// Will be automatically correlated in CloudWatch
	}
	
	return &Logger{
		ServiceName: serviceName,
		RequestID:   requestID,
	}
}

func (l *Logger) Info(message string, attrs ...map[string]interface{}) {
	l.log(INFO, message, nil, attrs...)
}

func (l *Logger) Warn(message string, attrs ...map[string]interface{}) {
	l.log(WARN, message, nil, attrs...)
}

func (l *Logger) Error(message string, err error, attrs ...map[string]interface{}) {
	l.log(ERROR, message, err, attrs...)
}

func (l *Logger) Debug(message string, attrs ...map[string]interface{}) {
	l.log(DEBUG, message, nil, attrs...)
}

func (l *Logger) WithTimer(operation string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start).Milliseconds()
		l.Info("Operation completed", map[string]interface{}{
			"operation":    operation,
			"durationMs":   duration,
			"metricType":   "LATENCY",
		})
	}
}

func (l *Logger) log(level LogLevel, message string, err error, attrs ...map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Service:   l.ServiceName,
		RequestID: l.RequestID,
		Message:   message,
	}
	
	if err != nil {
		entry.Error = err.Error()
	}
	
	if len(attrs) > 0 && attrs[0] != nil {
		entry.Attributes = attrs[0]
	}
	
	// Add trace ID if available
	if traceID := os.Getenv("_X_AMZN_TRACE_ID"); traceID != "" {
		entry.TraceID = traceID
	}
	
	// Structured logging for CloudWatch Insights
	output, _ := json.Marshal(entry)
	fmt.Println(string(output))
}
