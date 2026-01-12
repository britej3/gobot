#!/bin/bash

# Mainnet API Key Fix Script
# Identifies and helps fix signature validation errors

set -e

echo "🚨 BINANCE MAINNET API DIAGNOSIS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Load env file values
API_KEY=$(awk -F'=' '/^BINANCE_API_KEY=/ {print $2}' .env | tr -d '"' | tr -d ' ')
API_SECRET=$(awk -F'=' '/^BINANCE_API_SECRET=/ {print $2}' .env | tr -d '"' | tr -d ' ')

echo "📋 Current .env Configuration:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "BINANCE_API_KEY:     ${API_KEY:0:15}...${API_KEY: -15}"
echo "BINANCE_API_SECRET:  ${API_SECRET:0:15}...${API_SECRET: -15}"
echo ""

# Check if key and secret are identical
if [ "$API_KEY" = "$API_SECRET" ]; then
    echo "❌ ${RED}CRITICAL ISSUE FOUND!${NC}"
    echo ""
    echo "Your API Key and API Secret are ${RED}IDENTICAL${NC}!"
    echo "This is why you're getting 'Signature for this request is not valid.'"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔧 SOLUTION: Update Your API Credentials"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "1. Go to Binance API Management:"
    echo "   ${BLUE}https://www.binance.com/en/my/settings/api-management${NC}"
    echo ""
    echo "2. ${YELLOW}Do one of the following:${NC}"
    echo ""
    echo "   Option A: Create NEW API Keys"
    echo "   ──────────────────────────────────────────────────"
    echo "   • Click 'Create API' button"
    echo "   • Choose 'System Generated' or 'Self Generated'"
    echo "   • Label it 'Cognee-Mainnet' or similar"
    echo "   • ${GREEN}ENABLE these permissions:${NC}"
    echo "     ☑ Enable Reading"
    echo "     ☑ Enable Futures"
    echo "   • Set IP restriction (recommended) or leave unrestricted"
    echo "   • Complete 2FA verification"
    echo "   • ${GREEN}Copy BOTH the API Key AND Secret${NC}"
    echo "   • ${YELLOW}Secret is only shown once - save it immediately!${NC}"
    echo ""
    echo "   Option B: Verify Existing Keys"
    echo "   ──────────────────────────────────────────────────"
    echo "   • Find your existing API key in the list"
    echo "   • Click 'Edit' or 'View'"
    echo "   • Verify these are enabled:"
    echo "     ☑ Enable Reading"
    echo "     ☑ Enable Futures"
    echo "   • Check your IP is whitelisted (if using restrictions)"
    echo "   • If needed, generate a new secret (may need to recreate)"
    echo ""
    echo "3. ${GREEN}Update your .env file:${NC}"
    echo ""
    echo "   Make sure you have TWO DIFFERENT values:"
    echo ""
    cat << 'EOF'
   # CORRECT FORMAT (different values):
   BINANCE_API_KEY=LpV3kD3f9TqR8s... (long string from Binance)
   BINANCE_API_SECRET=sY9mN2vB5xK7pQ... (DIFFERENT long string from Binance)
   BINANCE_USE_TESTNET=false

   # YOUR CURRENT FORMAT (WRONG - same values):
   # BINANCE_API_KEY=LpV3kD3f9TqR8s... 
   # BINANCE_API_SECRET=LpV3kD3f9TqR8s... ← THIS MUST BE DIFFERENT!
EOF
    echo ""
    echo "4. ${YELLOW}After updating .env, test again:${NC}"
    echo "   ${BLUE}./test_mainnet_quick.sh${NC}"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    # Offer to create a verification script
    read -p "Would you like me to create a verification script for after you update keys? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cat > verify_mainnet_credentials.sh << 'EOF'
#!/bin/bash
# Verification script - run after updating .env
echo "Verifying Binance Mainnet Credentials..."
echo ""

API_KEY=$(awk -F'=' '/^BINANCE_API_KEY=/ {print $2}' .env | tr -d '"' | tr -d ' ')
API_SECRET=$(awk -F'=' '/^BINANCE_API_SECRET=/ {print $2}' .env | tr -d '"' | tr -d ' ')

if [ "$API_KEY" = "$API_SECRET" ]; then
    echo "❌ STILL BROKEN: Key and secret are identical!"
    exit 1
fi

if [ -z "$API_KEY" ] || [ -z "$API_SECRET" ]; then
    echo "❌ MISSING: Key or secret is empty!"
    exit 1
fi

if [ ${#API_KEY} -lt 50 ] || [ ${#API_SECRET} -lt 50 ]; then
    echo "❌ INVALID: Key or secret appears too short!"
    exit 1
fi

echo "✅ Key and secret are different (good!)"
echo "✅ Key length: ${#API_KEY} characters"
echo "✅ Secret length: ${#API_SECRET} characters"
echo ""
echo "Testing API connection..."

TIMESTAMP=$(date +%s000)
SIGNATURE=$(echo -n "timestamp=$TIMESTAMP" | openssl dgst -sha256 -hmac "$API_SECRET" | sed 's/^.* //')

RESPONSE=$(curl -s -H "X-MBX-APIKEY: $API_KEY" \
  "https://fapi.binance.com/fapi/v2/account?timestamp=$TIMESTAMP&signature=$SIGNATURE")

if echo "$RESPONSE" | grep -q '"totalWalletBalance"'; then
    echo "✅ API connection successful!"
    BALANCE=$(echo "$RESPONSE" | grep -o '"totalWalletBalance":"[^"]*"' | grep -o '[0-9.]*')
    echo "💰 Balance: $BALANCE USDT"
else
    echo "❌ API connection failed"
    echo "Response: $RESPONSE"
    exit 1
fi
EOF
        chmod +x verify_mainnet_credentials.sh
        echo "✅ Created verify_mainnet_credentials.sh"
        echo "Run this after fixing your .env to verify everything works!"
    fi
    
else
    echo "✅ Key and secret are different (good)"
    echo "🔍 The issue may be:"
    echo "   - Incorrect API secret (typo)"
    echo "   - API key doesn't have Futures enabled"
    echo "   - IP restriction issue"
    echo ""
    echo "Try running ./test_mainnet_quick.sh for detailed diagnostics"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"