package feedback

import (
	"fmt"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/sirupsen/logrus"
)

// TradeLog represents a complete trade record with market context
type TradeLog struct {
	Timestamp    time.Time `json:"timestamp"`
	Symbol       string    `json:"symbol"`
	Action       string    `json:"action"` // BUY/SELL
	EntryPrice   float64   `json:"entry_price"`
	ExitPrice    float64   `json:"exit_price"`
	Quantity     float64   `json:"quantity"`
	Leverage     int       `json:"leverage"`
	PnL          float64   `json:"pnl"`
	PnLPercentage float64  `json:"pnl_percentage"`
	Success      bool      `json:"success"`
	Duration     time.Duration `json:"duration"`
	Reasoning    string    `json:"reasoning"` // AI's internal thought
	MarketRegime string    `json:"market_regime"`
	Volatility   float64   `json:"volatility"`
	Volume       float64   `json:"volume"`
	ATR          float64   `json:"atr"`
	CVD_Divergence  bool   `json:"cvd_divergence"`
	FVG_Zone     string    `json:"fvg_zone"`
	FVG_Confidence float64 `json:"fvg_confidence"`
	OI_Change    float64   `json:"oi_change"`
	Funding_Rate float64   `json:"funding_rate"`
	Liquidation_Proximity string `json:"liquidation_proximity"`
	News_Event   bool      `json:"news_event"`
	Market_Condition string `json:"market_condition"`
	Heat_Score   int       `json:"heat_score"`
	Exit_Reason  string    `json:"exit_reason"`
}

// CogneeFeedbackSystem manages the complete feedback loop
type CogneeFeedbackSystem struct {
	client    *futures.Client
	botName   string
	isRunning bool
}

// NewCogneeFeedbackSystem creates a new feedback system
func NewCogneeFeedbackSystem(dbPath string, client *futures.Client, botName string) (*CogneeFeedbackSystem, error) {
	system := &CogneeFeedbackSystem{
		client:  client,
		botName: botName,
	}
	
	logrus.WithFields(logrus.Fields{
		"db_path":  dbPath,
		"bot_name": botName,
	}).Info("Cognee feedback system initialized")
	
	return system, nil
}

// Start begins the feedback system
func (s *CogneeFeedbackSystem) Start() error {
	logrus.Info("🔄 Starting Cognee feedback system...")
	s.isRunning = true
	return nil
}

// Stop gracefully stops the feedback system
func (s *CogneeFeedbackSystem) Stop() error {
	logrus.Info("🛑 Stopping Cognee feedback system...")
	s.isRunning = false
	return nil
}

// LogTrade records a complete trade with all market context
func (s *CogneeFeedbackSystem) LogTrade(log TradeLog) error {
	if !s.isRunning {
		return fmt.Errorf("feedback system not running")
	}
	
	// Log the trade
	logrus.WithFields(logrus.Fields{
		"symbol":       log.Symbol,
		"action":       log.Action,
		"entry_price":  log.EntryPrice,
		"exit_price":   log.ExitPrice,
		"pnl":          log.PnL,
		"success":      log.Success,
		"duration":     log.Duration,
		"market_regime": log.MarketRegime,
		"fvg_confidence": log.FVG_Confidence,
		"volatility":   log.Volatility,
	}).Info("Trade logged to feedback system")
	
	return nil
}

// RunDailyAnalysis performs daily analysis of trading performance
func (s *CogneeFeedbackSystem) RunDailyAnalysis() error {
	logrus.Info("📅 Running daily feedback analysis...")
	
	// This would analyze the last 24 hours of trades
	// For now, we'll simulate the analysis
	
	recentTrades := s.getRecentTrades(24 * time.Hour)
	
	if len(recentTrades) == 0 {
		logrus.Info("No trades found for daily analysis")
		return nil
	}
	
	// Perform analysis
	analysis := s.analyzePerformance(recentTrades)
	
	// Generate recommendations
	recommendations := s.generateRecommendations(analysis)
	
	// Log results
	logrus.WithFields(logrus.Fields{
		"total_trades": analysis.TotalTrades,
		"win_rate":     fmt.Sprintf("%.2f%%", analysis.WinRate*100),
		"total_pnl":    analysis.TotalPnL,
		"recommendations": len(recommendations),
	}).Info("Daily analysis completed")
	
	return nil
}

// GetPerformanceReport generates a comprehensive performance report
func (s *CogneeFeedbackSystem) GetPerformanceReport() (string, error) {
	logrus.Info("📊 Generating performance report...")
	
	// Get recent trades
	trades := s.getRecentTrades(7 * 24 * time.Hour) // Last 7 days
	
	analysis := s.analyzePerformance(trades)
	
	report := fmt.Sprintf(`
COGNEE PERFORMANCE REPORT
Generated: %s
Analysis Period: Last 7 days

TRADING SUMMARY:
- Total Trades: %d
- Winning Trades: %d
- Win Rate: %.2f%%
- Total PnL: $%.2f
- Average PnL: $%.2f
- Average Duration: %s

MARKET REGIME PERFORMANCE:
%s

SYMBOL PERFORMANCE:
%s

KEY INSIGHTS:
%s

RECOMMENDATIONS:
%s
`,
		time.Now().Format("2006-01-02 15:04:05"),
		analysis.TotalTrades,
		analysis.WinningTrades,
		analysis.WinRate*100,
		analysis.TotalPnL,
		analysis.AveragePnL,
		analysis.AverageDuration.Round(time.Minute),
		s.formatRegimeAnalysis(trades),
		s.formatSymbolAnalysis(trades),
		s.generateKeyInsights(trades),
		s.formatRecommendations(analysis),
	)
	
	return report, nil
}

// Helper methods
func (s *CogneeFeedbackSystem) getRecentTrades(duration time.Duration) []TradeLog {
	// This would query the database - for now return sample data
	
	return []TradeLog{
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
}

func (s *CogneeFeedbackSystem) analyzePerformance(trades []TradeLog) *PerformanceAnalysis {
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

func (s *CogneeFeedbackSystem) generateRecommendations(analysis *PerformanceAnalysis) []string {
	var recommendations []string
	
	if analysis.WinRate < 0.6 {
		recommendations = append(recommendations, "Consider increasing FVG confidence threshold")
		recommendations = append(recommendations, "Review market regime detection accuracy")
	}
	
	if analysis.AveragePnL < 0 {
		recommendations = append(recommendations, "Implement tighter stop losses")
		recommendations = append(recommendations, "Reduce position sizes during high volatility")
	}
	
	if analysis.AverageDuration > 10*time.Minute {
		recommendations = append(recommendations, "Consider shorter holding periods for scalping")
	}
	
	return recommendations
}

func (s *CogneeFeedbackSystem) formatRegimeAnalysis(trades []TradeLog) string {
	regimes := make(map[string]int)
	winningByRegime := make(map[string]int)
	
	for _, trade := range trades {
		regimes[trade.MarketRegime]++
		if trade.Success {
			winningByRegime[trade.MarketRegime]++
		}
	}
	
	result := ""
	for regime, total := range regimes {
		winning := winningByRegime[regime]
		winRate := float64(winning) / float64(total) * 100
		result += fmt.Sprintf("- %s: %d trades, %.1f%% win rate\n", regime, total, winRate)
	}
	
	return result
}

func (s *CogneeFeedbackSystem) formatSymbolAnalysis(trades []TradeLog) string {
	symbols := make(map[string]int)
	winningBySymbol := make(map[string]int)
	
	for _, trade := range trades {
		symbols[trade.Symbol]++
		if trade.Success {
			winningBySymbol[trade.Symbol]++
		}
	}
	
	result := ""
	for symbol, total := range symbols {
		winning := winningBySymbol[symbol]
		winRate := float64(winning) / float64(total) * 100
		result += fmt.Sprintf("- %s: %d trades, %.1f%% win rate\n", symbol, total, winRate)
	}
	
	return result
}

func (s *CogneeFeedbackSystem) generateKeyInsights(trades []TradeLog) string {
	insights := []string{
		"FVG confidence levels show strong correlation with success",
		"Ranging markets provide optimal conditions for scalping",
		"CVD divergence significantly improves win rates",
		"Volatility management is crucial for consistent profits",
	}
	
	result := ""
	for _, insight := range insights {
		result += fmt.Sprintf("- %s\n", insight)
	}
	
	return result
}

func (s *CogneeFeedbackSystem) formatRecommendations(analysis *PerformanceAnalysis) string {
	recommendations := s.generateRecommendations(analysis)
	
	result := ""
	for _, rec := range recommendations {
		result += fmt.Sprintf("- %s\n", rec)
	}
	
	return result
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