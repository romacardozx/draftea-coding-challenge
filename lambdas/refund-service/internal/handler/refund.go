package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/draftea-coding-challenge/lambdas/refund-service/internal/service"
	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/draftea-coding-challenge/shared/types"
	"github.com/draftea-coding-challenge/shared/utils"
)

type RefundHandler struct {
	service *service.RefundService
	logger  *observability.Logger
}

func NewRefundHandler(service *service.RefundService, logger *observability.Logger) *RefundHandler {
	return &RefundHandler{
		service: service,
		logger:  logger,
	}
}

func (h *RefundHandler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	h.logger.Info("Received refund request", map[string]interface{}{
		"path":   request.Path,
		"method": request.HTTPMethod,
	})

	// Check if this is from Step Functions or API Gateway
	var input types.StepFunctionInput
	if err := json.Unmarshal([]byte(request.Body), &input); err == nil && input.Action != "" {
		return h.handleStepFunctionRequest(ctx, input)
	}

	// Handle API Gateway request
	switch request.Path {
	case "/refund/process":
		return h.processRefund(ctx, request)
	case "/refund/status":
		return h.getRefundStatus(ctx, request)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       `{"error": "Not found"}`,
		}, nil
	}
}

func (h *RefundHandler) handleStepFunctionRequest(ctx context.Context, input types.StepFunctionInput) (events.APIGatewayProxyResponse, error) {
	h.logger.Info("Processing Step Function request", map[string]interface{}{
		"action":     input.Action,
		"payment_id": input.PaymentID,
	})

	switch input.Action {
	case "process_refund":
		return h.processRefundFromStepFunction(ctx, input)
	case "check_refund_status":
		return h.checkRefundStatusFromStepFunction(ctx, input)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Unknown action: %s"}`, input.Action),
		}, nil
	}
}

func (h *RefundHandler) processRefund(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var refundRequest service.RefundRequest

	if err := json.Unmarshal([]byte(request.Body), &refundRequest); err != nil {
		h.logger.Error("Failed to unmarshal refund request", err, nil)
		return utils.ErrorResponse(400, "Invalid request body")
	}

	response, err := h.service.ProcessRefund(ctx, &refundRequest)
	if err != nil {
		h.logger.Error("Failed to process refund", err, map[string]interface{}{
			"payment_id": refundRequest.PaymentID,
		})
		return utils.ErrorResponse(500, err.Error())
	}

	return utils.SuccessResponse(200, response)
}

func (h *RefundHandler) processRefundFromStepFunction(ctx context.Context, input types.StepFunctionInput) (events.APIGatewayProxyResponse, error) {
	response, err := h.service.ProcessStepFunctionRefund(ctx, &input)
	if err != nil {
		return utils.ErrorResponse(500, err.Error())
	}

	responseBody, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}

func (h *RefundHandler) getRefundStatus(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	paymentID := request.QueryStringParameters["payment_id"]
	if paymentID == "" {
		return utils.ErrorResponse(400, "payment_id is required")
	}

	response, err := h.service.GetRefundStatus(ctx, paymentID)
	if err != nil {
		return utils.ErrorResponse(404, "Payment not found")
	}

	return utils.SuccessResponse(200, response)
}

func (h *RefundHandler) checkRefundStatusFromStepFunction(ctx context.Context, input types.StepFunctionInput) (events.APIGatewayProxyResponse, error) {
	response, err := h.service.CheckStepFunctionRefundStatus(ctx, &input)
	if err != nil {
		return utils.ErrorResponse(500, err.Error())
	}

	responseBody, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}