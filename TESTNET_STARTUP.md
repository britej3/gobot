# 🧪 GOBOT Testnet Startup Guide

## ✅ TESTNET CONFIGURATION SUCCESSFUL!

Your bot is now configured and ready to trade on Binance Testnet with fake money.

---

## 📊 Testnet Status

```
✅ API Connection:   ONLINE
✅ Environment:      TESTNET (Safe)
✅ Testnet Balance:  5000 USDT (fake money)
✅ Futures API:      Connected
✅ Spot API:         Connected
✅ Permissions:      Valid
```

---

## 🚀 Quick Start Command

Run this single command to start GOBOT in testnet mode:

```bash
cd /Users/britebrt/GOBOT && \
BINANCE_USE_TESTNET=true \
BINANCE_TESTNET_API=oS63iBelbUHxTO5UYy39weUDLMPTO5Ia9OZEZ2N41oq79drDcKfvdEhPuStG5WFN \
BINANCE_TESTNET_SECRET=oROz7w1P01Xj7wwKp5jvCQEZxIvWYbuyEtzVZCEFiXLsO5zh3uprND2fQ61uVElv \
go run cmd/cognee/main.go
```

**Or use the convenience script:**
```bash
/tmp/run_testnet.sh
```

---

## 📂 What Happens When You Run

1. **Pre-flight Audit** (10-15 seconds)
   - ✅ Checks API connectivity
   - ✅ Verifies permissions
   - ✅ Confirms testnet balance

2. **Platform Initialization** (20-30 seconds)
   - 🧠 Initializes AI brain
   - 📊 Sets up monitoring
   - 🔧 Configures risk management
   - 💾 Loads recovery systems

3. **Trading Loop** (Continuous)
   - 📈 Scans market for opportunities
   - 🤖 AI analyzes patterns
   - 🎯 Executes trades at high leverage
   - 📋 Logs all activities

---

## 🔍 Monitoring the Bot

### View Logs:
```bash
# In another terminal, tail the logs:
tail -f /Users/britebrt/.cache/amp/logs/cli.log
```

### Check Status:
Press `Ctrl+C` to stop the bot gracefully. It will:
- Close all positions
- Save state
- Exit safely

### Test AI Connection:
```bash
# Run a test trade without starting full platform
BINANCE_USE_TESTNET=true \
BINANCE_TESTNET_API=oS63iBelbUHxTO5UYy39weUDLMPTO5Ia9OZEZ2N41oq79drDcKfvdEhPuStG5WFN \
BINANCE_TESTNET_SECRET=oROz7w1P01Xj7wwKp5jvCQEZxIvWYbuyEtzVZCEFiXLsO5zh3uprND2fQ61uVElv \
go run cmd/cognee/main.go -test-trade -symbol BTCUSDT -side BUY
```

---

## ⚙️ Configuration Files

### Environment Variables:
**File:** `/Users/britebrt/GOBOT/.env`

Key settings:
```bash
BINANCE_USE_TESTNET=true              # ✅ Set to true (testnet mode)
BINANCE_TESTNET_API=...               # ✅ Your testnet API key
BINANCE_TESTNET_SECRET=...            # ✅ Your testnet secret

# Trading Configuration
MIN_ATR_PERCENT=0.5                   # Mid-cap volatility requirement
MIN_24H_VOLUME_USD=10000000          # $10M minimum volume
WATCHLIST_SYMBOLS="ADAUSDT,DOTUSDT..."  # Trading symbols

# AI Thresholds
MIN_FVG_CONFIDENCE=0.6               # AI confidence threshold
MAX_VOLATILITY=0.05                  # Max volatility allowed

# Safe-Stop Protection
SAFE_STOP_ENABLED=true               # Auto-stop if balance drops
SAFE_STOP_THRESHOLD_PERCENT=10       # Stop at 10% loss
SAFE_STOP_MIN_BALANCE_USD=1000       # Minimum $1000 balance
```

### To Edit Configuration:
```bash
nano /Users/britebrt/GOBOT/.env
```

---

## 🛡️ Safety Features (Testnet)

Even in testnet mode, the bot includes:

1. **Safe-Stop Protection**
   - Automatically stops if balance drops 10%
   - Minimum balance protection at $1000
   - Prevents runaway losses

2. **Testnet Safeguards**
   - All trades use fake USDT
   - No real money at risk
   - Resettable testnet account

3. **Risk Management**
   - High leverage (20-50x) with small position sizes
   - Automatic stop-loss on every trade
   - Trailing take-profit to lock gains

---

## 🎯 What to Expect

**First 5 Minutes:**
- System initializes and connects
- AI brain loads the model
- Market scanning begins

**Next 10-30 Minutes:**
- AI analyzes market patterns
- Scanning for high-probability setups
- May find multiple opportunities

**When Trades Execute:**
```
🎯 Trade Signal Detected:
  Symbol: BTCUSDT
  Side: LONG
  Entry: $43,500
  Leverage: 20x
  Confidence: 85%

🔄 Position Opened:
  Order filled at $43,501
  Quantity: 0.0023 BTC
  Stop Loss: $43,413 (-0.2%)
  Take Profit: $43,762 (+0.6%)

✅ Trade Closed:
  Exit: $43,780
  PnL: +$15.23 (+0.65%)
  Duration: 3m 42s
```

---

## 📈 Testnet Details

**Your Testnet Account:**
- Balance: 5000 USDT (fake money)
- Can be reset anytime at: https://testnet.binancefuture.com
- Mirrors real market prices
- All trades are simulated

**To Reset Testnet Balance:**
1. Visit: https://testnet.binancefuture.com
2. Login with your testnet API key
3. Click "Reset Balance" anytime
4. Returns to 5000 USDT

---

## 🔧 Troubleshooting

### If Bot Doesn't Start:

**Check 1: Verify keys are set**
```bash
echo $BINANCE_TESTNET_API
# Should show: oS63iBelbUHxTO5UYy39w...
```

**Check 2: Run audit only**
```bash
BINANCE_USE_TESTNET=true \
BINANCE_TESTNET_API=... \
BINANCE_TESTNET_SECRET=... \
go run cmd/cognee/main.go -audit
```

**Check 3: Check compilation**
```bash
cd /Users/britebrt/GOBOT
go build -o /tmp/cognee cmd/cognee/main.go
```

### Common Issues:

1. **"API keys not configured"**
   → Keys not exported correctly. Use the full command with exports.

2. **"Connection refused"**
   → Check internet connection

3. **"Permission denied"**
   → Verify keys have Futures trading enabled in Binance Testnet

---

## 🎓 Next Steps

### 1. Run in Testnet (Now)
```bash
# Start trading with fake money
/tmp/run_testnet.sh
```

### 2. Monitor Performance (First Hour)
- Watch the logs for trade executions
- Check PnL (should be in testnet dashboard)
- Note any errors or warnings

### 3. When Ready for Mainnet:
1. Get real Binance API keys
2. Change `BINANCE_USE_TESTNET=false` in .env
3. Set `BINANCE_API_KEY` and `BINANCE_API_SECRET`
4. Start with small amounts (1-2 USDT per trade)
5. Gradually increase as you gain confidence

---

## 📞 Support

If you encounter issues:

1. **Check logs:** `tail -f ~/.cache/amp/logs/cli.log`
2. **Run verification:** `./verify_repositories.sh`
3. **Review docs:** `cat REPOSITORY_USAGE_GUIDE.md`
4. **Check compilation:** `go build -buildvcs=false ./...`

---

## ✅ Summary

**TESTNET STATUS:** ✅ FULLY CONFIGURED

- API keys: ✅ Set and verified
- Testnet balance: ✅ 5000 USDT available
- Compilation: ✅ All packages build
- Safety: ✅ No real money at risk
- Ready to trade: ✅ YES!

**Run this to start:**
```bash
cd /Users/britebrt/GOBOT && BINANCE_USE_TESTNET=true BINANCE_TESTNET_API=oS63iBelbUHxTO5UYy39weUDLMPTO5Ia9OZEZ2N41oq79drDcKfvdEhPuStG5WFN BINANCE_TESTNET_SECRET=oROz7w1P01Xj7wwKp5jvCQEZxIvWYbuyEtzVZCEFiXLsO5zh3uprND2fQ61uVElv go run cmd/cognee/main.go
```

Happy testing! 🚀
