package circuitbreaker

import (
	"context"
	"fmt"
	"time"

	"github.com/draftea-coding-challenge/lambdas/payments-adapter/internal/gateway"
	"github.com/draftea-coding-challenge/shared/types"
	"github.com/sony/gobreaker"
)

// CircuitBreakerClient wraps a payment gateway client with circuit breaker functionality
type CircuitBreakerClient struct {
	client  gateway.PaymentGatewayClient
	breaker *gobreaker.CircuitBreaker
}

// NewCircuitBreakerClient creates a new circuit breaker wrapped client
func NewCircuitBreakerClient(client gateway.PaymentGatewayClient, name string) *CircuitBreakerClient {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,                // Number of requests allowed to pass through when half-open
		Interval:    60 * time.Second, // Time window for failure rate calculation
		Timeout:     30 * time.Second, // Time before attempting to recover from open state
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6 // Trip if 60% of requests fail
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			fmt.Printf("Circuit breaker %s: state changed from %s to %s\n", name, from, to)
		},
	}

	return &CircuitBreakerClient{
		client:  client,
		breaker: gobreaker.NewCircuitBreaker(settings),
	}
}

// ProcessPayment processes a payment with circuit breaker protection
func (c *CircuitBreakerClient) ProcessPayment(ctx context.Context, payment *types.Payment) (*gateway.GatewayResponse, error) {
	result, err := c.breaker.Execute(func() (interface{}, error) {
		return c.client.ProcessPayment(ctx, payment)
	})

	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return nil, fmt.Errorf("circuit breaker open: %w", err)
		}
		return nil, err
	}

	return result.(*gateway.GatewayResponse), nil
}

// GetPaymentStatus gets payment status with circuit breaker protection
func (c *CircuitBreakerClient) GetPaymentStatus(ctx context.Context, externalID string) (*gateway.GatewayResponse, error) {
	result, err := c.breaker.Execute(func() (interface{}, error) {
		return c.client.GetPaymentStatus(ctx, externalID)
	})

	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return nil, fmt.Errorf("circuit breaker open: %w", err)
		}
		return nil, err
	}

	return result.(*gateway.GatewayResponse), nil
}

// RefundPayment processes a refund with circuit breaker protection
func (c *CircuitBreakerClient) RefundPayment(ctx context.Context, externalID string, amount float64) (*gateway.GatewayResponse, error) {
	result, err := c.breaker.Execute(func() (interface{}, error) {
		return c.client.RefundPayment(ctx, externalID, amount)
	})

	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return nil, fmt.Errorf("circuit breaker open: %w", err)
		}
		return nil, err
	}

	return result.(*gateway.GatewayResponse), nil
}

// GetState returns the current state of the circuit breaker
func (c *CircuitBreakerClient) GetState() string {
	counts := c.breaker.Counts()
	state := "unknown"
	
	// Determine state based on breaker behavior
	if counts.ConsecutiveFailures > 0 && counts.ConsecutiveSuccesses == 0 {
		if counts.Requests == 0 {
			state = "open"
		} else {
			state = "half-open"
		}
	} else {
		state = "closed"
	}

	return fmt.Sprintf("state: %s, requests: %d, failures: %d, successes: %d", 
		state, counts.Requests, counts.TotalFailures, counts.TotalSuccesses)
}
