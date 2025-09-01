package utils

import (
	"regexp"
	"strings"

	"github.com/draftea-coding-challenge/shared/errors"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	uuidRegex  = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
)

// ValidatePaymentRequest validates payment request data
func ValidatePaymentRequest(paymentID, userID string, amount float64) error {
	validationErrors := make(map[string]interface{})

	if paymentID == "" {
		validationErrors["payment_id"] = "Payment ID is required"
	}

	if userID == "" {
		validationErrors["user_id"] = "User ID is required"
	}

	if amount <= 0 {
		validationErrors["amount"] = "Amount must be greater than 0"
	}

	if amount > 1000000 {
		validationErrors["amount"] = "Amount exceeds maximum limit"
	}

	if len(validationErrors) > 0 {
		return errors.NewValidationError("Invalid payment request", validationErrors)
	}

	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// ValidateUUID validates UUID format
func ValidateUUID(uuid string) bool {
	return uuidRegex.MatchString(uuid)
}

// SanitizeString removes potentially harmful characters
func SanitizeString(s string) string {
	// Remove leading/trailing whitespace
	s = strings.TrimSpace(s)
	
	// Remove control characters
	s = regexp.MustCompile(`[\x00-\x1F\x7F]`).ReplaceAllString(s, "")
	
	return s
}
