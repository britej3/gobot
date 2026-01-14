#!/bin/bash
echo "Starting GOBOT with qwen3:0.6b (LFM2.5 backup)..."
echo "API: Binance Testnet"
echo "Assets: 45 mid-cap assets"
echo "Strategy: High-frequency FVG scalping"
echo ""

# Export settings explicitly
export BINANCE_USE_TESTNET=true
export MIN_FVG_CONFIDENCE=0.4  # Very aggressive for more trades
export MAX_VOLATILITY=0.08     # Allow higher volatility

# Use qwen3 since LFM2.5 has tensor errors
export OLLAMA_MODEL=qwen3:0.6b
export OLLAMA_BASE_URL=http://localhost:11964

echo "Starting in 3 seconds..."
sleep 3

./cognee 2>&1 | tee bot_run_$(date +%Y%m%d_%H%M%S).log
