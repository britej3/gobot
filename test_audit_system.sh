#!/bin/bash

# Cognee API Audit & Safe-Stop Test Script
echo "🚀 Cognee API Audit & Safe-Stop Test System"
echo "============================================="
echo ""

# Test 1: Audit without API keys
echo "1️⃣ Testing Audit Mode (No API Keys)"
echo "--------------------------------------"
./cognee --audit
echo ""

# Test 2: Audit with dummy keys (should fail)
echo "2️⃣ Testing Audit Mode (Invalid API Keys)"
echo "-----------------------------------------"
export BINANCE_API_KEY="test_key_12345"
export BINANCE_API_SECRET="test_secret_67890"
export BINANCE_USE_TESTNET="true"
./cognee --audit
echo ""

# Test 3: Show current configuration
echo "3️⃣ Current Configuration"
echo "-------------------------"
echo "BINANCE_USE_TESTNET: $BINANCE_USE_TESTNET"
echo "MIN_FVG_CONFIDENCE: ${MIN_FVG_CONFIDENCE:-0.6}"
echo "MAX_VOLATILITY: ${MAX_VOLATILITY:-0.05}"
echo "SAFE_STOP_ENABLED: ${SAFE_STOP_ENABLED:-true}"
echo "SAFE_STOP_THRESHOLD_PERCENT: ${SAFE_STOP_THRESHOLD_PERCENT:-10}"
echo ""

# Test 4: Test trade with audit
echo "4️⃣ Testing Trade with Audit"
echo "----------------------------"
./cognee --test-trade --symbol BTCUSDT --side BUY --aggressive
echo ""

# Test 5: Full platform startup (if keys are valid)
if [[ -n "$BINANCE_API_KEY" && "$BINANCE_API_KEY" != "test_key_12345" ]]; then
    echo "5️⃣ Starting Full Platform (Valid Keys Detected)"
    echo "-----------------------------------------------"
    echo "Starting Cognee with Safe-Stop monitoring..."
    echo "Press Ctrl+C to stop"
    ./cognee
else
    echo "5️⃣ Skipping Full Platform (Invalid/Dummy Keys)"
    echo "----------------------------------------------"
    echo "To test the full platform, set valid API keys:"
    echo "export BINANCE_API_KEY=your_real_api_key"
    echo "export BINANCE_API_SECRET=your_real_secret"
    echo "export BINANCE_USE_TESTNET=true  # for testing"
fi

echo ""
echo "🎉 Audit system test completed!"
echo "================================="