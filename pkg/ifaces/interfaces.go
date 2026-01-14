package ifaces

import (
	"context"
	"github.com/britebrt/cognee/domain/asset"
)

type Scanner interface {
	Scan(ctx context.Context) ([]asset.Asset, error)
	Criteria() asset.Criteria
	SetCriteria(criteria asset.Criteria)
}

type Executor interface {
	Execute(ctx context.Context, order interface{}) (interface{}, error)
	Cancel(ctx context.Context, orderID string) error
	GetOrder(ctx context.Context, orderID string) (interface{}, error)
	GetPosition(ctx context.Context, symbol string) (interface{}, error)
	GetPositions(ctx context.Context) ([]interface{}, error)
	GetBalance(ctx context.Context) (float64, error)
	ClosePosition(ctx context.Context, position interface{}, reason string) error
}

type Analyzer interface {
	Analyze(ctx context.Context, data interface{}) (*AnalysisResult, error)
}

type AnalysisResult struct {
	Action       string
	Symbol       string
	PositionSize float64
	EntryPrice   float64
	StopLoss     float64
	TakeProfit   float64
	Confidence   float64
	Reasoning    string
}

type Monitor interface {
	Start(ctx context.Context) error
	Stop() error
	Health(ctx context.Context, position interface{}) (float64, string, error)
}

type Scheduler interface {
	Start(ctx context.Context) error
	Stop() error
	Schedule(task interface{}) error
}
