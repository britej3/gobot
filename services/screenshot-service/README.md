# GOBOT AI Trading Service

Agent-browser based trading analysis service.

## Architecture

- **agent-browser** (vercel-labs) - Headless browser automation
- **QuantCrawler** - Professional trading analysis with screenshot upload
- **Free Tier AI Models** - Groq, OpenRouter free models, Gemini
- **GOBOT webhook** - Trade signal execution

## Files

| File | Purpose |
|------|---------|
| `auto-trade.js` | Main trading workflow with QuantCrawler |
| `quantcrawler-integration.js` | QuantCrawler screenshot upload & report retrieval |
| `ai-analyzer.js` | AI chart analysis (Groq, OpenRouter, Gemini) |

## Usage

```bash
cd /Users/britebrt/GOBOT/services/screenshot-service

# Basic trading cycle with QuantCrawler
node auto-trade.js BTCUSDT 1000

# QuantCrawler analysis only
node quantcrawler-integration.js ETHUSDT 500

# AI analysis only (Groq, OpenRouter, Gemini)
node ai-analyzer.js XRPUSDT 500
```

## Environment Variables

```bash
export QUANTCRAWLER_EMAIL=britej3@gmail.com      # For TradingView login
export QUANTCRAWLER_PASSWORD=xxxx...           # Google App Password
export GROQ_API_KEY=gsk_...                     # Groq API (30 RPM, ~10K tokens/min)
export OPENROUTER_API_KEY=sk-or-...             # OpenRouter free models
export GEMINI_API_KEY=AIza...                   # Gemini API
```

## Free Tier AI Models

**Priority Order:**
1. **Groq** - llama-3.3-70b-versatile (30 RPM, ~10K tokens/min)
2. **OpenRouter** - meta-llama/llama-3.3-70b-instruct:free (10-50 RPM)
3. **OpenRouter** - deepseek/deepseek-r1-0528:free (10-50 RPM)
4. **OpenRouter** - google/gemini-2.0-flash-exp:free (10-50 RPM)
5. **Gemini** - gemini-2.5-flash (10 RPM, 250 RPD)
6. **Fallback** - Random analysis

## Chart Capture

agent-browser captures TradingView charts:
- 1m timeframe
- 5m timeframe
- 15m timeframe

Charts saved to `screenshots/` directory.

## QuantCrawler Flow

1. Login to TradingView with Google OAuth
2. Capture 3 screenshots (1m, 5m, 15m)
3. Upload screenshots to QuantCrawler
4. Get detailed trading report (entry, exit, SL, TP, confidence)
5. Send signal to GOBOT webhook

## Output

QuantCrawler analysis returns structured signal:
```json
{
  "symbol": "BTCUSDT",
  "direction": "LONG",
  "confidence": 75,
  "entry": 96000.00,
  "stop_loss": 94000.00,
  "take_profit": 100000.00,
  "reasoning": "Strong bullish momentum...",
  "timeframe_analysis": {...},
  "key_levels": {...}
}
```

## Requirements

- Node.js 18+
- agent-browser (`npm install -g agent-browser`)
- Google App Password (for TradingView login)
- Free tier API keys (Groq, OpenRouter, or Gemini)
