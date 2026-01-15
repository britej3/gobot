#!/bin/bash

echo "=========================================="
echo "GOBOT MAINNET CONFIGURATION SETUP"
echo "=========================================="
echo ""
echo "This script will help you configure GOBOT for mainnet trading."
echo ""

read -p "Enter your BINANCE API KEY: " api_key
read -s -p "Enter your BINANCE API SECRET: " api_secret
echo ""

read -p "Enter your TELEGRAM BOT TOKEN (optional, press Enter to skip): " tg_token
read -p "Enter your TELEGRAM CHAT ID (optional, press Enter to skip): " tg_chat
echo ""

read -p "Enter KILL SWITCH password (default: STOP123): " kill_pw
kill_pw=${kill_pw:-STOP123}

echo ""
echo "Configuring .env.mainnet..."

# Update API keys
sed -i '' "s|YOUR_BINANCE_MAINNET_API_KEY_HERE|$api_key|g" .env.mainnet
sed -i '' "s|YOUR_BINANCE_MAINNET_API_SECRET_HERE|$api_secret|g" .env.mainnet

# Update Telegram
if [ -n "$tg_token" ]; then
    sed -i '' "s|YOUR_TELEGRAM_BOT_TOKEN_HERE|$tg_token|g" .env.mainnet
fi

if [ -n "$tg_chat" ]; then
    sed -i '' "s|YOUR_TELEGRAM_CHAT_ID_HERE|$tg_chat|g" .env.mainnet
fi

# Update kill switch
sed -i '' "s|KILL_SWITCH_PASSWORD=STOP123|KILL_SWITCH_PASSWORD=$kill_pw|g" .env.mainnet

echo ""
echo "Configuration updated!"
echo ""
echo "Next steps:"
echo "  1. Validate: ./mainnet-deploy.sh check"
echo "  2. Test connectivity: ./mainnet-deploy.sh connect"
echo "  3. Deploy to mainnet: ./mainnet-deploy.sh deploy --confirm"
