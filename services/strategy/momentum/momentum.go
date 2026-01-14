package momentum

import (
	"context"

	"github.com/britebrt/cognee/domain/strategy"
	"github.com/britebrt/cognee/domain/trade"
)

type MomentumStrategy struct {
	cfg strategy.StrategyConfig
}

func (s *MomentumStrategy) Type() strategy.StrategyType {
	return strategy.StrategyMomentum
}

func (s *MomentumStrategy) Name() string {
	return "momentum_strategy"
}

func (s *MomentumStrategy) Version() string {
	return "1.0.0"
}

func (s *MomentumStrategy) Configure(config strategy.StrategyConfig) error {
	s.cfg = config
	return nil
}

func (s *MomentumStrategy) Validate() error {
	return nil
}

func (s *MomentumStrategy) ShouldEnter(ctx context.Context, market trade.MarketData) (bool, string, error) {
	if market.RSI > 50 && market.RSI < 80 {
		if market.EMAFast > market.EMASlow {
			if market.EMAFast > market.EMASlow*1.01 {
				return true, "Strong bullish momentum", nil
			}
		}
	}

	if market.RSI > 70 {
		return true, "Overbought but strong momentum", nil
	}

	return false, "No momentum signal", nil
}

func (s *MomentumStrategy) ShouldExit(ctx context.Context, position *trade.Position, market trade.MarketData) (bool, string, error) {
	if position.PnLPercent >= 3.0 {
		return true, "Take profit target reached", nil
	}

	if position.PnLPercent <= -1.5 {
		return true, "Stop loss triggered", nil
	}

	if market.RSI > 85 {
		return true, "Overbought - taking profits", nil
	}

	if market.EMAFast < market.EMASlow {
		return true, "Trend reversal detected", nil
	}

	return false, "", nil
}

func (s *MomentumStrategy) CalculatePositionSize(ctx context.Context, market trade.MarketData, balance float64) (float64, error) {
	riskPerTrade := s.cfg.RiskParameters.RiskPerTrade
	stopLossPercent := s.cfg.RiskParameters.StopLossPercent * 1.5

	riskAmount := balance * riskPerTrade
	stopLossDistance := market.CurrentPrice * stopLossPercent
	positionSize := riskAmount / stopLossDistance

	return positionSize, nil
}

func (s *MomentumStrategy) CalculateStopLoss(ctx context.Context, entryPrice float64, market trade.MarketData) (float64, error) {
	stopLossPercent := s.cfg.RiskParameters.StopLossPercent * 1.5
	return entryPrice * (1 - stopLossPercent), nil
}

func (s *MomentumStrategy) CalculateTakeProfit(ctx context.Context, entryPrice float64, market trade.MarketData) (float64, error) {
	takeProfitPercent := s.cfg.RiskParameters.TakeProfitPercent * 2
	return entryPrice * (1 + takeProfitPercent), nil
}

func (s *MomentumStrategy) CalculateTrailingStop(ctx context.Context, position *trade.Position, market trade.MarketData) (float64, error) {
	trailingPercent := 0.5
	if position.Side == trade.SideBuy {
		return market.CurrentPrice * (1 - trailingPercent), nil
	}
	return market.CurrentPrice * (1 + trailingPercent), nil
}

func (s *MomentumStrategy) OnTick(ctx context.Context, position *trade.Position, market trade.MarketData) error {
	return nil
}

func (s *MomentumStrategy) OnOrderFill(ctx context.Context, order *trade.Order, position *trade.Position) error {
	return nil
}

func (s *MomentumStrategy) OnPositionClose(ctx context.Context, position *trade.Position, reason string) error {
	return nil
}

func (s *MomentumStrategy) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"risk_per_trade":      s.cfg.RiskParameters.RiskPerTrade,
		"stop_loss_percent":   s.cfg.RiskParameters.StopLossPercent * 1.5,
		"take_profit_percent": s.cfg.RiskParameters.TakeProfitPercent * 2,
	}
}
