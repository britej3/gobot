# GOBOT Project Reorganization Plan

## Overview

This document outlines the reorganization of GOBOT to follow **go-kata idiomatic Go patterns** and create a clean architecture ready for new logic integration (asset selection and trade execution).

## Current State Analysis

### Problems Identified

1. **Mixed Concerns**: Packages have overlapping responsibilities
2. **Circular Dependencies**: `internal/watcher` ↔ `internal/striker` ↔ `pkg/brain`
3. **Inconsistent Error Handling**: Some errors wrapped, others ignored
4. **Testing Gaps**: No table-driven tests, no parallel testing
5. **Configuration Scattered**: Environment variables, hardcoded values mixed
6. **No Clear Boundaries**: Platform package handles too much

### Go-Kata Patterns to Apply

| Pattern | Description | Apply To |
|---------|-------------|----------|
| Context Cancellation | Graceful shutdown, fail-fast | All goroutines |
| Error Wrapping | `%w` for context, `errors.Join` | All error returns |
| Interface Composition | Small interfaces, Tokyo | Public APIs |
| Zero Allocation | Buffer reuse, sync.Pool | Hot paths |
| Table-Driven Tests | Organized test cases | Critical functions |
| io/fs Patterns | Embed configs, testable | Configuration |
| Worker Pools | Rate limiting, backpressure | Scanner, Executor |

---

## Target Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           GOBOT SYSTEM                              │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                          DOMAIN LAYERS                              │
└─────────────────────────────────────────────────────────────────────┘

  ┌─────────────────────────────────────────────────────────────────┐
  │  1. ENTRY POINT (cmd/)                                          │
  │     ├─ cobot/main.go              - Main CLI entry              │
  │     ├─ analyzer/main.go           - Analyzer CLI                │
  │     └─ tester/main.go             - Test runner                 │
  └─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
  ┌─────────────────────────────────────────────────────────────────┐
  │  2. CONFIGURATION (config/)                                     │
  │     ├─ config.go                     - Config loader             │
  │     ├─ loader.go                    - Environment/vfile loader  │
  │     ├─ defaults.go                  - Default values            │
  │     ├─ validate.go                  - Validation logic          │
  │     └─ config_test.go               - Tests                     │
  └─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
  ┌─────────────────────────────────────────────────────────────────┐
  │  3. CORE DOMAIN (domain/)                                       │
  │     ├─ asset/                                                 │
  │     │   ├─ asset.go                   - Asset entity            │
  │     │   ├─ scorer.go                  - Scoring interface       │
  │     │   ├─ criteria.go                - Selection criteria      │
  │     │   └─ asset_test.go              - Tests                   │
  │     ├─ trade/                                                 │
  │     │   ├─ order.go                   - Order entity            │
  │     │   ├─ position.go                - Position entity         │
  │     │   ├─ strategy.go                - Strategy interface      │
  │     │   ├─ executor.go                - Execution interface     │
  │     │   └─ trade_test.go              - Tests                   │
  │     ├─ market/                                                │
  │     │   ├─ market.go                  - Market entity           │
  │     │   ├─ kline.go                   - Candlestick data        │
  │     │   ├─ indicators.go              - Technical indicators    │
  │     │   └─ market_test.go             - Tests                   │
  │     └─ errors/                                                │
  │         ├─ errors.go                  - Custom error types      │
  │         └─ errors_test.go             - Error tests             │
  └─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
  ┌─────────────────────────────────────────────────────────────────┐
  │  4. SERVICES (internal/)                                        │
  │     ├─ scanner/               - Asset selection service         │
  │     │   ├─ scanner.go         - Main scanner                   │
  │     │   ├─ filter.go          - Asset filtering                │
  │     │   ├─ ranker.go          - Asset ranking                  │
  │     │   └─ scanner_test.go    - Table-driven tests             │
  │     ├─ analyzer/              - External analyzer client        │
  │     │   ├─ client.go          - HTTP client with retries       │
  │     │   ├─ parser.go          - Response parsing               │
  │     │   └─ analyzer_test.go   - Mock tests                     │
  │     ├─ executor/              - Trade execution service        │
  │     │   ├─ executor.go        - Main executor                  │
  │     │   ├─ binance.go         - Binance implementation         │
  │     │   ├─ risk.go            - Risk management                │
  │     │   └─ executor_test.go   - Parallel tests                │
  │     ├─ monitor/               - Position monitoring            │
  │     │   ├─ monitor.go         - Main monitor                   │
  │     │   ├─ health.go          - Health assessment              │
  │     │   └─ monitor_test.go    - Tests                          │
  │     └─ scheduler/             - Task scheduling                │
  │         ├─ scheduler.go       - Main scheduler                 │
  │         ├─ worker.go          - Worker pool                    │
  │         └─ scheduler_test.go  - Tests                          │
  └─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
  ┌─────────────────────────────────────────────────────────────────┐
  │  5. INFRASTRUCTURE (infra/)                                     │
  │     ├─ binance/                - Binance client wrapper         │
  │     │   ├─ client.go           - HTTP client                   │
  │     │   ├─ streams.go          - WebSocket streams             │
  │     │   ├─ auth.go             - Authentication                │
  │     │   └─ binance_test.go     - Tests                         │
  │     ├─ storage/                - Persistence layer             │
  │     │   ├─ wal.go              - Write-Ahead Log               │
  │     │   ├─ state.go            - State manager                 │
  │     │   └─ storage_test.go     - Tests                         │
  │     ├─ cache/                  - Caching layer                 │
  │     │   ├─ cache.go            - In-memory cache               │
  │     │   └─ cache_test.go       - Tests                         │
  │     └─ notify/                 - Notification services         │
  │         ├─ telegram.go         - Telegram notifications        │
  │         └─ notify_test.go      - Tests                         │
  └─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
  ┌─────────────────────────────────────────────────────────────────┐
  │  6. INTERFACES (pkg/interface/)                                 │
  │     ├─ scanner.go              - Scanner interface              │
  │     ├─ executor.go             - Executor interface             │
  │     ├─ analyzer.go             - Analyzer interface             │
  │     ├─ monitor.go              - Monitor interface              │
  │     └─ scheduler.go            - Scheduler interface           │
  └─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
  ┌─────────────────────────────────────────────────────────────────┐
  │  7. UTILITIES (pkg/util/)                                       │
  │     ├─ math.go                 - Math helpers                  │
  │     ├─ time.go                 - Time helpers                  │
  │     ├─ slice.go                - Slice operations              │
  │     ├─ err.go                  - Error utilities               │
  │     └─ util_test.go            - Tests                         │
  └─────────────────────────────────────────────────────────────────┘
```

---

## Package Responsibilities

### `domain/` - Pure Business Logic

No external dependencies. Only Go standard library.

```go
// domain/asset/asset.go
type Asset struct {
    Symbol        string
    CurrentPrice  float64
    Volume24h     float64
    Volatility    float64
    Confidence    float64
    ScoredAt      time.Time
}

type Scorer interface {
    Score(ctx context.Context, asset Asset) (float64, error)
}

type Criteria struct {
    MinVolume     float64
    MaxVolatility float64
    MinConfidence float64
}
```

### `config/` - Configuration Management

Loads from env vars, files, with validation.

```go
// config/config.go
type Config struct {
    Binance    BinanceConfig
    Scanner    ScannerConfig
    Executor   ExecutorConfig
    Monitor    MonitorConfig
    Scheduler  SchedulerConfig
}

func Load(ctx context.Context, path string) (*Config, error) {
    // Load from file, apply env overrides, validate
}
```

### `services/` - Business Workflows

Orchestrate domain objects. Use interfaces from `pkg/interface/`.

```go
// services/scanner/scanner.go
type Scanner struct {
    repo       market.Repository
    scorer     domain.Scorer
    criteria   domain.Criteria
    limit      int
}

func (s *Scanner) Scan(ctx context.Context) ([]domain.Asset, error) {
    // Context cancellation support
    // Error wrapping with %w
    // Concurrent processing with worker pool
}
```

### `infra/` - External Integrations

Binance API, storage, caching. Test with interfaces.

```go
// infra/binance/client.go
type Client struct {
    apiKey     string
    secretKey  string
    httpClient *http.Client
    wsClient   *WSClient
}

func (c *Client) GetKlines(ctx context.Context, symbol string, interval string, limit int) ([]Kline, error) {
    // Request with retry
    // Context cancellation
}
```

---

## Migration Plan

### Phase 1: Create New Structure (Week 1)

#### 1.1 Create Directory Structure

```bash
mkdir -p cmd/{cobot,analyzer,tester}
mkdir -p config
mkdir -p domain/{asset,trade,market,errors}
mkdir -p services/{scanner,analyzer,executor,monitor,scheduler}
mkdir -p infra/{binance,storage,cache,notify}
mkdir -p pkg/interface
mkdir -p pkg/util
mkdir -p test/integration
mkdir -p test/functional
```

#### 1.2 Create Configuration Package

```go
// config/config.go
package config

import (
    "context"
    "os"
    "time"
)

type Config struct {
    Binance   BinanceConfig   `json:"binance"`
    Scanner   ScannerConfig   `json:"scanner"`
    Executor  ExecutorConfig  `json:"executor"`
    Monitor   MonitorConfig   `json:"monitor"`
    Scheduler SchedulerConfig `json:"scheduler"`
}

type BinanceConfig struct {
    APIKey       string `json:"api_key"`
    APISecret    string `json:"api_secret"`
    Testnet      bool   `json:"testnet"`
    Timeout      time.Duration `json:"timeout"`
    MaxRetries   int    `json:"max_retries"`
}

type ScannerConfig struct {
    Interval     time.Duration `json:"interval"`
    MinVolume    float64       `json:"min_volume"`
    MaxAssets    int           `json:"max_assets"`
    MinConfidence float64      `json:"min_confidence"`
}

type ExecutorConfig struct {
    DefaultSize   float64       `json:"default_size"`
    StopLoss      float64       `json:"stop_loss"`
    TakeProfit    float64       `json:"take_profit"`
    MaxPositions  int           `json:"max_positions"`
}

type MonitorConfig struct {
    CheckInterval time.Duration `json:"check_interval"`
    HealthThreshold float64     `json:"health_threshold"`
}

type SchedulerConfig struct {
    Workers       int           `json:"workers"`
    QueueSize     int           `json:"queue_size"`
}

func Load(ctx context.Context) (*Config, error) {
    // Load from environment
    cfg := &Config{
        Binance: BinanceConfig{
            APIKey:      os.Getenv("BINANCE_API_KEY"),
            APISecret:   os.Getenv("BINANCE_API_SECRET"),
            Testnet:     os.Getenv("BINANCE_USE_TESTNET") == "true",
            Timeout:     10 * time.Second,
            MaxRetries:  3,
        },
        Scanner: ScannerConfig{
            Interval:     2 * time.Minute,
            MinVolume:    1000000,
            MaxAssets:    15,
            MinConfidence: 0.65,
        },
        Executor: ExecutorConfig{
            DefaultSize:  0.001,
            StopLoss:     0.005,
            TakeProfit:   0.015,
            MaxPositions: 5,
        },
        Monitor: MonitorConfig{
            CheckInterval:  30 * time.Second,
            HealthThreshold: 45,
        },
        Scheduler: SchedulerConfig{
            Workers:    4,
            QueueSize:  100,
        },
    }

    return cfg, nil
}
```

#### 1.3 Create Domain Entities

```go
// domain/asset/asset.go
package asset

import "time"

type Asset struct {
    Symbol        string
    CurrentPrice  float64
    Volume24h     float64
    Volatility    float64
    RSI           float64
    EMAFast       float64
    EMASlow       float64
    Confidence    float64
    ScoredAt      time.Time
}

// Score calculates a composite score for the asset
func (a *Asset) Score(criteria Criteria) float64 {
    score := 0.0
    
    // Volatility score (0-30 points)
    if a.Volatility >= criteria.MinVolatility && a.Volatility <= criteria.MaxVolatility {
        score += 30.0
    }
    
    // Volume score (0-30 points)
    if a.Volume24h >= criteria.MinVolume {
        score += 30.0
    }
    
    // Confidence score (0-40 points)
    score += a.Confidence * 40
    
    return score
}

type Criteria struct {
    MinVolume     float64
    MaxVolume     float64
    MinVolatility float64
    MaxVolatility float64
    MinConfidence float64
}
```

```go
// domain/trade/order.go
package trade

import (
    "errors"
    "time"
)

var (
    ErrInvalidOrder    = errors.New("invalid order parameters")
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrOrderNotFound   = errors.New("order not found")
)

type Side string

const (
    SideBuy  Side = "BUY"
    SideSell Side = "SELL"
)

type OrderType string

const (
    OrderTypeMarket  OrderType = "MARKET"
    OrderTypeLimit   OrderType = "LIMIT"
    OrderTypeStopLoss OrderType = "STOP_LOSS"
)

type Order struct {
    ID            string
    Symbol        string
    Side          Side
    Type          OrderType
    Quantity      float64
    Price         float64
    StopLoss      float64
    TakeProfit    float64
    Status        OrderStatus
    FilledQty     float64
    AvgFillPrice  float64
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type OrderStatus string

const (
    OrderStatusPending   OrderStatus = "PENDING"
    OrderStatusFilled    OrderStatus = "FILLED"
    OrderStatusCancelled OrderStatus = "CANCELLED"
    OrderStatusRejected  OrderStatus = "REJECTED"
)
```

#### 1.4 Create Service Interfaces

```go
// pkg/interface/scanner.go
package interface

import (
    "context"
    "github.com/britebrt/cognee/domain/asset"
)

type Scanner interface {
    // Scan performs asset selection and returns ranked assets
    Scan(ctx context.Context) ([]asset.Asset, error)
    
    // Criteria returns the current selection criteria
    Criteria() asset.Criteria
    
    // SetCriteria updates the selection criteria
    SetCriteria(criteria asset.Criteria)
}
```

```go
// pkg/interface/executor.go
package interface

import (
    "context"
    "github.com/britebrt/cognee/domain/trade"
)

type Executor interface {
    // Execute places a new order
    Execute(ctx context.Context, order *trade.Order) (*trade.Order, error)
    
    // Cancel cancels an existing order
    Cancel(ctx context.Context, orderID string) error
    
    // GetPosition returns the current position for a symbol
    GetPosition(ctx context.Context, symbol string) (*trade.Position, error)
    
    // GetBalance returns the available balance
    GetBalance(ctx context.Context) (float64, error)
}
```

```go
// pkg/interface/analyzer.go
package interface

import (
    "context"
    "github.com/britebrt/cognee/domain/market"
)

type Analyzer interface {
    // Analyze sends market data to external analyzer
    Analyze(ctx context.Context, data market.Data) (*Advice, error)
}

type Advice struct {
    Action       string
    Symbol       string
    PositionSize float64
    EntryPrice   float64
    StopLoss     float64
    TakeProfit   float64
    Confidence   float64
    Reasoning    string
}
```

### Phase 2: Implement Core Services (Week 2)

#### 2.1 Scanner Service

```go
// services/scanner/scanner.go
package scanner

import (
    "context"
    "sort"
    "sync"

    "github.com/britebrt/cognee/domain/asset"
    "github.com/britebrt/cognee/infra/binance"
    "github.com/britebrt/cognee/pkg/interface"
)

type Scanner struct {
    client  *binance.Client
    criteria asset.Criteria
    limit   int
    mu      sync.RWMutex
}

func New(client *binance.Client, criteria asset.Criteria, limit int) *Scanner {
    return &Scanner{
        client:   client,
        criteria: criteria,
        limit:    limit,
    }
}

func (s *Scanner) Scan(ctx context.Context) ([]asset.Asset, error) {
    // Get all symbols
    symbols, err := s.client.GetSymbols(ctx)
    if err != nil {
        return nil, err
    }

    // Use worker pool for concurrent processing
    const workers = 4
    jobs := make(chan string, len(symbols))
    results := make(chan *asset.Asset, len(symbols))
    var wg sync.WaitGroup

    // Start workers
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for symbol := range jobs {
                a, err := s.scoreAsset(ctx, symbol)
                if err != nil {
                    continue
                }
                results <- a
            }
        }()
    }

    // Send jobs
    go func() {
        for _, symbol := range symbols {
            jobs <- symbol
        }
        close(jobs)
    }()

    // Wait for completion
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    var scoredAssets []asset.Asset
    for a := range results {
        scoredAssets = append(scoredAssets, *a)
    }

    // Sort by score descending
    sort.Slice(scoredAssets, func(i, j int) bool {
        return scoredAssets[i].Score(s.criteria) > scoredAssets[j].Score(s.criteria)
    })

    // Return top N
    if len(scoredAssets) > s.limit {
        return scoredAssets[:s.limit], nil
    }
    return scoredAssets, nil
}

func (s *Scanner) scoreAsset(ctx context.Context, symbol string) (*asset.Asset, error) {
    // Get klines for analysis
    klines, err := s.client.GetKlines(ctx, symbol, "5m", 50)
    if err != nil {
        return nil, err
    }

    // Calculate metrics
    price := klines.Last().Close
    volume := klines.Volume24h()
    volatility := klines.Volatility()
    rsi := klines.RSI(14)
    emaFast := klines.EMA(12)
    emaSlow := klines.EMA(26)

    return &asset.Asset{
        Symbol:       symbol,
        CurrentPrice: price,
        Volume24h:    volume,
        Volatility:   volatility,
        RSI:          rsi,
        EMAFast:      emaFast,
        EMASlow:      emaSlow,
        Confidence:   0.5, // Default, improved by analyzer
        ScoredAt:     time.Now(),
    }, nil
}

func (s *Scanner) Criteria() asset.Criteria {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.criteria
}

func (s *Scanner) SetCriteria(criteria asset.Criteria) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.criteria = criteria
}
```

#### 2.2 Executor Service

```go
// services/executor/executor.go
package executor

import (
    "context"
    "fmt"
    "sync"

    "github.com/adshao/go-binance/v2/futures"
    "github.com/britebrt/cognee/domain/trade"
    "github.com/britebrt/cognee/infra/binance"
    "github.com/britebrt/cognee/pkg/interface"
)

type Executor struct {
    client       *binance.Client
    config       ExecutorConfig
    mu           sync.RWMutex
    openPositions map[string]*trade.Position
}

type ExecutorConfig struct {
    DefaultSize  float64
    StopLoss     float64
    TakeProfit   float64
    MaxPositions int
}

func New(client *binance.Client, config ExecutorConfig) *Executor {
    return &Executor{
        client:        client,
        config:        config,
        openPositions: make(map[string]*trade.Position),
    }
}

func (e *Executor) Execute(ctx context.Context, order *trade.Order) (*trade.Order, error) {
    // Validate order
    if err := e.validateOrder(order); err != nil {
        return nil, fmt.Errorf("%w: %w", trade.ErrInvalidOrder, err)
    }

    // Check balance
    balance, err := e.GetBalance(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get balance: %w", err)
    }

    required := order.Quantity * order.Price
    if balance < required {
        return nil, trade.ErrInsufficientBalance
    }

    // Place order
    binanceOrder, err := e.client.CreateOrder(ctx, order)
    if err != nil {
        return nil, fmt.Errorf("failed to place order: %w", err)
    }

    // Update position tracking
    e.mu.Lock()
    e.openPositions[order.Symbol] = &trade.Position{
        Symbol:      order.Symbol,
        Side:        order.Side,
        Quantity:    order.Quantity,
        EntryPrice:  order.Price,
        StopLoss:    order.StopLoss,
        TakeProfit:  order.TakeProfit,
    }
    e.mu.Unlock()

    return order, nil
}

func (e *Executor) validateOrder(order *trade.Order) error {
    if order.Quantity <= 0 {
        return fmt.Errorf("quantity must be positive")
    }
    if order.Price <= 0 {
        return fmt.Errorf("price must be positive")
    }
    if order.Symbol == "" {
        return fmt.Errorf("symbol is required")
    }
    return nil
}

func (e *Executor) Cancel(ctx context.Context, orderID string) error {
    return e.client.CancelOrder(ctx, orderID)
}

func (e *Executor) GetPosition(ctx context.Context, symbol string) (*trade.Position, error) {
    e.mu.RLock()
    defer e.mu.RUnlock()
    
    pos, ok := e.openPositions[symbol]
    if !ok {
        return nil, nil // No position
    }
    
    // Update current price
    currentPrice, err := e.client.GetPrice(ctx, symbol)
    if err != nil {
        return nil, err
    }
    pos.CurrentPrice = currentPrice
    pos.UpdatePnL()
    
    return pos, nil
}

func (e *Executor) GetBalance(ctx context.Context) (float64, error) {
    return e.client.GetBalance(ctx)
}
```

### Phase 3: Tests & Migration (Week 3)

#### 3.1 Table-Driven Tests

```go
// services/scanner/scanner_test.go
package scanner

import (
    "context"
    "testing"
)

func TestScanner_Scan(t *testing.T) {
    tests := []struct {
        name       string
        criteria   asset.Criteria
        mockSetup  func(*mockBinanceClient)
        wantCount  int
        wantErr    bool
    }{
        {
            name: "valid scan with assets",
            criteria: asset.Criteria{
                MinVolume:     1000000,
                MinVolatility: 0.5,
                MaxVolatility: 5.0,
                MinConfidence: 0.5,
            },
            mockSetup: func(m *mockBinanceClient) {
                m.On("GetSymbols").Return([]string{"BTCUSDT", "ETHUSDT"})
                m.On("GetKlines", "BTCUSDT").Return(mockKlines(50))
                m.On("GetKlines", "ETHUSDT").Return(mockKlines(50))
            },
            wantCount: 2,
            wantErr:   false,
        },
        {
            name: "no assets meet criteria",
            criteria: asset.Criteria{
                MinVolume:     1000000000, // Very high
                MinVolatility: 0.5,
                MaxVolatility: 5.0,
                MinConfidence: 0.5,
            },
            mockSetup: func(m *mockBinanceClient) {
                m.On("GetSymbols").Return([]string{"BTCUSDT"})
                m.On("GetKlines", "BTCUSDT").Return(mockKlines(50))
            },
            wantCount: 0,
            wantErr:   false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockClient := newMockBinanceClient()
            tt.mockSetup(mockClient)

            s := New(mockClient, tt.criteria, 10)
            assets, err := s.Scan(context.Background())

            if (err != nil) != tt.wantErr {
                t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if len(assets) != tt.wantCount {
                t.Errorf("Scan() got %d assets, want %d", len(assets), tt.wantCount)
            }
        })
    }
}
```

#### 3.2 Parallel Testing

```go
// services/executor/executor_test.go
package executor

import (
    "context"
    "testing"
)

func TestExecutor_Execute(t *testing.T) {
    t.Run("parallel order execution", func(t *testing.T) {
        t.Parallel() // Mark as parallel test

        // Setup
        mockClient := newMockBinanceClient()
        executor := New(mockClient, ExecutorConfig{
            DefaultSize: 0.001,
            StopLoss:    0.005,
            TakeProfit:  0.015,
        })

        // Test concurrent orders
        orders := []*trade.Order{
            {Symbol: "BTCUSDT", Side: trade.SideBuy, Quantity: 0.001, Price: 98000},
            {Symbol: "ETHUSDT", Side: trade.SideBuy, Quantity: 0.01, Price: 3200},
        }

        var wg sync.WaitGroup
        results := make(chan *trade.Order, len(orders))

        for _, order := range orders {
            wg.Add(1)
            go func(o *trade.Order) {
                defer wg.Done()
                result, err := executor.Execute(context.Background(), o)
                if err != nil {
                    t.Errorf("Execute() error = %v", err)
                    return
                }
                results <- result
            }(order)
        }

        wg.Wait()
        close(results)

        count := 0
        for range results {
            count++
        }

        if count != len(orders) {
            t.Errorf("Expected %d orders, got %d", len(orders), count)
        }
    })
}
```

### Phase 4: Migration Script (Week 4)

```bash
#!/bin/bash
# migrate.sh - Migrate from old structure to new structure

echo "GOBOT Migration Script"
echo "======================"
echo ""

# Create new directories
echo "[1/6] Creating new directory structure..."
mkdir -p cmd/{cobot,analyzer,tester}
mkdir -p config
mkdir -p domain/{asset,trade,market,errors}
mkdir -p services/{scanner,analyzer,executor,monitor,scheduler}
mkdir -p infra/{binance,storage,cache,notify}
mkdir -p pkg/interface
mkdir -p pkg/util
mkdir -p test/{unit,integration,functional}
echo "   Done."

# Copy existing implementations (to be refactored)
echo "[2/6] Copying existing implementations..."
cp -r internal/watcher/*.go services/scanner/ 2>/dev/null || true
cp -r internal/striker/*.go services/executor/ 2>/dev/null || true
cp -r internal/position/*.go services/monitor/ 2>/dev/null || true
cp -r pkg/brain/*.go services/analyzer/ 2>/dev/null || true
cp -r internal/platform/*.go infra/ 2>/dev/null || true
echo "   Done."

# Create new interface files
echo "[3/6] Creating new interfaces..."
cat > pkg/interface/scanner.go << 'EOF'
package interface

import (
    "context"
    "github.com/britebrt/cognee/domain/asset"
)

type Scanner interface {
    Scan(ctx context.Context) ([]asset.Asset, error)
    Criteria() asset.Criteria
    SetCriteria(asset.Criteria)
}
EOF

cat > pkg/interface/executor.go << 'EOF'
package interface

import (
    "context"
    "github.com/britebrt/cognee/domain/trade"
)

type Executor interface {
    Execute(ctx context.Context, order *trade.Order) (*trade.Order, error)
    Cancel(ctx context.Context, orderID string) error
    GetPosition(ctx context.Context, symbol string) (*trade.Position, error)
    GetBalance(ctx context.Context) (float64, error)
}
EOF

cat > pkg/interface/analyzer.go << 'EOF'
package interface

import (
    "context"
    "github.com/britebrt/cognee/domain/market"
)

type Analyzer interface {
    Analyze(ctx context.Context, data market.Data) (*Advice, error)
}

type Advice struct {
    Action       string
    Symbol       string
    PositionSize float64
    EntryPrice   float64
    StopLoss     float64
    TakeProfit   float64
    Confidence   float64
    Reasoning    string
}
EOF
echo "   Done."

# Create domain entities
echo "[4/6] Creating domain entities..."
cat > domain/asset/asset.go << 'EOF'
package asset

import "time"

type Asset struct {
    Symbol        string
    CurrentPrice  float64
    Volume24h     float64
    Volatility    float64
    RSI           float64
    EMAFast       float64
    EMASlow       float64
    Confidence    float64
    ScoredAt      time.Time
}

type Criteria struct {
    MinVolume     float64
    MaxVolume     float64
    MinVolatility float64
    MaxVolatility float64
    MinConfidence float64
}

func (a *Asset) Score(c Criteria) float64 {
    score := 0.0
    if a.Volatility >= c.MinVolatility && a.Volatility <= c.MaxVolatility {
        score += 30.0
    }
    if a.Volume24h >= c.MinVolume {
        score += 30.0
    }
    score += a.Confidence * 40
    return score
}
EOF

cat > domain/trade/order.go << 'EOF'
package trade

import (
    "errors"
    "time"
)

var (
    ErrInvalidOrder = errors.New("invalid order parameters")
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrOrderNotFound = errors.New("order not found")
)

type Side string

const (
    SideBuy  Side = "BUY"
    SideSell Side = "SELL"
)

type OrderType string

const (
    OrderTypeMarket   OrderType = "MARKET"
    OrderTypeLimit    OrderType = "LIMIT"
    OrderTypeStopLoss OrderType = "STOP_LOSS"
)

type Order struct {
    ID           string
    Symbol       string
    Side         Side
    Type         OrderType
    Quantity     float64
    Price        float64
    StopLoss     float64
    TakeProfit   float64
    Status       OrderStatus
    FilledQty    float64
    AvgFillPrice float64
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type OrderStatus string

const (
    OrderStatusPending  OrderStatus = "PENDING"
    OrderStatusFilled   OrderStatus = "FILLED"
    OrderStatusCancelled OrderStatus = "CANCELLED"
    OrderStatusRejected OrderStatus = "REJECTED"
)

type Position struct {
    Symbol      string
    Side        Side
    Quantity    float64
    EntryPrice  float64
    CurrentPrice float64
    StopLoss    float64
    TakeProfit  float64
    PnL         float64
    PnLPercent  float64
}

func (p *Position) UpdatePnL() {
    if p.Side == SideBuy {
        p.PnL = (p.CurrentPrice - p.EntryPrice) * p.Quantity
        p.PnLPercent = (p.CurrentPrice - p.EntryPrice) / p.EntryPrice * 100
    } else {
        p.PnL = (p.EntryPrice - p.CurrentPrice) * p.Quantity
        p.PnLPercent = (p.EntryPrice - p.CurrentPrice) / p.EntryPrice * 100
    }
}
EOF
echo "   Done."

# Create configuration
echo "[5/6] Creating configuration..."
cat > config/config.go << 'EOF'
package config

import (
    "context"
    "os"
    "time"
)

type Config struct {
    Binance   BinanceConfig
    Scanner   ScannerConfig
    Executor  ExecutorConfig
    Monitor   MonitorConfig
}

type BinanceConfig struct {
    APIKey     string
    APISecret  string
    Testnet    bool
    Timeout    time.Duration
    MaxRetries int
}

type ScannerConfig struct {
    Interval     time.Duration
    MinVolume    float64
    MaxAssets    int
    MinConfidence float64
}

type ExecutorConfig struct {
    DefaultSize  float64
    StopLoss     float64
    TakeProfit   float64
    MaxPositions int
}

type MonitorConfig struct {
    CheckInterval   time.Duration
    HealthThreshold float64
}

func Load(ctx context.Context) (*Config, error) {
    return &Config{
        Binance: BinanceConfig{
            APIKey:     os.Getenv("BINANCE_API_KEY"),
            APISecret:  os.Getenv("BINANCE_API_SECRET"),
            Testnet:    os.Getenv("BINANCE_USE_TESTNET") == "true",
            Timeout:    10 * time.Second,
            MaxRetries: 3,
        },
        Scanner: ScannerConfig{
            Interval:     2 * time.Minute,
            MinVolume:    1000000,
            MaxAssets:    15,
            MinConfidence: 0.65,
        },
        Executor: ExecutorConfig{
            DefaultSize:  0.001,
            StopLoss:     0.005,
            TakeProfit:   0.015,
            MaxPositions: 5,
        },
        Monitor: MonitorConfig{
            CheckInterval:   30 * time.Second,
            HealthThreshold: 45,
        },
    }, nil
}
EOF
echo "   Done."

# Create main entry point
echo "[6/6] Creating main entry point..."
cat > cmd/cobot/main.go << 'EOF'
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/britebrt/cognee/config"
    "github.com/britebrt/cognee/infra/binance"
    "github.com/britebrt/cognee/services/executor"
    "github.com/britebrt/cognee/services/scanner"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Load configuration
    cfg, err := config.Load(ctx)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize Binance client
    client := binance.NewClient(binance.Config{
        APIKey:     cfg.Binance.APIKey,
        APISecret:  cfg.Binance.APISecret,
        Testnet:    cfg.Binance.Testnet,
        Timeout:    cfg.Binance.Timeout,
    })

    // Initialize services
    scannerService := scanner.New(client, asset.Criteria{
        MinVolume: cfg.Scanner.MinVolume,
    }, cfg.Scanner.MaxAssets)

    exec := executor.New(client, executor.Config{
        DefaultSize:  cfg.Executor.DefaultSize,
        StopLoss:     cfg.Executor.StopLoss,
        TakeProfit:   cfg.Executor.TakeProfit,
        MaxPositions: cfg.Executor.MaxPositions,
    })

    // Setup graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        log.Println("Shutting down...")
        cancel()
    }()

    // Main loop
    log.Println("Starting GOBOT...")
    
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Scan for assets
            assets, err := scannerService.Scan(ctx)
            if err != nil {
                log.Printf("Scan failed: %v", err)
                continue
            }

            log.Printf("Found %d candidate assets", len(assets))
            
            for _, asset := range assets {
                // Execute trades for top assets
                log.Printf("Processing %s at $%.2f", asset.Symbol, asset.CurrentPrice)
            }
        }
    }
}
EOF
echo "   Done."

echo ""
echo "Migration complete!"
echo "Run 'go mod tidy' to update dependencies."
```

---

## Summary

This reorganization provides:

1. **Clean Architecture**: Domain → Services → Infrastructure
2. **Go-Kata Compliance**: 
   - Context cancellation throughout
   - Error wrapping with `%w`
   - Interface composition
   - Table-driven tests
   - Worker pools for concurrency
3. **Easy Integration**: New asset selection and trade execution logic can be added to the appropriate domain/service layer
4. **Testable**: All components have interfaces for mocking
5. **Production-Ready**: Configuration management, graceful shutdown, error handling

## Next Steps

1. Review and approve this plan
2. Run migration script
3. Refactor existing code to new structure
4. Add unit tests for all components
5. Integration testing
6. Deploy to production
