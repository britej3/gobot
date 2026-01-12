# GOBOT LLM Brain Workflow & SOP

## Trading Persona

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║   🎯 AGGRESSIVE TRADER | HIGH RISK | HIGH LEVERAGE | SMALL POSITION SIZE    ║
║                                                                              ║
║   "Strike fast, cut losses faster, let winners run with trailing stops"     ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

### Core Philosophy

| Principle | Implementation |
|-----------|----------------|
| **Aggressive Entry** | Enter on momentum confirmation, don't wait for perfect setup |
| **High Leverage** | 20-50x to maximize gains on small moves |
| **Small Position** | Risk only 1-2% of capital per trade (small absolute size) |
| **Quick Exits** | Trailing TP to lock profits, tight stops to limit losses |
| **High Frequency** | Multiple trades per session, compound small gains |

### Risk-Reward Profile

```
Position Size:  SMALL (1-2% risk per trade)
Leverage:       HIGH (20-50x)
Stop Loss:      TIGHT (0.3-0.5% from entry)
Take Profit:    TRAILING (activate at 0.3%, trail 0.15%)
Win Rate Target: 55-65% (edge comes from R:R ratio)
```

---

## LLM Decision Framework

### NOT a Fixed Process

The LLM brain operates within defined **boundaries** but has **flexibility** in:
- Which signals to prioritize
- How to interpret market context
- When to be more/less aggressive
- Which exit strategy to use

### Decision Boundaries

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        LLM DECISION BOUNDARIES                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  FIXED (Cannot Override):                                                   │
│  ├── First trade = 1 USDT target (session validation)                      │
│  ├── Maximum leverage = 50x                                                 │
│  ├── Maximum position = 2% of balance at risk                               │
│  ├── Circuit breakers (daily loss, consecutive losses)                      │
│  └── Liquidation distance minimum = 5%                                      │
│                                                                             │
│  FLEXIBLE (LLM Decides):                                                    │
│  ├── Entry timing and price                                                 │
│  ├── Exact leverage within range (20-50x)                                   │
│  ├── Position size within limits                                            │
│  ├── Long vs Short direction                                                │
│  ├── Exit strategy (fixed TP vs trailing)                                   │
│  ├── Hold duration                                                          │
│  └── Skip trade if conditions unfavorable                                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Workflow Phases

### Phase 1: Market Scanning (Every 10 seconds)

```
┌─────────────────┐
│  FETCH DATA     │
├─────────────────┤
│ • Top Movers    │
│ • 24hr Tickers  │
│ • Order Books   │
│ • Funding Rates │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  PARSE & VALIDATE│
├─────────────────┤
│ • Type parsing  │
│ • Range checks  │
│ • Anomaly detect│
│ • Data freshness│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  CATEGORIZE     │
├─────────────────┤
│ • Small Rise/Fall (3-7%)   │
│ • Mid Rise/Fall (7-11%)    │
│ • High Rise/Fall (>11%)    │
│ • Price + High Volume      │
│ • Pullback                 │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  MOMENTUM SCORE │
├─────────────────┤
│ • RSI weight: 25%          │
│ • MACD weight: 25%         │
│ • Volume weight: 30%       │
│ • Trend weight: 20%        │
└────────┬────────┘
         │
         ▼
    TOP 5 ASSETS
```

### Phase 2: LLM Analysis (Per Opportunity)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                           LLM ANALYSIS PROMPT                                │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  CONTEXT:                                                                    │
│  You are an AGGRESSIVE SCALPER with HIGH RISK tolerance.                    │
│  You use HIGH LEVERAGE (20-50x) with SMALL POSITION SIZES.                  │
│  Your goal is to capture quick momentum moves on Binance Futures Top Movers.│
│                                                                              │
│  CURRENT MARKET DATA:                                                        │
│  - Symbol: {{symbol}}                                                        │
│  - Category: {{category}} (e.g., MID_5MIN_RISE)                              │
│  - Price: ${{price}} ({{change_pct}}% 24h)                                   │
│  - Volume: ${{volume_24h}} ({{volume_mult}}x average)                        │
│  - Spread: {{spread}}%                                                       │
│  - RSI(6): {{rsi}}                                                           │
│  - MACD: {{macd_signal}}                                                     │
│  - Funding Rate: {{funding}}%                                                │
│                                                                              │
│  ACCOUNT STATE:                                                              │
│  - Balance: ${{balance}}                                                     │
│  - Available: ${{available}}                                                 │
│  - Today's PnL: ${{daily_pnl}} ({{daily_pnl_pct}}%)                          │
│  - Open Positions: {{open_positions}}                                        │
│  - Session Trades: {{session_trades}} (W:{{wins}} L:{{losses}})              │
│                                                                              │
│  MEMORY CONTEXT:                                                             │
│  {{memory_similar_trades}}                                                   │
│  {{memory_market_patterns}}                                                  │
│                                                                              │
│  CONSTRAINTS:                                                                │
│  - Max leverage: 50x                                                         │
│  - Max risk per trade: 2% of balance                                         │
│  - Is first trade of session: {{is_first_trade}}                             │
│  - If first trade: Target exactly 1 USDT profit                              │
│                                                                              │
│  DECISION REQUIRED:                                                          │
│  Analyze this opportunity as an aggressive scalper. Respond with:            │
│                                                                              │
│  {                                                                           │
│    "action": "LONG" | "SHORT" | "SKIP",                                      │
│    "confidence": 0.0-1.0,                                                    │
│    "leverage": 20-50,                                                        │
│    "position_size_pct": 0.5-2.0,                                             │
│    "entry_type": "MARKET" | "LIMIT",                                         │
│    "entry_price": null or limit price,                                       │
│    "stop_loss_pct": 0.2-1.0,                                                 │
│    "take_profit_strategy": "FIXED" | "TRAILING",                             │
│    "take_profit_pct": 0.3-2.0,                                               │
│    "trailing_activation_pct": 0.2-0.5,                                       │
│    "trailing_distance_pct": 0.1-0.3,                                         │
│    "max_hold_minutes": 1-30,                                                 │
│    "reasoning": "Brief explanation",                                         │
│    "risk_notes": "Any concerns"                                              │
│  }                                                                           │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Phase 3: Decision Validation

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        DECISION VALIDATION                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. PARSE LLM RESPONSE                                                      │
│     └── Validate JSON structure                                             │
│     └── Check all required fields present                                   │
│     └── Fallback to SKIP if parse fails                                     │
│                                                                             │
│  2. ENFORCE HARD LIMITS                                                     │
│     └── Clamp leverage to 50x max                                           │
│     └── Clamp position size to 2% max                                       │
│     └── Ensure liquidation distance >= 5%                                   │
│     └── Override if first trade (use 1 USDT rule)                           │
│                                                                             │
│  3. CHECK CIRCUIT BREAKERS                                                  │
│     └── Daily loss limit not exceeded                                       │
│     └── Consecutive losses < 5                                              │
│     └── Session trade limit not exceeded                                    │
│                                                                             │
│  4. LIQUIDITY CHECK                                                         │
│     └── Spread < 0.1%                                                       │
│     └── Order book depth sufficient                                         │
│     └── Slippage estimation acceptable                                      │
│                                                                             │
│  5. FUNDING RATE CHECK                                                      │
│     └── Don't LONG if funding > +0.1%                                       │
│     └── Don't SHORT if funding < -0.1%                                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Phase 4: Trade Execution

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  PRE-EXECUTION  │────▶│   EXECUTION     │────▶│ POST-EXECUTION  │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ • Log to WAL    │     │ • Set leverage  │     │ • Confirm fill  │
│ • Calculate size│     │ • Place order   │     │ • Set SL/TP     │
│ • Add jitter    │     │ • Stealth mode  │     │ • Start monitor │
│ • Size obfuscate│     │ • IOC if needed │     │ • Log to memory │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### Phase 5: Position Management

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                        POSITION MONITORING LOOP                              │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  EVERY 100ms:                                                                │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │                                                                        │  │
│  │  1. CHECK STOP LOSS                                                    │  │
│  │     └── If mark price hits SL → CLOSE IMMEDIATELY                      │  │
│  │                                                                        │  │
│  │  2. CHECK TAKE PROFIT                                                  │  │
│  │     └── If FIXED: Close at target                                      │  │
│  │     └── If TRAILING:                                                   │  │
│  │         ├── Track highest PnL                                          │  │
│  │         ├── If PnL > activation → enable trailing                      │  │
│  │         ├── Move stop up as price moves in favor                       │  │
│  │         └── Close when price retraces past trailing stop               │  │
│  │                                                                        │  │
│  │  3. CHECK TIME STOP                                                    │  │
│  │     └── If held > max_hold_minutes → Consider exit                     │  │
│  │                                                                        │  │
│  │  4. CHECK LIQUIDATION DISTANCE                                         │  │
│  │     └── If < 3% → EMERGENCY REDUCE                                     │  │
│  │                                                                        │  │
│  │  5. LLM RE-EVALUATION (every 30 seconds)                               │  │
│  │     └── Ask LLM: "Hold, add, or exit?"                                 │  │
│  │     └── LLM can suggest early exit or position adjustment              │  │
│  │                                                                        │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Phase 6: Trade Completion

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  CLOSE TRADE    │────▶│  RECORD OUTCOME │────▶│ LEARN & ADAPT   │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ • Execute close │     │ • Calculate PnL │     │ • Store in memory│
│ • Log to WAL    │     │ • Record fees   │     │ • Update stats  │
│ • Update state  │     │ • Log reason    │     │ • Adjust params │
│ • Clear monitors│     │ • TUI update    │     │ • If first trade:│
│                 │     │                 │     │   enable LLM mode│
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

---

## LLM Prompt Templates

### Entry Decision Prompt

```
PERSONA: Aggressive Scalper | High Leverage | Small Size

MARKET: {{symbol}} is showing {{category}} signal
- Price: ${{price}} ({{change}}% in {{timeframe}})
- Volume: {{volume_mult}}x average
- Momentum: RSI={{rsi}}, MACD={{macd}}

ACCOUNT: ${{balance}} ({{daily_pnl_pct}}% today)

As an aggressive scalper, should I take this trade?
Consider: momentum strength, entry timing, optimal leverage (20-50x)
```

### Exit Decision Prompt

```
POSITION: {{side}} {{symbol}} @ ${{entry}} (current: ${{mark}})
- Unrealized PnL: {{pnl_pct}}%
- Duration: {{duration}}
- Trailing active: {{trailing_active}}

Should I:
1. HOLD - Let it run with trailing stop
2. CLOSE - Take profit now
3. ADJUST - Move stops or reduce size

Consider: momentum continuation, exhaustion signals, time in trade
```

### Skip Trade Reasoning

The LLM should SKIP a trade when:
- Confidence < 50%
- Spread too wide (> 0.1%)
- Volume declining
- Counter-trend setup without strong reversal signal
- Too many open positions
- Daily loss limit approaching
- Recent consecutive losses
- Funding rate unfavorable

---

## Strategy Options (LLM Flexible)

The LLM can choose from these strategies based on market conditions:

### Strategy 1: Momentum Scalp
```
Entry:     Strong momentum in Top Mover direction
Leverage:  35-50x
Target:    0.3-0.5% (fast exit)
Stop:      0.2%
Duration:  30s - 3min
Best for:  High 5min Rise/Fall categories
```

### Strategy 2: Pullback Entry
```
Entry:     Wait for 30-50% pullback in trending move
Leverage:  25-35x
Target:    0.5-1.0% trailing
Stop:      0.3%
Duration:  2-10min
Best for:  Pullback category, continuation plays
```

### Strategy 3: Volume Spike
```
Entry:     Price + High Volume signal
Leverage:  40-50x
Target:    0.3% quick scalp
Stop:      0.15%
Duration:  15s - 1min
Best for:  [Mid/High] Price Up/Down with High Vol
```

### Strategy 4: Counter-Trend (High Risk)
```
Entry:     Exhaustion at extreme, reversal confirmation
Leverage:  20-25x (lower due to risk)
Target:    1-2% (larger target for reversal)
Stop:      0.5%
Duration:  5-15min
Best for:  New 24h High/Low with exhaustion signals
```

---

## Memory Integration

### What to Remember

```go
type TradeMemory struct {
    // Trade details
    Symbol, Side, EntryPrice, ExitPrice
    PnL, PnLPercent
    Leverage, PositionSize
    
    // Context
    Category (Top Mover type)
    Indicators (RSI, MACD, Volume)
    MarketCondition
    
    // Outcome analysis
    WhatWorked
    WhatFailed
    LessonLearned
}
```

### Memory Query Before Trade

```
Query: "What were outcomes of previous {{side}} trades on {{symbol}}?"
       "What worked in similar {{category}} setups?"
       "Any warnings about trading {{symbol}}?"
```

### Memory Store After Trade

```
If PnL > 0: Store what indicators/conditions led to win
If PnL < 0: Store what went wrong, why the loss occurred
Store lesson learned for future reference
```

---

## Error Handling & Recovery

### LLM Response Errors

```
If LLM response invalid:
  └── Retry with simpler prompt (max 2 retries)
  └── If still fails: SKIP trade (don't guess)
  └── Log error for debugging
```

### API Errors

```
If Binance API fails:
  └── Check if order went through (don't duplicate)
  └── Retry with exponential backoff
  └── If critical: halt trading, alert user
```

### Position Sync Errors

```
If position mismatch detected:
  └── Fetch actual positions from Binance
  └── Reconcile with local state
  └── Log discrepancy
  └── If orphan position: adopt and manage it
```

---

## Performance Metrics

### What to Track

```
- Win rate (target: 55-65%)
- Average win size
- Average loss size
- Profit factor (wins/losses)
- Sharpe ratio
- Max drawdown
- Average trade duration
- Trades per hour
- Slippage analysis
- Fee analysis
```

### Adaptation Triggers

```
If win rate < 50% over 20 trades:
  └── LLM: Increase confidence threshold
  └── LLM: Be more selective

If avg loss > avg win:
  └── LLM: Tighten stops
  └── LLM: Consider earlier exits

If slippage > 0.1%:
  └── System: Reduce position size
  └── System: Use limit orders more
```

---

## Session Lifecycle

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SESSION LIFECYCLE                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  SESSION START                                                              │
│  ├── Run health checks                                                      │
│  ├── Fetch account balance                                                  │
│  ├── Set daily_start_balance                                                │
│  ├── Set is_first_trade = true                                              │
│  └── Display startup banner                                                 │
│                                                                             │
│  FIRST TRADE                                                                │
│  ├── Use 1 USDT profit target (FIXED, not LLM)                              │
│  ├── Conservative leverage (10-15x)                                         │
│  ├── Validates market behavior and bot execution                            │
│  └── On completion: is_first_trade = false, enable full LLM mode            │
│                                                                             │
│  ACTIVE TRADING                                                             │
│  ├── LLM brain makes all decisions within boundaries                        │
│  ├── Continuous monitoring and adaptation                                   │
│  ├── Memory accumulation                                                    │
│  └── TUI updates in real-time                                               │
│                                                                             │
│  SESSION END (any of these)                                                 │
│  ├── 4-hour duration reached → 30 min pause                                 │
│  ├── Daily profit target (10%) reached → celebrate, pause                   │
│  ├── Daily loss limit (5%) reached → halt for day                           │
│  ├── 5 consecutive losses → pause, require confirmation                     │
│  ├── Manual stop command                                                    │
│  └── Critical error                                                         │
│                                                                             │
│  SESSION RESET                                                              │
│  ├── After 30 min pause OR after 4 hours of no trades                       │
│  ├── is_first_trade = true again                                            │
│  └── Fresh validation before aggressive trading                             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Summary: LLM Operating Principles

1. **Be Aggressive**: You are a high-risk scalper. Enter on momentum, not perfection.

2. **High Leverage, Small Size**: Use 20-50x leverage but risk only 1-2% per trade.

3. **Fast Decisions**: Markets move fast. Analyze and decide in <2 seconds.

4. **Cut Losses Quick**: 0.2-0.5% stop losses. No hoping, no averaging down.

5. **Let Winners Run**: Use trailing stops. Don't exit winners too early.

6. **Learn From Memory**: Query past trades. Avoid repeated mistakes.

7. **Respect Boundaries**: Never exceed hard limits. Skip if uncertain.

8. **First Trade = Validation**: Always start session with 1 USDT target trade.

9. **Adapt Constantly**: Adjust based on what's working today, not yesterday.

10. **When in Doubt, Skip**: There's always another trade. Preserve capital.
