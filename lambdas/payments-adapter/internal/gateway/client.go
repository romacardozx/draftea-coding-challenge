package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/draftea-coding-challenge/shared/types"
)

// PaymentGatewayClient interface for payment gateway operations
type PaymentGatewayClient interface {
	ProcessPayment(ctx context.Context, payment *types.Payment) (*GatewayResponse, error)
	GetPaymentStatus(ctx context.Context, externalID string) (*GatewayResponse, error)
	RefundPayment(ctx context.Context, externalID string, amount float64) (*GatewayResponse, error)
}

// GatewayResponse represents a response from the payment gateway
type GatewayResponse struct {
	ExternalID string `json:"externalId"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	Timestamp  int64  `json:"timestamp"`
}

// GatewayRequest represents a request to the payment gateway
type GatewayRequest struct {
	PaymentID     string            `json:"paymentId"`
	Amount        float64           `json:"amount"`
	Currency      string            `json:"currency"`
	UserID        string            `json:"userId"`
	CorrelationID string            `json:"correlationId"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Client implements PaymentGatewayClient
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewClient creates a new payment gateway client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProcessPayment sends a payment request to the gateway
func (c *Client) ProcessPayment(ctx context.Context, payment *types.Payment) (*GatewayResponse, error) {
	request := &GatewayRequest{
		PaymentID:     payment.ID,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		UserID:        payment.UserID,
		CorrelationID: payment.CorrelationID,
		Metadata:      payment.Metadata,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payment/process", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var gatewayResp GatewayResponse
	if err := json.NewDecoder(resp.Body).Decode(&gatewayResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &gatewayResp, fmt.Errorf("gateway returned status %d: %s", resp.StatusCode, gatewayResp.Message)
	}

	return &gatewayResp, nil
}

// GetPaymentStatus retrieves the status of a payment from the gateway
func (c *Client) GetPaymentStatus(ctx context.Context, externalID string) (*GatewayResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/payment/status?externalId="+externalID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var gatewayResp GatewayResponse
	if err := json.NewDecoder(resp.Body).Decode(&gatewayResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &gatewayResp, fmt.Errorf("gateway returned status %d: %s", resp.StatusCode, gatewayResp.Message)
	}

	return &gatewayResp, nil
}

// RefundPayment processes a refund through the gateway
func (c *Client) RefundPayment(ctx context.Context, externalID string, amount float64) (*GatewayResponse, error) {
	request := map[string]interface{}{
		"externalId": externalID,
		"amount":     amount,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payment/refund", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var gatewayResp GatewayResponse
	if err := json.NewDecoder(resp.Body).Decode(&gatewayResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &gatewayResp, fmt.Errorf("gateway returned status %d: %s", resp.StatusCode, gatewayResp.Message)
	}

	return &gatewayResp, nil
}