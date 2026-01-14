# GOBOT - Complete Trading System

Local setup for automated meme coin trading with QuantCrawler AI analysis.

## Quick Start (3 Steps)

### Step 1: Install Dependencies
```bash
./setup-local.sh
```

### Step 2: Configure Credentials
Edit `.env` and add:
```bash
QUANTCRAWLER_EMAIL=your-email@gmail.com
QUANTCRAWLER_PASSWORD=your-16-char-app-password
```

**For 2FA accounts:** Use App Password from https://myaccount.google.com/apppasswords

### Step 3: Start All Services
```bash
./start-all.sh
```

---

## Services Started

| Service | URL | Description |
|---------|-----|-------------|
| QuantCrawler | http://localhost:3456 | Puppeteer automation |
| N8N | http://localhost:5678 | Workflow automation |
| GOBOT | http://localhost:8080 | Trading bot (run separately: `./gobot`) |

**N8N Credentials:** `gobot` / `secure_password`

---

## Testing

### Test Puppeteer Server
```bash
curl -X POST http://localhost:3456/webhook \
  -H "Content-Type: application/json" \
  -d '{"symbol":"1000PEPEUSDT","account_balance":1000}'
```

### Test N8N Workflow
```bash
curl -X POST http://localhost:5678/webhook/quantcrawler-analysis \
  -H "Content-Type: application/json" \
  -d '{"symbol":"1000PEPEUSDT","account_balance":1000}'
```

### Test All Meme Coins
```bash
for coin in 1000PEPEUSDT WIFUSDT POPCATUSDT TURBOUSDT MOGUSDT; do
  echo "=== $coin ==="
  curl -s -X POST http://localhost:5678/webhook/quantcrawler-analysis \
    -H "Content-Type: application/json" \
    -d "{\"symbol\":\"$coin\",\"account_balance\":1000}" | jq -c '{dir:.direction,conf:.confidence}'
done
```

---

## File Structure

```
GOBOT/
├── gobot                    # Main trading bot binary
├── start-all.sh            # Start all services
├── setup-local.sh          # Install dependencies
├── docker-compose.yml      # Docker alternative
├── .env                    # Credentials (add here)
├── n8n/
│   ├── workflows/
│   │   └── 04-quantcrawler-analysis.json  # N8N workflow
│   └── scripts/
│       └── quantcrawler.js  # Puppeteer automation
└── n8n-sessions/           # Saved Google sessions (auto-created)
```

---

## Managing Services

### Start All
```bash
./start-all.sh
```

### Start Only QuantCrawler
```bash
./start-quantcrawler.sh
```

### Start Only N8N
```bash
n8n start
```

### Start Only GOBOT
```bash
./gobot
```

---

## Docker Alternative

```bash
# Start all with Docker
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

---

## Session Management

### Reset Session (if login issues)
```bash
rm -rf n8n-sessions/
# Next run will prompt for new Google login
```

### View Session
```bash
cat n8n-sessions/quantcrawler-session.json
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Module not found: puppeteer" | Run `./setup-local.sh` |
| "Missing credentials" | Add to `.env` file |
| Port in use | Stop existing service or change port |
| 2FA popup | Use App Password instead |
| N8N won't start | Run `npm install -g n8n` |

---

## Stopping Services

```bash
# Ctrl+C in the terminal running start-all.sh

# Or kill manually
pkill -f "node.*quantcrawler"
pkill -f "n8n"
pkill -f "gobot"
```
