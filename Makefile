# Payment Processing System Makefile
# Commands for development, testing and deployment

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# ==================== SETUP ====================

.PHONY: setup
setup: ## Set executable permissions on all scripts
	@echo "🔧 Setting up executable permissions..."
	@chmod +x scripts/*.sh 2>/dev/null || true
	@echo "✅ Scripts are now executable"

.PHONY: init
init: setup ## Initialize LocalStack and DynamoDB
	@echo "🚀 Initializing LocalStack..."
	@cd lambdas/invoice-processor && go mod tidy
	@cd lambdas/wallet-service && go mod tidy
	@cd lambdas/payments-adapter && go mod tidy
	@cd lambdas/refund-service && go mod tidy
	@cd shared && go mod tidy
	@cd mock-gateway && go mod tidy
	@cd tests && go mod tidy
	@echo "✅ Dependencies installed"

.PHONY: setup-local
setup-local: ## Setup complete local environment
	@echo "🚀 Setting up local environment..."
	@./scripts/init-local.sh
	@echo "✅ Local environment ready"

.PHONY: create-tables
create-tables: ## Create DynamoDB tables locally
	@echo "📊 Creating DynamoDB tables..."
	@./scripts/create-tables.sh
	@echo "✅ Tables created"

.PHONY: seed-data
seed-data: ## Load test data into DynamoDB
	@echo "🌱 Loading test data..."
	@./scripts/seed-data.sh
	@echo "✅ Data loaded"

# ==================== BUILD ====================

.PHONY: build
build: ## Build all Lambda services
	@echo "🔨 Building services..."
	@./scripts/build-lambdas.sh
	@echo "✅ Build completed"

.PHONY: build-invoice
build-invoice: ## Build Invoice Processor
	@echo "🔨 Building Invoice Processor..."
	@cd lambdas/invoice-processor && GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/main.go
	@echo "✅ Invoice Processor built"

.PHONY: build-wallet
build-wallet: ## Build Wallet Service
	@echo "🔨 Building Wallet Service..."
	@cd lambdas/wallet-service && GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/main.go
	@echo "✅ Wallet Service built"

.PHONY: build-payments
build-payments: ## Build Payments Adapter
	@echo "🔨 Building Payments Adapter..."
	@cd lambdas/payments-adapter && GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/main.go
	@echo "✅ Payments Adapter built"

.PHONY: build-refund
build-refund: ## Build Refund Service
	@echo "🔨 Building Refund Service..."
	@cd lambdas/refund-service && GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/main.go
	@echo "✅ Refund Service built"

# ==================== DOCKER ====================

.PHONY: docker-up
docker-up: ## Start Docker services (DynamoDB, etc)
	@echo "🐳 Starting Docker services..."
	@docker-compose up -d
	@echo "✅ Docker services started"

.PHONY: docker-down
docker-down: ## Stop Docker services
	@echo "🐳 Stopping Docker services..."
	@docker-compose down
	@echo "✅ Docker services stopped"

.PHONY: docker-logs
docker-logs: ## Show container logs
	@docker-compose logs -f

# ==================== GATEWAY ====================

.PHONY: start-gateway
start-gateway: ## Start mock payment gateway
	@echo "🌐 Starting Mock Gateway..."
	@cd mock-gateway && go run main.go

.PHONY: gateway-background
gateway-background: ## Start gateway in background
	@echo "🌐 Starting Mock Gateway in background..."
	@cd mock-gateway && nohup go run main.go > gateway.log 2>&1 &
	@echo "✅ Gateway running in background (PID saved)"

# ==================== DEPLOY ====================

.PHONY: all
all: clean setup deploy ## Build and deploy everything

.PHONY: deploy-lambdas
deploy-lambdas: build ## Deploy Lambda functions to LocalStack
	@echo "🚀 Deploying Lambda functions..."
	@./scripts/deploy-lambdas.sh
	@echo "✅ Lambda deployment completed"

.PHONY: create-state-machine
create-state-machine: ## Create Step Functions state machine
	@echo "📊 Creating Step Functions state machine..."
	@AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws stepfunctions create-state-machine \
		--name PaymentProcessingStateMachine \
		--definition file://state-machine/stateMachine.json \
		--role-arn arn:aws:iam::000000000000:role/stepfunctions-role \
		--endpoint-url http://localhost:4566 \
		--region us-east-1 2>/dev/null && echo "✅ State machine created" || echo "⚠️  State machine already exists"

.PHONY: update-state-machine
update-state-machine: ## Update Step Functions state machine
	@echo "🔄 Updating Step Functions state machine..."
	@AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws stepfunctions update-state-machine \
		--state-machine-arn arn:aws:states:us-east-1:000000000000:stateMachine:PaymentProcessingStateMachine \
		--definition file://state-machine/stateMachine.json \
		--endpoint-url http://localhost:4566 \
		--region us-east-1 && echo "✅ State machine updated" || echo "❌ Failed to update state machine"

.PHONY: full-setup
full-setup: ## Complete setup: Docker, build, deploy everything
	@echo "🎯 Starting complete setup..."
	@$(MAKE) docker-up
	@sleep 5
	@$(MAKE) init
	@$(MAKE) build
	@./scripts/create-tables.sh
	@$(MAKE) deploy-lambdas
	@$(MAKE) create-state-machine
	@echo "✅ Full setup completed! System ready for testing."

.PHONY: restart-all
restart-all: ## Restart everything from scratch
	@echo "🔄 Restarting all services..."
	@$(MAKE) clean
	@$(MAKE) docker-down
	@sleep 2
	@$(MAKE) full-setup
	@echo "✅ All services restarted successfully!"

.PHONY: deploy-local
deploy-local: build ## Deploy to local environment with SAM
	@echo "🚀 Deploying with SAM local..."
	@./scripts/deploy-local.sh
	@echo "✅ Local deployment completed"

.PHONY: sam-local
sam-local: ## Start SAM local API
	@echo "🚀 Starting SAM local API..."
	@sam local start-api --env-vars env.json --docker-network payment-network

.PHONY: deploy-aws
deploy-aws: build ## Deploy to AWS
	@echo "☁️  Deploying to AWS..."
	@sam deploy --guided
	@echo "✅ AWS deployment completed"

# ==================== TESTS ====================

.PHONY: test
test: ## Run all tests
	@echo "🧪 Running all tests..."
	@$(MAKE) test-unit
	@$(MAKE) test-lambdas
	@echo "✅ All tests passed"

.PHONY: test-unit
test-unit: ## Run unit tests
	@echo "🧪 Running unit tests..."
	@cd lambdas/invoice-processor && go test ./...
	@cd lambdas/wallet-service && go test ./...
	@cd lambdas/payments-adapter && go test ./...
	@cd lambdas/refund-service && go test ./...
	@cd shared && go test ./...
	@echo "✅ Unit tests completed"

.PHONY: test-lambdas
test-lambdas: ## Test Lambda functions with curl
	@echo "🧪 Testing Lambda functions..."
	@$(MAKE) test-invoice-lambda
	@$(MAKE) test-wallet-lambda
	@$(MAKE) test-payment-adapter-lambda
	@echo "✅ Lambda tests completed"

.PHONY: test-invoice-lambda
test-invoice-lambda: ## Test invoice processor Lambda
	@echo "📄 Testing Invoice Processor Lambda..."
	@curl -s -X POST http://localhost:4566/2015-03-31/functions/invoice-processor/invocations \
		-H "Content-Type: application/json" \
		-d '{"httpMethod":"GET","path":"/health"}' | jq
	@echo "✅ Invoice Lambda test completed"

.PHONY: test-wallet-lambda
test-wallet-lambda: ## Test wallet service Lambda
	@echo "💰 Testing Wallet Service Lambda..."
	@curl -s -X POST http://localhost:4566/2015-03-31/functions/wallet-service/invocations \
		-H "Content-Type: application/json" \
		-d '{"httpMethod":"GET","path":"/health"}' | jq
	@echo "✅ Wallet Lambda test completed"

.PHONY: test-payment-adapter-lambda
test-payment-adapter-lambda: ## Test payment adapter Lambda
	@echo "💳 Testing Payment Adapter Lambda..."
	@curl -s -X POST http://localhost:4566/2015-03-31/functions/payments-adapter/invocations \
		-H "Content-Type: application/json" \
		-d '{"httpMethod":"GET","path":"/health"}' | jq
	@echo "✅ Payment Adapter Lambda test completed"

.PHONY: test-e2e
test-e2e: setup ## Run complete E2E payment flow test suite
	@echo "🧪 Running E2E Payment Flow Tests..."
	@echo "═══════════════════════════════════════════════════════════════════"
	@echo "Test 1: Credit wallet with initial funds"
	@$(MAKE) test-curl-wallet-credit
	@sleep 1
	@echo ""
	@echo "Test 2: Small payment (should succeed)"
	@./scripts/monitor-payment-flow.sh user_test_001 20 USD e2e_test_1
	@sleep 2
	@echo ""
	@echo "Test 3: Large payment (should fail - insufficient funds)"
	@./scripts/monitor-payment-flow.sh user_test_001 10000 USD e2e_test_2
	@echo ""
	@echo "✅ E2E tests completed"

.PHONY: test-create-payment
test-create-payment: ## Test payment creation
	@echo "Testing wallet credit..."
	@curl -s -X POST http://localhost:4566/2015-03-31/functions/wallet-service/invocations \
		-H "Content-Type: application/json" \
		-d '{"httpMethod":"POST","path":"/wallet/credit","body":"{\"userId\":\"user_test_001\",\"amount\":2000,\"paymentId\":\"payment_001\",\"reason\":\"initial_deposit\"}"}' | jq

test-curl-wallet-debit:
	@echo "Testing wallet debit..."
	@curl -s -X POST http://localhost:4566/2015-03-31/functions/wallet-service/invocations \
		-H "Content-Type: application/json" \
		-d '{"httpMethod":"POST","path":"/wallet/debit","body":"{\"userId\":\"user_test_001\",\"amount\":100,\"paymentId\":\"payment_002\"}"}' | jq

test-curl-payment-process:
	@echo "Testing payment processing via Lambda..."
	@curl -s -X POST http://localhost:4566/2015-03-31/functions/payments-adapter/invocations \
		-H "Content-Type: application/json" \
		-d '{"httpMethod":"POST","path":"/process","body":"{\"paymentId\":\"test_payment_001\",\"amount\":100,\"currency\":\"USD\"}"}' | jq

.PHONY: test-payment
test-payment: setup ## Test successful payment with monitor (50 USD)
	@./scripts/monitor-payment-flow.sh user_test_001 50 USD order_success_$(shell date +%s)

.PHONY: test-payment-custom
test-payment-custom: setup ## Test payment with custom parameters
	@./scripts/monitor-payment-flow.sh $(USER_ID) $(AMOUNT) $(CURRENCY) $(ORDER_ID)

.PHONY: test-payment-fail
test-payment-fail: setup ## Test payment with insufficient balance (5000 USD)
	@./scripts/monitor-payment-flow.sh user_test_001 5000 USD order_fail_$(shell date +%s)

.PHONY: test-payment-small
test-payment-small: setup ## Test small payment (10 USD)
	@./scripts/monitor-payment-flow.sh user_test_001 10 USD order_small_$(shell date +%s)

.PHONY: test-payment-large
test-payment-large: setup ## Test large payment (500 USD)
	@./scripts/monitor-payment-flow.sh user_test_001 500 USD order_large_$(shell date +%s)

.PHONY: test-check-wallet-balance
test-check-wallet-balance: ## Check wallet balance for user_test_001
	@echo "Checking wallet balance for user_test_001..."
	@AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 \
		dynamodb get-item --table-name Wallets \
		--key '{"UserID":{"S":"user_test_001"}}' \
		--region us-east-1 --output json | jq '.Item.Balance.N'

.PHONY: test-stepfunction
test-stepfunction: ## Test Step Function execution
	@./scripts/test-refund.sh
	@echo "✅ Refund processed successfully"

.PHONY: test-circuit-breaker
test-circuit-breaker: ## Test circuit breaker
	@echo "🔌 Testing circuit breaker..."
	@./scripts/test-circuit-breaker.sh
	@echo "✅ Circuit breaker working"

# ==================== MONITORING ====================

.PHONY: logs
logs: ## Show logs for all services
	@echo "📋 Showing logs..."
	@sam logs --tail

.PHONY: logs-invoice
logs-invoice: ## Invoice Processor logs
	@sam logs -n InvoiceProcessorFunction --tail

.PHONY: logs-wallet
logs-wallet: ## Wallet Service logs
	@sam logs -n WalletServiceFunction --tail

.PHONY: logs-payments
logs-payments: ## Payments Adapter logs
	@sam logs -n PaymentsAdapterFunction --tail

.PHONY: logs-refund
logs-refund: ## Refund Service logs
	@sam logs -n RefundServiceFunction --tail

# ==================== CLEANUP ====================

.PHONY: clean
clean: ## Clean generated files
	@echo "🧹 Cleaning files..."
	@rm -f lambdas/*/bootstrap
	@rm -rf .aws-sam
	@rm -f gateway.log
	@echo "✅ Cleanup completed"

.PHONY: clean-data
clean-data: ## Clean DynamoDB local data
	@echo "🧹 Cleaning DynamoDB data..."
	@aws dynamodb delete-table --table-name Payments --endpoint-url http://localhost:8000 2>/dev/null || true
	@aws dynamodb delete-table --table-name Wallets --endpoint-url http://localhost:8000 2>/dev/null || true
	@aws dynamodb delete-table --table-name PaymentEvents --endpoint-url http://localhost:8000 2>/dev/null || true
	@aws dynamodb delete-table --table-name Invoices --endpoint-url http://localhost:8000 2>/dev/null || true
	@echo "✅ Data deleted"

.PHONY: reset
reset: clean clean-data ## Complete environment reset
	@echo "🔄 Complete reset..."
	@$(MAKE) docker-down
	@$(MAKE) docker-up
	@sleep 3
	@$(MAKE) create-tables
	@$(MAKE) seed-data
	@echo "✅ Reset completed"

# ==================== COMPLETE FLOWS ====================

.PHONY: run-local
run-local: ## Run complete system locally
	@echo "🎯 Starting complete system..."
	@$(MAKE) docker-up
	@sleep 3
	@$(MAKE) create-tables
	@$(MAKE) build
	@$(MAKE) gateway-background
	@$(MAKE) sam-local

.PHONY: test-all-scenarios
test-all-scenarios: ## Run all test scenarios
	@echo "🎯 Running all scenarios..."
	@echo "\n1️⃣ Happy Path - Successful Payment"
	@$(MAKE) test-payment
	@sleep 2
	@echo "\n2️⃣ Insufficient Balance"
	@curl -X POST http://localhost:3000/wallet/debit \
		-H "Content-Type: application/json" \
		-d '{"user_id":"user_poor","payment_id":"pay_001","amount":10000}'
	@sleep 2
	@echo "\n3️⃣ Gateway Timeout"
	@curl -X POST http://localhost:3000/payment/process \
		-H "Content-Type: application/json" \
		-H "X-API-Key: test-key" \
		-H "X-Simulate-Timeout: true" \
		-d '{"id":"pay_timeout","amount":100}'
	@sleep 2
	@echo "\n4️⃣ Circuit Breaker"
	@$(MAKE) test-circuit-breaker
	@sleep 2
	@echo "\n5️⃣ Refund"
	@$(MAKE) test-refund
	@echo "\n✅ All scenarios completed"

.PHONY: quick-start
quick-start: ## Quick start for development
	@echo "⚡ Quick start..."
	@$(MAKE) install
	@$(MAKE) docker-up
	@sleep 3
	@$(MAKE) create-tables
	@$(MAKE) seed-data
	@$(MAKE) build
	@echo "✅ System ready for development"
	@echo "📝 Run 'make run-local' to start services"
	@echo "🧪 Run 'make test-all-scenarios' to test all flows"

# ==================== VALIDATION ====================

.PHONY: validate
validate: ## Validate SAM template
	@echo "✔️  Validating SAM template..."
	@sam validate
	@echo "✅ Template valid"

.PHONY: lint
lint: ## Run Go linters
	@echo "🔍 Running linters..."
	@golangci-lint run ./...
	@echo "✅ Code clean"

.PHONY: fmt
fmt: ## Format Go code
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@echo "✅ Code formatted"