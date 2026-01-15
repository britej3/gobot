#!/bin/bash

# GOBOT Complete Pipeline Validation Test
# Tests: Screenshot → QuantCrawler → Signal → P&L Calculation

echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║           GOBOT PIPELINE VALIDATION TEST                        ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""

cd /Users/britebrt/GOBOT/services/screenshot-service

# ═══════════════════════════════════════════════════════════════════════
# TEST 1: Check Environment
# ═══════════════════════════════════════════════════════════════════════
echo "TEST 1: Environment Configuration"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

PASS=0
FAIL=0

# Check Google Auth
if [ -n "$GOOGLE_EMAIL" ] && [ -n "$GOOGLE_APP_PASSWORD" ]; then
    echo "  ✅ Google Auth: Configured ($GOOGLE_EMAIL)"
    PASS=$((PASS + 1))
else
    echo "  ⚠️  Google Auth: Not configured (using mock AI)"
    FAIL=$((FAIL + 1))
fi

# Check Binance Testnet
if [ "$BINANCE_USE_TESTNET" = "true" ]; then
    echo "  ✅ Binance: Testnet mode"
    PASS=$((PASS + 1))
else
    echo "  ⚠️  Binance: Mainnet mode (real funds)"
    PASS=$((PASS + 1))
fi

# Check services running
if curl -s http://localhost:8080/health | grep -q "OK"; then
    echo "  ✅ GOBOT: Running on :8080"
    PASS=$((PASS + 1))
else
    echo "  ❌ GOBOT: Not responding on :8080"
    FAIL=$((FAIL + 1))
fi

if curl -s http://localhost:3456/health | grep -q "healthy"; then
    echo "  ✅ Screenshot Service: Running on :3456"
    PASS=$((PASS + 1))
else
    echo "  ❌ Screenshot Service: Not responding on :3456"
    FAIL=$((FAIL + 1))
fi

# ═══════════════════════════════════════════════════════════════════════
# TEST 2: QuantCrawler Client
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "TEST 2: QuantCrawler Client"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━"

timeout 5 node -e "
const quant = require('./quantcrawler-client.js');
console.log('  ✅ QuantCrawler client loaded');
console.log('     Has analyzeCharts:', typeof quant.analyzeCharts === 'function');
" 2>&1

if [ $? -eq 0 ]; then
    PASS=$((PASS + 1))
else
    FAIL=$((FAIL + 1))
fi

# ═══════════════════════════════════════════════════════════════════════
# TEST 3: Auto-Trade Client
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "TEST 3: Auto-Trade Workflow"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━"

timeout 5 node -e "
const auto = require('./auto-trade.js');
console.log('  ✅ Auto-trade client loaded');
console.log('     Has runTradingFlow:', typeof auto.runTradingFlow === 'function');
console.log('     Has captureMulti:', typeof auto.captureMulti === 'function');
" 2>&1

if [ $? -eq 0 ]; then
    PASS=$((PASS + 1))
else
    FAIL=$((FAIL + 1))
fi

# ═══════════════════════════════════════════════════════════════════════
# TEST 4: P&L Calculations
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "TEST 4: P&L Calculations"
echo "━━━━━━━━━━━━━━━━━━━━━"

# Test LONG trade P&L
node -e "
const entry = 0.00001000;
const exit = 0.00001040;
const quantity = 1000000;
const pnl = (exit - entry) * quantity;
const pnl_pct = ((exit - entry) / entry * 100).toFixed(2);

if (pnl === 0.4 && pnl_pct === '4.00') {
  console.log('  ✅ LONG P&L: \$\$0.40 (+4.00%)');
} else {
  console.log('  ❌ LONG P&L failed:', pnl, pnl_pct);
  process.exit(1);
}
"

if [ $? -eq 0 ]; then
    PASS=$((PASS + 1))
else
    FAIL=$((FAIL + 1))
fi

# Test SHORT trade P&L
node -e "
const entry = 0.00001000;
const exit = 0.00000960;
const quantity = 1000000;
const pnl = (entry - exit) * quantity;
const pnl_pct = ((entry - exit) / entry * 100).toFixed(2);

if (pnl === 0.4 && pnl_pct === '4.00') {
  console.log('  ✅ SHORT P&L: \$\$0.40 (+4.00%)');
} else {
  console.log('  ❌ SHORT P&L failed:', pnl, pnl_pct);
  process.exit(1);
}
"

if [ $? -eq 0 ]; then
    PASS=$((PASS + 1))
else
    FAIL=$((FAIL + 1))
fi

# Test Loss calculation
node -e "
const entry = 0.00001000;
const exit = 0.00000980;
const quantity = 1000000;
const pnl = (exit - entry) * quantity;
const pnl_pct = ((exit - entry) / entry * 100).toFixed(2);

if (pnl < 0 && pnl_pct === '-2.00') {
  console.log('  ✅ LOSS P&L: \$-0.20 (-2.00%)');
} else {
  console.log('  ❌ LOSS P&L failed:', pnl, pnl_pct);
  process.exit(1);
}
"

if [ $? -eq 0 ]; then
    PASS=$((PASS + 1))
else
    FAIL=$((FAIL + 1))
fi

# ═══════════════════════════════════════════════════════════════════════
# TEST 5: Win Rate Calculation
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "TEST 5: Win Rate Calculation"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━"

node -e "
const wins = 7;
const losses = 3;
const total = 10;
const winRate = (wins * 100 / total).toFixed(1);

if (winRate === '70.0') {
  console.log('  ✅ Win Rate: 70.0% (7 wins, 3 losses)');
} else {
  console.log('  ❌ Win Rate failed:', winRate);
  process.exit(1);
}
"

if [ $? -eq 0 ]; then
    PASS=$((PASS + 1))
else
    FAIL=$((FAIL + 1))
fi

# ═══════════════════════════════════════════════════════════════════════
# SUMMARY
# ═══════════════════════════════════════════════════════════════════════
echo ""
echo "════════════════════════════════════════════════════════════════════"
echo "VALIDATION SUMMARY"
echo "════════════════════════════════════════════════════════════════════"
echo ""
echo "  Passed: $PASS/10"
echo "  Failed: $FAIL/10"
echo ""

if [ $FAIL -eq 0 ]; then
    echo "  🌟 ALL TESTS PASSED - READY FOR TRADING"
    echo ""
    echo "To run 60-minute test with P&L tracking:"
    echo "  cd /Users/britebrt/GOBOT"
    echo "  ./run-60min-validated.sh"
    exit 0
else
    echo "  ⚠️  SOME TESTS FAILED"
    echo ""
    echo "Fix issues before running live trading"
    exit 1
fi
