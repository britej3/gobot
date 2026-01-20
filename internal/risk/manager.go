package risk

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britej3/gobot/pkg/feedback"
)

// RiskConfig holds risk management configuration
type RiskConfig struct {
	MaxRiskPerTrade       float64 `json:"max_risk_per_trade"`       // Maximum risk per trade (e.g., 0.02 = 2%)
	MaxTotalRisk          float64 `json:"max_total_risk"`           // Maximum total portfolio risk (e.g., 0.10 = 10%)
	KellyMultiplier       float64 `json:"kelly_multiplier"`         // Kelly criterion multiplier (e.g., 0.5 = half Kelly)
	VolatilityMultiplier  float64 `json:"volatility_multiplier"`    // Volatility adjustment factor
	MaxLeverage           int     `json:"max_leverage"`             // Maximum leverage allowed
	MinLeverage           int     `json:"min_leverage"`             // Minimum leverage
	StopLossMultiplier    float64 `json:"stop_loss_multiplier"`     // Stop loss distance multiplier
	TakeProfitMultiplier  float64 `json:"take_profit_multiplier"`   // Take profit distance multiplier
	MaxCorrelationRisk    float64 `json:"max_correlation_risk"`     // Maximum correlation risk (0.0-1.0)
	MinAccountBalance     float64 `json:"min_account_balance"`      // Minimum account balance before stopping
}

// DefaultRiskConfig returns default risk management configuration
func DefaultRiskConfig() RiskConfig {
	return RiskConfig{
		MaxRiskPerTrade:      0.02,  // 2% risk per trade
		MaxTotalRisk:         0.10,  // 10% total portfolio risk
		KellyMultiplier:      0.5,   // Half Kelly
		VolatilityMultiplier: 1.0,   // No volatility adjustment
		MaxLeverage:          25,    // Maximum 25x leverage
		MinLeverage:          5,     // Minimum 5x leverage
		StopLossMultiplier:   1.0,   // Standard stop loss
		TakeProfitMultiplier: 2.0,   // 2:1 risk-reward ratio
		MaxCorrelationRisk:   0.3,   // Maximum 30% correlation risk
		MinAccountBalance:    1000,  // Minimum $1000 balance
	}
}

// RiskManager handles advanced risk management
type RiskManager struct {
	config     RiskConfig
	client     *futures.Client
	feedback   *feedback.CogneeFeedbackSystem
	symbols    []string
	mu         sync.RWMutex
	correlationMatrix map[string]map[string]float64
	volatilityCache   map[string]float64
	lastUpdate        time.Time
}

// NewRiskManager creates a new risk manager
func NewRiskManager(client *futures.Client, feedback *feedback.CogneeFeedbackSystem, symbols []string) *RiskManager {
	return &RiskManager{
		config:            DefaultRiskConfig(),
		client:            client,
		feedback:          feedback,
		symbols:           symbols,
		correlationMatrix: make(map[string]map[string]float64),
		volatilityCache:   make(map[string]float64),
		lastUpdate:        time.Now(),
	}
}

// UpdateConfig updates risk management configuration
func (rm *RiskManager) UpdateConfig(config RiskConfig) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.config = config
}

// CalculatePositionSize calculates optimal position size using Kelly Criterion
func (rm *RiskManager) CalculatePositionSize(ctx context.Context, symbol string, entryPrice, stopLoss, takeProfit float64, confidence float64) (float64, int, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Get account balance
	balance, err := rm.getAvailableBalance(ctx)
	if err != nil {
		return 0, 0, err
	}
	
	// Calculate volatility-adjusted risk
	volatility := rm.getVolatility(symbol)
	volatilityAdjustment := 1.0
	if volatility > 0.02 { // High volatility
		volatilityAdjustment = 0.5
	} else if volatility < 0.005 { // Low volatility
		volatilityAdjustment = 1.5
	}
	
	// Calculate Kelly Criterion position size
	winRate := confidence
	avgWin := math.Abs(takeProfit - entryPrice) / math.Abs(entryPrice - stopLoss)
	avgLoss := 1.0
	
	kellyFraction := (winRate*avgWin - (1-winRate)*avgLoss) / avgWin
	if kellyFraction < 0 {
		kellyFraction = 0
	}
	
	// Apply Kelly multiplier and volatility adjustment
	adjustedKelly := kellyFraction * rm.config.KellyMultiplier * volatilityAdjustment
	
	// Calculate risk-based position size
	riskAmount := balance * rm.config.MaxRiskPerTrade * adjustedKelly
	riskPerUnit := math.Abs(entryPrice - stopLoss)
	
	if riskPerUnit <= 0 {
		return 0, 0, nil
	}
	
	positionSize := riskAmount / riskPerUnit
	
	// Apply leverage optimization
	optimalLeverage := rm.calculateOptimalLeverage(volatility, confidence)
	
	// Ensure position size doesn't exceed maximum risk
	maxPositionSize := (balance * rm.config.MaxTotalRisk) / riskPerUnit
	if positionSize > maxPositionSize {
		positionSize = maxPositionSize
	}
	
	// Apply correlation risk adjustment
	correlationRisk := rm.getCorrelationRisk(symbol)
	if correlationRisk > rm.config.MaxCorrelationRisk {
		positionSize *= (1 - correlationRisk)
	}
	
	return positionSize, optimalLeverage, nil
}

// calculateOptimalLeverage calculates optimal leverage based on volatility and confidence
func (rm *RiskManager) calculateOptimalLeverage(volatility, confidence float64) int {
	// Base leverage on confidence
	baseLeverage := int(confidence * float64(rm.config.MaxLeverage))
	
	// Adjust for volatility
	if volatility > 0.03 {
		baseLeverage = int(float64(baseLeverage) * 0.5) // Reduce leverage in high volatility
	} else if volatility < 0.005 {
		baseLeverage = int(float64(baseLeverage) * 1.5) // Increase leverage in low volatility
	}
	
	// Ensure within bounds
	if baseLeverage < rm.config.MinLeverage {
		baseLeverage = rm.config.MinLeverage
	}
	if baseLeverage > rm.config.MaxLeverage {
		baseLeverage = rm.config.MaxLeverage
	}
	
	return baseLeverage
}

// CalculateDynamicStopLoss calculates dynamic stop loss based on volatility
func (rm *RiskManager) CalculateDynamicStopLoss(entryPrice, volatility float64, side string) float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	// Use ATR-based stop loss
	atr := volatility * entryPrice * 100 // Convert to price terms
	
	// Dynamic stop loss distance based on volatility
	var stopDistance float64
	if volatility > 0.02 {
		stopDistance = atr * 2.0 // Wider stops in high volatility
	} else if volatility < 0.005 {
		stopDistance = atr * 0.5 // Tighter stops in low volatility
	} else {
		stopDistance = atr * 1.0
	}
	
	if side == "LONG" {
		return entryPrice - stopDistance
	} else {
		return entryPrice + stopDistance
	}
}

// CalculateDynamicTakeProfit calculates dynamic take profit based on risk-reward ratio
func (rm *RiskManager) CalculateDynamicTakeProfit(entryPrice, stopLoss float64, side string) float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	risk := math.Abs(entryPrice - stopLoss)
	reward := risk * rm.config.TakeProfitMultiplier
	
	if side == "LONG" {
		return entryPrice + reward
	} else {
		return entryPrice - reward
	}
}

// CheckRiskLimits checks if trade violates risk limits
func (rm *RiskManager) CheckRiskLimits(ctx context.Context, symbol string, positionSize float64) error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	// Check minimum account balance
	balance, err := rm.getAvailableBalance(ctx)
	if err != nil {
		return err
	}
	
	if balance < rm.config.MinAccountBalance {
		return fmt.Errorf("account balance %.2f below minimum threshold %.2f", balance, rm.config.MinAccountBalance)
	}
	
	// Check correlation risk
	correlationRisk := rm.getCorrelationRisk(symbol)
	if correlationRisk > rm.config.MaxCorrelationRisk {
		return fmt.Errorf("correlation risk %.2f exceeds maximum %.2f", correlationRisk, rm.config.MaxCorrelationRisk)
	}
	
	// Check position size limits
	maxPositionSize := balance * rm.config.MaxTotalRisk
	if positionSize > maxPositionSize {
		return fmt.Errorf("position size %.4f exceeds maximum %.4f", positionSize, maxPositionSize)
	}
	
	return nil
}

// UpdateCorrelationMatrix updates the correlation matrix
func (rm *RiskManager) UpdateCorrelationMatrix(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Fetch recent price data for all symbols
	priceData := make(map[string][]float64)
	
	for _, symbol := range rm.symbols {
		klines, err := rm.client.NewKlinesService().
			Symbol(symbol).
			Interval("1m").
			Limit(100).
			Do(ctx)
		
		if err != nil {
			continue
		}
		
		var prices []float64
		for _, kline := range klines {
			prices = append(prices, parseFloatSafe(kline.Close))
		}
		priceData[symbol] = prices
	}
	
	// Calculate correlations
	for symbol1 := range priceData {
		if rm.correlationMatrix[symbol1] == nil {
			rm.correlationMatrix[symbol1] = make(map[string]float64)
		}
		
		for symbol2 := range priceData {
			if symbol1 == symbol2 {
				rm.correlationMatrix[symbol1][symbol2] = 1.0
				continue
			}
			
			corr := calculateCorrelation(priceData[symbol1], priceData[symbol2])
			rm.correlationMatrix[symbol1][symbol2] = corr
		}
	}
	
	rm.lastUpdate = time.Now()
	return nil
}

// getCorrelationRisk calculates correlation risk for a symbol
func (rm *RiskManager) getCorrelationRisk(symbol string) float64 {
	if rm.correlationMatrix[symbol] == nil {
		return 0
	}
	
	var maxCorrelation float64
	for _, corr := range rm.correlationMatrix[symbol] {
		if corr > maxCorrelation {
			maxCorrelation = corr
		}
	}
	
	return maxCorrelation
}

// getVolatility gets cached volatility for a symbol
func (rm *RiskManager) getVolatility(symbol string) float64 {
	if vol, exists := rm.volatilityCache[symbol]; exists {
		return vol
	}
	
	// Calculate and cache volatility
	vol := rm.calculateVolatility(symbol)
	rm.volatilityCache[symbol] = vol
	return vol
}

// calculateVolatility calculates volatility for a symbol
func (rm *RiskManager) calculateVolatility(symbol string) float64 {
	klines, err := rm.client.NewKlinesService().
		Symbol(symbol).
		Interval("1m").
		Limit(50).
		Do(context.Background())
	
	if err != nil {
		return 0
	}
	
	var returns []float64
	for i := 1; i < len(klines); i++ {
		prevClose := parseFloatSafe(klines[i-1].Close)
		currClose := parseFloatSafe(klines[i].Close)
		if prevClose > 0 {
			returns = append(returns, (currClose-prevClose)/prevClose)
		}
	}
	
	if len(returns) == 0 {
		return 0
	}
	
	// Calculate standard deviation
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))
	
	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))
	
	return math.Sqrt(variance)
}

// getAvailableBalance gets available balance
func (rm *RiskManager) getAvailableBalance(ctx context.Context) (float64, error) {
	acc, err := rm.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return 0, err
	}
	
	// Parse USDT balance from TotalWalletBalance
	if acc.TotalWalletBalance != "" {
		return parseFloatSafe(acc.TotalWalletBalance), nil
	}
	
	return 0, nil
}

// Helper functions
func calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0
	}
	
	n := float64(len(x))
	
	// Calculate means
	meanX, meanY := 0.0, 0.0
	for i := 0; i < len(x); i++ {
		meanX += x[i]
		meanY += y[i]
	}
	meanX /= n
	meanY /= n
	
	// Calculate correlation
	numerator := 0.0
	denomX, denomY := 0.0, 0.0
	
	for i := 0; i < len(x); i++ {
		numerator += (x[i] - meanX) * (y[i] - meanY)
		denomX += (x[i] - meanX) * (x[i] - meanX)
		denomY += (y[i] - meanY) * (y[i] - meanY)
	}
	
	if denomX == 0 || denomY == 0 {
		return 0
	}
	
	return numerator / (math.Sqrt(denomX) * math.Sqrt(denomY))
}

func parseFloatSafe(s string) float64 {
	if s == "" {
		return 0
	}
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	if err != nil {
		return 0
	}
	return val
}