#!/bin/bash

# ============================================================
# Binance Testnet Connection Test
# ============================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  🔗 Binance Testnet Connection Test${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo ""

# Load environment variables
set -a
source .env
set +a

# Check testnet mode
if [ "$BINANCE_USE_TESTNET" != "true" ]; then
    echo -e "${YELLOW}⚠️  BINANCE_USE_TESTNET is not set to 'true'${NC}"
    echo -e "${YELLOW}   Current value: $BINANCE_USE_TESTNET${NC}"
    echo ""
    echo -e "${BLUE}   To enable testnet mode, run:${NC}"
    echo -e "${BLUE}   export BINANCE_USE_TESTNET=true${NC}"
    echo ""
fi

# Testnet URL
TESTNET_URL="https://testnet.binancefuture.com"
MAINNET_URL="https://fapi.binance.com"

echo -e "${BLUE}Mode:${NC} $([ "$BINANCE_USE_TESTNET" = "true" ] && echo -e "${GREEN}TESTNET 🧪" || echo -e "${YELLOW}MAINNET 💰" )"
echo ""

# Test 1: Ping
echo -e "${BLUE}[1/3] Testing API Ping...${NC}"
if [ "$BINANCE_USE_TESTNET" = "true" ]; then
    PING_RESPONSE=$(curl -s -X GET "$TESTNET_URL/fapi/v1/ping")
else
    PING_RESPONSE=$(curl -s -X GET "$MAINNET_URL/fapi/v1/ping")
fi

if echo "$PING_RESPONSE" | grep -q "{}"; then
    echo -e "${GREEN}   ✓ Ping successful${NC}"
else
    echo -e "${RED}   ✗ Ping failed${NC}"
    echo "   Response: $PING_RESPONSE"
fi
echo ""

# Test 2: Server Time
echo -e "${BLUE}[2/3] Testing Server Time...${NC}"
if [ "$BINANCE_USE_TESTNET" = "true" ]; then
    TIME_RESPONSE=$(curl -s -X GET "$TESTNET_URL/fapi/v1/time")
else
    TIME_RESPONSE=$(curl -s -X GET "$MAINNET_URL/fapi/v1/time")
fi

SERVER_TIME=$(echo "$TIME_RESPONSE" | grep -o '"serverTime":[0-9]*' | cut -d':' -f2)
if [ -n "$SERVER_TIME" ]; then
    HUMAN_TIME=$(date -r $((SERVER_TIME / 1000)) '+%Y-%m-%d %H:%M:%S')
    echo -e "${GREEN}   ✓ Server time: $HUMAN_TIME${NC}"
else
    echo -e "${RED}   ✗ Failed to get server time${NC}"
    echo "   Response: $TIME_RESPONSE"
fi
echo ""

# Test 3: Account Info (requires API key)
echo -e "${BLUE}[3/3] Testing Account Info...${NC}"
if [ "$BINANCE_USE_TESTNET" = "true" ]; then
    if [ -n "$BINANCE_TESTNET_API" ] && [ -n "$BINANCE_TESTNET_SECRET" ]; then
        TIMESTAMP=$(date +%s000)
        ACCOUNT_RESPONSE=$(curl -s -X GET "$TESTNET_URL/fapi/v2/account" \
            -H "X-MBX-APIKEY: $BINANCE_TESTNET_API" \
            -d "timestamp=$TIMESTAMP" \
            -d "signature=$(echo -n "timestamp=$TIMESTAMP" | openssl dgst -sha256 -hmac "$BINANCE_TESTNET_SECRET" | cut -d' ' -f2)")
    else
        ACCOUNT_RESPONSE='{"code":-2015,"msg":"Invalid API-key, IP, or permissions for action"}'
    fi
else
    if [ -n "$BINANCE_API_KEY" ] && [ -n "$BINANCE_API_SECRET" ]; then
        TIMESTAMP=$(date +%s000)
        ACCOUNT_RESPONSE=$(curl -s -X GET "$MAINNET_URL/fapi/v2/account" \
            -H "X-MBX-APIKEY: $BINANCE_API_KEY" \
            -d "timestamp=$TIMESTAMP" \
            -d "signature=$(echo -n "timestamp=$TIMESTAMP" | openssl dgst -sha256 -hmac "$BINANCE_API_SECRET" | cut -d' ' -f2)")
    else
        ACCOUNT_RESPONSE='{"code":-2015,"msg":"Invalid API-key, IP, or permissions for action"}'
    fi
fi

if echo "$ACCOUNT_RESPONSE" | grep -q '"totalWalletBalance"'; then
    echo -e "${GREEN}   ✓ Account info successful${NC}"
    WALLET=$(echo "$ACCOUNT_RESPONSE" | grep -o '"totalWalletBalance":"[^"]*"' | cut -d'"' -f4)
    echo -e "${GREEN}   ✓ Wallet balance: $WALLET${NC}"
elif echo "$ACCOUNT_RESPONSE" | grep -q '"code":-2015'; then
    echo -e "${RED}   ✗ API Key invalid or missing permissions${NC}"
    echo -e "${YELLOW}   ℹ️  See README_FIX_API.md for how to fix this${NC}"
elif echo "$ACCOUNT_RESPONSE" | grep -q '"code":-1002'; then
    echo -e "${RED}   ✗ API Key invalid signature${NC}"
    echo -e "${YELLOW}   ℹ️  Check your API secret key${NC}"
else
    echo -e "${YELLOW}   ⚠️  Unexpected response${NC}"
    echo "   Response: $ACCOUNT_RESPONSE"
fi
echo ""

# Summary
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Summary${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo ""
echo "Configuration in .env:"
echo "  BINANCE_USE_TESTNET=$BINANCE_USE_TESTNET"
if [ "$BINANCE_USE_TESTNET" = "true" ]; then
    if [ -n "$BINANCE_TESTNET_API" ]; then
        echo "  BINANCE_TESTNET_API=✓ Set (${#BINANCE_TESTNET_API} chars)"
    else
        echo "  BINANCE_TESTNET_API=✗ Not set"
    fi
else
    if [ -n "$BINANCE_API_KEY" ]; then
        echo "  BINANCE_API_KEY=✓ Set (${#BINANCE_API_KEY} chars)"
    else
        echo "  BINANCE_API_KEY=✗ Not set"
    fi
fi
echo ""
echo "Next steps:"
if [ "$BINANCE_USE_TESTNET" = "true" ]; then
    echo "  1. Start GOBOT: ./gobot"
    echo "  2. Test auto-trade: cd services/screenshot-service && node auto-trade.js 1000PEPEUSDT 10000"
else
    echo "  1. Enable testnet: export BINANCE_USE_TESTNET=true"
    echo "  2. Or set in .env: BINANCE_USE_TESTNET=true"
    echo "  3. Restart GOBOT"
fi
