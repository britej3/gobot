#!/bin/bash

# ============================================================================
# GOBOT MAINNET DEPLOYMENT SCRIPT
# ============================================================================

set -e  # Exit on error

echo "=================================================="
echo "GOBOT MAINNET DEPLOYMENT"
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
    echo -e "${YELLOW}⚠ .env file not found${NC}"
    echo "Creating .env from .env.example..."
    cp .env.example .env
    echo -e "${YELLOW}⚠ Please edit .env with your API keys before continuing${NC}"
    echo "Press Enter to continue or Ctrl+C to abort..."
    read
fi

if [ ! -d "logs" ]; then
    echo "Creating logs directory..."
    mkdir -p logs
fi

# Check if .env is configured
if grep -q "your_binance_api_key_here" .env; then
    echo -e "${RED}✗ .env file is not configured with API keys${NC}"
    echo "Please edit .env and fill in your API keys"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites check passed${NC}"
echo ""

# Build the bot
echo "🔨 Building GOBOT..."
go build -o cobot ./cmd/cobot
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi
echo ""

# Verify configuration
echo "🔍 Verifying configuration..."
if grep -q "BINANCE_USE_TESTNET: false" config/config.yaml; then
    echo -e "${GREEN}✓ Mainnet mode enabled${NC}"
else
    echo -e "${YELLOW}⚠ Warning: Testnet mode may still be enabled${NC}"
fi

if grep -q "initial_capital_usd: 30" config/config.yaml; then
    echo -e "${GREEN}✓ Balance configured for 30 USDT${NC}"
else
    echo -e "${YELLOW}⚠ Warning: Balance may not be set to 30 USDT${NC}"
fi
echo ""

# Final confirmation
echo "=================================================="
echo "DEPLOYMENT SUMMARY"
echo "=================================================="
echo "Opening Balance: 30 USDT"
echo "Min Position: 10 USDT"
echo "Max Position: 20 USDT"
echo "Stop Loss: 2%"
echo "Take Profit: 4%"
echo "Daily Loss Limit: 15 USDT"
echo ""
echo -e "${RED}⚠️  THIS IS MAINNET - REAL MONEY TRADING${NC}"
echo ""

read -p "Are you sure you want to deploy to mainnet? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    echo "Deployment cancelled"
    exit 0
fi

echo ""
echo "🚀 Starting GOBOT on mainnet..."
echo "Press Ctrl+C to stop"
echo ""

# Start the bot
./cobot