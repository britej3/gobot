package executor

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/sirupsen/logrus"
)

// SelfOptimizingExecutor adds self-optimization to the execution phase
type SelfOptimizingExecutor struct {
	client          *futures.Client
	config          *SelfOptimizingConfig
	metrics         *ExecutionMetrics
	mu              sync.RWMutex
	optimizationLog []ExecutionRecord
}

// SelfOptimizingConfig configures self-optimizing behavior
type SelfOptimizingConfig struct {
	// Dynamic Order Type Selection
	UseSmartOrderType      bool
	MarketOrderThreshold  float64 // Volatility threshold for market orders
	LimitOrderTimeout     time.Duration

	// Adaptive Slippage Tolerance
	BaseSlippageTolerance  float64
	MaxSlippageTolerance   float64
	SlippageAdaptationRate float64

	// Smart Entry Timing
	EnableSmartEntry      bool
	MaxWaitTime           time.Duration
	EntryQualityThreshold float64

	// Adaptive Retry Logic
	MaxRetries         int
	BaseRetryDelay     time.Duration
	ExponentialBackoff bool

	// Fee Optimization
	UseLimitOrdersForFeeReduction bool
	LimitOrderSlippageBuffer     float64

	// Order Splitting
	EnableOrderSplitting  bool
	MinSplitSize          float64
	MaxSplitSize          float64
	SplitInterval         time.Duration

	// Execution Quality Monitoring
	TrackExecutionQuality    bool
	MinExecutionQualityScore float64
	ExecutionQualityWindow   int
}

// ExecutionMetrics tracks execution performance
type ExecutionMetrics struct {
	TotalOrders       int64
	SuccessfulOrders  int64
	FailedOrders      int64
	AverageSlippage   float64
	AverageFillTime   time.Duration
	AverageExecutionQuality float64
	MarketOrderCount  int64
	LimitOrderCount   int64
	LastOptimization time.Time
}

// ExecutionRecord records each execution for learning
type ExecutionRecord struct {
	Timestamp       time.Time
	Symbol          string
	OrderType       string
	Slippage        float64
	FillTime        time.Duration
	ExecutionQuality float64
	Volatility      float64
	Volume          float64
	Success         bool
	Reasoning       string
}

// NewSelfOptimizingExecutor creates a new self-optimizing executor
func NewSelfOptimizingExecutor(client *futures.Client, config *SelfOptimizingConfig) *SelfOptimizingExecutor {
	if config == nil {
		config = DefaultSelfOptimizingConfig()
	}

	return &SelfOptimizingExecutor{
		client:          client,
		config:          config,
		metrics:         &ExecutionMetrics{},
		optimizationLog: make([]ExecutionRecord, 0),
	}
}

// DefaultSelfOptimizingConfig returns default configuration
func DefaultSelfOptimizingConfig() *SelfOptimizingConfig {
	return &SelfOptimizingConfig{
		UseSmartOrderType:           true,
		MarketOrderThreshold:       0.03, // 3% volatility
		LimitOrderTimeout:           30 * time.Second,
		BaseSlippageTolerance:       0.001, // 0.1%
		MaxSlippageTolerance:        0.005, // 0.5%
		SlippageAdaptationRate:      0.1,
		EnableSmartEntry:            true,
		MaxWaitTime:                 15 * time.Second,
		EntryQualityThreshold:       0.7,
		MaxRetries:                  3,
		BaseRetryDelay:              1 * time.Second,
		ExponentialBackoff:          true,
		UseLimitOrdersForFeeReduction: true,
		LimitOrderSlippageBuffer:    0.002, // 0.2%
		EnableOrderSplitting:        true,
		MinSplitSize:                10.0,  // 10 USDT
		MaxSplitSize:                50.0,  // 50 USDT
		SplitInterval:               2 * time.Second,
		TrackExecutionQuality:       true,
		MinExecutionQualityScore:    0.6,
		ExecutionQualityWindow:      50,
	}
}

// Execute executes a trade with self-optimization
func (e *SelfOptimizingExecutor) Execute(
	ctx context.Context,
	symbol string,
	side futures.SideType,
	quantity float64,
	price float64,
) (*futures.CreateOrderResponse, error) {
	startTime := time.Now()

	logrus.WithFields(logrus.Fields{
		"symbol":   symbol,
		"side":     side,
		"quantity": quantity,
		"price":    price,
	}).Info("🎯 Self-optimizing execution started")

	// Get market conditions
	volatility, volume, err := e.getMarketConditions(ctx, symbol)
	if err != nil {
		logrus.WithError(err).Warn("Failed to get market conditions, using defaults")
		volatility = 0.02
		volume = 1000000
	}

	// Determine optimal order type
	orderType := e.selectOptimalOrderType(volatility, volume)

	// Smart entry timing
	if e.config.EnableSmartEntry && orderType == futures.OrderTypeLimit {
		optimizedPrice, shouldWait := e.optimizeEntryTiming(ctx, symbol, side, price, volatility)
		if shouldWait {
			price = optimizedPrice
			logrus.WithFields(logrus.Fields{
				"original_price": price,
				"optimized_price": optimizedPrice,
				"symbol":         symbol,
			}).Info("⏱️ Smart entry timing applied")
		}
	}

	// Order splitting
	var orders []*futures.CreateOrderResponse
	if e.config.EnableOrderSplitting && quantity*e.priceToUSD(symbol, price) > e.config.MaxSplitSize {
		orders, err = e.executeSplitOrder(ctx, symbol, side, quantity, price, orderType)
		if err != nil {
			logrus.WithError(err).Error("Split order execution failed, falling back to single order")
			orders = nil
		}
	}

	// Single order execution
	if len(orders) == 0 {
		order, err := e.executeOrderWithRetry(ctx, symbol, side, quantity, price, orderType)
		if err != nil {
			e.recordExecution(symbol, orderType, 0, 0, 0, volatility, volume, false, "Execution failed")
			return nil, err
		}
		orders = append(orders, order)
	}

	// Calculate execution metrics
	fillTime := time.Since(startTime)
	avgSlippage := e.calculateSlippage(orders, price)
	executionQuality := e.calculateExecutionQuality(avgSlippage, fillTime, volatility)

	// Record execution
	e.recordExecution(symbol, orderType, avgSlippage, fillTime, executionQuality, volatility, volume, true, "Success")

	// Update metrics
	e.updateMetrics(orderType, avgSlippage, fillTime, executionQuality, true)

	// Optimize based on execution
	if e.config.TrackExecutionQuality {
		e.optimizeParameters()
	}

	logrus.WithFields(logrus.Fields{
		"symbol":            symbol,
		"order_type":        orderType,
		"slippage":          avgSlippage,
		"fill_time":         fillTime,
		"execution_quality": executionQuality,
		"orders_count":      len(orders),
	}).Info("✅ Self-optimizing execution completed")

	// Return first order as primary
	return orders[0], nil
}

// selectOptimalOrderType selects the best order type based on market conditions
func (e *SelfOptimizingExecutor) selectOptimalOrderType(volatility, volume float64) futures.OrderType {
	if !e.config.UseSmartOrderType {
		return futures.OrderTypeMarket
	}

	// High volatility + low volume = use limit orders to avoid slippage
	if volatility > e.config.MarketOrderThreshold && volume < 5000000 {
		logrus.WithFields(logrus.Fields{
			"volatility": volatility,
			"volume":     volume,
		}).Info("📊 High volatility + low volume detected, using limit orders")
		return futures.OrderTypeLimit
	}

	// Normal conditions = use market orders for speed
	if volatility < e.config.MarketOrderThreshold && volume > 10000000 {
		logrus.WithFields(logrus.Fields{
			"volatility": volatility,
			"volume":     volume,
		}).Info("📊 Normal market conditions, using market orders")
		return futures.OrderTypeMarket
	}

	// Fee optimization: use limit orders when possible
	if e.config.UseLimitOrdersForFeeReduction {
		logrus.Info("📊 Fee optimization mode, using limit orders")
		return futures.OrderTypeLimit
	}

	return futures.OrderTypeMarket
}

// optimizeEntryTiming optimizes entry timing for limit orders
func (e *SelfOptimizingExecutor) optimizeEntryTiming(
	ctx context.Context,
	symbol string,
	side futures.SideType,
	price float64,
	volatility float64,
) (optimizedPrice float64, shouldWait bool) {
	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"side":       side,
		"price":      price,
		"volatility": volatility,
	}).Info("⏱️ Optimizing entry timing")

	// Get recent price action
	klines, err := e.client.NewKlinesService().
		Symbol(symbol).
		Interval("1m").
		Limit(10).
		Do(ctx)

	if err != nil || len(klines) < 5 {
		return price, false
	}

	// Calculate price quality score
	priceQuality := e.calculatePriceQuality(klines, price, side)

	if priceQuality < e.config.EntryQualityThreshold {
		logrus.WithField("quality_score", priceQuality).Info("⏱️ Entry quality below threshold, optimizing price")

		// Calculate optimal entry price
		if side == futures.SideTypeBuy {
			// Wait for dip
			optimizedPrice = price * (1.0 - volatility*0.5)
		} else {
			// Wait for rally
			optimizedPrice = price * (1.0 + volatility*0.5)
		}

		return optimizedPrice, true
	}

	return price, false
}

// calculatePriceQuality calculates quality of entry price
func (e *SelfOptimizingExecutor) calculatePriceQuality(
	klines []*futures.Kline,
	price float64,
	side futures.SideType,
) float64 {
	if len(klines) < 5 {
		return 0.5
	}

	// Calculate moving average
	sum := 0.0
	for _, k := range klines[:5] {
		sum += parseFloat(k.Close)
	}
	ma := sum / 5.0

	// Calculate quality based on side
	quality := 0.5
	if side == futures.SideTypeBuy {
		// Better to buy below MA
		if price < ma {
			quality = 0.5 + (ma-price)/ma
		} else {
			quality = 0.5 - (price-ma)/ma
		}
	} else {
		// Better to sell above MA
		if price > ma {
			quality = 0.5 + (price-ma)/ma
		} else {
			quality = 0.5 - (ma-price)/ma
		}
	}

	// Clamp between 0 and 1
	if quality < 0 {
		quality = 0
	} else if quality > 1 {
		quality = 1
	}

	return quality
}

// executeOrderWithRetry executes order with adaptive retry logic
func (e *SelfOptimizingExecutor) executeOrderWithRetry(
	ctx context.Context,
	symbol string,
	side futures.SideType,
	quantity float64,
	price float64,
	orderType futures.OrderType,
) (*futures.CreateOrderResponse, error) {
	var lastErr error
	retryDelay := e.config.BaseRetryDelay

	for attempt := 0; attempt < e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			logrus.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"max":     e.config.MaxRetries,
				"delay":   retryDelay,
			}).Info("🔄 Retrying order execution")

			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			// Exponential backoff
			if e.config.ExponentialBackoff {
				retryDelay *= 2
			}
		}

		// Execute order
		order, err := e.executeSingleOrder(ctx, symbol, side, quantity, price, orderType)
		if err == nil {
			return order, nil
		}

		lastErr = err
		
		// Log detailed error information
		e.LogErrorDetails(err, symbol, side, quantity, price)

		// Adapt based on error type
		if e.isRetryableError(err) {
			logrus.WithFields(logrus.Fields{
				"attempt": attempt + 1,
			}).Info("♻️ Error is retryable, adjusting parameters...")
			
			// Adjust slippage tolerance for next attempt
			price = e.adjustPriceForRetry(price, orderType, attempt)
		} else {
			logrus.WithFields(logrus.Fields{
				"attempt": attempt + 1,
			}).Error("❌ Error is not retryable, aborting...")
			break
		}
	}

	return nil, fmt.Errorf("order execution failed after %d attempts: %w", e.config.MaxRetries, lastErr)
}

// executeSingleOrder executes a single order
func (e *SelfOptimizingExecutor) executeSingleOrder(
	ctx context.Context,
	symbol string,
	side futures.SideType,
	quantity float64,
	price float64,
	orderType futures.OrderType,
) (*futures.CreateOrderResponse, error) {
	// Get symbol precision
	quantityPrecision, pricePrecision, err := e.getSymbolPrecision(ctx, symbol)
	if err != nil {
		logrus.WithError(err).Warn("Failed to get symbol precision, using defaults")
		quantityPrecision = 3
		pricePrecision = 8
	}

	// Format quantity with correct precision
	quantityStr := fmt.Sprintf(fmt.Sprintf("%%.%df", quantityPrecision), quantity)

	orderService := e.client.NewCreateOrderService().
		Symbol(symbol).
		Side(side).
		Type(orderType).
		Quantity(quantityStr)

	if orderType == futures.OrderTypeLimit {
		// Add slippage buffer for limit orders
		adjustedPrice := e.adjustPriceForSlippage(price, side)
		priceStr := fmt.Sprintf(fmt.Sprintf("%%.%df", pricePrecision), adjustedPrice)
		orderService = orderService.
			Price(priceStr).
			TimeInForce(futures.TimeInForceTypeGTC)

		// Set timeout for limit orders
		orderCtx, cancel := context.WithTimeout(ctx, e.config.LimitOrderTimeout)
		defer cancel()
		order, err := orderService.Do(orderCtx)
		if err != nil {
			// Log detailed error information
			e.LogErrorDetails(err, symbol, side, quantity, adjustedPrice)
			return nil, fmt.Errorf("failed to create order: %w", err)
		}
		return order, nil
	}

	order, err := orderService.Do(ctx)
	if err != nil {
		// Log detailed error information
		e.LogErrorDetails(err, symbol, side, quantity, price)
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil
}

// getSymbolPrecision gets the quantity and price precision for a symbol
func (e *SelfOptimizingExecutor) getSymbolPrecision(ctx context.Context, symbol string) (quantityPrecision, pricePrecision int, err error) {
	// Get exchange info
	exchangeInfo, err := e.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return 3, 8, fmt.Errorf("failed to get exchange info: %w", err)
	}

	// Find symbol info
	for _, s := range exchangeInfo.Symbols {
		if s.Symbol == symbol {
			// Use symbol's precision directly
			return s.QuantityPrecision, s.PricePrecision, nil
		}
	}

	return 3, 8, fmt.Errorf("symbol %s not found in exchange info", symbol)
}

// countDecimals counts the number of decimal places in a string
func countDecimals(s string) int {
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return 0
	}
	return len(strings.TrimRight(parts[1], "0"))
}

// executeSplitOrder executes a large order in smaller chunks
func (e *SelfOptimizingExecutor) executeSplitOrder(
	ctx context.Context,
	symbol string,
	side futures.SideType,
	totalQuantity float64,
	price float64,
	orderType futures.OrderType,
) ([]*futures.CreateOrderResponse, error) {
	logrus.WithFields(logrus.Fields{
		"symbol":   symbol,
		"side":     side,
		"quantity": totalQuantity,
	}).Info("📦 Executing split order")

	// Calculate number of splits
	orderValue := totalQuantity * e.priceToUSD(symbol, price)
	splits := int(math.Ceil(orderValue / e.config.MaxSplitSize))
	if splits < 2 {
		splits = 2
	}

	// Split quantity
	splitQuantity := totalQuantity / float64(splits)

	orders := make([]*futures.CreateOrderResponse, 0, splits)
	for i := 0; i < splits; i++ {
		// Execute split
		order, err := e.executeOrderWithRetry(ctx, symbol, side, splitQuantity, price, orderType)
		if err != nil {
			logrus.WithError(err).WithField("split", i+1).Error("Split order failed")
			return orders, fmt.Errorf("split order %d failed: %w", i+1, err)
		}

		orders = append(orders, order)
		logrus.WithField("split", i+1).Info("✅ Split order executed")

		// Wait between splits
		if i < splits-1 {
			select {
			case <-time.After(e.config.SplitInterval):
			case <-ctx.Done():
				return orders, ctx.Err()
			}
		}
	}

	return orders, nil
}

// adjustPriceForSlippage adjusts price for slippage tolerance
func (e *SelfOptimizingExecutor) adjustPriceForSlippage(price float64, side futures.SideType) float64 {
	// Get adaptive slippage tolerance
	slippage := e.getAdaptiveSlippageTolerance()

	if side == futures.SideTypeBuy {
		return price * (1.0 + slippage)
	}
	return price * (1.0 - slippage)
}

// adjustPriceForRetry adjusts price for retry attempts
func (e *SelfOptimizingExecutor) adjustPriceForRetry(price float64, orderType futures.OrderType, attempt int) float64 {
	if orderType == futures.OrderTypeMarket {
		return price
	}

	// Increase slippage tolerance for retries
	adjustment := float64(attempt+1) * e.config.LimitOrderSlippageBuffer
	return price * (1.0 + adjustment)
}

// getAdaptiveSlippageTolerance returns adaptive slippage tolerance
func (e *SelfOptimizingExecutor) getAdaptiveSlippageTolerance() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// If we have no history, use base tolerance
	if e.metrics.TotalOrders == 0 {
		return e.config.BaseSlippageTolerance
	}

	// Adapt based on recent execution quality
	adaptiveTolerance := e.config.BaseSlippageTolerance
	if e.metrics.AverageSlippage > e.config.BaseSlippageTolerance*2 {
		// Increase tolerance if slippage is high
		adaptiveTolerance *= (1.0 + e.config.SlippageAdaptationRate)
	} else if e.metrics.AverageSlippage < e.config.BaseSlippageTolerance*0.5 {
		// Decrease tolerance if slippage is low
		adaptiveTolerance *= (1.0 - e.config.SlippageAdaptationRate)
	}

	// Clamp to max tolerance
	if adaptiveTolerance > e.config.MaxSlippageTolerance {
		adaptiveTolerance = e.config.MaxSlippageTolerance
	}

	return adaptiveTolerance
}

// calculateSlippage calculates average slippage from orders
func (e *SelfOptimizingExecutor) calculateSlippage(orders []*futures.CreateOrderResponse, expectedPrice float64) float64 {
	if len(orders) == 0 {
		return 0
	}

	totalSlippage := 0.0
	for _, order := range orders {
		// For market orders, calculate slippage from avgPrice
		if order.AvgPrice != "" {
			avgPrice := parseFloat(order.AvgPrice)
			slippage := math.Abs(avgPrice-expectedPrice) / expectedPrice
			totalSlippage += slippage
		} else if order.Price != "" {
			// For limit orders, use the order price
			orderPrice := parseFloat(order.Price)
			slippage := math.Abs(orderPrice-expectedPrice) / expectedPrice
			totalSlippage += slippage
		}
	}

	return totalSlippage / float64(len(orders))
}

// calculateExecutionQuality calculates execution quality score
func (e *SelfOptimizingExecutor) calculateExecutionQuality(
	slippage float64,
	fillTime time.Duration,
	volatility float64,
) float64 {
	// Normalize slippage (lower is better)
	slippageScore := 1.0 - math.Min(slippage/e.config.MaxSlippageTolerance, 1.0)

	// Normalize fill time (faster is better)
	fillTimeScore := 1.0 - math.Min(float64(fillTime.Seconds())/5.0, 1.0)

	// Combined score
	quality := (slippageScore*0.7 + fillTimeScore*0.3)

	return quality
}

// recordExecution records an execution for learning
func (e *SelfOptimizingExecutor) recordExecution(
	symbol string,
	orderType futures.OrderType,
	slippage float64,
	fillTime time.Duration,
	executionQuality float64,
	volatility float64,
	volume float64,
	success bool,
	reasoning string,
) {
	e.mu.Lock()
	defer e.mu.Unlock()

	record := ExecutionRecord{
		Timestamp:        time.Now(),
		Symbol:           symbol,
		OrderType:        string(orderType),
		Slippage:         slippage,
		FillTime:         fillTime,
		ExecutionQuality: executionQuality,
		Volatility:       volatility,
		Volume:           volume,
		Success:          success,
		Reasoning:        reasoning,
	}

	e.optimizationLog = append(e.optimizationLog, record)

	// Keep only recent records
	if len(e.optimizationLog) > e.config.ExecutionQualityWindow {
		e.optimizationLog = e.optimizationLog[1:]
	}
}

// updateMetrics updates execution metrics
func (e *SelfOptimizingExecutor) updateMetrics(
	orderType futures.OrderType,
	slippage float64,
	fillTime time.Duration,
	executionQuality float64,
	success bool,
) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.metrics.TotalOrders++
	if success {
		e.metrics.SuccessfulOrders++
	} else {
		e.metrics.FailedOrders++
	}

	// Update running averages
	count := float64(e.metrics.SuccessfulOrders)
	e.metrics.AverageSlippage = (e.metrics.AverageSlippage*(count-1) + slippage) / count
	e.metrics.AverageFillTime = time.Duration(
		(float64(e.metrics.AverageFillTime)*(count-1) + float64(fillTime)) / count,
	)
	e.metrics.AverageExecutionQuality = (e.metrics.AverageExecutionQuality*(count-1) + executionQuality) / count

	if orderType == futures.OrderTypeMarket {
		e.metrics.MarketOrderCount++
	} else {
		e.metrics.LimitOrderCount++
	}

	e.metrics.LastOptimization = time.Now()
}

// optimizeParameters optimizes execution parameters based on history
func (e *SelfOptimizingExecutor) optimizeParameters() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.optimizationLog) < 10 {
		return
	}

	// Analyze recent executions
	marketOrderQuality := 0.0
	limitOrderQuality := 0.0
	marketCount := 0
	limitCount := 0

	for _, record := range e.optimizationLog {
		if record.OrderType == string(futures.OrderTypeMarket) {
			marketOrderQuality += record.ExecutionQuality
			marketCount++
		} else {
			limitOrderQuality += record.ExecutionQuality
			limitCount++
		}
	}

	// Adjust order type preference
	if marketCount > 0 && limitCount > 0 {
		avgMarketQuality := marketOrderQuality / float64(marketCount)
		avgLimitQuality := limitOrderQuality / float64(limitCount)

		if avgLimitQuality > avgMarketQuality*1.1 {
			logrus.Info("📈 Limit orders performing better, increasing preference")
			e.config.MarketOrderThreshold *= 1.1
		} else if avgMarketQuality > avgLimitQuality*1.1 {
			logrus.Info("📈 Market orders performing better, increasing preference")
			e.config.MarketOrderThreshold *= 0.9
		}
	}

	logrus.WithFields(logrus.Fields{
		"market_threshold": e.config.MarketOrderThreshold,
		"slippage_tolerance": e.getAdaptiveSlippageTolerance(),
	}).Info("🔧 Execution parameters optimized")
}

// getMarketConditions gets current market conditions
func (e *SelfOptimizingExecutor) getMarketConditions(
	ctx context.Context,
	symbol string,
) (volatility float64, volume float64, err error) {
	// Get klines for volatility
	klines, err := e.client.NewKlinesService().
		Symbol(symbol).
		Interval("5m").
		Limit(20).
		Do(ctx)

	if err == nil && len(klines) > 1 {
		volatility = e.calculateVolatility(klines)
	}

	// Get 24h ticker for volume
	ticker, err := e.client.NewListPriceChangeStatsService().Symbol(symbol).Do(ctx)
	if err == nil && len(ticker) > 0 {
		volume = parseFloat(ticker[0].QuoteVolume)
	}

	return volatility, volume, nil
}

// calculateVolatility calculates price volatility
func (e *SelfOptimizingExecutor) calculateVolatility(klines []*futures.Kline) float64 {
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

	return stdDev / mean
}

// BinanceError represents a Binance API error with detailed information
type BinanceError struct {
	Code    int
	Message string
	Details string
	Solution string
	Critical bool
}

// GetBinanceErrorDetails returns detailed error information for Binance API errors
func GetBinanceErrorDetails(err error) *BinanceError {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Parse error code from message
	var code int
	fmt.Sscanf(errMsg, "<APIError> code=%d", &code)

	// Map error codes to detailed information
	switch code {
	case -1000:
		return &BinanceError{
			Code:    -1000,
			Message: "Unknown error",
			Details: "An unknown error occurred while processing the request",
			Solution: "Check network connection and retry. If persists, contact Binance support",
			Critical: false,
		}
	case -1001:
		return &BinanceError{
			Code:    -1001,
			Message: "Disconnected",
			Details: "Internal error; unable to process your request",
			Solution: "Wait a few seconds and retry. Check your network connection",
			Critical: false,
		}
	case -1002:
		return &BinanceError{
			Code:    -1002,
			Message: "Unauthorized",
			Details: "API key or signature is invalid",
			Solution: "Check your API key and secret in .env file. Ensure they are correct and have proper permissions",
			Critical: true,
		}
	case -1003:
		return &BinanceError{
			Code:    -1003,
			Message: "Too many requests",
			Details: "API request rate limit exceeded",
			Solution: "Wait and retry. Reduce request frequency. Consider increasing delays between requests",
			Critical: false,
		}
	case -1006:
		return &BinanceError{
			Code:    -1006,
			Message: "Unexpected response",
			Details: "An unexpected response was received from the server",
			Solution: "Retry the request. Check if Binance services are operational",
			Critical: false,
		}
	case -1007:
		return &BinanceError{
			Code:    -1007,
			Message: "Timeout",
			Details: "Server did not respond in time",
			Solution: "Increase timeout duration. Check network latency to Binance servers",
			Critical: false,
		}
	case -1013:
		return &BinanceError{
			Code:    -1013,
			Message: "Invalid quantity",
			Details: "Order quantity is invalid (too small, too large, or not a multiple of lot size)",
			Solution: "Check order quantity. Ensure it meets minimum order size and lot size requirements for the symbol",
			Critical: false,
		}
	case -1014:
		return &BinanceError{
			Code:    -1014,
			Message: "Invalid order type",
			Details: "Order type is not supported for this symbol or trading pair",
			Solution: "Check if the order type (MARKET, LIMIT, etc.) is supported for this symbol",
			Critical: false,
		}
	case -1015:
		return &BinanceError{
			Code:    -1015,
			Message: "Too many orders",
			Details: "Too many orders placed in a short time",
			Solution: "Wait and retry. Reduce order frequency. Check your account's order rate limits",
			Critical: false,
		}
	case -1021:
		return &BinanceError{
			Code:    -1021,
			Message: "Timestamp out of sync",
			Details: "Timestamp for the request is outside the recvWindow",
			Solution: "Sync your system time with NTP servers. Check your system clock accuracy",
			Critical: true,
		}
	case -1022:
		return &BinanceError{
			Code:    -1022,
			Message: "Invalid signature",
			Details: "Signature for the request is invalid",
			Solution: "Check your API secret key. Ensure it matches exactly what's in your Binance account",
			Critical: true,
		}
	case -1100:
		return &BinanceError{
			Code:    -1100,
			Message: "Illegal characters",
			Details: "Illegal characters found in a parameter",
			Solution: "Check all parameters for invalid characters. Ensure symbols and numbers are properly formatted",
			Critical: false,
		}
	case -1101:
		return &BinanceError{
			Code:    -1101,
			Message: "Too many parameters",
			Details: "Too many parameters sent for this endpoint",
			Solution: "Check API documentation. Remove unnecessary parameters from the request",
			Critical: false,
		}
	case -1102:
		return &BinanceError{
			Code:    -1102,
			Message: "Mandatory parameter empty",
			Details: "A mandatory parameter was not sent or was empty/null",
			Solution: "Check all required parameters. Ensure none are missing or empty",
			Critical: false,
		}
	case -1103:
		return &BinanceError{
			Code:    -1103,
			Message: "Unknown parameter",
			Details: "An unknown parameter was sent",
			Solution: "Check API documentation. Remove or correct unknown parameters",
			Critical: false,
		}
	case -1104:
		return &BinanceError{
			Code:    -1104,
			Message: "Unambiguous parameter",
			Details: "Unambiguous parameters sent for a parameter that should be unique",
			Solution: "Check for duplicate or conflicting parameters",
			Critical: false,
		}
	case -1105:
		return &BinanceError{
			Code:    -1105,
			Message: "Parameter empty",
			Details: "A parameter was empty when it was not expected to be",
			Solution: "Check all parameters. Ensure they have valid values",
			Critical: false,
		}
	case -1106:
		return &BinanceError{
			Code:    -1106,
			Message: "Parameter not required",
			Details: "A parameter was sent when not required",
			Solution: "Remove unnecessary parameters from the request",
			Critical: false,
		}
	case -1111:
		return &BinanceError{
			Code:    -1111,
			Message: "Precision mismatch",
			Details: "Parameter precision does not match requirements",
			Solution: "Check parameter precision. Ensure numbers match the required decimal places",
			Critical: false,
		}
	case -1112:
		return &BinanceError{
			Code:    -1112,
			Message: "No depth",
			Details: "No depth for this order",
			Solution: "Market may be closed or illiquid. Check if the symbol is actively trading",
			Critical: false,
		}
	case -1114:
		return &BinanceError{
			Code:    -1114,
			Message: "New order rejected",
			Details: "Order rejected by Binance",
			Solution: "Check order parameters, account balance, and trading permissions",
			Critical: false,
		}
	case -1115:
		return &BinanceError{
			Code:    -1115,
			Message: "Cancel rejected",
			Details: "Order cancellation rejected",
			Solution: "Order may already be filled or cancelled. Check order status",
			Critical: false,
		}
	case -1116:
		return &BinanceError{
			Code:    -1116,
			Message: "No such ticker",
			Details: "Symbol does not exist",
			Solution: "Check the trading symbol. Ensure it's a valid Binance futures symbol",
			Critical: true,
		}
	case -1117:
		return &BinanceError{
			Code:    -1117,
			Message: "Invalid API key",
			Details: "API key is invalid",
			Solution: "Check your API key in .env file. Ensure it matches your Binance account",
			Critical: true,
		}
	case -1118:
		return &BinanceError{
			Code:    -1118,
			Message: "Invalid IP",
			Details: "IP address not allowed",
			Solution: "Check Binance API key IP whitelist settings. Add your current IP address",
			Critical: true,
		}
	case -1119:
		return &BinanceError{
			Code:    -1119,
			Message: "Operation too fast",
			Details: "Operation executed too fast",
			Solution: "Add delays between operations. Reduce request frequency",
			Critical: false,
		}
	case -1120:
		return &BinanceError{
			Code:    -1120,
			Message: "Invalid listen key",
			Details: "Invalid listen key for user data stream",
			Solution: "Generate a new listen key using the appropriate API endpoint",
			Critical: false,
		}
	case -1121:
		return &BinanceError{
			Code:    -1121,
			Message: "More than X hours",
			Details: "Timestamp is more than X hours before server time",
			Solution: "Sync your system time with NTP servers",
			Critical: true,
		}
	case -1125:
		return &BinanceError{
			Code:    -1125,
			Message: "Invalid API key type",
			Details: "API key type does not match the request type",
			Solution: "Check API key permissions. Ensure it has futures trading permissions",
			Critical: true,
		}
	case -1127:
		return &BinanceError{
			Code:    -1127,
			Message: "Target address error",
			Details: "Target address is invalid",
			Solution: "Check the target address format. Ensure it's a valid address",
			Critical: true,
		}
	case -1128:
		return &BinanceError{
			Code:    -1128,
			Message: "Combination not allowed",
			Details: "Combination of optional parameters is not allowed",
			Solution: "Check parameter combinations. Refer to API documentation",
			Critical: false,
		}
	case -1130:
		return &BinanceError{
			Code:    -1130,
			Message: "Invalid parameter",
			Details: "Invalid parameter value",
			Solution: "Check parameter values. Ensure they are within valid ranges",
			Critical: false,
		}
	case -1131:
		return &BinanceError{
			Code:    -1131,
			Message: "Bad request",
			Details: "Bad request format",
			Solution: "Check request format. Ensure all parameters are correctly formatted",
			Critical: false,
		}
	case -2010:
		return &BinanceError{
			Code:    -2010,
			Message: "New order rejected",
			Details: "Order rejected by Binance (reason unspecified)",
			Solution: "Check account balance, position limits, and trading permissions",
			Critical: false,
		}
	case -2011:
		return &BinanceError{
			Code:    -2011,
			Message: "Cancel rejected",
			Details: "Order cancellation rejected",
			Solution: "Order may already be filled or cancelled. Check order status",
			Critical: false,
		}
	case -2013:
		return &BinanceError{
			Code:    -2013,
			Message: "No such order",
			Details: "Order does not exist",
			Solution: "Check order ID. Order may have been filled or cancelled",
			Critical: false,
		}
	case -2014:
		return &BinanceError{
			Code:    -2014,
			Message: "Bad API key format",
			Details: "API key format is invalid",
			Solution: "Check API key format. Ensure it's a valid Binance API key",
			Critical: true,
		}
	case -2015:
		return &BinanceError{
			Code:    -2015,
			Message: "Rejected from endpoint",
		 Details: "Request rejected from endpoint",
			Solution: "Check if your IP is whitelisted and API key has correct permissions",
			Critical: true,
		}
	case -2016:
		return &BinanceError{
			Code:    -2016,
			Message: "Order service error",
			Details: "Order service error occurred",
			Solution: "Retry the request. Check if Binance order service is operational",
			Critical: false,
		}
	case -2018:
		return &BinanceError{
			Code:    -2018,
			Message: "Balance insufficient",
			Details: "Account has insufficient balance for this order",
			Solution: "Check account balance. Ensure you have enough margin for the order",
			Critical: true,
		}
	case -2019:
		return &BinanceError{
			Code:    -2019,
			Message: "Margin insufficient",
			Details: "Account has insufficient margin for this order",
			Solution: "Check available margin. Reduce position size or add more funds",
			Critical: true,
		}
	case -2020:
		return &BinanceError{
			Code:    -2020,
			Message: "Account has insufficient balance",
			Details: "Account has insufficient balance for the requested action",
			Solution: "Check account balance. Ensure you have enough funds",
			Critical: true,
		}
	case -2021:
		return &BinanceError{
			Code:    -2021,
			Message: "Order would immediately trigger",
			Details: "Order would immediately trigger a stop order",
			Solution: "Adjust stop price. Ensure it's not too close to current price",
			Critical: false,
		}
	case -2022:
		return &BinanceError{
			Code:    -2022,
			Message: "Reduce only rejected",
			Details: "Reduce-only order rejected: no position to reduce",
			Solution: "Check if you have an open position. Reduce-only orders require existing positions",
			Critical: false,
		}
	case -2023:
		return &BinanceError{
			Code:    -2023,
			Message: "Order would trigger immediately",
			Details: "Order would trigger immediately",
			Solution: "Adjust order price. Ensure it's not too close to current price",
			Critical: false,
		}
	case -2024:
		return &BinanceError{
			Code:    -2024,
			Message: "Current order type does not support modify order",
			Details: "Cannot modify this order type",
			Solution: "Cancel and replace the order instead of modifying",
			Critical: false,
		}
	case -2025:
		return &BinanceError{
			Code:    -2025,
			Message: "Current order type does not support cancel",
			Details: "Cannot cancel this order type",
			Solution: "Order may already be filled or cancelled",
			Critical: false,
		}
	case -2026:
		return &BinanceError{
			Code:    -2026,
			Message: "Order would immediately match",
			Details: "Order would immediately match on the order book",
			Solution: "Adjust order price. Ensure it's not crossing the spread",
			Critical: false,
		}
	case -2027:
		return &BinanceError{
			Code:    -2027,
			Message: "Order would immediately match and take",
			Details: "Order would immediately match and take liquidity",
			Solution: "Adjust order price. Use market orders instead if you want immediate execution",
			Critical: false,
		}
	case -2028:
		return &BinanceError{
			Code:    -2028,
			Message: "Stop price would trigger immediately",
			Details: "Stop price would trigger immediately",
			Solution: "Adjust stop price. Ensure it's not too close to current price",
			Critical: false,
		}
	case -2029:
		return &BinanceError{
			Code:    -2029,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2030:
		return &BinanceError{
			Code:    -2030,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2031:
		return &BinanceError{
			Code:    -2031,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2032:
		return &BinanceError{
			Code:    -2032,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2033:
		return &BinanceError{
			Code:    -2033,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2034:
		return &BinanceError{
			Code:    -2034,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2035:
		return &BinanceError{
			Code:    -2035,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2036:
		return &BinanceError{
			Code:    -2036,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2037:
		return &BinanceError{
			Code:    -2037,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2038:
		return &BinanceError{
			Code:    -2038,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2039:
		return &BinanceError{
			Code:    -2039,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2040:
		return &BinanceError{
			Code:    -2040,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2041:
		return &BinanceError{
			Code:    -2041,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2042:
		return &BinanceError{
			Code:    -2042,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2043:
		return &BinanceError{
			Code:    -2043,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2044:
		return &BinanceError{
			Code:    -2044,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2045:
		return &BinanceError{
			Code:    -2045,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2046:
		return &BinanceError{
			Code:    -2046,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2047:
		return &BinanceError{
			Code:    -2047,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2048:
		return &BinanceError{
			Code:    -2048,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2049:
		return &BinanceError{
			Code:    -2049,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2050:
		return &BinanceError{
			Code:    -2050,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2051:
		return &BinanceError{
			Code:    -2051,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2052:
		return &BinanceError{
			Code:    -2052,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2053:
		return &BinanceError{
			Code:    -2053,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2054:
		return &BinanceError{
			Code:    -2054,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2055:
		return &BinanceError{
			Code:    -2055,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2056:
		return &BinanceError{
			Code:    -2056,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2057:
		return &BinanceError{
			Code:    -2057,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2058:
		return &BinanceError{
			Code:    -2058,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2059:
		return &BinanceError{
			Code:    -2059,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2060:
		return &BinanceError{
			Code:    -2060,
			Message: "Order would be rejected by Exchange",
			Details: "Order would be rejected by Binance",
			Solution: "Check order parameters. Ensure they meet Binance requirements",
			Critical: false,
		}
	case -2061:
		return &BinanceError{
			Code:    -2061,
			Message: "Position side does not match user's setting",
			Details: "Order position side (LONG/SHORT) does not match your account position mode setting",
			Solution: "Check your Binance Futures account position mode (Hedge vs One-Way). Go to Binance Futures > Preferences > Position Mode and ensure it matches your order type. OR update your orders to match your current position mode",
			Critical: true,
		}
	case -2062:
		return &BinanceError{
			Code:    -2062,
			Message: "Position side mismatch",
			Details: "Position side mismatch between order and position",
			Solution: "Check existing positions. Ensure order side matches position side",
			Critical: true,
		}
	case -2063:
		return &BinanceError{
			Code:    -2063,
			Message: "Position limit exceeded",
			Details: "Maximum position limit exceeded",
			Solution: "Reduce position size or close existing positions",
			Critical: true,
		}
	case -2064:
		return &BinanceError{
			Code:    -2064,
			Message: "Notional value limit exceeded",
			Details: "Notional value limit exceeded",
			Solution: "Reduce position size. Check your account's notional value limits",
			Critical: true,
		}
	case -2065:
		return &BinanceError{
			Code:    -2065,
			Message: "Maintenance margin exceeded",
			Details: "Maintenance margin requirement exceeded",
			Solution: "Reduce position size or add more margin to your account",
			Critical: true,
		}
	case -2066:
		return &BinanceError{
			Code:    -2066,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2067:
		return &BinanceError{
			Code:    -2067,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2068:
		return &BinanceError{
			Code:    -2068,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2069:
		return &BinanceError{
			Code:    -2069,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2070:
		return &BinanceError{
			Code:    -2070,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2071:
		return &BinanceError{
			Code:    -2071,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2072:
		return &BinanceError{
			Code:    -2072,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2073:
		return &BinanceError{
			Code:    -2073,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2074:
		return &BinanceError{
			Code:    -2074,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2075:
		return &BinanceError{
			Code:    -2075,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2076:
		return &BinanceError{
			Code:    -2076,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2077:
		return &BinanceError{
			Code:    -2077,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2078:
		return &BinanceError{
			Code:    -2078,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2079:
		return &BinanceError{
			Code:    -2079,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2080:
		return &BinanceError{
			Code:    -2080,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2081:
		return &BinanceError{
			Code:    -2081,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2082:
		return &BinanceError{
			Code:    -2082,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2083:
		return &BinanceError{
			Code:    -2083,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2084:
		return &BinanceError{
			Code:    -2084,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2085:
		return &BinanceError{
			Code:    -2085,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2086:
		return &BinanceError{
			Code:    -2086,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2087:
		return &BinanceError{
			Code:    -2087,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2088:
		return &BinanceError{
			Code:    -2088,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2089:
		return &BinanceError{
			Code:    -2089,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2090:
		return &BinanceError{
			Code:    -2090,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2091:
		return &BinanceError{
			Code:    -2091,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2092:
		return &BinanceError{
			Code:    -2092,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2093:
		return &BinanceError{
			Code:    -2093,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2094:
		return &BinanceError{
			Code:    -2094,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2095:
		return &BinanceError{
			Code:    -2095,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2096:
		return &BinanceError{
			Code:    -2096,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2097:
		return &BinanceError{
			Code:    -2097,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2098:
		return &BinanceError{
			Code:    -2098,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2099:
		return &BinanceError{
			Code:    -2099,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2100:
		return &BinanceError{
			Code:    -2100,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2101:
		return &BinanceError{
			Code:    -2101,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2102:
		return &BinanceError{
			Code:    -2102,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2103:
		return &BinanceError{
			Code:    -2103,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2104:
		return &BinanceError{
			Code:    -2104,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2105:
		return &BinanceError{
			Code:    -2105,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2106:
		return &BinanceError{
			Code:    -2106,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2107:
		return &BinanceError{
			Code:    -2107,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2108:
		return &BinanceError{
			Code:    -2108,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2109:
		return &BinanceError{
			Code:    -2109,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2110:
		return &BinanceError{
			Code:    -2110,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2111:
		return &BinanceError{
			Code:    -2111,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2112:
		return &BinanceError{
			Code:    -2112,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2113:
		return &BinanceError{
			Code:    -2113,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2114:
		return &BinanceError{
			Code:    -2114,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2115:
		return &BinanceError{
			Code:    -2115,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2116:
		return &BinanceError{
			Code:    -2116,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2117:
		return &BinanceError{
			Code:    -2117,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2118:
		return &BinanceError{
			Code:    -2118,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2119:
		return &BinanceError{
			Code:    -2119,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2120:
		return &BinanceError{
			Code:    -2120,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2121:
		return &BinanceError{
			Code:    -2121,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2122:
		return &BinanceError{
			Code:    -2122,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2123:
		return &BinanceError{
			Code:    -2123,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2124:
		return &BinanceError{
			Code:    -2124,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2125:
		return &BinanceError{
			Code:    -2125,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2126:
		return &BinanceError{
			Code:    -2126,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2127:
		return &BinanceError{
			Code:    -2127,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2128:
		return &BinanceError{
			Code:    -2128,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2129:
		return &BinanceError{
			Code:    -2129,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2130:
		return &BinanceError{
			Code:    -2130,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2131:
		return &BinanceError{
			Code:    -2131,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2132:
		return &BinanceError{
			Code:    -2132,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2133:
		return &BinanceError{
			Code:    -2133,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2134:
		return &BinanceError{
			Code:    -2134,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2135:
		return &BinanceError{
			Code:    -2135,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2136:
		return &BinanceError{
			Code:    -2136,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2137:
		return &BinanceError{
			Code:    -2137,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2138:
		return &BinanceError{
			Code:    -2138,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2139:
		return &BinanceError{
			Code:    -2139,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2140:
		return &BinanceError{
			Code:    -2140,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2141:
		return &BinanceError{
			Code:    -2141,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2142:
		return &BinanceError{
			Code:    -2142,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2143:
		return &BinanceError{
			Code:    -2143,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2144:
		return &BinanceError{
			Code:    -2144,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2145:
		return &BinanceError{
			Code:    -2145,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2146:
		return &BinanceError{
			Code:    -2146,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2147:
		return &BinanceError{
			Code:    -2147,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2148:
		return &BinanceError{
			Code:    -2148,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2149:
		return &BinanceError{
			Code:    -2149,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2150:
		return &BinanceError{
			Code:    -2150,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2151:
		return &BinanceError{
			Code:    -2151,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2152:
		return &BinanceError{
			Code:    -2152,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2153:
		return &BinanceError{
			Code:    -2153,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2154:
		return &BinanceError{
			Code:    -2154,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2155:
		return &BinanceError{
			Code:    -2155,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2156:
		return &BinanceError{
			Code:    -2156,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2157:
		return &BinanceError{
			Code:    -2157,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2158:
		return &BinanceError{
			Code:    -2158,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2159:
		return &BinanceError{
			Code:    -2159,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2160:
		return &BinanceError{
			Code:    -2160,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2161:
		return &BinanceError{
			Code:    -2161,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2162:
		return &BinanceError{
			Code:    -2162,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2163:
		return &BinanceError{
			Code:    -2163,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2164:
		return &BinanceError{
			Code:    -2164,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2165:
		return &BinanceError{
			Code:    -2165,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2166:
		return &BinanceError{
			Code:    -2166,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2167:
		return &BinanceError{
			Code:    -2167,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2168:
		return &BinanceError{
			Code:    -2168,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2169:
		return &BinanceError{
			Code:    -2169,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2170:
		return &BinanceError{
			Code:    -2170,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2171:
		return &BinanceError{
			Code:    -2171,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2172:
		return &BinanceError{
			Code:    -2172,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2173:
		return &BinanceError{
			Code:    -2173,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2174:
		return &BinanceError{
			Code:    -2174,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2175:
		return &BinanceError{
			Code:    -2175,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2176:
		return &BinanceError{
			Code:    -2176,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2177:
		return &BinanceError{
			Code:    -2177,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2178:
		return &BinanceError{
			Code:    -2178,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2179:
		return &BinanceError{
			Code:    -2179,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2180:
		return &BinanceError{
			Code:    -2180,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2181:
		return &BinanceError{
			Code:    -2181,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2182:
		return &BinanceError{
			Code:    -2182,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2183:
		return &BinanceError{
			Code:    -2183,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2184:
		return &BinanceError{
			Code:    -2184,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2185:
		return &BinanceError{
			Code:    -2185,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2186:
		return &BinanceError{
			Code:    -2186,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2187:
		return &BinanceError{
			Code:    -2187,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2188:
		return &BinanceError{
			Code:    -2188,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2189:
		return &BinanceError{
			Code:    -2189,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2190:
		return &BinanceError{
			Code:    -2190,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2191:
		return &BinanceError{
			Code:    -2191,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2192:
		return &BinanceError{
			Code:    -2192,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2193:
		return &BinanceError{
			Code:    -2193,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2194:
		return &BinanceError{
			Code:    -2194,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2195:
		return &BinanceError{
			Code:    -2195,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2196:
		return &BinanceError{
			Code:    -2196,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2197:
		return &BinanceError{
			Code:    -2197,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2198:
		return &BinanceError{
			Code:    -2198,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2199:
		return &BinanceError{
			Code:    -2199,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2200:
		return &BinanceError{
			Code:    -2200,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2201:
		return &BinanceError{
			Code:    -2201,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2202:
		return &BinanceError{
			Code:    -2202,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2203:
		return &BinanceError{
			Code:    -2203,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2204:
		return &BinanceError{
			Code:    -2204,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2205:
		return &BinanceError{
			Code:    -2205,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2206:
		return &BinanceError{
			Code:    -2206,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2207:
		return &BinanceError{
			Code:    -2207,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2208:
		return &BinanceError{
			Code:    -2208,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2209:
		return &BinanceError{
			Code:    -2209,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2210:
		return &BinanceError{
			Code:    -2210,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2211:
		return &BinanceError{
			Code:    -2211,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2212:
		return &BinanceError{
			Code:    -2212,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2213:
		return &BinanceError{
			Code:    -2213,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2214:
		return &BinanceError{
			Code:    -2214,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2215:
		return &BinanceError{
			Code:    -2215,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2216:
		return &BinanceError{
			Code:    -2216,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2217:
		return &BinanceError{
			Code:    -2217,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2218:
		return &BinanceError{
			Code:    -2218,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2219:
		return &BinanceError{
			Code:    -2219,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2220:
		return &BinanceError{
			Code:    -2220,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2221:
		return &BinanceError{
			Code:    -2221,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2222:
		return &BinanceError{
			Code:    -2222,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2223:
		return &BinanceError{
			Code:    -2223,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2224:
		return &BinanceError{
			Code:    -2224,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2225:
		return &BinanceError{
			Code:    -2225,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2226:
		return &BinanceError{
			Code:    -2226,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2227:
		return &BinanceError{
			Code:    -2227,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2228:
		return &BinanceError{
			Code:    -2228,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2229:
		return &BinanceError{
			Code:    -2229,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2230:
		return &BinanceError{
			Code:    -2230,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2231:
		return &BinanceError{
			Code:    -2231,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2232:
		return &BinanceError{
			Code:    -2232,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2233:
		return &BinanceError{
			Code:    -2233,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2234:
		return &BinanceError{
			Code:    -2234,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2235:
		return &BinanceError{
			Code:    -2235,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2236:
		return &BinanceError{
			Code:    -2236,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2237:
		return &BinanceError{
			Code:    -2237,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2238:
		return &BinanceError{
			Code:    -2238,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2239:
		return &BinanceError{
			Code:    -2239,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2240:
		return &BinanceError{
			Code:    -2240,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2241:
		return &BinanceError{
			Code:    -2241,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2242:
		return &BinanceError{
			Code:    -2242,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2243:
		return &BinanceError{
			Code:    -2243,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2244:
		return &BinanceError{
			Code:    -2244,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2245:
		return &BinanceError{
			Code:    -2245,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2246:
		return &BinanceError{
			Code:    -2246,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2247:
		return &BinanceError{
			Code:    -2247,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2248:
		return &BinanceError{
			Code:    -2248,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2249:
		return &BinanceError{
			Code:    -2249,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2250:
		return &BinanceError{
			Code:    -2250,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2251:
		return &BinanceError{
			Code:    -2251,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2252:
		return &BinanceError{
			Code:    -2252,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2253:
		return &BinanceError{
			Code:    -2253,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2254:
		return &BinanceError{
			Code:    -2254,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2255:
		return &BinanceError{
			Code:    -2255,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2256:
		return &BinanceError{
			Code:    -2256,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2257:
		return &BinanceError{
			Code:    -2257,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2258:
		return &BinanceError{
			Code:    -2258,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2259:
		return &BinanceError{
			Code:    -2259,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2260:
		return &BinanceError{
			Code:    -2260,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2261:
		return &BinanceError{
			Code:    -2261,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2262:
		return &BinanceError{
			Code:    -2262,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2263:
		return &BinanceError{
			Code:    -2263,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2264:
		return &BinanceError{
			Code:    -2264,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2265:
		return &BinanceError{
			Code:    -2265,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2266:
		return &BinanceError{
			Code:    -2266,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2267:
		return &BinanceError{
			Code:    -2267,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2268:
		return &BinanceError{
			Code:    -2268,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2269:
		return &BinanceError{
			Code:    -2269,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2270:
		return &BinanceError{
			Code:    -2270,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2271:
		return &BinanceError{
			Code:    -2271,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2272:
		return &BinanceError{
			Code:    -2272,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2273:
		return &BinanceError{
			Code:    -2273,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2274:
		return &BinanceError{
			Code:    -2274,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2275:
		return &BinanceError{
			Code:    -2275,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2276:
		return &BinanceError{
			Code:    -2276,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2277:
		return &BinanceError{
			Code:    -2277,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2278:
		return &BinanceError{
			Code:    -2278,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2279:
		return &BinanceError{
			Code:    -2279,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2280:
		return &BinanceError{
			Code:    -2280,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2281:
		return &BinanceError{
			Code:    -2281,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2282:
		return &BinanceError{
			Code:    -2282,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2283:
		return &BinanceError{
			Code:    -2283,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2284:
		return &BinanceError{
			Code:    -2284,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2285:
		return &BinanceError{
			Code:    -2285,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2286:
		return &BinanceError{
			Code:    -2286,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2287:
		return &BinanceError{
			Code:    -2287,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2288:
		return &BinanceError{
			Code:    -2288,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2289:
		return &BinanceError{
			Code:    -2289,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2290:
		return &BinanceError{
			Code:    -2290,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2291:
		return &BinanceError{
			Code:    -2291,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2292:
		return &BinanceError{
			Code:    -2292,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2293:
		return &BinanceError{
			Code:    -2293,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2294:
		return &BinanceError{
			Code:    -2294,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2295:
		return &BinanceError{
			Code:    -2295,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2296:
		return &BinanceError{
			Code:    -2296,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2297:
		return &BinanceError{
			Code:    -2297,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2298:
		return &BinanceError{
			Code:    -2298,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2299:
		return &BinanceError{
			Code:    -2299,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2300:
		return &BinanceError{
			Code:    -2300,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2301:
		return &BinanceError{
			Code:    -2301,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2302:
		return &BinanceError{
			Code:    -2302,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2303:
		return &BinanceError{
			Code:    -2303,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -2304:
		return &BinanceError{
			Code:    -2304,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2305:
		return &BinanceError{
			Code:    -2305,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2306:
		return &BinanceError{
			Code:    -2306,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2307:
		return &BinanceError{
			Code:    -2307,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2308:
		return &BinanceError{
			Code:    -2308,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2309:
		return &BinanceError{
			Code:    -2309,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2310:
		return &BinanceError{
			Code:    -2310,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2311:
		return &BinanceError{
			Code:    -2311,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2312:
		return &BinanceError{
			Code:    -2312,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2313:
		return &BinanceError{
			Code:    -2313,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2314:
		return &BinanceError{
			Code:    -2314,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2315:
		return &BinanceError{
			Code:    -2315,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2316:
		return &BinanceError{
			Code:    -2316,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2317:
		return &BinanceError{
			Code:    -2317,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2318:
		return &BinanceError{
			Code:    -2318,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2319:
		return &BinanceError{
			Code:    -2319,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2320:
		return &BinanceError{
			Code:    -2320,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2321:
		return &BinanceError{
			Code:    -2321,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2322:
		return &BinanceError{
			Code:    -2322,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2323:
		return &BinanceError{
			Code:    -2323,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2324:
		return &BinanceError{
			Code:    -2324,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2325:
		return &BinanceError{
			Code:    -2325,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2326:
		return &BinanceError{
			Code:    -2326,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2327:
		return &BinanceError{
			Code:    -2327,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2328:
		return &BinanceError{
			Code:    -2328,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2329:
		return &BinanceError{
			Code:    -2329,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2330:
		return &BinanceError{
			Code:    -2330,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2331:
		return &BinanceError{
			Code:    -2331,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2332:
		return &BinanceError{
			Code:    -2332,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2333:
		return &BinanceError{
			Code:    -2333,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2334:
		return &BinanceError{
			Code:    -2334,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2335:
		return &BinanceError{
			Code:    -2335,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2336:
		return &BinanceError{
			Code:    -2336,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2337:
		return &BinanceError{
			Code:    -2337,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2338:
		return &BinanceError{
			Code:    -2338,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2339:
		return &BinanceError{
			Code:    -2339,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2340:
		return &BinanceError{
			Code:    -2340,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2341:
		return &BinanceError{
			Code:    -2341,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2342:
		return &BinanceError{
			Code:    -2342,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2343:
		return &BinanceError{
			Code:    -2343,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2344:
		return &BinanceError{
			Code:    -2344,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2345:
		return &BinanceError{
			Code:    -2345,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2346:
		return &BinanceError{
			Code:    -2346,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2347:
		return &BinanceError{
			Code:    -2347,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2348:
		return &BinanceError{
			Code:    -2348,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2349:
		return &BinanceError{
			Code:    -2349,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2350:
		return &BinanceError{
			Code:    -2350,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2351:
		return &BinanceError{
			Code:    -2351,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2352:
		return &BinanceError{
			Code:    -2352,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2353:
		return &BinanceError{
			Code:    -2353,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2354:
		return &BinanceError{
			Code:    -2354,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2355:
		return &BinanceError{
			Code:    -2355,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2356:
		return &BinanceError{
			Code:    -2356,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2357:
		return &BinanceError{
			Code:    -2357,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2358:
		return &BinanceError{
			Code:    -2358,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2359:
		return &BinanceError{
			Code:    -2359,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2360:
		return &BinanceError{
			Code:    -2360,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2361:
		return &BinanceError{
			Code:    -2361,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2362:
		return &BinanceError{
			Code:    -2362,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2363:
		return &BinanceError{
			Code:    -2363,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2364:
		return &BinanceError{
			Code:    -2364,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2365:
		return &BinanceError{
			Code:    -2365,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2366:
		return &BinanceError{
			Code:    -2366,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2367:
		return &BinanceError{
			Code:    -2367,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2368:
		return &BinanceError{
			Code:    -2368,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2369:
		return &BinanceError{
			Code:    -2369,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2370:
		return &BinanceError{
			Code:    -2370,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2371:
		return &BinanceError{
			Code:    -2371,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2372:
		return &BinanceError{
			Code:    -2372,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2373:
		return &BinanceError{
			Code:    -2373,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2374:
		return &BinanceError{
			Code:    -2374,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2375:
		return &BinanceError{
			Code:    -2375,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2376:
		return &BinanceError{
			Code:    -2376,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2377:
		return &BinanceError{
			Code:    -2377,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2378:
		return &BinanceError{
			Code:    -2378,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2379:
		return &BinanceError{
			Code:    -2379,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2380:
		return &BinanceError{
			Code:    -2380,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2381:
		return &BinanceError{
			Code:    -2381,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2382:
		return &BinanceError{
			Code:    -2382,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2383:
		return &BinanceError{
			Code:    -2383,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2384:
		return &BinanceError{
			Code:    -2384,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2385:
		return &BinanceError{
			Code:    -2385,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2386:
		return &BinanceError{
			Code:    -2386,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2387:
		return &BinanceError{
			Code:    -2387,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2388:
		return &BinanceError{
			Code:    -2388,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2389:
		return &BinanceError{
			Code:    -2389,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2390:
		return &BinanceError{
			Code:    -2390,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2391:
		return &BinanceError{
			Code:    -2391,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2392:
		return &BinanceError{
			Code:    -2392,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2393:
		return &BinanceError{
			Code:    -2393,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2394:
		return &BinanceError{
			Code:    -2394,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2395:
		return &BinanceError{
			Code:    -2395,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2396:
		return &BinanceError{
			Code:    -2396,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2397:
		return &BinanceError{
			Code:    -2397,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2398:
		return &BinanceError{
			Code:    -2398,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2399:
		return &BinanceError{
			Code:    -2399,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -2400:
		return &BinanceError{
			Code:    -2400,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4014:
		return &BinanceError{
			Code:    -4014,
			Message: "Price not increased by tick size",
			Details: "Order price does not match the required tick size for this symbol",
			Solution: "Adjust price to match the symbol's tick size. Use exchange info to get the correct price precision and tick size",
			Critical: false,
		}
	case -4061:
		return &BinanceError{
			Code:    -4061,
			Message: "Position side does not match user's setting",
			Details: "Order position side (LONG/SHORT) does not match your account position mode setting",
			Solution: "Check your Binance Futures account position mode (Hedge vs One-Way). Go to Binance Futures > Preferences > Position Mode and ensure it matches your order type. OR update your orders to match your current position mode",
			Critical: true,
		}
	case -4065:
		return &BinanceError{
			Code:    -4065,
			Message: "Position side mismatch",
			Details: "Position side mismatch between order and position",
			Solution: "Check existing positions. Ensure order side matches position side",
			Critical: true,
		}
	case -4066:
		return &BinanceError{
			Code:    -4066,
			Message: "Position limit exceeded",
			Details: "Maximum position limit exceeded",
			Solution: "Reduce position size or close existing positions",
			Critical: true,
		}
	case -4067:
		return &BinanceError{
			Code:    -4067,
			Message: "Notional value limit exceeded",
			Details: "Notional value limit exceeded",
			Solution: "Reduce position size. Check your account's notional value limits",
			Critical: true,
		}
	case -4068:
		return &BinanceError{
			Code:    -4068,
			Message: "Maintenance margin exceeded",
			Details: "Maintenance margin requirement exceeded",
			Solution: "Reduce position size or add more margin to your account",
			Critical: true,
		}
	case -4069:
		return &BinanceError{
			Code:    -4069,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4070:
		return &BinanceError{
			Code:    -4070,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4071:
		return &BinanceError{
			Code:    -4071,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4072:
		return &BinanceError{
			Code:    -4072,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4073:
		return &BinanceError{
			Code:    -4073,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4074:
		return &BinanceError{
			Code:    -4074,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4075:
		return &BinanceError{
			Code:    -4075,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4076:
		return &BinanceError{
			Code:    -4076,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4077:
		return &BinanceError{
			Code:    -4077,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4078:
		return &BinanceError{
			Code:    -4078,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4079:
		return &BinanceError{
			Code:    -4079,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4080:
		return &BinanceError{
			Code:    -4080,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4081:
		return &BinanceError{
			Code:    -4081,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4082:
		return &BinanceError{
			Code:    -4082,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4083:
		return &BinanceError{
			Code:    -4083,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4084:
		return &BinanceError{
			Code:    -4084,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4085:
		return &BinanceError{
			Code:    -4085,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4086:
		return &BinanceError{
			Code:    -4086,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4087:
		return &BinanceError{
			Code:    -4087,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4088:
		return &BinanceError{
			Code:    -4088,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4089:
		return &BinanceError{
			Code:    -4089,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4090:
		return &BinanceError{
			Code:    -4090,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4091:
		return &BinanceError{
			Code:    -4091,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4092:
		return &BinanceError{
			Code:    -4092,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4093:
		return &BinanceError{
			Code:    -4093,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4094:
		return &BinanceError{
			Code:    -4094,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4095:
		return &BinanceError{
			Code:    -4095,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4096:
		return &BinanceError{
			Code:    -4096,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4097:
		return &BinanceError{
			Code:    -4097,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4098:
		return &BinanceError{
			Code:    -4098,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4099:
		return &BinanceError{
			Code:    -4099,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4100:
		return &BinanceError{
			Code:    -4100,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4101:
		return &BinanceError{
			Code:    -4101,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4102:
		return &BinanceError{
			Code:    -4102,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4103:
		return &BinanceError{
			Code:    -4103,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set position mode first, then change leverage",
			Critical: false,
		}
	case -4104:
		return &BinanceError{
			Code:    -4104,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4105:
		return &BinanceError{
			Code:    -4105,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4106:
		return &BinanceError{
			Code:    -4106,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4107:
		return &BinanceError{
			Code:    -4107,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4108:
		return &BinanceError{
			Code:    -4108,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4109:
		return &BinanceError{
			Code:    -4109,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4110:
		return &BinanceError{
			Code:    -4110,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4111:
		return &BinanceError{
			Code:    -4111,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4112:
		return &BinanceError{
			Code:    -4112,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4113:
		return &BinanceError{
			Code:    -4113,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4114:
		return &BinanceError{
			Code:    -4114,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4115:
		return &BinanceError{
			Code:    -4115,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4116:
		return &BinanceError{
			Code:    -4116,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4117:
		return &BinanceError{
			Code:    -4117,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4118:
		return &BinanceError{
			Code:    -4118,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4119:
		return &BinanceError{
			Code:    -4119,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4120:
		return &BinanceError{
			Code:    -4120,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4121:
		return &BinanceError{
			Code:    -4121,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4122:
		return &BinanceError{
			Code:    -4122,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4123:
		return &BinanceError{
			Code:    -4123,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4124:
		return &BinanceError{
			Code:    -4124,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4125:
		return &BinanceError{
			Code:    -4125,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4126:
		return &BinanceError{
			Code:    -4126,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4127:
		return &BinanceError{
			Code:    -4127,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4128:
		return &BinanceError{
			Code:    -4128,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4129:
		return &BinanceError{
			Code:    -4129,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4130:
		return &BinanceError{
			Code:    -4130,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4131:
		return &BinanceError{
			Code:    -4131,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4132:
		return &BinanceError{
			Code:    -4132,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4133:
		return &BinanceError{
			Code:    -4133,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4134:
		return &BinanceError{
			Code:    -4134,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4135:
		return &BinanceError{
			Code:    -4135,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4136:
		return &BinanceError{
			Code:    -4136,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4137:
		return &BinanceError{
			Code:    -4137,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4138:
		return &BinanceError{
			Code:    -4138,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4139:
		return &BinanceError{
			Code:    -4139,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4140:
		return &BinanceError{
			Code:    -4140,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4141:
		return &BinanceError{
			Code:    -4141,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4142:
		return &BinanceError{
			Code:    -4142,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4143:
		return &BinanceError{
			Code:    -4143,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4144:
		return &BinanceError{
			Code:    -4144,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4145:
		return &BinanceError{
			Code:    -4145,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4146:
		return &BinanceError{
			Code:    -4146,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4147:
		return &BinanceError{
			Code:    -4147,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4148:
		return &BinanceError{
			Code:    -4148,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4149:
		return &BinanceError{
			Code:    -4149,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4150:
		return &BinanceError{
			Code:    -4150,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4151:
		return &BinanceError{
			Code:    -4151,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4152:
		return &BinanceError{
			Code:    -4152,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4153:
		return &BinanceError{
			Code:    -4153,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4154:
		return &BinanceError{
			Code:    -4154,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4155:
		return &BinanceError{
			Code:    -4155,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4156:
		return &BinanceError{
			Code:    -4156,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4157:
		return &BinanceError{
			Code:    -4157,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4158:
		return &BinanceError{
			Code:    -4158,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4159:
		return &BinanceError{
			Code:    -4159,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4160:
		return &BinanceError{
			Code:    -4160,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4161:
		return &BinanceError{
			Code:    -4161,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4162:
		return &BinanceError{
			Code:    -4162,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4163:
		return &BinanceError{
			Code:    -4163,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4164:
		return &BinanceError{
			Code:    -4164,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4165:
		return &BinanceError{
			Code:    -4165,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4166:
		return &BinanceError{
			Code:    -4166,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4167:
		return &BinanceError{
			Code:    -4167,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4168:
		return &BinanceError{
			Code:    -4168,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4169:
		return &BinanceError{
			Code:    -4169,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4170:
		return &BinanceError{
			Code:    -4170,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4171:
		return &BinanceError{
			Code:    -4171,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4172:
		return &BinanceError{
			Code:    -4172,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4173:
		return &BinanceError{
			Code:    -4173,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4174:
		return &BinanceError{
			Code:    -4174,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4175:
		return &BinanceError{
			Code:    -4175,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4176:
		return &BinanceError{
			Code:    -4176,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4177:
		return &BinanceError{
			Code:    -4177,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4178:
		return &BinanceError{
			Code:    -4178,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4179:
		return &BinanceError{
			Code:    -4179,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4180:
		return &BinanceError{
			Code:    -4180,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4181:
		return &BinanceError{
			Code:    -4181,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4182:
		return &BinanceError{
			Code:    -4182,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4183:
		return &BinanceError{
			Code:    -4183,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4184:
		return &BinanceError{
			Code:    -4184,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4185:
		return &BinanceError{
			Code:    -4185,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4186:
		return &BinanceError{
			Code:    -4186,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4187:
		return &BinanceError{
			Code:    -4187,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4188:
		return &BinanceError{
			Code:    -4188,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4189:
		return &BinanceError{
			Code:    -4189,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4190:
		return &BinanceError{
			Code:    -4190,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4191:
		return &BinanceError{
			Code:    -4191,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4192:
		return &BinanceError{
			Code:    -4192,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4193:
		return &BinanceError{
			Code:    -4193,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4194:
		return &BinanceError{
			Code:    -4194,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4195:
		return &BinanceError{
			Code:    -4195,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4196:
		return &BinanceError{
			Code:    -4196,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4197:
		return &BinanceError{
			Code:    -4197,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4198:
		return &BinanceError{
			Code:    -4198,
			Message: "Leverage not changed",
			Details: "Leverage not changed because margin mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	case -4199:
		return &BinanceError{
			Code:    -4199,
			Message: "Leverage not changed",
			Details: "Leverage not changed because position mode is changed",
			Solution: "Set margin mode first, then change leverage",
			Critical: false,
		}
	default:
		// Check for common error patterns in message
		if contains(errMsg, "precision") {
			return &BinanceError{
				Code:    code,
				Message: "Precision error",
				Details: "Order precision does not match requirements",
				Solution: "Check order precision. Ensure quantity and price match symbol's precision requirements",
				Critical: false,
			}
		}
		if contains(errMsg, "tick") {
			return &BinanceError{
				Code:    code,
				Message: "Tick size error",
				Details: "Order price does not match tick size",
				Solution: "Adjust price to match the symbol's tick size. Use exchange info to get the correct tick size",
				Critical: false,
			}
		}
		if contains(errMsg, "balance") || contains(errMsg, "insufficient") {
			return &BinanceError{
				Code:    code,
				Message: "Insufficient balance",
				Details: "Account has insufficient balance or margin",
				Solution: "Check account balance. Ensure you have enough funds or margin for the order",
				Critical: true,
			}
		}
		if contains(errMsg, "network") || contains(errMsg, "connection") || contains(errMsg, "timeout") {
			return &BinanceError{
				Code:    code,
				Message: "Network/Connection error",
				Details: "Network or connection issue occurred",
				Solution: "Check network connection. Retry the request. If persists, check Binance service status",
				Critical: false,
			}
		}
		if contains(errMsg, "rate") {
			return &BinanceError{
				Code:    code,
				Message: "Rate limit exceeded",
				Details: "API request rate limit exceeded",
				Solution: "Wait and retry. Reduce request frequency. Check your account's rate limits",
				Critical: false,
			}
		}
		if contains(errMsg, "symbol") {
			return &BinanceError{
				Code:    code,
				Message: "Symbol error",
				Details: "Symbol-related error occurred",
				Solution: "Check if the symbol is valid and actively trading on Binance Futures",
				Critical: true,
			}
		}
		if contains(errMsg, "leverage") {
			return &BinanceError{
				Code:    code,
				Message: "Leverage error",
				Details: "Leverage-related error occurred",
				Solution: "Check leverage settings. Ensure leverage is within allowed range for the symbol",
				Critical: false,
			}
		}
		if contains(errMsg, "position") {
			return &BinanceError{
				Code:    code,
				Message: "Position error",
				Details: "Position-related error occurred",
				Solution: "Check existing positions. Ensure position mode (Hedge/One-Way) matches order type",
				Critical: true,
			}
		}
		if contains(errMsg, "margin") {
			return &BinanceError{
				Code:    code,
				Message: "Margin error",
				Details: "Margin-related error occurred",
				Solution: "Check available margin. Reduce position size or add more margin",
				Critical: true,
			}
		}
		// Generic error
		return &BinanceError{
			Code:    code,
			Message: "Unknown error",
			Details: errMsg,
			Solution: "Check error message for details. If issue persists, contact support",
			Critical: false,
		}
	}
}

// isRetryableError checks if error is retryable
func (e *SelfOptimizingExecutor) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorDetails := GetBinanceErrorDetails(err)
	
	// Non-retryable errors
	nonRetryableCodes := []int{
		-1002, // Unauthorized
		-1013, // Invalid quantity
		-1014, // Invalid order type
		-1022, // Invalid signature
		-1116, // No such ticker
		-1117, // Invalid API key
		-1118, // Invalid IP
		-1125, // Invalid API key type
		-2018, // Balance insufficient
		-2019, // Margin insufficient
		-2020, // Account has insufficient balance
		-2022, // Reduce only rejected
		-2061, // Position side does not match
		-2062, // Position side mismatch
		-2063, // Position limit exceeded
		-2064, // Notional value limit exceeded
		-2065, // Maintenance margin exceeded
		-4014, // Price not increased by tick size
		-4061, // Position side does not match user's setting
		-4065, // Position side mismatch
		-4066, // Position limit exceeded
		-4067, // Notional value limit exceeded
		-4068, // Maintenance margin exceeded
	}

	for _, code := range nonRetryableCodes {
		if errorDetails.Code == code {
			return false
		}
	}

	// Retry on network errors and rate limits
	retryablePatterns := []string{
		"timeout",
		"rate limit",
		"network",
		"connection",
		"temporary",
		"disconnected",
		"unexpected response",
		"too many requests",
	}

	for _, pattern := range retryablePatterns {
		if contains(err.Error(), pattern) {
			return true
		}
	}

	return false
}

// LogErrorDetails logs detailed error information
func (e *SelfOptimizingExecutor) LogErrorDetails(err error, symbol string, side futures.SideType, quantity float64, price float64) {
	if err == nil {
		return
	}

	errorDetails := GetBinanceErrorDetails(err)

	logrus.WithFields(logrus.Fields{
		"error_code":    errorDetails.Code,
		"error_message": errorDetails.Message,
		"symbol":        symbol,
		"side":          side,
		"quantity":      quantity,
		"price":         price,
		"critical":      errorDetails.Critical,
	}).Error("🚨 Binance API Error")

	logrus.WithFields(logrus.Fields{
		"error_code": errorDetails.Code,
	}).Error("📋 Error Details: " + errorDetails.Details)

	logrus.WithFields(logrus.Fields{
		"error_code": errorDetails.Code,
	}).Error("💡 Solution: " + errorDetails.Solution)

	if errorDetails.Critical {
		logrus.Error("⚠️ CRITICAL ERROR - Requires immediate attention")
	}
}

// priceToUSD estimates price in USD (simplified)
func (e *SelfOptimizingExecutor) priceToUSD(symbol string, price float64) float64 {
	// Simplified: assume USDT pairs
	return price
}

// GetMetrics returns current execution metrics
func (e *SelfOptimizingExecutor) GetMetrics() *ExecutionMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return &ExecutionMetrics{
		TotalOrders:            e.metrics.TotalOrders,
		SuccessfulOrders:       e.metrics.SuccessfulOrders,
		FailedOrders:           e.metrics.FailedOrders,
		AverageSlippage:        e.metrics.AverageSlippage,
		AverageFillTime:        e.metrics.AverageFillTime,
		AverageExecutionQuality: e.metrics.AverageExecutionQuality,
		MarketOrderCount:       e.metrics.MarketOrderCount,
		LimitOrderCount:        e.metrics.LimitOrderCount,
		LastOptimization:       e.metrics.LastOptimization,
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}