# Position Manager - Intelligent Trade Monitoring

## Overview

The Position Manager monitors all open positions and automatically closes them when the probability of losing is higher than winning.

## Features

### 1. Position Takeover
- Automatically takes ownership of all open positions on startup
- Logs each position taken over
- No manual intervention needed

### 2. Health Monitoring
- Checks every 30 seconds
- Analyzes market trend (BULLISH/BEARISH/NEUTRAL)
- Calculates position health score (0-100)
- AI-powered win probability prediction

### 3. Intelligent Closing
Closes positions when:
- **Stop Loss Hit:** 0.5% loss (hard protection)
- **Take Profit Hit:** 1.5% gain (target achieved)
- **AI Predicts Loss:** Health score < 45 (high risk)
- **Combined Risk:** Loss > 0.2% + AI score < 50

## How It Works

```
1. Startup
   └─ Check all open positions
   └─ Take ownership of each
   └─ "🛡️ Took over position: BTCUSDT LONG"

2. Monitor Loop (every 30s)
   └─ Get position details
   ├─ Current price
   ├─ Entry price
   └─ Unrealized PnL
   
3. Market Analysis
   ├─ Get 5m klines (last 20 candles)
   ├─ Calculate trend direction
   ├─ Determine trend strength
   └─ "Market is BEARISH (strength: 72%)"

4. AI Assessment
   └─ Send to Gemini:
       Symbol: BTCUSDT
       Side: LONG
       Current Price: $98,000
       Entry Price: $98,100
       Price Movement: -0.1%
       Market Trend: BEARISH
       Trend Strength: 72%
       Task: "Assess if position will WIN or LOSE"
   
   └─ Receive prediction:
       Decision: SELL (bearish signal)
       Confidence: 0.75
       Win Probability: 15% (unfavorable)

5. Health Score Calculation
   └─ Base: 15% (AI prediction)
   └─ Trend Adjustment: -10% (LONG vs BEARISH)
   └─ PnL Adjustment: +5% (small profit)
   └─ Final Health: 10% (CRITICAL)

6. Closing Decision
   IF (Health < 45): CLOSE
   └─ "⚠️ Closing position: High loss probability"
   └─ "Reason: Win probability: 15%"
   
7. Execution
   └─ Place opposite market order
   └─ "✅ Position closed: +$45.20"
```

## Health Score Logic

### Base Score (AI Prediction)
```
LONG Position:
  - AI says BUY  → 70-90 health (favorable)
  - AI says SELL → 10-40 health (unfavorable)
  - AI says HOLD → 50 health (neutral)

SHORT Position:
  - AI says SELL → 70-90 health (favorable)
  - AI says BUY  → 10-40 health (unfavorable)
  - AI says HOLD → 50 health (neutral)
```

### Adjustments
```
Trend Alignment:
  +10 if position aligns with strong trend
  -10 if position opposes strong trend

PnL Impact:
  +10 if profit > 0.5%
  -10 if loss > -0.3%
  -20 if loss > -0.5%

Final Range: 0-100
```

## Closing Rules

### Priority 1: Hard Stop Loss
- Trigger: PnL < -0.5%
- Action: Immediately close
- Reason: Risk management

### Priority 2: Take Profit
- Trigger: PnL > 1.5%
- Action: Immediately close
- Reason: Target achieved

### Priority 3: AI Risk Warning
- Trigger: Health score < 45
- Action: Close position
- Reason: High probability of loss

### Priority 4: Combined Risk
- Trigger: PnL < -0.2% AND Health < 50
- Action: Close position
- Reason: Loss + unfavorable conditions

## Example Scenarios

### Scenario 1: Healthy Position
```
LONG BTCUSDT at $98,000
Current: $98,200 (+0.2%)
Trend: BULLISH (strength 80%)
AI: BUY, confidence 0.85

Health Score: 90 (excellent)
Action: Hold position ✅
```

### Scenario 2: Unfavorable Trend
```
LONG ETHUSDT at $3,200
Current: $3,190 (-0.3%)
Trend: BEARISH (strength 75%)
AI: SELL, confidence 0.72

Health Score: 25 (critical)
Action: Close position ⚠️
Reason: Win probability: 18% (trend opposes position)
```

### Scenario 3: Stop Loss Hit
```
SHORT SOLUSDT at $100
Current: $100.6 (+0.6% loss for SHORT)
PnL: -0.6%

Action: Close position 🛑
Reason: Stop loss exceeded (0.5%)
```

### Scenario 4: Early Warning
```
LONG AVAXUSDT at $35
Current: $34.9 (-0.3%)
Trend: NEUTRAL
AI: HOLD, confidence 0.50

Health Score: 40 (borderline)
PnL: -0.3%

Action: Close position ⚠️
Reason: Combined risk (loss + low AI confidence)
```

## AI Prompt for Position Assessment

```
You are a trading position risk assessor. Analyze this position:

Symbol: {symbol}
Side: {LONG|SHORT}
Entry Price: ${entry}
Current Price: ${current}
Price Movement: {movement}%
Market Trend: {BULLISH|BEARISH|NEUTRAL}
Trend Strength: {strength}%

Task: Assess if this position will WIN or LOSE in the next 10 minutes.

Provide:
1. Trade decision (BUY/SELL/HOLD)
2. Confidence (0.0-1.0)
3. Reasoning

Example response:
{
  "decision": "SELL",
  "confidence": 0.75,
  "reasoning": "Bearish trend with strong momentum, position opposes trend"
}
```

## Configuration

### Check Interval
- Default: 30 seconds
- Location: `position_manager.go:231`
- Adjustable: Modify ticker duration

### Stop Loss
- Default: 0.5% from entry
- Location: `position_manager.go:226-229`
- Adjustable: Modify multiplier

### Take Profit
- Default: 1.5% from entry
- Location: `position_manager.go:226-229`
- Adjustable: Modify multiplier

### AI Health Threshold
- Default: Close if health < 45
- Location: `position_manager.go:411`
- Adjustable: Modify threshold

## Integration

The Position Manager is automatically integrated into the Platform:

```go
type Platform struct {
    ...
    positionMgr *position.PositionManager
    ...
}
```

Startup:
1. Initialize Position Manager
2. Take over open positions
3. Start monitoring loop

Shutdown:
1. Stop monitoring loop
2. Close any remaining positions (optional)
3. Log final state

## Logs

### Position Takeover
```
{"level":"info","msg":"🛡️ Took over position","symbol":"BTCUSDT","position_amt":0.001,"entry_price":98000}
```

### Position Monitoring
```
{"level":"debug","msg":"📊 Position state","symbol":"BTCUSDT","side":"LONG","current_price":98050,"entry_price":98000,"pnl_percent":0.05,"health_score":85}
```

### Position Closing
```
{"level":"warn","msg":"⚠️ Closing position","symbol":"BTCUSDT","side":"LONG","pnl_percent":-0.3,"pnl":-12.50,"health_score":35,"reason":"Risk management triggered"}

{"level":"info","msg":"✅ Position closed","symbol":"BTCUSDT","side":"LONG","quantity":0.001,"pnl":-12.50,"pnl_percent":-0.3,"reason":"Risk management triggered"}
```

## Benefits

1. **Automatic Risk Management** - No manual monitoring needed
2. **AI-Powered Decisions** - Uses market intelligence
3. **Trend Awareness** - Considers market direction
4. **Early Warning** - Closes before large losses
5. **Profit Protection** - Takes profit when targets hit

## Files Modified

- `internal/position/position_manager.go` - New file (472 lines)
- `pkg/platform/platform.go` - Integration (3 changes)

## Testing

```bash
# Start bot with position manager
./gobot-production

# Watch position monitoring logs
tail -f startup.log | grep -E "Position|position|Closing"

# Expected output:
# 🛡️ Took over position: BTCUSDT LONG
# 📊 Position monitoring loop started (every 30s)
# 📊 Position state: BTCUSDT LONG, health: 85
# (if health < 45)
# ⚠️ Closing position: BTCUSDT
# ✅ Position closed
```

## Future Enhancements

1. **Screenshot-Based Analysis**
   - Capture chart screenshots (1m, 5m, 15m)
   - Send to Gemini Vision
   - Combine with trend analysis

2. **Machine Learning**
   - Train on historical outcomes
   - Improve prediction accuracy
   - Adaptive thresholds

3. **Multi-Timeframe Analysis**
   - Compare signals across timeframes
   - Higher confidence on confluence
   - Filter conflicting signals

4. **News Integration**
   - Check for high-impact news
   - Adjust risk based on sentiment
   - Close positions on negative news
