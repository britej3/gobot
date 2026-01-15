#!/bin/bash

# ============================================================
# GOBOT Testnet Observer - 180 Minute Monitoring Session
# ============================================================

set -a
source /Users/britebrt/GOBOT/.env
set +a

# Configuration
TOTAL_DURATION_MINUTES=180
CHECK_INTERVAL_MINUTES=5
SYMBOL="1000PEPEUSDT"
TRADE_AMOUNT=5000

# Files
LOG_DIR="/Users/britebrt/GOBOT/logs"
OBSERVATION_LOG="$LOG_DIR/observer_$(date +%Y%m%d_%H%M%S).log"
TRADE_LOG="$LOG_DIR/trades_$(date +%Y%m%d_%H%M%S).json"
SUMMARY_LOG="$LOG_DIR/summary_$(date +%Y%m%d_%H%M%S).txt"

mkdir -p "$LOG_DIR"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$OBSERVATION_LOG"
}

log_section() {
    echo "" | tee -a "$OBSERVATION_LOG"
    echo "══════════════════════════════════════════════════════════════════" | tee -a "$OBSERVATION_LOG"
    echo "  $1" | tee -a "$OBSERVATION_LOG"
    echo "══════════════════════════════════════════════════════════════════" | tee -a "$OBSERVATION_LOG"
}

get_binance_balance() {
    API_KEY="$BINANCE_TESTNET_API"
    SECRET="$BINANCE_TESTNET_SECRET"
    TIMESTAMP=$(date +%s000)
    SIGNATURE=$(echo -n "timestamp=$TIMESTAMP" | openssl dgst -sha256 -hmac "$SECRET" 2>/dev/null | sed 's/.*= //')
    
    RESPONSE=$(curl -s -X GET "https://testnet.binancefuture.com/fapi/v2/balance?timestamp=$TIMESTAMP&signature=$SIGNATURE" \
        -H "X-MBX-APIKEY: $API_KEY" 2>/dev/null)
    
    echo "$RESPONSE" | grep -o '"asset":"USDT"[^}]*' | grep -o '"availableBalance":"[^"]*"' | cut -d'"' -f4
}

get_binance_positions() {
    API_KEY="$BINANCE_TESTNET_API"
    SECRET="$BINANCE_TESTNET_SECRET"
    TIMESTAMP=$(date +%s000)
    SIGNATURE=$(echo -n "timestamp=$TIMESTAMP" | openssl dgst -sha256 -hmac "$SECRET" 2>/dev/null | sed 's/.*= //')
    
    RESPONSE=$(curl -s -X GET "https://testnet.binancefuture.com/fapi/v2/account?timestamp=$TIMESTAMP&signature=$SIGNATURE" \
        -H "X-MBX-APIKEY: $API_KEY" 2>/dev/null)
    
    echo "$RESPONSE" | grep -o '"positionAmt":"[^"]*"' | grep -v '"0"' | grep -v '"0.0"' | grep -v '"0.00"' | wc -l
}

get_binance_open_orders() {
    API_KEY="$BINANCE_TESTNET_API"
    SECRET="$BINANCE_TESTNET_SECRET"
    TIMESTAMP=$(date +%s000)
    SIGNATURE=$(echo -n "timestamp=$TIMESTAMP&symbol=${SYMBOL}" | openssl dgst -sha256 -hmac "$SECRET" 2>/dev/null | sed 's/.*= //')
    
    curl -s -X GET "https://testnet.binancefuture.com/fapi/v1/openOrders?timestamp=$TIMESTAMP&symbol=${SYMBOL}&signature=$SIGNATURE" \
        -H "X-MBX-APIKEY: $API_KEY" 2>/dev/null | grep -o '"orderId"' | wc -l
}

get_market_data() {
    PRICE=$(curl -s "https://testnet.binancefuture.com/fapi/v1/ticker/price?symbol=$SYMBOL" 2>/dev/null | grep -o '"price":"[^"]*"' | cut -d'"' -f4)
    [ -z "$PRICE" ] && PRICE="0"
    
    CHANGE=$(curl -s "https://testnet.binancefuture.com/fapi/v1/ticker/24hr?symbol=$SYMBOL" 2>/dev/null | grep -o '"priceChangePercent":"[^"]*"' | cut -d'"' -f4)
    [ -z "$CHANGE" ] && CHANGE="0"
    
    echo "$PRICE|$CHANGE"
}

run_trading_cycle() {
    CYCLE=$((CYCLE + 1))
    ELAPSED_MINUTES=$(( ($(date +%s) - START_TIME) / 60 ))
    REMAINING_MINUTES=$((TOTAL_DURATION_MINUTES - ELAPSED_MINUTES))
    
    log_section "📊 CYCLE $CYCLE - Elapsed: ${ELAPSED_MINUTES}min | Remaining: ${REMAINING_MINUTES}min"
    
    # Get market data
    MARKET_DATA=$(get_market_data)
    PRICE=$(echo "$MARKET_DATA" | cut -d'|' -f1)
    CHANGE=$(echo "$MARKET_DATA" | cut -d'|' -f2)
    
    log "📈 Market: $SYMBOL = \$$PRICE (24h: ${CHANGE}%)"
    
    # Get Binance status
    BALANCE=$(get_binance_balance)
    POSITIONS=$(get_binance_positions)
    OPEN_ORDERS=$(get_binance_open_orders)
    
    log "💰 Balance: \$$BALANCE USDT"
    log "📋 Positions: $POSITIONS | Open Orders: $OPEN_ORDERS"
    
    # Run auto-trade
    log "🔄 Running auto-trade analysis..."
    TRADE_RESULT=$(cd /Users/britebrt/GOBOT/services/screenshot-service && \
        BINANCE_USE_TESTNET=true timeout 120 node auto-trade.js "$SYMBOL" "$TRADE_AMOUNT" 2>&1)
    
    # Parse result
    if echo "$TRADE_RESULT" | grep -q "✓ Signal sent to GOBOT"; then
        SIGNALS_RECEIVED=$((SIGNALS_RECEIVED + 1))
        DIRECTION=$(echo "$TRADE_RESULT" | grep "Direction:" | head -1 | awk '{print $2}')
        CONFIDENCE=$(echo "$TRADE_RESULT" | grep "Confidence:" | head -1 | awk '{print $2}')
        log "✅ Signal Sent: $DIRECTION (${CONFIDENCE})"
        
        echo "{\"cycle\": $CYCLE, \"timestamp\": \"$(date -Iseconds)\", \"symbol\": \"$SYMBOL\", \"price\": \"$PRICE\", \"direction\": \"$DIRECTION\", \"confidence\": \"$CONFIDENCE\", \"balance\": \"$BALANCE\"}" >> "$TRADE_LOG"
        
    elif echo "$TRADE_RESULT" | grep -q "✓ Analysis complete"; then
        SUCCESSFUL_ANALYSES=$((SUCCESSFUL_ANALYSES + 1))
        DIRECTION=$(echo "$TRADE_RESULT" | grep "Direction:" | head -1 | awk '{print $2}')
        CONFIDENCE=$(echo "$TRADE_RESULT" | grep "Confidence:" | head -1 | awk '{print $2}')
        log "✅ Analysis Complete: $DIRECTION (${CONFIDENCE})"
        
    elif echo "$TRADE_RESULT" | grep -q "✓ 1m captured"; then
        SUCCESSFUL_ANALYSES=$((SUCCESSFUL_ANALYSES + 1))
        log "✅ Screenshots Captured Successfully"
        
    else
        FAILED_CYCLES=$((FAILED_CYCLES + 1))
        log "⚠️  Analysis issue - will retry next cycle"
    fi
    
    # Check GOBOT status
    GOBOT_STATUS=$(curl -s http://localhost:8080/health 2>/dev/null)
    log "🤖 GOBOT Status: $([ "$GOBOT_STATUS" = "OK" ] && echo "✅ ONLINE" || echo "⚠️  CHECKING")"
    
    # Check screenshot service
    SCREENSHOT_STATUS=$(curl -s http://localhost:3456/health 2>/dev/null)
    log "📸 Screenshot: $(echo "$SCREENSHOT_STATUS" | grep -q "healthy" && echo "✅ ONLINE" || echo "⚠️  CHECKING")"
    
    # Summary
    log ""
    log "📊 Session Stats: Cycles=$CYCLE | Signals=$SIGNALS_RECEIVED | Analyses=$SUCCESSFUL_ANALYSES | Failed=$FAILED_CYCLES | Balance=\$$BALANCE"
}

generate_summary() {
    END_TIME=$(date +%s)
    DURATION_MINUTES=$(( (END_TIME - START_TIME) / 60 ))
    FINAL_BALANCE=$(get_binance_balance)
    FINAL_POSITIONS=$(get_binance_positions)
    
    cat > "$SUMMARY_LOG" << EOF
╔══════════════════════════════════════════════════════════════════════════╗
║              GOBOT TESTNET OBSERVATION - FINAL REPORT                   ║
╚══════════════════════════════════════════════════════════════════════════╝

📅 Session: $(date '+%Y-%m-%d %H:%M')
⏱️  Duration: ${DURATION_MINUTES} minutes (planned: 180)
📊 Pair: $SYMBOL | Amount: $TRADE_AMOUNT USDT

═══════════════════════════════════════════════════════════════════════════
                           RESULTS
═══════════════════════════════════════════════════════════════════════════

📈 Activity:
   • Cycles: $CYCLE
   • Signals: $SIGNALS_RECEIVED
   • Analyses: $SUCCESSFUL_ANALYSES
   • Failed: $FAILED_CYCLES

💰 Account:
   • Final Balance: \$$FINAL_BALANCE USDT
   • Open Positions: $FINAL_POSITIONS

📁 Logs:
   • $OBSERVATION_LOG
   • $TRADE_LOG

═══════════════════════════════════════════════════════════════════════════
EOF

    cat "$SUMMARY_LOG"
}

# Main
log_section "🚀 GOBOT TESTNET OBSERVATION - 180 MINUTES"

log "Starting session... Check interval: ${CHECK_INTERVAL_MINUTES}min"
log "Logs: $OBSERVATION_LOG"

# Ensure services
cd /Users/britebrt/GOBOT && ./gobot > /tmp/gobot.log 2>&1 &
sleep 2

cd /Users/britebrt/GOBOT/services/screenshot-service && node server.js > /tmp/screenshot.log 2>&1 &
sleep 3

# Initial status
log_section "📊 INITIAL STATUS"
log "Balance: \$$(get_binance_balance) USDT"
log "Positions: $(get_binance_positions)"

# Run cycles
TOTAL_CYCLES=$((TOTAL_DURATION_MINUTES / CHECK_INTERVAL_MINUTES))
log_section "⏱️  STARTING ${TOTAL_CYCLES} CYCLES"

for i in $(seq 1 $TOTAL_CYCLES); do
    run_trading_cycle
    [ $i -lt $TOTAL_CYCLES ] && sleep $((CHECK_INTERVAL_MINUTES * 60))
done

log_section "📋 FINAL SUMMARY"
generate_summary
log "✅ COMPLETE"
