# QuantCrawler Integration for GOBOT

This integration adds AI-powered trade analysis using QuantCrawler to your GOBOT trading bot.

## Architecture

```
┌──────────────┐     POST /webhook/quantcrawler-analysis     ┌─────────────┐
│   Go Bot     │ ─────────────────────────────────────────►  │    N8N      │
│  Screener    │                                            │  Workflow   │
└──────────────┘                                            └──────┬──────┘
                                                                  │
                                                                  ▼
                                                       ┌─────────────────────┐
                                                       │  Puppeteer Server   │
                                                       │  (localhost:3456)   │
                                                       └──────────┬──────────┘
                                                                  │
         ┌────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     GOOGLE OAUTH FLOW                               │
│                                                                     │
│  1. Load saved session from n8n-sessions/                           │
│  2. If expired/not found → Redirect to Google OAuth                 │
│  3. Enter email + password                                          │
│  4. If 2FA → Complete manually (or use App Password)                │
│  5. Session saved for future use                                    │
└─────────────────────────────────────────────────────────────────────┘
                                                                  │
                                                                  ▼
                                                           ┌─────────────┐
                                                           │ QuantCrawler│
                                                           │   Futures   │
                                                           │   Analyzer  │
                                                           └─────────────┘
```

## Quick Start

### 1. Install Dependencies

```bash
npm install
```

Or install Puppeteer manually:
```bash
cd n8n/scripts
npm install puppeteer
```

### 2. Configure Credentials

Edit `.env` and add your Google credentials:

```bash
QUANTCRAWLER_EMAIL=your-email@gmail.com
QUANTCRAWLER_PASSWORD=your-16-char-app-password
```

**Important:** If your Google account has 2FA enabled:
1. Go to https://myaccount.google.com/apppasswords
2. Create a new app password for "QuantCrawler"
3. Use the 16-character password (e.g., `abcd efgh ijkl mnop`)

### 3. Start the Puppeteer Server

```bash
# Option A: Use the launcher script
./start-quantcrawler.sh

# Option B: Run directly
node n8n/scripts/quantcrawler.js --webhook &
```

Expected output:
```
==============================================
  GOBOT + QuantCrawler Launcher
==============================================
Checking ports...
Port 3456 available (QuantCrawler)
...

[QuantCrawler] Webhook server running on http://localhost:3456/webhook
```

### 4. Test the Integration

```bash
# Test from terminal
curl -X POST http://localhost:3456/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "1000PEPEUSDT",
    "account_balance": 1000,
    "current_price": 0.00001234
  }'
```

Expected response:
```json
{
  "symbol": "1000PEPEUSDT",
  "direction": "LONG",
  "confidence": 75,
  "entry": 0.00001234,
  "stop_loss": 0.00001210,
  "take_profit": 0.00001280,
  "options": [...],
  "timeframes": {...},
  ...
}
```

## File Structure

```
GOBOT/
├── n8n/
│   ├── scripts/
│   │   └── quantcrawler.js       # Puppeteer automation script
│   └── workflows/
│       ├── 04-quantcrawler-analysis.json  # N8N workflow
│       └── ...
├── n8n-sessions/                 # Saved Google sessions
├── start-quantcrawler.sh         # Launcher script
├── .env                          # Credentials (add here)
└── gobot                         # Main bot binary
```

## Managing Sessions

### View Session
```bash
cat n8n-sessions/quantcrawler-session.json
```

### Reset Session (if login issues)
```bash
rm -rf n8n-sessions/
# Next run will prompt for new login
```

### Session Expiration
- Sessions expire after 30 days
- Auto-reauthentication on next use
- No manual intervention needed

## N8N Workflow Setup

1. Open N8N at http://localhost:5678
2. Create new workflow
3. Import `n8n/workflows/04-quantcrawler-analysis.json`
4. Activate the workflow
5. Test with:
```bash
curl -X POST http://localhost:5678/webhook/quantcrawler-analysis \
  -H "Content-Type: application/json" \
  -d '{"symbol":"1000PEPEUSDT","account_balance":1000}'
```

## If QuantCrawler UI Changes

Update selectors in `n8n/scripts/quantcrawler.js`:

```javascript
const CONFIG = {
  selectors: {
    tickerInput: 'input[placeholder*="Ticker"]',    // Update this
    analyzeButton: 'button:has-text("Analyze")',     // Update this
    responseContainer: '[class*="response"]',       // Update this
    // ...
  }
};
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Missing credentials" | Set `QUANTCRAWLER_EMAIL` and `QUANTCRAWLER_PASSWORD` in `.env` |
| 2FA popup | Use App Password instead of regular password |
| Login fails | Try `rm -rf n8n-sessions/` then login again |
| Session not saved | Check directory permissions on `n8n-sessions/` |
| Timeout on analysis | QuantCrawler can take 60-90s, increase timeout |
| Port already in use | Stop existing process or change port in script |

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `QUANTCRAWLER_EMAIL` | Yes | Google account email |
| `QUANTCRAWLER_PASSWORD` | Yes | Password or App Password |
| `GOOGLE_EMAIL` | Alternative | Alternative env var |
| `GOOGLE_PASSWORD` | Alternative | Alternative env var |

## Security Notes

- Credentials stored in `.env` (not committed to git)
- Sessions stored in `n8n-sessions/` (gitignored)
- Use App Passwords for 2FA accounts
- Never commit real credentials
