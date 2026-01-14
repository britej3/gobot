# GOBOT Modular Architecture - Swapable Components

## Overview
GOBOT now has a fully modular architecture allowing you to easily swap:
- Trading Strategies
- Coin Selection Logic
- Execution Methods
- N8N Automation Integration

## Directory Structure

```
GOBOT/
├── cmd/cobot/main.go           # Main entry point
├── config/config.go            # Configuration loading
├── domain/
│   ├── strategy/               # Strategy interfaces & types
│   │   └── strategy.go
│   ├── selector/               # Coin selection interfaces
│   │   └── selector.go
│   ├── executor/               # Execution interfaces
│   │   └── executor.go
│   ├── automation/             # N8N automation interfaces
│   │   └── automation.go
│   ├── platform/               # Platform & engine
│   │   └── platform.go
│   └── trade/                  # Trade types
├── services/
│   ├── strategy/
│   │   ├── scalper/            # Scalper implementation
│   │   │   └── scalper.go
│   │   └── momentum/           # Momentum implementation
│   │       └── momentum.go
│   ├── selector/
│   │   └── volume/             # Volume-based selector
│   │       └── volume.go
│   ├── executor/
│   │   └── market/             # Market executor
│   ├── scanner/scanner.go
│   ├── monitor/monitor.go
│   ├── analyzer/client.go
│   └── scheduler/scheduler.go
├── infra/
│   ├── binance/client.go       # Binance API client
│   ├── storage/wal.go          # Write-Ahead Log
│   └── cache/cache.go          # In-memory cache
└── pkg/ifaces/interfaces.go
```

## How to Swap Components

### 1. Swap Trading Strategy

**Current Strategies:**
- `StrategyScalper` - Short-term scalping (1-15 min)
- `StrategyMomentum` - Momentum-based trading

**To add a new strategy:**

```go
// 1. Create new file: services/strategy/yourname/yourname.go
package yourname

import (
    "context"
    "github.com/britebrt/cognee/domain/strategy"
    "github.com/britebrt/cognee/domain/trade"
)

type YourStrategy struct {
    cfg strategy.StrategyConfig
}

func (s *YourStrategy) Type() strategy.StrategyType {
    return strategy.StrategyCustom // or new type
}

func (s *YourStrategy) Name() string {
    return "your_strategy"
}

func (s *YourStrategy) Configure(config strategy.StrategyConfig) error {
    s.cfg = config
    return nil
}

func (s *YourStrategy) ShouldEnter(ctx context.Context, market trade.MarketData) (bool, string, error) {
    // Your logic here
    return true, "buy signal", nil
}

// ... implement other required methods
```

**To use it:**

```go
engine.RegisterStrategy(strategy.StrategyCustom, func() strategy.Strategy {
    return &yourname.YourStrategy{}
})

cfg.StrategyConfig.Type = strategy.StrategyCustom
```

### 2. Swap Coin Selector

**Current Selectors:**
- `SelectorVolume` - Volume-based filtering
- `SelectorVolatility` - Volatility-based filtering
- `SelectorAI` - AI-powered selection

**To add a new selector:**

```go
// 1. Create new file: services/selector/yourname/yourname.go
package yourname

import (
    "context"
    "github.com/britebrt/cognee/domain/asset"
    "github.com/britebrt/cognee/domain/selector"
    "github.com/britebrt/cognee/domain/trade"
)

type YourSelector struct {
    cfg selector.SelectorConfig
}

func (s *YourSelector) Type() selector.SelectorType {
    return selector.SelectorCustom
}

func (s *YourSelector) Select(ctx context.Context, marketData map[string]*trade.MarketData) ([]asset.Asset, error) {
    // Your selection logic
    return assets, nil
}

// ... implement other required methods
```

**To use it:**

```go
engine.RegisterSelector(selector.SelectorCustom, func() selector.Selector {
    return &yourname.YourSelector{}
})

cfg.SelectorConfig.Type = selector.SelectorCustom
```

### 3. Swap Execution Method

**Current Executors:**
- `ExecutionMarket` - Market orders
- `ExecutionLimit` - Limit orders
- `ExecutionTWAP` - Time-weighted average price
- `ExecutionSmart` - Smart order routing

**To add a new executor:**

```go
// 1. Create new file: services/executor/yourname/yourname.go
package yourname

import (
    "context"
    "github.com/britebrt/cognee/domain/executor"
    "github.com/britebrt/cognee/domain/strategy"
    "github.com/britebrt/cognee/domain/trade"
)

type YourExecutor struct {
    cfg executor.ExecutionConfig
}

func (e *YourExecutor) Type() executor.ExecutionType {
    return executor.ExecutionCustom
}

func (e *YourExecutor) Execute(ctx context.Context, signal strategy.StrategyResult, market trade.MarketData) (*trade.Order, error) {
    // Your execution logic
    return order, nil
}

// ... implement other required methods
```

**To use it:**

```go
engine.RegisterExecutor(executor.ExecutionCustom, func() executor.Executor {
    return &yourname.YourExecutor{}
})

cfg.ExecutorConfig.Type = executor.ExecutionCustom
```

### 4. Swap N8N Automation

**Current Automations:**
- `AutomationN8N` - N8N webhook integration

**To configure N8N:**

```json
{
  "automation_config": {
    "type": "n8n",
    "n8n_config": {
      "base_url": "http://localhost:5678",
      "api_key": "your-api-key",
      "workflows": [
        {
          "id": "trade_signal",
          "trigger_type": "trade_signal",
          "enabled": true
        },
        {
          "id": "risk_alert", 
          "trigger_type": "risk_alert",
          "enabled": true
        }
      ]
    }
  }
}
```

**N8N Workflow Setup:**

1. Create webhook in N8N: `http://localhost:5678/webhook/trade_signal`
2. Add workflow to process trade signals
3. Enable in config

### 5. Complete Example - Switching All Components

```go
func main() {
    engine := platform.NewPlatformEngine()
    
    // Register all components
    engine.RegisterStrategy(strategy.StrategyMomentum, func() strategy.Strategy {
        return &momentum.MomentumStrategy{}
    })
    
    engine.RegisterSelector(selector.SelectorVolume, func() selector.Selector {
        return &volume.VolumeSelector{}
    })
    
    engine.RegisterExecutor(executor.ExecutionSmart, func() executor.Executor {
        return &smart.SmartExecutor{}
    })
    
    engine.RegisterAutomation(automation.AutomationN8N, func() automation.Automation {
        return &automation.N8NAutomation{}
    })
    
    // Configure platform
    platform := &platform.Platform{
        Config: platform.PlatformConfig{
            StrategyConfig: strategy.StrategyConfig{
                Type: strategy.StrategyMomentum,
            },
            SelectorConfig: selector.SelectorConfig{
                Type: selector.SelectorVolume,
            },
            ExecutorConfig: executor.ExecutionConfig{
                Type: executor.ExecutionSmart,
            },
            AutomationConfig: automation.AutomationConfig{
                Type: automation.AutomationN8N,
                N8NConfig: automation.N8NConfig{
                    BaseURL: "http://localhost:5678",
                },
            },
        },
        Engine: engine,
    }
}
```

## Configuration Files

### Strategy Configuration
```json
{
  "strategy_config": {
    "type": "scalper",
    "name": "my_strategy",
    "enabled": true,
    "parameters": {
      "risk_per_trade": 0.02,
      "stop_loss_percent": 0.005,
      "take_profit_percent": 0.015
    },
    "risk_parameters": {
      "max_position_size": 0.1,
      "max_order_value": 1000
    }
  }
}
```

### Selector Configuration
```json
{
  "selector_config": {
    "type": "volume",
    "name": "my_selector",
    "enabled": true,
    "min_volume": 1000000,
    "max_assets": 15,
    "min_confidence": 0.65,
    "weightings": {
      "volume": 0.4,
      "volatility": 0.3,
      "rsi": 0.3
    }
  }
}
```

### Executor Configuration
```json
{
  "executor_config": {
    "type": "market",
    "name": "my_executor",
    "enabled": true,
    "slippage_tolerance": 0.001,
    "max_retries": 3,
    "timeout": 10000000000
  }
}
```

## Environment Variables

```bash
# Binance
BINANCE_API_KEY=your-api-key
BINANCE_API_SECRET=your-secret
BINANCE_USE_TESTNET=true

# N8N
N8N_BASE_URL=http://localhost:5678
N8N_API_KEY=your-api-key

# Platform
GOBOT_ENV=production
GOBOT_LOG_LEVEL=info
```

## Running the Platform

```bash
# Build
go build -o gobot ./cmd/cobot

# Run
./gobot

# With custom config
./gobot --config /path/to/config.json
```

## Testing Components

```go
// Test a strategy
func TestScalperStrategy(t *testing.T) {
    s := &scalper.ScalperStrategy{}
    s.Configure(strategy.StrategyConfig{
        RiskParameters: strategy.RiskConfig{
            RiskPerTrade: 0.02,
        },
    })
    
    market := trade.MarketData{
        RSI:     50,
        EMAFast: 100,
        EMASlow: 99,
    }
    
    enter, reason, err := s.ShouldEnter(context.Background(), market)
    assert.NoError(t, err)
    assert.True(t, enter)
    assert.Contains(t, reason, "bullish")
}

// Test a selector
func TestVolumeSelector(t *testing.T) {
    sel := &volume.VolumeSelector{}
    sel.Configure(selector.SelectorConfig{
        MinVolume: 1000000,
        MaxAssets: 10,
    })
    
    marketData := map[string]*trade.MarketData{
        "BTCUSDT": {
            Symbol:    "BTCUSDT",
            Volume24h: 2000000000,
            RSI:       55,
        },
    }
    
    assets, err := sel.Select(context.Background(), marketData)
    assert.NoError(t, err)
    assert.Len(t, assets, 1)
}
```

## Remaining Tasks

| Task | Status | Priority |
|------|--------|----------|
| Create strategy interfaces | DONE | High |
| Create selector interfaces | DONE | High |
| Create executor interfaces | DONE | High |
| Create automation interfaces | DONE | High |
| Implement Scalper strategy | DONE | Medium |
| Implement Momentum strategy | DONE | Medium |
| Implement Volume selector | DONE | Medium |
| Implement Market executor | PENDING | Medium |
| Implement N8N automation | DONE | Medium |
| Create platform engine | DONE | High |
| Update main.go | DONE | High |
| Add unit tests | PENDING | High |
| Compile and verify | PENDING | High |
| Create strategy factory | PENDING | Medium |
| Create config loader for strategies | PENDING | Medium |

## Next Steps

1. **Complete Market Executor** - Implement `services/executor/market/market.go`
2. **Add Unit Tests** - Create table-driven tests for all components
3. **Compile Verification** - Run `go build` and fix any errors
4. **Strategy Factory** - Create factory patterns for dynamic loading
5. **Config Loader** - Support JSON/YAML config files for strategies
6. **Integration Tests** - Test complete trading cycle
7. **Documentation** - Add godoc comments and examples
