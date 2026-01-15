# GOBOT AI Providers Configuration

## Active Providers (All FREE)

| Priority | Provider | Model | API | Status |
|----------|----------|-------|-----|--------|
| 1 | OpenRouter | Qwen 2.5 72B | openrouter.ai | ✅ Working |
| 2 | OpenRouter | DeepSeek Chat | openrouter.ai | ✅ Working |
| 3 | Groq | Kimi-K2 (Moonshot) | groq.com | ⚠️ Rate Limited |
| 4 | Groq | Llama 3.3 70B | groq.com | ⚠️ Rate Limited |
| 5 | Google | Gemini 1.5 Flash | googleapis.com | Optional |

## API Keys (In .env)

```bash
GROQ_API_KEY=<GROQ_API_KEY>
OPENROUTER_API_KEY=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
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1/chat/completions
```

## Token Usage

```
Per Request: ~400 prompt + 150 response = 550 tokens
Hourly Limit: 20 requests (safety margin)
Total Hourly: ~11,000 tokens
Daily: ~264,000 tokens
Monthly: ~7.9M tokens (well within free tiers)
```

## Sample Output

```json
{
  "symbol": "XRPUSDT",
  "action": "HOLD",
  "confidence": 0.8,
  "reasoning": "XRPUSDT is currently in a consolidation phase...",
  "source": "openrouter-qwen"
}
```

## Fallback Chain

```
Qwen 2.5 72B → DeepSeek → Kimi-K2 → Llama 3.3 → Gemini → Random
```

## Quick Test

```bash
cd /Users/britebrt/GOBOT/services/screenshot-service

# Test with OpenRouter (primary)
OPENROUTER_API_KEY=... node ai-analyzer.js BTCUSDT 100

# Test full workflow
OPENROUTER_API_KEY=... node auto-trade.js BTCUSDT 100
```

## Rate Limit Safety

- OpenRouter: Varies by model, generally generous
- Groq: ~30 RPM, ~10K tokens/min
- Safe: 20 requests/hour max

## Notes

- Groq may have rate limits during peak hours
- OpenRouter provides consistent access
- Qwen 2.5 72B on OpenRouter is highly reliable for crypto analysis
- All providers are completely FREE
