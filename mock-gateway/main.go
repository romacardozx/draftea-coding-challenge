package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// PaymentRequest represents incoming payment request
type PaymentRequest struct {
	PaymentID     string            `json:"paymentId"`
	Amount        float64           `json:"amount"`
	Currency      string            `json:"currency"`
	UserID        string            `json:"userId"`
	CorrelationID string            `json:"correlationId"`
	Metadata      map[string]string `json:"metadata"`
}

// PaymentResponse represents the gateway response
type PaymentResponse struct {
	ExternalID string `json:"externalId"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Timestamp  int64  `json:"timestamp"`
}

// PaymentStore stores payment states
type PaymentStore struct {
	mu       sync.RWMutex
	payments map[string]*PaymentResponse
}

var store = &PaymentStore{
	payments: make(map[string]*PaymentResponse),
}

func main() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Setup routes
	http.HandleFunc("/payment/process", handleProcessPayment)
	http.HandleFunc("/payment/status", handleGetPaymentStatus)
	http.HandleFunc("/payment/refund", handleRefundPayment)
	http.HandleFunc("/health", handleHealth)

	// Start server
	port := ":3000"
	log.Printf("Mock Payment Gateway starting on port %s", port)
	log.Fatal(http.ListenAndServe(port, logMiddleware(http.DefaultServeMux)))
}

// logMiddleware logs incoming requests
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// handleProcessPayment processes a new payment
func handleProcessPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify API key
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		log.Printf("Missing API key")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Generate external ID
	externalID := fmt.Sprintf("ext_%s_%d", req.PaymentID, time.Now().Unix())

	// Simulate payment processing with random outcomes
	var status, message string
	
	// Special case for testing circuit breaker - amount 999.99 always fails
	if req.Amount == 999.99 {
		status = "error"
		message = "Simulated gateway error for testing"
	} else {
		random := rand.Float64()
		switch {
		case random < 0.7: // 70% success rate
			status = "approved"
			message = "Payment processed successfully"
		case random < 0.85: // 15% pending (async processing)
			status = "pending"
			message = "Payment is being processed"
			// Simulate async approval after delay
			go func() {
				time.Sleep(5 * time.Second)
				store.mu.Lock()
				if payment, exists := store.payments[externalID]; exists {
					payment.Status = "approved"
					payment.Message = "Payment approved after async processing"
				}
				store.mu.Unlock()
			}()
		case random < 0.95: // 10% declined
			status = "declined"
			message = "Payment declined by issuer"
		default: // 5% error
			status = "error"
			message = "Internal processing error"
		}
	}

	// Store payment response
	response := &PaymentResponse{
		ExternalID: externalID,
		Status:     status,
		Message:    message,
		Timestamp:  time.Now().Unix(),
	}

	store.mu.Lock()
	store.payments[externalID] = response
	store.mu.Unlock()

	log.Printf("Payment %s processed: status=%s, externalId=%s", req.PaymentID, status, externalID)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetPaymentStatus retrieves payment status
func handleGetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify API key
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		log.Printf("Missing API key")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	externalID := r.URL.Query().Get("externalId")
	if externalID == "" {
		http.Error(w, "Missing externalId parameter", http.StatusBadRequest)
		return
	}

	store.mu.RLock()
	payment, exists := store.payments[externalID]
	store.mu.RUnlock()

	if !exists {
		log.Printf("Payment not found: %s", externalID)
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	log.Printf("Status check for %s: %s", externalID, payment.Status)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

// handleRefundPayment processes a refund
func handleRefundPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify API key
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		log.Printf("Missing API key")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ExternalID string  `json:"externalId"`
		Amount     float64 `json:"amount"`
		Reason     string  `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	store.mu.RLock()
	payment, exists := store.payments[req.ExternalID]
	store.mu.RUnlock()

	if !exists {
		log.Printf("Payment not found for refund: %s", req.ExternalID)
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	// Check if payment can be refunded
	if payment.Status != "approved" {
		log.Printf("Cannot refund payment %s with status %s", req.ExternalID, payment.Status)
		http.Error(w, "Payment cannot be refunded", http.StatusBadRequest)
		return
	}

	// Process refund (90% success rate)
	var status, message string
	if rand.Float64() < 0.9 {
		status = "refunded"
		message = fmt.Sprintf("Refund processed: %s", req.Reason)
	} else {
		status = "refund_failed"
		message = "Refund could not be processed"
	}

	refundResponse := &PaymentResponse{
		ExternalID: fmt.Sprintf("refund_%s_%d", req.ExternalID, time.Now().Unix()),
		Status:     status,
		Message:    message,
		Timestamp:  time.Now().Unix(),
	}

	// Update original payment status if refund successful
	if status == "refunded" {
		store.mu.Lock()
		payment.Status = "refunded"
		payment.Message = message
		store.mu.Unlock()
	}

	log.Printf("Refund processed for %s: status=%s", req.ExternalID, status)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refundResponse)
}

// handleHealth returns health status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "mock-payment-gateway",
	})
}
