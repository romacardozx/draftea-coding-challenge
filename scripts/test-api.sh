#!/bin/bash

echo "🧪 Testing Payment Processing API"
echo "================================="

API_URL=${1:-http://localhost:3001}
USER_ID="user-123"
PAYMENT_ID="payment-test-$(date +%s)"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 1. Create/Credit wallet
echo "1. Creating wallet with initial balance..."
curl -X POST "$API_URL/wallet/credit" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"amount\": 100.00,
    \"payment_id\": \"initial-credit\"
  }" | jq .

# 2. Check wallet balance
echo -e "\n2. Checking wallet balance..."
curl -X GET "$API_URL/wallet/balance/$USER_ID" | jq .

# 3. Create payment request
echo -e "\n3. Creating payment request..."
PAYMENT_RESPONSE=$(curl -X POST "$API_URL/payments" \
  -H "Content-Type: application/json" \
  -d "{
    \"payment_id\": \"$PAYMENT_ID\",
    \"user_id\": \"$USER_ID\",
    \"amount\": 50.00,
    \"currency\": \"USD\",
    \"description\": \"Test payment\"
  }")

echo "$PAYMENT_RESPONSE" | jq .

# 4. Check payment status
echo -e "\n4. Checking payment status..."
sleep 2
curl -X GET "$API_URL/payments/$PAYMENT_ID" | jq .

# 5. Check wallet balance after payment
echo -e "\n5. Checking wallet balance after payment..."
curl -X GET "$API_URL/wallet/balance/$USER_ID" | jq .

# 6. Process refund
echo -e "\n6. Processing refund..."
curl -X POST "$API_URL/refunds" \
  -H "Content-Type: application/json" \
  -d "{
    \"payment_id\": \"$PAYMENT_ID\",
    \"amount\": 50.00,
    \"reason\": \"Test refund\"
  }" | jq .

# 7. Check final wallet balance
echo -e "\n7. Checking final wallet balance..."
sleep 1
curl -X GET "$API_URL/wallet/balance/$USER_ID" | jq .

echo -e "\n${GREEN}✅ API tests completed!${NC}"
