# GOBOT AI Configuration - Kimi-K2 (Moonshot AI)

## Summary

GOBOT now uses **Kimi-K2** from Moonshot AI as the primary AI model for trading signals, with multiple fallbacks.

## Models (All FREE)

| Priority | Model | Provider | Context | Speed | Best For |
|----------|-------|----------|---------|-------|----------|
| 1 | moonshotai/kimi-k2-instruct | Groq | 8K | Fast | Crypto analysis |
| 2 | llama-3.3-70b-versatile | Groq | 128K | Medium | Complex reasoning |
| 3 | gemini-1.5-flash | Google | 1M | Fast | Last resort |

## Token Calculation

```
Prompt:           ~400 tokens
Response:         ~150 tokens
─────────────────────────
Total/Request:    ~550 tokens

Free Tier Limits:
- Groq: ~10,000 tokens/minute, ~30 requests/minute
- Safe Limit: 20 requests/hour = 11,000 tokens = OK ✅

Monthly Cost: $0 (all free tiers)
```

## Environment Variables

```bash
# Required for AI analysis
export GROQ_API_KEY=<GROQ_API_KEY>

# Optional (fallback)
export GEMINI_API_KEY=your_gemini_key
```

## Sample Output

```json
{
  "symbol": "BTCUSDT",
  "action": "HOLD",
  "confidence": 0.75,
  "reasoning": "Price compressed under $68k resistance with declining volume and RSI flattening",
  "source": "moonshot-kimi-k2",
  "model": "moonshotai/kimi-k2-instruct"
}
```

## Usage

```bash
# Run AI analysis
cd /Users/britebrt/GOBOT/services/screenshot-service
GROQ_API_KEY=... node ai-analyzer.js BTCUSDT 100

# Run full trading workflow
cd /Users/britebrt/GOBOT/services/screenshot-service
GROQ_API_KEY=... node auto-trade.js BTCUSDT 100
```

## Rate Limit Safety

```
Hourly Requests: 20 (out of 30 available)
Hourly Tokens: 11,000 (out of 10,000 limit - tight but OK)

Recommendations:
- Add 3 second delay between requests
- Use fallback models sparingly
- Monitor API response times
```

## Why Kimi-K2?

1. **Free on Groq** - No OpenAI costs
2. **Fast inference** - Great for trading
3. **Good crypto knowledge** - Moonshot AI trained on diverse data
4. **Simple prompts work well** - No complex prompting needed
5. **Reliable JSON output** - Consistent format

## Fallback Chain

```
Kimi-K2 → Llama 3.3 70B → Gemini 1.5 Flash → Random Fallback
   ↓           ↓               ↓                ↓
  ✅           ✅               ✅               ⚠️
```

If primary model fails (rate limit, error), automatically tries next model.
