package adaptive

import (
	"time"
)

// TradingSession represents different market session characteristics
type TradingSession struct {
	Name              string
	StartHour         int // UTC hour
	EndHour           int // UTC hour
	VolumeThreshold   float64
	DeltaThreshold    float64
	MomentumMin       float64
	MomentumMax       float64
	ExpectedSignals   int // Expected signals per hour
	PositionSizeMulti float64
	Description       string
}

// AdaptiveConfig manages time-based threshold adjustments
type AdaptiveConfig struct {
	CurrentSession      TradingSession
	NoSignalMinutes     int
	LastSignalTime      time.Time
	AutoRelaxEnabled    bool
	RelaxationLevel     int // 0=strict, 1=normal, 2=relaxed, 3=max
	SessionHistory      []SessionPerformance
}

type SessionPerformance struct {
	Session    string
	Timestamp  time.Time
	SignalCount int
	WinRate    float64
	AvgProfit  float64
}

// Define trading sessions with optimal parameters (only normal and high volatility)
var TradingSessions = []TradingSession{
	{
		Name:              "US_SESSION_PRIME",
		StartHour:         13, // 13:30 UTC
		EndHour:           16,
		VolumeThreshold:   8.0,  // Strict - high quality
		DeltaThreshold:    0.80,
		MomentumMin:       3.0,
		MomentumMax:       8.0,
		ExpectedSignals:   15, // High activity expected
		PositionSizeMulti: 1.0, // Full aggressive
		Description:       "US market open - highest volatility",
	},
	{
		Name:              "EU_SESSION",
		StartHour:         8,
		EndHour:           10,
		VolumeThreshold:   7.0,  // Slightly relaxed
		DeltaThreshold:    0.78,
		MomentumMin:       2.5,
		MomentumMax:       7.0,
		ExpectedSignals:   10,
		PositionSizeMulti: 0.95,
		Description:       "EU morning - good volatility",
	},
	{
		Name:              "ASIA_VOLATILITY",
		StartHour:         0,
		EndHour:           2,
		VolumeThreshold:   7.5,
		DeltaThreshold:    0.78,
		MomentumMin:       2.5,
		MomentumMax:       7.0,
		ExpectedSignals:   8,
		PositionSizeMulti: 0.9,
		Description:       "Asia session volatility window",
	},
	{
		Name:              "US_CONTINUATION",
		StartHour:         16,
		EndHour:           20,
		VolumeThreshold:   6.5,  // Relaxed
		DeltaThreshold:    0.75,
		MomentumMin:       2.0,
		MomentumMax:       6.0,
		ExpectedSignals:   6,
		PositionSizeMulti: 0.85,
		Description:       "US afternoon - moderate activity",
	},
	{
		Name:              "NORMAL_SESSION",
		StartHour:         2,
		EndHour:           8,
		VolumeThreshold:   6.0,  // Normal volatility
		DeltaThreshold:    0.73,
		MomentumMin:       2.0,
		MomentumMax:       6.0,
		ExpectedSignals:   5,
		PositionSizeMulti: 0.8,
		Description:       "Normal trading session - consistent opportunities",
	},
	{
		Name:              "NORMAL_SESSION_LUNCH",
		StartHour:         10,
		EndHour:           13,
		VolumeThreshold:   6.0,  // Normal volatility
		DeltaThreshold:    0.73,
		MomentumMin:       2.0,
		MomentumMax:       6.0,
		ExpectedSignals:   5,
		PositionSizeMulti: 0.8,
		Description:       "Normal trading session - consistent opportunities",
	},
	{
		Name:              "NORMAL_SESSION_EVENING",
		StartHour:         20,
		EndHour:           24,
		VolumeThreshold:   6.0,  // Normal volatility
		DeltaThreshold:    0.73,
		MomentumMin:       2.0,
		MomentumMax:       6.0,
		ExpectedSignals:   5,
		PositionSizeMulti: 0.8,
		Description:       "Normal trading session - consistent opportunities",
	},
}

// GetCurrentSession returns the appropriate session for current UTC time
func GetCurrentSession() TradingSession {
	now := time.Now().UTC()
	hour := now.Hour()
	// weekday := now.Weekday()

	// Weekend check DISABLED - crypto markets are 24/7 with scalping opportunities
	// if weekday == time.Saturday || weekday == time.Sunday {
	//     return TradingSessions[len(TradingSessions)-1] // OFF_HOURS
	// }

	// Check each session
	for _, session := range TradingSessions {
		if hour >= session.StartHour && hour < session.EndHour {
			return session
		}
		// Handle wrap-around (e.g., 23:00-01:00)
		if session.StartHour > session.EndHour {
			if hour >= session.StartHour || hour < session.EndHour {
				return session
			}
		}
	}

	// Default to off-hours if no match
	return TradingSessions[len(TradingSessions)-1]
}

// AdaptThresholds adjusts parameters based on signal drought
func (ac *AdaptiveConfig) AdaptThresholds() TradingSession {
	if !ac.AutoRelaxEnabled {
		return ac.CurrentSession
	}

	minutesSinceSignal := int(time.Since(ac.LastSignalTime).Minutes())
	ac.NoSignalMinutes = minutesSinceSignal
	adapted := ac.CurrentSession

	// Progressive relaxation based on time without signals
	if minutesSinceSignal >= 30 && minutesSinceSignal < 45 {
		// Level 1: Minor relaxation
		ac.RelaxationLevel = 1
		adapted.VolumeThreshold *= 0.85
		adapted.DeltaThreshold -= 0.03
		adapted.MomentumMin -= 0.5
		adapted.Description += " [AUTO-RELAXED-L1]"
	} else if minutesSinceSignal >= 45 && minutesSinceSignal < 60 {
		// Level 2: Moderate relaxation
		ac.RelaxationLevel = 2
		adapted.VolumeThreshold *= 0.75
		adapted.DeltaThreshold -= 0.05
		adapted.MomentumMin -= 1.0
		adapted.Description += " [AUTO-RELAXED-L2]"
	} else if minutesSinceSignal >= 60 {
		// Level 3: Maximum relaxation (but still safe)
		ac.RelaxationLevel = 3
		adapted.VolumeThreshold *= 0.65
		adapted.DeltaThreshold -= 0.08
		adapted.MomentumMin -= 1.5
		adapted.Description += " [AUTO-RELAXED-L3-MAX]"
	} else {
		ac.RelaxationLevel = 0
	}

	return adapted
}

// ResetRelaxation called when a signal is found
func (ac *AdaptiveConfig) ResetRelaxation() {
	ac.LastSignalTime = time.Now()
	ac.NoSignalMinutes = 0
	ac.RelaxationLevel = 0
}

// ShouldTrade determines if current time is suitable for trading
func ShouldTrade() (bool, string) {
	// Weekend check DISABLED - crypto markets are 24/7 with scalping opportunities
	// now := time.Now().UTC()
	// weekday := now.Weekday()
	// if weekday == time.Saturday || weekday == time.Sunday {
	//     return false, "Weekend - extremely low crypto volume"
	// }

	// Get current session
	session := GetCurrentSession()

	// Always allow trading with normal and high volatility sessions
	if session.ExpectedSignals >= 8 {
		return true, "High volatility session - excellent trading opportunities"
	}

	return true, "Normal volatility session - consistent trading opportunities"
}

// GetOptimalPositionSize returns position size based on session and capital
func GetOptimalPositionSize(capital float64, session TradingSession) float64 {
	baseSize := capital * 0.90 // 90% for aggressive
	// Adjust based on session multiplier
	return baseSize * session.PositionSizeMulti
}

// GetSessionStrategy returns strategy recommendations for current session
func GetSessionStrategy() string {
	session := GetCurrentSession()
	now := time.Now().UTC()
	strategy := ""

	switch session.Name {
	case "US_SESSION_PRIME":
		strategy = "🔥 US SESSION PRIME - AGGRESSIVE MODE ENABLED\n   - Expect 15+ signals/hour\n   - Use strict filters (Volume 8x+, Delta 0.80+)\n   - Full position sizes (90% capital)\n   - Fast rotation every 60-90 seconds\n   - Best time for Below $50 setup\n   - Target: +60% session possible"
	case "EU_SESSION":
		strategy = "✅ EU SESSION - ACTIVE MODE\n   - Expect 10+ signals/hour\n   - Slightly relaxed filters (Volume 7x+, Delta 0.78+)\n   - 95% normal position sizes\n   - Good for multi-position rotation\n   - Target: +40-50% session"
	case "ASIA_VOLATILITY":
		strategy = "⚡ ASIA VOLATILITY - ACTIVE MODE\n   - Expect 8+ signals/hour\n   - Good volatility (Volume 7.5x+, Delta 0.78+)\n   - 90% normal position sizes\n   - Excellent for scalp trades\n   - Target: +30-40% session"
	case "US_CONTINUATION":
		strategy = "📈 US CONTINUATION - NORMAL MODE\n   - Expect 6+ signals/hour\n   - Relaxed filters (Volume 6.5x+, Delta 0.75+)\n   - 85% normal position sizes\n   - Good for steady gains\n   - Target: +25-35% session"
	case "NORMAL_SESSION":
		strategy = "✨ NORMAL SESSION - BALANCED MODE\n   - Expect 5+ signals/hour\n   - Normal filters (Volume 6.0x+, Delta 0.73+)\n   - 80% normal position sizes\n   - Consistent trading opportunities\n   - Target: +20-30% session"
	case "NORMAL_SESSION_LUNCH":
		strategy = "✨ NORMAL SESSION - BALANCED MODE\n   - Expect 5+ signals/hour\n   - Normal filters (Volume 6.0x+, Delta 0.73+)\n   - 80% normal position sizes\n   - Consistent trading opportunities\n   - Target: +20-30% session"
	case "NORMAL_SESSION_EVENING":
		strategy = "✨ NORMAL SESSION - BALANCED MODE\n   - Expect 5+ signals/hour\n   - Normal filters (Volume 6.0x+, Delta 0.73+)\n   - 80% normal position sizes\n   - Consistent trading opportunities\n   - Target: +20-30% session"
	default:
		strategy = "📊 NORMAL SESSION - Balanced approach recommended"
	}

	// Add time until next prime session
	nextPrime := 13 // US session start
	hoursUntil := nextPrime - now.Hour()
	if hoursUntil < 0 {
		hoursUntil += 24
	}

	strategy += "\n\n⏰ Next US Prime Session: " + time.Duration(hoursUntil).String() + " hours"
	return strategy
}

// NewAdaptiveConfig creates a new adaptive configuration
func NewAdaptiveConfig() *AdaptiveConfig {
	return &AdaptiveConfig{
		CurrentSession:   GetCurrentSession(),
		AutoRelaxEnabled: true,
		LastSignalTime:   time.Now(),
		SessionHistory:    make([]SessionPerformance, 0),
	}
}
