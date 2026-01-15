#!/bin/bash

# GOBOT QuantCrawler + Google Auth Setup
# Required for AI-powered trading signals

echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║          GOBOT QUANTCRAWLER SETUP - GOOGLE AUTH                ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""

echo "This script configures Google authentication for QuantCrawler."
echo ""

# Check for existing config
if [ -f .env ]; then
    echo "Found existing .env file"
    if grep -q "GOOGLE_EMAIL=" .env && ! grep -q "YOUR_" .env .env 2>/dev/null; then
        echo "Google credentials already configured!"
        echo ""
        grep "GOOGLE_EMAIL" .env
    fi
fi

echo ""
echo "Step 1: Enter your Google credentials"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

read -p "Google Email (e.g., your-email@gmail.com): " GOOGLE_EMAIL

echo ""
read -s -p "Google App Password (16 chars, hidden): " GOOGLE_PASSWORD
echo ""

echo ""
echo ""
echo "Step 2: Creating configuration..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Create/update .env with Google credentials
cat >> .env << EOF

# ====================================
# QUANTCRAWLER GOOGLE AUTH (Required for AI Analysis)
# ====================================
GOOGLE_EMAIL=$GOOGLE_EMAIL
GOOGLE_APP_PASSWORD=$GOOGLE_PASSWORD
EOF

echo ""
echo "✅ Configuration saved to .env"
echo ""

echo "Step 3: Testing QuantCrawler connection..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Test the QuantCrawler client
cd services/screenshot-service

echo ""
echo "Testing QuantCrawler client..."
timeout 10 node -e "
const quant = require('./quantcrawler-client.js');
if (quant.CONFIG.googleEmail && quant.CONFIG.googleAppPassword) {
  console.log('✅ Google credentials loaded');
  console.log('   Email:', quant.CONFIG.googleEmail);
  console.log('   App Password: ****' + quant.CONFIG.googleAppPassword.slice(-4));
} else {
  console.log('❌ Credentials not loaded');
  process.exit(1);
}
" 2>&1

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ QuantCrawler is ready for AI analysis!"
    echo ""
    echo "Next steps:"
    echo "  1. Run test: cd services/screenshot-service && node quantcrawler-client.js 1000PEPEUSDT chart_1m.png"
    echo "  2. Or run full workflow: node auto-trade.js 1000PEPEUSDT 5000"
else
    echo ""
    echo "❌ Configuration test failed"
    echo "Please check your credentials and try again"
fi
