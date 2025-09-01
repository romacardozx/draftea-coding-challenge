package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/draftea-coding-challenge/lambdas/payments-adapter/internal/service"
	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/draftea-coding-challenge/shared/types"
	"github.com/draftea-coding-challenge/shared/utils"
)

// PaymentAdapterHandler handles payment adapter requests
type PaymentAdapterHandler struct {
	service *service.PaymentAdapterService
	logger  *observability.Logger
}

// NewPaymentAdapterHandler creates a new payment adapter handler
func NewPaymentAdapterHandler(service *service.PaymentAdapterService, logger *observability.Logger) *PaymentAdapterHandler {
	return &PaymentAdapterHandler{
		service: service,
		logger:  logger,
	}
}

// HandleRequest processes incoming Lambda requests
func (h *PaymentAdapterHandler) HandleRequest(ctx context.Context, request interface{}) (interface{}, error) {
	// Try to handle as API Gateway request first
	if apiReq, ok := request.(events.APIGatewayProxyRequest); ok {
		return h.handleAPIGatewayRequest(ctx, apiReq)
	}
	
	// Try to handle as Step Function input or direct invocation
	var inputMap map[string]interface{}
	
	// If request is a string, parse it as JSON
	if strReq, ok := request.(string); ok {
		if err := json.Unmarshal([]byte(strReq), &inputMap); err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       `{"error": "invalid JSON input"}`,
			}, nil
		}
	} else {
		// Try to convert request to map
		requestBytes, err := json.Marshal(request)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       `{"error": "failed to process request"}`,
			}, nil
		}
		if err := json.Unmarshal(requestBytes, &inputMap); err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       `{"error": "invalid request format"}`,
			}, nil
		}
	}
	
	// Check if this is an API Gateway-style request (has httpMethod and path)
	if httpMethod, hasMethod := inputMap["httpMethod"].(string); hasMethod {
		if path, hasPath := inputMap["path"].(string); hasPath {
			// Handle health check
			if path == "/health" {
				return events.APIGatewayProxyResponse{
					StatusCode: 200,
					Body:       `{"status":"healthy"}`,
				}, nil
			}
			
			// Convert to API Gateway request and handle
			apiReq := events.APIGatewayProxyRequest{
				HTTPMethod: httpMethod,
				Path:       path,
			}
			if body, ok := inputMap["body"].(string); ok {
				apiReq.Body = body
			}
			
			return h.handleAPIGatewayRequest(ctx, apiReq)
		}
	}
	
	// Check if this is a Step Function input (has action field)
	if _, ok := inputMap["action"].(string); ok {
		var stepFuncInput types.StepFunctionInput
		stepFuncInputBytes, _ := json.Marshal(inputMap)
		if err := json.Unmarshal(stepFuncInputBytes, &stepFuncInput); err == nil {
			return h.handleStepFunctionRequest(ctx, &stepFuncInput)
		}
	}
	
	return events.APIGatewayProxyResponse{
		StatusCode: 400,
		Body:       `{"error": "unsupported request type"}`,
	}, nil
}

// handleAPIGatewayRequest handles HTTP API Gateway requests
func (h *PaymentAdapterHandler) handleAPIGatewayRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	h.logger.Info("Processing API Gateway request", map[string]interface{}{
		"path":   request.Path,
		"method": request.HTTPMethod,
	})

	switch request.Path {
	case "/payment/process":
		return h.handleProcessPayment(ctx, request)
	case "/payment/status":
		return h.handleGetStatus(ctx, request)
	case "/circuit/status":
		return h.handleCircuitStatus(ctx)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       `{"error": "not found"}`,
		}, nil
	}
}

// handleStepFunctionRequest handles Step Function inputs
func (h *PaymentAdapterHandler) handleStepFunctionRequest(ctx context.Context, input *types.StepFunctionInput) (types.LambdaResponse, error) {
	h.logger.Info("Processing Step Function request", map[string]interface{}{
		"action": input.Action,
	})

	switch input.Action {
	case "process_payment":
		return h.processPaymentFromStepFunction(ctx, input)
	case "check_status":
		return h.checkStatusFromStepFunction(ctx, input)
	default:
		return types.LambdaResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", input.Action),
		}, nil
	}
}

// handleProcessPayment handles payment processing requests
func (h *PaymentAdapterHandler) handleProcessPayment(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var payment types.Payment
	if err := json.Unmarshal([]byte(request.Body), &payment); err != nil {
		return utils.ErrorResponse(400, "invalid request format")
	}

	resp, err := h.service.ProcessPaymentFromPayment(ctx, &payment)
	if err != nil {
		h.logger.Error("Failed to process payment", err, map[string]interface{}{
			"paymentId": payment.ID,
		})
		return utils.ErrorResponse(503, fmt.Sprintf("gateway error: %s", err.Error()))
	}

	return utils.SuccessResponse(200, resp)
}

// handleGetStatus handles payment status requests
func (h *PaymentAdapterHandler) handleGetStatus(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	externalID := request.QueryStringParameters["externalId"]
	if externalID == "" {
		return utils.ErrorResponse(400, "externalId is required")
	}

	resp, err := h.service.GetPaymentStatus(ctx, externalID)
	if err != nil {
		h.logger.Error("Failed to get payment status", err, map[string]interface{}{
			"externalId": externalID,
		})
		return utils.ErrorResponse(503, fmt.Sprintf("gateway error: %s", err.Error()))
	}

	return utils.SuccessResponse(200, resp)
}

// handleCircuitStatus returns the circuit breaker status
func (h *PaymentAdapterHandler) handleCircuitStatus(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	status := h.service.GetCircuitBreakerState()
	
	return utils.SuccessResponse(200, map[string]string{"status": status})
}

// processPaymentFromStepFunction processes payment via Step Functions
func (h *PaymentAdapterHandler) processPaymentFromStepFunction(ctx context.Context, input *types.StepFunctionInput) (types.LambdaResponse, error) {
	resp, err := h.service.ProcessStepFunctionPayment(ctx, input)
	if err != nil {
		return types.LambdaResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	return *resp, nil
}

// checkStatusFromStepFunction checks payment status via Step Functions
func (h *PaymentAdapterHandler) checkStatusFromStepFunction(ctx context.Context, input *types.StepFunctionInput) (types.LambdaResponse, error) {
	resp, err := h.service.CheckStepFunctionStatus(ctx, input)
	if err != nil {
		return types.LambdaResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	return *resp, nil
}
