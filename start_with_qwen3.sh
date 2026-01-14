#!/bin/bash
# Emergency startup using qwen3 (works, just no trades due to API)

echo "🚀 Starting GOBOT with qwen3:0.6b..."
echo ""

# Set environment
export BINANCE_USE_TESTNET=true
export OLLAMA_MODEL=qwen3:0.6b
export OLLAMA_BASE_URL=http://localhost:11964
export MIN_FVG_CONFIDENCE=0.3  # Very aggressive

# Start and show logs only
echo "Bot is running. Looking for FVG opportunities..."
echo "Press Ctrl+C to stop"
echo ""

./cognee 2>&1 | grep -E "(🎯|✅|🔍|FVG|Trade|BUY|SELL)" --color=never
