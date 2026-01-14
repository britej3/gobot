package striker

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britebrt/cognee/internal/platform"
	"github.com/britebrt/cognee/pkg/brain"
	"github.com/sirupsen/logrus"
)

// Striker executes trading decisions with precision and risk management
type Striker struct {
	client    *futures.Client
	brain     *brain.BrainEngine
	isRunning bool
}

// NewStriker creates a new trading striker
func NewStriker(client *futures.Client, brain *brain.BrainEngine) *Striker {
	return &Striker{
		client: client,
		brain:  brain,
	}
}

// Execute performs real striker analysis and trade execution
func (s *Striker) Execute(ctx context.Context, topAssets []interface{}) (*brain.StrikerDecision, error) {
	if len(topAssets) == 0 {
		return &brain.StrikerDecision{
			Timestamp:    time.Now().Format(time.RFC3339),
			TopTargets:   []brain.TargetAsset{},
			MarketRegime: "RANGING",
		}, nil
	}

	// Select the top asset as a target
	topAsset := topAssets[0]

	// Use reflection to extract fields from ScoredAsset (avoiding import cycle with watcher package)
	var symbol string
	var currentPrice float64
	var confidence float64

	// Use reflection to access struct fields (ScoredAsset from watcher package)
	v := reflect.ValueOf(topAsset)
	if v.Kind() == reflect.Struct {
		// Try to get Symbol field
		if symbolField := v.FieldByName("Symbol"); symbolField.IsValid() && symbolField.Kind() == reflect.String {
			symbol = symbolField.String()
		}
		// Try to get CurrentPrice field
		if priceField := v.FieldByName("CurrentPrice"); priceField.IsValid() && priceField.Kind() == reflect.Float64 {
			currentPrice = priceField.Float()
		}
		// Try to get Confidence field
		if confField := v.FieldByName("Confidence"); confField.IsValid() && confField.Kind() == reflect.Float64 {
			confidence = confField.Float()
		}

		if symbol != "" && currentPrice > 0 {
			logrus.WithFields(logrus.Fields{
				"symbol":     symbol,
				"price":      currentPrice,
				"confidence": confidence,
			}).Info("🎯 Processing ScoredAsset from scanner")
		} else {
			logrus.WithField("type", fmt.Sprintf("%T", topAsset)).Warn("Unknown asset type, skipping")
			return &brain.StrikerDecision{
				Timestamp:    time.Now().Format(time.RFC3339),
				TopTargets:   []brain.TargetAsset{},
				MarketRegime: "RANGING",
			}, nil
		}
	} else if assetMap, ok := topAsset.(map[string]interface{}); ok {
		// Fallback: Try map-based approach
		if sym, ok := assetMap["Symbol"].(string); ok {
			symbol = sym
		}
		if price, ok := assetMap["CurrentPrice"].(float64); ok {
			currentPrice = price
		}
		if conf, ok := assetMap["Confidence"].(float64); ok {
			confidence = conf
		}
		logrus.WithField("symbol", symbol).Info("🎯 Processing asset from map")
	} else {
		// Log the actual type for debugging
		logrus.WithField("type", fmt.Sprintf("%T", topAsset)).Warn("Unknown asset type, skipping")
		return &brain.StrikerDecision{
			Timestamp:    time.Now().Format(time.RFC3339),
			TopTargets:   []brain.TargetAsset{},
			MarketRegime: "RANGING",
		}, nil
	}

	// Get market conditions for the asset
	hasPosition := s.checkPosition(ctx, symbol)

	// Get kline data for volatility calculation
	klines, err := s.client.NewKlinesService().
		Symbol(symbol).
		Interval("5m").
		Limit(50).
		Do(ctx)

	volatility := 0.02
	volumeSpike := false
	if err == nil && len(klines) > 1 {
		// Calculate volatility
		prices := make([]float64, len(klines))
		for i, k := range klines {
			prices[i] = parseFloat(k.Close)
		}

		// Simple volatility calculation (standard deviation)
		sum := 0.0
		for _, price := range prices {
			sum += price
		}
		mean := sum / float64(len(prices))
		sumSqDiff := 0.0
		for _, price := range prices {
			diff := price - mean
			sumSqDiff += diff * diff
		}
		stdDev := (sumSqDiff / float64(len(prices)))
		volatility = (stdDev / mean) * 100

		// Volume spike detection (last 3 candles vs average)
		if len(klines) >= 4 {
			recentVol := 0.0
			for i := len(klines) - 4; i < len(klines); i++ {
				recentVol += parseFloat(klines[i].Volume)
			}
			avgVol := 0.0
			for i := 0; i < len(klines)-4; i++ {
				avgVol += parseFloat(klines[i].Volume)
			}
			avgVol = avgVol / float64(len(klines)-4)
			volumeSpike = recentVol/3 > avgVol*1.5
		}
	}

	// Get 24h ticker for additional context
	tickerInfo, _ := s.client.NewListPriceChangeStatsService().Symbol(symbol).Do(ctx)
	fvgConfidence := confidence
	cvDivergence := false

	if len(tickerInfo) > 0 {
		priceChangePercent := parseFloat(tickerInfo[0].PriceChangePercent)
		if priceChangePercent > 2 || priceChangePercent < -2 {
			cvDivergence = true
		}
	}

	markets := map[string]interface{}{
		"symbol":         symbol,
		"current_price":  currentPrice,
		"position":       hasPosition,
		"timestamp":      time.Now(),
		"volatility":     volatility,
		"volume_spike":   volumeSpike,
		"price_action":   "NEUTRAL",
		"fvg_confidence": fvgConfidence,
		"cvd_divergence": cvDivergence,
		"market_regime":  "VOLATILE",
	}

	// Query AI for trading decision
	decision, err := s.brain.MakeTradingDecision(ctx, markets)
	if err != nil {
		return nil, fmt.Errorf("brain decision failed: %w", err)
	}

	// Execute trade if confidence is high (0.0-1.0 scale)
	// Lowered to 0.65 for aggressive scalping
	if decision.Confidence > 0.65 && (decision.Decision == "BUY" || decision.Decision == "SELL") {
		logrus.WithFields(logrus.Fields{
			"symbol":     symbol,
			"decision":   decision.Decision,
			"confidence": decision.Confidence,
		}).Info("🎯 High confidence signal - executing trade")

		s.executeDecision(ctx, symbol, decision)

		// Create target for response
		action := "LONG"
		if decision.Decision == "SELL" {
			action = "SHORT"
		}

		target := brain.TargetAsset{
			Symbol:               symbol,
			Action:               action,
			ConfidenceScore:      decision.Confidence * 100, // Convert to percentage for display
			ProbabilityReason:    decision.Reasoning,
			EntryZone:            currentPrice,
			TakeProfit:           currentPrice * 1.015,
			StopLoss:             currentPrice * 0.995,
			AllocationMultiplier: float64(decision.RecommendedLeverage) / 25.0,
		}

		return &brain.StrikerDecision{
			Timestamp:    time.Now().Format(time.RFC3339),
			TopTargets:   []brain.TargetAsset{target},
			MarketRegime: "VOLATILE_EXPANSION",
		}, nil
	}

	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"decision":   decision.Decision,
		"confidence": decision.Confidence,
	}).Debug("Signal below threshold or HOLD - skipping execution")

	return &brain.StrikerDecision{
		Timestamp:    time.Now().Format(time.RFC3339),
		TopTargets:   []brain.TargetAsset{},
		MarketRegime: "RANGING",
	}, nil
}

// Check if position already exists
func (s *Striker) checkPosition(ctx context.Context, symbol string) map[string]interface{} {
	positions, err := s.client.NewGetPositionRiskService().
		Symbol(symbol).
		Do(ctx)

	if err != nil || len(positions) == 0 {
		return nil
	}

	pos := positions[0]
	positionAmt := parseFloat(pos.PositionAmt)
	if positionAmt == 0 {
		return nil
	}

	return map[string]interface{}{
		"amount": positionAmt,
		"entry":  parseFloat(pos.EntryPrice),
		"side":   getPositionSide(positionAmt),
	}
}

// Start begins trade execution
func (s *Striker) Start(ctx context.Context) error {
	logrus.Info("⚡ Starting trading striker...")

	s.isRunning = true

	// Start listening for trading signals
	go s.processTradingSignals(ctx)

	logrus.Info("✅ Trading striker started")
	return nil
}

// Stop gracefully stops the striker
func (s *Striker) Stop() error {
	logrus.Info("🛑 Stopping trading striker...")
	s.isRunning = false
	return nil
}

func (s *Striker) processTradingSignals(ctx context.Context) {
	logrus.Info("📡 Listening for trading signals...")

	// In a real implementation, this would connect to a message queue
	// For now, we'll simulate signal processing

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for s.isRunning {
		select {
		case <-ticker.C:
			// Simulate receiving a trading signal
			s.simulateTradingSignal(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Striker) simulateTradingSignal(ctx context.Context) {
	// This simulates receiving a trading signal from the watcher
	// In production, this would come from a message queue or direct call

	symbol := "ZECUSDT"

	// Get current market conditions
	marketConditions := s.getCurrentMarketConditions(ctx, symbol)
	if marketConditions == nil {
		return
	}

	// Get trading decision from brain
	decision, err := s.brain.MakeTradingDecision(ctx, marketConditions)
	if err != nil {
		logrus.WithError(err).Error("Failed to get trading decision")
		return
	}

	// Execute the decision
	s.executeDecision(ctx, symbol, decision)
}

func (s *Striker) getCurrentMarketConditions(ctx context.Context, symbol string) interface{} {
	// Get recent price data
	klines, err := s.client.NewKlinesService().
		Symbol(symbol).
		Interval("1m").
		Limit(10).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to get kline data")
		return nil
	}

	if len(klines) == 0 {
		return nil
	}

	// Calculate current conditions
	latestKline := klines[len(klines)-1]
	currentPrice := parseFloat(latestKline.Close)

	// Get position info
	positions, err := s.client.NewGetPositionRiskService().
		Symbol(symbol).
		Do(ctx)

	var currentPosition interface{}
	if err == nil && len(positions) > 0 {
		pos := positions[0]
		positionAmt := parseFloat(pos.PositionAmt)
		if positionAmt != 0 {
			currentPosition = map[string]interface{}{
				"amount": positionAmt,
				"entry":  parseFloat(pos.EntryPrice),
				"side":   getPositionSide(positionAmt),
			}
		}
	}

	return map[string]interface{}{
		"symbol":           symbol,
		"current_price":    currentPrice,
		"current_position": currentPosition,
		"timestamp":        time.Now(),
		"volatility":       0.02, // Would be calculated from klines
		"volume":           parseFloat(latestKline.Volume),
	}
}

func (s *Striker) executeDecision(ctx context.Context, symbol string, decision *brain.TradingDecision) {
	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"decision":   decision.Decision,
		"confidence": decision.Confidence,
		"leverage":   decision.RecommendedLeverage,
		"reasoning":  decision.Reasoning,
	}).Info("Executing trading decision")

	switch decision.Decision {
	case "BUY":
		s.ExecuteBuyOrder(ctx, symbol, decision)
	case "SELL":
		s.ExecuteSellOrder(ctx, symbol, decision)
	case "HOLD":
		logrus.WithField("symbol", symbol).Info("Holding position - no action taken")
	default:
		logrus.WithField("decision", decision.Decision).Error("Unknown trading decision")
	}
}

func (s *Striker) ExecuteBuyOrder(ctx context.Context, symbol string, decision *brain.TradingDecision) {
	// Calculate order parameters
	quantity := s.calculateOrderQuantity(decision.RecommendedLeverage)

	// Get current price for market order
	ticker, err := s.client.NewListPricesService().
		Symbol(symbol).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to get current price")
		return
	}

	if len(ticker) == 0 {
		logrus.Error("No price data received")
		return
	}

	currentPrice := parseFloat(ticker[0].Price)

	// Apply anti-sniffer jitter before order placement
	// Per reply_unknown.md technical specs: 5-25ms normal distribution
	logrus.Debug("🎲 Applying anti-sniffer jitter...")
	platform.ApplyJitter()

	// Place market buy order
	order, err := s.client.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeBuy).
		Type(futures.OrderTypeMarket).
		Quantity(fmt.Sprintf("%.6f", quantity)).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to place buy order")
		return
	}

	// Log successful order
	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"order_id":   order.OrderID,
		"quantity":   quantity,
		"price":      currentPrice,
		"leverage":   decision.RecommendedLeverage,
		"confidence": decision.Confidence,
	}).Info("Buy order executed successfully")

	// Set stop loss and take profit
	s.setRiskManagement(ctx, symbol, currentPrice, decision, "LONG")
}

func (s *Striker) ExecuteSellOrder(ctx context.Context, symbol string, decision *brain.TradingDecision) {
	// Calculate order parameters
	quantity := s.calculateOrderQuantity(decision.RecommendedLeverage)

	// Get current price for market order
	ticker, err := s.client.NewListPricesService().
		Symbol(symbol).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to get current price")
		return
	}

	if len(ticker) == 0 {
		logrus.Error("No price data received")
		return
	}

	currentPrice := parseFloat(ticker[0].Price)

	// Apply anti-sniffer jitter before order placement
	// Per reply_unknown.md technical specs: 5-25ms normal distribution
	logrus.Debug("🎲 Applying anti-sniffer jitter...")
	platform.ApplyJitter()

	// Place market sell order
	order, err := s.client.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeSell).
		Type(futures.OrderTypeMarket).
		Quantity(fmt.Sprintf("%.6f", quantity)).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to place sell order")
		return
	}

	// Log successful order
	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"order_id":   order.OrderID,
		"quantity":   quantity,
		"price":      currentPrice,
		"leverage":   decision.RecommendedLeverage,
		"confidence": decision.Confidence,
	}).Info("Sell order executed successfully")

	// Set stop loss and take profit
	s.setRiskManagement(ctx, symbol, currentPrice, decision, "SHORT")
}

func (s *Striker) calculateOrderQuantity(leverage int) float64 {
	// This is a simplified calculation
	// In production, this would consider:
	// - Account balance
	// - Risk per trade settings
	// - Current market conditions
	// - Position sizing rules

	// For demonstration, use a fixed quantity based on leverage
	baseQuantity := 0.001                          // Base quantity in BTC/ETH terms
	return baseQuantity * float64(leverage) / 25.0 // Normalize to max leverage
}

func (s *Striker) setRiskManagement(ctx context.Context, symbol string, entryPrice float64, decision *brain.TradingDecision, side string) {
	// Calculate stop loss and take profit levels
	var stopLoss, takeProfit float64

	if side == "LONG" {
		stopLoss = entryPrice * 0.995   // 0.5% stop loss
		takeProfit = entryPrice * 1.005 // 0.5% take profit
	} else {
		stopLoss = entryPrice * 1.005   // 0.5% stop loss
		takeProfit = entryPrice * 0.995 // 0.5% take profit
	}

	// Set stop loss order
	// Note: STOP and TAKE_PROFIT are string literals as they're not defined in OrderType constants
	stopOrder, err := s.client.NewCreateOrderService().
		Symbol(symbol).
		Side(getOppositeSide(side)).
		Type("STOP").
		Quantity(fmt.Sprintf("%.6f", s.calculateOrderQuantity(decision.RecommendedLeverage))).
		StopPrice(fmt.Sprintf("%.2f", stopLoss)).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to set stop loss order")
	} else {
		logrus.WithFields(logrus.Fields{
			"symbol":     symbol,
			"order_id":   stopOrder.OrderID,
			"stop_price": stopLoss,
			"type":       "stop_loss",
		}).Info("Stop loss order placed")
	}

	// Set take profit order
	tpOrder, err := s.client.NewCreateOrderService().
		Symbol(symbol).
		Side(getOppositeSide(side)).
		Type("TAKE_PROFIT").
		Quantity(fmt.Sprintf("%.6f", s.calculateOrderQuantity(decision.RecommendedLeverage))).
		StopPrice(fmt.Sprintf("%.2f", takeProfit)).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to set take profit order")
	} else {
		logrus.WithFields(logrus.Fields{
			"symbol":   symbol,
			"order_id": tpOrder.OrderID,
			"tp_price": takeProfit,
			"type":     "take_profit",
		}).Info("Take profit order placed")
	}
}

// Helper functions
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func getPositionSide(amt float64) string {
	if amt > 0 {
		return "LONG"
	}
	return "SHORT"
}

func getOppositeSide(side string) futures.SideType {
	if side == "LONG" {
		return futures.SideTypeSell
	}
	return futures.SideTypeBuy
}
