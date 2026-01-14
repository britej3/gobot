package market

import (
	"context"
	"time"

	"github.com/britebrt/cognee/domain/executor"
	"github.com/britebrt/cognee/domain/strategy"
	"github.com/britebrt/cognee/domain/trade"
)

type MarketExecutor struct {
	cfg    executor.ExecutionConfig
	client ExchangeClient
}

type ExchangeClient interface {
	CreateOrder(ctx context.Context, order *trade.Order) (*trade.Order, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetOrder(ctx context.Context, orderID string) (*trade.Order, error)
	ClosePosition(ctx context.Context, position *trade.Position) error
	GetBalance(ctx context.Context) (float64, error)
}

func NewMarketExecutor() *MarketExecutor {
	return &MarketExecutor{}
}

func (e *MarketExecutor) Type() executor.ExecutionType {
	return executor.ExecutionMarket
}

func (e *MarketExecutor) Name() string {
	return "market_executor"
}

func (e *MarketExecutor) Configure(config executor.ExecutionConfig) error {
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

func (e *MarketExecutor) Modify(ctx context.Context, orderID string, modifications executor.OrderModifications) (*trade.Order, error) {
	order, err := e.client.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if modifications.Price > 0 {
		order.Price = modifications.Price
	}
	if modifications.Quantity > 0 {
		order.Quantity = modifications.Quantity
	}
	if modifications.StopLoss > 0 {
		order.StopLoss = modifications.StopLoss
	}
	if modifications.TakeProfit > 0 {
		order.TakeProfit = modifications.TakeProfit
	}

	return order, nil
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

func generateOrderID() string {
	return time.Now().Format("20060102150405.000000")
}
