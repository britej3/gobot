package striker

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britebrt/cognee/internal/executor"
	"github.com/britebrt/cognee/internal/position"
	"github.com/britebrt/cognee/pkg/brain"
	"github.com/sirupsen/logrus"
)

// EnhancedStriker executes trading decisions with advanced features
type EnhancedStriker struct {
	client                 *futures.Client
	brain                  *brain.BrainEngine
	dynamicManager         *position.DynamicManager
	trailingManager        *position.TrailingManager
	selfOptimizingExecutor *executor.SelfOptimizingExecutor
	isRunning              bool
	highRiskMode           bool
}

// NewEnhancedStriker creates a new enhanced trading striker
func NewEnhancedStriker(
	client *futures.Client,
	brain *brain.BrainEngine,
	dynamicManager *position.DynamicManager,
	trailingManager *position.TrailingManager,
	highRiskMode bool,
) *EnhancedStriker {
	return &EnhancedStriker{
		client:                 client,
		brain:                  brain,
		dynamicManager:         dynamicManager,
		trailingManager:        trailingManager,
		selfOptimizingExecutor: executor.NewSelfOptimizingExecutor(client, nil),
		isRunning:              false,
		highRiskMode:           highRiskMode,
	}
}

// Execute performs enhanced striker analysis and trade execution
// ExecuteEnhancedTrade executes a trade with aggressive parameters
func (es *EnhancedStriker) ExecuteEnhancedTrade(
	ctx context.Context,
	symbol string,
	action string,
	positionSizeUSD float64,
	leverage int,
	entryPrice float64,
	takeProfit float64,
	stopLoss float64,
) error {
	logrus.WithFields(logrus.Fields{
		"symbol":         symbol,
		"action":         action,
		"position_size":  positionSizeUSD,
		"leverage":       leverage,
		"entry":          entryPrice,
		"tp":             takeProfit,
		"sl":             stopLoss,
	}).Info("🎯 Executing enhanced trade with self-optimization")

	// Set leverage
	if err := es.setLeverage(ctx, symbol, leverage); err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	// Calculate quantity
	quantity := positionSizeUSD / (entryPrice * float64(leverage))

	// Determine side
	var side futures.SideType
	if action == "BUY" {
		side = futures.SideTypeBuy
	} else {
		side = futures.SideTypeSell
	}

	// Use self-optimizing executor for order execution
	order, err := es.selfOptimizingExecutor.Execute(ctx, symbol, side, quantity, entryPrice)
	if err != nil {
		return fmt.Errorf("self-optimizing execution failed: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"symbol":    symbol,
		"order_id":  order.OrderID,
		"quantity":  quantity,
		"side":      side,
	}).Info("✅ Order executed successfully with self-optimization")

	// Set take profit and stop loss
	go es.setEnhancedRiskManagement(ctx, symbol, entryPrice, takeProfit, stopLoss, side)

	return nil
}

// setEnhancedRiskManagement sets take profit and stop loss orders
func (es *EnhancedStriker) setEnhancedRiskManagement(
	ctx context.Context,
	symbol string,
	entryPrice float64,
	takeProfit float64,
	stopLoss float64,
	side futures.SideType,
) {
	// Calculate TP and SL prices based on side
	var tpPrice, slPrice float64
	if side == futures.SideTypeBuy {
		tpPrice = entryPrice * (1 + takeProfit/100)
		slPrice = entryPrice * (1 - stopLoss/100)
	} else {
		tpPrice = entryPrice * (1 - takeProfit/100)
		slPrice = entryPrice * (1 + stopLoss/100)
	}

	// Get position quantity
	positions, err := es.client.NewGetPositionRiskService().Symbol(symbol).Do(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get position for risk management")
		return
	}

	if len(positions) == 0 {
		return
	}

	positionAmt, _ := strconv.ParseFloat(positions[0].PositionAmt, 64)
	if positionAmt == 0 {
		return
	}

	// Create take profit order
	_, err = es.client.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeSell).
		Type(futures.OrderTypeLimit).
		Price(fmt.Sprintf("%.8f", tpPrice)).
		Quantity(fmt.Sprintf("%.6f", math.Abs(positionAmt)*0.3)). // Take 30% at TP
		TimeInForce(futures.TimeInForceTypeGTC).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to create take profit order")
	}

	// Create stop loss order
	_, err = es.client.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeSell).
		Type("STOP").
		StopPrice(fmt.Sprintf("%.8f", slPrice)).
		Quantity(fmt.Sprintf("%.6f", math.Abs(positionAmt))).
		ClosePosition(true).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to create stop loss order")
	}

	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"tp_price":   tpPrice,
		"sl_price":   slPrice,
	}).Info("Risk management orders set")
}

func (es *EnhancedStriker) Execute(ctx context.Context, topAssets []interface{}) (*brain.StrikerDecision, error) {
	if len(topAssets) == 0 {
		return &brain.StrikerDecision{
			Timestamp:    time.Now().Format(time.RFC3339),
			TopTargets:   []brain.TargetAsset{},
			MarketRegime: "RANGING",
		}, nil
	}

	// Select the top asset as a target
	topAsset := topAssets[0]

	// Extract asset information
	symbol, currentPrice, confidence := es.extractAssetInfo(topAsset)
	if symbol == "" {
		return &brain.StrikerDecision{
			Timestamp:    time.Now().Format(time.RFC3339),
			TopTargets:   []brain.TargetAsset{},
			MarketRegime: "RANGING",
		}, nil
	}

	// Get market conditions
	hasPosition := es.checkPosition(ctx, symbol)

	// Get kline data for volatility calculation
	klines, err := es.client.NewKlinesService().
		Symbol(symbol).
		Interval("5m").
		Limit(50).
		Do(ctx)

	volatility := 0.02
	volumeSpike := false
	if err == nil && len(klines) > 1 {
		volatility = es.calculateVolatility(klines)
		volumeSpike = es.checkVolumeSpike(klines)
	}

	// Get 24h ticker for additional context
	tickerInfo, _ := es.client.NewListPriceChangeStatsService().Symbol(symbol).Do(ctx)
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
		"high_risk_mode": es.highRiskMode,
	}

	// Query AI for trading decision
	decision, err := es.brain.MakeTradingDecision(ctx, markets)
	if err != nil {
		return nil, fmt.Errorf("brain decision failed: %w", err)
	}

	// Execute trade if confidence is high
	threshold := 0.65
	if es.highRiskMode {
		threshold = 0.60 // Lower threshold for high risk mode
	}

	if decision.Confidence > threshold && (decision.Decision == "BUY" || decision.Decision == "SELL") {
		logrus.WithFields(logrus.Fields{
			"symbol":     symbol,
			"decision":   decision.Decision,
			"confidence": decision.Confidence,
			"volatility": volatility,
		}).Info("⚡ High confidence signal - executing enhanced trade")

		// Calculate dynamic position size and leverage
		stopLoss := es.calculateStopLoss(currentPrice, volatility, decision.Decision)
		quantity, leverage, err := es.dynamicManager.CalculatePositionSize(
			ctx,
			symbol,
			currentPrice,
			stopLoss,
			decision.Confidence,
			volatility,
		)

		if err != nil {
			logrus.WithError(err).Warn("Failed to calculate dynamic position size, using default")
			quantity = es.calculateDefaultQuantity(decision.RecommendedLeverage)
			leverage = decision.RecommendedLeverage
		}

		// Execute the trade
		es.executeEnhancedDecision(ctx, symbol, decision, quantity, leverage, currentPrice, volatility)

		// Create target for response
		action := "LONG"
		if decision.Decision == "SELL" {
			action = "SHORT"
		}

		takeProfit := es.calculateTakeProfit(currentPrice, volatility, decision.Decision)

		target := brain.TargetAsset{
			Symbol:               symbol,
			Action:               action,
			ConfidenceScore:      decision.Confidence * 100,
			ProbabilityReason:    decision.Reasoning,
			EntryZone:            currentPrice,
			TakeProfit:           takeProfit,
			StopLoss:             stopLoss,
			AllocationMultiplier: float64(leverage) / 25.0,
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
		"threshold":  threshold,
	}).Debug("Signal below threshold or HOLD - skipping execution")

	return &brain.StrikerDecision{
		Timestamp:    time.Now().Format(time.RFC3339),
		TopTargets:   []brain.TargetAsset{},
		MarketRegime: "RANGING",
	}, nil
}

// executeEnhancedDecision executes an enhanced trading decision
func (es *EnhancedStriker) executeEnhancedDecision(
	ctx context.Context,
	symbol string,
	decision *brain.TradingDecision,
	quantity float64,
	leverage int,
	currentPrice float64,
	volatility float64,
) {
	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"decision":   decision.Decision,
		"confidence": decision.Confidence,
		"leverage":   leverage,
		"quantity":   quantity,
		"volatility": volatility,
	}).Info("⚡ Executing enhanced trading decision")

	switch decision.Decision {
	case "BUY":
		es.ExecuteEnhancedBuyOrder(ctx, symbol, decision, quantity, leverage, currentPrice, volatility)
	case "SELL":
		es.ExecuteEnhancedSellOrder(ctx, symbol, decision, quantity, leverage, currentPrice, volatility)
	case "HOLD":
		logrus.WithField("symbol", symbol).Info("Holding position - no action taken")
	default:
		logrus.WithField("decision", decision.Decision).Error("Unknown trading decision")
	}
}

// ExecuteEnhancedBuyOrder executes an enhanced buy order
func (es *EnhancedStriker) ExecuteEnhancedBuyOrder(
	ctx context.Context,
	symbol string,
	decision *brain.TradingDecision,
	quantity float64,
	leverage int,
	currentPrice float64,
	volatility float64,
) {
	// Set leverage first
	if err := es.setLeverage(ctx, symbol, leverage); err != nil {
		logrus.WithError(err).Error("Failed to set leverage")
		return
	}

	// Calculate stop loss and take profit
	stopLoss := es.calculateStopLoss(currentPrice, volatility, "BUY")
	takeProfit := es.calculateTakeProfit(currentPrice, volatility, "BUY")

	// Use self-optimizing executor for order execution
	order, err := es.selfOptimizingExecutor.Execute(ctx, symbol, futures.SideTypeBuy, quantity, currentPrice)
	if err != nil {
		logrus.WithError(err).Error("Self-optimizing buy order execution failed")
		return
	}

	logrus.WithFields(logrus.Fields{
		"symbol":       symbol,
		"order_id":     order.OrderID,
		"quantity":     quantity,
		"price":        currentPrice,
		"leverage":     leverage,
		"stop_loss":    stopLoss,
		"take_profit":  takeProfit,
	}).Info("⚡ Enhanced buy order executed with self-optimization")

	// Set trailing stop loss and take profit
	es.setEnhancedRiskManagement(ctx, symbol, currentPrice, stopLoss, takeProfit, futures.SideTypeBuy)
}

// ExecuteEnhancedSellOrder executes an enhanced sell order
func (es *EnhancedStriker) ExecuteEnhancedSellOrder(
	ctx context.Context,
	symbol string,
	decision *brain.TradingDecision,
	quantity float64,
	leverage int,
	currentPrice float64,
	volatility float64,
) {
	// Set leverage first
	if err := es.setLeverage(ctx, symbol, leverage); err != nil {
		logrus.WithError(err).Error("Failed to set leverage")
		return
	}

	// Calculate stop loss and take profit
	stopLoss := es.calculateStopLoss(currentPrice, volatility, "SELL")
	takeProfit := es.calculateTakeProfit(currentPrice, volatility, "SELL")

	// Use self-optimizing executor for order execution
	order, err := es.selfOptimizingExecutor.Execute(ctx, symbol, futures.SideTypeSell, quantity, currentPrice)
	if err != nil {
		logrus.WithError(err).Error("Self-optimizing sell order execution failed")
		return
	}

	logrus.WithFields(logrus.Fields{
		"symbol":       symbol,
		"order_id":     order.OrderID,
		"quantity":     quantity,
		"price":        currentPrice,
		"leverage":     leverage,
		"stop_loss":    stopLoss,
		"take_profit":  takeProfit,
	}).Info("⚡ Enhanced sell order executed with self-optimization")

	// Set trailing stop loss and take profit
	es.setEnhancedRiskManagement(ctx, symbol, currentPrice, stopLoss, takeProfit, futures.SideTypeSell)
}

// setEnhancedRiskManagementWithTrailing sets enhanced risk management with trailing SL/TP
func (es *EnhancedStriker) setEnhancedRiskManagementWithTrailing(
	ctx context.Context,
	symbol string,
	entryPrice float64,
	quantity float64,
	stopLoss float64,
	takeProfit float64,
	side string,
) {
	// Place initial stop loss order
	stopOrderSide := futures.SideTypeSell
	if side == "SHORT" {
		stopOrderSide = futures.SideTypeBuy
	}

	stopOrder, err := es.client.NewCreateOrderService().
		Symbol(symbol).
		Side(stopOrderSide).
		Type("STOP").
		Quantity(fmt.Sprintf("%.6f", quantity)).
		StopPrice(fmt.Sprintf("%.2f", stopLoss)).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to set stop loss order")
	} else {
		logrus.WithFields(logrus.Fields{
			"symbol":     symbol,
			"order_id":   stopOrder.OrderID,
			"stop_price": stopLoss,
		}).Info("⚡ Stop loss order placed")
	}

	// Place initial take profit order
	tpOrder, err := es.client.NewCreateOrderService().
		Symbol(symbol).
		Side(stopOrderSide).
		Type("TAKE_PROFIT").
		Quantity(fmt.Sprintf("%.6f", quantity)).
		StopPrice(fmt.Sprintf("%.2f", takeProfit)).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to set take profit order")
	} else {
		logrus.WithFields(logrus.Fields{
			"symbol":   symbol,
			"order_id": tpOrder.OrderID,
			"tp_price": takeProfit,
		}).Info("⚡ Take profit order placed")
	}

	// Add position to trailing manager
	if es.trailingManager != nil {
		es.trailingManager.AddPosition(
			symbol,
			side,
			entryPrice,
			quantity,
			stopLoss,
			takeProfit,
			fmt.Sprintf("%d", stopOrder.OrderID),
			fmt.Sprintf("%d", tpOrder.OrderID),
		)
	}
}

// calculateStopLoss calculates dynamic stop loss based on volatility
func (es *EnhancedStriker) calculateStopLoss(currentPrice, volatility float64, decision string) float64 {
	// Dynamic stop loss based on volatility
	stopDistance := volatility * 2.0 // 2x ATR

	if es.highRiskMode {
		stopDistance *= 1.5 // Wider stop for high risk
	}

	if decision == "BUY" {
		return currentPrice * (1.0 - stopDistance)
	}
	return currentPrice * (1.0 + stopDistance)
}

// calculateTakeProfit calculates dynamic take profit based on volatility
func (es *EnhancedStriker) calculateTakeProfit(currentPrice, volatility float64, decision string) float64 {
	// Dynamic take profit based on volatility
	profitDistance := volatility * 3.0 // 3x ATR

	if es.highRiskMode {
		profitDistance *= 2.0 // Higher target for high risk
	}

	if decision == "BUY" {
		return currentPrice * (1.0 + profitDistance)
	}
	return currentPrice * (1.0 - profitDistance)
}

// setLeverage sets leverage for a symbol
func (es *EnhancedStriker) setLeverage(ctx context.Context, symbol string, leverage int) error {
	_, err := es.client.NewChangeLeverageService().
		Symbol(symbol).
		Leverage(leverage).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"symbol":   symbol,
		"leverage": leverage,
	}).Info("⚡ Leverage set")

	return nil
}

// calculateVolatility calculates volatility from klines
func (es *EnhancedStriker) calculateVolatility(klines []*futures.Kline) float64 {
	if len(klines) < 2 {
		return 0.02
	}

	prices := make([]float64, len(klines))
	for i, k := range klines {
		prices[i] = parseFloat(k.Close)
	}

	// Calculate standard deviation
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
	variance := sumSqDiff / float64(len(prices))
	stdDev := math.Sqrt(variance)

	// Convert to percentage
	volatility := (stdDev / mean)

	return volatility
}

// checkVolumeSpike checks for volume spike
func (es *EnhancedStriker) checkVolumeSpike(klines []*futures.Kline) bool {
	if len(klines) < 5 {
		return false
	}

	// Calculate recent volume vs average volume
	recentVol := 0.0
	for i := len(klines) - 5; i < len(klines); i++ {
		recentVol += parseFloat(klines[i].Volume)
	}
	recentVol /= 5.0

	avgVol := 0.0
	for i := 0; i < len(klines)-5; i++ {
		avgVol += parseFloat(klines[i].Volume)
	}
	avgVol /= float64(len(klines) - 5)

	threshold := 1.5
	if es.highRiskMode {
		threshold = 2.0
	}

	return recentVol/avgVol > threshold
}

// extractAssetInfo extracts asset information from interface
func (es *EnhancedStriker) extractAssetInfo(asset interface{}) (string, float64, float64) {
	// Type assert to *futures.PriceChangeStats
	if priceChangeStats, ok := asset.(*futures.PriceChangeStats); ok {
		symbol := priceChangeStats.Symbol
		price := parseFloat(priceChangeStats.LastPrice)
		priceChangePercent := parseFloat(priceChangeStats.PriceChangePercent)
		
		// Calculate confidence based on price change
		confidence := 0.5 + (math.Abs(priceChangePercent) / 20.0)
		if confidence > 1.0 {
			confidence = 1.0
		}
		
		return symbol, price, confidence
	}
	
	// Fallback: try to extract from map
	if assetMap, ok := asset.(map[string]interface{}); ok {
		symbol, _ := assetMap["symbol"].(string)
		price := 0.0
		confidence := 0.5
		
		if priceVal, ok := assetMap["price"].(float64); ok {
			price = priceVal
		} else if priceStr, ok := assetMap["price"].(string); ok {
			price = parseFloat(priceStr)
		}
		
		if priceChange, ok := assetMap["priceChangePercent"].(float64); ok {
			confidence = 0.5 + (math.Abs(priceChange) / 20.0)
			if confidence > 1.0 {
				confidence = 1.0
			}
		}
		
		return symbol, price, confidence
	}
	
	return "", 0, 0
}

// checkPosition checks if position exists
func (es *EnhancedStriker) checkPosition(ctx context.Context, symbol string) map[string]interface{} {
	positions, err := es.client.NewGetPositionRiskService().
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

// calculateDefaultQuantity calculates default quantity if dynamic sizing fails
func (es *EnhancedStriker) calculateDefaultQuantity(leverage int) float64 {
	baseQuantity := 0.001
	return baseQuantity * float64(leverage) / 25.0
}

// Start begins enhanced striker
func (es *EnhancedStriker) Start(ctx context.Context) error {
	logrus.Info("⚡ Starting enhanced striker...")
	es.isRunning = true
	logrus.Info("✅ Enhanced striker started")
	return nil
}

// Stop gracefully stops the enhanced striker
func (es *EnhancedStriker) Stop() error {
	logrus.Info("🛑 Stopping enhanced striker...")
	es.isRunning = false
	return nil
}

// GetExecutionMetrics returns execution metrics from self-optimizing executor
func (es *EnhancedStriker) GetExecutionMetrics() *executor.ExecutionMetrics {
	if es.selfOptimizingExecutor != nil {
		return es.selfOptimizingExecutor.GetMetrics()
	}
	return nil
}
