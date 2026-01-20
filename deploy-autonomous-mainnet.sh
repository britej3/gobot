#!/bin/bash

# ============================================================================
# GOBOT AUTONOMOUS MAINNET DEPLOYMENT SCRIPT
# ============================================================================

set -e  # Exit on error

echo "=================================================="
echo "GOBOT AUTONOMOUS MAINNET DEPLOYMENT"
echo "=================================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
echo "🔍 Checking prerequisites..."

if ! command -v go &> /dev/null; then
    echo -e "${RED}✗ Go is not installed${NC}"
    exit 1
fi

if [ ! -f ".env" ]; then
    echo -e "${RED}✗ .env file not found${NC}"
    exit 1
fi

if [ ! -d "logs" ]; then
    echo "Creating logs directory..."
    mkdir -p logs
fi

if [ ! -d "state" ]; then
    echo "Creating state directory..."
    mkdir -p state
fi

# Check if .env is configured
if grep -q "your_api_key_here" .env; then
    echo -e "${RED}✗ .env file is not configured with API keys${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites check passed${NC}"
echo ""

# Stop existing bot if running
echo "🛑 Stopping existing bot (if running)..."
pkill -f gobot-autonomous || true
sleep 2
echo ""

# Build the bot
echo "🔨 Building GOBOT Autonomous..."
go build -o gobot-autonomous ./cmd/autonomous
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi
echo ""

# Verify configuration
echo "🔍 Verifying configuration..."
if grep -q "BINANCE_USE_TESTNET=false" .env; then
    echo -e "${GREEN}✓ Mainnet mode enabled${NC}"
else
    echo -e "${YELLOW}⚠ Warning: Testnet mode may still be enabled${NC}"
fi

BALANCE=$(grep "initial_capital_usd" config/config.yaml | awk '{print $2}')
echo -e "${GREEN}✓ Balance configured for ${BALANCE} USDT${NC}"
echo ""

# Final confirmation
echo "=================================================="
echo "DEPLOYMENT SUMMARY"
echo "=================================================="
echo "Opening Balance: 26 USDT"
echo "Min Position: 8 USDT"
echo "Max Position: 13 USDT"
echo "Stop Loss: 3%"
echo "Take Profit: 15%"
echo "Trailing Stop: 1%"
echo "Max Positions: 3"
echo "Min Score: 120 points"
echo "Trading Interval: 3 minutes"
echo "Screener Interval: 30 seconds"
echo ""
echo -e "${RED}⚠️  THIS IS MAINNET - REAL MONEY TRADING${NC}"
echo ""

read -p "Are you sure you want to deploy to mainnet? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    echo "Deployment cancelled"
    exit 0
fi

echo ""
echo "🚀 Starting GOBOT Autonomous on mainnet..."
echo ""

# Start the bot in background
nohup ./gobot-autonomous > logs/autonomous.log 2>&1 &
BOT_PID=$!

echo -e "${GREEN}✓ Bot started with PID: ${BOT_PID}${NC}"
echo ""
echo "=================================================="
echo "MONITORING COMMANDS"
echo "=================================================="
echo "View logs:     tail -f logs/autonomous.log"
echo "Check status:  ps aux | grep gobot-autonomous"
echo "Stop bot:     pkill -f gobot-autonomous"
echo ""
echo "Bot is running in background. Use the commands above to monitor."
echo ""