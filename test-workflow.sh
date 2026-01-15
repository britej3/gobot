#!/bin/bash

# ═══════════════════════════════════════════════════════════════════════════
# Test TradingView QuantCrawler Workflow
# Tests all components without N8N
# ═══════════════════════════════════════════════════════════════════════════

set -e

BASE_URL="http://localhost"
PORTS=("3456" "5678" "8080")
SYMBOL="${1:-1000PEPEUSDT}"
BALANCE="${2:-10000}"

echo "=============================================="
echo "  TradingView QuantCrawler Workflow Test"
echo "=============================================="
echo ""
echo "Symbol: $SYMBOL"
echo "Balance: $BALANCE"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to check port
check_port() {
    if lsof -i:$1 > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Port $1 running${NC}"
        return 0
    else
        echo -e "${RED}✗ Port $1 NOT running${NC}"
        return 1
    fi
}

# Check all services
echo "[1/4] Checking Services..."
echo ""

running=0
for port in "${PORTS[@]}"; do
    if check_port $port; then
        ((running++))
    fi
done

echo ""
if [[ $running -lt 3 ]]; then
    echo -e "${YELLOW}Some services not running. Starting missing services...${NC}"
    
    # Start screenshot service if not running
    if ! check_port 3456; then
        echo "  Starting screenshot service..."
        cd services/screenshot-service
        npm start > /dev/null 2>&1 &
        sleep 3
    fi
    
    # Start N8N alternative if not running
    if ! check_port 5678; then
        echo "  Starting N8N alternative..."
        cd /Users/britebrt/GOBOT
        node n8n-local.js > /dev/null 2>&1 &
        sleep 2
    fi
    
    # Start GOBOT if not running
    if ! check_port 8080; then
        echo "  Starting GOBOT..."
        cd /Users/britebrt/GOBOT
        ./gobot > /dev/null 2>&1 &
        sleep 2
    fi
fi

echo ""
echo "[2/4] Testing Screenshot Service (Port 3456)..."
echo ""

# Test screenshot service
response=$(curl -s -X POST "$BASE_URL:3456/capture" \
    -H "Content-Type: application/json" \
    -d "{\"symbol\":\"$SYMBOL\",\"interval\":\"1m\"}")

if echo "$response" | jq -e '.screenshot' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Screenshot captured successfully${NC}"
    duration=$(echo "$response" | jq -r '.duration_ms')
    echo "  Duration: ${duration}ms"
else
    echo -e "${YELLOW}⚠ Screenshot service returned: $response${NC}"
fi

echo ""
echo "[3/4] Testing N8N Alternative (Port 5678)..."
echo ""

# Test N8N alternative
response=$(curl -s -X POST "$BASE_URL:5678/webhook/quantcrawler-analysis" \
    -H "Content-Type: application/json" \
    -d "{\"symbol\":\"$SYMBOL\",\"account_balance\":$BALANCE}")

if echo "$response" | jq -e '.success' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ N8N workflow executed${NC}"
    direction=$(echo "$response" | jq -r '.direction')
    confidence=$(echo "$response" | jq -r '.confidence')
    echo "  Direction: $direction"
    echo "  Confidence: $confidence%"
else
    echo -e "${RED}✗ N8N workflow failed${NC}"
    echo "  Response: $response"
fi

echo ""
echo "[4/4] Testing GOBOT Webhooks (Port 8080)..."
echo ""

# Test GOBOT capture endpoint
response=$(curl -s -X POST "$BASE_URL:8080/webhook/capture-chart" \
    -H "Content-Type: application/json" \
    -d "{\"symbol\":\"$SYMBOL\",\"intervals\":[\"1m\",\"5m\",\"15m\"]}")

if echo "$response" | jq -e '.symbol' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ GOBOT capture endpoint working${NC}"
    echo "  Symbol: $(echo "$response" | jq -r '.symbol')"
else
    echo -e "${YELLOW}⚠ GOBOT capture: $response${NC}"
fi

echo ""
echo "=============================================="
echo "  Test Complete"
echo "=============================================="
echo ""
echo "To run the full N8N workflow:"
echo ""
echo "1. Open N8N at http://localhost:5678"
echo "2. Import: n8n/workflows/06-tradingview-quantcrawler.json"
echo "3. Create webhook: http://localhost:5678/webhook/tradingview-analysis"
echo "4. Test with:"
echo "   curl -X POST http://localhost:5678/webhook/tradingview-analysis \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"symbol\":\"$SYMBOL\",\"account_balance\":$BALANCE}'"
echo ""
