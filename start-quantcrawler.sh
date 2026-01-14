#!/bin/bash

# ═══════════════════════════════════════════════════════════════════════════
# GOBOT + QuantCrawler Startup Script
# ═══════════════════════════════════════════════════════════════════════════

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=============================================="
echo "  GOBOT + QuantCrawler Launcher"
echo "=============================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check for required environment variables
if [[ -z "$QUANTCRAWLER_EMAIL" ]] || [[ -z "$QUANTCRAWLER_PASSWORD" ]]; then
    echo -e "${YELLOW}Warning: QUANTCRAWLER_EMAIL and QUANTCRAWLER_PASSWORD not set${NC}"
    echo "Please set them in your shell or .env file:"
    echo ""
    echo "  export QUANTCRAWLER_EMAIL=\"your-email@gmail.com\""
    echo "  export QUANTCRAWLER_PASSWORD=\"your-password-or-app-password\""
    echo ""
    echo "For 2FA accounts, use an App Password from:"
    echo "  https://myaccount.google.com/apppasswords"
    echo ""
fi

# Function to check if a port is in use
check_port() {
    if lsof -i:$1 > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Check for existing processes
echo "Checking ports..."

N8N_PORT=5678
QC_PORT=3456
GOBOT_PORT=8080

if check_port $N8N_PORT; then
    echo -e "${YELLOW}N8N already running on port $N8N_PORT${NC}"
else
    echo -e "${GREEN}Port $N8N_PORT available (N8N)${NC}"
fi

if check_port $QC_PORT; then
    echo -e "${YELLOW}QuantCrawler server already running on port $QC_PORT${NC}"
else
    echo -e "${GREEN}Port $QC_PORT available (QuantCrawler)${NC}"
fi

if check_port $GOBOT_PORT; then
    echo -e "${YELLOW}GOBOT already running on port $GOBOT_PORT${NC}"
else
    echo -e "${GREEN}Port $GOBOT_PORT available (GOBOT)${NC}"
fi

echo ""
echo "=============================================="
echo "  Starting Services"
echo "=============================================="

# Create session directory if it doesn't exist
mkdir -p n8n-sessions

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "Shutting down..."
    if [[ ! -z "$QC_PID" ]] && kill -0 $QC_PID 2>/dev/null; then
        kill $QC_PID 2>/dev/null || true
    fi
    exit 0
}

trap cleanup SIGINT SIGTERM

# Start QuantCrawler webhook server in background
echo -e "${GREEN}[1/3]${NC} Starting QuantCrawler Puppeteer server..."
node n8n/scripts/quantcrawler.js --webhook &
QC_PID=$!
echo "       QuantCrawler PID: $QC_PID"

# Wait for QuantCrawler to be ready
sleep 2
echo "       Waiting for QuantCrawler..."
for i in {1..10}; do
    if curl -s http://localhost:$QC_PORT/webhook > /dev/null 2>&1; then
        echo -e "       ${GREEN}QuantCrawler ready!${NC}"
        break
    fi
    sleep 1
done

echo ""
echo "=============================================="
echo "  Services Started"
echo "=============================================="
echo ""
echo "  QuantCrawler: http://localhost:$QC_PORT/webhook"
echo "  N8N:         http://localhost:$N8N_PORT (start separately)"
echo "  GOBOT:       http://localhost:$GOBOT_PORT (start separately)"
echo ""
echo "  To test QuantCrawler:"
echo "    curl -X POST http://localhost:$QC_PORT/webhook \\"
echo "      -H 'Content-Type: application/json' \\"
echo "      -d '{\"symbol\":\"1000PEPEUSDT\",\"account_balance\":1000}'"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"
echo ""

# Keep script running
while true; do
    sleep 1
    # Check if QuantCrawler is still running
    if ! kill -0 $QC_PID 2>/dev/null; then
        echo "QuantCrawler server stopped"
        break
    fi
done
