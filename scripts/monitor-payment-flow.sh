#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default values
USER_ID="${1:-user_test_001}"
AMOUNT="${2:-50}"
CURRENCY="${3:-USD}"
ORDER_ID="${4:-order_$(date +%s)}"

echo -e "${CYAN}═══════════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}                   PAYMENT PROCESSING MONITOR                      ${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════════════════════${NC}"
echo ""

# Display payment details
echo -e "${YELLOW}📋 Payment Details:${NC}"
echo -e "   User ID: ${GREEN}$USER_ID${NC}"
echo -e "   Amount: ${GREEN}$AMOUNT $CURRENCY${NC}"
echo -e "   Order ID: ${GREEN}$ORDER_ID${NC}"
echo ""

# Check initial wallet balance
echo -e "${BLUE}💰 Checking initial wallet balance...${NC}"
INITIAL_BALANCE=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 \
    dynamodb get-item --table-name Wallets \
    --key "{\"UserID\":{\"S\":\"$USER_ID\"}}" \
    --region us-east-1 --output json 2>/dev/null | jq -r '.Item.Balance.N // "0"')

if [ "$INITIAL_BALANCE" != "0" ]; then
    echo -e "   Initial Balance: ${GREEN}$INITIAL_BALANCE${NC}"
else
    echo -e "   ${YELLOW}Wallet not found - will be created${NC}"
fi
echo ""

# Start Step Function execution
echo -e "${BLUE}🚀 Starting payment flow...${NC}"
EXECUTION_RESPONSE=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 \
    stepfunctions start-execution \
    --state-machine-arn arn:aws:states:us-east-1:000000000000:stateMachine:PaymentProcessingStateMachine \
    --input "{\"userId\":\"$USER_ID\",\"amount\":$AMOUNT,\"currency\":\"$CURRENCY\",\"metadata\":{\"orderId\":\"$ORDER_ID\"}}" \
    --region us-east-1 --output json 2>/dev/null)

EXECUTION_ARN=$(echo $EXECUTION_RESPONSE | jq -r '.executionArn')

if [ -z "$EXECUTION_ARN" ] || [ "$EXECUTION_ARN" == "null" ]; then
    echo -e "${RED}❌ Failed to start execution${NC}"
    exit 1
fi

echo -e "   Execution ID: ${CYAN}${EXECUTION_ARN##*:}${NC}"
echo ""

# Monitor execution status
echo -e "${BLUE}📊 Monitoring execution...${NC}"
echo ""

PREVIOUS_STATE=""
STEP_COUNT=0
MAX_WAIT=30
WAIT_COUNT=0

while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    # Get execution history
    EXECUTION_DETAILS=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 \
        stepfunctions describe-execution \
        --execution-arn "$EXECUTION_ARN" \
        --region us-east-1 --output json 2>/dev/null)
    
    STATUS=$(echo $EXECUTION_DETAILS | jq -r '.status')
    
    # Get current state from history
    HISTORY=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 \
        stepfunctions get-execution-history \
        --execution-arn "$EXECUTION_ARN" \
        --region us-east-1 --output json 2>/dev/null)
    
    # Find the latest state entered
    CURRENT_STATE=$(echo $HISTORY | jq -r '.events[] | select(.type == "TaskStateEntered") | .taskStateEnteredEventDetails.name' | tail -1)
    
    # Display state change
    if [ "$CURRENT_STATE" != "$PREVIOUS_STATE" ] && [ ! -z "$CURRENT_STATE" ]; then
        STEP_COUNT=$((STEP_COUNT + 1))
        
        case "$CURRENT_STATE" in
            "PaymentCreate")
                echo -e "   ${STEP_COUNT}. ${YELLOW}📝 Creating payment record...${NC}"
                ;;
            "HasSufficientBalance")
                echo -e "   ${STEP_COUNT}. ${YELLOW}💳 Checking wallet balance...${NC}"
                ;;
            "DebitWallet")
                echo -e "   ${STEP_COUNT}. ${YELLOW}💸 Debiting wallet...${NC}"
                ;;
            "ProcessPayment")
                echo -e "   ${STEP_COUNT}. ${YELLOW}⚙️  Processing payment with gateway...${NC}"
                ;;
            "WaitForProcessing")
                echo -e "   ${STEP_COUNT}. ${YELLOW}⏳ Waiting for payment confirmation...${NC}"
                ;;
            "UpdatePaymentStatus")
                echo -e "   ${STEP_COUNT}. ${YELLOW}✏️  Updating payment status...${NC}"
                ;;
            *)
                echo -e "   ${STEP_COUNT}. ${YELLOW}🔄 $CURRENT_STATE${NC}"
                ;;
        esac
        
        PREVIOUS_STATE=$CURRENT_STATE
    fi
    
    # Check if execution completed
    if [ "$STATUS" == "SUCCEEDED" ]; then
        echo ""
        echo -e "${GREEN}✅ Payment completed successfully!${NC}"
        
        # Parse output for details
        OUTPUT=$(echo $EXECUTION_DETAILS | jq -r '.output' | jq '.')
        
        # Extract payment details
        PAYMENT_ID=$(echo $OUTPUT | jq -r '.invoiceResult.Payload.data.id // "N/A"')
        EXTERNAL_ID=$(echo $OUTPUT | jq -r '.paymentResult.Payload.data.externalId // "N/A"')
        FINAL_STATUS=$(echo $OUTPUT | jq -r '.finalUpdate.Payload.data.status // "N/A"')
        
        echo ""
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}                        PAYMENT SUMMARY                            ${NC}"
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════════${NC}"
        echo -e "   Payment ID: ${GREEN}$PAYMENT_ID${NC}"
        echo -e "   External ID: ${GREEN}$EXTERNAL_ID${NC}"
        echo -e "   Status: ${GREEN}$FINAL_STATUS${NC}"
        
        # Check final wallet balance
        FINAL_BALANCE=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 \
            dynamodb get-item --table-name Wallets \
            --key "{\"UserID\":{\"S\":\"$USER_ID\"}}" \
            --region us-east-1 --output json 2>/dev/null | jq -r '.Item.Balance.N // "0"')
        
        echo -e "   Final Balance: ${GREEN}$FINAL_BALANCE${NC}"
        
        if [ "$INITIAL_BALANCE" != "0" ]; then
            DEBIT_AMOUNT=$(echo "$INITIAL_BALANCE - $FINAL_BALANCE" | bc)
            echo -e "   Amount Debited: ${YELLOW}$DEBIT_AMOUNT${NC}"
        fi
        
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════════${NC}"
        break
        
    elif [ "$STATUS" == "FAILED" ]; then
        echo ""
        echo -e "${RED}❌ Payment failed!${NC}"
        
        # Get error details
        ERROR_INFO=$(echo $HISTORY | jq -r '.events[] | select(.type == "ExecutionFailed") | .executionFailedEventDetails')
        ERROR_MSG=$(echo $ERROR_INFO | jq -r '.error // "Unknown error"')
        ERROR_CAUSE=$(echo $ERROR_INFO | jq -r '.cause // "No details available"')
        
        echo ""
        echo -e "${RED}Error: $ERROR_MSG${NC}"
        echo -e "${RED}Cause: $ERROR_CAUSE${NC}"
        
        # Check if wallet balance was affected
        if [ "$INITIAL_BALANCE" != "0" ]; then
            CURRENT_BALANCE=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 \
                dynamodb get-item --table-name Wallets \
                --key "{\"UserID\":{\"S\":\"$USER_ID\"}}" \
                --region us-east-1 --output json 2>/dev/null | jq -r '.Item.Balance.N // "0"')
            
            if [ "$CURRENT_BALANCE" == "$INITIAL_BALANCE" ]; then
                echo -e "${GREEN}✓ Wallet balance unchanged: $CURRENT_BALANCE${NC}"
            else
                echo -e "${YELLOW}⚠ Wallet balance changed: $INITIAL_BALANCE -> $CURRENT_BALANCE${NC}"
            fi
        fi
        break
        
    elif [ "$STATUS" == "RUNNING" ]; then
        # Continue monitoring
        sleep 0.5
        WAIT_COUNT=$((WAIT_COUNT + 1))
    else
        echo -e "${YELLOW}⚠ Unexpected status: $STATUS${NC}"
        break
    fi
done

if [ $WAIT_COUNT -ge $MAX_WAIT ]; then
    echo -e "${RED}⏱ Timeout waiting for execution to complete${NC}"
    exit 1
fi
