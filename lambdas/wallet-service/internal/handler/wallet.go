package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/draftea-coding-challenge/lambdas/wallet-service/internal/service"
	"github.com/draftea-coding-challenge/shared/observability"
	"github.com/draftea-coding-challenge/shared/types"
	"github.com/draftea-coding-challenge/shared/utils"
)

type WalletHandler struct {
	service *service.WalletService
	logger  *observability.Logger
}

func NewWalletHandler(service *service.WalletService, logger *observability.Logger) *WalletHandler {
	return &WalletHandler{
		service: service,
		logger:  logger,
	}
}

func (h *WalletHandler) HandleRequest(ctx context.Context, request interface{}) (interface{}, error) {
	// Try to handle as API Gateway request first
	if apiReq, ok := request.(events.APIGatewayProxyRequest); ok {
		h.logger.Info("Processing API Gateway wallet request", map[string]interface{}{
			"path":   apiReq.Path,
			"method": apiReq.HTTPMethod,
		})

		switch apiReq.Path {
		case "/wallet/debit":
			return h.handleDebit(ctx, apiReq)
		case "/wallet/credit":
			return h.handleCredit(ctx, apiReq)
		case "/wallet/balance":
			return h.handleGetBalance(ctx, apiReq)
		default:
			return events.APIGatewayProxyResponse{
				StatusCode: 404,
				Body:       `{"error": "not found"}`,
			}, nil
		}
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

			switch path {
			case "/wallet/debit":
				return h.handleDebit(ctx, apiReq)
			case "/wallet/credit":
				return h.handleCredit(ctx, apiReq)
			case "/wallet/balance":
				return h.handleGetBalance(ctx, apiReq)
			default:
				return events.APIGatewayProxyResponse{
					StatusCode: 404,
					Body:       `{"error": "not found"}`,
				}, nil
			}
		}
	}

	// Check if this is a Step Function input (has action field)
	if action, ok := inputMap["action"].(string); ok {
		return h.handleStepFunctionInput(ctx, inputMap, action)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 400,
		Body:       `{"error": "unsupported request type"}`,
	}, nil
}

func (h *WalletHandler) handleDebit(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var req service.DebitRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		h.logger.Error("Failed to unmarshal request", err, nil)
		return utils.ErrorResponse(400, "invalid request format")
	}

	h.logger.Info("Processing debit request", map[string]interface{}{
		"userId":    req.UserID,
		"amount":    req.Amount,
		"paymentId": req.PaymentID,
	})

	wallet, err := h.service.DebitWallet(ctx, req)
	if err != nil {
		if err.Error()[:12] == "insufficient" {
			return utils.ErrorResponse(400, err.Error())
		}
		h.logger.Error("Failed to debit wallet", err, nil)
		return utils.ErrorResponse(500, "failed to debit wallet")
	}

	return utils.SuccessResponse(200, wallet)
}

func (h *WalletHandler) handleCredit(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var req service.CreditRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		h.logger.Error("Failed to unmarshal request", err, nil)
		return utils.ErrorResponse(400, "invalid request format")
	}

	h.logger.Info("Processing credit request", map[string]interface{}{
		"userId":    req.UserID,
		"amount":    req.Amount,
		"paymentId": req.PaymentID,
		"reason":    req.RefundReason,
	})

	wallet, err := h.service.CreditWallet(ctx, req)
	if err != nil {
		h.logger.Error("Failed to credit wallet", err, nil)
		return utils.ErrorResponse(500, "failed to credit wallet")
	}

	return utils.SuccessResponse(200, wallet)
}

func (h *WalletHandler) handleStepFunctionInput(ctx context.Context, input map[string]interface{}, action string) (interface{}, error) {
	h.logger.Info("Processing Step Function wallet request", map[string]interface{}{
		"action": action,
		"input":  input,
	})

	switch action {
	case "check_balance":
		userID, _ := input["userId"].(string)
		amount, _ := input["amount"].(float64)

		wallet, err := h.service.GetBalance(ctx, userID)
		if err != nil {
			h.logger.Error("Failed to get balance", err, nil)
			return map[string]interface{}{
				"statusCode": 500,
				"body":       `{"error": "failed to get balance"}`,
			}, nil
		}

		// Check if balance is sufficient
		hasSufficient := wallet.Balance >= amount
		return types.LambdaResponse{
			Success: hasSufficient,
			Data: map[string]interface{}{
				"balance":              wallet.Balance,
				"hasSufficientBalance": hasSufficient,
			},
		}, nil

	case "debit":
		userID, _ := input["userId"].(string)
		amount, _ := input["amount"].(float64)
		paymentID, _ := input["paymentId"].(string)

		req := service.DebitRequest{
			UserID:    userID,
			Amount:    amount,
			PaymentID: paymentID,
		}

		wallet, err := h.service.DebitWallet(ctx, req)
		if err != nil {
			h.logger.Error("Failed to debit wallet", err, nil)
			return types.LambdaResponse{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		return types.LambdaResponse{
			Success: true,
			Data:    wallet,
		}, nil

	case "credit":
		userID, _ := input["userId"].(string)
		amount, _ := input["amount"].(float64)
		paymentID, _ := input["paymentId"].(string)
		reason, _ := input["reason"].(string)

		req := service.CreditRequest{
			UserID:       userID,
			Amount:       amount,
			PaymentID:    paymentID,
			RefundReason: reason,
		}

		_, err := h.service.CreditWallet(ctx, req)
		if err != nil {
			h.logger.Error("Failed to credit wallet", err, nil)
			return map[string]interface{}{
				"statusCode": 500,
				"body":       fmt.Sprintf(`{"error": "%s"}`, err.Error()),
			}, nil
		}

		return map[string]interface{}{
			"statusCode": 200,
			"body":       `{"success": true}`,
		}, nil

	default:
		return map[string]interface{}{
			"statusCode": 400,
			"body":       fmt.Sprintf(`{"error": "unknown action: %s"}`, action),
		}, nil
	}
}

func (h *WalletHandler) handleGetBalance(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID := request.QueryStringParameters["userId"]
	if userID == "" {
		return utils.ErrorResponse(400, "userId is required")
	}

	wallet, err := h.service.GetBalance(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get balance", err, map[string]interface{}{
			"userId": userID,
		})
		return utils.ErrorResponse(500, "failed to get balance")
	}

	body, _ := json.Marshal(map[string]interface{}{
		"userId":  wallet.UserID,
		"balance": wallet.Balance,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(body),
	}, nil
}
