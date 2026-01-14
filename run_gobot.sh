#!/bin/bash
# GOBOT Startup Script - Fixed Configuration

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║         GOBOT TRADING BOT - LIQUIDAI LFM2.5 Edition         ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Kill any existing bot instance
pkill -f cognee 2>/dev/null
sleep 2

# Set correct environment
export BINANCE_USE_TESTNET=true
export MIN_FVG_CONFIDENCE=0.4
export MAX_VOLATILITY=0.08
export MARKET_REGIME_TOLERANCE=true

# Use qwen3:0.6b (LFM2.5 has tensor errors)
export OLLAMA_MODEL=qwen3:0.6b
export OLLAMA_BASE_URL=http://localhost:11964

echo "✓ Testnet: ENABLED"
echo "✓ AI Model: qwen3:0.6b (port 11964)"
echo "✓ Confidence: 0.4 (aggressive)"
echo "✓ Max Volatility: 8%"
echo "✓ Assets: 45 mid-cap cryptocurrencies"
echo ""
echo "Starting bot..."
echo "════════════════════════════════════════════════════════════"

# Run bot with error handling
./cognee 2>&1 | tee "gobot_run_$(date +%Y%m%d_%H%M%S).log"
