package position

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britebrt/cognee/pkg/brain"
	"github.com/sirupsen/logrus"
)

// PositionManager monitors and manages open positions
type PositionManager struct {
	client    *futures.Client
	brain     *brain.BrainEngine
	stopChan  chan struct{}
	isRunning bool
}

// PositionState represents the state of an open position
type PositionState struct {
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"` // LONG or SHORT
	Quantity      float64   `json:"quantity"`
	EntryPrice    float64   `json:"entry_price"`
	CurrentPrice  float64   `json:"current_price"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	PnLPercent    float64   `json:"pnl_percent"`
	StopLoss      float64   `json:"stop_loss"`
	TakeProfit    float64   `json:"take_profit"`
	OpenedAt      time.Time `json:"opened_at"`
	HealthScore   float64   `json:"health_score"` // 0-100, lower = worse
	Reasoning     string    `json:"reasoning"`
}

// NewPositionManager creates a new position manager
func NewPositionManager(client *futures.Client, brain *brain.BrainEngine) *PositionManager {
	return &PositionManager{
		client:   client,
		brain:    brain,
		stopChan: make(chan struct{}),
	}
}

// Start begins position monitoring
func (pm *PositionManager) Start(ctx context.Context) error {
	logrus.Info("🛡️  Starting position manager...")

	pm.isRunning = true

	// Initial position takeover
	if err := pm.takeOverPositions(ctx); err != nil {
		logrus.WithError(err).Warn("Failed to take over positions, will retry")
	}

	// Start monitoring loop
	go pm.monitorPositions(ctx)

	logrus.Info("✅ Position manager started")
	return nil
}

// Stop gracefully stops the position manager
func (pm *PositionManager) Stop() error {
	logrus.Info("🛑 Stopping position manager...")

	pm.isRunning = false
	close(pm.stopChan)

	return nil
}

// takeOverPositions takes ownership of all open positions
func (pm *PositionManager) takeOverPositions(ctx context.Context) error {
	positions, err := pm.client.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	takenOver := 0
	for _, pos := range positions {
		positionAmt := parseFloat(pos.PositionAmt)
		if positionAmt == 0 {
			continue // No position
		}

		takenOver++
		logrus.WithFields(logrus.Fields{
			"symbol":       pos.Symbol,
			"position_amt": positionAmt,
			"entry_price":  pos.EntryPrice,
		}).Info("🛡️  Took over position")
	}

	if takenOver > 0 {
		logrus.WithField("count", takenOver).Info("🛡️  Took over open positions")
	}

	return nil
}

// monitorPositions continuously checks position health
func (pm *PositionManager) monitorPositions(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	logrus.Info("📊 Position monitoring loop started (every 30s)")

	for pm.isRunning {
		select {
		case <-ticker.C:
			if err := pm.checkAndManagePositions(ctx); err != nil {
				logrus.WithError(err).Error("Failed to check positions")
			}

		case <-pm.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// checkAndManagePositions checks all positions and manages them
func (pm *PositionManager) checkAndManagePositions(ctx context.Context) error {
	positions, err := pm.client.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	for _, pos := range positions {
		positionAmt := parseFloat(pos.PositionAmt)
		if positionAmt == 0 {
			continue // No position
		}

		// Analyze position
		state, err := pm.analyzePosition(ctx, pos)
		if err != nil {
			logrus.WithError(err).WithField("symbol", pos.Symbol).Warn("Failed to analyze position")
			continue
		}

		// Log position state
		pm.logPositionState(state)

		// Check if position should be closed
		if shouldClosePosition(state) {
			pm.closePosition(ctx, state, "Risk management triggered")
		}
	}

	return nil
}

// analyzePosition analyzes a position and returns its state
func (pm *PositionManager) analyzePosition(ctx context.Context, pos *futures.PositionRisk) (*PositionState, error) {
	symbol := pos.Symbol
	positionAmt := parseFloat(pos.PositionAmt)
	entryPrice := parseFloat(pos.EntryPrice)

	// Get current price
	prices, err := pm.client.NewListPricesService().Symbol(symbol).Do(ctx)
	if err != nil || len(prices) == 0 {
		return nil, fmt.Errorf("failed to get current price: %w", err)
	}
	currentPrice := parseFloat(prices[0].Price)

	// Determine side
	side := "LONG"
	if positionAmt < 0 {
		side = "SHORT"
	}

	// Calculate PnL
	var unrealizedPnL, pnlPercent float64
	if side == "LONG" {
		unrealizedPnL = (currentPrice - entryPrice) * math.Abs(positionAmt)
		pnlPercent = ((currentPrice - entryPrice) / entryPrice) * 100
	} else {
		unrealizedPnL = (entryPrice - currentPrice) * math.Abs(positionAmt)
		pnlPercent = ((entryPrice - currentPrice) / entryPrice) * 100
	}

	// Calculate stop loss and take profit based on entry
	var stopLoss, takeProfit float64
	if side == "LONG" {
		stopLoss = entryPrice * 0.995   // 0.5% stop
		takeProfit = entryPrice * 1.015 // 1.5% target
	} else {
		stopLoss = entryPrice * 1.005   // 0.5% stop
		takeProfit = entryPrice * 0.985 // 1.5% target
	}

	// Get AI health assessment
	healthScore, reasoning, err := pm.assessPositionHealth(ctx, symbol, side, currentPrice, entryPrice, pnlPercent)
	if err != nil {
		logrus.WithError(err).Warn("Failed to get AI health assessment, using default")
		healthScore = 50 // Neutral
		reasoning = "AI unavailable"
	}

	return &PositionState{
		Symbol:        symbol,
		Side:          side,
		Quantity:      math.Abs(positionAmt),
		EntryPrice:    entryPrice,
		CurrentPrice:  currentPrice,
		UnrealizedPnL: unrealizedPnL,
		PnLPercent:    pnlPercent,
		StopLoss:      stopLoss,
		TakeProfit:    takeProfit,
		OpenedAt:      time.Now(), // Simplified - should get from trade history
		HealthScore:   healthScore,
		Reasoning:     reasoning,
	}, nil
}

// assessPositionHealth uses AI to assess if position is likely to win or lose
func (pm *PositionManager) assessPositionHealth(ctx context.Context, symbol, side string, currentPrice, entryPrice, pnlPercent float64) (float64, string, error) {
	// Get market trend data
	trendDirection, trendStrength, err := pm.getMarketTrend(ctx, symbol)
	if err != nil {
		logrus.WithError(err).Debug("Failed to get market trend, using neutral")
		trendDirection = "NEUTRAL"
		trendStrength = 0
	}

	// Calculate how much price has moved against/with position
	priceMovement := 0.0
	if side == "LONG" {
		priceMovement = ((currentPrice - entryPrice) / entryPrice) * 100
	} else {
		priceMovement = ((entryPrice - currentPrice) / entryPrice) * 100
	}

	// Prepare market data for AI - focus on PREDICTING outcome
	marketData := map[string]interface{}{
		"symbol":                 symbol,
		"position_side":          side,
		"current_price":          currentPrice,
		"entry_price":            entryPrice,
		"price_movement_percent": priceMovement,
		"market_trend":           trendDirection,
		"trend_strength":         trendStrength,
		"market_regime":          "VOLATILE",
		"task":                   "Assess if position will WIN or LOSE in next 10 minutes",
	}

	// Get AI prediction of position outcome
	decision, err := pm.brain.MakeTradingDecision(ctx, marketData)
	if err != nil {
		return 50, "AI unavailable", err
	}

	// Calculate win probability based on AI decision
	winProbability := 0.50 // 50% = neutral

	if side == "LONG" {
		// For LONG position: BUY = good, SELL = bad
		if decision.Decision == "BUY" {
			winProbability = 0.5 + (decision.Confidence * 0.4) // 50-90%
		} else if decision.Decision == "SELL" {
			winProbability = 0.5 - (decision.Confidence * 0.4) // 10-50%
		}
	} else {
		// For SHORT position: SELL = good, BUY = bad
		if decision.Decision == "SELL" {
			winProbability = 0.5 + (decision.Confidence * 0.4) // 50-90%
		} else if decision.Decision == "BUY" {
			winProbability = 0.5 - (decision.Confidence * 0.4) // 10-50%
		}
	}

	// Adjust based on trend alignment
	if side == "LONG" && trendDirection == "BULLISH" && trendStrength > 50 {
		winProbability += 0.10 // Boost for favorable trend
	} else if side == "LONG" && trendDirection == "BEARISH" && trendStrength > 50 {
		winProbability -= 0.10 // Reduce for unfavorable trend
	} else if side == "SHORT" && trendDirection == "BEARISH" && trendStrength > 50 {
		winProbability += 0.10 // Boost for favorable trend
	} else if side == "SHORT" && trendDirection == "BULLISH" && trendStrength > 50 {
		winProbability -= 0.10 // Reduce for unfavorable trend
	}

	// Clamp to 0-1
	if winProbability < 0 {
		winProbability = 0
	} else if winProbability > 1 {
		winProbability = 1
	}

	// Health score is win probability * 100
	healthScore := winProbability * 100

	return healthScore, fmt.Sprintf("Win probability: %.1f%% - %s", winProbability*100, decision.Reasoning), nil
}

// getMarketTrend analyzes market trend for a symbol
func (pm *PositionManager) getMarketTrend(ctx context.Context, symbol string) (string, float64, error) {
	// Get kline data for trend analysis
	klines, err := pm.client.NewKlinesService().
		Symbol(symbol).
		Interval("5m").
		Limit(20).
		Do(ctx)

	if err != nil || len(klines) < 5 {
		return "NEUTRAL", 0, nil
	}

	// Calculate simple moving averages
	var prices []float64
	for _, k := range klines {
		prices = append(prices, parseFloat(k.Close))
	}

	// Calculate trend based on recent price action
	recentAvg := 0.0
	for i := len(prices) - 5; i < len(prices); i++ {
		recentAvg += prices[i]
	}
	recentAvg /= 5

	olderAvg := 0.0
	for i := 0; i < 5; i++ {
		olderAvg += prices[i]
	}
	olderAvg /= 5

	// Determine trend direction and strength
	priceChange := ((recentAvg - olderAvg) / olderAvg) * 100
	trendStrength := math.Abs(priceChange) * 10 // Convert to 0-100 range

	if trendStrength > 100 {
		trendStrength = 100
	}

	if priceChange > 0.1 {
		return "BULLISH", trendStrength, nil
	} else if priceChange < -0.1 {
		return "BEARISH", trendStrength, nil
	}

	return "NEUTRAL", trendStrength, nil
}

// shouldClosePosition determines if a position should be closed
func shouldClosePosition(state *PositionState) bool {
	// Rule 1: Hard stop loss (0.5% loss)
	if state.PnLPercent < -0.5 {
		return true
	}

	// Rule 2: Take profit hit (1.5% gain)
	if state.PnLPercent > 1.5 {
		return true
	}

	// Rule 3: AI predicts higher chance of losing than winning (health score < 45)
	if state.HealthScore < 45 {
		return true
	}

	// Rule 4: Loss of 0.2% + AI score < 50 (loss + unfavorable prediction)
	if state.PnLPercent < -0.2 && state.HealthScore < 50 {
		return true
	}

	return false
}

// closePosition closes a position
func (pm *PositionManager) closePosition(ctx context.Context, state *PositionState, reason string) {
	logrus.WithFields(logrus.Fields{
		"symbol":       state.Symbol,
		"side":         state.Side,
		"pnl_percent":  state.PnLPercent,
		"pnl":          state.UnrealizedPnL,
		"health_score": state.HealthScore,
		"reason":       reason,
	}).Warn("⚠️  Closing position")

	var side futures.SideType
	if state.Side == "LONG" {
		side = futures.SideTypeSell // Close LONG with SELL
	} else {
		side = futures.SideTypeBuy // Close SHORT with BUY
	}

	// Place market order to close
	_, err := pm.client.NewCreateOrderService().
		Symbol(state.Symbol).
		Side(side).
		Type(futures.OrderTypeMarket).
		Quantity(fmt.Sprintf("%.6f", state.Quantity)).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to close position")
		return
	}

	logrus.WithFields(logrus.Fields{
		"symbol":      state.Symbol,
		"side":        state.Side,
		"quantity":    state.Quantity,
		"pnl":         state.UnrealizedPnL,
		"pnl_percent": state.PnLPercent,
		"reason":      reason,
	}).Info("✅ Position closed")
}

// logPositionState logs the current state of a position
func (pm *PositionManager) logPositionState(state *PositionState) {
	logrus.WithFields(logrus.Fields{
		"symbol":        state.Symbol,
		"side":          state.Side,
		"current_price": state.CurrentPrice,
		"entry_price":   state.EntryPrice,
		"pnl_percent":   state.PnLPercent,
		"health_score":  state.HealthScore,
	}).Debug("📊 Position state")
}

// parseFloat safely parses a string to float64
func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
