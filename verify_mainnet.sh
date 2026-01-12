#!/bin/bash

# Mainnet Verification Script
# Use this BEFORE trading with real money

echo -e "🏦 Binance Mainnet API Verification"
echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Get IP
echo -e "📍 Your IP Address: \033[1;33m$(curl -s ifconfig.me)\033[0m"
echo ""

# Check if IP is whitelisted
echo -e "🔐 Checking Binance Mainnet IP Whitelist..."
echo -e "\033[0;33m  (If this fails, add your IP to Binance → API Management → IP Restrictions)\033[0m"
echo ""

# Export environment
export $(grep -v '^#' .env | grep BINANCE)

# Test API connection
echo -e "📡 Testing API connection..."
./cognee 2>&1 | head -30 | tee test_mainnet.log

# Check results
if grep -q "API permissions verified" test_mainnet.log; then
    echo ""
    echo -e "✅ \033[1;32mSUCCESS!\033[0m API connection verified."
    echo ""
    
    # Show account info
    if grep -q "total_wallet_balance" test_mainnet.log; then
        BALANCE=$(grep "total_wallet_balance" test_mainnet.log | tail -1 | jq -r '.total_wallet_balance // "Unknown"')
        echo -e "💰 Account Balance: \033[1;32m$BALANCE USDT\033[0m"
    fi
    
    # Show mode
    if grep -q "MAINNET (Real Money)" test_mainnet.log; then
        echo -e "⚠️  MODE: \033[1;31mMAINNET (Real Money Trading!)\033[0m"
        echo ""
        echo -e "\033[1;31m🚨 WARNING: This is REAL money trading!\033[0m"
        echo -e "\033[1;33m⚠️  Safe-Stop is ENABLED at 10% loss threshold\033[0m"
        echo -e "\033[1;33m⚠️  Minimum balance before stop: $1000 USD\033[0m"
    else
        echo -e "🧪 MODE: Testnet (Safe)"
    fi
    
else
    echo ""
    echo -e "❌ \033[1;31mFAILED!\033[0m API connection error."
    echo ""
    echo -e "\033[0;33mPossible causes:\033[0m"
    echo -e "  1. IP not whitelisted in Binance → Add your IP"
    echo -e "  2. API keys incorrect → Verify keys in .env"
    echo -e "  3. API doesn't have Futures permissions → Enable in Binance"
    echo ""
    echo -e "\033[1;31m⚠️  DO NOT trade until this is fixed!\033[0m"
fi

# Cleanup
rm -f test_mainnet.log

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
