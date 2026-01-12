#!/bin/bash

# GOBOT API Key Setup Script
# This script helps you configure your Binance API keys for the aggressive scalper AI

echo "🚀 GOBOT Aggressive Scalper AI - API Key Setup"
echo "=============================================="
echo ""

# Function to safely read sensitive input
read_sensitive() {
    local prompt="$1"
    local input=""
    while IFS= read -p "$prompt" -r -s -n 1 char; do
        if [[ $char == $'\x08' || $char == $'\x7f' ]]; then
            if [[ ${#input} -gt 0 ]]; then
                input="${input%?}"
                echo -ne "\b \b"
            fi
        else
            input+="$char"
            echo -n "*"
        fi
    done
    echo
    echo "$input"
}

# Check if keys are already set
if [[ -n "$BINANCE_API_KEY" && -n "$BINANCE_API_SECRET" ]]; then
    echo "✅ API keys are already configured!"
    echo "Current configuration:"
    echo "BINANCE_API_KEY: ${BINANCE_API_KEY:0:10}..."
    echo "BINANCE_USE_TESTNET: $BINANCE_USE_TESTNET"
    echo ""
    read -p "Do you want to reconfigure? (y/N): " reconfigure
    if [[ ! "$reconfigure" =~ ^[Yy]$ ]]; then
        echo "Setup cancelled. Current configuration preserved."
        exit 0
    fi
fi

echo "🔑 Binance API Configuration"
echo ""
echo "⚠️  IMPORTANT NOTES:"
echo "- Start with TESTNET for safety (recommended)"
echo "- Mainnet is for real money trading (high risk)"
echo "- Ensure your API keys have futures trading permissions"
echo ""

# Choose network
read -p "Use Binance Testnet? (recommended for testing) (Y/n): " use_testnet
if [[ "$use_testnet" =~ ^[Nn]$ ]]; then
    export BINANCE_USE_TESTNET="false"
    echo "🚨 WARNING: You selected MAINNET (real money trading!)"
    read -p "Are you sure you want to use mainnet? Type 'YES' to confirm: " confirm
    if [[ "$confirm" != "YES" ]]; then
        echo "Switching to testnet for safety..."
        export BINANCE_USE_TESTNET="true"
    fi
else
    export BINANCE_USE_TESTNET="true"
    echo "✅ Using Binance Testnet (safe for testing)"
fi

echo ""
echo "🔐 Enter your Binance API credentials:"
echo ""

# Read API Key
echo -n "Binance API Key: "
API_KEY=$(read_sensitive "Enter API Key: ")
export BINANCE_API_KEY="$API_KEY"

# Read API Secret
echo -n "Binance API Secret: "
API_SECRET=$(read_sensitive "Enter API Secret: ")
export BINANCE_API_SECRET="$API_SECRET"

echo ""
echo "🔍 Verifying configuration..."
echo "BINANCE_API_KEY: ${BINANCE_API_KEY:0:10}..."
echo "BINANCE_USE_TESTNET: $BINANCE_USE_TESTNET"
echo ""

# Save to shell profile
read -p "Save configuration to ~/.bashrc? (recommended) (Y/n): " save_profile
if [[ ! "$save_profile" =~ ^[Nn]$ ]]; then
    echo "" >> ~/.bashrc
    echo "# GOBOT Binance API Configuration" >> ~/.bashrc
    echo "export BINANCE_API_KEY=\"$BINANCE_API_KEY\"" >> ~/.bashrc
    echo "export BINANCE_API_SECRET=\"$API_SECRET\"" >> ~/.bashrc
    echo "export BINANCE_USE_TESTNET=\"$BINANCE_USE_TESTNET\"" >> ~/.bashrc
    echo "✅ Configuration saved to ~/.bashrc"
    echo ""
    echo "To apply changes immediately, run:"
    echo "source ~/.bashrc"
fi

# Create .env file for Docker/container support
echo ""
read -p "Create .env file? (recommended for containers) (Y/n): " create_env
if [[ ! "$create_env" =~ ^[Nn]$ ]]; then
    cat > .env << EOF
# GOBOT Binance API Configuration
BINANCE_API_KEY=$BINANCE_API_KEY
BINANCE_API_SECRET=$API_SECRET
BINANCE_USE_TESTNET=$BINANCE_USE_TESTNET

# Optional: Other configurations
OLLAMA_BASE_URL=http://localhost:11454
OLLAMA_MODEL=LiquidAI/LFM2.5-1.2B-Instruct-GGUF/LFM2.5-1.2B-Instruct-Q8_0.gguf
EOF
    echo "✅ .env file created"
fi

echo ""
echo "🎉 API Key Setup Complete!"
echo "=============================================="
echo ""
echo "✅ Configuration Summary:"
echo "- Binance API Key: Configured"
echo "- Binance API Secret: Configured"
echo "- Network: $([ "$BINANCE_USE_TESTNET" == "true" ] && echo "Testnet (Safe)" || echo "Mainnet (Real Money)")"
echo ""
echo "🚀 NEXT STEPS:"
echo "1. Apply environment: source ~/.bashrc (if saved)"
echo "2. Start the platform: ./cognee"
echo "3. Monitor logs: tail -f startup.log"
echo ""
echo "⚠️  SECURITY REMINDER:"
echo "- Never share your API keys"
echo "- Use testnet for initial testing"
echo "- Monitor your API usage regularly"
echo ""
echo "Your aggressive scalper AI is ready to trade! 🎯"