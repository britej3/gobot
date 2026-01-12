#!/bin/bash
# Cognee Pre-Flight Validation Script
# Comprehensive system check before mainnet trading

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COGNEE_DIR="${COGNEE_DIR:-$(pwd)}"
LOG_DIR="$COGNEE_DIR/logs"
BINARY="$COGNEE_DIR/cognee"
ENV_FILE="$COGNEE_DIR/.env"

echo "🛡️  Cognee Pre-Flight Validation"
echo "===================================="
echo "Directory: $COGNEE_DIR"
echo "Binary: $BINARY"
echo ""

# Test counters
PASSED=0
FAILED=0
WARNINGS=0

# Helper functions
log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}✅${NC} $1"
    ((PASSED++))
}

log_fail() {
    echo -e "${RED}❌${NC} $1"
    ((FAILED++))
}

log_warn() {
    echo -e "${YELLOW}⚠️${NC} $1"
    ((WARNINGS++))
}

# Test 1: Binary Architecture
log_test "Verifying binary architecture..."
if [ -f "$BINARY" ]; then
    ARCH=$(file "$BINARY" | grep -o "x86_64\|x86-64")
    if [ -n "$ARCH" ]; then
        log_pass "Binary compiled for Intel architecture: $ARCH"
    else
        log_fail "Binary not compiled for Intel architecture"
    fi
else
    log_fail "Binary not found at $BINARY"
fi
echo ""

# Test 2: File Permissions
log_test "Checking file permissions..."
for file in "$ENV_FILE" "$COGNEE_DIR/state.json" "$COGNEE_DIR/trade.wal"; do
    if [ -f "$file" ]; then
        PERMS=$(ls -l "$file" | awk '{print $1}')
        if [[ "$PERMS" == "-rw-------" ]]; then
            log_pass "$(basename $file) permissions: 600"
        else
            log_fail "$(basename $file) permissions: $PERMS (expected 600)"
        fi
    else
        log_warn "$(basename $file) does not exist (will be created)"
    fi
done
echo ""

# Test 3: Chrony Time Sync (macOS/Linux)
log_test "Checking Chrony time synchronization..."
if command -v chronyc &> /dev/null; then
    OFFSET=$(chronyc tracking 2>/dev/null | awk '/Last offset/ {print $4}')
    if [ -n "$OFFSET" ]; then
        OFFSET_VAL=$(echo "$OFFSET" | bc -l)
        if (( $(echo "$OFFSET_VAL < 0.0005" | bc -l) )); then
            log_pass "Chrony offset: ${OFFSET}s (< 500μs)"
        else
            log_fail "Chrony offset: ${OFFSET}s (expected < 500μs)"
        fi
    else
        log_warn "Could not get Chrony offset (Chrony may not be running)"
    fi
else
    log_warn "chronyc not found (install with: brew install chrony)"
fi
echo ""

# Test 4: Run Automated Audit
log_test "Running Cognee automated audit..."
if [ -x "$BINARY" ]; then
    cd "$COGNEE_DIR"
    AUDIT_OUTPUT=$(./cognee --audit 2>&1)
    
    if echo "$AUDIT_OUTPUT" | grep -q "✅"; then
        log_pass "Audit passed"
        echo "$AUDIT_OUTPUT" | grep "✅"
    else
        log_fail "Audit failed"
        echo "$AUDIT_OUTPUT"
    fi
else
    log_fail "Binary not executable"
fi
echo ""

# Test 5: Jitter Test
log_test "Testing anti-sniffer jitter implementation..."
if [ -d "$COGNEE_DIR/cmd/test_jitter" ]; then
    cd "$COGNEE_DIR"
    JITTER_OUTPUT=$(go run cmd/test_jitter/main.go 2>&1)
    
    if echo "$JITTER_OUTPUT" | grep -q "Test complete"; then
        # Extract delays and check they're in 5-25ms range
        DELAYS=$(echo "$JITTER_OUTPUT" | grep -oP '\d+ ms' | grep -oP '\d+')
        VALID_RANGES=0
        for delay in $DELAYS; do
            if [ "$delay" -ge 5 ] && [ "$delay" -le 25 ]; then
                ((VALID_RANGES++))
            fi
        done
        
        if [ "$VALID_RANGES" -ge 8 ]; then
            log_pass "Jitter test: Delays in expected 5-25ms range"
        else
            log_fail "Jitter test: Delays outside expected range"
        fi
    else
        log_fail "Jitter test failed to run"
    fi
else
    log_warn "Jitter test not found"
fi
echo ""

# Test 6: MarketCap API (if implemented)
log_test "Testing MarketCap integration..."
if grep -q "GetMarketCap" "$COGNEE_DIR/internal/platform/market_data.go" 2>/dev/null; then
    log_pass "MarketCap integration detected"
    # Note: Full test requires API key and network
    log_warn "MarketCap test requires network access (skip if offline)"
else
    log_warn "MarketCap integration not found in codebase"
fi
echo ""

# Test 7: Telegram Integration (if token present)
log_test "Checking Telegram configuration..."
if [ -f "$ENV_FILE" ] && grep -q "TELEGRAM_TOKEN" "$ENV_FILE"; then
    TOKEN=$(grep "TELEGRAM_TOKEN" "$ENV_FILE" | cut -d'=' -f2)
    if [ -n "$TOKEN" ]; then
        log_pass "Telegram token configured"
        
        # Check if bot is mentioned in code
        if grep -q "SecureBot" "$COGNEE_DIR/internal/platform/telegram.go" 2>/dev/null; then
            log_pass "Telegram security implementation detected"
        else
            log_warn "Telegram implementation not fully integrated"
        fi
    else
        log_warn "Telegram token empty in .env"
    fi
else
    log_warn "Telegram token not in .env (optional feature)"
fi
echo ""

# Test 8: Service Status (launchd/systemd)
log_test "Checking service status..."
if systemctl list-units --type=service 2>/dev/null | grep -q cognee; then
    # Linux with systemd
    if systemctl is-active cognee &>/dev/null; then
        log_pass "Systemd service cognee is active"
    else
        log_fail "Systemd service cognee is not active"
    fi
elif launchctl list 2>/dev/null | grep -q cognee; then
    # macOS with launchd
    if launchctl list | grep cognee | awk '{print $2}' | grep -q "0"; then
        log_pass "Launchd service cognee is active (PID shown)"
    else
        log_fail "Launchd service cognee is not active"
    fi
else
    log_warn "Service not installed (run setup scripts to install)"
fi
echo ""

# Test 9: Log Files
log_test "Checking log files..."
if [ -d "$LOG_DIR" ]; then
    if [ -f "$LOG_DIR/cognee.log" ]; then
        LOG_SIZE=$(wc -c < "$LOG_DIR/cognee.log")
        if [ "$LOG_SIZE" -gt 0 ]; then
            log_pass "Main log exists and has content"
            
            # Check for recent WebSocket activity
            if tail -n 20 "$LOG_DIR/cognee.log" | grep -q "🔌"; then
                log_pass "Recent WebSocket activity detected"
            fi
        else
            log_warn "Main log is empty (service may not have started)")
        fi
    else
        log_warn "cognee.log not found"
    fi
    
    if [ -f "$LOG_DIR/error.log" ]; then
        ERR_COUNT=$(grep -c "ERROR" "$LOG_DIR/error.log")
        if [ "$ERR_COUNT" -lt 5 ]; then
            log_pass "Error log has minimal errors ($ERR_COUNT found)"
        else
            log_warn "Error log has $ERR_COUNT errors (review: tail logs/error.log)"
        fi
    else
        log_warn "error.log not found"
    fi
else
    log_warn "Logs directory does not exist"
fi
echo ""

# Test 10: Ghost Position Recovery (requires manual position)
log_test "Ghost position recovery (manual verification recommended)"
echo "   This test requires manually opening a position on Binance"
echo "   and then restarting Cognee. Recommendation:"
echo "   1. Open small position on Binance website"
echo "   2. Restart Cognee service"
echo "   3. Check logs for: Ghost position detected"
echo "   4. Verify position was adopted"
echo ""

# Summary
echo "==================================="
echo "📊 VALIDATION SUMMARY"
echo "==================================="
echo -e "${GREEN}Passed:${NC} $PASSED tests"
echo -e "${YELLOW}Warnings:${NC} $WARNINGS"
echo -e "${RED}Failed:${NC} $FAILED tests"
echo ""

if [ "$FAILED" -eq 0 ]; then
    if [ "$PASSED" -ge 7 ]; then  
        echo -e "${GREEN}🎉 PRE-FLIGHT VALIDATION PASSED${NC}"
        echo "Cognee is ready for mainnet deployment!"
        echo ""
        echo "⚠️  Remaining items (manual verification):"
        echo "   - Chrony offset < 500μs (check above)"
        echo "   - Ghost position recovery (manual test)"
        echo "   - Telegram bot (optional but recommended)"
        echo "   - Mainnet config in .env"
    else
        echo -e "${YELLOW}⚠️  PARTIAL VALIDATION PASSED${NC}"
        echo "Core components OK, but some tests skipped"
    fi
else
    echo -e "${RED}❌ PRE-FLIGHT VALIDATION FAILED${NC}"
    echo "Please address failed tests before mainnet deployment"
fi

echo ""
echo "📋 Next Steps:"
echo "1. Review any warnings above"
echo "2. Test ghost position recovery manually"
echo "3. Configure Binance mainnet API keys"
echo "4. Start with small position sizes"
echo "5. Monitor for 24-48 hours"

exit $FAILED
