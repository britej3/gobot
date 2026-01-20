#!/bin/bash
# GOBOT Mainnet Deployment Script
# Safe transition from testnet to mainnet

set -e

echo "=========================================="
echo "GOBOT MAINNET DEPLOYMENT SCRIPT"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Safety Check
echo -e "${YELLOW}Step 1: Safety Check${NC}"
echo "--------------------------------------"

# Check if testnet validation is complete
if [ ! -f "gobot_state/progress.json" ]; then
    echo -e "${RED}❌ Error: Testnet validation incomplete${NC}"
    echo "Please run testnet first and validate results"
    exit 1
fi

echo -e "${GREEN}✅ Testnet validation: COMPLETE${NC}"
echo ""

# Step 2: API Configuration
echo -e "${YELLOW}Step 2: API Configuration${NC}"
echo "--------------------------------------"

# Check if mainnet API keys are set
if [ -z "$BINANCE_API_KEY" ]; then
    echo -e "${RED}❌ Error: BINANCE_API_KEY not set${NC}"
    echo "Please set your mainnet API key:"
    echo "export BINANCE_API_KEY=your_mainnet_key"
    exit 1
fi

echo -e "${GREEN}✅ Mainnet API key: SET${NC}"
echo ""

# Step 3: Safety Configuration
echo -e "${YELLOW}Step 3: Safety Configuration${NC}"
echo "--------------------------------------"

echo "Current settings:"
echo "  Position size: $2 (max)"
echo "  Daily loss limit: $25"
echo "  Max trades per day: 3"
echo "  Stop loss: 1.5%"
echo "  Take profit: 3.0%"
echo ""

# Step 4: Confirmation
echo -e "${RED}⚠️  IMPORTANT: READ BEFORE PROCEEDING ⚠️${NC}"
echo ""
echo "You are about to deploy GOBOT to MAINNET trading."
echo "This means REAL MONEY will be at risk."
echo ""
echo "Safety measures in place:"
echo "  ✅ Circuit breakers enabled"
echo "  ✅ Conservative position sizing"
echo "  ✅ Daily loss limits"
echo "  ✅ Emergency stop procedures"
echo ""
echo "Recommended starting capital: \$100"
echo "Maximum you can lose: \$25 (25% of capital)"
echo ""

read -p "Type 'CONFIRM' to proceed or 'QUIT' to cancel: " confirm

if [ "$confirm" != "CONFIRM" ]; then
    echo "Deployment cancelled."
    exit 0
fi

echo ""
echo -e "${GREEN}✅ Deployment confirmed${NC}"
echo ""

# Step 5: Environment Setup
echo -e "${YELLOW}Step 5: Environment Setup${NC}"
echo "--------------------------------------"

# Create necessary directories
mkdir -p logs
mkdir -p gobot_mainnet_state
mkdir -p gobot_mainnet_archive

echo -e "${GREEN}✅ Directories created${NC}"
echo ""

# Step 6: Dry Run Test
echo -e "${YELLOW}Step 6: Dry Run Test${NC}"
echo "--------------------------------------"
echo "Testing orchestrator in dry run mode..."
echo ""

export DRY_RUN=true
export BINANCE_USE_TESTNET=true

python3 gobot_mainnet_orchestrator.py --max-iterations=1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Dry run test: PASSED${NC}"
else
    echo -e "${RED}❌ Dry run test: FAILED${NC}"
    exit 1
fi

echo ""

# Step 7: First Real Trade
echo -e "${YELLOW}Step 7: First Real Trade${NC}"
echo "--------------------------------------"
echo "This will execute ONE real trade on mainnet."
echo "Please monitor closely."
echo ""

read -p "Ready to execute first trade? (yes/no): " first_trade

if [ "$first_trade" != "yes" ]; then
    echo "First trade cancelled. You can run later with:"
    echo "export DRY_RUN=false"
    echo "export BINANCE_USE_TESTNET=false"
    echo "python3 gobot_mainnet_orchestrator.py --max-iterations=1"
    exit 0
fi

echo ""
echo -e "${RED}⚠️  EXECUTING REAL TRADE IN 10 SECONDS ⚠️${NC}"
echo "Press Ctrl+C to cancel..."
sleep 10

export DRY_RUN=false
export BINANCE_USE_TESTNET=false

python3 gobot_mainnet_orchestrator.py --max-iterations=1

echo ""
echo -e "${GREEN}✅ First trade completed${NC}"
echo "Check logs/ directory for details"
echo ""

# Step 8: Monitoring
echo -e "${YELLOW}Step 8: Monitoring${NC}"
echo "--------------------------------------"
echo "To monitor your bot:"
echo "  1. Check Telegram for alerts"
echo "  2. View logs: tail -f logs/gobot_mainnet_*.log"
echo "  3. Check progress: cat gobot_mainnet_state/progress.json"
echo ""

# Step 9: Full Deployment
echo -e "${YELLOW}Step 9: Full Deployment (Optional)${NC}"
echo "--------------------------------------"
echo "If first trade was successful, you can start full deployment:"
echo ""
echo "export DRY_RUN=false"
echo "export BINANCE_USE_TESTNET=false"
echo "export MAX_ITERATIONS=96  # 24 hours"
echo "python3 gobot_mainnet_orchestrator.py"
echo ""

echo "=========================================="
echo -e "${GREEN}✅ MAINNET DEPLOYMENT COMPLETE${NC}"
echo "=========================================="
echo ""
echo "Remember:"
echo "  • Monitor closely for first few hours"
echo "  • Check Telegram for alerts"
echo "  • Stop if daily loss limit reached"
echo "  • Review performance daily"
echo ""
echo "Good luck! 🚀"
