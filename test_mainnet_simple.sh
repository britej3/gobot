#!/bin/bash

# Simple Mainnet Test - bypass .env parsing issues
# Loads keys directly from .env file using awk

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Binance Mainnet API Test${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Extract values using awk (more reliable)
API_KEY=$(awk -F'=' '/^BINANCE_API_KEY=/ {print $2}' .env | tr -d '"' | tr -d ' ')
API_SECRET=$(awk -F'=' '/^BINANCE_API_SECRET=/ {print $2}' .env | tr -d '"' | tr -d ' ')
TESTNET_SETTING=$(awk -F'=' '/^BINANCE_USE_TESTNET=/ {print $2}' .env | tr -d '"' | tr -d ' ')

echo -e "${YELLOW}📋 Current Settings:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "BINANCE_USE_TESTNET: ${TESTNET_SETTING:-"not set"}"
echo -e "BINANCE_API_KEY:     ${GREEN}${API_KEY:0:10}...${API_KEY: -10}${NC}"
echo -e "BINANCE_API_SECRET:  ${GREEN}${API_SECRET:0:10}...${API_SECRET: -10}${NC}"
echo ""

# Check if keys exist
if [ -z "$API_KEY" ] || [ -z "$API_SECRET" ]; then
    echo -e "${RED}❌ ERROR: API keys not found in .env${NC}"
    exit 1
fi

# API endpoint
BASE_URL="https://fapi.binance.com"
echo -e "API Endpoint: ${BLUE}$BASE_URL${NC}"
echo ""

# Test 1: Server Time (no auth)
echo -e "${YELLOW}📡 Test 1: Server Time (No Auth Required)${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
SERVER_TIME=$(curl -s "${BASE_URL}/fapi/v1/time")
if command -v jq &> /dev/null; then
    echo "Response: $SERVER_TIME" | jq .
else
    echo "Response: $SERVER_TIME"
    echo "(Install jq for better formatting: brew install jq)"
fi
echo ""

echo -e "${YELLOW}📡 Test 2: Account Info (WITH Authentication)${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Generating signature..."
echo ""

# Generate timestamp
TIMESTAMP=$(date +%s000)
echo "⏱️  Timestamp: $TIMESTAMP"

# Create query string
QUERY="timestamp=${TIMESTAMP}"
echo "📝 Query String: $QUERY"

# Generate signature
SIGNATURE=$(echo -n "$QUERY" | openssl dgst -sha256 -hmac "$API_SECRET" | sed 's/^.* //')
echo "🔑 Signature: ${SIGNATURE:0:20}..."
echo ""

# Make the API call
echo "📨 Sending request to ${BASE_URL}/fapi/v2/account"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

RESPONSE=$(curl -s -H "X-MBX-APIKEY: $API_KEY" "${BASE_URL}/fapi/v2/account?${QUERY}&signature=${SIGNATURE}")

# Check response
if echo "$RESPONSE" | grep -q '"code"'; then
    ERROR_CODE=$(echo "$RESPONSE" | grep -o '"code":[^,]*' | grep -o '[0-9\-]*')
    ERROR_MSG=$(echo "$RESPONSE" | grep -o '"msg":"[^"]*"' | sed 's/"msg":"\(.*\)"/\1/')
    
    echo -e "${RED}❌ API Error Code: $ERROR_CODE${NC}"
    echo -e "${RED}❌ Error Message: $ERROR_MSG${NC}"
    echo ""
    
    case $ERROR_CODE in
        -1022)
            echo -e "${YELLOW}💡 This is an IP restriction issue${NC}"
            echo -e "${YELLOW}→ Go to Binance API settings and add your IP${NC}"
            ;;
        -2015)
            echo -e "${YELLOW}💡 API Key is invalid or doesn't have Futures enabled${NC}"
            echo -e "${YELLOW}→ Check your API key permissions in Binance${NC}"
            ;;
        -2014)
            echo -e "${YELLOW}💡 Signature is invalid - check your API secret${NC}"
            ;;
        -1102)
            echo -e "${YELLOW}💡 Missing timestamp parameter (shouldn't happen)${NC}"
            ;;
        *)
            echo -e "${YELLOW}💡 Unknown error - check Binance API documentation${NC}"
            ;;
    esac
    
    echo ""
    echo -e "${YELLOW}📋 Your current IP: $(curl -s ifconfig.me)${NC}"
    echo ""
    echo -e "${YELLOW}📝 Next Steps:${NC}"
    echo "1. Go to https://www.binance.com/en/my/settings/api-management"
    echo "2. Find your API key and check permissions"
    echo "3. Verify 'Enable Futures' is turned ON"
    echo "4. Check IP restrictions - add your IP if needed"
    echo "5. If keys are wrong, create new ones and update .env"
    
elif echo "$RESPONSE" | grep -q '"totalWalletBalance"'; then
    echo -e "${GREEN}✅ SUCCESS! Account data retrieved:${NC}"
    echo ""
    if command -v jq &> /dev/null; then
        BALANCE=$(echo "$RESPONSE" | jq -r '.totalWalletBalance')
        AVAILABLE=$(echo "$RESPONSE" | jq -r '.availableBalance')
        UNREALIZED=$(echo "$RESPONSE" | jq -r '.totalUnrealizedProfit')
        
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo -e "💰 Total Wallet Balance: ${GREEN}$BALANCE USDT${NC}"
        echo -e "💵 Available Balance:    ${GREEN}$AVAILABLE USDT${NC}"
        echo -e "📊 Unrealized PnL:       ${GREEN}$UNREALIZED USDT${NC}"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    else
        echo "$RESPONSE" | head -c 300
    fi
else
    echo -e "${YELLOW}⚠️  Unexpected response format${NC}"
    echo "$RESPONSE" | head -c 200
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""