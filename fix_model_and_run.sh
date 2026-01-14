#!/bin/bash

# Fix LFM2.5 Model and Run GOBOT

echo "=== LFM2.5 Model Fix and Bot Startup ==="
echo ""

# 1. Stop any running bot
pkill -f cognee 2>/dev/null
sleep 2

echo "1. Checking LFM2.5 model integrity..."
cd "/Users/britebrt/Library/Application Support/MstyStudio/models-llamacpp/LiquidAI/LFM2.5-1.2B-Instruct-GGUF/"

# Check if model has correct size (should be ~1.2GB)
MODEL_SIZE=$(stat -f%z LFM2.5-1.2B-Instruct-Q8_0.gguf 2>/dev/null || echo 0)
if [ "$MODEL_SIZE" -lt 1000000000 ]; then
    echo "   ✗ Model file too small or missing ($MODEL_SIZE bytes)"
    echo "   → Re-downloading from HuggingFace..."
    
    # Backup corrupted file
    if [ -f LFM2.5-1.2B-Instruct-Q8_0.gguf ]; then
        mv LFM2.5-1.2B-Instruct-Q8_0.gguf LFM2.5-1.2B-Instruct-Q8_0.gguf.corrupted.$(date +%Y%m%d_%H%M%S)
    fi
    
    # Download with huggingface-cli (more reliable)
    if command -v huggingface-cli &> /dev/null; then
        huggingface-cli download LiquidAI/LFM2.5-1.2B-Instruct-GGUF \
            LFM2.5-1.2B-Instruct-Q8_0.gguf \
            --local-dir . \
            --local-dir-use-symlinks False
    else
        # Fallback to wget with redirect handling
        wget --quiet --content-disposition \
            "https://huggingface.co/LiquidAI/LFM2.5-1.2B-Instruct-GGUF/resolve/main/LFM2.5-1.2B-Instruct-Q8_0.gguf"
    fi
    
    echo "   ✓ Model downloaded"
else
    echo "   ✓ Model file exists ($MODEL_SIZE bytes)"
fi

echo ""
echo "2. Restarting msty llama-server..."
pkill -f "llama-server.*LFM2.5"
sleep 3

# Start new llama-server instance in background
/Users/britebrt/Library/Application\ Support/MstyStudio/llama-cpp/msty-llama-server \
    --port 5800 \
    --host 127.0.0.1 \
    --model "/Users/britebrt/Library/Application Support/MstyStudio/models-llamacpp/LiquidAI/LFM2.5-1.2B-Instruct-GGUF/LFM2.5-1.2B-Instruct-Q8_0.gguf" \
    --threads -1 \
    --parallel 2 \
    --jinja \
    --no-webui \
    --ctx-size 4096 \
    --log-disable &

LLAMA_PID=$!
echo "   → Llama server started (PID: $LLAMA_PID)"

# Wait for model to load
echo ""
echo "3. Waiting for model to load..."
sleep 5

echo ""
echo "4. Testing model response..."
for i in {1..10}; do
    response=$(curl -s http://localhost:5800/api/generate \
        -H "Content-Type: application/json" \
        -d '{"model":"LFM2.5-1.2B-Instruct-Q8_0.gguf","prompt":"Return JSON: {\"test\":\"ok\"}","stream":false}' \
        | grep -c "response" || echo 0)
    
    if [ "$response" -gt 0 ]; then
        echo "   ✓ Model responding correctly!"
        break
    fi
    
    if [ $i -eq 10 ]; then
        echo "   ✗ Model not responding after 10 attempts"
        echo "   → Using fallback qwen3:0.6b model"
        export OLLAMA_MODEL="qwen3:0.6b"
        export OLLAMA_BASE_URL="http://localhost:11964"
    fi
    
    echo "   Attempt $i/10..."
    sleep 2
done

echo ""
echo "5. Starting GOBOT with LFM2.5..."
cd /Users/britebrt/GOBOT

# Use testnet for safety
export BINANCE_USE_TESTNET=true
export MIN_FVG_CONFIDENCE=0.4  # More aggressive for testing

./cognee 2>&1 | tee startup.log
