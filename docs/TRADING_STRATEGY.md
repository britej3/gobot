# GOBOT Trading Strategy

## Core Philosophy

**Conservative First, Aggressive Later**

The bot starts each session with a fixed 1 USDT PnL target on the first trade. This validates market conditions and bot behavior before scaling up. Subsequent trades are governed by the LLM brain based on market conditions, momentum, and account performance.

---

## Asset Selection: Binance Futures Perpetual Top Movers

### Binance Futures Top Movers Categories

Binance provides a built-in **Top Movers** system for USDⓈ-M Perpetual Futures with the following official categories:

| Status | Period | Market Condition |
|--------|--------|------------------|
| **New 24hr High** | 1 Day | Highest price in last 1 min = highest of day |
| **New 7day High** | 1 Week | Highest price in last 1 min = highest of week |
| **New 30day High** | 1 Month | Highest price in last 1 min = highest of month |
| **New 24hr Low** | 1 Day | Lowest price in last 1 min = lowest of day |
| **New 7day Low** | 1 Week | Lowest price in last 1 min = lowest of week |
| **New 30day Low** | 1 Month | Lowest price in last 1 min = lowest of month |
| **[Small] 5min Rise** | 5 min | 3% ≤ price increase < 7% |
| **[Small] 2hr Rise** | 2 hours | 3% ≤ price increase < 7% |
| **[Mid] 5min Rise** | 5 min | 7% ≤ price increase < 11% |
| **[Mid] 2hr Rise** | 2 hours | 7% ≤ price increase < 11% |
| **[High] 5min Rise** | 5 min | Price increase ≥ 11% |
| **[High] 2hr Rise** | 2 hours | Price increase ≥ 11% |
| **[Small] 5min Fall** | 5 min | 3% ≤ price decrease < 7% |
| **[Mid] 5min Fall** | 5 min | 7% ≤ price decrease < 11% |
| **[High] 5min Fall** | 5 min | Price decrease ≥ 11% |
| **Pullback** | Daily | (High - Open) / Open ≥ 8% |
| **[Small] Price Up + High Vol** | 15 min | 7-11% rise + Vol ≥ 50x avg |
| **[Mid] Price Up + High Vol** | 15 min | 11-15% rise + Vol ≥ 50x avg |
| **[High] Price Up + High Vol** | 15 min | ≥15% rise + Vol ≥ 50x avg |
| **[Small] Price Down + High Vol** | 15 min | 7-11% drop + Vol ≥ 50x avg |
| **[Mid] Price Down + High Vol** | 15 min | 11-15% drop + Vol ≥ 50x avg |
| **[High] Price Down + High Vol** | 15 min | ≥15% drop + Vol ≥ 50x avg |

### API Endpoints

#### Primary: 24hr Ticker Statistics
```
GET https://fapi.binance.com/fapi/v1/ticker/24hr
```

**Response Fields for Top Mover Detection:**
```json
{
  "symbol": "BTCUSDT",
  "priceChange": "-94.99999800",
  "priceChangePercent": "-95.960",    // Key: 24h % change
  "weightedAvgPrice": "0.29628482",
  "lastPrice": "4.00000200",
  "highPrice": "100.00000000",        // Key: 24h high
  "lowPrice": "0.10000000",           // Key: 24h low
  "volume": "8913.30000000",          // Key: 24h volume (base)
  "quoteVolume": "15.30000000",       // Key: 24h volume (quote/USDT)
  "openTime": 1499783499040,
  "closeTime": 1499869899040,
  "count": 76                         // Trade count
}
```

#### Secondary: Klines for Short-Term Moves
```
GET https://fapi.binance.com/fapi/v1/klines?symbol=BTCUSDT&interval=5m&limit=1
```

### Implementation: Top Movers Screener

```go
// TopMoverCategory defines Binance's official categories
type TopMoverCategory string

const (
    CategoryNew24hHigh      TopMoverCategory = "NEW_24H_HIGH"
    CategoryNew7dHigh       TopMoverCategory = "NEW_7D_HIGH"
    CategoryNew30dHigh      TopMoverCategory = "NEW_30D_HIGH"
    CategoryNew24hLow       TopMoverCategory = "NEW_24H_LOW"
    CategorySmall5minRise   TopMoverCategory = "SMALL_5MIN_RISE"   // 3-7%
    CategoryMid5minRise     TopMoverCategory = "MID_5MIN_RISE"     // 7-11%
    CategoryHigh5minRise    TopMoverCategory = "HIGH_5MIN_RISE"    // >11%
    CategorySmall5minFall   TopMoverCategory = "SMALL_5MIN_FALL"   // 3-7%
    CategoryMid5minFall     TopMoverCategory = "MID_5MIN_FALL"     // 7-11%
    CategoryHigh5minFall    TopMoverCategory = "HIGH_5MIN_FALL"    // >11%
    CategoryPullback        TopMoverCategory = "PULLBACK"          // 8%+ then retrace
    CategoryPriceUpHighVol  TopMoverCategory = "PRICE_UP_HIGH_VOL" // Rise + 50x volume
    CategoryPriceDownHighVol TopMoverCategory = "PRICE_DOWN_HIGH_VOL"
)

// TopMover represents a detected top mover
type TopMover struct {
    Symbol              string
    Category            TopMoverCategory
    PriceChangePercent  float64
    Volume24h           float64
    QuoteVolume24h      float64
    HighPrice           float64
    LowPrice            float64
    LastPrice           float64
    VolumeMultiplier    float64  // vs average
    DetectedAt          time.Time
}

// TopMoversScreener fetches and categorizes top movers
type TopMoversScreener struct {
    client          *futures.Client
    refreshInterval time.Duration  // 10 seconds per Binance spec
}

// FetchTopMovers gets all USDⓈ-M perpetual futures tickers and categorizes
func (s *TopMoversScreener) FetchTopMovers(ctx context.Context) ([]TopMover, error) {
    // Fetch all 24hr tickers (weight: 40)
    tickers, err := s.client.NewListPriceChangeStatsService().Do(ctx)
    if err != nil {
        return nil, err
    }
    
    var movers []TopMover
    
    for _, t := range tickers {
        priceChange, _ := strconv.ParseFloat(t.PriceChangePercent, 64)
        volume, _ := strconv.ParseFloat(t.QuoteVolume, 64)
        
        // Skip low volume (< $10M)
        if volume < 10_000_000 {
            continue
        }
        
        // Categorize based on Binance's official thresholds
        absPriceChange := math.Abs(priceChange)
        
        var category TopMoverCategory
        switch {
        case priceChange >= 11:
            category = CategoryHigh5minRise
        case priceChange >= 7:
            category = CategoryMid5minRise
        case priceChange >= 3:
            category = CategorySmall5minRise
        case priceChange <= -11:
            category = CategoryHigh5minFall
        case priceChange <= -7:
            category = CategoryMid5minFall
        case priceChange <= -3:
            category = CategorySmall5minFall
        default:
            continue // Not a top mover
        }
        
        movers = append(movers, TopMover{
            Symbol:             t.Symbol,
            Category:           category,
            PriceChangePercent: priceChange,
            Volume24h:          volume,
            LastPrice:          parseFloat(t.LastPrice),
            HighPrice:          parseFloat(t.HighPrice),
            LowPrice:           parseFloat(t.LowPrice),
            DetectedAt:         time.Now(),
        })
    }
    
    // Sort by absolute price change (most volatile first)
    sort.Slice(movers, func(i, j int) bool {
        return math.Abs(movers[i].PriceChangePercent) > math.Abs(movers[j].PriceChangePercent)
    })
    
    return movers, nil
}

// FilterByCategory returns movers matching specific categories
func (s *TopMoversScreener) FilterByCategory(movers []TopMover, categories ...TopMoverCategory) []TopMover {
    categorySet := make(map[TopMoverCategory]bool)
    for _, c := range categories {
        categorySet[c] = true
    }
    
    var filtered []TopMover
    for _, m := range movers {
        if categorySet[m.Category] {
            filtered = append(filtered, m)
        }
    }
    return filtered
}
```

### Selection Criteria for Trading

```go
type TopMoverCriteria struct {
    // Categories to trade (Binance official categories)
    AllowedCategories   []TopMoverCategory
    
    // Volatility filters (using Binance's thresholds)
    MinPriceChange      float64 // 3% minimum (Small category)
    MaxPriceChange      float64 // 15% maximum (avoid extreme)
    
    // Volume filters
    MinQuoteVolume24h   float64 // $10M minimum
    MinVolumeMultiplier float64 // 1.5x average
    
    // Liquidity filters
    MaxSpread           float64 // < 0.1%
    
    // Preferred categories for scalping
    PreferredCategories []TopMoverCategory
}

// Default criteria for GOBOT scalping
var DefaultScalpingCriteria = TopMoverCriteria{
    AllowedCategories: []TopMoverCategory{
        CategorySmall5minRise,    // 3-7% rise - good for longs
        CategoryMid5minRise,      // 7-11% rise - momentum
        CategorySmall5minFall,    // 3-7% fall - good for shorts
        CategoryMid5minFall,      // 7-11% fall - momentum
        CategoryPriceUpHighVol,   // High volume moves
        CategoryPriceDownHighVol,
    },
    MinPriceChange:      3.0,
    MaxPriceChange:      15.0,
    MinQuoteVolume24h:   10_000_000,
    MinVolumeMultiplier: 1.5,
    MaxSpread:           0.1,
    PreferredCategories: []TopMoverCategory{
        CategorySmall5minRise,
        CategorySmall5minFall,
    },
}
```

### Screening Process

1. **Fetch Tickers**: `GET /fapi/v1/ticker/24hr` (all USDⓈ-M perpetuals)
2. **Categorize**: Apply Binance's official thresholds
3. **Filter Volume**: > $10M 24h quote volume
4. **Filter Spread**: Check order book, < 0.1% spread
5. **Score Momentum**: RSI + MACD + Volume composite
6. **Rank and Select**: Top 5 assets by momentum score

---

## Position Sizing: Smart Dynamic

### Account-Based Sizing

```go
type DynamicPositionSizer struct {
    AccountBalance     float64
    MaxRiskPerTrade    float64  // 1-2% of balance
    MaxTotalExposure   float64  // 10% of balance max open
    CurrentExposure    float64
    ConsecutiveWins    int
    ConsecutiveLosses  int
}

func (d *DynamicPositionSizer) CalculateSize(asset *Asset, confidence float64) float64 {
    // Base risk: 1% of account
    baseRisk := d.AccountBalance * 0.01
    
    // Adjust for confidence (0.5 to 2.0 multiplier)
    confidenceMultiplier := 0.5 + (confidence * 1.5)
    
    // Adjust for win/loss streak
    streakMultiplier := 1.0
    if d.ConsecutiveWins >= 3 {
        streakMultiplier = 1.2  // Increase after wins
    }
    if d.ConsecutiveLosses >= 2 {
        streakMultiplier = 0.5  // Decrease after losses
    }
    
    // Adjust for volatility (lower size for higher vol)
    volMultiplier := 1.0 / (1.0 + asset.ATRPercent)
    
    // Adjust for remaining exposure capacity
    remainingCapacity := d.MaxTotalExposure - d.CurrentExposure
    capacityMultiplier := min(1.0, remainingCapacity / baseRisk)
    
    return baseRisk * confidenceMultiplier * streakMultiplier * volMultiplier * capacityMultiplier
}
```

---

## Dynamic Leverage

### Leverage Calculation

```go
type DynamicLeverage struct {
    MinLeverage      int     // 5x minimum
    MaxLeverage      int     // 50x maximum
    VolatilityFactor float64 // Higher vol = lower leverage
    AccountHealth    float64 // Margin ratio consideration
}

func (d *DynamicLeverage) Calculate(asset *Asset, confidence float64) int {
    // Base leverage from confidence (10x at 50% confidence, 25x at 75%)
    baseLeverage := int(confidence * 40)
    
    // Adjust for volatility (inverse relationship)
    // High volatility (>5%) = reduce leverage
    volAdjust := 1.0 - (asset.ATRPercent / 10.0)
    if volAdjust < 0.3 {
        volAdjust = 0.3  // Floor at 30%
    }
    
    // Adjust for account health
    healthAdjust := 1.0
    if d.AccountHealth < 0.5 {
        healthAdjust = 0.5  // Reduce if margin stressed
    }
    
    // Calculate final leverage
    leverage := int(float64(baseLeverage) * volAdjust * healthAdjust)
    
    // Clamp to range
    if leverage < d.MinLeverage {
        leverage = d.MinLeverage
    }
    if leverage > d.MaxLeverage {
        leverage = d.MaxLeverage
    }
    
    return leverage
}
```

### Leverage Rules

| Account Health | Max Volatility | Max Leverage |
|---------------|----------------|--------------|
| > 80% | < 3% | 50x |
| > 80% | 3-5% | 35x |
| > 80% | > 5% | 20x |
| 50-80% | Any | 15x |
| < 50% | Any | 5x |

---

## Trailing Take Profit

### Strategy

```go
type TrailingTakeProfit struct {
    InitialTPPercent   float64 // Initial TP target (e.g., 0.5%)
    ActivationPercent  float64 // When to activate trailing (e.g., 0.3%)
    TrailingDistance   float64 // Distance to trail (e.g., 0.15%)
    MinProfit          float64 // Minimum profit to lock (e.g., 0.1%)
    
    // State
    HighestPnL         float64
    TrailingActive     bool
    TrailingStopPrice  float64
}

func (t *TrailingTakeProfit) Update(currentPnLPercent float64, currentPrice float64, side string) (shouldClose bool, reason string) {
    // Track highest PnL
    if currentPnLPercent > t.HighestPnL {
        t.HighestPnL = currentPnLPercent
    }
    
    // Activate trailing once we hit activation threshold
    if !t.TrailingActive && currentPnLPercent >= t.ActivationPercent {
        t.TrailingActive = true
        t.updateTrailingStop(currentPrice, side)
    }
    
    // Update trailing stop if price moves in our favor
    if t.TrailingActive {
        t.updateTrailingStop(currentPrice, side)
        
        // Check if trailing stop hit
        if t.isTrailingStopHit(currentPrice, side) {
            return true, fmt.Sprintf("Trailing TP hit at %.2f%% (high was %.2f%%)", currentPnLPercent, t.HighestPnL)
        }
    }
    
    // Check initial TP target
    if currentPnLPercent >= t.InitialTPPercent {
        return true, fmt.Sprintf("Initial TP target hit: %.2f%%", currentPnLPercent)
    }
    
    return false, ""
}

func (t *TrailingTakeProfit) updateTrailingStop(currentPrice float64, side string) {
    if side == "LONG" {
        newStop := currentPrice * (1 - t.TrailingDistance/100)
        if newStop > t.TrailingStopPrice {
            t.TrailingStopPrice = newStop
        }
    } else { // SHORT
        newStop := currentPrice * (1 + t.TrailingDistance/100)
        if t.TrailingStopPrice == 0 || newStop < t.TrailingStopPrice {
            t.TrailingStopPrice = newStop
        }
    }
}
```

### Trailing TP Settings by Market Condition

| Volatility | Activation | Trail Distance | Initial TP |
|------------|------------|----------------|------------|
| Low (<2%) | 0.2% | 0.1% | 0.4% |
| Medium (2-5%) | 0.4% | 0.2% | 0.8% |
| High (>5%) | 0.6% | 0.3% | 1.2% |

---

## First Trade Rule: 1 USDT PnL Target

### Philosophy

The first trade of each session uses a **fixed 1 USDT profit target** regardless of account size or market conditions. This serves as:

1. **Market Validation**: Confirms the market is behaving as expected
2. **System Check**: Validates bot execution is working correctly
3. **Psychology Reset**: Starts with a guaranteed small win
4. **Risk Limiter**: Prevents over-aggressive first trade

### Implementation

```go
type FirstTradeRule struct {
    TargetPnL          float64  // 1.0 USDT
    IsFirstTrade       bool
    SessionStartTime   time.Time
    SessionResetHours  int      // Reset after 4 hours of no trades
}

func (f *FirstTradeRule) CalculatePosition(entryPrice float64, leverage int, side string) (size float64, tpPrice float64, slPrice float64) {
    if !f.IsFirstTrade {
        return 0, 0, 0  // Let normal logic handle
    }
    
    // Calculate position size for 1 USDT profit at 0.5% move
    targetMovePercent := 0.5
    
    // PnL = size * leverage * movePercent
    // 1 = size * leverage * 0.005
    // size = 1 / (leverage * 0.005)
    size = f.TargetPnL / (float64(leverage) * targetMovePercent / 100)
    
    // Set tight TP and SL
    if side == "LONG" {
        tpPrice = entryPrice * (1 + targetMovePercent/100)
        slPrice = entryPrice * (1 - targetMovePercent/100)  // 1:1 RR for first trade
    } else {
        tpPrice = entryPrice * (1 - targetMovePercent/100)
        slPrice = entryPrice * (1 + targetMovePercent/100)
    }
    
    return size, tpPrice, slPrice
}

func (f *FirstTradeRule) OnTradeComplete(outcome string) {
    f.IsFirstTrade = false
    // Now subsequent trades use LLM brain logic
}

func (f *FirstTradeRule) ShouldResetSession() bool {
    return time.Since(f.SessionStartTime) > time.Duration(f.SessionResetHours) * time.Hour
}
```

### First Trade vs Subsequent Trades

| Aspect | First Trade | Subsequent Trades |
|--------|-------------|-------------------|
| **PnL Target** | Fixed 1 USDT | LLM brain decides |
| **Position Size** | Calculated for 1 USDT | Dynamic based on balance |
| **Leverage** | Conservative (10-15x) | Dynamic (5-50x) |
| **TP Strategy** | Fixed at 0.5% move | Trailing TP |
| **SL Strategy** | Fixed 1:1 R:R | Dynamic based on ATR |
| **Asset Selection** | Any top mover | Momentum-filtered |

---

## Trade Flow Sequence

```
┌─────────────────────────────────────────────────────────────────┐
│                        GOBOT TRADE FLOW                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Fetch Top Movers │
                    │  (Binance API)   │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Filter by:       │
                    │ - Volatility     │
                    │ - Volume         │
                    │ - Momentum       │
                    │ - Liquidity      │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Is First Trade?  │
                    └────────┬─────────┘
                              │
               ┌──────────────┴──────────────┐
               │ YES                         │ NO
               ▼                             ▼
    ┌────────────────────┐        ┌────────────────────┐
    │ FIRST TRADE RULE   │        │ LLM BRAIN DECIDES  │
    │ - Target: 1 USDT   │        │ - Query memory     │
    │ - Fixed size       │        │ - Analyze momentum │
    │ - 0.5% TP/SL       │        │ - Dynamic size     │
    │ - 10-15x leverage  │        │ - Dynamic leverage │
    └─────────┬──────────┘        └─────────┬──────────┘
               │                             │
               └──────────────┬──────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Execute Trade    │
                    │ (Stealth mode)   │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Monitor Position │
                    │ - Trailing TP    │
                    │ - Dynamic SL     │
                    │ - Time stop      │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Close & Record   │
                    │ - Store in memory│
                    │ - Update stats   │
                    │ - Log outcome    │
                    └──────────────────┘
```

---

## Session Management

### Session Rules

1. **Session Start**: Reset `IsFirstTrade = true`
2. **Session Duration**: 4 hours max, then pause 30 minutes
3. **Daily Limit**: Max 50 trades per day
4. **Loss Limit**: Stop trading after 5% daily drawdown
5. **Win Streak**: After 5 consecutive wins, reduce size by 20% (avoid overconfidence)

### Session Reset Triggers

- 4 hours of continuous trading
- 30 minutes of no opportunities
- Circuit breaker activation
- Manual pause command
- Daily profit target reached (10%)

---

## Example Trade Scenarios

### Scenario 1: First Trade of Session

```
Account: $1000
Asset: BTCUSDT (Top mover +4.2%)
Entry: $98,500
Leverage: 12x (conservative for first trade)
Target PnL: 1 USDT
Position Size: 1 / (12 * 0.005) = 16.67 USDT notional
TP: $98,992.50 (+0.5%)
SL: $98,007.50 (-0.5%)

Result: TP hit → +1 USDT → First trade complete → Enable LLM brain
```

### Scenario 2: Subsequent Trade (LLM Brain)

```
Account: $1001
Asset: ETHUSDT (Momentum: RSI 65, MACD bullish, Volume +180%)
Entry: $3,850
Confidence: 72%
Leverage: calculate(72%, ATR=2.8%) = 28x
Position Size: dynamic($1001, 72%, streak=1 win) = $15 risk = 389 USDT notional
TP: Trailing (activate at 0.4%, trail 0.2%)
SL: ATR-based = 1.5 * ATR = $3,785

Result: Trailing TP locks profit at 1.1% → +12.08 USDT
```

---

## Configuration

```go
// config/trading_strategy.go
type TradingStrategyConfig struct {
    // Screening
    TopMoversCount        int     `json:"top_movers_count" default:"50"`
    MinPriceChange24h     float64 `json:"min_price_change_24h" default:"3.0"`
    MaxPriceChange24h     float64 `json:"max_price_change_24h" default:"15.0"`
    MinVolume24h          float64 `json:"min_volume_24h" default:"10000000"`
    
    // Position Sizing
    MaxRiskPerTrade       float64 `json:"max_risk_per_trade" default:"0.01"`
    MaxTotalExposure      float64 `json:"max_total_exposure" default:"0.10"`
    
    // Leverage
    MinLeverage           int     `json:"min_leverage" default:"5"`
    MaxLeverage           int     `json:"max_leverage" default:"50"`
    
    // Trailing TP
    TrailingActivation    float64 `json:"trailing_activation" default:"0.3"`
    TrailingDistance      float64 `json:"trailing_distance" default:"0.15"`
    InitialTPPercent      float64 `json:"initial_tp_percent" default:"0.5"`
    
    // First Trade Rule
    FirstTradePnL         float64 `json:"first_trade_pnl" default:"1.0"`
    FirstTradeTPPercent   float64 `json:"first_trade_tp_percent" default:"0.5"`
    SessionResetHours     int     `json:"session_reset_hours" default:"4"`
}
```
