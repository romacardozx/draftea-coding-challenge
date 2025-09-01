package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/draftea-coding-challenge/lambdas/invoice-processor/internal/service"
	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/draftea-coding-challenge/shared/types"
)

type InvoiceHandler struct {
	service *service.PaymentService
	logger  *observability.Logger
}

func NewInvoiceHandler(service *service.PaymentService, logger *observability.Logger) *InvoiceHandler {
	return &InvoiceHandler{
		service: service,
		logger:  logger,
	}
}

// HandleRequest processes incoming requests
func (h *InvoiceHandler) HandleRequest(ctx context.Context, request interface{}) (interface{}, error) {
	// Check if this is an API Gateway request
	if apiReq, ok := request.(events.APIGatewayProxyRequest); ok {
		h.logger.Info("Processing API Gateway request", nil, map[string]interface{}{
			"method": apiReq.HTTPMethod,
			"path":   apiReq.Path,
		})

		switch {
		case apiReq.HTTPMethod == "GET" && apiReq.Path == "/health":
			return h.handleHealth()
		case apiReq.HTTPMethod == "POST" && apiReq.Path == "/payment":
			return h.handleCreatePayment(ctx, apiReq)
		case apiReq.HTTPMethod == "GET" && strings.HasPrefix(apiReq.Path, "/payment/"):
			return h.handleGetPayment(ctx, apiReq)
		case apiReq.HTTPMethod == "PUT" && strings.HasPrefix(apiReq.Path, "/payment/"):
			return h.handleUpdatePayment(ctx, apiReq)
		default:
			return events.APIGatewayProxyResponse{
				StatusCode: 404,
				Body:       `{"error": "not found"}`,
			}, nil
		}
	}

	// Handle direct invocation or Step Function input
	var input map[string]interface{}
	if inputMap, ok := request.(map[string]interface{}); ok {
		input = inputMap
	} else {
		// Try to unmarshal if it's a JSON string
		if jsonStr, ok := request.(string); ok {
			if err := json.Unmarshal([]byte(jsonStr), &input); err != nil {
				return map[string]interface{}{
					"statusCode": 400,
					"body":       `{"error": "invalid input format"}`,
				}, nil
			}
		} else {
			// Try to marshal and unmarshal to convert to map
			data, _ := json.Marshal(request)
			if err := json.Unmarshal(data, &input); err != nil {
				return map[string]interface{}{
					"statusCode": 400,
					"body":       `{"error": "unsupported request type"}`,
				}, nil
			}
		}
	}

	// Check for action field (Step Function input)
	if action, ok := input["action"].(string); ok {
		return h.handleStepFunctionInput(ctx, action, input)
	}

	// Check for httpMethod field (API Gateway-like input)
	if httpMethod, ok := input["httpMethod"].(string); ok {
		// Convert to APIGatewayProxyRequest
		apiReq := events.APIGatewayProxyRequest{
			HTTPMethod: httpMethod,
		}
		if path, ok := input["path"].(string); ok {
			apiReq.Path = path
		}
		if body, ok := input["body"].(string); ok {
			apiReq.Body = body
		}
		if pathParams, ok := input["pathParameters"].(map[string]interface{}); ok {
			apiReq.PathParameters = make(map[string]string)
			for k, v := range pathParams {
				apiReq.PathParameters[k] = fmt.Sprintf("%v", v)
			}
		}
		return h.HandleRequest(ctx, apiReq)
	}

	return map[string]interface{}{
		"statusCode": 400,
		"body":       `{"error": "unsupported request format"}`,
	}, nil
}

func (h *InvoiceHandler) handleHealth() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       `{"status":"healthy"}`,
	}, nil
}

func (h *InvoiceHandler) handleCreatePayment(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var payment types.Payment
	if err := json.Unmarshal([]byte(request.Body), &payment); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"invalid request body"}`,
		}, nil
	}

	paymentResp, err := h.service.CreatePayment(ctx, service.CreatePaymentRequest{
		UserID:   payment.UserID,
		Amount:   payment.Amount,
		Currency: payment.Currency,
		Metadata: payment.Metadata,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error":"%s"}`, err.Error()),
		}, nil
	}

	response, _ := json.Marshal(paymentResp)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(response),
	}, nil
}

func (h *InvoiceHandler) handleGetPayment(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	paymentID := strings.TrimPrefix(request.Path, "/payment/")
	if paymentID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"payment ID required"}`,
		}, nil
	}

	payment, err := h.service.GetPayment(ctx, paymentID)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       `{"error":"payment not found"}`,
		}, nil
	}

	response, _ := json.Marshal(payment)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(response),
	}, nil
}

func (h *InvoiceHandler) handleUpdatePayment(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	paymentID := strings.TrimPrefix(request.Path, "/payment/")
	if paymentID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"payment ID required"}`,
		}, nil
	}

	var updateReq struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(request.Body), &updateReq); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"invalid request body"}`,
		}, nil
	}

	_, err := h.service.UpdatePaymentStatus(ctx, paymentID, types.PaymentStatus(updateReq.Status), "")
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error":"%s"}`, err.Error()),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       `{"success":true}`,
	}, nil
}

// handleStepFunctionInput handles requests from Step Functions
func (h *InvoiceHandler) handleStepFunctionInput(ctx context.Context, action string, input map[string]interface{}) (interface{}, error) {
	h.logger.Info("Processing Step Function request", map[string]interface{}{
		"action": action,
		"input":  input,
	})

	switch action {
	case "create_payment":
		// Create payment record
		paymentID, _ := input["paymentId"].(string)
		userID, _ := input["userId"].(string)
		amount, _ := input["amount"].(float64)
		currency, _ := input["currency"].(string)
		correlationID, _ := input["correlationId"].(string)
		
		// Convert metadata from map[string]interface{} to map[string]string
		var metadata map[string]string
		if metaInterface, ok := input["metadata"].(map[string]interface{}); ok {
			metadata = make(map[string]string)
			for k, v := range metaInterface {
				if strVal, ok := v.(string); ok {
					metadata[k] = strVal
				} else {
					metadata[k] = fmt.Sprintf("%v", v)
				}
			}
		}

		// Use the service's CreatePaymentFromStepFunction method
		stepInput := types.StepFunctionInput{
			Action:        "create_payment",
			PaymentID:     paymentID,
			UserID:        userID,
			Amount:        amount,
			Currency:      currency,
			CorrelationID: correlationID,
			Metadata:      metadata,
		}

		payment, err := h.service.CreatePaymentFromStepFunction(ctx, stepInput)
		if err != nil {
			h.logger.Error("Failed to create payment", err, nil)
			return types.LambdaResponse{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		return types.LambdaResponse{
			Success: true,
			Data:    payment,
		}, nil

	case "update_payment", "update_status":
		paymentID, _ := input["paymentId"].(string)
		status, _ := input["status"].(string)
		externalID, _ := input["externalId"].(string)

		paymentStatus := types.PaymentStatus(status)
		payment, err := h.service.UpdatePaymentStatus(ctx, paymentID, paymentStatus, externalID)
		if err != nil {
			h.logger.Error("Failed to update payment", err, nil)
			return types.LambdaResponse{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		return types.LambdaResponse{
			Success: true,
			Data:    payment,
		}, nil

	default:
		return map[string]interface{}{
			"statusCode": 400,
			"body":       fmt.Sprintf(`{"error": "unknown action: %s"}`, action),
		}, nil
	}
}

