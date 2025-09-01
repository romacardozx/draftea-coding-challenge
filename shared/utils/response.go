package utils

import (
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/draftea-coding-challenge/shared/errors"
)

// APIResponse creates a standard API Gateway response
func APIResponse(statusCode int, body interface{}) (events.APIGatewayProxyResponse, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       `{"error": "Failed to marshal response"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       string(bodyBytes),
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
	}, nil
}

// ErrorResponse creates an error API response with status code
func ErrorResponse(statusCode int, message string) (events.APIGatewayProxyResponse, error) {
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}

	body, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       string(body),
	}, nil
}

// SuccessResponse creates a success API response with status code
func SuccessResponse(statusCode int, data interface{}) (events.APIGatewayProxyResponse, error) {
	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}

	body, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       string(body),
	}, nil
}

// ParseJSON parses JSON from request body
func ParseJSON(body string, v interface{}) error {
	if body == "" {
		return errors.NewValidationError("Request body is empty", nil)
	}
	
	if err := json.Unmarshal([]byte(body), v); err != nil {
		return errors.NewValidationError("Invalid JSON", map[string]interface{}{
			"error": err.Error(),
		})
	}
	
	return nil
}
