#!/bin/bash

echo "🔍 DEBUG: Binance API Signature Error"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Get current IP
echo -e "📍 Your IP: \033[1;33m$(curl -s ifconfig.me)\033[0m"
echo ""

# Check .env settings
echo "📄 Checking API configuration..."
API_KEY=$(grep BINANCE_API_KEY .env | cut -d'=' -f2 | tr -d '"' | tr -d ' ')
API_SECRET=$(grep BINANCE_API_SECRET .env | cut -d'=' -f2 | tr -d '"' | tr -d ' ')
TESTNET=$(grep BINANCE_USE_TESTNET .env | cut -d'=' -f2)

echo -e "BINANCE_API_KEY:    ${API_KEY:0:10}...${API_KEY: -10}"
echo -e "BINANCE_API_SECRET: ${API_SECRET:0:10}...${API_SECRET: -10}"
echo -e "BINANCE_USE_TESTNET: \033[1;33m$TESTNET\033[0m"
echo ""

# Determine environment
if [ "$TESTNET" = "true" ]; then
    echo -e "🧪 ENVIRONMENT: \033[1;32mTESTNET\033[0m (Paper Trading)"
    BASE_URL="https://testnet.binancefuture.com"
else
    echo -e "🚨 ENVIRONMENT: \033[1;31mMAINNET\033[0m (Real Money!)"
    BASE_URL="https://fapi.binance.com"
fi
echo -e "API Base URL: \033[1;36m$BASE_URL\033[0m"
echo ""

# Test API connectivity with detailed error
echo "📡 Testing API connection..."
echo ""

# Simple timestamp test for signature validity
timestamp=$(date +%s000)

# Check using curl directly
echo "🔍 Direct API Test (without signature - connectivity only):"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
curl -s "$BASE_URL/fapi/v1/time" | jq . 2>/dev/null || echo "❌ Cannot connect to API endpoint"
echo ""

# Check account status
echo "🔍 Account Status Test (with signature):"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Try to check account info using the actual bot
echo -e "\033[0;33mRunning full API test...\033[0m"
echo ""

export BINANCE_API_KEY="$API_KEY"
export BINANCE_API_SECRET="$API_SECRET"
export BINANCE_USE_TESTNET="$TESTNET"

timeout 10s ./cognee 2>&1 | tee debug_output.log | head -50

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Parse errors
echo "🔍 Error Analysis:"
echo ""

if grep -q "Signature for this request is not valid" debug_output.log; then
    echo -e "❌ Error: \033[1;31mSignature not valid\033[0m"
    echo ""
    echo -e "\033[1;33mPossible causes:\033[0m"
    echo "  1. API keys don't match Binance account"
    echo "  2. Keys are for testnet but using mainnet (or vice versa)"
    echo "  3. API secret is incorrect"
    echo "  4. API key doesn't have Futures enabled"
    echo ""
    echo -e "\033[1;36mWhat to check:\033[0m"
    echo "  1. In Binance → API Management → Verify key exists"
    echo "  2. Check 'Enable Futures' permission is ON"
    echo "  3. Copy keys directly from Binance again"
    echo "  4. Ensure no spaces at start/end of keys"
    
elif grep -q "APIError.*code=-1022" debug_output.log; then
    echo -e "❌ Error: \033[1;31mIP restriction issue\033[0m"
    echo ""
    echo -e "\033[1;33mYour IP: $(curl -s ifconfig.me)\033[0m"
    echo ""
    echo -e "\033[1;31mACTION REQUIRED:\033[0m"
    echo "  Add your IP to Binance → API Management → IP Restrictions"
    
elif grep -q "APIError.*code=-2015" debug_output.log; then
    echo -e "❌ Error: \033[1;31mInvalid API key or insufficient permissions\033[0m"
    echo ""
    echo -e "\033[1;31mACTION REQUIRED:\033[0m"
    echo "  1. Verify API key in Binance → API Management"
    echo "  2. Enable 'Enable Futures' permission"
    echo "  3. API may be expired - create new one if needed"
    
elif grep -q "✅ Futures API connection established" debug_output.log; then
    echo -e "✅ \033[1;32mAPI connection successful!\033[0m"
    echo ""
    echo -e "\033[1;32mThe API is working correctly.\033[0m"
    echo ""
    if grep -q "total_wallet_balance" debug_output.log; then
        BALANCE=$(grep "total_wallet_balance" debug_output.log | tail -1 | jq -r '.total_wallet_balance // "Unknown"')
        echo -e "💰 Balance: \033[1;32m$BALANCE USDT\033[0m"
    fi
    
else
    echo -e "⚠️  Unable to determine error from logs"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Cleanup
rm -f debug_output.log

echo ""
echo -e "\033[1;33m💡 Tip: If keys are wrong, regenerate them in Binance:\033[0m"
echo "  1. Binance.com → Profile → API Management"
echo "  2. Create new API key (or edit existing)"
echo "  3. Enable: Enable Reading + Enable Futures"
echo "  4. Add IP: $(curl -s ifconfig.me)"
echo "  5. Copy new keys to .env file"
echo ""
echo -e "\033[1;36m⚠️  NEVER share your API keys!\033[0m"