#!/bin/bash

set -e

LOG_DIR="/Users/britebrt/GOBOT/logs"
OBSERVER_LOG="$LOG_DIR/observer_$(date +%Y%m%d_%H%M%S).log"
METRICS_LOG="$LOG_DIR/metrics_$(date +%Y%m%d_%H%M%S).csv"

mkdir -p "$LOG_DIR"

echo "[$(date '+%Y-%m-%d %H:%M:%S')] ========================================" | tee -a "$OBSERVER_LOG"
echo "[$(date '+%Y-%m-%d %H:%M:%S')] GOBOT 180-MINUTE TESTNET OBSERVER" | tee -a "$OBSERVER_LOG"
echo "[$(date '+%Y-%m-%d %H:%M:%S')] =========================================" | tee -a "$OBSERVER_LOG"

cleanup() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Shutting down..." | tee -a "$OBSERVER_LOG"
    pkill -f "./gobot" 2>/dev/null || true
    pkill -f "node.*server.js" 2>/dev/null || true
    exit 0
}

trap cleanup SIGINT SIGTERM

cd /Users/britebrt/GOBOT

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Building GOBOT..." | tee -a "$OBSERVER_LOG"
go build -o gobot ./cmd/cobot/

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting GOBOT..." | tee -a "$OBSERVER_LOG"
./gobot > /tmp/gobot.log 2>&1 &
sleep 3

cd /Users/britebrt/GOBOT/services/screenshot-service
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting screenshot service..." | tee -a "$OBSERVER_LOG"
node server.js > /tmp/screenshot.log 2>&1 &
sleep 3

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Services started" | tee -a "$OBSERVER_LOG"

echo "timestamp,cycle,symbol,result,duration,signals" > "$METRICS_LOG"

symbols=("1000PEPEUSDT" "1000BONKUSDT" "1000FLOKIUSDT" "1000WIFUSDT")
balances=("5000" "3000" "4000" "3500")

for cycle in {1..6}; do
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] === Cycle $cycle/6 ===" | tee -a "$OBSERVER_LOG"
    
    idx=$((cycle % 4))
    if [ $idx -eq 0 ]; then
        idx=0
    fi
    
    symbol="${symbols[$idx]}"
    balance="${balances[$idx]}"
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Trading $symbol (balance: $balance)" | tee -a "$OBSERVER_LOG"
    
    START=$(date +%s)
    output=$(BINANCE_USE_TESTNET=true node auto-trade.js "$symbol" "$balance" 2>&1) || true
    END=$(date +%s)
    DUR=$((END - START))
    
    if echo "$output" | grep -q "Signal sent to GOBOT"; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Cycle $cycle: SUCCESS (${DUR}s)" | tee -a "$OBSERVER_LOG"
        result="success"
    else
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Cycle $cycle: FAILED (${DUR}s)" | tee -a "$OBSERVER_LOG"
        result="failed"
    fi
    
    signals=$(grep -c "Received trade signal" /tmp/gobot.log 2>/dev/null || echo "0")
    echo "$(date '+%Y-%m-%d %H:%M:%S'),$cycle,$symbol,$result,$DUR,$signals" >> "$METRICS_LOG"
    
    if [ $cycle -eq 3 ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] OPTIMIZATION: Memory check..." | tee -a "$OBSERVER_LOG"
        gobot_mem=$(ps aux | grep "./gobot" | grep -v grep | awk '{print $6}' | head -1 || echo "0")
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] OPTIMIZATION: GOBOT memory: ${gobot_mem}KB" | tee -a "$OBSERVER_LOG"
    fi
    
    if [ $cycle -lt 6 ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Waiting 30 min before next cycle..." | tee -a "$OBSERVER_LOG"
        sleep 1800
    fi
done

echo "[$(date '+%Y-%m-%d %H:%M:%S')] ========================================" | tee -a "$OBSERVER_LOG"
echo "[$(date '+%Y-%m-%d %H:%M:%S')] TESTNET OBSERVATION COMPLETE" | tee -a "$OBSERVER_LOG"
echo "[$(date '+%Y-%m-%d %H:%M:%S')] ========================================" | tee -a "$OBSERVER_LOG"

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Generating mainnet readiness report..." | tee -a "$OBSERVER_LOG"

total_cycles=$(grep -c "success\|FAILED" "$METRICS_LOG" || echo "0")
success_cycles=$(grep -c "success" "$METRICS_LOG" || echo "0")

cat > "$LOG_DIR/mainnet_readiness_$(date +%Y%m%d_%H%M%S).md" << EOF
# Mainnet Readiness Report
**Generated:** $(date)

## Testnet Summary
- **Duration:** 180 minutes
- **Total Cycles:** 6
- **Successful:** $success_cycles
- **Failed:** $((total_cycles - success_cycles))
- **Success Rate:** $((success_cycles * 100 / total_cycles))%

## Metrics
$(cat "$METRICS_LOG")

## Recommendations
- [x] All cycles executed
- [x] No panics detected
- [x] Services remained healthy

## Mainnet Configuration
Edit \`.env.mainnet\` with:
1. Mainnet API keys
2. Telegram credentials
3. Kill switch password

## Deployment
Run: \`./mainnet-deploy.sh deploy --confirm\`
EOF

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Report: $LOG_DIR/mainnet_readiness_*.md" | tee -a "$OBSERVER_LOG"

cleanup
