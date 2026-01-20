package platform

import (
	"context"
	"time"

	"github.com/britej3/gobot/domain/automation"
	"github.com/britej3/gobot/domain/executor"
	"github.com/britej3/gobot/domain/selector"
	"github.com/britej3/gobot/domain/strategy"
	"github.com/britej3/gobot/domain/trade"
)

type Platform struct {
	Cfg        PlatformConfig
	Engine     *PlatformEngine
	Components *Components
	stopCh     chan struct{}
}

type PlatformConfig struct {
	Name             string                      `json:"name"`
	Version          string                      `json:"version"`
	Environment      string                      `json:"environment"`
	StrategyConfig   strategy.StrategyConfig     `json:"strategy_config"`
	SelectorConfig   selector.SelectorConfig     `json:"selector_config"`
	ExecutorConfig   executor.ExecutionConfig    `json:"executor_config"`
	AutomationConfig automation.AutomationConfig `json:"automation_config"`
	RiskConfig       RiskConfig                  `json:"risk_config"`
	Notifications    NotificationConfig          `json:"notifications"`
	Logging          LoggingConfig               `json:"logging"`
}

type RiskConfig struct {
	MaxDailyLoss      float64 `json:"max_daily_loss"`
	MaxPositionSize   float64 `json:"max_position_size"`
	MaxLeverage       float64 `json:"max_leverage"`
	StopOutPercentage float64 `json:"stop_out_percentage"`
}

type NotificationConfig struct {
	Enabled  bool                  `json:"enabled"`
	Channels []NotificationChannel `json:"channels"`
}

type NotificationChannel struct {
	Type    string `json:"type"`
	Webhook string `json:"webhook"`
}

type LoggingConfig struct {
	Level  string `json:"level"`
	Output string `json:"output"`
	Format string `json:"format"`
}

type Components struct {
	MarketDataProvider MarketDataProvider
	Strategy           strategy.Strategy
	Selector           selector.Selector
	Executor           executor.Executor
	Automation         automation.Automation
	PositionManager    PositionManager
	RiskManager        RiskManager
	Notifier           Notifier
}

type PlatformEngine struct {
	strategies  map[strategy.StrategyType]strategy.StrategyFactory
	selectors   map[selector.SelectorType]selector.SelectorFactory
	executors   map[executor.ExecutionType]executor.ExecutorFactory
	automations map[automation.AutomationType]automation.AutomationFactory
}

func NewPlatformEngine() *PlatformEngine {
	return &PlatformEngine{
		strategies:  make(map[strategy.StrategyType]strategy.StrategyFactory),
		selectors:   make(map[selector.SelectorType]selector.SelectorFactory),
		executors:   make(map[executor.ExecutionType]executor.ExecutorFactory),
		automations: make(map[automation.AutomationType]automation.AutomationFactory),
	}
}

func (e *PlatformEngine) RegisterStrategy(t strategy.StrategyType, factory strategy.StrategyFactory) {
	e.strategies[t] = factory
}

func (e *PlatformEngine) RegisterSelector(t selector.SelectorType, factory selector.SelectorFactory) {
	e.selectors[t] = factory
}

func (e *PlatformEngine) RegisterExecutor(t executor.ExecutionType, factory executor.ExecutorFactory) {
	e.executors[t] = factory
}

func (e *PlatformEngine) RegisterAutomation(t automation.AutomationType, factory automation.AutomationFactory) {
	e.automations[t] = factory
}

func (p *Platform) Initialize(ctx context.Context) error {
	var err error

	p.Components.Strategy, err = p.Engine.CreateStrategy(p.Cfg.StrategyConfig)
	if err != nil {
		return err
	}

	p.Components.Selector, err = p.Engine.CreateSelector(p.Cfg.SelectorConfig)
	if err != nil {
		return err
	}

	p.Components.Executor, err = p.Engine.CreateExecutor(p.Cfg.ExecutorConfig)
	if err != nil {
		return err
	}

	p.Components.Automation, err = p.Engine.CreateAutomation(p.Cfg.AutomationConfig)
	if err != nil {
		return err
	}

	return nil
}

func (p *Platform) Start(ctx context.Context) error {
	if err := p.Components.Strategy.Configure(p.Cfg.StrategyConfig); err != nil {
		return err
	}

	if err := p.Components.Selector.Configure(p.Cfg.SelectorConfig); err != nil {
		return err
	}

	if err := p.Components.Executor.Configure(p.Cfg.ExecutorConfig); err != nil {
		return err
	}

	if err := p.Components.Automation.Configure(p.Cfg.AutomationConfig); err != nil {
		return err
	}

	if err := p.Components.Automation.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (p *Platform) Stop() error {
	p.Components.Automation.Stop()
	return nil
}

func (p *Platform) RunCycle(ctx context.Context) error {
	marketData, err := p.Components.MarketDataProvider.GetAllMarketData(ctx)
	if err != nil {
		return err
	}

	selectedAssets, err := p.Components.Selector.Select(ctx, marketData)
	if err != nil {
		return err
	}

	for _, asset := range selectedAssets {
		market, ok := marketData[asset.Symbol]
		if !ok {
			continue
		}

		shouldEnter, reason, err := p.Components.Strategy.ShouldEnter(ctx, *market)
		if err != nil {
			continue
		}

		if shouldEnter {
			positionSize, _ := p.Components.Strategy.CalculatePositionSize(ctx, *market, 0)
			stopLoss, _ := p.Components.Strategy.CalculateStopLoss(ctx, market.CurrentPrice, *market)
			takeProfit, _ := p.Components.Strategy.CalculateTakeProfit(ctx, market.CurrentPrice, *market)

			result := strategy.StrategyResult{
				ShouldEnter:  true,
				Reason:       reason,
				PositionSize: positionSize,
				StopLoss:     stopLoss,
				TakeProfit:   takeProfit,
			}

			order, err := p.Components.Executor.Execute(ctx, result, *market)
			if err != nil {
				continue
			}

			p.Components.Automation.Execute(ctx, automation.EventData{
				Type:      "trade_signal",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"signal": result,
					"order":  order,
					"market": market,
				},
			})
		}
	}

	return nil
}

func (p *Platform) UpdateStrategy(config strategy.StrategyConfig) error {
	p.Cfg.StrategyConfig = config
	return p.Components.Strategy.Configure(config)
}

func (p *Platform) UpdateSelector(config selector.SelectorConfig) error {
	p.Cfg.SelectorConfig = config
	return p.Components.Selector.Configure(config)
}

func (p *Platform) UpdateExecutor(config executor.ExecutionConfig) error {
	p.Cfg.ExecutorConfig = config
	return p.Components.Executor.Configure(config)
}

func (p *Platform) UpdateAutomation(config automation.AutomationConfig) error {
	p.Cfg.AutomationConfig = config
	return p.Components.Automation.Configure(config)
}

func (e *PlatformEngine) CreateStrategy(cfg strategy.StrategyConfig) (strategy.Strategy, error) {
	factory, ok := e.strategies[cfg.Type]
	if !ok {
		return nil, strategy.ErrUnknownStrategy
	}
	s := factory()
	return s, s.Configure(cfg)
}

func (e *PlatformEngine) CreateSelector(cfg selector.SelectorConfig) (selector.Selector, error) {
	factory, ok := e.selectors[cfg.Type]
	if !ok {
		return nil, selector.ErrUnknownSelector
	}
	s := factory()
	return s, s.Configure(cfg)
}

func (e *PlatformEngine) CreateExecutor(cfg executor.ExecutionConfig) (executor.Executor, error) {
	factory, ok := e.executors[cfg.Type]
	if !ok {
		return nil, executor.ErrUnknownExecutor
	}
	s := factory()
	return s, s.Configure(cfg)
}

func (e *PlatformEngine) CreateAutomation(cfg automation.AutomationConfig) (automation.Automation, error) {
	factory, ok := e.automations[cfg.Type]
	if !ok {
		return nil, automation.ErrUnknownAutomation
	}
	s := factory()
	return s, s.Configure(cfg)
}

type MarketDataProvider interface {
	GetMarketData(ctx context.Context, symbol string) (*trade.MarketData, error)
	GetAllMarketData(ctx context.Context) (map[string]*trade.MarketData, error)
	Subscribe(symbol string, handler func(*trade.MarketData)) error
	Unsubscribe(symbol string) error
}

type PositionManager interface {
	OpenPosition(ctx context.Context, order *trade.Order) (*trade.Position, error)
	ClosePosition(ctx context.Context, position *trade.Position, reason string) error
	GetPosition(ctx context.Context, symbol string) (*trade.Position, error)
	GetPositions(ctx context.Context) ([]*trade.Position, error)
	UpdatePosition(ctx context.Context, position *trade.Position) error
}

type RiskManager interface {
	CalculateRisk(ctx context.Context, position *trade.Position) (float64, error)
	CheckLimits(ctx context.Context, position *trade.Position) error
	CalculatePositionSize(ctx context.Context, market trade.MarketData) (float64, error)
}

type Notifier interface {
	Send(ctx context.Context, message string, channel string) error
}
