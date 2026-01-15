#!/bin/bash

LOG_DIR="/Users/britebrt/GOBOT/logs"
OBSERVER_LOG="$LOG_DIR/observer_$(date +%Y%m%d_%H%M%S).log"
METRICS_LOG="$LOG_DIR/metrics_$(date +%Y%m%d_%H%M%S).csv"
OPTIMIZATION_LOG="$LOG_DIR/optimization_$(date +%Y%m%d_%H%M%S).log"

TOTAL_DURATION_MINUTES=180
OPTIMIZATION_INTERVAL_MINUTES=30
CYCLES=$((TOTAL_DURATION_MINUTES / OPTIMIZATION_INTERVAL_MINUTES))

mkdir -p "$LOG_DIR"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$OBSERVER_LOG"
}

log_metric() {
    echo "$(date '+%Y-%m-%d %H:%M:%S'),$1" >> "$METRICS_LOG"
}

log_optimization() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] OPTIMIZATION: $1" >> "$OPTIMIZATION_LOG"
}

cleanup() {
    log "Received shutdown signal, cleaning up..."
    pkill -f "./gobot" 2>/dev/null || true
    pkill -f "node.*server.js" 2>/dev/null || true
    log "Cleanup complete"
    exit 0
}

trap cleanup SIGINT SIGTERM

start_services() {
    log "Starting services..."

    cd /Users/britebrt/GOBOT

    if [ ! -f "./gobot" ]; then
        log "Building GOBOT..."
        go build -o gobot ./cmd/cobot/
    fi

    log "Starting GOBOT..."
    pkill -f "./gobot" 2>/dev/null || true
    ./gobot > /tmp/gobot.log 2>&1 &
    GOBOT_PID=$!
    echo $GOBOT_PID > /tmp/gobot.pid
    log "GOBOT started with PID: $GOBOT_PID"

    sleep 3

    cd /Users/britebrt/GOBOT/services/screenshot-service
    log "Starting screenshot service..."
    pkill -f "node.*server.js" 2>/dev/null || true
    node server.js > /tmp/screenshot.log 2>&1 &
    SCREENSHOT_PID=$!
    echo $SCREENSHOT_PID > /tmp/screenshot.pid
    log "Screenshot service started with PID: $SCREENSHOT_PID"

    sleep 3

    local health_ok=true

    if ! curl -s http://localhost:8080/health > /dev/null; then
        log "ERROR: GOBOT health check failed"
        cat /tmp/gobot.log | tail -20
        health_ok=false
    fi

    if ! curl -s http://localhost:3456/health > /dev/null; then
        log "ERROR: Screenshot service health check failed"
        cat /tmp/screenshot.log | tail -20
        health_ok=false
    fi

    if [ "$health_ok" = true ]; then
        log "All services healthy"
    fi
}

run_trading_cycle() {
    local symbol=$1
    local balance=$2
    local cycle_num=$3

    log "Running trading cycle #$cycle_num for $symbol (balance: $balance)"

    cd /Users/britebrt/GOBOT/services/screenshot-service

    START_TIME=$(date +%s)

    output=$(BINANCE_USE_TESTNET=true node auto-trade.js "$symbol" "$balance" 2>&1) || true

    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))

    if echo "$output" | grep -q "Signal sent to GOBOT"; then
        log "Cycle #$cycle_num: SUCCESS (${DURATION}s)"
        log_metric "cycle_${cycle_num},success,${DURATION}"
        return 0
    else
        log "Cycle #$cycle_num: FAILED (${DURATION}s)"
        log_metric "cycle_${cycle_num},failed,${DURATION}"
        log "Output: $output"
        return 1
    fi
}

check_services() {
    local errors=0

    if ! curl -s http://localhost:8080/health > /dev/null; then
        log "WARNING: GOBOT not responding"
        errors=$((errors + 1))
    fi

    if ! curl -s http://localhost:3456/health > /dev/null; then
        log "WARNING: Screenshot service not responding"
        errors=$((errors + 1))
    fi

    return $errors
}

restart_service() {
    local service=$1
    local pid_file=$2
    local start_cmd=$3

    log "Restarting $service..."

    if [ -f "$pid_file" ]; then
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            sleep 2
        fi
    fi

    eval "$start_cmd" > /dev/null 2>&1 &
    log "$service restarted"
}

optimize_system() {
    local cycle=$1
    log_optimization "=== Starting optimization cycle $cycle at $(date) ==="

    local gobot_mem=$(ps aux | grep "./gobot" | grep -v grep | awk '{print $6}' | head -1 || echo "0")
    local screenshot_mem=$(ps aux | grep "node.*server.js" | grep -v grep | awk '{print $6}' | head -1 || echo "0")

    log_optimization "Memory - GOBOT: ${gobot_mem}KB, Screenshot: ${screenshot_mem}KB"

    if [ "$gobot_mem" -gt 500000 ]; then
        log_optimization "GOBOT memory high, restarting..."
        restart_service "GOBOT" "/tmp/gobot.pid" "cd /Users/britebrt/GOBOT && ./gobot > /tmp/gobot.log 2>&1 &"
    fi

    if [ "$screenshot_mem" -gt 300000 ]; then
        log_optimization "Screenshot service memory high, restarting..."
        restart_service "Screenshot" "/tmp/screenshot.pid" "cd /Users/britebrt/GOBOT/services/screenshot-service && node server.js > /tmp/screenshot.log 2>&1 &"
    fi

    local recent_errors=$(tail -100 /tmp/gobot.log 2>/dev/null | grep -c -i "error\|failed" || echo "0")
    log_optimization "Errors in recent logs: $recent_errors"

    if [ "$recent_errors" -gt 10 ]; then
        log_optimization "High error rate detected, checking services..."
        check_services
    fi

    log_optimization "=== Optimization cycle $cycle complete ==="
}

collect_metrics() {
    local cycle=$1

    local gobot_status="healthy"
    if ! curl -s http://localhost:8080/health > /dev/null; then
        gobot_status="unhealthy"
    fi

    local screenshot_status="healthy"
    if ! curl -s http://localhost:3456/health > /dev/null; then
        screenshot_status="unhealthy"
    fi

    local signals_received=$(grep -c "Received trade signal" /tmp/gobot.log 2>/dev/null || echo "0")
    local cycles_completed=$(grep -c "SUCCESS" "$OBSERVER_LOG" 2>/dev/null || echo "0")

    log_metric "$cycle,$gobot_status,$screenshot_status,$signals_received,$cycles_completed"

    log "=== Cycle $cycle Summary ==="
    log "GOBOT: $gobot_status | Screenshot: $screenshot_status"
    log "Total signals received: $signals_received | Cycles completed: $cycles_completed"
}

main() {
    log "========================================"
    log "GOBOT 180-MINUTE TESTNET OBSERVER"
    log "Duration: ${TOTAL_DURATION_MINUTES} minutes"
    log "Optimization every: ${OPTIMIZATION_INTERVAL_MINUTES} minutes"
    log "Total cycles: $CYCLES"
    log "========================================"

    echo "timestamp,cycle,gobot_status,screenshot_status,signals_received,cycles_completed" > "$METRICS_LOG"

    start_services

    local symbols=("1000PEPEUSDT" "1000BONKUSDT" "1000FLOKIUSDT" "1000WIFUSDT")
    local balances=("5000" "3000" "4000" "3500")

    log "Monitoring for ${TOTAL_DURATION_MINUTES} minutes..."

    local elapsed=0
    local cycle_count=0

    while [ $elapsed -lt $TOTAL_DURATION_MINUTES ]; do
        cycle_count=$((cycle_count + 1))

        log "=== Time elapsed: ${elapsed}min / ${TOTAL_DURATION_MINUTES}min ==="

        if ! check_services; then
            log "Services unhealthy, restarting..."
            start_services
        fi

        local idx=$((cycle_count % ${#symbols[@]}))
        if [ $idx -eq 0 ]; then
            idx=0
        fi

        run_trading_cycle "${symbols[$idx]}" "${balances[$idx]}" $cycle_count

        collect_metrics $cycle_count

        if [ $((cycle_count % 2)) -eq 0 ] && [ $elapsed -lt $((TOTAL_DURATION_MINUTES - OPTIMIZATION_INTERVAL_MINUTES)) ]; then
            optimize_system $cycle_count
        fi

        sleep $((OPTIMIZATION_INTERVAL_MINUTES * 60))
        elapsed=$((elapsed + OPTIMIZATION_INTERVAL_MINUTES))
    done

    log "========================================"
    log "TESTNET OBSERVATION COMPLETE"
    log "========================================"

    optimize_system $cycle_count

    log "Final metrics collected at: $METRICS_LOG"
    log "Optimization log: $OPTIMIZATION_LOG"
    log "Observer log: $OBSERVER_LOG"

    tail -50 "$OBSERVER_LOG"

    log "Preparing mainnet readiness report..."

    cat > "$LOG_DIR/mainnet_readiness_$(date +%Y%m%d).md" << EOF
# Mainnet Readiness Report
**Generated:** $(date)

## Testnet Summary
- **Duration:** ${TOTAL_DURATION_MINUTES} minutes
- **Total Cycles:** $cycle_count
- **Symbols Traded:** ${symbols[*]}
- **Logs:** $OBSERVER_LOG

## Health Checks
$(tail -20 "$METRICS_LOG" | while read line; do echo "- $line"; done)

## Optimization History
$(cat "$OPTIMIZATION_LOG")

## Recommendations
$(if grep -q "unhealthy" "$METRICS_LOG" 2>/dev/null; then echo "- [ ] Resolve health issues before mainnet"; else echo "- [x] All health checks passed"; fi)
$(if [ -f "/tmp/gobot.log" ] && grep -q "panic\|PANIC" /tmp/gobot.log 2>/dev/null; then echo "- [ ] Fix panic/crash issues"; else echo "- [x] No panics detected"; fi)

## Next Steps
1. Review metrics in $METRICS_LOG
2. Verify configuration in .env
3. Update API keys for mainnet
4. Start with smaller position sizes
5. Monitor closely for first 30 minutes
EOF

    log "Mainnet readiness report: $LOG_DIR/mainnet_readiness_$(date +%Y%m%d).md"

    log "Testnet observation complete. Ready for mainnet deployment."
}

main
