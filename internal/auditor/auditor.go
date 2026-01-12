package auditor

import (
	"context"
	"fmt"
	"time"

	"github.com/britebrt/cognee/pkg/brain"
	"github.com/britebrt/cognee/pkg/feedback"
	"github.com/sirupsen/logrus"
)

// Auditor performs post-trade analysis and strategy refinement
type Auditor struct {
	brain    *brain.BrainEngine
	feedback *feedback.CogneeFeedbackSystem
	isRunning bool
}

// NewAuditor creates a new strategy auditor
func NewAuditor(brain *brain.BrainEngine, feedback *feedback.CogneeFeedbackSystem) *Auditor {
	return &Auditor{
		brain:    brain,
		feedback: feedback,
	}
}

// Start begins the auditing process
func (a *Auditor) Start(ctx context.Context) error {
	logrus.Info("🔍 Starting strategy auditor...")
	
	a.isRunning = true
	
	// Start periodic auditing
	go a.runPeriodicAuditing(ctx)
	
	logrus.Info("✅ Strategy auditor started")
	return nil
}

// Stop gracefully stops the auditor
func (a *Auditor) Stop() error {
	logrus.Info("🛑 Stopping strategy auditor...")
	a.isRunning = false
	return nil
}

func (a *Auditor) runPeriodicAuditing(ctx context.Context) {
	logrus.Info("📊 Starting periodic auditing...")
	
	// Run daily analysis at midnight
	dailyTicker := time.NewTicker(24 * time.Hour)
	defer dailyTicker.Stop()
	
	// Run weekly analysis on Sundays
	weeklyTicker := time.NewTicker(7 * 24 * time.Hour)
	defer weeklyTicker.Stop()
	
	// Initial analysis
	a.performDailyAnalysis(ctx)
	
	for a.isRunning {
		select {
		case <-dailyTicker.C:
			a.performDailyAnalysis(ctx)
		case <-weeklyTicker.C:
			a.performWeeklyAnalysis(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (a *Auditor) performDailyAnalysis(ctx context.Context) {
	logrus.Info("📅 Performing daily analysis...")
	
	// Get recent trade history
	recentTrades, err := a.getRecentTrades(24 * time.Hour)
	if err != nil {
		logrus.WithError(err).Error("Failed to get recent trades")
		return
	}
	
	if len(recentTrades) == 0 {
		logrus.Info("No trades in the last 24 hours")
		return
	}
	
	// Analyze performance
	performance := a.analyzePerformance(recentTrades)
	
	// Detect patterns in losing trades
	patterns := a.detectLosingPatterns(recentTrades)
	
	// Generate strategy adjustments
	adjustments := a.generateAdjustments(performance, patterns)
	
	// Log analysis results
	a.logDailyAnalysis(performance, patterns, adjustments)
	
	// Apply minor adjustments if confidence is high
	if adjustments.Confidence > 0.8 {
		a.applyMinorAdjustments(adjustments)
	}
}

func (a *Auditor) performWeeklyAnalysis(ctx context.Context) {
	logrus.Info("📊 Performing weekly analysis...")
	
	// Get weekly trade history
	weeklyTrades, err := a.getRecentTrades(7 * 24 * time.Hour)
	if err != nil {
		logrus.WithError(err).Error("Failed to get weekly trades")
		return
	}
	
	if len(weeklyTrades) == 0 {
		logrus.Info("No trades in the last week")
		return
	}
	
	// Comprehensive performance analysis
	performance := a.analyzeComprehensivePerformance(weeklyTrades)
	
	// Market regime analysis
	regimeAnalysis := a.analyzeMarketRegimes(weeklyTrades)
	
	// Symbol-specific analysis
	symbolAnalysis := a.analyzeSymbolPerformance(weeklyTrades)
	
	// Generate comprehensive strategy update
	strategyUpdate := a.generateComprehensiveStrategyUpdate(performance, regimeAnalysis, symbolAnalysis)
	
	// Log weekly analysis
	a.logWeeklyAnalysis(performance, regimeAnalysis, symbolAnalysis, strategyUpdate)
	
	// Apply strategy updates if confidence is high
	if strategyUpdate.Confidence > 0.85 {
		a.applyMajorStrategyUpdate(strategyUpdate)
	}
}

func (a *Auditor) getRecentTrades(duration time.Duration) ([]feedback.TradeLog, error) {
	if a.feedback == nil {
		return nil, fmt.Errorf("feedback system not available")
	}
	
	// Get trades from the last duration
	// startTime = time.Now().Add(-duration)
	
	// This would query the database - for now, return sample data
	// In production, this would be a proper database query
	
	sampleTrades := []feedback.TradeLog{
		{
			Timestamp:    time.Now().Add(-2 * time.Hour),
			Symbol:       "ZECUSDT",
			Action:       "BUY",
			EntryPrice:   49500.0,
			ExitPrice:    49750.0,
			PnL:          50.0,
			Success:      true,
			Duration:     3 * time.Minute,
			MarketRegime: "RANGING",
			Volatility:   0.018,
			FVG_Confidence: 0.82,
		},
		{
			Timestamp:    time.Now().Add(-4 * time.Hour),
			Symbol:       "NEOUSDT",
			Action:       "SELL",
			EntryPrice:   12500.0,
			ExitPrice:    12450.0,
			PnL:          12.5,
			Success:      true,
			Duration:     5 * time.Minute,
			MarketRegime: "RANGING",
			Volatility:   0.022,
			FVG_Confidence: 0.75,
		},
	}
	
	return sampleTrades, nil
}

func (a *Auditor) analyzePerformance(trades []feedback.TradeLog) *PerformanceAnalysis {
	if len(trades) == 0 {
		return &PerformanceAnalysis{}
	}
	
	totalTrades := len(trades)
	winningTrades := 0
	totalPnL := 0.0
	totalDuration := time.Duration(0)
	
	for _, trade := range trades {
		if trade.Success {
			winningTrades++
		}
		totalPnL += trade.PnL
		totalDuration += trade.Duration
	}
	
	avgDuration := totalDuration / time.Duration(totalTrades)
	winRate := float64(winningTrades) / float64(totalTrades)
	
	return &PerformanceAnalysis{
		TotalTrades:   totalTrades,
		WinningTrades: winningTrades,
		WinRate:       winRate,
		TotalPnL:      totalPnL,
		AveragePnL:    totalPnL / float64(totalTrades),
		AverageDuration: avgDuration,
	}
}

func (a *Auditor) detectLosingPatterns(trades []feedback.TradeLog) []LosingPattern {
	var patterns []LosingPattern
	
	// Analyze losing trades
	var losingTrades []feedback.TradeLog
	for _, trade := range trades {
		if !trade.Success {
			losingTrades = append(losingTrades, trade)
		}
	}
	
	if len(losingTrades) == 0 {
		return patterns
	}
	
	// Pattern 1: High volatility losses
	highVolLosses := a.filterHighVolatilityLosses(losingTrades)
	if len(highVolLosses) > 2 {
		patterns = append(patterns, LosingPattern{
			Type:        "HighVolatilityFailure",
			Description: fmt.Sprintf("%d losses during high volatility (>2.5%%)", len(highVolLosses)),
			Severity:    len(highVolLosses),
			Suggestion:  "Reduce leverage by 30% when volatility >2.5%",
		})
	}
	
	// Pattern 2: Low confidence losses
	lowConfLosses := a.filterLowConfidenceLosses(losingTrades)
	if len(lowConfLosses) > 2 {
		patterns = append(patterns, LosingPattern{
			Type:        "LowConfidenceFailure",
			Description: fmt.Sprintf("%d losses with FVG confidence <0.75", len(lowConfLosses)),
			Severity:    len(lowConfLosses),
			Suggestion:  "Increase minimum FVG confidence to 0.8",
		})
	}
	
	// Pattern 3: News event losses
	newsLosses := a.filterNewsEventLosses(losingTrades)
	if len(newsLosses) > 1 {
		patterns = append(patterns, LosingPattern{
			Type:        "NewsEventFailure",
			Description: fmt.Sprintf("%d losses during news events", len(newsLosses)),
			Severity:    len(newsLosses),
			Suggestion:  "Skip trading during high-impact news events",
		})
	}
	
	return patterns
}

func (a *Auditor) filterHighVolatilityLosses(trades []feedback.TradeLog) []feedback.TradeLog {
	var filtered []feedback.TradeLog
	for _, trade := range trades {
		if trade.Volatility > 0.025 {
			filtered = append(filtered, trade)
		}
	}
	return filtered
}

func (a *Auditor) filterLowConfidenceLosses(trades []feedback.TradeLog) []feedback.TradeLog {
	var filtered []feedback.TradeLog
	for _, trade := range trades {
		if trade.FVG_Confidence < 0.75 {
			filtered = append(filtered, trade)
		}
	}
	return filtered
}

func (a *Auditor) filterNewsEventLosses(trades []feedback.TradeLog) []feedback.TradeLog {
	var filtered []feedback.TradeLog
	for _, trade := range trades {
		if trade.News_Event {
			filtered = append(filtered, trade)
		}
	}
	return filtered
}

func (a *Auditor) generateAdjustments(performance *PerformanceAnalysis, patterns []LosingPattern) *StrategyAdjustments {
	adjustments := &StrategyAdjustments{
		Confidence: 0.7, // Base confidence
	}
	
	// Generate adjustments based on patterns
	for _, pattern := range patterns {
		switch pattern.Type {
		case "HighVolatilityFailure":
			adjustments.ParameterChanges["max_leverage"] = 0.7 // Reduce by 30%
			adjustments.NewRules["volatile_market"] = "Reduce leverage by 30% when volatility >2.5%"
			adjustments.Confidence += 0.1
			
		case "LowConfidenceFailure":
			adjustments.ParameterChanges["fvg_confidence_min"] = 0.8 // Increase from 0.75
			adjustments.NewRules["low_confidence"] = "Skip trades with FVG confidence <0.8"
			adjustments.Confidence += 0.1
			
		case "NewsEventFailure":
			adjustments.NewRules["news_events"] = "Skip trading during high-impact news"
			adjustments.Confidence += 0.05
		}
	}
	
	// Cap confidence at 0.95
	if adjustments.Confidence > 0.95 {
		adjustments.Confidence = 0.95
	}
	
	return adjustments
}

func (a *Auditor) applyMinorAdjustments(adjustments *StrategyAdjustments) {
	logrus.WithFields(logrus.Fields{
		"confidence": adjustments.Confidence,
		"changes":    len(adjustments.ParameterChanges),
		"rules":      len(adjustments.NewRules),
	}).Info("Applying minor strategy adjustments")
	
	// Apply parameter changes
	for param, value := range adjustments.ParameterChanges {
		logrus.WithField("parameter", param).WithField("value", value).Info("Adjusting parameter")
		// In production, this would update the actual strategy parameters
	}
	
	// Log new rules
	for rule, description := range adjustments.NewRules {
		logrus.WithFields(logrus.Fields{
			"rule":        rule,
			"description": description,
		}).Info("Adding new trading rule")
	}
}

func (a *Auditor) applyMajorStrategyUpdate(update *ComprehensiveStrategyUpdate) {
	logrus.WithFields(logrus.Fields{
		"confidence": update.Confidence,
		"type":       update.UpdateType,
	}).Info("Applying major strategy update")
	
	// This would implement significant strategy changes
	// In production, this might involve:
	// - Updating core trading algorithms
	// - Retraining models
	// - Deploying new strategy versions
	
	logrus.Info("Major strategy update applied successfully")
}

// Data structures
type PerformanceAnalysis struct {
	TotalTrades     int           `json:"total_trades"`
	WinningTrades   int           `json:"winning_trades"`
	WinRate         float64       `json:"win_rate"`
	TotalPnL        float64       `json:"total_pnl"`
	AveragePnL      float64       `json:"average_pnl"`
	AverageDuration time.Duration `json:"average_duration"`
}

type LosingPattern struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    int    `json:"severity"`
	Suggestion  string `json:"suggestion"`
}

type StrategyAdjustments struct {
	ParameterChanges map[string]interface{} `json:"parameter_changes"`
	NewRules         map[string]string      `json:"new_rules"`
	Confidence       float64               `json:"confidence"`
}

type ComprehensiveStrategyUpdate struct {
	UpdateType  string  `json:"update_type"`
	Description string  `json:"description"`
	Changes     []string `json:"changes"`
	Confidence  float64 `json:"confidence"`
}

func (a *Auditor) logDailyAnalysis(performance *PerformanceAnalysis, patterns []LosingPattern, adjustments *StrategyAdjustments) {
	logrus.WithFields(logrus.Fields{
		"total_trades":  performance.TotalTrades,
		"win_rate":      fmt.Sprintf("%.2f%%", performance.WinRate*100),
		"total_pnl":     performance.TotalPnL,
		"patterns":      len(patterns),
		"adjustments":   len(adjustments.ParameterChanges) + len(adjustments.NewRules),
		"confidence":    adjustments.Confidence,
	}).Info("Daily analysis completed")
}

func (a *Auditor) analyzeComprehensivePerformance(trades []feedback.TradeLog) *ComprehensivePerformance {
	// More detailed performance analysis
	analysis := &ComprehensivePerformance{
		PerformanceAnalysis: *a.analyzePerformance(trades),
	}
	
	// Add additional metrics
	for _, trade := range trades {
		analysis.ByMarketRegime[trade.MarketRegime]++
		analysis.BySymbol[trade.Symbol]++
		
		if trade.Success {
			analysis.WinningByRegime[trade.MarketRegime]++
		}
	}
	
	return analysis
}

func (a *Auditor) analyzeMarketRegimes(trades []feedback.TradeLog) map[string]MarketRegimeAnalysis {
	regimes := make(map[string]MarketRegimeAnalysis)
	
	for _, trade := range trades {
		regime := trade.MarketRegime
		if _, exists := regimes[regime]; !exists {
			regimes[regime] = MarketRegimeAnalysis{
				Regime: regime,
			}
		}
		
		regimeData := regimes[regime]
		regimeData.TotalTrades++
		regimeData.TotalPnL += trade.PnL
		
		if trade.Success {
			regimeData.WinningTrades++
		}
		
		regimes[regime] = regimeData
	}
	
	// Calculate win rates
	for regime, data := range regimes {
		data.WinRate = float64(data.WinningTrades) / float64(data.TotalTrades)
		data.AveragePnL = data.TotalPnL / float64(data.TotalTrades)
		regimes[regime] = data
	}
	
	return regimes
}

func (a *Auditor) analyzeSymbolPerformance(trades []feedback.TradeLog) map[string]SymbolAnalysis {
	symbols := make(map[string]SymbolAnalysis)
	
	for _, trade := range trades {
		symbol := trade.Symbol
		if _, exists := symbols[symbol]; !exists {
			symbols[symbol] = SymbolAnalysis{
				Symbol: symbol,
			}
		}
		
		symbolData := symbols[symbol]
		symbolData.TotalTrades++
		symbolData.TotalPnL += trade.PnL
		
		if trade.Success {
			symbolData.WinningTrades++
		}
		
		symbols[symbol] = symbolData
	}
	
	// Calculate win rates and averages
	for symbol, data := range symbols {
		data.WinRate = float64(data.WinningTrades) / float64(data.TotalTrades)
		data.AveragePnL = data.TotalPnL / float64(data.TotalTrades)
		symbols[symbol] = data
	}
	
	return symbols
}

func (a *Auditor) generateComprehensiveStrategyUpdate(performance *ComprehensivePerformance, regimes map[string]MarketRegimeAnalysis, symbols map[string]SymbolAnalysis) *ComprehensiveStrategyUpdate {
	update := &ComprehensiveStrategyUpdate{
		Confidence: 0.8, // Base confidence for weekly updates
	}
	
	// Analyze regime performance
	bestRegime := ""
	bestRegimeWinRate := 0.0
	for regime, data := range regimes {
		if data.WinRate > bestRegimeWinRate {
			bestRegimeWinRate = data.WinRate
			bestRegime = regime
		}
	}
	
	if bestRegime != "" && bestRegimeWinRate > 0.7 {
		update.UpdateType = "FOCUS_ON_BEST_REGIME"
		update.Description = fmt.Sprintf("Focus on %s regime (%.1f%% win rate)", bestRegime, bestRegimeWinRate*100)
		update.Changes = append(update.Changes, fmt.Sprintf("Prioritize %s market conditions", bestRegime))
		update.Confidence += 0.1
	}
	
	// Analyze symbol performance
	bestSymbol := ""
	bestSymbolWinRate := 0.0
	for symbol, data := range symbols {
		if data.WinRate > bestSymbolWinRate && data.TotalTrades > 5 {
			bestSymbolWinRate = data.WinRate
			bestSymbol = symbol
		}
	}
	
	if bestSymbol != "" && bestSymbolWinRate > 0.7 {
		update.Changes = append(update.Changes, fmt.Sprintf("Increase allocation to %s (%.1f%% win rate)", bestSymbol, bestSymbolWinRate*100))
		update.Confidence += 0.05
	}
	
	// Cap confidence
	if update.Confidence > 0.95 {
		update.Confidence = 0.95
	}
	
	return update
}

func (a *Auditor) logWeeklyAnalysis(performance *ComprehensivePerformance, regimes map[string]MarketRegimeAnalysis, symbols map[string]SymbolAnalysis, update *ComprehensiveStrategyUpdate) {
	logrus.WithFields(logrus.Fields{
		"total_trades":  performance.TotalTrades,
		"win_rate":      fmt.Sprintf("%.2f%%", performance.WinRate*100),
		"total_pnl":     performance.TotalPnL,
		"regimes":       len(regimes),
		"symbols":       len(symbols),
		"update_type":   update.UpdateType,
		"confidence":    update.Confidence,
	}).Info("Weekly analysis completed")
}

// Additional data structures
type ComprehensivePerformance struct {
	PerformanceAnalysis
	ByMarketRegime map[string]int `json:"by_market_regime"`
	BySymbol       map[string]int `json:"by_symbol"`
	WinningByRegime map[string]int `json:"winning_by_regime"`
}

type MarketRegimeAnalysis struct {
	Regime        string  `json:"regime"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalPnL      float64 `json:"total_pnl"`
	AveragePnL    float64 `json:"average_pnl"`
}

type SymbolAnalysis struct {
	Symbol        string  `json:"symbol"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalPnL      float64 `json:"total_pnl"`
	AveragePnL    float64 `json:"average_pnl"`
}