# GOBOT AI Trading Service

Agent-browser based trading analysis service.

## Architecture

- **agent-browser** (vercel-labs) - Headless browser automation
- **GPT-4o Vision** - Technical analysis of TradingView charts
- **GOBOT webhook** - Trade signal execution

## Files

| File | Purpose |
|------|---------|
| `auto-trade.js` | Main trading workflow |
| `ai-analyzer.js` | AI chart analysis |
| `aggressive-mode.js` | Advanced trading mode |

## Usage

```bash
cd /Users/britebrt/GOBOT/services/screenshot-service

# Basic trading cycle
node auto-trade.js BTCUSDT 1000

# AI analysis only
node ai-analyzer.js ETHUSDT 500
```

## Environment Variables

```bash
export OPENAI_API_KEY=sk-...          # For GPT-4o Vision
export GOOGLE_EMAIL=you@gmail.com      # For authenticated TradingView
export GOOGLE_APP_PASSWORD=xxxx...     # App password
```

## Chart Capture

agent-browser captures TradingView charts:
- 1m timeframe
- 5m timeframe
- 15m timeframe

Charts saved to `screenshots/` directory.

## Output

AI analysis returns structured signal:
```json
{
  "symbol": "BTCUSDT",
  "action": "LONG",
  "confidence": 0.85,
  "entry_price": "96000.00",
  "stop_loss": "94000.00",
  "take_profit": "100000.00",
  "reasoning": "Bullish momentum..."
}
```

## Requirements

- Node.js 18+
- agent-browser (`npm install -g agent-browser`)
- OpenAI API key (optional, fallback analysis available)
