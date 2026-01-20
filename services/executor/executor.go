package executor

import (
	"context"
	"fmt"
	"sync"

	"github.com/britej3/gobot/domain/trade"
)

type Config struct {
	DefaultSize  float64
	StopLoss     float64
	TakeProfit   float64
	MaxPositions int
}

type Executor struct {
	cfg       Config
	mu        sync.RWMutex
	orders    map[string]*trade.Order
	positions map[string]*trade.Position
	binance   BinanceClient
}

type BinanceClient interface {
	CreateOrder(ctx context.Context, order *trade.Order) (*trade.Order, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetOrder(ctx context.Context, orderID string) (*trade.Order, error)
	GetPosition(ctx context.Context, symbol string) (*trade.Position, error)
	GetBalance(ctx context.Context) (float64, error)
	ClosePosition(ctx context.Context, position *trade.Position) error
}

func New(cfg Config, client BinanceClient) *Executor {
	return &Executor{
		cfg:       cfg,
		orders:    make(map[string]*trade.Order),
		positions: make(map[string]*trade.Position),
		binance:   client,
	}
}

func (e *Executor) Execute(ctx context.Context, order *trade.Order) (*trade.Order, error) {
	if err := order.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", trade.ErrInvalidOrder, err)
	}

	e.mu.Lock()
	if len(e.positions) >= e.cfg.MaxPositions {
		e.mu.Unlock()
		return nil, trade.ErrMaxPositionsReached
	}
	e.mu.Unlock()

	balance, err := e.binance.GetBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	required := order.Quantity * order.Price
	if balance < required {
		return nil, trade.ErrInsufficientBalance
	}

	result, err := e.binance.CreateOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	e.mu.Lock()
	e.orders[result.ID] = result
	if order.Side == trade.SideBuy {
		e.positions[order.Symbol] = &trade.Position{
			Symbol:     order.Symbol,
			Side:       order.Side,
			Quantity:   order.Quantity,
			EntryPrice: order.Price,
			StopLoss:   order.StopLoss,
			TakeProfit: order.TakeProfit,
			OpenedAt:   order.CreatedAt,
		}
	}
	e.mu.Unlock()

	return result, nil
}

func (e *Executor) Cancel(ctx context.Context, orderID string) error {
	e.mu.RLock()
	order, ok := e.orders[orderID]
	e.mu.RUnlock()

	if !ok {
		return trade.ErrOrderNotFound
	}

	if err := e.binance.CancelOrder(ctx, orderID); err != nil {
		return err
	}

	e.mu.Lock()
	order.Status = trade.OrderStatusCancelled
	e.mu.Unlock()

	return nil
}

func (e *Executor) GetOrder(ctx context.Context, orderID string) (*trade.Order, error) {
	e.mu.RLock()
	order, ok := e.orders[orderID]
	e.mu.RUnlock()

	if !ok {
		return nil, trade.ErrOrderNotFound
	}

	return order, nil
}

func (e *Executor) GetPosition(ctx context.Context, symbol string) (*trade.Position, error) {
	e.mu.RLock()
	pos, ok := e.positions[symbol]
	e.mu.RUnlock()

	if !ok {
		return nil, trade.ErrPositionNotFound
	}

	price, err := e.binance.GetBalance(ctx)
	if err != nil {
		return nil, err
	}
	_ = price

	currentPrice, err := e.binance.GetPosition(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if currentPrice != nil {
		pos.CurrentPrice = currentPrice.CurrentPrice
		pos.UpdatePnL(currentPrice.CurrentPrice)
	}

	return pos, nil
}

func (e *Executor) GetPositions(ctx context.Context) ([]*trade.Position, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	positions := make([]*trade.Position, 0, len(e.positions))
	for _, pos := range e.positions {
		positions = append(positions, pos)
	}
	return positions, nil
}

func (e *Executor) GetBalance(ctx context.Context) (float64, error) {
	return e.binance.GetBalance(ctx)
}

func (e *Executor) ClosePosition(ctx context.Context, position *trade.Position, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, ok := e.positions[position.Symbol]
	if !ok {
		return trade.ErrPositionNotFound
	}

	if err := e.binance.ClosePosition(ctx, position); err != nil {
		return err
	}

	delete(e.positions, position.Symbol)
	return nil
}

func (e *Executor) Config() Config {
	return e.cfg
}
