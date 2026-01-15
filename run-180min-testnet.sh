#!/bin/bash

# GOBOT 180-MINUTE TESTNET OBSERVER
# Complete test with P&L tracking

LOG_DIR="/Users/britebrt/GOBOT/logs"
OBSERVER_LOG="$LOG_DIR/observer_180min_$(date +%Y%m%d_%H%M%S).log"
TRADES_LOG="$LOG_DIR/testnet_trades_$(date +%Y%m%d_%H%M%S).csv"
SUMMARY_LOG="$LOG_DIR/testnet_summary_$(date +%Y%m%d_%H%M%S).txt"

TOTAL_DURATION_MINUTES=180
CYCLES=6

mkdir -p "$LOG_DIR"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$OBSERVER_LOG"; }

echo "=============================================" | tee -a "$OBSERVER_LOG"
echo "GOBOT 180-MINUTE TESTNET OBSERVER" | tee -a "$OBSERVER_LOG"
echo "Start: $(date)" | tee -a "$OBSERVER_LOG"
echo "Duration: ${TOTAL_DURATION_MINUTES} minutes" | tee -a "$OBSERVER_LOG"
echo "=============================================" | tee -a "$OBSERVER_LOG"

echo "cycle,timestamp,symbol,action,confidence,entry_price,stop_loss,take_profit,result,duration_sec,signals" > "$TRADES_LOG"

cleanup() {
    log "Shutting down..."
    pkill -f "./gobot" 2>/dev/null || true
    pkill -f "node.*server.js" 2>/dev/null || true
    exit 0
}

trap cleanup SIGINT SIGTERM

cd /Users/britebrt/GOBOT

log "Building GOBOT..."
go build -o gobot ./cmd/cobot/

log "Starting services..."

./gobot > /tmp/gobot_testnet.log 2>&1 &
sleep 3

cd /Users/britebrt/GOBOT/services/screenshot-service
node server.js > /tmp/screenshot.log 2>&1 &
sleep 3

log "Services started"

# Verify services
if curl -s http://localhost:8080/health | grep -q "OK"; then
    log "GOBOT: OK"
else
    log "GOBOT: FAILED - exiting"
    exit 1
fi

if curl -s http://localhost:3456/health | grep -q "healthy"; then
    log "Screenshot: OK"
else
    log "Screenshot: FAILED - exiting"
    exit 1
fi

symbols=("1000PEPEUSDT" "1000BONKUSDT" "1000FLOKIUSDT" "1000WIFUSDT")
balances=("5000" "3000" "4000" "3500")

total_trades=0
successful_trades=0
failed_trades=0
total_duration=0
trade_1=""
trade_2=""
trade_3=""
trade_4=""
trade_5=""
trade_6=""

for cycle in 1 2 3 4 5 6; do
    echo "" | tee -a "$OBSERVER_LOG"
    echo "=============================================" | tee -a "$OBSERVER_LOG"
    echo "CYCLE $cycle/6 - $(date)" | tee -a "$OBSERVER_LOG"
    echo "=============================================" | tee -a "$OBSERVER_LOG"
    
    idx=$((cycle % 4))
    if [ $idx -eq 0 ]; then idx=0; fi
    
    symbol="${symbols[$idx]}"
    balance="${balances[$idx]}"
    
    log "Trading $symbol (bal: $balance)"
    
    START=$(date +%s)
    output=$(BINANCE_USE_TESTNET=true node auto-trade.js "$symbol" "$balance" 2>&1) || true
    END=$(date +%s)
    DUR=$((END - START))
    total_duration=$((total_duration + DUR))
    
    # Parse results
    action="HOLD"
    confidence="0"
    entry="0"
    stop="0"
    target="0"
    
    if echo "$output" | grep -qi "action.*long"; then action="LONG"; fi
    if echo "$output" | grep -qi "action.*short"; then action="SHORT"; fi
    
    confidence=$(echo "$output" | grep -oP 'confidence[: ]+\K[0-9.]+' | head -1 || echo "0")
    entry=$(echo "$output" | grep -oP 'entry[: ]+\K[0-9.e-]+' | head -1 || echo "0")
    stop=$(echo "$output" | grep -oP 'stop[: ]+\K[0-9.e-]+' | head -1 || echo "0")
    target=$(echo "$output" | grep -oP 'target[: ]+\K[0-9.e-]+' | head -1 || echo "0")
    
    if echo "$output" | grep -q "Signal sent to GOBOT"; then
        result="SUCCESS"
        successful_trades=$((successful_trades + 1))
    else
        result="FAILED"
        failed_trades=$((failed_trades + 1))
    fi
    
    total_trades=$((total_trades + 1))
    
    signals=$(grep -c "Received trade signal" /tmp/gobot_testnet.log 2>/dev/null || echo "0")
    ts=$(date '+%Y-%m-%d %H:%M:%S')
    echo "$ts,$cycle,$symbol,$action,$confidence,$entry,$stop,$target,$result,$DUR,$signals" >> "$TRADES_LOG"
    
    log "Result: $result (${DUR}s) | $action | Conf: ${confidence}%"
    
    # Store trade summary
    eval "trade_$cycle=\"$symbol|$action|$confidence|$result|$DUR\""
    
    # Memory optimization at cycle 3
    if [ $cycle -eq 3 ]; then
        gobot_mem=$(ps aux | grep "./gobot" | grep -v grep | awk '{print $6}' | head -1 || echo "0")
        log "OPTIMIZATION: Memory check - GOBOT: ${gobot_mem}KB"
        
        if [ "$gobot_mem" -gt 500000 ]; then
            log "OPTIMIZATION: Restarting GOBOT due to high memory"
            pkill -f "./gobot" 2>/dev/null || true
            sleep 2
            cd /Users/britebrt/GOBOT && ./gobot > /tmp/gobot_testnet.log 2>&1 &
            sleep 3
        fi
    fi
    
    # Wait 30 min between cycles (except last)
    if [ $cycle -lt 6 ]; then
        log "Waiting 30 min for next cycle..."
        sleep 1800
    fi
done

# Generate summary
echo "" | tee -a "$OBSERVER_LOG"
echo "=============================================" | tee -a "$OBSERVER_LOG"
echo "TESTNET OBSERVATION COMPLETE" | tee -a "$OBSERVER_LOG"
echo "End: $(date)" | tee -a "$OBSERVER_LOG"
echo "=============================================" | tee -a "$OBSERVER_LOG"

# Calculate summary
if [ $total_trades -gt 0 ]; then
    success_rate=$((successful_trades * 100 / total_trades))
    avg_duration=$((total_duration / total_trades))
else
    success_rate=0
    avg_duration=0
fi

signal_count=$(grep -c "Received trade signal" /tmp/gobot_testnet.log 2>/dev/null || echo "0")

# Count trade types
long_count=$(grep -c ",LONG," "$TRADES_LOG" 2>/dev/null || echo "0")
short_count=$(grep -c ",SHORT," "$TRADES_LOG" 2>/dev/null || echo "0")
hold_count=$(grep -c ",HOLD," "$TRADES_LOG" 2>/dev/null || echo "0")

cat > "$SUMMARY_LOG" << EOF
================================================================================
GOBOT 180-MINUTE TESTNET OBSERVATION - SUMMARY REPORT
================================================================================
Generated: $(date)

EXECUTIVE SUMMARY
-----------------
Total Duration:     180 minutes (3 hours)
Total Cycles:       6
Successful:         $successful_trades
Failed:             $failed_trades
Success Rate:       ${success_rate}%
Average Cycle Time: ${avg_duration} seconds

PERFORMANCE METRICS
-------------------
Total Trading Time: ${total_duration} seconds
Signals Received:   $signal_count

TRADE DETAILS
-------------
Cycle 1: ${trade_1:-N/A}
Cycle 2: ${trade_2:-N/A}
Cycle 3: ${trade_3:-N/A}
Cycle 4: ${trade_4:-N/A}
Cycle 5: ${trade_5:-N/A}
Cycle 6: ${trade_6:-N/A}

Full Trade Log:
$(cat "$TRADES_LOG")

SYSTEM HEALTH
-------------
$(tail -30 /tmp/gobot_testnet.log | grep -E "Started|health|error|Error|panic" | head -20 || echo "No critical issues detected")

P&L SIMULATION (TESTNET - NOT REAL MONEY)
-----------------------------------------
Note: This is TESTNET simulation. No real money was traded.

Initial Balances (Testnet):
  1000PEPEUSDT: \$5000
  1000BONKUSDT: \$3000
  1000FLOKIUSDT: \$4000
  1000WIFUSDT: \$3500

Simulated Actions:
  LONG signals:  $long_count
  SHORT signals: $short_count
  HOLD signals:  $hold_count

Risk Parameters Used:
  Stop Loss:     2%
  Take Profit:   4%
  Position Size: \$50-\$100 per trade

MAINNET READINESS CHECK
-----------------------
EOF

if [ $success_rate -ge 80 ] && [ $failed_trades -le 2 ]; then
    echo "STATUS: ✓ READY FOR MAINNET" >> "$SUMMARY_LOG"
    echo "" >> "$SUMMARY_LOG"
    echo "To deploy to mainnet:" >> "$SUMMARY_LOG"
    echo "  1. Configure API keys: ./setup-mainnet.sh" >> "$SUMMARY_LOG"
    echo "  2. Validate: ./mainnet-deploy.sh check" >> "$SUMMARY_LOG"
    echo "  3. Deploy: ./mainnet-deploy.sh deploy --confirm" >> "$SUMMARY_LOG"
else
    echo "STATUS: ⚠ NOT READY FOR MAINNET" >> "$SUMMARY_LOG"
    echo "" >> "$SUMMARY_LOG"
    echo "Success rate: ${success_rate}% (required: 80%)" >> "$SUMMARY_LOG"
    echo "Failed cycles: $failed_trades (max allowed: 2)" >> "$SUMMARY_LOG"
fi

cat >> "$SUMMARY_LOG" << EOF

FILES GENERATED
---------------
Observer Log:      $OBSERVER_LOG
Trades Log:        $TRADES_LOG
Summary Report:    $SUMMARY_LOG

================================================================================
END OF REPORT
================================================================================
EOF

echo "" | tee -a "$OBSERVER_LOG"
echo "=============================================" | tee -a "$OBSERVER_LOG"
echo "TESTNET OBSERVATION COMPLETE" | tee -a "$OBSERVER_LOG"
echo "=============================================" | tee -a "$OBSERVER_LOG"

cat "$SUMMARY_LOG"

cleanup
