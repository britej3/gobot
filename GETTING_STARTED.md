# 🚀 GOBOT v2.0 - Getting Started Guide

## Prerequisites

- Go 1.25+
- Docker (for N8N)
- Binance API credentials

## Quick Start

### 1. Configure Environment

Edit `.env` file:

```bash
# Binance
BINANCE_API_KEY=your_key
BINANCE_API_SECRET=your_secret
BINANCE_USE_TESTNET=false

# N8N
N8N_BASE_URL=http://localhost:5678
N8N_WEBHOOK_USER=gobot
N8N_WEBHOOK_PASS=secure_password

# LLM Providers (Free Tier)
GROQ_API_KEY=your_groq_key
DEEPSEEK_API_KEY=your_deepseek_key
GEMINI_API_KEY=your_gemini_key
```

### 2. Build GOBOT

```bash
go build -o gobot ./cmd/cobot
```

### 3. Start Everything

```bash
# Start GOBOT + N8N
./gobot.sh start

# Or manually:
./gobot.sh start    # Terminal 1
# Open new terminal:
./gobot.sh n8n-import  # Import N8N workflows
```

### 4. Configure N8N

1. Open http://localhost:5678
2. Login: `gobot` / `gobot` (or from .env)
3. Import workflows:
   - `01-trade-signal.json`
   - `02-risk-alert.json`
   - `03-market-analysis.json`
4. Activate each workflow

### 5. Test

```bash
# Test webhooks
./gobot.sh test

# View logs
./gobot.sh logs
```

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `./gobot.sh start` | Start GOBOT and N8N |
| `./gobot.sh stop` | Stop all services |
| `./gobot.sh status` | Check status |
| `./gobot.sh test` | Test webhooks |
| `./gobot.sh logs` | View logs |
| `./gobot.sh n8n-import` | Import workflows to N8N |

---

## Webhook Endpoints

| Endpoint | URL |
|----------|-----|
| Trade Signal | `POST http://localhost:8080/webhook/trade_signal` |
| Risk Alert | `POST http://localhost:8080/webhook/risk-alert` |
| Market Analysis | `POST http://localhost:8080/webhook/market-analysis` |
| Health Check | `GET http://localhost:8080/health` |

---

## Test Webhooks

```bash
# Trade Signal
curl -X POST http://localhost:8080/webhook/trade_signal \
  -H "Content-Type: application/json" \
  -d '{"symbol":"BTCUSDT","action":"buy","confidence":0.85,"price":65000,"reason":"RSI oversold"}'

# Risk Alert
curl -X POST http://localhost:8080/webhook/risk-alert \
  -H "Content-Type: application/json" \
  -d '{"position":"BTCUSDT","pnl_percent":-5.5,"health_score":35,"reason":"Large drawdown"}'
```

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  GOBOT v2.0                      │
├─────────────────────────────────────────────────┤
│                                                 │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │Strategy │  │Selector │  │Executor │        │
│  └────┬────┘  └────┬────┘  └────┬────┘        │
│       │            │            │               │
│       └────────────┴────────────┘               │
│                    │                            │
│               ┌────┴────┐                       │
│               │  LLM    │                       │
│               │  Router │                       │
│               │(Groq/Gemini/DeepSeek)          │
│               └────┬────┘                       │
│                    │                            │
│       ┌────────────┼────────────┐               │
│       ▼            ▼            ▼               │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │ Binance │  │  N8N    │  │Telegram │        │
│  │   API   │  │Webhooks │  │  Alerts │        │
│  └─────────┘  └─────────┘  └─────────┘        │
│                                                 │
└─────────────────────────────────────────────────┘
```

---

## Directory Structure

```
GOBOT/
├── gobot                    # CLI script
├── gobot                    # Compiled binary
├── 01-trade-signal.json     # N8N workflow
├── 02-risk-alert.json       # N8N workflow
├── 03-market-analysis.json  # N8N workflow
├── .env                     # Configuration
├── cmd/cobot/main.go        # Main application
├── config/                  # Configuration
├── domain/                  # Domain models
├── services/                # Business logic
├── infra/                   # Infrastructure
├── n8n/                     # N8N workflows & guides
└── MODULAR_ARCHITECTURE.md  # Architecture docs
```

---

## Troubleshooting

### GOBOT won't start
```bash
# Check logs
./gobot.sh logs

# Verify .env
cat .env
```

### N8N won't start
```bash
# Check Docker
docker ps

# Check port
lsof -i:5678
```

### Webhooks not working
```bash
# Test health
curl http://localhost:8080/health

# Check firewall
sudo ufw status
```

---

## Next Steps

1. ✅ Configure API keys in `.env`
2. ✅ Run `./gobot.sh start`
3. ⏳ Import N8N workflows
4. ⏳ Activate workflows in N8N
5. ⏳ Test with sample data
6. ⏳ Deploy to production

---

## Support

- Documentation: `MODULAR_ARCHITECTURE.md`
- N8N Setup: `n8n/SETUP_GUIDE.md`
- LLM Integration: `LLM_ROUTING_N8N_INTEGRATION.md`
