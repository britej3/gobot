#!/bin/bash

# Quick launcher for GOBOT TUI

echo -e "🚀 Starting GOBOT TUI Dashboard..."
echo -e "📊 Press Ctrl+C to exit"
echo ""

# Check if bot is running
if ! pgrep -f "cognee" > /dev/null; then
    echo -e "⚠️  Bot is not running. Starting it first..."
    
    # Export environment variables
    export $(grep -v '^#' .env | grep -v '^#' | grep BINANCE)
    
    # Start bot in background
    ./cognee > startup.log 2>&1 &
    BOT_PID=$!
    
    echo -e "✅ Bot started (PID: $BOT_PID)"
    echo -e "⏳ Waiting 10 seconds for initialization..."
    sleep 10
fi

echo -e "✅ Bot is running"
echo -e "📊 Starting TUI dashboard..."
echo ""

# Start the dashboard
./tui_dashboard.sh
