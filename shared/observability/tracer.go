package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-xray-sdk-go/xray"
)

type Tracer struct {
	logger *Logger
}

func NewTracer(logger *Logger) *Tracer {
	return &Tracer{
		logger: logger,
	}
}

// StartSegment creates a new trace segment for distributed tracing
func (t *Tracer) StartSegment(ctx context.Context, name string) (context.Context, *xray.Segment) {
	ctx, seg := xray.BeginSegment(ctx, name)
	
	t.logger.Debug("Trace segment started", map[string]interface{}{
		"segmentName": name,
		"traceId":     seg.TraceID,
	})
	
	return ctx, seg
}

// StartSubsegment creates a subsegment within an existing trace
func (t *Tracer) StartSubsegment(ctx context.Context, name string) (context.Context, *xray.Segment) {
	ctx, subseg := xray.BeginSubsegment(ctx, name)
	
	t.logger.Debug("Subsegment started", map[string]interface{}{
		"subsegmentName": name,
	})
	
	return ctx, subseg
}

// AddAnnotation adds searchable metadata to the current segment
func (t *Tracer) AddAnnotation(ctx context.Context, key string, value interface{}) {
	xray.AddAnnotation(ctx, key, value)
}

// AddMetadata adds non-searchable metadata to the current segment
func (t *Tracer) AddMetadata(ctx context.Context, key string, value interface{}) {
	xray.AddMetadata(ctx, key, value)
}

// RecordError records an error in the current segment
func (t *Tracer) RecordError(ctx context.Context, err error) {
	if err != nil {
		xray.AddError(ctx, err)
		t.logger.Error("Error recorded in trace", err, nil)
	}
}

// TraceRemoteCall traces external service calls
func (t *Tracer) TraceRemoteCall(ctx context.Context, service string, operation string, fn func() error) error {
	ctx, subseg := t.StartSubsegment(ctx, fmt.Sprintf("%s:%s", service, operation))
	defer subseg.Close(nil)
	
	start := time.Now()
	err := fn()
	duration := time.Since(start).Milliseconds()
	
	// Add metadata
	t.AddMetadata(ctx, "service", service)
	t.AddMetadata(ctx, "operation", operation)
	t.AddMetadata(ctx, "duration_ms", duration)
	
	if err != nil {
		t.RecordError(ctx, err)
		subseg.AddError(err)
	}
	
	return err
}

// TraceDynamoDBOperation traces DynamoDB operations
func (t *Tracer) TraceDynamoDBOperation(ctx context.Context, table string, operation string, fn func() error) error {
	ctx, subseg := t.StartSubsegment(ctx, fmt.Sprintf("DynamoDB:%s", operation))
	defer subseg.Close(nil)
	
	// AWS X-Ray recognizes DynamoDB namespace
	subseg.Namespace = "aws"
	subseg.Name = "DynamoDB"
	
	// Add DynamoDB-specific annotations
	t.AddAnnotation(ctx, "table_name", table)
	t.AddAnnotation(ctx, "operation", operation)
	
	err := fn()
	if err != nil {
		t.RecordError(ctx, err)
	}
	
	return err
}

// TraceWithRetry traces operations with retry logic
func (t *Tracer) TraceWithRetry(ctx context.Context, operation string, attempt int, fn func() error) error {
	ctx, subseg := t.StartSubsegment(ctx, fmt.Sprintf("%s:attempt_%d", operation, attempt))
	defer subseg.Close(nil)
	
	t.AddAnnotation(ctx, "retry_attempt", attempt)
	
	err := fn()
	if err != nil {
		t.AddMetadata(ctx, "retry_error", err.Error())
	}
	
	return err
}
