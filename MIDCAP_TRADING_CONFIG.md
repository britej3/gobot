# Mid-Cap Trading Configuration - GOBOT

## ✅ Configuration Updated for Mid-Cap Focus

Your GOBOT is now properly configured to trade **mid-cap Binance Futures perpetual assets** exclusively.

### What Changed

#### 1. Environment File (`.env`)
**Before:**
```
WATCHLIST_SYMBOLS="BTCUSDT,ETHUSDT,ZECUSDT,SOLUSDT"  # Large caps
MIN_24H_VOLUME_USD=1000000                          # Too low for mid-caps
```

**After:**
```
WATCHLIST_SYMBOLS="ADAUSDT,DOTUSDT,AVAXUSDT,MATICUSDT,LINKUSDT,UNIUSDT,LTCUSDT,BCHUSDT,FILUSDT,ETCUSDT,XLMUSDT,VETUSDT,TRXUSDT,ALGOUSDT,AXSUSDT,ICPUSDT,NEARUSDT,ATOMUSDT,XMRUSDT,GRTUSDT,FTMUSDT,MANAUSDT,HBARUSDT,EGLDUSDT,FLOWUSDT,XTZUSDT,KSMUSDT,AAVEUSDT,MKRUSDT,RUNEUSDT,ENSUSDT,IMXUSDT,API3USDT,INJUSDT,BLURUSDT,OPUSDT,LDOUSDT,OMUSDT,ARKMUSDT,ALPHAUSDT,YGGUSDT,PENDLEUSDT,REZUSDT"
MIN_24H_VOLUME_USD=50000000  # $50M minimum for true mid-caps
```

#### 2. Asset Scanner (`internal/watcher/scanner.go`)
- **Function:** `getFuturesSymbols()`
- **Changed:** Static symbol list expanded from 15 to 45+ mid-cap assets
- **Criteria:** Excludes large caps (BTC, ETH, BNB, SOL) with market caps > $50B

#### 3. Watcher Config (`internal/watcher/watcher.go`)
- **Default watchlist:** Updated to mid-cap symbols
- **Volume filter:** Aligned with $50M minimum for mid-cap screening

## 🎯 Mid-Cap Criteria

### Volume Requirements
- **Minimum 24h Volume:** $50M USD
- **Rationale:** Ensures sufficient liquidity for scalping without slippage

### Asset Selection (45 Mid-Cap Assets)

**Tier 1 - Established Mid-Caps** ($1B-$10B market cap):
- ADA, DOT, AVAX, MATIC, LINK, UNI, LTC, BCH, ETC, ATOM, XMR

**Tier 2 - Growth Mid-Caps** ($500M-$2B market cap):
- FIL, ALGO, ICP, NEAR, GRT, FTM, MANA, HBAR, EGLD, FLOW, XTZ
- AAVE, MKR, RUNE, ENS, IMX, INJ, LDO, OP

**Tier 3 - Emerging Mid-Caps** ($100M-$1B market cap):
- API3, BLUR, ARKM, ALPHA, YGG, PENDLE, REZ, SUI, SEI, TIA
- MANTA, STRK, ZK, AEVO, XLM, VET, TRX, AXS, KSM

## 📊 Why Mid-Caps for Scalping?

**Advantages:**
- ✅ Higher volatility than large caps (more profit opportunities)
- ✅ Better liquidity than small caps (reduced slippage)
- ✅ Less manipulated by institutional players
- ✅ More responsive to technical patterns
- ✅ Lower correlation during market stress

**Filter Settings:**
```go
Min24hVolumeUSD:   50_000_000   // $50M minimum
MinATRPercent:     0.5          // 0.5% minimum volatility
MaxAssets:         15           // Focus on top 15 opportunities
VolumeMultiplier:  3.0          // 3x volume spike detection
```

## 🔧 Configuration Files Modified

1. **`.env`** - Main environment configuration
2. **`internal/watcher/scanner.go`** - Dynamic asset scanning
3. **`internal/watcher/watcher.go`** - Default watchlist settings

## 🚀 Next Steps

1. **Verify your API keys are set:**
   ```bash
   grep BINANCE_API_ .env
   ```

2. **Start the bot:**
   ```bash
   ./cognee
   ```

3. **Monitor the scanner logs:**
   ```bash
   tail -f startup.log | grep "mid-cap"
   ```

4. **Watch for trading opportunities:**
   ```bash
   tail -f startup.log | grep "FVG opportunity"
   ```

## ⚠️ Important Notes

- **Testnet First:** Keep `BINANCE_USE_TESTNET=true` until profitable
- **Volume Filter:** $50M ensures you're trading liquid mid-caps, not illiquid assets
- **Scanning Interval:** Assets are re-scored every 10 minutes for fresh opportunities
- **Top 15 Focus:** Bot automatically selects the 15 highest-scoring mid-caps

## 📈 Expected Behavior

1. Bot scans **45+ mid-cap symbols** continuously
2. Filters for **$50M+ daily volume** and **0.5%+ ATR**
3. Scores assets by: volatility + volume spike + RSI + EMA alignment
4. Trades only the **top 15 highest-scoring** mid-caps
5. FVG opportunities trigger in the **1-15 minute timeframe**

---

Your GOBOT is now configured to systematically scan, filter, and trade high-volatility mid-cap assets on Binance Futures! 🎯
