#!/bin/bash

# Binance API Test - Get Account Information
# This script demonstrates how to properly call /fapi/v2/account with authentication

set -e

# Load environment variables from .env file
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Configuration
API_KEY="${BINANCE_API_KEY}"
API_SECRET="${BINANCE_API_SECRET}"
USE_TESTNET="${BINANCE_USE_TESTNET:-false}"

# Determine base URL
if [ "$USE_TESTNET" = "true" ]; then
    BASE_URL="https://testnet.binancefuture.com"
    echo "🧪 Using TESTNET environment"
else
    BASE_URL="https://fapi.binance.com"
    echo "🚨 Using MAINNET environment (REAL MONEY)!"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 Testing Binance API Authentication"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if API keys are set
if [ -z "$API_KEY" ] || [ -z "$API_SECRET" ]; then
    echo "❌ Error: BINANCE_API_KEY or BINANCE_API_SECRET not set in .env file"
    exit 1
fi

echo "✓ API Key configured: ${API_KEY:0:10}...${API_KEY: -10}"
echo "✓ API Secret configured: ${API_SECRET:0:10}...${API_SECRET: -10}"
echo ""

# Test 1: Server Time (no authentication required)
echo "📡 Test 1: Checking server time (no auth required)..."
SERVER_TIME=$(curl -s "${BASE_URL}/fapi/v1/time")
echo "Server response: $SERVER_TIME"
echo ""

# Test 2: Exchange Info (no authentication required)
echo "📡 Test 2: Checking exchange info (no auth required)..."
EXCHANGE_INFO=$(curl -s "${BASE_URL}/fapi/v1/exchangeInfo" | head -c 200)
echo "Server response: ${EXCHANGE_INFO}..."
echo ""

# Test 3: Account Information (requires authentication)
echo "🔐 Test 3: Fetching account information (WITH authentication)..."
echo ""

# Generate timestamp (milliseconds)
TIMESTAMP=$(date +%s000)

echo "⏱️  Generated timestamp: $TIMESTAMP"

# Create query string
QUERY_STRING="timestamp=${TIMESTAMP}"

echo "📝 Query string: $QUERY_STRING"

# Generate signature (HMAC SHA256)
SIGNATURE=$(echo -n "$QUERY_STRING" | openssl dgst -sha256 -hmac "$API_SECRET" | sed 's/^.* //')

echo "🔑 Generated signature: ${SIGNATURE:0:20}..."
echo ""

# Make the API call
echo "📨 Sending request to ${BASE_URL}/fapi/v2/account"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

curl -s -H "X-MBX-APIKEY: $API_KEY" "${BASE_URL}/fapi/v2/account?${QUERY_STRING}&signature=${SIGNATURE}" | jq .

EXIT_CODE=$?

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Request completed successfully"
else
    echo "❌ jq not installed or JSON parsing failed"
    echo "Install jq with: brew install jq (macOS) or apt-get install jq (Linux)"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "💡 Note: If you get signature/permission errors:"
echo "  1. Verify your API key and secret in .env file"
echo "  2. Check that 'Enable Futures' is enabled in Binance API settings"
echo "  3. Verify IP restrictions (if any) include your current IP"
echo "  4. Ensure you're using the correct environment (testnet vs mainnet)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
