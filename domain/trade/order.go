package trade

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidOrder        = errors.New("invalid order parameters")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrOrderNotFound       = errors.New("order not found")
	ErrPositionNotFound    = errors.New("position not found")
	ErrInvalidQuantity     = errors.New("quantity must be positive")
	ErrInvalidPrice        = errors.New("price must be positive")
	ErrInvalidSymbol       = errors.New("symbol is required")
	ErrContextCancelled    = errors.New("operation cancelled by context")
	ErrRiskLimitExceeded   = errors.New("risk limit exceeded")
	ErrMaxPositionsReached = errors.New("maximum positions reached")
)

type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

func (s Side) IsValid() bool {
	return s == SideBuy || s == SideSell
}

func (s Side) Opposite() Side {
	if s == SideBuy {
		return SideSell
	}
	return SideBuy
}

type OrderType string

const (
	OrderTypeMarket     OrderType = "MARKET"
	OrderTypeLimit      OrderType = "LIMIT"
	OrderTypeStopLoss   OrderType = "STOP_LOSS"
	OrderTypeTakeProfit OrderType = "TAKE_PROFIT"
)

func (ot OrderType) IsValid() bool {
	switch ot {
	case OrderTypeMarket, OrderTypeLimit, OrderTypeStopLoss, OrderTypeTakeProfit:
		return true
	}
	return false
}

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusSubmitted OrderStatus = "SUBMITTED"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusPartially OrderStatus = "PARTIALLY_FILLED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRejected  OrderStatus = "REJECTED"
	OrderStatusExpired   OrderStatus = "EXPIRED"
)

func (s OrderStatus) IsTerminal() bool {
	switch s {
	case OrderStatusFilled, OrderStatusCancelled, OrderStatusRejected, OrderStatusExpired:
		return true
	}
	return false
}

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
	Commission   float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (o *Order) Validate() error {
	if o.Symbol == "" {
		return ErrInvalidSymbol
	}
	if o.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	if o.Price <= 0 {
		return ErrInvalidPrice
	}
	if !o.Side.IsValid() {
		return ErrInvalidOrder
	}
	if !o.Type.IsValid() {
		return ErrInvalidOrder
	}
	return nil
}

func (o *Order) Remaining() float64 {
	return o.Quantity - o.FilledQty
}

func (o *Order) IsFilled() bool {
	return o.Status == OrderStatusFilled
}

func (o *Order) Fill(qty, price float64) {
	o.FilledQty += qty
	o.AvgFillPrice = (o.AvgFillPrice*(o.FilledQty-qty) + price*qty) / o.FilledQty
	o.UpdatedAt = time.Now()

	if o.FilledQty >= o.Quantity {
		o.Status = OrderStatusFilled
	} else {
		o.Status = OrderStatusPartially
	}
}

type Position struct {
	Symbol       string
	Side         Side
	Quantity     float64
	EntryPrice   float64
	CurrentPrice float64
	StopLoss     float64
	TakeProfit   float64
	MarginUsed   float64
	PnL          float64
	PnLPercent   float64
	OpenedAt     time.Time
	UpdatedAt    time.Time
}

type Kline struct {
	OpenTime  time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime time.Time
}

func (p *Position) UpdatePnL(currentPrice float64) {
	p.CurrentPrice = currentPrice
	p.UpdatedAt = time.Now()

	if p.Side == SideBuy {
		p.PnL = (p.CurrentPrice - p.EntryPrice) * p.Quantity
		p.PnLPercent = (p.CurrentPrice - p.EntryPrice) / p.EntryPrice * 100
	} else {
		p.PnL = (p.EntryPrice - p.CurrentPrice) * p.Quantity
		p.PnLPercent = (p.EntryPrice - p.CurrentPrice) / p.EntryPrice * 100
	}
}

func (p *Position) LiquidationPrice() float64 {
	if p.Side == SideBuy {
		return p.EntryPrice * (1 - 0.9)
	}
	return p.EntryPrice * (1 + 0.9)
}

func (p *Position) IsHealthy(healthThreshold float64) bool {
	return p.HealthScore() >= healthThreshold
}

func (p *Position) HealthScore() float64 {
	if p.PnLPercent > 0 {
		return 50 + p.PnLPercent*10
	}
	return 50 + p.PnLPercent*10
}

type MarketData struct {
	Symbol       string
	CurrentPrice float64
	High24h      float64
	Low24h       float64
	Volume24h    float64
	Volatility   float64
	RSI          float64
	EMAFast      float64
	EMASlow      float64
	Timestamp    time.Time
}

type Strategy interface {
	Name() string
	ShouldEnter(ctx context.Context, market MarketData) (bool, error)
	ShouldExit(ctx context.Context, position Position, market MarketData) (bool, error)
	CalculateSize(ctx context.Context, market MarketData, balance float64) (float64, error)
	CalculateStopLoss(ctx context.Context, entryPrice float64, market MarketData) (float64, error)
	CalculateTakeProfit(ctx context.Context, entryPrice float64, market MarketData) (float64, error)
}

type Executor interface {
	Execute(ctx context.Context, order *Order) (*Order, error)
	Cancel(ctx context.Context, orderID string) error
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	GetPosition(ctx context.Context, symbol string) (*Position, error)
	GetPositions(ctx context.Context) ([]*Position, error)
	GetBalance(ctx context.Context) (float64, error)
	ClosePosition(ctx context.Context, position *Position, reason string) error
}
