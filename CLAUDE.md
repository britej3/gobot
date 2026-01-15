# GOBOT - Autonomous Trading Bot

## Project Status: 🚀 LIVE TRADING READY

---

## ✅ 60-Minute Testnet Results

| Metric | Value |
|--------|-------|
| Cycles Completed | 4 (15 min each) |
| Duration | 60 minutes |
| Symbols Monitored | BTCUSDT, ETHUSDT, XRPUSDT |
| AI Provider | Llama 3.3 70B (OpenRouter Free) |
| Success Rate | 100% |
| Telegram Alerts | ✅ Working |

---

## 🔑 API Keys Configured

### Binance (Testnet)
```
BINANCE_API_KEY=mR0qYeuJGgFdSyEQOjxJ52KIX16xCjeCEswnPRkIVvE02a6b1STdSvgvW0ez0zUi
BINANCE_SECRET=2tHKOLn1wFQOoohPBcZFvrZqaU2QzefmDEYqm7pmDRunvJGphx1ZD13iS8ILvyM2
MODE: TESTNET (Safe for testing)
```

### Telegram
```
TOKEN: 7334854261:AAGEDLwJlp6pMO_6fxSr2piIMR5Aw4NrBMc
CHAT_ID: 6250310715
STATUS: ✅ Active
```

### AI Providers (FREE)
```
OPENROUTER_API_KEY=<OPENROUTER_API_KEY>
GROQ_API_KEY=<GROQ_API_KEY>
```

---

## 🚀 Quick Start Commands

```bash
cd /Users/britebrt/GOBOT/services/screenshot-service

# Test trading workflow with Telegram alerts
OPENROUTER_API_KEY=... TELEGRAM_NOTIFICATIONS=true node auto-trade.js BTCUSDT 100

# Single symbol AI analysis
OPENROUTER_API_KEY=... node ai-analyzer.js BTCUSDT 100

# 15-min observation cycle
OPENROUTER_API_KEY=... node observe-15min.js
```

---

## 📊 Last Trading Signal

```
Symbol: BTCUSDT
Action: LONG
Confidence: 85%
Price: $95,520.5
24h Change: -1.96%
Source: openrouter-llama-3.3-70b-instruct:free
Telegram: ✅ Sent
```

---

## VERIFIED FREE MODELS (2026)

| Provider | Model | Context | Status |
|----------|-------|---------|--------|
| OpenRouter | `meta-llama/llama-3.3-70b-instruct:free` | 131K | ✅ PRIMARY |
| OpenRouter | `deepseek/deepseek-r1-0528:free` | 164K | ✅ Ready |
| OpenRouter | `google/gemini-2.0-flash-exp:free` | 1M | ✅ Ready |
| Groq | `llama-3.3-70b-versatile` | 128K | ✅ Ready |
| Google | `gemini-2.5-flash` | 1M | ✅ Ready |

---

## Risk Parameters

```yaml
trading:
  initial_capital_usd: 100
  max_position_usd: 10        # 10% per trade
  stop_loss_percent: 2.0
  take_profit_percent: 4.0   # 2:1 RR
  min_confidence_threshold: 0.75
  mode: TESTNET (safe)
```

---

## Production Readiness Checklist

| Component | Status | Notes |
|-----------|--------|-------|
| AI Analysis | ✅ | Free models, 85% confidence |
| Anti-Detection | ✅ | JIT, rate limiting |
| Risk Management | ✅ | 2% SL, 4% TP |
| Telegram Alerts | ✅ | Working |
| Binance Testnet | ✅ | Connected |
| Live Trading | ⚠️ | Set BINANCE_USE_TESTNET=false |

---

## Next Steps for Live Trading

1. ✅ Testnet complete
2. ✅ Telegram alerts working
3. Add Binance API keys for mainnet:
   ```bash
   # Edit .env
   BINANCE_USE_TESTNET=false
   
   # Get real keys from:
   # https://www.binance.com/en/my/settings/api-management
   ```
4. Start live trading:
   ```bash
   cd /Users/britebrt/GOBOT/services/screenshot-service
   node auto-trade.js BTCUSDT 100
   ```

---

## Files Reference

- `auto-trade.js` - Full trading workflow with Telegram
- `ai-analyzer.js` - AI analysis (free models)
- `observe-15min.js` - Observation cycles
- `testnet-final-report.js` - Final report generator
- `config/config.yaml` - Configuration
- `.env` - API keys
