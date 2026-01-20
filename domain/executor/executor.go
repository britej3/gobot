package executor

import (
	"context"
	"sync"
	"time"

	"github.com/britej3/gobot/domain/strategy"
	"github.com/britej3/gobot/domain/trade"
)

type ExecutionType string

const (
	ExecutionMarket   ExecutionType = "market"
	ExecutionLimit    ExecutionType = "limit"
	ExecutionStopLoss ExecutionType = "stop_loss"
	ExecutionTWAP     ExecutionType = "twap"
	ExecutionVP       ExecutionType = "volume_participation"
	ExecutionIceberg  ExecutionType = "iceberg"
	ExecutionSmart    ExecutionType = "smart"
)

type Executor interface {
	Type() ExecutionType
	Name() string
	Configure(config ExecutionConfig) error
	Validate() error
	Execute(ctx context.Context, signal strategy.StrategyResult, market trade.MarketData) (*trade.Order, error)
	Cancel(ctx context.Context, orderID string) error
	Modify(ctx context.Context, orderID string, modifications OrderModifications) (*trade.Order, error)
	GetOrder(ctx context.Context, orderID string) (*trade.Order, error)
	GetOpenOrders(ctx context.Context, symbol string) ([]*trade.Order, error)
	ClosePosition(ctx context.Context, position *trade.Position, reason string) error
}

type OrderModifications struct {
	Price       float64
	Quantity    float64
	StopLoss    float64
	TakeProfit  float64
	TimeInForce string
}

type ExecutionConfig struct {
	Type               ExecutionType     `json:"type"`
	Name               string            `json:"name"`
	Enabled            bool              `json:"enabled"`
	OrderTypes         []trade.OrderType `json:"order_types"`
	DefaultTimeInForce string            `json:"default_time_in_force"`
	SlippageTolerance  float64           `json:"slippage_tolerance"`
	MaxRetries         int               `json:"max_retries"`
	RetryDelay         time.Duration     `json:"retry_delay"`
	Timeout            time.Duration     `json:"timeout"`
	PostOnly           bool              `json:"post_only"`
	ReduceOnly         bool              `json:"reduce_only"`
	IcebergConfig      IcebergConfig     `json:"iceberg_config"`
	TWAPConfig         TWAPConfig        `json:"twap_config"`
}

type IcebergConfig struct {
	DisplayQty     float64
	MaxNumIcebergs int
}

type TWAPConfig struct {
	Interval          time.Duration
	MaxDuration       time.Duration
	RandomizeInterval bool
	MinOrderSize      float64
}

type ExecutionResult struct {
	Order      *trade.Order
	FillPrice  float64
	FillTime   time.Time
	Slippage   float64
	Executions []Execution
	TotalValue float64
	TotalFees  float64
}

type Execution struct {
	OrderID     string
	Price       float64
	Quantity    float64
	Fee         float64
	FeeCurrency string
	Timestamp   time.Time
}

type ExecutionRegistry interface {
	Register(name string, factory ExecutorFactory) error
	Get(name string) (Executor, bool)
	List() []ExecutionType
	Create(cfg ExecutionConfig) (Executor, error)
}

type ExecutorFactory func() Executor

type ExecutionEngine struct {
	registry  map[ExecutionType]ExecutorFactory
	mu        sync.RWMutex
	executors map[ExecutionType]Executor
}

func NewExecutionEngine() *ExecutionEngine {
	return &ExecutionEngine{
		registry:  make(map[ExecutionType]ExecutorFactory),
		executors: make(map[ExecutionType]Executor),
	}
}

func (e *ExecutionEngine) Register(t ExecutionType, factory ExecutorFactory) error {
	e.registry[t] = factory
	return nil
}

func (e *ExecutionEngine) Get(t ExecutionType) (Executor, bool) {
	e.mu.RLock()
	exec, ok := e.executors[t]
	e.mu.RUnlock()
	return exec, ok
}

func (e *ExecutionEngine) Create(cfg ExecutionConfig) (Executor, error) {
	factory, ok := e.registry[cfg.Type]
	if !ok {
		return nil, ErrUnknownExecutor
	}

	exec := factory()
	if err := exec.Configure(cfg); err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.executors[cfg.Type] = exec
	e.mu.Unlock()

	return exec, nil
}

func (e *ExecutionEngine) List() []ExecutionType {
	types := make([]ExecutionType, 0, len(e.registry))
	for t := range e.registry {
		types = append(types, t)
	}
	return types
}

type MarketExecutor struct {
	cfg    ExecutionConfig
	client ExchangeClient
}

func NewMarketExecutor() *MarketExecutor {
	return &MarketExecutor{}
}

func (e *MarketExecutor) Type() ExecutionType {
	return ExecutionMarket
}

func (e *MarketExecutor) Name() string {
	return "market_executor"
}

func (e *MarketExecutor) Configure(config ExecutionConfig) error {
	e.cfg = config
	return nil
}

func (e *MarketExecutor) Validate() error {
	return nil
}

func (e *MarketExecutor) Execute(ctx context.Context, signal strategy.StrategyResult, market trade.MarketData) (*trade.Order, error) {
	order := &trade.Order{
		ID:         generateOrderID(),
		Symbol:     market.Symbol,
		Side:       trade.SideBuy,
		Type:       trade.OrderTypeMarket,
		Quantity:   signal.PositionSize,
		StopLoss:   signal.StopLoss,
		TakeProfit: signal.TakeProfit,
		Status:     trade.OrderStatusPending,
		CreatedAt:  time.Now(),
	}

	return e.client.CreateOrder(ctx, order)
}

func (e *MarketExecutor) Cancel(ctx context.Context, orderID string) error {
	return e.client.CancelOrder(ctx, orderID)
}

func (e *MarketExecutor) Modify(ctx context.Context, orderID string, modifications OrderModifications) (*trade.Order, error) {
	return nil, nil
}

func (e *MarketExecutor) GetOrder(ctx context.Context, orderID string) (*trade.Order, error) {
	return e.client.GetOrder(ctx, orderID)
}

func (e *MarketExecutor) GetOpenOrders(ctx context.Context, symbol string) ([]*trade.Order, error) {
	return nil, nil
}

func (e *MarketExecutor) ClosePosition(ctx context.Context, position *trade.Position, reason string) error {
	return e.client.ClosePosition(ctx, position)
}

type ExchangeClient interface {
	CreateOrder(ctx context.Context, order *trade.Order) (*trade.Order, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetOrder(ctx context.Context, orderID string) (*trade.Order, error)
	ClosePosition(ctx context.Context, position *trade.Position) error
}

func generateOrderID() string {
	return time.Now().Format("20060102150405") + randomSuffix()
}

func randomSuffix() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

var (
	ErrUnknownExecutor = &ExecutionError{Message: "unknown executor type"}
)

type ExecutionError struct {
	Message string
}

func (e *ExecutionError) Error() string {
	return e.Message
}
