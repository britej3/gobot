#!/bin/bash

# GOBOT TUI Dashboard
# Real-time monitoring for your trading bot

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration
LOG_FILE="startup.log"
UPDATE_INTERVAL=2  # seconds

# Check if log file exists
if [ ! -f "$LOG_FILE" ]; then
    echo -e "${RED}❌ Error: $LOG_FILE not found${NC}"
    echo "Start the bot first with: ./cognee"
    exit 1
fi

# Get current timestamp for log parsing
get_timestamp() {
    date '+%Y-%m-%dT%H:%M:%S'
}

# Parse JSON logs (macOS compatible - no tac)
parse_log() {
    local log_file=$1
    local pattern=$2
    local count=${3:-1}
    
    if command -v jq &> /dev/null; then
        # Use tail + grep for macOS compatibility
        tail -1000 "$log_file" 2>/dev/null | grep "$pattern" | tail -n "$count" | jq -r '.msg' 2>/dev/null | tail -1
    else
        # Fallback to grep only
        tail -1000 "$log_file" 2>/dev/null | grep "$pattern" | tail -n "$count" | sed 's/.*"msg":"\([^"]*\)".*/\1/' | tail -1
    fi
}

# Get log count
get_log_count() {
    local pattern=$1
    grep -c "$pattern" "$LOG_FILE" 2>/dev/null || echo "0"
}

# Get account balance from logs (macOS compatible)
get_balance() {
    if command -v jq &> /dev/null; then
        tail -1000 "$LOG_FILE" | grep "total_wallet_balance" | tail -1 | jq -r '.total_wallet_balance // .available_margin' 2>/dev/null
    else
        tail -1000 "$LOG_FILE" | grep "total_wallet_balance" | tail -1 | sed 's/.*"total_wallet_balance":"\([^"]*\)".*/\1/'
    fi
}

# Get recent trades
get_recent_trades() {
    grep -a "TRADE\|FVG opportunity\|trading decision\|position\|ORDER" "$LOG_FILE" | tail -5 | sed 's/.*"msg":"\([^"]*\)".*/\1/' | head -5
}

# Get errors (macOS compatible)
get_errors() {
    tail -500 "$LOG_FILE" | grep '"level":"error"\|"level":"fatal"' | tail -3 | sed 's/.*"msg":"\([^"]*\)".*/\1/'
}

# Get watched symbols (macOS compatible)
get_symbols() {
    local symbols=$(grep "WATCHLIST_SYMBOLS" .env 2>/dev/null | cut -d'"' -f2 | tr ',' '\n' | wc -l | tr -d ' ')
    if [ -z "$symbols" ] || [ "$symbols" -eq 0 ]; then
        echo "43"
    else
        echo "$symbols"
    fi
}

# Main dashboard loop
clear
echo -e "${BOLD}${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BOLD}${CYAN}║         GOBOT Mid-Cap Trading Dashboard                    ║${NC}"
echo -e "${BOLD}${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if bot is running
if ! pgrep -f "cognee" > /dev/null; then
    echo -e "${RED}⚠️  Bot is not running${NC}"
    echo "Start with: ./cognee"
    exit 1
fi

echo -e "${GREEN}✅ Bot is running (PID: $(pgrep -f cognee))${NC}"
echo ""

# Display dashboard
while true; do
    # Move cursor to top of dashboard area
    tput cup 8 0
    
    # Get current status
    BALANCE=$(get_balance)
    FVG_OPPS=$(get_log_count "FVG opportunity detected")
    TRADE_DECISIONS=$(get_log_count "trading decision received")
    SYMBOL_COUNT=$(get_symbols)
    
    # Check for errors
    ERRORS=$(get_errors)
    
    # Status
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${BLUE}  REAL-TIME STATUS${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # Account
    if [ -n "$BALANCE" ]; then
        echo -e "${GREEN}💰 Account Balance: ${BOLD}$BALANCE USDT${NC}"
    else
        echo -e "${YELLOW}⏳ Waiting for balance info...${NC}"
    fi
    
    # Mode
    if grep -q "TESTNET (Safe)" "$LOG_FILE"; then
        echo -e "${YELLOW}🧪 Mode: TESTNET (Paper Trading)${NC}"
    else
        echo -e "${RED}🚨 Mode: MAINNET (Real Money)${NC}"
    fi
    
    # Monitoring
    echo -e "${CYAN}📊 Monitoring: ${SYMBOL_COUNT} mid-cap assets${NC}"
    echo ""
    
    # Trading Stats
    echo -e "${BOLD}${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${PURPLE}  TRADING STATS${NC}"
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}🎯 FVG Opportunities Detected: $FVG_OPPS${NC}"
    echo -e "${GREEN}🤖 Trade Decisions Made: $TRADE_DECISIONS${NC}"
    echo ""
    
    # Recent Activity
    echo -e "${BOLD}${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${YELLOW}  RECENT ACTIVITY${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    RECENT=$(get_recent_trades)
    if [ -n "$RECENT" ]; then
        echo "$RECENT" | while read -r line; do
            if [ -n "$line" ]; then
                echo -e "${CYAN}•${NC} $line"
            fi
        done
    else
        echo -e "${YELLOW}⏳ No recent trading activity${NC}"
    fi
    echo ""
    
    # Errors
    if [ -n "$ERRORS" ]; then
        echo -e "${BOLD}${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${BOLD}${RED}  ⚠️  RECENT ERRORS${NC}"
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo "$ERRORS" | while read -r line; do
            if [ -n "$line" ]; then
                echo -e "${RED}✗${NC} $line"
            fi
        done
        echo ""
    fi
    
    # Timestamp
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}Last Update: $(date '+%Y-%m-%d %H:%M:%S')${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # Instructions
    echo ""
    echo -e "${YELLOW}Press Ctrl+C to exit${NC}"
    
    # Wait before next update
    sleep $UPDATE_INTERVAL
done
