package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application-specific error
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"statusCode"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Err        error                  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Common error codes
const (
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeInsufficientFunds = "INSUFFICIENT_FUNDS"
	ErrCodePaymentFailed     = "PAYMENT_FAILED"
	ErrCodeCircuitOpen       = "CIRCUIT_BREAKER_OPEN"
	ErrCodeInternal          = "INTERNAL_ERROR"
	ErrCodeTimeout           = "TIMEOUT"
	ErrCodeDuplicatePayment  = "DUPLICATE_PAYMENT"
)

// Constructor functions for common errors
func NewValidationError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       ErrCodeValidation,
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Details:    details,
	}
}

func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Code:       ErrCodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound,
	}
}

func NewInsufficientFundsError(available, required float64) *AppError {
	return &AppError{
		Code:       ErrCodeInsufficientFunds,
		Message:    "Insufficient funds",
		StatusCode: http.StatusPaymentRequired,
		Details: map[string]interface{}{
			"available": available,
			"required":  required,
		},
	}
}

func NewPaymentFailedError(reason string) *AppError {
	return &AppError{
		Code:       ErrCodePaymentFailed,
		Message:    fmt.Sprintf("Payment failed: %s", reason),
		StatusCode: http.StatusPaymentRequired,
	}
}

func NewCircuitOpenError(service string) *AppError {
	return &AppError{
		Code:       ErrCodeCircuitOpen,
		Message:    fmt.Sprintf("Service %s is temporarily unavailable", service),
		StatusCode: http.StatusServiceUnavailable,
	}
}

func NewInternalError(err error) *AppError {
	return &AppError{
		Code:       ErrCodeInternal,
		Message:    "Internal server error",
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

func NewDuplicatePaymentError(paymentID string) *AppError {
	return &AppError{
		Code:       ErrCodeDuplicatePayment,
		Message:    fmt.Sprintf("Payment %s already exists", paymentID),
		StatusCode: http.StatusConflict,
	}
}

// Helper function to wrap errors
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}