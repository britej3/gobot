package position

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/sirupsen/logrus"
)

// DynamicConfig holds dynamic position sizing and leveraging configuration
type DynamicConfig struct {
	// Position Sizing
	MinPositionSizeUSD   float64 `json:"min_position_size_usd"`   // Minimum position size ($10)
	MaxPositionSizeUSD   float64 `json:"max_position_size_usd"`   // Maximum position size ($40)
	MaxTotalExposureUSD  float64 `json:"max_total_exposure_usd"`  // Maximum total exposure ($100)
	MaxPositions         int     `json:"max_positions"`           // Maximum concurrent positions (3)

	// Risk Tolerance
	BaseRiskPercent      float64 `json:"base_risk_percent"`       // Base risk per trade (2%)
	MaxRiskPercent       float64 `json:"max_risk_percent"`        // Maximum risk per trade (5%)
	RiskToleranceMode    string  `json:"risk_tolerance_mode"`     // "conservative", "moderate", "aggressive", "high"

	// Leverage
	MinLeverage          int     `json:"min_leverage"`            // Minimum leverage (5)
	MaxLeverage          int     `json:"max_leverage"`            // Maximum leverage (25)
	BaseLeverage         int     `json:"base_leverage"`           // Base leverage (10)
	HighRiskMaxLeverage  int     `json:"high_risk_max_leverage"`  // High risk max leverage (50)

	// Kelly Criterion
	KellyMultiplier       float64 `json:"kelly_multiplier"`        // Kelly multiplier (0.5 = half Kelly)
	KellyFraction         float64 `json:"kelly_fraction"`          // Fixed Kelly fraction (0.02)
	EnableKelly           bool    `json:"enable_kelly"`            // Enable Kelly criterion
	TakeProfitMultiplier  float64 `json:"take_profit_multiplier"`  // Take profit multiplier (2.0 = 2:1 risk-reward)

	// Confidence-Based Sizing
	EnableConfidenceSizing bool  `json:"enable_confidence_sizing"` // Enable confidence-based sizing
	MinConfidence          float64 `json:"min_confidence"`           // Minimum confidence for trade (0.65)
	MaxConfidence          float64 `json:"max_confidence"`           // Maximum confidence (1.0)

	// Volatility Adjustment
	EnableVolatilityAdjustment bool    `json:"enable_volatility_adjustment"` // Enable volatility adjustment
	LowVolatilityThreshold     float64 `json:"low_volatility_threshold"`      // Low volatility threshold (<0.5%)
	HighVolatilityThreshold    float64 `json:"high_volatility_threshold"`     // High volatility threshold (>3%)

	// Self-Optimization
	EnableSelfOptimization   bool          `json:"enable_self_optimization"`   // Enable self-optimization
	OptimizationInterval     time.Duration `json:"optimization_interval"`       // Optimization interval (1 hour)
	PerformanceWindow        time.Duration `json:"performance_window"`          // Performance window (24 hours)
	MinTradesForOptimization int           `json:"min_trades_for_optimization"` // Min trades for optimization (10)
}

// DefaultDynamicConfig returns default dynamic configuration
func DefaultDynamicConfig() DynamicConfig {
	return DynamicConfig{
		MinPositionSizeUSD:   8.0,  // 8 USDT minimum
		MaxPositionSizeUSD:   13.0, // 13 USDT maximum (50% of balance)
		MaxTotalExposureUSD:  26.0, // 26 USDT total
		MaxPositions:         3,

		BaseRiskPercent:      0.03, // 3% (more aggressive)
		MaxRiskPercent:       0.05, // 5%
		RiskToleranceMode:    "aggressive",

		MinLeverage:          10, // Higher minimum
		MaxLeverage:          30, // Higher maximum
		BaseLeverage:         15, // Higher base
		HighRiskMaxLeverage:  50, // Very high max

		KellyMultiplier:       0.6, // More aggressive Kelly
		KellyFraction:         0.03, // Higher fixed fraction
		EnableKelly:           true,

		EnableConfidenceSizing: true,
		MinConfidence:          0.60, // Lower threshold
		MaxConfidence:          1.0,

		EnableVolatilityAdjustment: true,
		LowVolatilityThreshold:     0.003,  // 0.3% (lower threshold)
		HighVolatilityThreshold:    0.04,   // 4% (higher threshold)

		EnableSelfOptimization:   true,
		OptimizationInterval:     1 * time.Hour,
		PerformanceWindow:        24 * time.Hour,
		MinTradesForOptimization: 10,
	}
}

// HighRiskDynamicConfig returns high risk dynamic configuration
func HighRiskDynamicConfig() DynamicConfig {
	return DynamicConfig{
		MinPositionSizeUSD:   8.0,  // 8 USDT minimum
		MaxPositionSizeUSD:   18.0, // 18 USDT maximum (70% of balance)
		MaxTotalExposureUSD:  26.0, // 26 USDT total
		MaxPositions:         3,

		BaseRiskPercent:      0.04, // 4% (very aggressive)
		MaxRiskPercent:       0.08, // 8% (much higher)
		RiskToleranceMode:    "high",

		MinLeverage:          15, // Higher minimum
		MaxLeverage:          50, // Much higher maximum
		BaseLeverage:         25, // Higher base
		HighRiskMaxLeverage:  75, // Very high max

		KellyMultiplier:       0.9, // Very aggressive Kelly (0.9 = 90% Kelly)
		KellyFraction:         0.04, // Higher fixed fraction
		EnableKelly:           true,

		EnableConfidenceSizing: true,
		MinConfidence:          0.55, // Even lower threshold
		MaxConfidence:          1.0,

		EnableVolatilityAdjustment: true,
		LowVolatilityThreshold:     0.002, // 0.2% (very low threshold)
		HighVolatilityThreshold:    0.05, // 5% (very high threshold)

		EnableSelfOptimization:   true,
		OptimizationInterval:     30 * time.Minute, // More frequent optimization
		PerformanceWindow:        12 * time.Hour,   // Shorter window
		MinTradesForOptimization: 5,               // Fewer trades needed
	}
}

// PerformanceMetrics tracks trading performance for self-optimization
type PerformanceMetrics struct {
	TotalTrades        int       `json:"total_trades"`
	WinningTrades      int       `json:"winning_trades"`
	LosingTrades       int       `json:"losing_trades"`
	WinRate            float64   `json:"win_rate"`
	AvgWin             float64   `json:"avg_win"`
	AvgLoss            float64   `json:"avg_loss"`
	ProfitFactor       float64   `json:"profit_factor"`
	MaxDrawdown        float64   `json:"max_drawdown"`
	TotalPnL           float64   `json:"total_pnl"`
	TradesWindow       []Trade   `json:"trades_window"`
	LastOptimization   time.Time `json:"last_optimization"`
}

// Trade represents a completed trade for performance tracking
type Trade struct {
	Symbol        string
	Side          string
	EntryPrice    float64
	ExitPrice     float64
	Quantity      float64
	PnL           float64
	PnLPercent    float64
	Leverage      int
	Confidence    float64
	Volatility    float64
	EntryTime     time.Time
	ExitTime      time.Time
	Duration      time.Duration
	ExitReason    string
}

// DynamicManager manages dynamic position sizing and leveraging
type DynamicManager struct {
	client              *futures.Client
	config              DynamicConfig
	performance         PerformanceMetrics
	mu                  sync.RWMutex
	accountBalance      float64
	currentExposureUSD  float64
	openPositions       int
	stopChan            chan struct{}
	isRunning           bool
	volatilityCache     map[string]float64
	lastVolatilityUpdate time.Time
}

// NewDynamicManager creates a new dynamic manager
func NewDynamicManager(client *futures.Client, config DynamicConfig) *DynamicManager {
	return &DynamicManager{
		client:            client,
		config:            config,
		performance:       PerformanceMetrics{
			TradesWindow: make([]Trade, 0),
		},
		stopChan:           make(chan struct{}),
		volatilityCache:    make(map[string]float64),
	}
}

// Start begins dynamic management
func (dm *DynamicManager) Start(ctx context.Context) error {
	logrus.Info("📊 Starting dynamic manager...")

	dm.isRunning = true

	// Initialize account balance
	if err := dm.updateAccountBalance(ctx); err != nil {
		logrus.WithError(err).Warn("Failed to get initial account balance")
	}

	// Start self-optimization if enabled
	if dm.config.EnableSelfOptimization {
		go dm.runSelfOptimization(ctx)
	}

	logrus.Info("✅ Dynamic manager started")
	return nil
}

// Stop gracefully stops the dynamic manager
func (dm *DynamicManager) Stop() error {
	logrus.Info("🛑 Stopping dynamic manager...")
	dm.isRunning = false
	close(dm.stopChan)
	return nil
}

// CalculatePositionSize calculates optimal position size based on multiple factors
func (dm *DynamicManager) CalculatePositionSize(ctx context.Context, symbol string, entryPrice, stopLoss, confidence, volatility float64) (float64, int, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// Update account balance
	dm.updateAccountBalance(ctx)

	// Calculate base position size based on risk
	basePositionSize := dm.calculateRiskBasedPositionSize(entryPrice, stopLoss)

	// Apply confidence-based adjustment
	if dm.config.EnableConfidenceSizing {
		basePositionSize = dm.applyConfidenceAdjustment(basePositionSize, confidence)
	}

	// Apply volatility adjustment
	if dm.config.EnableVolatilityAdjustment {
		basePositionSize = dm.applyVolatilityAdjustment(basePositionSize, volatility)
	}

	// Apply Kelly criterion if enabled
	if dm.config.EnableKelly {
		kellySize := dm.calculateKellyPositionSize(entryPrice, stopLoss, confidence, volatility)
		// Blend base and Kelly sizes
		basePositionSize = (basePositionSize + kellySize) / 2.0
	}

	// Calculate optimal leverage
	optimalLeverage := dm.calculateOptimalLeverage(confidence, volatility)

	// Apply leverage to position size
	leveragedPositionSize := basePositionSize * float64(optimalLeverage)

	// Convert to quantity
	quantity := leveragedPositionSize / entryPrice

	// Apply position size limits
	minQuantity := dm.config.MinPositionSizeUSD / entryPrice
	maxQuantity := dm.config.MaxPositionSizeUSD / entryPrice

	if quantity < minQuantity {
		quantity = minQuantity
	}
	if quantity > maxQuantity {
		quantity = maxQuantity
	}

	// Check total exposure
	if dm.currentExposureUSD+leveragedPositionSize > dm.config.MaxTotalExposureUSD {
		// Reduce position size to fit within total exposure
		availableExposure := dm.config.MaxTotalExposureUSD - dm.currentExposureUSD
		quantity = availableExposure / entryPrice
	}

	// Check max positions
	if dm.openPositions >= dm.config.MaxPositions {
		return 0, 0, fmt.Errorf("max positions (%d) reached", dm.config.MaxPositions)
	}

	logrus.WithFields(logrus.Fields{
		"symbol":             symbol,
		"base_position_size": basePositionSize,
		"optimal_leverage":   optimalLeverage,
		"leveraged_size":     leveragedPositionSize,
		"quantity":           quantity,
		"confidence":         confidence,
		"volatility":         volatility,
	}).Info("📊 Dynamic position size calculated")

	return quantity, optimalLeverage, nil
}

// calculateRiskBasedPositionSize calculates position size based on risk
func (dm *DynamicManager) calculateRiskBasedPositionSize(entryPrice, stopLoss float64) float64 {
	riskPercent := dm.getEffectiveRiskPercent()
	riskAmount := dm.accountBalance * riskPercent

	riskPerUnit := math.Abs(entryPrice - stopLoss)
	if riskPerUnit == 0 {
		return 0
	}

	positionSize := riskAmount / riskPerUnit

	// Ensure minimum position size
	minSize := dm.config.MinPositionSizeUSD
	if positionSize < minSize {
		positionSize = minSize
	}

	return positionSize
}

// applyConfidenceAdjustment adjusts position size based on confidence
func (dm *DynamicManager) applyConfidenceAdjustment(positionSize, confidence float64) float64 {
	// Normalize confidence to 0-1 range
	normalizedConf := (confidence - dm.config.MinConfidence) / (dm.config.MaxConfidence - dm.config.MinConfidence)
	if normalizedConf < 0 {
		normalizedConf = 0
	}
	if normalizedConf > 1 {
		normalizedConf = 1
	}

	// Apply confidence multiplier (0.5x to 1.5x)
	confidenceMultiplier := 0.5 + (normalizedConf * 1.0)

	return positionSize * confidenceMultiplier
}

// applyVolatilityAdjustment adjusts position size based on volatility
func (dm *DynamicManager) applyVolatilityAdjustment(positionSize, volatility float64) float64 {
	adjustment := 1.0

	if volatility < dm.config.LowVolatilityThreshold {
		// Low volatility: increase position size (up to 2x)
		adjustment = 1.0 + (1.0 - volatility/dm.config.LowVolatilityThreshold)
	} else if volatility > dm.config.HighVolatilityThreshold {
		// High volatility: decrease position size (down to 0.5x)
		adjustment = 1.0 - ((volatility - dm.config.HighVolatilityThreshold) / dm.config.HighVolatilityThreshold)
		if adjustment < 0.5 {
			adjustment = 0.5
		}
	}

	return positionSize * adjustment
}

// calculateKellyPositionSize calculates position size using Kelly criterion
func (dm *DynamicManager) calculateKellyPositionSize(entryPrice, stopLoss, confidence, volatility float64) float64 {
	// Use confidence as win rate proxy
	winRate := confidence

	// Estimate average win/loss ratio based on risk-reward
	avgWin := dm.config.TakeProfitMultiplier // Default 2:1 risk-reward
	avgLoss := 1.0

	// Calculate Kelly fraction
	kellyFraction := (winRate*avgWin - (1-winRate)*avgLoss) / avgWin

	// Apply Kelly multiplier
	kellyFraction *= dm.config.KellyMultiplier

	// Use fixed fraction if Kelly is negative
	if kellyFraction < 0 {
		kellyFraction = dm.config.KellyFraction
	}

	// Calculate position size
	riskAmount := dm.accountBalance * kellyFraction
	riskPerUnit := math.Abs(entryPrice - stopLoss)

	if riskPerUnit == 0 {
		return 0
	}

	positionSize := riskAmount / riskPerUnit

	// Apply volatility adjustment to Kelly
	if volatility > dm.config.HighVolatilityThreshold {
		positionSize *= 0.5 // Reduce Kelly in high volatility
	}

	return positionSize
}

// calculateOptimalLeverage calculates optimal leverage based on confidence and volatility
func (dm *DynamicManager) calculateOptimalLeverage(confidence, volatility float64) int {
	baseLeverage := dm.config.BaseLeverage

	// Adjust for confidence
	if dm.config.EnableConfidenceSizing {
		normalizedConf := (confidence - dm.config.MinConfidence) / (dm.config.MaxConfidence - dm.config.MinConfidence)
		if normalizedConf < 0 {
			normalizedConf = 0
		}
		if normalizedConf > 1 {
			normalizedConf = 1
		}

		// Confidence multiplier (0.5x to 2x)
		confMultiplier := 0.5 + (normalizedConf * 1.5)
		baseLeverage = int(float64(baseLeverage) * confMultiplier)
	}

	// Adjust for volatility
	if dm.config.EnableVolatilityAdjustment {
		if volatility < dm.config.LowVolatilityThreshold {
			// Low volatility: increase leverage
			baseLeverage = int(float64(baseLeverage) * 1.5)
		} else if volatility > dm.config.HighVolatilityThreshold {
			// High volatility: decrease leverage
			baseLeverage = int(float64(baseLeverage) * 0.5)
		}
	}

	// Apply risk tolerance mode
	switch dm.config.RiskToleranceMode {
	case "conservative":
		baseLeverage = int(float64(baseLeverage) * 0.5)
	case "aggressive":
		baseLeverage = int(float64(baseLeverage) * 1.5)
	case "high":
		baseLeverage = int(float64(baseLeverage) * 2.0)
	}

	// Ensure within bounds
	if baseLeverage < dm.config.MinLeverage {
		baseLeverage = dm.config.MinLeverage
	}
	if baseLeverage > dm.config.HighRiskMaxLeverage && dm.config.RiskToleranceMode == "high" {
		baseLeverage = dm.config.HighRiskMaxLeverage
	} else if baseLeverage > dm.config.MaxLeverage {
		baseLeverage = dm.config.MaxLeverage
	}

	return baseLeverage
}

// RecordTrade records a completed trade for performance tracking
func (dm *DynamicManager) RecordTrade(trade Trade) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.performance.TotalTrades++
	dm.performance.TradesWindow = append(dm.performance.TradesWindow, trade)

	if trade.PnL > 0 {
		dm.performance.WinningTrades++
		dm.performance.AvgWin = (dm.performance.AvgWin*float64(dm.performance.WinningTrades-1) + trade.PnL) / float64(dm.performance.WinningTrades)
	} else {
		dm.performance.LosingTrades++
		dm.performance.AvgLoss = (dm.performance.AvgLoss*float64(dm.performance.LosingTrades-1) + math.Abs(trade.PnL)) / float64(dm.performance.LosingTrades)
	}

	dm.performance.TotalPnL += trade.PnL
	dm.performance.WinRate = float64(dm.performance.WinningTrades) / float64(dm.performance.TotalTrades)

	// Calculate profit factor
	if dm.performance.AvgLoss > 0 {
		dm.performance.ProfitFactor = (dm.performance.AvgWin * float64(dm.performance.WinningTrades)) / (dm.performance.AvgLoss * float64(dm.performance.LosingTrades))
	}

	// Calculate max drawdown
	if dm.performance.TotalPnL < dm.performance.MaxDrawdown {
		dm.performance.MaxDrawdown = dm.performance.TotalPnL
	}

	// Clean up old trades outside performance window
	cutoffTime := time.Now().Add(-dm.config.PerformanceWindow)
	trades := make([]Trade, 0, len(dm.performance.TradesWindow))
	for _, t := range dm.performance.TradesWindow {
		if t.ExitTime.After(cutoffTime) {
			trades = append(trades, t)
		}
	}
	dm.performance.TradesWindow = trades

	logrus.WithFields(logrus.Fields{
		"total_trades":  dm.performance.TotalTrades,
		"win_rate":      dm.performance.WinRate,
		"total_pnl":     dm.performance.TotalPnL,
		"profit_factor": dm.performance.ProfitFactor,
		"max_drawdown":  dm.performance.MaxDrawdown,
	}).Info("📊 Trade recorded, performance updated")
}

// runSelfOptimization runs periodic self-optimization
func (dm *DynamicManager) runSelfOptimization(ctx context.Context) {
	ticker := time.NewTicker(dm.config.OptimizationInterval)
	defer ticker.Stop()

	logrus.Info("📊 Self-optimization loop started")

	for dm.isRunning {
		select {
		case <-ticker.C:
			if err := dm.optimizeParameters(ctx); err != nil {
				logrus.WithError(err).Error("Failed to optimize parameters")
			}

		case <-dm.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// optimizeParameters optimizes trading parameters based on performance
func (dm *DynamicManager) optimizeParameters(ctx context.Context) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Check if we have enough trades
	if len(dm.performance.TradesWindow) < dm.config.MinTradesForOptimization {
		logrus.Debug("📊 Not enough trades for optimization")
		return nil
	}

	logrus.Info("📊 Running self-optimization...")

	// Analyze performance and adjust parameters
	originalConfig := dm.config

	// Adjust risk tolerance based on win rate
	if dm.performance.WinRate > 0.7 {
		// High win rate: increase risk
		dm.config.BaseRiskPercent = math.Min(dm.config.MaxRiskPercent, dm.config.BaseRiskPercent*1.2)
		dm.config.RiskToleranceMode = "aggressive"
		logrus.Info("📊 Increased risk tolerance due to high win rate")
	} else if dm.performance.WinRate < 0.5 {
		// Low win rate: decrease risk
		dm.config.BaseRiskPercent = math.Max(0.01, dm.config.BaseRiskPercent*0.8)
		dm.config.RiskToleranceMode = "conservative"
		logrus.Info("📊 Decreased risk tolerance due to low win rate")
	}

	// Adjust Kelly multiplier based on profit factor
	if dm.performance.ProfitFactor > 2.0 {
		// High profit factor: increase Kelly
		dm.config.KellyMultiplier = math.Min(1.0, dm.config.KellyMultiplier*1.1)
		logrus.Info("📊 Increased Kelly multiplier due to high profit factor")
	} else if dm.performance.ProfitFactor < 1.0 {
		// Low profit factor: decrease Kelly
		dm.config.KellyMultiplier = math.Max(0.25, dm.config.KellyMultiplier*0.9)
		logrus.Info("📊 Decreased Kelly multiplier due to low profit factor")
	}

	// Adjust base leverage based on performance
	if dm.performance.TotalPnL > 0 && dm.performance.WinRate > 0.6 {
		// Profitable with good win rate: increase leverage
		dm.config.BaseLeverage = int(math.Min(float64(dm.config.MaxLeverage), float64(dm.config.BaseLeverage)*1.1))
		logrus.Info("📊 Increased base leverage due to good performance")
	} else if dm.performance.TotalPnL < 0 {
		// Losses: decrease leverage
		dm.config.BaseLeverage = int(math.Max(float64(dm.config.MinLeverage), float64(dm.config.BaseLeverage)*0.9))
		logrus.Info("📊 Decreased base leverage due to losses")
	}

	dm.performance.LastOptimization = time.Now()

	logrus.WithFields(logrus.Fields{
		"original_risk_percent":   originalConfig.BaseRiskPercent,
		"new_risk_percent":        dm.config.BaseRiskPercent,
		"original_kelly_multiplier": originalConfig.KellyMultiplier,
		"new_kelly_multiplier":    dm.config.KellyMultiplier,
		"original_base_leverage":  originalConfig.BaseLeverage,
		"new_base_leverage":       dm.config.BaseLeverage,
	}).Info("📊 Self-optimization complete")

	return nil
}

// updateAccountBalance updates the account balance
func (dm *DynamicManager) updateAccountBalance(ctx context.Context) error {
	acc, err := dm.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return err
	}

	if acc.TotalWalletBalance != "" {
		dm.accountBalance = parseFloat(acc.TotalWalletBalance)
	}

	return nil
}

// getEffectiveRiskPercent returns effective risk percent based on config
func (dm *DynamicManager) getEffectiveRiskPercent() float64 {
	switch dm.config.RiskToleranceMode {
	case "conservative":
		return dm.config.BaseRiskPercent * 0.5
	case "aggressive":
		return dm.config.BaseRiskPercent * 1.5
	case "high":
		return dm.config.MaxRiskPercent
	default:
		return dm.config.BaseRiskPercent
	}
}

// GetPerformanceMetrics returns current performance metrics
func (dm *DynamicManager) GetPerformanceMetrics() PerformanceMetrics {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.performance
}

// GetConfig returns current configuration
func (dm *DynamicManager) GetConfig() DynamicConfig {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.config
}
