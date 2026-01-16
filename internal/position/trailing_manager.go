package position

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/sirupsen/logrus"
)

// TrailingConfig holds trailing stop loss and take profit configuration
type TrailingConfig struct {
	// Trailing Stop Loss
	TrailingStopEnabled   bool    `json:"trailing_stop_enabled"`
	TrailingStopPercent   float64 `json:"trailing_stop_percent"`   // Distance from current price (e.g., 0.003 = 0.3%)
	TrailingStopActivation float64 `json:"trailing_stop_activation"` // Profit threshold to activate trail (e.g., 0.005 = 0.5%)

	// Trailing Take Profit
	TrailingTakeProfitEnabled   bool    `json:"trailing_take_profit_enabled"`
	TrailingTakeProfitPercent   float64 `json:"trailing_take_profit_percent"`   // Distance from current price
	TrailingTakeProfitActivation float64 `json:"trailing_take_profit_activation"` // Profit threshold to activate trail

	// High Risk Tolerance Mode
	HighRiskMode  bool    `json:"high_risk_mode"`
	RiskMultiplier float64 `json:"risk_multiplier"` // Multiplier for risk parameters (e.g., 2.0 = double risk)
}

// DefaultTrailingConfig returns default trailing configuration
func DefaultTrailingConfig() TrailingConfig {
	return TrailingConfig{
		TrailingStopEnabled:      true,
		TrailingStopPercent:      0.002, // 0.2% trailing stop (tighter)
		TrailingStopActivation:   0.003, // Activate after 0.3% profit (sooner)
		TrailingTakeProfitEnabled: true,
		TrailingTakeProfitPercent: 0.002, // 0.2% trailing TP (tighter)
		TrailingTakeProfitActivation: 0.003, // Activate after 0.3% profit (sooner)
		HighRiskMode:  false,
		RiskMultiplier: 1.0,
	}
}

// HighRiskTrailingConfig returns high risk trailing configuration
func HighRiskTrailingConfig() TrailingConfig {
	return TrailingConfig{
		TrailingStopEnabled:      true,
		TrailingStopPercent:      0.001, // 0.1% trailing stop (very tight)
		TrailingStopActivation:   0.002, // Activate after 0.2% profit (very soon)
		TrailingTakeProfitEnabled: true,
		TrailingTakeProfitPercent: 0.001, // 0.1% trailing TP (very tight)
		TrailingTakeProfitActivation: 0.002, // Activate after 0.2% profit (very soon)
		HighRiskMode:  true,
		RiskMultiplier: 2.0, // Double risk tolerance
	}
}

// AggressiveAutonomousConfig returns aggressive autonomous trailing configuration
func AggressiveAutonomousConfig() TrailingConfig {
	return TrailingConfig{
		TrailingStopEnabled:      true,
		TrailingStopPercent:      0.01,  // 1% trailing stop
		TrailingStopActivation:   0.005, // Activate after 0.5% profit
		TrailingTakeProfitEnabled: true,
		TrailingTakeProfitPercent: 0.01,  // 1% trailing TP
		TrailingTakeProfitActivation: 0.005, // Activate after 0.5% profit
		HighRiskMode:  true,
		RiskMultiplier: 3.0, // Triple risk tolerance
	}
}

// TrailingPosition represents a position with trailing SL/TP
type TrailingPosition struct {
	Symbol              string
	Side                string
	EntryPrice          float64
	CurrentPrice        float64
	Quantity            float64
	OriginalStopLoss    float64
	OriginalTakeProfit  float64
	CurrentStopLoss     float64
	CurrentTakeProfit   float64
	StopLossOrderID     string
	TakeProfitOrderID   string
	TrailActivated      bool
	MaxUnrealizedPnL    float64
	MaxUnrealizedPnLPct  float64
	OpenedAt            time.Time
	LastTrailTime       time.Time
	TrailCount          int
}

// TrailingManager manages trailing stop loss and take profit
type TrailingManager struct {
	client         *futures.Client
	config         TrailingConfig
	positions      map[string]*TrailingPosition
	stopChan       chan struct{}
	isRunning      bool
	checkInterval  time.Duration
}

// NewTrailingManager creates a new trailing manager
func NewTrailingManager(client *futures.Client, config TrailingConfig) *TrailingManager {
	return &TrailingManager{
		client:        client,
		config:        config,
		positions:     make(map[string]*TrailingPosition),
		stopChan:      make(chan struct{}),
		checkInterval: 10 * time.Second, // Check every 10 seconds
	}
}

// Start begins trailing management
func (tm *TrailingManager) Start(ctx context.Context) error {
	logrus.Info("🎯 Starting trailing manager...")

	tm.isRunning = true

	// Take over existing positions
	if err := tm.takeOverPositions(ctx); err != nil {
		logrus.WithError(err).Warn("Failed to take over positions for trailing")
	}

	// Start monitoring loop
	go tm.monitorTrailing(ctx)

	logrus.Info("✅ Trailing manager started")
	return nil
}

// Stop gracefully stops the trailing manager
func (tm *TrailingManager) Stop() error {
	logrus.Info("🛑 Stopping trailing manager...")
	tm.isRunning = false
	close(tm.stopChan)
	return nil
}

// AddPosition adds a position to trailing management
func (tm *TrailingManager) AddPosition(symbol, side string, entryPrice, quantity, stopLoss, takeProfit float64, stopOrderID, tpOrderID string) {
	tm.positions[symbol] = &TrailingPosition{
		Symbol:             symbol,
		Side:               side,
		EntryPrice:         entryPrice,
		CurrentPrice:       entryPrice,
		Quantity:           quantity,
		OriginalStopLoss:   stopLoss,
		OriginalTakeProfit: takeProfit,
		CurrentStopLoss:    stopLoss,
		CurrentTakeProfit:  takeProfit,
		StopLossOrderID:    stopOrderID,
		TakeProfitOrderID:  tpOrderID,
		TrailActivated:     false,
		OpenedAt:           time.Now(),
		LastTrailTime:      time.Now(),
		TrailCount:         0,
	}

	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"side":       side,
		"entry":      entryPrice,
		"stop_loss":  stopLoss,
		"take_profit": takeProfit,
	}).Info("🎯 Position added to trailing management")
}

// RemovePosition removes a position from trailing management
func (tm *TrailingManager) RemovePosition(symbol string) {
	delete(tm.positions, symbol)
	logrus.WithField("symbol", symbol).Info("🎯 Position removed from trailing management")
}

// takeOverPositions takes ownership of existing positions
func (tm *TrailingManager) takeOverPositions(ctx context.Context) error {
	positions, err := tm.client.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	for _, pos := range positions {
		positionAmt := parseFloat(pos.PositionAmt)
		if positionAmt == 0 {
			continue
		}

		symbol := pos.Symbol
		side := "LONG"
		if positionAmt < 0 {
			side = "SHORT"
		}

		entryPrice := parseFloat(pos.EntryPrice)
		quantity := math.Abs(positionAmt)

		// Calculate initial SL/TP based on config
		var stopLoss, takeProfit float64
		if side == "LONG" {
			stopLoss = entryPrice * (1.0 - tm.getEffectiveStopLossPercent())
			takeProfit = entryPrice * (1.0 + tm.getEffectiveTakeProfitPercent())
		} else {
			stopLoss = entryPrice * (1.0 + tm.getEffectiveStopLossPercent())
			takeProfit = entryPrice * (1.0 - tm.getEffectiveTakeProfitPercent())
		}

		tm.AddPosition(symbol, side, entryPrice, quantity, stopLoss, takeProfit, "", "")
	}

	return nil
}

// monitorTrailing continuously monitors and updates trailing SL/TP
func (tm *TrailingManager) monitorTrailing(ctx context.Context) {
	ticker := time.NewTicker(tm.checkInterval)
	defer ticker.Stop()

	logrus.Info("🎯 Trailing monitoring loop started")

	for tm.isRunning {
		select {
		case <-ticker.C:
			if err := tm.updateTrailing(ctx); err != nil {
				logrus.WithError(err).Error("Failed to update trailing")
			}

		case <-tm.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// updateTrailing updates trailing stop loss and take profit for all positions
func (tm *TrailingManager) updateTrailing(ctx context.Context) error {
	for symbol, pos := range tm.positions {
		// Get current price
		prices, err := tm.client.NewListPricesService().Symbol(symbol).Do(ctx)
		if err != nil || len(prices) == 0 {
			continue
		}

		currentPrice := parseFloat(prices[0].Price)
		pos.CurrentPrice = currentPrice

		// Calculate unrealized PnL
		unrealizedPnL := tm.calculateUnrealizedPnL(pos, currentPrice)
		unrealizedPnLPct := (unrealizedPnL / (pos.EntryPrice * pos.Quantity)) * 100

		// Update max unrealized PnL
		if unrealizedPnL > pos.MaxUnrealizedPnL {
			pos.MaxUnrealizedPnL = unrealizedPnL
			pos.MaxUnrealizedPnLPct = unrealizedPnLPct
		}

		// Check if trail should be activated
		activationThreshold := tm.getEffectiveActivationThreshold()

		if !pos.TrailActivated && unrealizedPnLPct >= activationThreshold*100 {
			pos.TrailActivated = true
			logrus.WithFields(logrus.Fields{
				"symbol":   symbol,
				"pnl_pct":  unrealizedPnLPct,
				"threshold": activationThreshold * 100,
			}).Info("🎯 Trailing activated")
		}

		// Update trailing SL/TP if activated
		if pos.TrailActivated {
			tm.updateTrailingLevels(ctx, pos, currentPrice)
		}

		// Log position state
		logrus.WithFields(logrus.Fields{
			"symbol":           symbol,
			"current_price":    currentPrice,
			"unrealized_pnl":   unrealizedPnL,
			"unrealized_pnl_pct": unrealizedPnLPct,
			"max_unrealized_pnl": pos.MaxUnrealizedPnL,
			"trail_activated":   pos.TrailActivated,
			"current_sl":       pos.CurrentStopLoss,
			"current_tp":       pos.CurrentTakeProfit,
			"trail_count":      pos.TrailCount,
		}).Debug("🎯 Trailing position state")
	}

	return nil
}

// updateTrailingLevels updates trailing stop loss and take profit levels
func (tm *TrailingManager) updateTrailingLevels(ctx context.Context, pos *TrailingPosition, currentPrice float64) {
	var newStopLoss, newTakeProfit float64
	stopLossUpdated := false
	takeProfitUpdated := false

	// Update trailing stop loss
	if tm.config.TrailingStopEnabled {
		trailDistance := tm.getEffectiveTrailingStopPercent()

		if pos.Side == "LONG" {
			newStopLoss = currentPrice * (1.0 - trailDistance)
			// Only move stop loss up (for LONG)
			if newStopLoss > pos.CurrentStopLoss {
				pos.CurrentStopLoss = newStopLoss
				stopLossUpdated = true
			}
		} else {
			newStopLoss = currentPrice * (1.0 + trailDistance)
			// Only move stop loss down (for SHORT)
			if newStopLoss < pos.CurrentStopLoss {
				pos.CurrentStopLoss = newStopLoss
				stopLossUpdated = true
			}
		}
	}

	// Update trailing take profit
	if tm.config.TrailingTakeProfitEnabled {
		trailDistance := tm.getEffectiveTrailingTakeProfitPercent()

		if pos.Side == "LONG" {
			newTakeProfit = currentPrice * (1.0 + trailDistance)
			// Only move take profit up (for LONG)
			if newTakeProfit > pos.CurrentTakeProfit {
				pos.CurrentTakeProfit = newTakeProfit
				takeProfitUpdated = true
			}
		} else {
			newTakeProfit = currentPrice * (1.0 - trailDistance)
			// Only move take profit down (for SHORT)
			if newTakeProfit < pos.CurrentTakeProfit {
				pos.CurrentTakeProfit = newTakeProfit
				takeProfitUpdated = true
			}
		}
	}

	// Update orders if levels changed
	if stopLossUpdated || takeProfitUpdated {
		tm.updateOrders(ctx, pos, stopLossUpdated, takeProfitUpdated)
	}
}

// updateOrders updates stop loss and take profit orders
func (tm *TrailingManager) updateOrders(ctx context.Context, pos *TrailingPosition, stopLossUpdated, takeProfitUpdated bool) {
	// Cancel old orders if they exist
	if pos.StopLossOrderID != "" && stopLossUpdated {
		var orderID int64
		fmt.Sscanf(pos.StopLossOrderID, "%d", &orderID)
		_, err := tm.client.NewCancelOrderService().
			Symbol(pos.Symbol).
			OrderID(orderID).
			Do(ctx)
		if err != nil {
			logrus.WithError(err).Warn("Failed to cancel old stop loss order")
		}
	}

	if pos.TakeProfitOrderID != "" && takeProfitUpdated {
		var orderID int64
		fmt.Sscanf(pos.TakeProfitOrderID, "%d", &orderID)
		_, err := tm.client.NewCancelOrderService().
			Symbol(pos.Symbol).
			OrderID(orderID).
			Do(ctx)
		if err != nil {
			logrus.WithError(err).Warn("Failed to cancel old take profit order")
		}
	}

	// Place new orders
	var side futures.SideType
	if pos.Side == "LONG" {
		side = futures.SideTypeSell
	} else {
		side = futures.SideTypeBuy
	}

	if stopLossUpdated {
		order, err := tm.client.NewCreateOrderService().
			Symbol(pos.Symbol).
			Side(side).
			Type("STOP").
			Quantity(fmt.Sprintf("%.6f", pos.Quantity)).
			StopPrice(fmt.Sprintf("%.2f", pos.CurrentStopLoss)).
			Do(ctx)

		if err != nil {
			logrus.WithError(err).Error("Failed to place new stop loss order")
		} else {
			pos.StopLossOrderID = fmt.Sprintf("%d", order.OrderID)
			pos.TrailCount++
			pos.LastTrailTime = time.Now()

			logrus.WithFields(logrus.Fields{
				"symbol":     pos.Symbol,
				"order_id":   order.OrderID,
				"stop_price": pos.CurrentStopLoss,
				"trail_count": pos.TrailCount,
			}).Info("🎯 Stop loss order updated")
		}
	}

	if takeProfitUpdated {
		order, err := tm.client.NewCreateOrderService().
			Symbol(pos.Symbol).
			Side(side).
			Type("TAKE_PROFIT").
			Quantity(fmt.Sprintf("%.6f", pos.Quantity)).
			StopPrice(fmt.Sprintf("%.2f", pos.CurrentTakeProfit)).
			Do(ctx)

		if err != nil {
			logrus.WithError(err).Error("Failed to place new take profit order")
		} else {
			pos.TakeProfitOrderID = fmt.Sprintf("%d", order.OrderID)
			pos.TrailCount++
			pos.LastTrailTime = time.Now()

			logrus.WithFields(logrus.Fields{
				"symbol":   pos.Symbol,
				"order_id": order.OrderID,
				"tp_price": pos.CurrentTakeProfit,
				"trail_count": pos.TrailCount,
			}).Info("🎯 Take profit order updated")
		}
	}
}

// calculateUnrealizedPnL calculates unrealized PnL for a position
func (tm *TrailingManager) calculateUnrealizedPnL(pos *TrailingPosition, currentPrice float64) float64 {
	if pos.Side == "LONG" {
		return (currentPrice - pos.EntryPrice) * pos.Quantity
	}
	return (pos.EntryPrice - currentPrice) * pos.Quantity
}

// getEffectiveStopLossPercent returns effective stop loss percent based on config
func (tm *TrailingManager) getEffectiveStopLossPercent() float64 {
	if tm.config.HighRiskMode {
		return 0.01 * tm.config.RiskMultiplier // 1% * multiplier for high risk
	}
	return 0.005 // 0.5% for normal risk
}

// getEffectiveTakeProfitPercent returns effective take profit percent based on config
func (tm *TrailingManager) getEffectiveTakeProfitPercent() float64 {
	if tm.config.HighRiskMode {
		return 0.03 * tm.config.RiskMultiplier // 3% * multiplier for high risk
	}
	return 0.015 // 1.5% for normal risk
}

// getEffectiveTrailingStopPercent returns effective trailing stop percent
func (tm *TrailingManager) getEffectiveTrailingStopPercent() float64 {
	return tm.config.TrailingStopPercent * tm.config.RiskMultiplier
}

// getEffectiveTrailingTakeProfitPercent returns effective trailing take profit percent
func (tm *TrailingManager) getEffectiveTrailingTakeProfitPercent() float64 {
	return tm.config.TrailingTakeProfitPercent * tm.config.RiskMultiplier
}

// getEffectiveActivationThreshold returns effective activation threshold
func (tm *TrailingManager) getEffectiveActivationThreshold() float64 {
	if tm.config.HighRiskMode {
		return tm.config.TrailingStopActivation * 0.5 // Sooner activation for high risk
	}
	return tm.config.TrailingStopActivation
}
