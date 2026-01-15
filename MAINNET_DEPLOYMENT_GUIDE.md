# GOBOT Mainnet Deployment Guide

## Prerequisites

1. **Binance Futures API Keys**
   - Go to: https://www.binance.com/en/my/settings/api-management
   - Create new API key with "Futures" permissions
   - Enable "Read" and "Trade" permissions

2. **Telegram Bot** (optional but recommended)
   - Message @BotFather on Telegram
   - Create new bot: `/newbot`
   - Get bot token
   - Get chat ID from @userinfobot

## Quick Start

### Step 1: Configure API Keys
```bash
cd /Users/britebrt/GOBOT
./setup-mainnet.sh
```

Or manually edit `.env.mainnet`:
```bash
nano .env.mainnet
# Update:
# - BINANCE_API_KEY
# - BINANCE_API_SECRET
# - TELEGRAM_TOKEN (optional)
# - AUTHORIZED_CHAT_ID (optional)
```

### Step 2: Validate Configuration
```bash
./mainnet-deploy.sh check
```

### Step 3: Test Connectivity
```bash
./mainnet-deploy.sh connect
```

### Step 4: Deploy to Mainnet (REAL TRADING)
```bash
./mainnet-deploy.sh deploy --confirm
```

## Safety Features

| Feature | Setting | Purpose |
|---------|---------|---------|
| Position Limit | $50 max | Limits per-trade exposure |
| Daily Limit | $200 max | Limits daily losses |
| Stop Loss | 2% | Auto-close losing trades |
| Take Profit | 4% | Auto-close winning trades |
| Kill Switch | File-based | Immediate halt |
| Telegram Alerts | Configurable | Real-time notifications |

## Emergency Controls

**Manual Kill Switch:**
```bash
echo "stop" > /tmp/gobot_kill_switch
```

**Stop All Trading:**
```bash
./mainnet-deploy.sh stop
```

**View Status:**
```bash
./mainnet-deploy.sh status
```

**View Logs:**
```bash
tail -f /tmp/gobot_mainnet.log
```

## File Locations

| File | Purpose |
|------|---------|
| `.env.mainnet` | Mainnet configuration |
| `mainnet-deploy.sh` | Deployment script |
| `/tmp/gobot_mainnet.log` | Live trading logs |
| `/Users/britebrt/GOBOT/logs/mainnet_audit.log` | Audit trail |
| `/Users/britebrt/GOBOT/logs/trades.log` | Trade history |

## Monitoring

### Check Status
```bash
./mainnet-deploy.sh status
```

### View Live Logs
```bash
./mainnet-deploy.sh logs
```

### Check API Health
```bash
curl https://api.binance.com/api/v3/ping
```

## Troubleshooting

### GOBOT won't start
```bash
tail -f /tmp/gobot_mainnet.log
```

### API errors
```bash
./mainnet-deploy.sh connect
```

### Need to stop immediately
```bash
./mainnet-deploy.sh stop
# or
echo "stop" > /tmp/gobot_kill_switch
```

## Important Warnings

⚠️ **REAL MONEY AT RISK**: This bot trades with real USDT
- Start with small position limits ($50)
- Monitor closely for first 30 minutes
- Keep Telegram alerts enabled
- Test connectivity before deploying

⚠️ **API SECURITY**:
- Never share your API keys
- Enable IP restrictions on Binance
- Use read-only keys when possible

⚠️ **NETWORK**:
- Ensure stable internet connection
- Monitor latency to Binance API
- Enable kill switch for emergencies
