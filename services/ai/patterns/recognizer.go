package patterns

import (
	"math"
)

// PatternType represents different chart patterns
type PatternType string

const (
	PatternHeadAndShoulders    PatternType = "HEAD_AND_SHOULDERS"
	PatternInverseHeadAndShoulders PatternType = "INVERSE_HEAD_AND_SHOULDERS"
	PatternDoubleTop           PatternType = "DOUBLE_TOP"
	PatternDoubleBottom        PatternType = "DOUBLE_BOTTOM"
	PatternTripleTop           PatternType = "TRIPLE_TOP"
	PatternTripleBottom        PatternType = "TRIPLE_BOTTOM"
	PatternBullFlag            PatternType = "BULL_FLAG"
	PatternBearFlag            PatternType = "BEAR_FLAG"
	PatternAscendingTriangle   PatternType = "ASCENDING_TRIANGLE"
	PatternDescendingTriangle  PatternType = "DESCENDING_TRIANGLE"
	PatternSymmetricalTriangle PatternType = "SYMMETRICAL_TRIANGLE"
	PatternCupAndHandle        PatternType = "CUP_AND_HANDLE"
	PatternPennant             PatternType = "PENNANT"
)

// Pattern represents a detected chart pattern
type Pattern struct {
	Type       PatternType // Type of pattern detected
	Symbol     string      // Trading symbol
	Timeframe  string      // Chart timeframe (1m, 5m, 15m, etc.)
	Confidence float64     // 0-100 confidence score
	Direction  string      // "bullish" or "bearish"
	StartPrice float64     // Pattern start price
	EndPrice   float64     // Pattern end price
	StartTime  int64       // Pattern start timestamp
	EndTime    int64       // Pattern end timestamp
	Volume     float64     // Average volume during pattern
	Breakout   bool        // Whether breakout occurred
	BreakoutPrice float64  // Breakout price if occurred
	Target     float64     // Price target
	StopLoss   float64     // Recommended stop loss
	Reasoning  string      // Explanation of detection
}

// Candle represents a price candle
type Candle struct {
	Open      float64 // Opening price
	High      float64 // High price
	Low       float64 // Low price
	Close     float64 // Closing price
	Volume    float64 // Trading volume
	Timestamp int64   // Candle timestamp
}

// PatternRecognizer detects chart patterns in price data
type PatternRecognizer struct {
	minPatternLength int // Minimum candles to form a pattern
	maxPatternLength int // Maximum candles for pattern detection
}

// NewPatternRecognizer creates a new pattern recognizer
func NewPatternRecognizer() *PatternRecognizer {
	return &PatternRecognizer{
		minPatternLength: 10,  // Minimum 10 candles
		maxPatternLength: 200, // Maximum 200 candles for pattern
	}
}

// DetectPatterns detects all patterns in the given candles
func (r *PatternRecognizer) DetectPatterns(candles []Candle, symbol string, timeframe string) []Pattern {
	if len(candles) < r.minPatternLength {
		return []Pattern{}
	}

	patterns := []Pattern{}

	// Detect various patterns
	if p := r.DetectHeadAndShoulders(candles, symbol, timeframe); p != nil {
		patterns = append(patterns, *p)
	}
	if p := r.DetectDoubleTopBottom(candles, symbol, timeframe); p != nil {
		patterns = append(patterns, *p)
	}
	if p := r.DetectFlags(candles, symbol, timeframe); p != nil {
		patterns = append(patterns, *p)
	}
	if p := r.DetectTriangles(candles, symbol, timeframe); p != nil {
		patterns = append(patterns, *p)
	}

	// Sort by confidence
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Confidence > patterns[i].Confidence {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}

	return patterns
}

// DetectHeadAndShoulders detects Head and Shoulders pattern
func (r *PatternRecognizer) DetectHeadAndShoulders(candles []Candle, symbol string, timeframe string) *Pattern {
	n := len(candles)
	if n < 30 {
		return nil
	}

	// Find local maxima and minima
	var peaks []int
	var troughs []int

	for i := 2; i < n-2; i++ {
		// Check for peak (high)
		if candles[i].High > candles[i-1].High &&
			candles[i].High > candles[i-2].High &&
			candles[i].High > candles[i+1].High &&
			candles[i].High > candles[i+2].High {
			peaks = append(peaks, i)
		}
		// Check for trough (low)
		if candles[i].Low < candles[i-1].Low &&
			candles[i].Low < candles[i-2].Low &&
			candles[i].Low < candles[i+1].Low &&
			candles[i].Low < candles[i+2].Low {
			troughs = append(troughs, i)
		}
	}

	// Look for Head and Shoulders pattern (left shoulder, head, right shoulder)
	// Pattern: peak - trough - higher peak - trough - lower peak
	if len(peaks) >= 3 && len(troughs) >= 2 {
		for i := 0; i < len(peaks)-2; i++ {
			leftShoulder := peaks[i]
			head := peaks[i+1]
			rightShoulder := peaks[i+2]

			// Head should be higher than shoulders
			if candles[head].High > candles[leftShoulder].High &&
				candles[head].High > candles[rightShoulder].High {

				// Shoulders should be roughly equal (within 5%)
				leftHeight := candles[leftShoulder].High - candles[troughs[i]].Low
				rightHeight := candles[rightShoulder].High - candles[troughs[i+1]].Low

				if leftHeight > 0 && rightHeight > 0 {
					ratio := leftHeight / rightHeight
					if ratio > 0.8 && ratio < 1.2 { // Within 20%

						// Check neckline (connection between troughs)
						necklineSlope := (candles[troughs[i+1]].Low - candles[troughs[i]].Low) /
							float64(troughs[i+1]-troughs[i])

						// Calculate confidence
						confidence := r.calculatePatternConfidence("head_and_shoulders", candles, head)

						direction := "bearish"
						target := candles[head].High - (candles[head].High - candles[troughs[i]].Low)*1.5
						stopLoss := candles[head].High + candles[head].High*0.01

						return &Pattern{
							Type:          PatternHeadAndShoulders,
							Symbol:        symbol,
							Timeframe:     timeframe,
							Confidence:    confidence,
							Direction:     direction,
							StartPrice:    candles[leftShoulder].High,
							EndPrice:      candles[rightShoulder].High,
							StartTime:     candles[leftShoulder].Timestamp,
							EndTime:       candles[rightShoulder].Timestamp,
							Volume:        r.calculateAverageVolume(candles),
							Breakout:      false,
							Target:        target,
							StopLoss:      stopLoss,
							Reasoning:     "Head and Shoulders pattern detected with matching shoulder heights",
						}
					}
				}
			}
		}
	}

	return nil
}

// DetectDoubleTopBottom detects Double Top and Double Bottom patterns
func (r *PatternRecognizer) DetectDoubleTopBottom(candles []Candle, symbol string, timeframe string) *Pattern {
	n := len(candles)
	if n < 20 {
		return nil
	}

	var peaks []int
	var troughs []int

	for i := 2; i < n-2; i++ {
		if candles[i].High > candles[i-1].High &&
			candles[i].High > candles[i-2].High &&
			candles[i].High > candles[i+1].High &&
			candles[i].High > candles[i+2].High {
			peaks = append(peaks, i)
		}
		if candles[i].Low < candles[i-1].Low &&
			candles[i].Low < candles[i-2].Low &&
			candles[i].Low < candles[i+1].Low &&
			candles[i].Low < candles[i+2].Low {
			troughs = append(troughs, i)
		}
	}

	// Double Top
	for i := 0; i < len(peaks)-1; i++ {
		p1, p2 := peaks[i], peaks[i+1]
		// Check if peaks are roughly equal
		avgPrice := (candles[p1].High + candles[p2].High) / 2
		deviation := math.Abs(candles[p1].High-candles[p2].High) / avgPrice

		if deviation < 0.02 { // Within 2%
			// Check for decline between peaks
			decline := candles[p1].High - candles[p2-1].Low
			if decline > avgPrice*0.02 {
				confidence := r.calculatePatternConfidence("double_top", candles, p2)

				return &Pattern{
					Type:          PatternDoubleTop,
					Symbol:        symbol,
					Timeframe:     timeframe,
					Confidence:    confidence,
					Direction:     "bearish",
					StartPrice:    candles[p1].High,
					EndPrice:      candles[p2].High,
					StartTime:     candles[p1].Timestamp,
					EndTime:       candles[p2].Timestamp,
					Volume:        r.calculateAverageVolume(candles),
					Breakout:      false,
					Target:        candles[p2].High - (candles[p1].High-candles[p2-1].Low)*1.5,
					StopLoss:      candles[p2].High + candles[p2].High*0.01,
					Reasoning:     "Double Top pattern detected with matching peaks",
				}
			}
		}
	}

	// Double Bottom
	for i := 0; i < len(troughs)-1; i++ {
		t1, t2 := troughs[i], troughs[i+1]
		avgPrice := (candles[t1].Low + candles[t2].Low) / 2
		deviation := math.Abs(candles[t1].Low-candles[t2].Low) / avgPrice

		if deviation < 0.02 {
			rise := candles[t1+1].High - candles[t2].Low
			if rise > avgPrice*0.02 {
				confidence := r.calculatePatternConfidence("double_bottom", candles, t2)

				return &Pattern{
					Type:          PatternDoubleBottom,
					Symbol:        symbol,
					Timeframe:     timeframe,
					Confidence:    confidence,
					Direction:     "bullish",
					StartPrice:    candles[t1].Low,
					EndPrice:      candles[t2].Low,
					StartTime:     candles[t1].Timestamp,
					EndTime:       candles[t2].Timestamp,
					Volume:        r.calculateAverageVolume(candles),
					Breakout:      false,
					Target:        candles[t2].Low + (candles[t1+1].High-candles[t2].Low)*1.5,
					StopLoss:      candles[t2].Low - candles[t2].Low*0.01,
					Reasoning:     "Double Bottom pattern detected with matching troughs",
				}
			}
		}
	}

	return nil
}

// DetectFlags detects Bull and Bear Flag patterns
func (r *PatternRecognizer) DetectFlags(candles []Candle, symbol string, timeframe string) *Pattern {
	n := len(candles)
	if n < 15 {
		return nil
	}

	// Look for strong impulse move followed by consolidation
	// Bull Flag: Strong up move + downward consolidation
	// Bear Flag: Strong down move + upward consolidation

	for i := 10; i < n-5; i++ {
		// Check for impulse move (flag pole)
		impulseCandles := candles[i-10 : i]
		var impulseUp, impulseDown float64
		var maxHigh, minLow float64
		var maxHighIdx, minLowIdx int

		for j, c := range impulseCandles {
			if c.High > maxHigh {
				maxHigh = c.High
				maxHighIdx = j
			}
			if c.Low < minLow || j == 0 {
				minLow = c.Low
				minLowIdx = j
			}
		}

		impulseRange := maxHigh - minLow
		impulsePercent := impulseRange / minLow * 100

		if impulsePercent < 2 { // Minimum 2% impulse
			continue
		}

		// Determine direction
		isBullFlag := maxHighIdx > minLowIdx // Upward impulse
		isBearFlag := maxHighIdx < minLowIdx // Downward impulse

		if isBullFlag {
			// Check for downward consolidation (flag)
			flagCandles := candles[i : i+5]
			flagHigh := flagCandles[0].High
			flagLow := flagCandles[0].Low
			for _, c := range flagCandles {
				if c.High > flagHigh {
					flagHigh = c.High
				}
				if c.Low < flagLow {
					flagLow = c.Low
				}
			}

			// Flag should be smaller than impulse
			flagRange := flagHigh - flagLow
			if flagRange < impulseRange*0.5 {
				confidence := r.calculatePatternConfidence("bull_flag", candles, i)

				return &Pattern{
					Type:          PatternBullFlag,
					Symbol:        symbol,
					Timeframe:     timeframe,
					Confidence:    confidence,
					Direction:     "bullish",
					StartPrice:    candles[i-10].Close,
					EndPrice:      candles[i+4].Close,
					StartTime:     candles[i-10].Timestamp,
					EndTime:       candles[i+4].Timestamp,
					Volume:        r.calculateAverageVolume(candles),
					Breakout:      false,
					Target:        candles[i-10].Close + impulseRange*1.5,
					StopLoss:      flagLow,
					Reasoning:     "Bull Flag pattern detected with strong upward impulse",
				}
			}
		}

		if isBearFlag {
			flagCandles := candles[i : i+5]
			flagHigh := flagCandles[0].High
			flagLow := flagCandles[0].Low
			for _, c := range flagCandles {
				if c.High > flagHigh {
					flagHigh = c.High
				}
				if c.Low < flagLow {
					flagLow = c.Low
				}
			}

			flagRange := flagHigh - flagLow
			if flagRange < impulseRange*0.5 {
				confidence := r.calculatePatternConfidence("bear_flag", candles, i)

				return &Pattern{
					Type:          PatternBearFlag,
					Symbol:        symbol,
					Timeframe:     timeframe,
					Confidence:    confidence,
					Direction:     "bearish",
					StartPrice:    candles[i-10].Close,
					EndPrice:      candles[i+4].Close,
					StartTime:     candles[i-10].Timestamp,
					EndTime:       candles[i+4].Timestamp,
					Volume:        r.calculateAverageVolume(candles),
					Breakout:      false,
					Target:        candles[i-10].Close - impulseRange*1.5,
					StopLoss:      flagHigh,
					Reasoning:     "Bear Flag pattern detected with strong downward impulse",
				}
			}
		}
	}

	return nil
}

// DetectTriangles detects various triangle patterns
func (r *PatternRecognizer) DetectTriangles(candles []Candle, symbol string, timeframe string) *Pattern {
	n := len(candles)
	if n < 20 {
		return nil
	}

	// Calculate trendlines
	// Ascending: Higher lows, flat highs
	// Descending: Flat lows, lower highs
	// Symmetrical: Higher lows, lower highs

	var highs []float64
	var lows []float64

	for i := n - 20; i < n; i++ {
		highs = append(highs, candles[i].High)
		lows = append(lows, candles[i].Low)
	}

	// Calculate slope of highs and lows
	highSlope := r.calculateSlope(highs)
	lowSlope := r.calculateSlope(lows)

	// Ascending Triangle
	if lowSlope > 0.001 && math.Abs(highSlope) < 0.001 {
		confidence := r.calculatePatternConfidence("ascending_triangle", candles, n-1)

		return &Pattern{
			Type:          PatternAscendingTriangle,
			Symbol:        symbol,
			Timeframe:     timeframe,
			Confidence:    confidence,
			Direction:     "bullish",
			StartPrice:    candles[n-20].High,
			EndPrice:      candles[n-1].Low,
			StartTime:     candles[n-20].Timestamp,
			EndTime:       candles[n-1].Timestamp,
			Volume:        r.calculateAverageVolume(candles),
			Breakout:      false,
			Target:        candles[n-1].High + (candles[n-20].High-candles[n-20].Low)*0.5,
			StopLoss:      candles[n-1].Low - candles[n-1].Low*0.005,
			Reasoning:     "Ascending Triangle pattern with higher lows and flat highs",
		}
	}

	// Descending Triangle
	if highSlope < -0.001 && math.Abs(lowSlope) < 0.001 {
		confidence := r.calculatePatternConfidence("descending_triangle", candles, n-1)

		return &Pattern{
			Type:          PatternDescendingTriangle,
			Symbol:        symbol,
			Timeframe:     timeframe,
			Confidence:    confidence,
			Direction:     "bearish",
			StartPrice:    candles[n-20].High,
			EndPrice:      candles[n-1].Low,
			StartTime:     candles[n-20].Timestamp,
			EndTime:       candles[n-1].Timestamp,
			Volume:        r.calculateAverageVolume(candles),
			Breakout:      false,
			Target:        candles[n-1].Low - (candles[n-20].High-candles[n-20].Low)*0.5,
			StopLoss:      candles[n-1].High + candles[n-1].High*0.005,
			Reasoning:     "Descending Triangle pattern with lower highs and flat lows",
		}
	}

	// Symmetrical Triangle
	if lowSlope > 0.001 && highSlope < -0.001 {
		confidence := r.calculatePatternConfidence("symmetrical_triangle", candles, n-1)

		// Predict breakout direction based on recent trend
		recentTrend := candles[n-1].Close - candles[n-10].Close
		direction := "bullish"
		if recentTrend < 0 {
			direction = "bearish"
		}

		return &Pattern{
			Type:          PatternSymmetricalTriangle,
			Symbol:        symbol,
			Timeframe:     timeframe,
			Confidence:    confidence,
			Direction:     direction,
			StartPrice:    candles[n-20].High,
			EndPrice:      candles[n-1].Low,
			StartTime:     candles[n-20].Timestamp,
			EndTime:       candles[n-1].Timestamp,
			Volume:        r.calculateAverageVolume(candles),
			Breakout:      false,
			Reasoning:     "Symmetrical Triangle pattern with converging trendlines",
		}
	}

	return nil
}

// calculateSlope calculates the slope of a price series
func (r *PatternRecognizer) calculateSlope(prices []float64) float64 {
	n := len(prices)
	if n < 2 {
		return 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i, price := range prices {
		x := float64(i)
		sumX += x
		sumY += price
		sumXY += x * price
		sumX2 += x * x
	}

	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumX2 - sumX*sumX)
	return slope
}

// calculateAverageVolume calculates the average volume
func (r *PatternRecognizer) calculateAverageVolume(candles []Candle) float64 {
	if len(candles) == 0 {
		return 0
	}

	var sum float64
	for _, c := range candles {
		sum += c.Volume
	}
	return sum / float64(len(candles))
}

// calculatePatternConfidence calculates confidence score for a detected pattern
func (r *PatternRecognizer) calculatePatternConfidence(patternType string, candles []Candle, index int) float64 {
	baseConfidence := 50.0 // Base confidence

	// Check volume confirmation
	avgVolume := r.calculateAverageVolume(candles)
	recentVolume := candles[index].Volume
	if recentVolume > avgVolume*1.5 {
		baseConfidence += 15 // Higher volume = more confidence
	}

	// Check pattern completion
	baseConfidence += 10

	// Reduce confidence if pattern is just forming
	if index < len(candles)-5 {
		baseConfidence -= 10
	}

	// Cap at 95%
	if baseConfidence > 95 {
		baseConfidence = 95
	}

	return baseConfidence
}

// CalculatePatternTarget calculates price target based on pattern
func (r *PatternRecognizer) CalculatePatternTarget(pattern *Pattern) float64 {
	switch pattern.Type {
	case PatternHeadAndShoulders, PatternDoubleTop:
		return pattern.StartPrice - (pattern.StartPrice-pattern.EndPrice)*1.5
	case PatternInverseHeadAndShoulders, PatternDoubleBottom:
		return pattern.EndPrice + (pattern.EndPrice-pattern.StartPrice)*1.5
	default:
		return pattern.EndPrice + (pattern.EndPrice-pattern.StartPrice)
	}
}

// GetPatternStatistics returns statistics about detected patterns
func (r *PatternRecognizer) GetPatternStatistics(patterns []Pattern) map[string]interface{} {
	stats := make(map[string]interface{})

	if len(patterns) == 0 {
		stats["total_patterns"] = 0
		stats["avg_confidence"] = 0.0
		stats["bullish_count"] = 0
		stats["bearish_count"] = 0
		return stats
	}

	var totalConfidence float64
	bullish := 0
	bearish := 0

	for _, p := range patterns {
		totalConfidence += p.Confidence
		if p.Direction == "bullish" {
			bullish++
		} else {
			bearish++
		}
	}

	stats["total_patterns"] = len(patterns)
	stats["avg_confidence"] = totalConfidence / float64(len(patterns))
	stats["bullish_count"] = bullish
	stats["bearish_count"] = bearish
	stats["market_sentiment"] = "bullish"
	if bullish < bearish {
		stats["market_sentiment"] = "bearish"
	}

	return stats
}
