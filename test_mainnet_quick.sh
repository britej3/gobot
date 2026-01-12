#!/bin/bash

# Quick Mainnet Test - Debug Mode

# Extract API credentials
API_KEY=$(awk -F'=' '/^BINANCE_API_KEY=/ {print $2}' .env | tr -d '"' | tr -d ' ')
API_SECRET=$(awk -F'=' '/^BINANCE_API_SECRET=/ {print $2}' .env | tr -d '"' | tr -d ' ')

echo "Using API Key: ${API_KEY:0:10}...${API_KEY: -10}"
echo "Using Secret: ${API_SECRET:0:10}...${API_SECRET: -10}"
echo ""

# Test server time
echo "Testing server time endpoint..."
curl -s "https://fapi.binance.com/fapi/v1/time"
echo ""
echo ""

# Test WITH proper authentication
echo "Testing account endpoint with authentication..."
TIMESTAMP=$(date +%s000)
QUERY="timestamp=${TIMESTAMP}"
SIGNATURE=$(echo -n "$QUERY" | openssl dgst -sha256 -hmac "$API_SECRET" | sed 's/^.* //')

echo "Timestamp: $TIMESTAMP"
echo "Signature: ${SIGNATURE:0:20}..."
echo ""
echo "Full URL: https://fapi.binance.com/fapi/v2/account?${QUERY}&signature=${SIGNATURE}"
echo ""
echo "Response:"
curl -v -H "X-MBX-APIKEY: $API_KEY" "https://fapi.binance.com/fapi/v2/account?${QUERY}&signature=${SIGNATURE}" 2>&1

echo ""