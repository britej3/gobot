package scalper

import (
	"context"

	"github.com/britebrt/cognee/domain/strategy"
	"github.com/britebrt/cognee/domain/trade"
)

type ScalperStrategy struct {
	cfg strategy.StrategyConfig
}

func (s *ScalperStrategy) Type() strategy.StrategyType {
	return strategy.StrategyScalper
}

func (s *ScalperStrategy) Name() string {
	return "scalper_strategy"
}

func (s *ScalperStrategy) Version() string {
	return "1.0.0"
}

func (s *ScalperStrategy) Configure(config strategy.StrategyConfig) error {
	s.cfg = config
	return nil
}

func (s *ScalperStrategy) Validate() error {
	return nil
}

func (s *ScalperStrategy) ShouldEnter(ctx context.Context, market trade.MarketData) (bool, string, error) {
	if market.RSI > 30 && market.RSI < 70 {
		if market.EMAFast > market.EMASlow {
			return true, "RSI in range with bullish trend", nil
		}
	}

	if market.Volatility > 0.5 && market.Volatility < 3.0 {
		return true, "Good volatility for scalping", nil
	}

	return false, "Conditions not met", nil
}

func (s *ScalperStrategy) ShouldExit(ctx context.Context, position *trade.Position, market trade.MarketData) (bool, string, error) {
	if position.PnLPercent >= 1.0 {
		return true, "Take profit target reached", nil
	}

	if position.PnLPercent <= -0.5 {
		return true, "Stop loss triggered", nil
	}

	return false, "", nil
}

func (s *ScalperStrategy) CalculatePositionSize(ctx context.Context, market trade.MarketData, balance float64) (float64, error) {
	riskPerTrade := s.cfg.RiskParameters.RiskPerTrade
	stopLossPercent := s.cfg.RiskParameters.StopLossPercent

	riskAmount := balance * riskPerTrade
	stopLossDistance := market.CurrentPrice * stopLossPercent
	positionSize := riskAmount / stopLossDistance

	return positionSize, nil
}

func (s *ScalperStrategy) CalculateStopLoss(ctx context.Context, entryPrice float64, market trade.MarketData) (float64, error) {
	stopLossPercent := s.cfg.RiskParameters.StopLossPercent
	return entryPrice * (1 - stopLossPercent), nil
}

func (s *ScalperStrategy) CalculateTakeProfit(ctx context.Context, entryPrice float64, market trade.MarketData) (float64, error) {
	takeProfitPercent := s.cfg.RiskParameters.TakeProfitPercent
	return entryPrice * (1 + takeProfitPercent), nil
}

func (s *ScalperStrategy) CalculateTrailingStop(ctx context.Context, position *trade.Position, market trade.MarketData) (float64, error) {
	trailingPercent := 0.3
	if position.Side == trade.SideBuy {
		return market.CurrentPrice * (1 - trailingPercent), nil
	}
	return market.CurrentPrice * (1 + trailingPercent), nil
}

func (s *ScalperStrategy) OnTick(ctx context.Context, position *trade.Position, market trade.MarketData) error {
	return nil
}

func (s *ScalperStrategy) OnOrderFill(ctx context.Context, order *trade.Order, position *trade.Position) error {
	return nil
}

func (s *ScalperStrategy) OnPositionClose(ctx context.Context, position *trade.Position, reason string) error {
	return nil
}

func (s *ScalperStrategy) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"risk_per_trade":      s.cfg.RiskParameters.RiskPerTrade,
		"stop_loss_percent":   s.cfg.RiskParameters.StopLossPercent,
		"take_profit_percent": s.cfg.RiskParameters.TakeProfitPercent,
	}
}
