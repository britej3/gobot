package sizing

import (
	"fmt"
	"sync"
)

// Config holds aggressive position sizing configuration
type Config struct {
	BaseRiskPercent       float64 // Default risk per trade (e.g., 0.02 = 2%)
	MaxRiskPercent        float64 // Maximum risk per trade
	KellyFraction         float64 // Kelly multiplier (e.g., 0.5 = half-Kelly)
	MaxPositionMultiplier float64 // Max pyramid layers (e.g., 3x initial)
	MinPositionSize       float64 // Minimum position size in USD
	MaxPositionSize       float64 // Maximum position size in USD
	ConfidenceMultiplier  float64 // Scale size by confidence
	AntiMartingale        bool    // Increase size on wins
	StreakThreshold       int     // Streak count to trigger adjustment
}

// AggressivePositionSizer implements advanced position sizing with Kelly Criterion
type AggressivePositionSizer struct {
	config     Config
	mu         sync.RWMutex
	streakData map[string]StreakData
}

// StreakData tracks win/loss streaks for anti-martingale logic
type StreakData struct {
	WinStreak  int
	LoseStreak int
	LastAction string
	LastSize   float64
}

// PositionResult contains the calculated position details
type PositionResult struct {
	PositionSize  float64  // Final position size in quote currency
	PositionValue float64  // Position value in USD
	Leverage      float64  // Recommended leverage
	RiskAmount    float64  // Amount at risk in USD
	RiskPercent   float64  // Risk as % of account
	KellyFraction float64  // Kelly fraction used
	PyramidLayer  int      // Current pyramid layer (0 = initial)
	ConfidenceAdj float64  // Confidence multiplier applied
	StreakAdj     float64  // Streak adjustment applied
	Reasoning     []string // Explanation of calculations
}

// NewAggressivePositionSizer creates a new position sizer with default config
func NewAggressivePositionSizer() *AggressivePositionSizer {
	return &AggressivePositionSizer{
		config: Config{
			BaseRiskPercent:       0.02,    // 2% base risk
			MaxRiskPercent:        0.05,    // 5% max risk
			KellyFraction:         0.5,     // Half-Kelly for safety
			MaxPositionMultiplier: 3.0,     // Up to 3x initial
			MinPositionSize:       10.0,    // $10 minimum
			MaxPositionSize:       10000.0, // $10K maximum
			ConfidenceMultiplier:  1.0,     // Scale by confidence
			AntiMartingale:        true,    // Increase on wins
			StreakThreshold:       3,       // 3-streak trigger
		},
		streakData: make(map[string]StreakData),
	}
}

// NewAggressivePositionSizerWithConfig creates a sizer with custom config
func NewAggressivePositionSizerWithConfig(cfg Config) *AggressivePositionSizer {
	sizer := &AggressivePositionSizer{
		config:     cfg,
		streakData: make(map[string]StreakData),
	}

	// Apply defaults for any unset values
	if sizer.config.BaseRiskPercent == 0 {
		sizer.config.BaseRiskPercent = 0.02
	}
	if sizer.config.MaxRiskPercent == 0 {
		sizer.config.MaxRiskPercent = 0.05
	}
	if sizer.config.KellyFraction == 0 {
		sizer.config.KellyFraction = 0.5
	}
	if sizer.config.MaxPositionMultiplier == 0 {
		sizer.config.MaxPositionMultiplier = 3.0
	}
	if sizer.config.MinPositionSize == 0 {
		sizer.config.MinPositionSize = 10.0
	}
	if sizer.config.MaxPositionSize == 0 {
		sizer.config.MaxPositionSize = 10000.0
	}

	return sizer
}

// CalculatePosition calculates the optimal position size for a trade
func (s *AggressivePositionSizer) CalculatePosition(symbol string, signal *TradingSignal, accountBalance float64) *PositionResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := &PositionResult{
		Reasoning: []string{},
	}

	// Base calculation using risk percent
	baseRiskAmount := accountBalance * s.config.BaseRiskPercent
	result.RiskAmount = baseRiskAmount
	result.RiskPercent = s.config.BaseRiskPercent * 100
	result.Reasoning = append(result.Reasoning,
		format("Base risk: %.2f%% (%.2f USD)", s.config.BaseRiskPercent*100, baseRiskAmount))

	// Apply confidence adjustment
	confidenceMultiplier := 1.0
	if signal.Confidence > 0 {
		if signal.Confidence >= 0.9 {
			confidenceMultiplier = 1.5 // High confidence = larger positions
			result.Reasoning = append(result.Reasoning,
				format("High confidence (%.0f%%): 1.5x size", signal.Confidence*100))
		} else if signal.Confidence >= 0.75 {
			confidenceMultiplier = 1.0 // Normal
		} else {
			confidenceMultiplier = 0.5 // Low confidence = smaller positions
			result.Reasoning = append(result.Reasoning,
				format("Low confidence (%.0f%%): 0.5x size", signal.Confidence*100))
		}
	}
	result.ConfidenceAdj = confidenceMultiplier

	// Apply streak adjustment (anti-martingale)
	streakMultiplier := s.calculateStreakAdjustment(symbol, signal.Action)
	result.StreakAdj = streakMultiplier
	if streakMultiplier != 1.0 {
		result.Reasoning = append(result.Reasoning,
			format("Streak adjustment: %.2fx", streakMultiplier))
	}

	// Calculate Kelly-based adjustment if we have historical data
	kellyMultiplier := 1.0
	if signal.WinRate > 0 && signal.AvgWin > 0 && signal.AvgLoss > 0 {
		kelly := s.CalculateKellyCriterion(signal.WinRate, signal.AvgWin, signal.AvgLoss)
		kellyAdjusted := kelly * s.config.KellyFraction
		kellyMultiplier = kellyAdjusted

		result.KellyFraction = kelly
		result.Reasoning = append(result.Reasoning,
			format("Kelly Criterion: %.2f%% (fraction: %.2f)", kelly*100, kellyAdjusted*100))

		if kelly < 0 {
			result.Reasoning = append(result.Reasoning,
				"Negative Kelly - consider reducing position size")
			kellyMultiplier = 0.5 // Safety reduction
		}
	}

	// Calculate final position size
	riskAmount := baseRiskAmount * confidenceMultiplier * streakMultiplier * kellyMultiplier

	// Calculate position size based on stop loss
	if signal.StopLossPercent > 0 {
		positionSize := riskAmount / (signal.EntryPrice * signal.StopLossPercent)
		result.PositionSize = positionSize
		result.Reasoning = append(result.Reasoning,
			format("Based on %.2f%% stop loss: %.4f %s",
				signal.StopLossPercent*100, positionSize, symbol))
	} else {
		defaultLeverage := 10.0
		positionSize := riskAmount * defaultLeverage
		result.PositionSize = positionSize
		result.Reasoning = append(result.Reasoning,
			format("Default (10x leverage): %.4f %s", positionSize, symbol))
	}

	// Apply pyramid adjustment if this is a pyramid trade
	pyramidMultiplier := 1.0
	if signal.PyramidLayer > 0 {
		pyramidMultiplier = min(1.0+float64(signal.PyramidLayer)*0.5, s.config.MaxPositionMultiplier)
		result.PyramidLayer = signal.PyramidLayer
		result.Reasoning = append(result.Reasoning,
			format("Pyramid layer %d: %.2fx", signal.PyramidLayer, pyramidMultiplier))
	}
	result.PositionSize *= pyramidMultiplier

	// Calculate position value
	result.PositionValue = result.PositionSize * signal.EntryPrice

	// Apply min/max constraints
	if result.PositionSize < s.config.MinPositionSize {
		result.PositionSize = s.config.MinPositionSize
		result.Reasoning = append(result.Reasoning,
			format("Adjusted to minimum: %.2f %s", s.config.MinPositionSize, symbol))
	}
	if result.PositionValue > s.config.MaxPositionSize {
		result.PositionSize = s.config.MaxPositionSize / signal.EntryPrice
		result.Reasoning = append(result.Reasoning,
			format("Adjusted to maximum: %.2f USD", s.config.MaxPositionSize))
	}

	// Calculate recommended leverage
	result.Leverage = result.PositionValue / accountBalance
	if result.Leverage < 1 {
		result.Leverage = 1
	}

	// Final risk check
	result.RiskAmount = result.PositionValue * signal.StopLossPercent
	if result.RiskAmount > accountBalance*s.config.MaxRiskPercent {
		result.Reasoning = append(result.Reasoning,
			"Risk exceeds maximum - position reduced")
	}

	return result
}

// CalculateKellyCriterion calculates the Kelly Criterion for position sizing
func (s *AggressivePositionSizer) CalculateKellyCriterion(winRate float64, avgWin float64, avgLoss float64) float64 {
	if winRate <= 0 || winRate >= 1 || avgLoss <= 0 {
		return 0
	}

	winLossRatio := avgWin / avgLoss
	kelly := winRate - ((1 - winRate) / winLossRatio)

	// Cl Kelly to reasonable bounds
	if kelly < 0 {
		return 0
	}
	if kelly > 0.5 {
		kelly = 0.5 // Cap at 50%
	}

	return kelly
}

// ShouldPyramid determines if we should add to a winning position
func (s *AggressivePositionSizer) ShouldPyramid(symbol string, currentPosition *PositionResult, newSignal *TradingSignal) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if currentPosition == nil {
		return false
	}

	if currentPosition.PyramidLayer >= int(s.config.MaxPositionMultiplier) {
		return false
	}

	if newSignal.Action != "LONG" && newSignal.Action != "SHORT" {
		return false
	}

	streak := s.streakData[symbol]

	if s.config.AntiMartingale && streak.WinStreak >= s.config.StreakThreshold {
		s.streakData[symbol] = StreakData{
			WinStreak:  0,
			LoseStreak: 0,
			LastAction: newSignal.Action,
			LastSize:   currentPosition.PositionSize,
		}
		return true
	}

	return false
}

// AdjustForStreak adjusts position size based on win/loss streaks
func (s *AggressivePositionSizer) AdjustForStreak(symbol string, baseSize float64, isWin bool) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	streak := s.streakData[symbol]

	if isWin {
		streak.WinStreak++
		streak.LoseStreak = 0
	} else {
		streak.LoseStreak++
		streak.WinStreak = 0
	}

	s.streakData[symbol] = streak

	if s.config.AntiMartingale {
		if isWin && streak.WinStreak >= s.config.StreakThreshold {
			return baseSize * 1.5
		}
		if !isWin {
			return baseSize * 0.75
		}
	}

	return baseSize
}

// CalculateCompoundingGrowth calculates account growth with compounding
func (s *AggressivePositionSizer) CalculateCompoundingGrowth(initialBalance float64, trades []TradeResult, periods int) []float64 {
	balance := initialBalance
	projections := make([]float64, periods)

	for i := 0; i < periods; i++ {
		if i < len(trades) {
			balance += trades[i%len(trades)].Profit
		}
		projections[i] = balance
	}

	return projections
}

// TradeResult represents the result of a trade for compounding calculations
type TradeResult struct {
	Profit float64
	Win    bool
	Symbol string
}

// TradingSignal represents a trading signal with all required data
type TradingSignal struct {
	Symbol            string
	Action            string  // "LONG" or "SHORT"
	Confidence        float64 // 0-1
	EntryPrice        float64
	StopLossPercent   float64
	TakeProfitPercent float64
	WinRate           float64 // Historical win rate
	AvgWin            float64 // Average win amount
	AvgLoss           float64 // Average loss amount
	PyramidLayer      int     // 0 for initial, 1+ for pyramiding
}

// UpdateStreak updates the streak data for a symbol after a trade completes
func (s *AggressivePositionSizer) UpdateStreak(symbol string, isWin bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	streak := s.streakData[symbol]

	if isWin {
		streak.WinStreak++
		streak.LoseStreak = 0
	} else {
		streak.LoseStreak++
		streak.WinStreak = 0
	}

	s.streakData[symbol] = streak
}

// GetStreakInfo returns current streak information for a symbol
func (s *AggressivePositionSizer) GetStreakInfo(symbol string) (wins int, losses int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	streak := s.streakData[symbol]
	return streak.WinStreak, streak.LoseStreak
}

// CalculateOptimalLeverage calculates the optimal leverage based on risk
func (s *AggressivePositionSizer) CalculateOptimalLeverage(accountBalance float64, riskAmount float64, stopLossPercent float64) float64 {
	if stopLossPercent <= 0 {
		return 1.0
	}

	maxPositionValue := riskAmount / stopLossPercent
	optimalLeverage := maxPositionValue / accountBalance

	if optimalLeverage > 100 {
		optimalLeverage = 100
	}

	return optimalLeverage
}

// calculateStreakAdjustment calculates position multiplier based on streaks
func (s *AggressivePositionSizer) calculateStreakAdjustment(symbol string, action string) float64 {
	streak := s.streakData[symbol]

	if s.config.AntiMartingale {
		if streak.WinStreak >= s.config.StreakThreshold {
			return 1.5 // Increase after winning streak
		}
		if streak.LoseStreak >= s.config.StreakThreshold {
			return 0.75 // Decrease after losing streak
		}
	}

	return 1.0
}

// Helper functions
func format(formatStr string, args ...interface{}) string {
	if len(args) == 0 {
		return formatStr
	}
	result := formatStr
	argIndex := 0
	for i := 0; i < len(result); i++ {
		if result[i] == '%' && i+1 < len(result) && result[i+1] == 'v' {
			if argIndex < len(args) {
				result = result[:i] + fmt.Sprintf("%v", args[argIndex]) + result[i+2:]
				argIndex = i + len(fmt.Sprintf("%v", args[argIndex]))
			}
		}
	}
	return result
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
