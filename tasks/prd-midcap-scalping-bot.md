# PRD: GOBOT Mid-Cap Scalping Bot Transformation

## Executive Summary

Transform GOBOT/Cognee from a partially-implemented test system into a production-ready, cost-aware, self-improving mid-cap scalping bot for Binance Futures with 20-50x leverage.

---

## Current State Analysis

### ✅ Working Components

| Component | Status | Notes |
|-----------|--------|-------|
| Brain Engine | Partial | Advanced LLM integration with recovery mechanisms |
| Market Data | Working | WebSocket streaming with real-time price feeds |
| WAL System | Working | Transaction logging with size-based rotation |
| Platform Architecture | Working | Modular design with component separation |
| Backtester | Working | Strategy testing capability |
| State Manager | Partial | State persistence (needs cleanup) |

### ❌ Critical Issues (Verified via Build)

| Issue | File | Error |
|-------|------|-------|
| Invalid Order Types | `internal/striker/striker.go:298,318` | `futures.OrderTypeStopMarket` and `OrderTypeTakeProfitMarket` undefined |
| Missing Type Definition | `internal/agent/reconciler.go:102,262` | `platform.PositionState` undefined |
| Non-existent API | `internal/agent/reconciler.go:291` | `NewMarkPriceService` doesn't exist in futures client |
| Unused Imports | `internal/auditor/auditor.go` | `encoding/json` imported but unused |
| No Active Trading Loop | Architecture | Components aren't connected for real trading |
| Testnet Default | Configuration | Platform defaults to testnet mode |

### Codebase Completion: ~45%

---

## Binance Trading Costs Analysis

### Current State: Missing Cost Calculations

After codebase analysis, Binance trading costs are **NOT** being calculated in position sizing, leverage, or trade execution. This is critical for high-leverage scalping.

### Required Trading Costs for Mid-Cap Scalping

| Cost Type | Rate | Impact on 20-50x Scalping |
|-----------|------|---------------------------|
| Taker Fee | 0.04% (standard) - 0.02% (VIP) | 1-3% of profits |
| Maker Fee | 0.02% (standard) - 0.00% (VIP) | Lower if limit orders |
| Funding Fees | -0.01% to +0.01% every 8h | Minimal for <5min trades |
| Slippage (Mid-Cap) | 0.05% - 0.5% typical | Critical at high leverage |
| Liquidation Insurance | 0.05% - 0.1% | Catastrophic if triggered |

---

## Implementation Phases

### Phase 1: Foundation & Core Fixes (Week 1-2)

#### 1.1 Fix Compilation Errors
- Fix `platform.PositionState` type definition in reconciler
- Replace `OrderTypeStopLoss` → valid Binance order types
- Remove `NewMarkPriceService` (use alternative API)
- Fix unused imports in auditor
- Enable mainnet mode in platform configuration

#### 1.2 Establish Active Trading Loop
```go
// Connect: Watcher → Brain → Striker
func (w *Watcher) StartTradingLoop(ctx context.Context) {
    for {
        opportunities := w.detectTradingOpportunities()
        decisions := w.brain.AnalyzeOpportunities(opportunities)
        for _, decision := range decisions {
            w.striker.ExecuteDecision(ctx, decision)
        }
        time.Sleep(100 * time.Millisecond)
    }
}
```

#### 1.3 Implement Missing Core Components
- `internal/agent/engine.go` - Central orchestration
- `internal/agent/liquidity.go` - Slippage protection
- `internal/platform/security.go` - File permission enforcement

### Phase 2: Advanced Technical Analysis (Week 3-4)

#### 2.1 Optimized Indicator Suite
```go
type ScalpingIndicators struct {
    EMA_9, EMA_21, EMA_50 float64  // Fast moving averages
    RSI_6                 float64  // Overbought/oversold
    Stoch_3               float64  // Fast stochastic
    ATR_14                float64  // Stop loss calculation
    VWAP                  float64  // Volume-weighted price
    OBV                   float64  // On-balance volume
}
```

#### 2.2 Multi-Timeframe Analysis
- 1-minute for entries
- 5-minute for trend
- 15-minute for confirmation

### Phase 3: LLM-Enhanced Decision Making (Week 5-6)

#### 3.1 Brain Engine Configuration
```go
type ScalpingBrainConfig struct {
    Model             string  // "gpt-4o-mini" for speed
    ContextWindow     int     // 8K tokens
    ResponseTime      int     // <2 seconds
    RiskTolerance     float64
    LeveragePreference int    // 25-50x
}
```

#### 3.2 Strategy Router
- Knowledge base loading (capabilities.json)
- Market regime to tool mapping
- RAG pattern for strategy selection

### Phase 4: Risk Management with Cost Calculations (Week 7-8)

#### 4.1 Trading Cost Integration
```go
type TradingCosts struct {
    MakerFee         float64 // 0.0002 (0.02%)
    TakerFee         float64 // 0.0004 (0.04%)
    ExpectedSlippage float64 // 0.001 (0.1%)
    FundingFee       float64 // 0.0001 (0.01%)
    LiquidationCost  float64 // 0.001 (0.1%)
}
```

#### 4.2 Dynamic Position Sizing
```go
func CalculatePositionSize(asset *Asset, confidence float64, costs TradingCosts) float64 {
    baseSize := capital * 0.02
    totalCosts := costs.MakerFee + costs.TakerFee + costs.ExpectedSlippage + costs.FundingFee
    riskMultiplier := 1.0 + (confidence - 50.0) / 100.0
    volMultiplier := 1.0 / asset.Volatility
    liquidityMultiplier := math.Min(asset.Volume/10000000.0, 2.0)
    costMultiplier := 1.0 / (1.0 + totalCosts)
    return baseSize * riskMultiplier * volMultiplier * liquidityMultiplier * costMultiplier
}
```

#### 4.3 Circuit Breaker System
```go
type CircuitBreaker struct {
    DailyLossLimit       float64       // 10% daily loss limit
    MaxConsecutiveLosses int           // 5 consecutive losses
    VolatilityThreshold  float64       // Stop if >10% volatility
    CostThreshold        float64       // Stop if costs > 2%
    AdaptiveLimits       bool
    PerformanceWindow    time.Duration // 1 hour
}
```

### Phase 5: Self-Improving System (Week 9-10)

#### 5.1 Performance Analytics
```go
type PerformanceAnalyzer struct {
    TradeHistory        []TradeRecord
    WinRate             float64
    ProfitFactor        float64
    MaxDrawdown         float64
    SharpeRatio         float64
    AverageTradingCosts float64
    SlippageAnalysis    map[string]float64
}
```

#### 5.2 Adaptive Strategy Optimization
```go
func (pa *PerformanceAnalyzer) OptimizeStrategy() {
    trades := pa.History.GetRecent(50)
    winRate := calculateWinRate(trades)
    if winRate < 0.45 {
        pa.Config.MinConfidenceScore += 2
    }
    if avgSlippage > 0.001 {
        pa.Config.PositionSizeMultiplier *= 0.9
    }
}
```

### Phase 6: Production Deployment (Week 11-12)

#### 6.1 Infrastructure
- Low-latency VPS (AWS Frankfurt/Amsterdam)
- Redundant connections
- Monitoring and alerting
- TUI Dashboard

#### 6.2 Gradual Rollout
- 2 weeks testnet validation
- 1 week with $100
- 2 weeks with $1000
- Gradual increase to target capital

---

## Technical Specifications

### Target Assets (Mid-Cap)

| Parameter | Value |
|-----------|-------|
| Market Cap | $50M - $2B |
| Daily Volume | $10M - $100M |
| Volatility | 3% - 8% daily |
| Leverage | 20x - 50x |
| Examples | ADA, SOL, AVAX, MATIC, DOT, LINK |

### Trading Parameters

| Parameter | Value |
|-----------|-------|
| Timeframe | 1-minute entries, 5-minute exits |
| Holding Time | 30 seconds - 5 minutes |
| Risk Per Trade | 1% - 2% of capital |
| Daily Target | 5% - 15% returns |
| Max Drawdown | 10% daily, 25% monthly |

### Performance Targets

| Metric | Target |
|--------|--------|
| Win Rate | 60% - 75% |
| Profit Factor | 1.5 - 2.5 |
| Sharpe Ratio | >2.0 |
| Max Consecutive Losses | <5 |
| Latency | <100ms signal to execution |
| Uptime | >99.5% |
| Error Rate | <0.1% failed trades |

### Cost Efficiency Targets

| Metric | Target |
|--------|--------|
| Trading Costs | <1% of position value |
| Slippage | <0.3% average |
| Funding Fee Impact | <0.05% monthly |

---

## Acceptance Criteria

### Phase 1
- [ ] Code compiles without errors (`go build -buildvcs=false ./...`)
- [ ] All type errors resolved
- [ ] Mainnet mode configurable
- [ ] Basic trading loop operational

### Phase 2
- [ ] Indicator suite implemented and tested
- [ ] Multi-timeframe analysis working
- [ ] Unit tests for all indicators

### Phase 3
- [ ] Brain engine optimized for scalping
- [ ] Strategy router with knowledge base
- [ ] <2 second decision time

### Phase 4
- [ ] Trading costs calculated in all positions
- [ ] Circuit breakers functional
- [ ] Dynamic position sizing tested

### Phase 5
- [ ] Performance analytics dashboard
- [ ] Self-optimization running
- [ ] Historical trade analysis

### Phase 6
- [ ] Production infrastructure deployed
- [ ] TUI monitoring operational
- [ ] Gradual rollout complete

---

## Risk Considerations

1. **High Leverage Risk**: 20-50x amplifies losses equally
2. **Liquidity Risk**: Mid-caps can have sudden liquidity gaps
3. **API Rate Limits**: Binance may throttle high-frequency requests
4. **Slippage**: Fast markets can cause significant slippage
5. **Model Drift**: LLM decisions may degrade over time without retraining

---

## Dependencies

- Go 1.21+
- go-binance futures client
- LLM API (gpt-4o-mini or local LFM2.5)
- Redis (optional, for state caching)
- PostgreSQL (optional, for trade history)

---

## Files to Create/Modify

### New Files
- `internal/agent/engine.go`
- `internal/agent/liquidity.go`
- `internal/platform/security.go`
- `internal/brain/router.go`
- `internal/brain/optimizer.go`
- `internal/brain/lfm.go`
- `internal/ui/dashboard.go`
- `internal/costs/calculator.go`
- `configs/knowledge_base.json`
- `scripts/panic.sh`
- `scripts/audit.sh`

### Files to Fix
- `internal/striker/striker.go` - Order types
- `internal/agent/reconciler.go` - PositionState, MarkPriceService
- `internal/auditor/auditor.go` - Unused imports
- `internal/platform/platform.go` - Mainnet config

---

## Implementation Timeline

| Week | Focus | Deliverable |
|------|-------|-------------|
| 1-2 | Core Fixes | Working mainnet bot with basic trading |
| 3-4 | Technical Analysis | Advanced indicators and multi-timeframe |
| 5-6 | LLM Integration | Optimized brain engine for scalping |
| 7-8 | Risk Management | Dynamic sizing + circuit breakers + costs |
| 9-10 | Self-Improvement | Analytics and adaptive optimization |
| 11-12 | Production | Full deployment with monitoring |
