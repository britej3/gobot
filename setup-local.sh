#!/bin/bash

# ═══════════════════════════════════════════════════════════════════════════
# GOBOT Local Setup Script
# Installs all dependencies for local development
# ═══════════════════════════════════════════════════════════════════════════

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=============================================="
echo "  GOBOT Local Setup"
echo "=============================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check Node.js
echo "[1/3] Checking Node.js..."
if command -v node &> /dev/null; then
    echo -e "      ${GREEN}✓ Node.js $(node --version)${NC}"
else
    echo -e "      ${YELLOW}Node.js not found. Installing...${NC}"
    brew install node  # macOS
    # For Linux: curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    # For Windows: Download from https://nodejs.org/
fi

# Check npm
echo "[2/3] Checking npm..."
if command -v npm &> /dev/null; then
    echo -e "      ${GREEN}✓ npm $(npm --version)${NC}"
else
    echo -e "      ${YELLOW}npm not found${NC}"
    exit 1
fi

# Install dependencies
echo "[3/3] Installing dependencies..."
echo ""

# Install Puppeteer
echo "Installing Puppeteer..."
cd n8n/scripts
npm install puppeteer 2>&1 | tail -10
cd "$SCRIPT_DIR"

# Install N8N globally (optional)
if ! command -v n8n &> /dev/null; then
    echo ""
    echo "Installing N8N globally..."
    npm install -g n8n 2>&1 | tail -5
fi

# Create directories
echo ""
echo "Creating directories..."
mkdir -p n8n-sessions
mkdir -p n8n/workflows

# Create .env template if not exists
if [[ ! -f ".env" ]]; then
    echo ""
    echo "Creating .env template..."
    cat >> .env << 'EOF'

# ====================================
# QuantCrawler Configuration
# ====================================
# Get credentials from your QuantCrawler account
# For 2FA accounts, use App Password:
# https://myaccount.google.com/apppasswords
QUANTCRAWLER_EMAIL=your-email@gmail.com
QUANTCRAWLER_PASSWORD=your-16-char-app-password
EOF
    echo -e "      ${GREEN}✓ Created .env with QuantCrawler template${NC}"
else
    echo ""
    echo -e "${YELLOW}.env already exists. Add QUANTCRAWLER_EMAIL and QUANTCRAWLER_PASSWORD manually.${NC}"
fi

echo ""
echo "=============================================="
echo "  Setup Complete!"
echo "=============================================="
echo ""
echo "Next steps:"
echo ""
echo "1. Edit .env and add your QuantCrawler credentials:"
echo "   QUANTCRAWLER_EMAIL=your-email@gmail.com"
echo "   QUANTCRAWLER_PASSWORD=your-app-password"
echo ""
echo "2. Start all services:"
echo "   ./start-all.sh"
echo ""
echo "3. Test the system:"
echo "   curl -X POST http://localhost:3456/webhook \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"symbol\":\"1000PEPEUSDT\",\"account_balance\":1000}'"
echo ""
echo "4. Import N8N workflow:"
echo "   1. Open http://localhost:5678"
echo "   2. Login: gobot / secure_password"
echo "   3. Import n8n/workflows/04-quantcrawler-analysis.json"
echo ""
