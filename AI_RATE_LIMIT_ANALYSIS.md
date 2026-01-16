# AI Inference Rate Limit Analysis for 24/7 Operation

## Executive Summary

This document provides a comprehensive analysis of rate limits for free tier AI providers (Gemini, OpenRouter, Groq) and determines the optimal configuration for 24/7 trading bot operation.

## Daily AI Requirements

### Current Bot Operations

| Operation | Frequency | Daily Count | AI Required |
|-----------|-----------|-------------|-------------|
| Trading Decisions | Every 15 minutes | 96 | ✓ Yes |
| Market Analysis | Every 30 minutes | 48 | ✓ Yes |
| Position Monitoring | Every 30 seconds | 2,880 | ✗ No (local logic) |
| **Total AI Calls** | - | **144** | - |

### With Position Monitoring (Optional)

| Operation | Frequency | Daily Count | AI Required |
|-----------|-----------|-------------|-------------|
| Trading Decisions | Every 15 minutes | 96 | ✓ Yes |
| Market Analysis | Every 30 minutes | 48 | ✓ Yes |
| Position Health Check | Every 5 minutes | 288 | ✓ Yes |
| **Total AI Calls** | - | **432** | - |

## Free Tier Provider Limits

### 1. Gemini (Google)

**Model**: `gemini-1.5-flash` (Free Tier)

| Metric | Limit | Notes |
|--------|-------|-------|
| Requests Per Minute | 15 | 1 request every 4 seconds |
| Requests Per Day | 1,500 | Conservative estimate |
| Tokens Per Day | 1,500,000 | ~1.5M tokens |
| Cost | $0.00 | Completely free |
| Latency | ~1 second | Very fast |

**Capacity Calculation**:
```
Daily Requirement: 144 requests
Gemini Daily Limit: 1,500 requests
Safety Margin (80%): 1,200 requests

Maximum Bots Per Key: 1,200 / 144 = 8.33 bots
Safe Bots Per Key: 8 bots
```

**Single Key Sufficiency**: ✓ YES (supports 8x current needs)

### 2. OpenRouter

**Model**: `qwen/qwen-2.5-72b-instruct` (Free Tier)

| Metric | Limit | Notes |
|--------|-------|-------|
| Requests Per Minute | 20 | 1 request every 3 seconds |
| Requests Per Day | 1,440 | ~1 request per minute average |
| Tokens Per Day | 200,000 | Varies by model |
| Cost | $0.00 | Free tier |
| Latency | ~2 seconds | Moderate |

**Capacity Calculation**:
```
Daily Requirement: 144 requests
OpenRouter Daily Limit: 1,440 requests
Safety Margin (80%): 1,152 requests

Maximum Bots Per Key: 1,152 / 144 = 8 bots
Safe Bots Per Key: 8 bots
```

**Single Key Sufficiency**: ✓ YES (supports 8x current needs)

### 3. Groq

**Model**: `llama-3.3-70b-versatile` (Free Tier)

| Metric | Limit | Notes |
|--------|-------|-------|
| Requests Per Minute | 30 | 1 request every 2 seconds |
| Requests Per Day | 1,440 | ~1 request per minute average |
| Tokens Per Day | 1,000,000 | ~1M tokens |
| Cost | $0.00 | Completely free |
| Latency | ~0.5 seconds | Extremely fast |

**Capacity Calculation**:
```
Daily Requirement: 144 requests
Groq Daily Limit: 1,440 requests
Safety Margin (80%): 1,152 requests

Maximum Bots Per Key: 1,152 / 144 = 8 bots
Safe Bots Per Key: 8 bots
```

**Single Key Sufficiency**: ✓ YES (supports 8x current needs)

## Comparative Analysis

### Provider Comparison

| Provider | RPM | RPD | Tokens/Day | Latency | Capacity | Cost |
|----------|-----|-----|------------|---------|----------|------|
| Gemini | 15 | 1,500 | 1.5M | ~1s | 8 bots | FREE |
| OpenRouter | 20 | 1,440 | 200K | ~2s | 8 bots | FREE |
| Groq | 30 | 1,440 | 1M | ~0.5s | 8 bots | FREE |

### Recommendation Ranking

1. **Groq** - Best for latency (0.5s), high throughput (30 RPM)
2. **Gemini** - Good balance, reliable, 1.5M tokens/day
3. **OpenRouter** - Moderate latency, model variety

## Multi-Key Strategy

### Single Key Configuration (Current Needs)

**Sufficient for**: 1 bot running 24/7

```yaml
providers:
  gemini:
    api_keys: ["YOUR_GEMINI_API_KEY"]
    model: "gemini-1.5-flash"
  openrouter:
    api_keys: ["YOUR_OPENROUTER_API_KEY"]
    model: "qwen/qwen-2.5-72b-instruct"
  groq:
    api_keys: ["YOUR_GROQ_API_KEY"]
    model: "llama-3.3-70b-versatile"
```

**Daily Usage**:
- Gemini: 144 requests / 1,500 limit = 9.6% usage
- OpenRouter: 144 requests / 1,440 limit = 10% usage
- Groq: 144 requests / 1,440 limit = 10% usage

### Multi-Key Configuration (Recommended for Production)

**Sufficient for**: Up to 3 bots running 24/7

```yaml
providers:
  gemini:
    api_keys:
      - "YOUR_GEMINI_API_KEY_1"
      - "YOUR_GEMINI_API_KEY_2"
    model: "gemini-1.5-flash"
  openrouter:
    api_keys:
      - "YOUR_OPENROUTER_API_KEY_1"
      - "YOUR_OPENROUTER_API_KEY_2"
    model: "qwen/qwen-2.5-72b-instruct"
  groq:
    api_keys:
      - "YOUR_GROQ_API_KEY_1"
      - "YOUR_GROQ_API_KEY_2"
    model: "llama-3.3-70b-versatile"
```

**Daily Usage**:
- Gemini: 144 requests / 3,000 limit (2 keys) = 4.8% usage
- OpenRouter: 144 requests / 2,880 limit (2 keys) = 5% usage
- Groq: 144 requests / 2,880 limit (2 keys) = 5% usage

## High-Load Scenario Analysis

### Scenario: Position Monitoring with AI

**Daily Requirement**: 432 requests

**Single Key Usage**:
- Gemini: 432 / 1,500 = 28.8% usage
- OpenRouter: 432 / 1,440 = 30% usage
- Groq: 432 / 1,440 = 30% usage

**Recommendation**: Use 2 keys per provider for safety margin

### Scenario: Multiple Bots (3 bots)

**Daily Requirement**: 432 requests (144 per bot × 3 bots)

**Multi-Key Usage**:
- Gemini: 432 / 3,000 (2 keys) = 14.4% usage
- OpenRouter: 432 / 2,880 (2 keys) = 15% usage
- Groq: 432 / 2,880 (2 keys) = 15% usage

**Recommendation**: Use 3 keys per provider for redundancy

## Token Usage Analysis

### Estimated Tokens Per Request

| Operation Type | Input Tokens | Output Tokens | Total |
|---------------|--------------|---------------|-------|
| Trading Decision | ~400 | ~150 | ~550 |
| Market Analysis | ~600 | ~200 | ~800 |
| Position Health Check | ~300 | ~100 | ~400 |

### Daily Token Usage

**Current Operations (144 requests)**:
- Trading Decisions (96 × 550): 52,800 tokens
- Market Analysis (48 × 800): 38,400 tokens
- **Total**: 91,200 tokens/day

**With Position Monitoring (432 requests)**:
- Trading Decisions (96 × 550): 52,800 tokens
- Market Analysis (48 × 800): 38,400 tokens
- Position Health (288 × 400): 115,200 tokens
- **Total**: 206,400 tokens/day

### Token Limits

| Provider | Daily Limit | Current Usage | With Monitoring |
|----------|-------------|---------------|-----------------|
| Gemini | 1,500,000 | 6% | 13.8% |
| OpenRouter | 200,000 | 45.6% | 103.2% ⚠️ |
| Groq | 1,000,000 | 9.1% | 20.6% |

**OpenRouter Token Limit Warning**: OpenRouter has a lower token limit (200K/day). For position monitoring with AI, OpenRouter may hit token limits.

## Recommended Configuration

### Option 1: Single Key (Minimal Setup)

**Best for**: Single bot, 24/7 operation, no position monitoring AI

```yaml
# .env
GEMINI_API_KEY=your_gemini_key
OPENROUTER_API_KEY=your_openrouter_key
GROQ_API_KEY=your_groq_key

# config.yaml
brain:
  inference_mode: "CLOUD"
  cloud_provider: "gemini"  # Primary: Gemini
  fallback_providers: ["groq", "openrouter"]  # Fallback order
```

**Pros**:
- Simple setup
- Sufficient for current needs
- Low resource usage

**Cons**:
- Single point of failure
- No redundancy
- Limited scalability

### Option 2: Multi-Key (Recommended)

**Best for**: Production, redundancy, future scalability

```yaml
# .env
GEMINI_API_KEY_1=your_gemini_key_1
GEMINI_API_KEY_2=your_gemini_key_2
OPENROUTER_API_KEY_1=your_openrouter_key_1
OPENROUTER_API_KEY_2=your_openrouter_key_2
GROQ_API_KEY_1=your_groq_key_1
GROQ_API_KEY_2=your_groq_key_2

# config.yaml
brain:
  inference_mode: "CLOUD"
  cloud_provider: "groq"  # Primary: Groq (fastest)
  fallback_providers: ["gemini", "openrouter"]  # Fallback order
  multi_key_enabled: true
  auto_fallback: true
```

**Pros**:
- High redundancy
- Load balancing
- Future-proof
- Better uptime

**Cons**:
- More API keys to manage
- Slightly more complex setup

### Option 3: Multi-Key with Position Monitoring

**Best for**: Advanced operation with AI-assisted position monitoring

```yaml
# .env
GEMINI_API_KEY_1=your_gemini_key_1
GEMINI_API_KEY_2=your_gemini_key_2
GEMINI_API_KEY_3=your_gemini_key_3
GROQ_API_KEY_1=your_groq_key_1
GROQ_API_KEY_2=your_groq_key_2
GROQ_API_KEY_3=your_groq_key_3
# Note: Avoid OpenRouter for high token usage scenarios

# config.yaml
brain:
  inference_mode: "CLOUD"
  cloud_provider: "gemini"  # Primary: Gemini (high token limit)
  fallback_providers: ["groq"]  # Fallback: Groq
  multi_key_enabled: true
  auto_fallback: true
  position_monitoring_ai: true
```

**Pros**:
- Full AI coverage
- High token limits
- Maximum redundancy

**Cons**:
- Most API keys required
- Higher token usage
- More complex

## Implementation Guide

### Step 1: Get API Keys

**Gemini**:
1. Go to https://makersuite.google.com/app/apikey
2. Create new API key
3. Copy key to `.env`

**OpenRouter**:
1. Go to https://openrouter.ai/keys
2. Create new API key
3. Copy key to `.env`

**Groq**:
1. Go to https://console.groq.com/keys
2. Create new API key
3. Copy key to `.env`

### Step 2: Configure Environment Variables

```bash
# .env
# Primary provider (recommended: Groq for speed)
GROQ_API_KEY=your_groq_key_here

# Fallback providers
GEMINI_API_KEY=your_gemini_key_here
OPENROUTER_API_KEY=your_openrouter_key_here

# Multi-key setup (optional but recommended)
GROQ_API_KEY_2=your_groq_key_2_here
GEMINI_API_KEY_2=your_gemini_key_2_here
OPENROUTER_API_KEY_2=your_openrouter_key_2_here
```

### Step 3: Update Configuration

```yaml
# config/config.yaml
brain:
  inference_mode: "CLOUD"
  cloud_provider: "groq"  # Primary: Groq (fastest)
  fallback_providers: ["gemini", "openrouter"]
  multi_key_enabled: true
  auto_fallback: true
  decision_timeout: 15s
  max_concurrent_decisions: 5
```

### Step 4: Test Configuration

```bash
# Test single provider
curl -X POST http://localhost:8080/test/brain \
  -H "Content-Type: application/json" \
  -d '{"provider": "gemini"}'

# Test multi-key fallback
curl -X POST http://localhost:8080/test/brain \
  -H "Content-Type: application/json" \
  -d '{"provider": "groq", "test_fallback": true}'
```

## Rate Limit Monitoring

### Key Metrics to Monitor

1. **Requests Per Minute (RPM)**
   - Current: 0.1 RPM (144 / 1440 minutes)
   - Limit: 15-30 RPM depending on provider
   - Usage: < 1% of limit

2. **Requests Per Day (RPD)**
   - Current: 144 RPD
   - Limit: 1,440-1,500 RPD
   - Usage: 10% of limit

3. **Tokens Per Day**
   - Current: 91,200 tokens
   - Limit: 200K-1.5M tokens
   - Usage: 6-45% of limit

### Monitoring Commands

```bash
# Check current usage
curl http://localhost:8080/metrics/brain

# Check rate limits
curl http://localhost:8080/metrics/rate-limits

# Check provider health
curl http://localhost:8080/health/providers
```

## Fallback Strategy

### Automatic Fallback Logic

1. **Primary Provider**: Groq (fastest)
2. **Fallback 1**: Gemini (reliable)
3. **Fallback 2**: OpenRouter (variety)

### Fallback Triggers

- Rate limit exceeded (429 error)
- API key invalid (401 error)
- Service unavailable (503 error)
- Timeout (> 15 seconds)
- 5 consecutive errors

### Fallback Behavior

```
Groq (Primary)
  ↓ (error)
Gemini (Fallback 1)
  ↓ (error)
OpenRouter (Fallback 2)
  ↓ (error)
Local Ollama (Last Resort)
```

## Cost Analysis

### Free Tier Costs

| Provider | Daily Cost | Monthly Cost | Annual Cost |
|----------|------------|--------------|-------------|
| Gemini | $0.00 | $0.00 | $0.00 |
| OpenRouter | $0.00 | $0.00 | $0.00 |
| Groq | $0.00 | $0.00 | $0.00 |
| **Total** | **$0.00** | **$0.00** | **$0.00** |

### Paid Tier Comparison (For Reference)

| Provider | Model | Cost/1K Tokens | Daily Cost |
|----------|-------|---------------|------------|
| OpenAI | GPT-4 Turbo | $0.01 | $0.91 |
| Anthropic | Claude Sonnet | $0.003 | $0.27 |

**Conclusion**: Free tiers are sufficient for current needs.

## Recommendations

### For Current Setup (Single Bot, 24/7)

✅ **Use Option 1 (Single Key)**:
- 1 Gemini API key
- 1 Groq API key
- 1 OpenRouter API key
- Total: 3 API keys

### For Production Setup (Redundancy)

✅ **Use Option 2 (Multi-Key)**:
- 2 Gemini API keys
- 2 Groq API keys
- 2 OpenRouter API keys
- Total: 6 API keys

### For Advanced Setup (Position Monitoring AI)

✅ **Use Option 3 (Multi-Key + High Token)**:
- 3 Gemini API keys (high token limit)
- 3 Groq API keys
- Skip OpenRouter (low token limit)
- Total: 6 API keys

## Conclusion

### Summary

1. **Single Key Configuration**: Sufficient for current 24/7 operation
   - Daily usage: 144 requests
   - Provider limits: 1,440-1,500 requests/day
   - Usage: 10% of limits
   - Recommendation: 1 key per provider (3 total)

2. **Multi-Key Configuration**: Recommended for production
   - Daily usage: 144 requests
   - Provider limits: 2,880-3,000 requests/day (2 keys)
   - Usage: 5% of limits
   - Recommendation: 2 keys per provider (6 total)

3. **High-Load Configuration**: For position monitoring AI
   - Daily usage: 432 requests
   - Provider limits: 2,880-3,000 requests/day (2-3 keys)
   - Usage: 15-20% of limits
   - Recommendation: 3 keys per provider (6-9 total)

### Final Recommendation

**Start with Option 1 (Single Key)**:
- 1 Gemini API key
- 1 Groq API key
- 1 OpenRouter API key

**Upgrade to Option 2 (Multi-Key) for production**:
- 2 Gemini API keys
- 2 Groq API keys
- 2 OpenRouter API keys

**Use Option 3 (High Token) if adding position monitoring AI**:
- 3 Gemini API keys
- 3 Groq API keys
- Skip OpenRouter

All configurations are **FREE** and support 24/7 operation with significant headroom.

---

**Last Updated**: 2026-01-16
**Version**: 1.0.0