#!/bin/bash

# Mainnet-specific Binance API Diagnostic Tool
# This script thoroughly tests mainnet connectivity and identifies specific issues

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | grep '=' | xargs)
fi

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Binance Mainnet API Diagnostic Tool${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Configuration
API_KEY="${BINANCE_API_KEY}"
API_SECRET="${BINANCE_API_SECRET}"
TESTNET_API="${BINANCE_TESTNET_API}"
TESTNET_SECRET="${BINANCE_TESTNET_SECRET}"
USE_TESTNET="${BINANCE_USE_TESTNET:-false}"

# URLs
MAINNET_URL="https://fapi.binance.com"
TESTNET_URL="https://testnet.binancefuture.com"

echo -e "${YELLOW}📋 Configuration Check:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check which credentials are being used
echo -e "BINANCE_USE_TESTNET: ${USE_TESTNET}"
if [ "$USE_TESTNET" = "true" ]; then
    echo -e "${RED}⚠️  WARNING: Currently set to TESTNET${NC}"
    echo -e "${YELLOW}To test mainnet, set BINANCE_USE_TESTNET=false in .env${NC}"
    echo ""
fi

echo ""
echo -e "${YELLOW}🔑 API Credentials:${NC}"
if [ -n "$API_KEY" ] && [ -n "$API_SECRET" ]; then
    echo -e "Mainnet API Key: ${GREEN}✓ Configured${NC} (${API_KEY:0:10}...${API_KEY: -10})"
else
    echo -e "Mainnet API Key: ${RED}✗ Missing or empty${NC}"
fi

if [ -n "$TESTNET_API" ] && [ -n "$TESTNET_SECRET" ]; then
    echo -e "Testnet API Key: ${GREEN}✓ Configured${NC} (${TESTNET_API:0:10}...${TESTNET_API: -10})"
else
    echo -e "Testnet API Key: ${YELLOW}○ Not configured${NC}"
fi

echo ""
echo -e "${YELLOW}🌐 Your IP Address:${NC}"
CURRENT_IP=$(curl -s ifconfig.me)
echo -e "Current IP: ${BLUE}$CURRENT_IP${NC}"
echo ""

echo -e "${YELLOW}📡 Phase 1: Testing Basic Connectivity${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Test 1: Mainnet server time
echo -e "Test 1: Fetching mainnet server time..."
MAINNET_TIME=$(curl -s "${MAINNET_URL}/fapi/v1/time")
if echo "$MAINNET_TIME" | grep -q "serverTime"; then
    echo -e "${GREEN}✓ Mainnet connectivity: OK${NC}"
    SERVER_TIME=$(echo "$MAINNET_TIME" | jq -r '.serverTime' 2>/dev/null || echo "N/A")
    echo -e "Server time: $SERVER_TIME"
else
    echo -e "${RED}✗ Mainnet connectivity: FAILED${NC}"
    echo "Response: $MAINNET_TIME"
fi
echo ""

# Test 2: Testnet server time (for comparison)
echo -e "Test 2: Fetching testnet server time..."
TESTNET_TIME=$(curl -s "${TESTNET_URL}/fapi/v1/time")
if echo "$TESTNET_TIME" | grep -q "serverTime"; then
    echo -e "${GREEN}✓ Testnet connectivity: OK${NC}"
else
    echo -e "${YELLOW}○ Testnet connectivity: Failed${NC}"
fi
echo ""

echo -e "${YELLOW}📡 Phase 2: Testing Authenticated Endpoints${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Function to test authenticated endpoint
test_authenticated_endpoint() {
    local api_key=$1
    local api_secret=$2
    local base_url=$3
    local env_name=$4
    
    # Check if we have credentials
    if [ -z "$api_key" ] || [ -z "$api_secret" ]; then
        echo -e "${YELLOW}○ Skipping $env_name test - credentials not configured${NC}"
        return 1
    fi
    
    echo -e "Testing ${BLUE}$envName${NC} with authentication..."
    
    # Generate timestamp and signature
    local timestamp=$(date +%s000)
    local query_string="timestamp=${timestamp}"
    local signature=$(echo -n "$query_string" | openssl dgst -sha256 -hmac "$api_secret" | sed 's/^.* //')
    
    # Make the API call
    local response=$(curl -s -H "X-MBX-APIKEY: $api_key" "${base_url}/fapi/v2/account?${query_string}&signature=${signature}")
    
    # Check for errors
    if echo "$response" | grep -q '"code"'; then
        local error_code=$(echo "$response" | jq -r '.code' 2>/dev/null || echo "unknown")
        local error_msg=$(echo "$response" | jq -r '.msg' 2>/dev/null || echo "unknown error")
        
        case $error_code in
            -1022)
                echo -e "${RED}✗ $env_name Error: IP Restriction${NC}"
                echo -e "   Message: $error_msg"
                echo -e "   ${YELLOW}→ Add your IP ($CURRENT_IP) to API whitelist${NC}"
                ;;
            -2015)
                echo -e "${RED}✗ $env_name Error: Invalid API Key${NC}"
                echo -e "   Message: $error_msg"
                echo -e "   ${YELLOW}→ Check API key permissions (Enable Futures)${NC}"
                echo -e "   ${YELLOW}→ Verify keys match $env_name environment${NC}"
                ;;
            -1102)
                echo -e "${RED}✗ $env_name Error: Missing Parameters${NC}"
                echo -e "   Message: $error_msg"
                echo -e "   ${YELLOW}→ This should not happen - check script${NC}"
                ;;
            -2014)
                echo -e "${RED}✗ $env_name Error: Bad Signature${NC}"
                echo -e "   Message: $error_msg"
                echo -e "   ${YELLOW}→ Check API secret is correct${NC}"
                ;;
            *)
                echo -e "${RED}✗ $env_name Error: Code $error_code${NC}"
                echo -e "   Message: $error_msg"
                ;;
        esac
    elif echo "$response" | grep -q '"totalWalletBalance"'; then
        echo -e "${GREEN}✓ $env_name Authentication: SUCCESS${NC}"
        local balance=$(echo "$response" | jq -r '.totalWalletBalance' 2>/dev/null || echo "unknown")
        echo -e "   Wallet Balance: ${GREEN}$balance USDT${NC}"
        return 0
    else
        echo -e "${YELLOW}○ $env_name: Unexpected response${NC}"
        echo "   Response: ${response:0:100}..."
    fi
    
    return 1
}

# Test mainnet authentication
test_authenticated_endpoint "$API_KEY" "$API_SECRET" "$MAINNET_URL" "Mainnet"

MAINNET_SUCCESS=$?
echo ""

# Test testnet authentication (for comparison)
test_authenticated_endpoint "$TESTNET_API" "$TESTNET_SECRET" "$TESTNET_URL" "Testnet"

echo ""
echo -e "${YELLOW}📋 Phase 3: Configuration Review${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo -e "${BLUE}Your .env file should contain:${NC}"
cat << 'EOF'
# For Mainnet (real trading):
BINANCE_API_KEY=your_mainnet_api_key
BINANCE_API_SECRET=your_mainnet_secret
BINANCE_USE_TESTNET=false

# For Testnet (paper trading):
BINANCE_TESTNET_API=your_testnet_api_key
BINANCE_TESTNET_SECRET=your_testnet_secret
# BINANCE_USE_TESTNET=true
EOF

echo ""
echo -e "${YELLOW}🔗 Quick Fix Steps:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [ $MAINNET_SUCCESS -ne 0 ]; then
    echo -e "${RED}Mainnet authentication failed. To fix:${NC}"
    echo ""
    echo "1. ${BLUE}Verify API Keys:${NC}"
    echo "   - Go to: https://www.binance.com/en/my/settings/api-management"
    echo "   - Ensure your API key exists and is active"
    echo ""
    echo "2. ${BLUE}Check Permissions:${NC}"
    echo "   - 'Enable Reading' must be ON"
    echo "   - 'Enable Futures' must be ON"
    echo "   - 'Enable Spot & Margin Trading' is NOT needed for futures"
    echo ""
    echo "3. ${BLUE}Check IP Restrictions:${NC}"
    echo "   - Your current IP: $CURRENT_IP"
    echo "   - Either: Set IP access restriction to include this IP"
    echo "   - Or: Disable IP restrictions (less secure)"
    echo ""
    echo "4. ${BLUE}Verify Environment:${NC}"
    echo "   - Keys for testnet DO NOT work on mainnet"
    echo "   - Keys for mainnet DO NOT work on testnet"
    echo "   - Use BINANCE_TESTNET keys when BINANCE_USE_TESTNET=true"
    echo ""
    echo "5. ${BLUE}Regenerate if needed:${NC}"
    echo "   - If still failing, create new API keys"
    echo "   - Update .env with new keys"
else
    echo -e "${GREEN}✓ Mainnet is working correctly!${NC}"
    echo ""
    echo "You can now run:"
    echo "  ${BLUE}./cognee${NC} to start the trading bot"
    echo "  ${BLUE}./debug_api.sh${NC} for detailed diagnostics"
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Diagnostic Complete${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""