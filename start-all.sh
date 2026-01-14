#!/bin/bash

# ═══════════════════════════════════════════════════════════════════════════
# GOBOT Complete Local Launcher
# Starts: N8N + Puppeteer Server + Go Bot
# ═══════════════════════════════════════════════════════════════════════════

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=============================================="
echo "  GOBOT Complete Local Launcher"
echo "=============================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# PIDs for cleanup
declare -a PIDS=()

cleanup() {
    echo ""
    echo "=============================================="
    echo "  Shutting down all services..."
    echo "=============================================="
    
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            echo "Stopped process: $pid"
        fi
    done
    
    echo ""
    echo "All services stopped."
    exit 0
}

trap cleanup SIGINT SIGTERM

# Check and create directories
echo "[1/4] Setting up directories..."
mkdir -p n8n-sessions
mkdir -p n8n/workflows
echo "      ✓ Directories ready"

# Check environment variables
echo "[2/4] Checking environment..."
if [[ -z "$QUANTCRAWLER_EMAIL" ]] || [[ -z "$QUANTCRAWLER_PASSWORD" ]]; then
    echo -e "      ${YELLOW}Warning: QUANTCRAWLER_EMAIL and QUANTCRAWLER_PASSWORD not set${NC}"
    echo "      Edit .env file to add credentials"
    echo ""
fi

# Check if ports are available
check_port() {
    if lsof -i:$1 > /dev/null 2>&1; then
        return 1
    fi
    return 0
}

echo "      Checking ports..."
PORTS_OK=true
for port in 3456 5678 8080; do
    if ! check_port $port; then
        echo -e "      ${RED}✗ Port $port in use${NC}"
        PORTS_OK=false
    else
        echo -e "      ${GREEN}✓ Port $port available${NC}"
    fi
done

if [[ "$PORTS_OK" == "false" ]]; then
    echo ""
    echo -e "${YELLOW}Some ports are in use. Continuing anyway...${NC}"
fi

# Start QuantCrawler Puppeteer Server
echo ""
echo "[3/4] Starting QuantCrawler Puppeteer Server..."
if ! check_port 3456; then
    echo -e "      ${YELLOW}Port 3456 in use, skipping...${NC}"
else
    # Check if puppeteer is installed
    if [[ ! -d "node_modules/puppeteer" ]]; then
        echo "      Installing Puppeteer..."
        npm install puppeteer 2>&1 | tail -5
    fi
    
    node n8n/scripts/quantcrawler.js --webhook &
    QC_PID=$!
    PIDS+=($QC_PID)
    echo "      ✓ QuantCrawler started (PID: $QC_PID)"
    
    # Wait for QuantCrawler to be ready
    echo "      Waiting for QuantCrawler..."
    for i in {1..15}; do
        if curl -s http://localhost:3456/webhook > /dev/null 2>&1; then
            echo -e "      ${GREEN}✓ QuantCrawler ready on port 3456${NC}"
            break
        fi
        sleep 1
    done
fi

# Start N8N
echo ""
echo "[4/4] Starting N8N..."
if ! check_port 5678; then
    echo -e "      ${YELLOW}Port 5678 in use, skipping...${NC}"
else
    if command -v n8n &> /dev/null; then
        n8n start &
        N8N_PID=$!
        PIDS+=($N8N_PID)
        echo "      ✓ N8N started (PID: $N8N_PID)"
    else
        echo -e "      ${YELLOW}N8N not found. Install with: npm install -g n8n${NC}"
    fi
    
    # Wait for N8N to be ready
    echo "      Waiting for N8N..."
    for i in {1..20}; do
        if curl -s http://localhost:5678/healthz > /dev/null 2>&1; then
            echo -e "      ${GREEN}✓ N8N ready on port 5678${NC}"
            break
        fi
        sleep 1
    done
fi

# Summary
echo ""
echo "=============================================="
echo "  All Services Started"
echo "=============================================="
echo ""
echo -e "${BLUE}QuantCrawler:${NC}  http://localhost:3456/webhook"
echo -e "${BLUE}N8N:${NC}           http://localhost:5678"
echo -e "${BLUE}GOBOT:${NC}         http://localhost:8080 (start separately)"
echo ""
echo "Credentials:"
echo "  N8N: gobot / secure_password"
echo ""
echo "To test QuantCrawler:"
echo "  curl -X POST http://localhost:3456/webhook \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"symbol\":\"1000PEPEUSDT\",\"account_balance\":1000}'"
echo ""
echo "To test N8N workflow:"
echo "  curl -X POST http://localhost:5678/webhook/quantcrawler-analysis \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"symbol\":\"1000PEPEUSDT\",\"account_balance\":1000}'"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"
echo ""

# Keep running and monitor
while true; do
    sleep 5
    
    # Check if any process died
    for pid in "${PIDS[@]}"; do
        if ! kill -0 "$pid" 2>/dev/null; then
            echo ""
            echo -e "${RED}Process $pid died${NC}"
        fi
    done
done
