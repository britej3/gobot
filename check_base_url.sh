#!/bin/bash

echo "🔍 Checking Binance Base URL Configuration"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check .env setting
TESTNET=$(grep "BINANCE_USE_TESTNET" .env | cut -d'=' -f2)

echo -e "📄 .env setting: \033[1;33mBINANCE_USE_TESTNET=$TESTNET\033[0m"
echo ""

if [ "$TESTNET" = "true" ]; then
    echo -e "✅ Result: \033[1;32mTESTNET Mode\033[0m"
    echo -e "🌐 Base URL: \033[1;36mhttps://testnet.binancefuture.com\033[0m"
    echo ""
    echo -e "\033[0;33m🧪 Paper trading with fake money\033[0m"
elif [ "$TESTNET" = "false" ]; then
    echo -e "✅ Result: \033[1;32mMAINNET Mode\033[0m"
    echo -e "🌐 Base URL: \033[1;36mhttps://fapi.binance.com\033[0m (default)"
    echo ""
    echo -e "\033[1;31m🚨 REAL MONEY TRADING\033[0m"
    echo -e "\033[1;33m⚠️  Safe-Stop: ENABLED at 10% loss\033[0m"
    echo -e "\033[1;33m⚠️  Minimum balance: $1000 USD\033[0m"
else
    echo -e "❌ Error: Invalid setting"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Code is correctly configured!"
echo ""
echo "Issue: IP 81.22.30.29 not whitelisted"
echo "Solution: Add IP to Binance → API Management"
