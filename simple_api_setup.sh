#!/bin/bash

# Simple API Key Setup for GOBOT
# No fancy input handling - just simple and works

echo "🚀 GOBOT API Key Setup (Simple Version)"
echo "======================================="
echo ""

# Simple read without fancy password masking
read -p "Enter your Binance API Key: " api_key
read -p "Enter your Binance API Secret: " api_secret

# Choose testnet or mainnet
echo ""
echo "Network Selection:"
echo "1) Testnet (recommended for testing - fake money)"
echo "2) Mainnet (real money - high risk)"
read -p "Choose (1 or 2): " network_choice

if [ "$network_choice" = "2" ]; then
    use_testnet="false"
    echo "🚨 WARNING: You selected MAINNET (real money!)"
else
    use_testnet="true"
    echo "✅ Using Testnet (safe for testing)"
fi

# Set environment variables
export BINANCE_API_KEY="$api_key"
export BINANCE_API_SECRET="$api_secret"
export BINANCE_USE_TESTNET="$use_testnet"

echo ""
echo "✅ Configuration applied for current session!"
echo ""
echo "Current settings:"
echo "BINANCE_API_KEY: ${BINANCE_API_KEY:0:10}..."
echo "BINANCE_USE_TESTNET: $BINANCE_USE_TESTNET"
echo ""

# Ask if they want to make it permanent
read -p "Save to ~/.bashrc for future sessions? (y/N): " save_perm
if [ "$save_perm" = "y" ] || [ "$save_perm" = "Y" ]; then
    echo "" >> ~/.bashrc
    echo "# GOBOT Binance API Configuration" >> ~/.bashrc
    echo "export BINANCE_API_KEY=\"$api_key\"" >> ~/.bashrc
    echo "export BINANCE_API_SECRET=\"$api_secret\"" >> ~/.bashrc
    echo "export BINANCE_USE_TESTNET=\"$use_testnet\"" >> ~/.bashrc
    echo "✅ Saved to ~/.bashrc"
fi

echo ""
echo "🎉 Setup complete! You can now start the trading platform:"
echo "./cognee"
echo ""