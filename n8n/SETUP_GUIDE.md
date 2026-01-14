# N8N Setup Guide for GOBOT

## Overview

This guide explains how to set up N8N workflows for GOBOT trading automation.

---

## 1. Start N8N

### Docker (Recommended)
```bash
docker run -it --rm \
  -p 5678:5678 \
  -v n8n_data:/home/node/.n8n \
  -e N8N_BASIC_AUTH_ACTIVE=true \
  -e N8N_BASIC_AUTH_USER=gobot \
  -e N8N_BASIC_AUTH_PASSWORD=secure_password \
  -e WEBHOOK_URL=https://your-domain.com/ \
  n8nio/n8n
```

### npm
```bash
npm install n8n -g
export N8N_BASIC_AUTH_ACTIVE=true
export N8N_BASIC_AUTH_USER=gobot
export N8N_BASIC_AUTH_PASSWORD=your_password
export WEBHOOK_URL=http://localhost:5678/
n8n start
```

---

## 2. Import Workflows

### Option A: Import from Files
1. Open N8N at http://localhost:5678
2. Login with credentials from `.env`
3. Click "Import from File"
4. Select files from `n8n/workflows/`:
   - `01-trade-signal.json`
   - `02-risk-alert.json`
   - `03-market-analysis.json`

### Option B: Manual Setup

#### Workflow 1: Trade Signal Handler
1. Create new workflow
2. Add **Webhook** node:
   - Method: POST
   - Path: `gobot-trade-signal`
3. Add **Function** node (Parse Signal):
```javascript
const signal = $input.first().json;
return [{
  json: {
    symbol: signal.symbol,
    action: signal.action,
    confidence: signal.confidence,
    price: signal.price,
    reason: signal.reason
  }
}];
```
4. Add **Telegram** node (configure credentials)
5. Connect nodes

---

## 3. Configure Credentials

### Telegram Bot
1. Create bot via @BotFather on Telegram
2. Get bot token
3. In N8N: Settings → Credentials → Telegram API
4. Add token

### OpenAI (for AI Analysis)
1. Get API key from https://platform.openai.com/api-keys
2. In N8N: Settings → Credentials → OpenAI API
3. Add key

### Binance (Optional)
1. Get API key from https://www.binance.com/en/my/settings/api-management
2. In N8N: Settings → Credentials → Binance API
3. Add key/secret

---

## 4. GOBOT Webhook Endpoints

GOBOT exposes these webhooks:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `http://localhost:8080/webhook/trade_signal` | POST | Receive trade signals |
| `http://localhost:8080/webhook/risk-alert` | POST | Receive risk alerts |
| `http://localhost:8080/webhook/market-analysis` | POST | AI market analysis |
| `http://localhost:8080/health` | GET | Health check |

### Configure N8N Webhooks

In each N8N workflow, update the webhook URLs:

**Trade Signal Workflow:**
```
Webhook URL: http://localhost:8080/webhook/trade_signal
```

**Risk Alert Workflow:**
```
Webhook URL: http://localhost:8080/webhook/risk-alert
```

**Market Analysis Workflow:**
```
Webhook URL: http://localhost:8080/webhook/market-analysis
```

---

## 5. N8N Environment Variables

Add to your `.env` or N8N settings:

```bash
# Telegram
TELEGRAM_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id

# OpenAI (for AI Analysis workflow)
OPENAI_API_KEY=sk-xxx

# N8N Authentication
N8N_BASIC_AUTH_USER=gobot
N8N_BASIC_AUTH_PASSWORD=secure_password

# Webhook URL (change for production)
WEBHOOK_URL=http://your-server:5678/
```

---

## 6. Test Webhooks

### Test Trade Signal
```bash
curl -X POST http://localhost:8080/webhook/trade_signal \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BTCUSDT",
    "action": "buy",
    "confidence": 0.85,
    "price": 65000.00,
    "reason": "RSI oversold, bullish divergence"
  }'
```

### Test Risk Alert
```bash
curl -X POST http://localhost:8080/webhook/risk-alert \
  -H "Content-Type: application/json" \
  -d '{
    "position": "BTCUSDT",
    "pnl_percent": -5.5,
    "pnl_value": -350.00,
    "health_score": 35,
    "reason": "Large drawdown detected",
    "action": "reduce"
  }'
```

### Health Check
```bash
curl http://localhost:8080/health
```

---

## 7. N8N Workflow Triggers

### Schedule-based (Daily Report)
Add **Schedule** node:
```
Expression: {{ $json.scheduled_time }}
Trigger: Every day at 9:00 AM
```

### Webhook from External Sources
Configure external services to call N8N webhooks:
```
TradingView Alerts → N8N Webhook → GOBOT
```

---

## 8. Production Setup

### 1. SSL/TLS with Nginx

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        return 301 https://$server_name$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    
    location / {
        proxy_pass http://localhost:5678;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

### 2. Environment Variables

```bash
# In docker-compose.yml
environment:
  - N8N_HOST=your-domain.com
  - N8N_PORT=5678
  - N8N_PROTOCOL=https
  - WEBHOOK_URL=https://your-domain.com/
  - N8N_BASIC_AUTH_ACTIVE=true
  - N8N_BASIC_AUTH_USER=gobot
  - N8N_BASIC_AUTH_PASSWORD=secure_password
  - GENERIC_TIMEZONE=UTC
```

### 3. Backup

```bash
# Backup N8N data
docker cp n8n_container:/home/node/.n8n ./n8n_backup
```

---

## 9. Troubleshooting

### Webhook Not Receiving
1. Check firewall rules (port 8080 for GOBOT, 5678 for N8N)
2. Verify webhook URL is accessible
3. Check GOBOT logs for errors

### Telegram Not Sending
1. Verify bot token
2. Check chat ID
3. Ensure bot has permission to send messages

### AI Analysis Failing
1. Check OpenAI API key
2. Verify account has credits
3. Check rate limits

---

## 10. Files Reference

| File | Path | Description |
|------|------|-------------|
| Trade Signal Workflow | `n8n/workflows/01-trade-signal.json` | Process trade signals |
| Risk Alert Workflow | `n8n/workflows/02-risk-alert.json` | Handle risk alerts |
| Market Analysis Workflow | `n8n/workflows/03-market-analysis.json` | AI market analysis |
| GOBOT Main App | `cmd/cobot/main.go` | Main application |
| LLM Config | `config/llm.go` | LLM configuration |
| Environment | `.env` | Secrets and settings |

---

## Quick Start Summary

```bash
# 1. Configure .env
nano .env

# 2. Start GOBOT
./gobot

# 3. Start N8N (separate terminal)
docker run -it --rm \
  -p 5678:5678 \
  -e N8N_BASIC_AUTH_ACTIVE=true \
  -e N8N_BASIC_AUTH_USER=gobot \
  -e N8N_BASIC_AUTH_PASSWORD=your_password \
  n8nio/n8n

# 4. Open N8N: http://localhost:5678
# 5. Import workflows from n8n/workflows/
# 6. Activate workflows
# 7. Test with curl commands above
```
