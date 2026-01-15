# GOBOT Testnet Observability & Optimization Configuration

## Overview

This document describes all observability modules enabled when running GOBOT on Binance Testnet.

## Active Modules

### 1. Brain Engine (AI Decision Making)
```
INFERENCE_MODE=LOCAL    # Uses local Ollama/MSTY for fast decisions
OLLAMA_MODEL=qwen3:0.6b # Lightweight model for testnet
ENABLE_RECOVERY=true    # Auto-recovery from failed trades
```

**Purpose**: Makes trading decisions based on market analysis
**Status**: ✅ Active when `OLLAMA_BASE_URL` is accessible

### 2. Feedback System (Learning)
```
FEEDBACK_ENABLED=true
FEEDBACK_DB_PATH=gobot_testnet.db
```

**Purpose**: Learns from past trades to improve decision making
**Status**: ✅ SQLite database tracks all decisions

### 3. Safe-Stop Protection
```
SAFE_STOP_ENABLED=true
SAFE_STOP_THRESHOLD_PERCENT=10
SAFE_STOP_MIN_BALANCE_USD=10
SAFE_STOP_CHECK_INTERVAL=60
```

**Purpose**: Auto-halt if balance drops significantly
**Status**: ✅ Monitors balance every 60 seconds

### 4. Meme Coin Screener
```
SCREENER_ENABLED=true
SCREENER_INTERVAL_SECONDS=300
SCREENER_MAX_PAIRS=5
SCREENER_MIN_VOLUME_24H=1000000
```

**Purpose**: Scans for high-volume, high-momentum opportunities
**Status**: ✅ Scans every 5 minutes

### 5. Telegram Notifications
```
TELEGRAM_NOTIFICATIONS=true
TELEGRAM_TOKEN=7334854261:...
AUTHORIZED_CHAT_ID=6250310715
```

**Purpose**: Real-time alerts for trades and alerts
**Status**: ✅ Sends notifications to configured chat

### 6. N8N Workflow Integration
```
N8N_BASE_URL=http://localhost:5678
N8N_TRADE_WEBHOOK=http://localhost:5678/webhook/trade_signal
N8N_ALERT_WEBHOOK=http://localhost:5678/webhook/alert
```

**Purpose**: Advanced automation workflows
**Status**: ✅ Webhooks ready for N8N

## Quick Reference

| Module | Env Var | Default | Testnet Value |
|--------|---------|---------|---------------|
| Brain Engine | `INFERENCE_MODE` | CLOUD | LOCAL |
| Feedback | `FEEDBACK_ENABLED` | true | true |
| Safe-Stop | `SAFE_STOP_ENABLED` | true | true |
| Screener | `SCREENER_ENABLED` | true | true |
| Telegram | `TELEGRAM_NOTIFICATIONS` | false | true |
| Recovery | `ENABLE_RECOVERY` | true | true |

## Starting Testnet with Full Observability

```bash
# 1. Set environment
export BINANCE_USE_TESTNET=true
export FEEDBACK_ENABLED=true
export SAFE_STOP_ENABLED=true
export SCREENER_ENABLED=true
export TELEGRAM_NOTIFICATIONS=true

# 2. Start GOBOT
./gobot

# 3. Check logs for module status
tail -f gobot.log | grep -E "Brain|Feedback|Safe-Stop|Screener"
```

## Monitoring Endpoints

| Service | URL | Status |
|---------|-----|--------|
| GOBOT | http://localhost:8080 | /health |
| Screenshot | http://localhost:3456 | /health |
| N8N | http://localhost:5678 | /health |

## Testnet Performance

Expected behavior on testnet:
- **Brain Engine**: ~100-500ms decision time (local model)
- **Screener**: Scans every 5 minutes
- **Safe-Stop**: Checks every 60 seconds
- **Feedback**: Writes to SQLite on every trade

## Optimization Tips

1. **For faster decisions**: Use LOCAL inference mode
2. **For better learning**: Keep FEEDBACK_ENABLED=true
3. **For safety**: Keep SAFE_STOP_ENABLED=true
4. **For alerts**: Configure TELEGRAM_NOTIFICATIONS=true

## Troubleshooting

### Brain Engine not responding?
```bash
# Check Ollama
curl http://localhost:11964/api/tags
```

### Feedback not recording?
```bash
# Check database
sqlite3 gobot_testnet.db "SELECT COUNT(*) FROM feedback;"
```

### Safe-Stop triggering too often?
```bash
# Increase threshold
export SAFE_STOP_THRESHOLD_PERCENT=20
```

## Next Steps

1. ✅ Test Binance connectivity: `./test-binance-testnet.sh`
2. ✅ Start GOBOT: `./gobot`
3. ✅ Monitor logs: `tail -f gobot.log`
4. ✅ Check Telegram for notifications
5. ✅ Review feedback learning in database
