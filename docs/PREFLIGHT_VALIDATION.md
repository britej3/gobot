# Cognee Pre-Flight Validation Guide
## Complete System Verification Before Mainnet Trading

**Validation Script:** `scripts/validation.sh`  
**Platform:** Linux / macOS Intel  
**Purpose:** Ensure all components operational before live trading  
**Criticality:** **MANDATORY** - Do not trade on mainnet without passing all tests

---

## 🚨 VALIDATION OVERVIEW

The pre-flight validation performs 10 systematic checks:

1. **Binary Architecture** - Correct Intel/darwin compilation
2. **File Permissions** - .env and state files secured (0600)
3. **Chrony Time Sync** - Sub-millisecond precision verified
4. **Automated Audit** - Built-in system self-check
5. **Jitter Test** - Anti-sniffer delays validated
6. **MarketCap API** - CoinGecko integration confirmed
7. **Telegram Security** - Bot and whitelist configured
8. **Service Status** - launchd/systemd active
9. **Log Files** - Logging operational with minimal errors
10. **Ghost Recovery** - Manual crash simulation recommended

---

## 📝 TEST EXECUTION

### Run All Tests
```bash
cd ~/cognee
./scripts/validation.sh
```

### Run Individual Tests
```bash
# Binary check
file ~/cognee/cognee

# Permissions check
ls -l .env state.json trade.wal

# Time sync check
chronyc tracking

# Audit check
./cognee --audit

# Jitter check
go run cmd/test_jitter/main.go

# Service check (Linux)
systemctl status cognee

# Service check (macOS)
launchctl list | grep cognee

# Logs check
tail -n 50 ~/cognee/logs/cognee.log
```

---

## 📊 TEST-BY-TEST BREAKDOWN

### TEST 1: Binary Architecture ✅

**Purpose:** Ensure binary compiled for Intel Mac architecture

**Command:**
```bash
file ~/cognee/cognee
```

**Expected Output:**
```
cognee: Mach-O 64-bit executable x86_64
```

**Validation Criteria:**
- ✅ Output contains "x86_64" or "x86-64"
- ✅ File type is "Mach-O 64-bit executable"
- ✅ Binary is executable (`-rwxr-xr-x` permissions)

**Failure Indicators:**
- ❌ "cannot execute binary file" - Wrong architecture
- ❌ No "x86_64" - Compiled for ARM or wrong platform

**Fix:**
```bash
cd ~/cognee
GOOS=darwin GOARCH=amd64 go build -o cognee ./cmd/cognee
```

---

### TEST 2: File Permissions ✅

**Purpose:** Ensure sensitive files are not world-readable

**Command:**
```bash
ls -l .env state.json trade.wal
```

**Expected Output:**
```
-rw-------  1 user  staff   48 Jan 10 12:00 .env
-rw-------  1 user  staff    0 Jan 10 12:00 state.json
-rw-------  1 user  staff    0 Jan 10 12:00 trade.wal
```

**Validation Criteria:**
- ✅ Permissions show "-rw-------" (0600)
- ✅ Only owner has read/write access
- ✅ No group or world access

**Failure Indicators:**
- ❌ "-rw-r--r--" (0644) - World readable!
- ❌ "-rw-rw-rw-" (0666) - World writable!

**Fix:**
```bash
chmod 600 .env state.json trade.wal
```

---

### TEST 3: Chrony Time Sync ✅

**Purpose:** Verify sub-millisecond time precision for HFT

**Command:**
```bash
chronyc tracking
```

**Expected Output:**
```
Reference ID    : 17.253.24.125 (time.apple.com)
Stratum         : 2
Ref time (UTC)  : Thu Jan 10 12:00:00 2026
System time     : 0.000000123 seconds slow of NTP time
Last offset     : -0.000000456 seconds
RMS offset      : 0.000000789 seconds
Frequency       : 1.234 ppm fast
Residual freq   : +0.001 ppm
Skew            : 0.006 ppm
Root delay      : 0.003456789 seconds
Root dispersion : 0.000123456 seconds
Update interval : 64.2 seconds
Leap status     : Normal
```

**Validation Criteria:**
- ✅ **Last offset** < 0.000500s (500 microseconds)
- ✅ **RMS offset** < 0.001000s (1 millisecond)
- ✅ **Leap status** = "Normal" (not "Not synchronized")

**Failure Indicators:**
- ❌ Last offset > 0.001s - Time sync too loose
- ❌ Leap status = "Not synchronized" - Not synced to NTP
- ❌ "Command not found" - Chrony not installed

**Fix:**
```bash
# Install chrony (macOS)
brew install chrony
sudo chronyd

# Force sync
sudo chronyc -a makestep

# Check again
chronyc tracking
```

**HFT Requirement:** ⚠️ **CRITICAL** - Binance will reject orders with timestamps >1s off

---

### TEST 4: Automated Audit ✅

**Purpose:** Run comprehensive system self-check

**Command:**
```bash
cd ~/cognee && ./cognee --audit
```

**Expected Output:**
```
🛡️  Cognee Pre-Flight Audit
====================================

✅ Binance API connection successful
✅ .env permissions secure (0600)
✅ Chrony offset within limits (0.000123s)
✅ WebSocket Stream Manager operational
✅ WAL (Write-Ahead Log) ready
✅ Reconciler initialized
✅ Telegram bot configured (optional)
✅ Binary compiled for correct architecture

🎉 AUDIT PASSED - System ready for trading
```

**Validation Criteria:**
- ✅ All green checkmarks
- ✅ Binance API connection successful
- ✅ Chrony offset < 500μs
- ✅ Permissions correct (0600)
- ✅ WebSocket operational
- ✅ WAL ready

**Failure Indicators:**
- ❌ "❌ AUDIT FAILED" - One or more critical checks failed
- ❌ "API connection failed" - Invalid keys or network issue
- ❌ "Permissions insecure" - .env not 0600

**Fix:**
```bash
# Check connectivity
curl https://api.binance.com/api/v3/time

# Check permissions
ls -l ~/.env
chmod 600 ~/.env

# Run audit with verbose
./cognee --audit --verbose
```

---

### TEST 5: Jitter Implementation ✅

**Purpose:** Verify anti-sniffer timing obfuscation

**Command:**
```bash
go run cmd/test_jitter/main.go
```

**Expected Output:**
```
🧪 Testing Anti-Sniffer Jitter Implementation
===============================================

Test 1: Measuring 10 jitter delays...
  Delay  1: 12 ms
  Delay  2: 18 ms
  Delay  3: 8 ms
  Delay  4: 21 ms
  Delay  5: 15 ms
  Delay  6: 9 ms
  Delay  7: 19 ms
  Delay  8: 14 ms
  Delay  9: 11 ms
  Delay 10: 17 ms

✅ Jitter test complete!
Expected: Delays should show natural distribution around 15ms mean
```

**Validation Criteria:**
- ✅ All delays between 5-25ms
- ✅ Clustered around 15ms (not uniform)
- ✅ Implements NormalFloat64 (not uniform rand)

**Failure Indicators:**
- ❌ All delays identical - Normal distribution not working
- ❌ Delays outside 5-25ms range - Wrong parameters
- ❌ Delays always <1ms - Jitter not being applied

**Verify Implementation:**
```bash
# Check source code uses NormalFloat64
grep -A 10 "func ApplyJitter" internal/platform/jitter.go

# Should show: rand.NormFloat64() * stdDev
```

**HFT Impact:** Jitter prevents pattern recognition by exchange monitoring

---

### TEST 6: MarketCap Integration ⚠️

**Purpose:** Verify CoinGecko API and cache

**Command:**
```bash
# Check implementation exists
grep -n "GetMarketCap" internal/platform/market_data.go
```

**Expected:**
```
45: func (c *MarketCapCache) GetMarketCap(symbol string, price float64) (float64, string, error) {
```

**Validation Criteria:**
- ✅ Implementation detected in codebase
- ✅ Cache struct with 24h TTL
- ✅ Error fallback to "high_risk" status
- ⚠️ **Note:** Full test requires network and API access

**Manual API Test:**
```bash
# Test CoinGecko API directly
curl -s "https://api.coingecko.com/api/v3/coins/bitcoin?localization=false" | \
  jq '.market_data.circulating_supply'

# Should return: 19600000-ish
```

**Integration Point:**
```go
// In scanner/striker logic:
mc, status, _ := cache.GetMarketCap("bitcoin", 45000.0)
if status == "high_risk" {
    quantity = quantity * 0.5 // Reduce by 50%
}
```

---

### TEST 7: Telegram Security ✅

**Purpose:** Verify secure bot with whitelist

**Configuration Check:**
```bash
grep "TELEGRAM\|AUTHORIZED" ~/.env
```

**Expected Output:**
```
TELEGRAM_TOKEN=716123456:AAH...your_token_here
AUTHORIZED_CHAT_ID=123456789
```

**Manual Test:**
```bash
# 1. Send message to bot from YOUR phone
# Type: /status

# 2. Expected response (in Telegram):
# "🚀 Cognee Status: Running
#  Active Positions: 0
#  PnL Today: $0.00
#  Balance: $1,234.56"

# 3. Have friend send /status
# Expected: No response OR "Unauthorized" message

# 4. Check bot logs:
tail -f ~/cognee/logs/cognee.log | grep Telegram

# Should show:
# ✅ Message from authorized user: 123456789
# NOT show any messages from unauthorized users
```

**Validation Criteria:**
- ✅ Only AUTHORIZED_CHAT_ID can issue commands
- ✅ Unauthorized users logged but ignored
- ✅ `/panic`, `/status`, `/halt` commands work
- ✅ Bot logs all access attempts

**Security Check:**
```bash
# Verify middleware checks ChatID
grep -A 5 "Whitelist ChatID" internal/platform/telegram.go
```

---

### TEST 8: Service Status ✅

**Purpose:** Verify launchd/systemd managing cognee

**Linux (systemd):**
```bash
systemctl status cognee
```

**Expected:**
```
● cognee.service - Cognee HFT Mainnet Engine
   Loaded: loaded (/etc/systemd/system/cognee.service; enabled)
   Active: active (running) since Thu 2026-01-10 12:00:00 UTC
 Main PID: 12345 (cognee)
   Tasks: 8
   Memory: 312.4M
   CGroup: /system.slice/cognee.service
           └─12345 /usr/local/bin/cognee --mainnet

Dec 10 12:00:00 server cognee[12345]: 🔌 WebSocket connected
```

**macOS (launchd):**
```bash
launchctl list | grep cognee
```

**Expected:**
```
-   0   com.cognee.mainnet
```

**Check Process:**
```bash
ps aux | grep cognee | grep -v grep
```

**Expected:**
```
user    12345   0.1  1.2  12345678  98765   ??  S     2:30PM   0:01.23 ./cognee --mainnet
```

**Validation Criteria:**
- ✅ Service loaded and active
- ✅ Process running with correct arguments
- ✅ CPU usage reasonable (5-15%)
- ✅ Memory usage reasonable (2-4GB)

**Failure Indicators:**
- ❌ "inactive (dead)" - Service crashed
- ❌ No process found - Not running
- ❌ High CPU (>50%) - Potential infinite loop
- ❌ High memory (>8GB) - Memory leak

**Restart (if needed):**
```bash
# Linux
sudo systemctl restart cognee
sudo systemctl status cognee

# macOS
launchctl stop com.cognee.mainnet
launchctl start com.cognee.mainnet
launchctl list | grep cognee
```

---

### TEST 9: Log File Health ✅

**Purpose:** Verify logging operational and error-free

**Commands:**
```bash
# Check logs exist
ls -lh ~/cognee/logs/cognee.log ~/cognee/logs/error.log

# Check recent activity
tail -n 20 ~/cognee/logs/cognee.log

# Count errors
wc -l ~/cognee/logs/error.log
grep -c "ERROR" ~/cognee/logs/error.log

# Check specific components
grep "🎲" ~/cognee/logs/cognee.log | tail -5  # Jitter
grep "GHOST" ~/cognee/logs/cognee.log | tail -5  # Reconciliation
grep "🔌" ~/cognee/logs/cognee.log | tail -5    # WebSocket
```

**Expected Patterns:**
```
# Normal operations
[12:00:00] ✅ WAL initialized
[12:00:01] 🔌 WebSocket connected
[12:00:02] 🎲 Applying anti-sniffer jitter...
[12:00:02] Order executed
[12:00:05] Syncing state to disk
[12:01:00] Health check: OK
```

**Error Patterns (BAD):**
```
❌ [12:00:00] WebSocket connection failed: timeout
❌ [12:00:01] WAL write error: permission denied
❌ [12:00:02] Reconciliation failed: API error
❌ [12:00:03] Telegram unauthorized access attempt
```

**Validation Criteria:**
- ✅ Logs updating (recent timestamp)
- ✅ All components logging activity
- ✅ Error log minimal (<5 errors/day)
- ✅ No repeated errors

**Failure Indicators:**
- ❌ Logs empty - Service not writing
- ❌ Error log has 100+ entries - System in error state
- ❌ Repeated "connection failed" - Network issues
- ❌ "permission denied" - File system issues

**Fix Log Issues:**
```bash
# Create log dir if missing
mkdir -p ~/cognee/logs
touch ~/cognee/logs/{cognee,error}.log
chmod 644 ~/cognee/logs/*.log

# Restart to open new log handles
# (launchd/systemd will restart if crashed)
```

---

### TEST 10: Ghost Position Recovery ⚠️

**Purpose:** Most critical test - ensures capital safety after crash

**WARNING:** ⚠️ **THIS TEST REQUIRES REAL MONEY AT RISK**

**Manual Test Protocol:**

```bash
# === STEP 1: Open Position ===
# 1. PAUSE Cognee bot
launchctl stop com.cognee.mainnet

# 2. Manually open SMALL position on Binance website
#    - Symbol: BTCUSDT
#    - Size: $10 worth (minimal risk)
#    - Type: Market order

# === STEP 2: Simulate Crash ===
# 3. KILL Cognee process brutally
kill -9 $(pgrep cognee)

# 4. Verify it's dead
ps aux | grep cognee  # Should show nothing

# === STEP 3: Restart and Watch Recovery ===
# 5. Start Cognee
tail -f ~/cognee/logs/cognee.log | grep RECONCILER &
launchctl start com.cognee.mainnet

# 6. EXPECTED LOG OUTPUT (within 30 seconds):
#
# [12:00:05] 🔍 [RECONCILER] Starting state reconciliation...
# [12:00:06] 👻 GHOST POSITION DETECTED: BTCUSDT (0.0002)
# [12:00:06] ✅ WAL intent found for ghost position
# [12:00:06] 🛡️  Emergency guards attached to ghost position
# [12:00:06] ✅ Ghost position adopted and secured
# [12:00:06] 🔍 Reconciliation completed (ghosts_detected:1)

# 7. Verify position in Binance matches local state
#    - Position should still be open
#    - SL/TP should be attached (check Binance UI)
#    - Local state should show position

# === STEP 4: Cleanup ===
# 8. Close position manually
# 9. Let Cognee detect closure on next reconcile

# SUCCESS CRITERIA:
# ✅ Ghost position detected within 30s of startup
# ✅ Position adopted into local state
# ✅ Emergency SL/TP attached
# ✅ WAL entry created with GHOST_ADOPTED
# ✅ No manual intervention needed
```

**Automated Test (if you have a test account):**
```bash
# Run provided ghost simulation (if exists)
./cognee --test-ghost-recovery --symbol BTCUSDT --size 0.001
```

**Validation Criteria:**
- ✅ Ghost detected within 30s of startup
- ✅ Position adopted with SL/TP
- ✅ WAL logs the adoption
- ✅ Reconciler reports: ghosts_detected=1

**Failure Indicators:**
- ❌ Ghost not detected - Reconciler not running
- ❌ Position not adopted - State manager issue
- ❌ No SL/TP attached - Emergency guard failure
- ❌ Duplicate orders - Both Cognee and manual order active

**Debugging:**
```bash
# Check reconcile logs in detail
tail -n 100 ~/cognee/logs/cognee.log | grep RECONCILER

# Check WAL for ghost entry
grep "GHOST_ADOPTED" ~/cognee/trade.wal | tail -1 | jq .

# Check Binance positions
curl -H "X-MBX-APIKEY: $BINANCE_API_KEY" \
  https://fapi.binance.com/fapi/v2/positionRisk
```

---

## 📋 VALIDATION SUMMARY TABLE

| Test | Command | Success | Critical | Fix Command |
|------|---------|--------|----------|-------------|
| Binary | `file cognee` | Shows x86_64 | ⚠️ Yes | `GOOS=darwin GOARCH=amd64 go build` |
| Permissions | `ls -l .env` | -rw------- | ⚠️ Yes | `chmod 600 .env` |
| Chrony | `chronyc tracking` | Offset < 500μs | 🔴 CRITICAL | `brew install chrony && sudo chronyd` |
| Audit | `./cognee --audit` | All checks pass | 🔴 CRITICAL | Run with `--verbose` |
| Jitter | `go run test_jitter` | Delays 5-25ms | ⚠️ Yes | Verify source code |
| MarketCap | `grep GetMarketCap` | Implementation exists | ⚠️ Yes | Manual API test |
| Telegram | Check .env | Token configured | ⚠️ No | Optional feature |
| Service | `launchctl list` | PID shown | ⚠️ Yes | `launchctl load` |
| Logs | `tail -f log` | Recent activity | ⚠️ Yes | Create log dir |
| Ghost | Manual test | Position adopted | 🔴 CRITICAL | Debug reconciler |

**Legend:**
- 🔴 **CRITICAL** - Do not trade on mainnet without this
- ⚠️ **IMPORTANT** - Should be fixed, but can trade with caution
- ✅ **RECOMMENDED** - Good to have, not blocking

---

## 🎯 PRE-FLIGHT CHECKLIST

Before starting mainnet trading, confirm ALL are true:

### Infrastructure (100% Required)
- [ ] Binary shows `x86_64` architecture
- [ ] All sensitive files have `0600` permissions
- [ ] `chronyc tracking` shows offset **< 500μs**
- [ ] `chronyc tracking` shows **Leap status: Normal**
- [ ] `./cognee --audit` passes ALL checks

### Features (100% Required)
- [ ] Jitter test shows delays in 5-25ms range
- [ ] MarketCap integration code exists
- [ ] Telegram token configured (if using bot)

### Services (100% Required)
- [ ] launchd/systemd shows cognee active
- [ ] `ps aux | grep cognee` shows running process
- [ ] `tail -f logs/cognee.log` shows recent activity
- [ ] Error log has **< 5 errors total**

### Critical Safety (100% Required)
- [ ] **Ghost position test performed successfully**
- [ ] Safety-Stop configured in .env
- [ ] Telegram /panic command tested
- [ ] Binance API keys have **NO withdrawal permission**

### Deployment (100% Required)
- [ ] .env configured with **testnet** `true`
- [ ] Service runs for 24h without crashes
- [ ] No ghost positions in first 24h
- [ ] Logs show proper WebSocket reconnection

---

## 🚀 MAINNET GO/NO-GO DECISION

### GO (Ready for Mainnet) ✅

**All must be true:**
- Binary architecture correct
- Permissions 0600 on all files
- Chrony offset **< 500μs**
- Audit passes completely
- Service running stable
- Ghost test **successfully completed**
- **Testnet running 24h without issues**

**Confidence Level:** High

### NO-GO (Not Ready) ❌

**Any of these:**
- Chrony offset **> 500μs**
- Audit shows API failures
- Service crashes repeatedly
- **Ghost test NOT performed**
- More than 10 errors in log
- **Testnet shows issues**

**Action:** Fix failures, re-run validation, test more

### PROCEED WITH CAUTION ⚠️

**Minor issues only:**
- Telegram not configured (optional)
- MarketCap test skipped (can trade without)
- Some warnings in audit (non-critical)
- High error count BUT understood why

**Action:** Can start with very small positions

---

## 📊 VALIDATION SCORE CALCULATION

Assign points:
- **Critical tests (5):** Binary, Chrony, Audit, Service, Ghost = 20 points each
- **Important tests (3):** Permissions, Jitter, Logs = 10 points each
- **Optional tests (2):** Telegram, MarketCap = 5 points each

**Maximum Score:** 100 points

**Scoring:**
- **95-100 points:** 🟢 **GO FOR MAINNET**
- **80-94 points:** 🟡 **PROCEED WITH CAUTION**
- **< 80 points:** 🔴 **NO-GO**

**Example Score:**
```
Binary: ✅ (20)
Chrony: ✅ (20)
Audit: ✅ (20)
Service: ✅ (20)
Ghost: ⚠️ Not tested (0)
Permissions: ✅ (10)
Jitter: ✅ (10)
Logs: ✅ (10)
Telegram: ❌ (0)
MarketCap: ✅ (5)

Total: 95/100 = 🟢 GO
```

---

## 🎉 VALIDATION COMPLETE

When all tests pass:

```
🛡️  Cognee Pre-Flight Validation

📊 VALIDATION SUMMARY
===================================
✅ Passed: 9 tests
⚠️  Warnings: 1
❌ Failed: 0

🎉 PRE-FLIGHT VALIDATION PASSED
Cognee is ready for mainnet deployment!

Next Steps:
1. Review any warnings above
2. Configure Binance mainnet keys
3. Start with small position sizes
4. Monitor for 24-48 hours
5. Keep /panic command ready
```

**Status:** ✅ **APPROVED FOR MAINNET TRADING**

---

**Script Location:** `scripts/validation.sh`  
**Documentation:** `docs/PREFLIGHT_VALIDATION.md`  
**Version:** 1.0.0  
**Last Updated:** 2026-01-10
