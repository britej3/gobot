# Advanced Trading Features Implementation

## Overview

This document describes the implementation of advanced trading features for GOBOT, including:

1. **Trailing Stop Loss and Take Profit** - Dynamic adjustment of SL/TP as price moves favorably
2. **Dynamic Position Sizing** - Position size based on account balance, confidence, and risk tolerance
3. **Dynamic Leveraging** - Leverage adjusts based on volatility, confidence, and risk tolerance
4. **High Risk Tolerance** - More aggressive risk parameters for screener and execution
5. **Self Optimization** - Periodic adjustment of parameters based on performance

## Components

### 1. Trailing Manager (`internal/position/trailing_manager.go`)

Manages trailing stop loss and take profit for open positions.

**Key Features:**
- Dynamic trailing SL/TP adjustment
- Configurable trail distance and activation threshold
- High risk mode with wider trails
- Automatic order updates as price moves

**Configuration:**
```go
// Default configuration
config := position.DefaultTrailingConfig()

// High risk configuration
config := position.HighRiskConfig()

// Custom configuration
config := position.TrailingConfig{
    TrailingStopEnabled:      true,
    TrailingStopPercent:      0.003,  // 0.3% trailing stop
    TrailingStopActivation:   0.005,  // Activate after 0.5% profit
    TrailingTakeProfitEnabled: true,
    TrailingTakeProfitPercent: 0.003,  // 0.3% trailing TP
    TrailingTakeProfitActivation: 0.005,  // Activate after 0.5% profit
    HighRiskMode:  false,
    RiskMultiplier: 1.0,
}
```

**Usage:**
```go
// Create trailing manager
trailingManager := position.NewTrailingManager(client, config)

// Start trailing manager
err := trailingManager.Start(ctx)

// Add position to trailing management
trailingManager.AddPosition(
    symbol,      // "BTCUSDT"
    side,        // "LONG" or "SHORT"
    entryPrice,  // 50000.0
    quantity,    // 0.001
    stopLoss,    // 49750.0
    takeProfit,  // 50250.0
    stopOrderID, // "123456"
    tpOrderID,   // "123457"
)

// Remove position when closed
trailingManager.RemovePosition(symbol)

// Stop trailing manager
trailingManager.Stop()
```

### 2. Dynamic Manager (`internal/position/dynamic_manager.go`)

Manages dynamic position sizing and leveraging with self-optimization.

**Key Features:**
- Kelly criterion-based position sizing
- Confidence-based sizing adjustment
- Volatility-based adjustment
- Dynamic leverage optimization
- Self-optimization based on performance
- High risk mode with aggressive parameters

**Configuration:**
```go
// Default configuration
config := position.DefaultDynamicConfig()

// High risk configuration
config := position.HighRiskConfig()

// Custom configuration
config := position.DynamicConfig{
    MinPositionSizeUSD:   10.0,
    MaxPositionSizeUSD:   40.0,
    MaxTotalExposureUSD:  26.0,
    MaxPositions:         3,
    BaseRiskPercent:      0.02,  // 2%
    MaxRiskPercent:       0.05,  // 5%
    RiskToleranceMode:    "moderate",  // "conservative", "moderate", "aggressive", "high"
    MinLeverage:          5,
    MaxLeverage:          25,
    BaseLeverage:         10,
    HighRiskMaxLeverage:  50,
    KellyMultiplier:       0.5,
    EnableKelly:           true,
    EnableConfidenceSizing: true,
    MinConfidence:          0.65,
    EnableVolatilityAdjustment: true,
    EnableSelfOptimization:   true,
    OptimizationInterval:     1 * time.Hour,
    PerformanceWindow:        24 * time.Hour,
}
```

**Usage:**
```go
// Create dynamic manager
dynamicManager := position.NewDynamicManager(client, config)

// Start dynamic manager
err := dynamicManager.Start(ctx)

// Calculate position size and leverage
quantity, leverage, err := dynamicManager.CalculatePositionSize(
    ctx,
    symbol,      // "BTCUSDT"
    entryPrice,  // 50000.0
    stopLoss,    // 49750.0
    confidence,  // 0.75
    volatility,  // 0.02
)

// Record completed trade for performance tracking
trade := position.Trade{
    Symbol:        "BTCUSDT",
    Side:          "LONG",
    EntryPrice:    50000.0,
    ExitPrice:     50250.0,
    Quantity:      0.001,
    PnL:           25.0,
    PnLPercent:    0.5,
    Leverage:      20,
    Confidence:    0.75,
    Volatility:    0.02,
    EntryTime:     time.Now(),
    ExitTime:      time.Now().Add(30 * time.Minute),
    Duration:      30 * time.Minute,
    ExitReason:    "TAKE_PROFIT",
}
dynamicManager.RecordTrade(trade)

// Get performance metrics
performance := dynamicManager.GetPerformanceMetrics()

// Get current configuration
config := dynamicManager.GetConfig()

// Stop dynamic manager
dynamicManager.Stop()
```

### 3. Dynamic Screener (`services/screener/dynamic_screener.go`)

Enhanced screener with dynamic adjustment and self-optimization.

**Key Features:**
- Dynamic scoring based on confidence, opportunity, and risk
- Self-optimization of filters
- High risk mode with aggressive filters
- Volume spike detection
- Volatility-based filtering

**Configuration:**
```go
// Default configuration
config := screener.DefaultDynamicScreenerConfig()

// High risk configuration
config := screener.HighRiskScreenerConfig()

// Custom configuration
config := screener.DynamicScreenerConfig{
    Interval:            5 * time.Minute,
    MaxPairs:            10,
    SortBy:              "confidence",  // "confidence", "volatility", "opportunity", "score"
    MinVolume24h:        10_000_000,
    MaxVolume24h:        100_000_000,
    MinPriceChange:      5.0,
    MaxPriceChange:      50.0,
    MinPrice:            0.01,
    MaxPrice:            10.0,
    RiskToleranceMode:   "moderate",  // "conservative", "moderate", "aggressive", "high"
    MinConfidence:       0.75,
    MaxConfidence:       1.0,
    EnableSelfOptimization: true,
    OptimizationInterval:     1 * time.Hour,
    PerformanceWindow:        24 * time.Hour,
    HighRiskMode:         false,
    VolatilityMultiplier:  1.0,
    VolumeSpikeThreshold: 1.5,
}
```

**Usage:**
```go
// Create dynamic screener
dynamicScreener := screener.NewDynamicScreener(client, config)

// Start dynamic screener
err := dynamicScreener.Start(ctx)

// Get selected assets
selectedAssets := dynamicScreener.GetSelectedAssets()

// Get all assets with scores
assets := dynamicScreener.GetAssets()

// Get screener performance
performance := dynamicScreener.GetPerformance()

// Get current configuration
config := dynamicScreener.GetConfig()

// Stop dynamic screener
dynamicScreener.Stop()
```

### 4. Enhanced Striker (`internal/striker/enhanced_striker.go`)

Enhanced trade execution with dynamic position sizing and trailing SL/TP.

**Key Features:**
- Integration with dynamic manager for position sizing
- Integration with trailing manager for SL/TP
- Dynamic leverage setting
- High risk mode support
- Volatility-based SL/TP calculation

**Usage:**
```go
// Create enhanced striker
enhancedStriker := striker.NewEnhancedStriker(
    client,
    brain,
    dynamicManager,
    trailingManager,
    highRiskMode,  // true for high risk mode
)

// Start enhanced striker
err := enhancedStriker.Start(ctx)

// Execute trading decision
decision, err := enhancedStriker.Execute(ctx, topAssets)

// Stop enhanced striker
enhancedStriker.Stop()
```

## Integration Example

Here's a complete example of integrating all components:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/adshao/go-binance/v2/futures"
    "github.com/britebrt/cognee/internal/position"
    "github.com/britebrt/cognee/internal/striker"
    "github.com/britebrt/cognee/pkg/brain"
    "github.com/britebrt/cognee/services/screener"
)

func main() {
    ctx := context.Background()

    // Initialize Binance client
    client := futures.NewClient(apiKey, apiSecret)

    // Initialize brain engine
    brainEngine := brain.NewBrainEngine(ollamaURL, model)

    // Configure for high risk mode
    highRiskMode := true

    // Create trailing manager
    trailingConfig := position.HighRiskConfig()
    trailingManager := position.NewTrailingManager(client, trailingConfig)
    err := trailingManager.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer trailingManager.Stop()

    // Create dynamic manager
    dynamicConfig := position.HighRiskConfig()
    dynamicManager := position.NewDynamicManager(client, dynamicConfig)
    err = dynamicManager.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer dynamicManager.Stop()

    // Create dynamic screener
    screenerConfig := screener.HighRiskScreenerConfig()
    dynamicScreener := screener.NewDynamicScreener(client, screenerConfig)
    err = dynamicScreener.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer dynamicScreener.Stop()

    // Create enhanced striker
    enhancedStriker := striker.NewEnhancedStriker(
        client,
        brainEngine,
        dynamicManager,
        trailingManager,
        highRiskMode,
    )
    err = enhancedStrriker.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer enhancedStriker.Stop()

    // Main trading loop
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // Get selected assets from screener
            selectedAssets := dynamicScreener.GetSelectedAssets()

            // Execute trading decisions
            decision, err := enhancedStriker.Execute(ctx, selectedAssets)
            if err != nil {
                log.Printf("Error executing trade: %v", err)
                continue
            }

            log.Printf("Trading decision: %+v", decision)

        case <-ctx.Done():
            return
        }
    }
}
```

## Configuration Comparison

### Normal Risk Mode

**Trailing Manager:**
- Trailing stop: 0.3%
- Activation: 0.5% profit
- Risk multiplier: 1.0x

**Dynamic Manager:**
- Min position: $10
- Max position: $40
- Base risk: 2%
- Max risk: 5%
- Min leverage: 5x
- Max leverage: 25x
- Kelly multiplier: 0.5

**Screener:**
- Min volume: $10M
- Max volume: $100M
- Min price change: 5%
- Max price change: 50%
- Min confidence: 0.75

### High Risk Mode

**Trailing Manager:**
- Trailing stop: 0.5% (wider)
- Activation: 0.3% profit (sooner)
- Risk multiplier: 2.0x

**Dynamic Manager:**
- Min position: $10
- Max position: $50 (higher)
- Base risk: 3% (higher)
- Max risk: 8% (much higher)
- Min leverage: 10x (higher)
- Max leverage: 50x (much higher)
- Kelly multiplier: 0.8 (more aggressive)

**Screener:**
- Min volume: $5M (lower)
- Max volume: $200M (higher)
- Min price change: 3% (lower)
- Max price change: 100% (higher)
- Min confidence: 0.60 (lower)

## Performance Metrics

The dynamic manager tracks the following metrics:

- **Total Trades**: Total number of trades executed
- **Winning Trades**: Number of profitable trades
- **Losing Trades**: Number of losing trades
- **Win Rate**: Percentage of winning trades
- **Avg Win**: Average profit per winning trade
- **Avg Loss**: Average loss per losing trade
- **Profit Factor**: Ratio of total wins to total losses
- **Max Drawdown**: Maximum loss from peak
- **Total PnL**: Total profit/loss

## Self-Optimization

Both the dynamic manager and dynamic screener include self-optimization features:

### Dynamic Manager Optimization

Adjusts parameters based on:
- **Win Rate**: Increases risk if win rate > 70%, decreases if < 50%
- **Profit Factor**: Increases Kelly if > 2.0, decreases if < 1.0
- **Total PnL**: Increases leverage if profitable, decreases if losses

### Screener Optimization

Adjusts parameters based on:
- **Asset Selection**: Relaxes filters if < 5 assets, tightens if > 80% of max
- **Market Volatility**: Increases frequency if high volatility, decreases if low

## Best Practices

1. **Start with Normal Risk Mode**: Test with normal risk parameters before switching to high risk
2. **Monitor Performance**: Regularly check performance metrics to ensure strategies are working
3. **Adjust Parameters**: Use self-optimization but also manually adjust based on market conditions
4. **Set Proper Limits**: Always set max exposure and position limits
5. **Test Thoroughly**: Test all components on testnet before deploying to mainnet
6. **Use Trailing SL/TP**: Enable trailing stop loss and take profit to maximize profits
7. **Monitor Leverage**: Keep leverage within reasonable limits for your risk tolerance
8. **Track Performance**: Record all trades and analyze performance regularly

## Troubleshooting

### Position Size Too Small

**Problem**: Calculated position size is below minimum ($10)

**Solution**: 
- Check account balance
- Reduce risk tolerance
- Increase confidence threshold
- Lower volatility adjustment

### Leverage Too Low

**Problem**: Leverage is lower than expected

**Solution**:
- Check volatility (high volatility reduces leverage)
- Increase confidence
- Switch to aggressive or high risk mode
- Adjust Kelly multiplier

### Trailing Not Activating

**Problem**: Trailing stop/take profit not activating

**Solution**:
- Check activation threshold
- Verify profit level
- Ensure trailing manager is running
- Check position is added to trailing manager

### Self-Optimization Not Working

**Problem**: Parameters not adjusting

**Solution**:
- Ensure enough trades recorded (default: 10)
- Check optimization interval
- Verify self-optimization is enabled
- Check performance window

## Future Enhancements

Potential future enhancements:

1. **Machine Learning**: Use ML for better parameter optimization
2. **Multi-Timeframe Analysis**: Incorporate multiple timeframes in decisions
3. **Sentiment Analysis**: Add social sentiment analysis
4. **Portfolio Optimization**: Optimize across multiple assets
5. **Risk Parity**: Implement risk parity allocation
6. **Correlation Analysis**: Better correlation risk management
7. **Adaptive Thresholds**: Dynamic threshold adjustment
8. **Market Regime Detection**: Detect and adapt to market regimes

## Support

For issues or questions:
- Check logs for error messages
- Verify configuration settings
- Test on testnet first
- Monitor performance metrics
- Review trade history

---

**Last Updated**: 2026-01-16
**Version**: 1.0.0