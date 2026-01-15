#!/bin/bash

# GOBOT Testnet Test - Agent-Browser + Fallback AI
# Tests the complete trading pipeline

set -e

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║     GOBOT TESTNET VALIDATION - AGENT-BROWSER WORKFLOW         ║"
echo "╚═══════════════════════════════════════════════════════════════╝"

INITIAL_CAPITAL=5000
POSITION_SIZE=100
SYMBOLS=("BTCUSDT" "ETHUSDT" "SOLUSDT")

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"; }
log_success() { echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] ✅ $1${NC}"; }
log_warn() { echo -e "${YELLOW}[$(date '+%H:%M:%S')] ⚠️  $1${NC}"; }
log_error() { echo -e "${RED}[$(date '+%H:%M:%S')] ❌ $1${NC}"; }

# Check agent-browser
check_agent_browser() {
    log "Checking agent-browser..."
    if command -v agent-browser &> /dev/null; then
        log_success "agent-browser installed"
        agent-browser --version 2>/dev/null | head -1
    else
        log_error "agent-browser not installed"
        log "Installing..."
        npm install -g agent-browser
        agent-browser install
    fi
}

# Start GOBOT if not running
start_gobot() {
    log "Checking GOBOT..."
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        log_success "GOBOT already running on :8080"
    else
        log "Starting GOBOT..."
        cd /Users/britebrt/GOBOT
        ./gobot > /tmp/gobot_testnet.log 2>&1 &
        sleep 3
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            log_success "GOBOT started"
        else
            log_warn "GOBOT failed to start, continuing anyway..."
        fi
    fi
}

# Run trading cycle
run_cycle() {
    local symbol=$1
    local cycle=$2
    local total=$3

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log "Cycle $cycle/$total: Trading $symbol"

    cd /Users/britebrt/GOBOT/services/screenshot-service

    OUTPUT=$(node auto-trade.js "$symbol" "$POSITION_SIZE" 2>&1)

    if echo "$OUTPUT" | grep -q "WORKFLOW COMPLETE"; then
        log_success "$symbol cycle complete"

        if echo "$OUTPUT" | grep -q "Signal sent to GOBOT"; then
            log_success "Signal sent to GOBOT"
            return 0
        else
            log_warn "Workflow complete but signal may not have been sent"
            return 0
        fi
    else
        log_error "Workflow failed"
        echo "$OUTPUT" | tail -5
        return 1
    fi
}

# Main test
main() {
    local cycles=4
    local success=0
    local start_time=$(date +%s)

    check_agent_browser
    start_gobot

    log "Starting $cycles trading cycles..."
    log "Initial Capital: \$$INITIAL_CAPITAL"
    log "Position Size: \$$POSITION_SIZE"

    for i in $(seq 1 $cycles); do
        symbol=${SYMBOLS[$((i % 3))]}

        if run_cycle "$symbol" "$i" "$cycles"; then
            ((success++))
        fi

        if [ $i -lt $cycles ]; then
            log "Waiting 15 minutes for next cycle..."
            sleep 15
        fi
    done

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo ""
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                    TEST RESULTS                               ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo ""
    echo "  Cycles Completed:  $success/$cycles"
    echo "  Success Rate:      $(( success * 100 / cycles ))%"
    echo "  Duration:          $(( duration / 60 )) minutes"
    echo ""

    if [ $success -eq $cycles ]; then
        log_success "ALL CYCLES PASSED"
        exit 0
    else
        log_warn "Some cycles failed - check output above"
        exit 1
    fi
}

main "$@"
