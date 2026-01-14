#!/bin/bash

# ═══════════════════════════════════════════════════════════════════════════
# GOBOT Complete Local Launcher (No Docker)
# ═══════════════════════════════════════════════════════════════════════════

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=============================================="
echo "  GOBOT - Complete Local System"
echo "=============================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Cleanup function
cleanup() {
    echo ""
    echo "Stopping all services..."
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    echo "Done."
    exit 0
}

trap cleanup SIGINT SIGTERM

declare -a PIDS=()

# Check/create directories
mkdir -p n8n-sessions
mkdir -p state

echo "[1/4] Checking services..."

# Check port availability
check_port() {
    if lsof -i:$1 > /dev/null 2>&1; then
        return 1
    fi
    return 0
}

# Start QuantCrawler Puppeteer Server (port 3456)
echo "[2/4] Starting QuantCrawler Puppeteer..."
if check_port 3456; then
    if [[ -d "node_modules/puppeteer" ]]; then
        node n8n/scripts/quantcrawler.js --webhook &
        PIDS+=($!)
        echo "  ✓ QuantCrawler (PID: ${PIDS[-1]})"
        sleep 3
    else
        echo "  ⚠ Puppeteer not installed, skipping..."
    fi
else
    echo "  ⚠ Port 3456 in use, skipping..."
fi

# Start N8N Alternative (port 5678)
echo "[3/4] Starting N8N Alternative..."
if check_port 5678; then
    npm install --silent 2>/dev/null || true
    node n8n-local.js &
    PIDS+=($!)
    echo "  ✓ N8N Alternative (PID: ${PIDS[-1]})"
    sleep 2
else
    echo "  ⚠ Port 5678 in use, skipping..."
fi

# Start Go Bot (port 8080)
echo "[4/4] Starting GOBOT..."
if check_port 8080; then
    if [[ -f "gobot" ]]; then
        ./gobot &
        PIDS+=($!)
        echo "  ✓ GOBOT (PID: ${PIDS[-1]})"
    else
        echo "  ⚠ gobot binary not found, run: go build -o gobot ./cmd/cobot"
    fi
else
    echo "  ⚠ Port 8080 in use, skipping..."
fi

echo ""
echo "=============================================="
echo "  All Services Started"
echo "=============================================="
echo ""
echo -e "${BLUE}Services:${NC}"
echo "  QuantCrawler:  http://localhost:3456/webhook"
echo "  N8N Alt:       http://localhost:5678"
echo "  GOBOT:         http://localhost:8080"
echo ""
echo -e "${BLUE}Test Commands:${NC}"
echo ""
echo "  # Test QuantCrawler:"
echo "  curl -X POST http://localhost:3456/webhook \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"symbol\":\"1000PEPEUSDT\"}'"
echo ""
echo "  # Test N8N Workflow:"
echo "  curl -X POST http://localhost:5678/webhook/quantcrawler-analysis \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"symbol\":\"1000PEPEUSDT\",\"account_balance\":1000}'"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"
echo ""

# Wait and monitor
while true; do
    sleep 5
    for pid in "${PIDS[@]}"; do
        if ! kill -0 "$pid" 2>/dev/null; then
            echo ""
            echo "Process $pid stopped"
        fi
    done
done
