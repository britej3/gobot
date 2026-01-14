package strategy

import (
	"context"
	"sync"
	"time"

	"github.com/britebrt/cognee/domain/trade"
)

type StrategyType string

const (
	StrategyScalper     StrategyType = "scalper"
	StrategyMomentum    StrategyType = "momentum"
	StrategySwing       StrategyType = "swing"
	StrategyGrid        StrategyType = "grid"
	StrategyAIAutomated StrategyType = "ai_automated"
	StrategyCustom      StrategyType = "custom"
)

type Strategy interface {
	Type() StrategyType
	Name() string
	Version() string
	Configure(config StrategyConfig) error
	Validate() error
	ShouldEnter(ctx context.Context, market trade.MarketData) (bool, string, error)
	ShouldExit(ctx context.Context, position *trade.Position, market trade.MarketData) (bool, string, error)
	CalculatePositionSize(ctx context.Context, market trade.MarketData, balance float64) (float64, error)
	CalculateStopLoss(ctx context.Context, entryPrice float64, market trade.MarketData) (float64, error)
	CalculateTakeProfit(ctx context.Context, entryPrice float64, market trade.MarketData) (float64, error)
	CalculateTrailingStop(ctx context.Context, position *trade.Position, market trade.MarketData) (float64, error)
	OnTick(ctx context.Context, position *trade.Position, market trade.MarketData) error
	OnOrderFill(ctx context.Context, order *trade.Order, position *trade.Position) error
	OnPositionClose(ctx context.Context, position *trade.Position, reason string) error
	GetParameters() map[string]interface{}
}

type StrategyConfig struct {
	Type           StrategyType       `json:"type"`
	Name           string             `json:"name"`
	Version        string             `json:"version"`
	Enabled        bool               `json:"enabled"`
	Parameters     map[string]float64 `json:"parameters"`
	RiskParameters RiskConfig         `json:"risk_parameters"`
	Filters        []FilterConfig     `json:"filters"`
	Timeframes     []string           `json:"timeframes"`
	MaxPositions   int                `json:"max_positions"`
	MaxDrawdown    float64            `json:"max_drawdown"`
	DailyLossLimit float64            `json:"daily_loss_limit"`
}

type RiskConfig struct {
	MaxPositionSize     float64 `json:"max_position_size"`
	MaxOrderValue       float64 `json:"max_order_value"`
	StopLossPercent     float64 `json:"stop_loss_percent"`
	TakeProfitPercent   float64 `json:"take_profit_percent"`
	TrailingStopPercent float64 `json:"trailing_stop_percent"`
	MaxLeverage         float64 `json:"max_leverage"`
	RiskPerTrade        float64 `json:"risk_per_trade"`
}

type FilterConfig struct {
	Type     string  `json:"type"`
	Field    string  `json:"field"`
	Operator string  `json:"operator"`
	Value    float64 `json:"value"`
	Enabled  bool    `json:"enabled"`
}

type StrategyResult struct {
	ShouldEnter  bool
	ShouldExit   bool
	PositionSize float64
	StopLoss     float64
	TakeProfit   float64
	TrailingStop float64
	Reason       string
	Confidence   float64
	RiskScore    float64
}

type StrategyRegistry interface {
	Register(name string, factory StrategyFactory) error
	Get(name string) (Strategy, bool)
	List() []StrategyType
	Create(cfg StrategyConfig) (Strategy, error)
}

type StrategyFactory func() Strategy

type StrategyEngine struct {
	registry   map[StrategyType]StrategyFactory
	mu         map[StrategyType]*sync.RWMutex
	strategies map[StrategyType]Strategy
}

func NewStrategyEngine() *StrategyEngine {
	return &StrategyEngine{
		registry:   make(map[StrategyType]StrategyFactory),
		mu:         make(map[StrategyType]*sync.RWMutex),
		strategies: make(map[StrategyType]Strategy),
	}
}

func (e *StrategyEngine) Register(t StrategyType, factory StrategyFactory) error {
	e.registry[t] = factory
	e.mu[t] = &sync.RWMutex{}
	return nil
}

func (e *StrategyEngine) Get(t StrategyType) (Strategy, bool) {
	e.mu[t].RLock()
	defer e.mu[t].RUnlock()
	s, ok := e.strategies[t]
	return s, ok
}

func (e *StrategyEngine) Create(cfg StrategyConfig) (Strategy, error) {
	factory, ok := e.registry[cfg.Type]
	if !ok {
		return nil, ErrUnknownStrategy
	}

	s := factory()
	if err := s.Configure(cfg); err != nil {
		return nil, err
	}

	e.mu[cfg.Type].Lock()
	e.strategies[cfg.Type] = s
	e.mu[cfg.Type].Unlock()

	return s, nil
}

func (e *StrategyEngine) List() []StrategyType {
	types := make([]StrategyType, 0, len(e.registry))
	for t := range e.registry {
		types = append(types, t)
	}
	return types
}

var (
	ErrUnknownStrategy = &StrategyError{Message: "unknown strategy type"}
	ErrInvalidConfig   = &StrategyError{Message: "invalid strategy configuration"}
)

type StrategyError struct {
	Message string
}

func (e *StrategyError) Error() string {
	return e.Message
}

type BacktestResult struct {
	Strategy    StrategyType
	StartTime   time.Time
	EndTime     time.Time
	TotalTrades int
	WinRate     float64
	TotalPnL    float64
	MaxDrawdown float64
	SharpeRatio float64
	Trades      []BacktestTrade
}

type BacktestTrade struct {
	EntryTime    time.Time
	ExitTime     time.Time
	Symbol       string
	EntryPrice   float64
	ExitPrice    float64
	Quantity     float64
	Side         trade.Side
	PnL          float64
	PnLPercent   float64
	HoldDuration time.Duration
}
