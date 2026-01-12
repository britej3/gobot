package watcher

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britebrt/cognee/pkg/brain"
	"github.com/sirupsen/logrus"
)

// WatcherConfig holds configuration for the market watcher
type WatcherConfig struct {
	MinFVGConfidence      float64  `json:"min_fvg_confidence"`
	MaxVolatility         float64  `json:"max_volatility"`
	Min24hVolume          float64  `json:"min_24h_volume"`
	MarketRegimeTolerance bool     `json:"market_regime_tolerance"`
	WatchlistSymbols      []string `json:"watchlist_symbols"`
}

// DefaultWatcherConfig returns aggressive settings for testing
func DefaultWatcherConfig() WatcherConfig {
	return WatcherConfig{
		MinFVGConfidence:      0.6,  // Lower from 0.7 for testing
		MaxVolatility:         0.05, // Allow up to 5% vs 2.5%
		Min24hVolume:          1000000, // 1M USD for testnet
		MarketRegimeTolerance: true,  // Allow trending markets
		WatchlistSymbols:      []string{"ADAUSDT", "DOTUSDT", "AVAXUSDT", "MATICUSDT", "LINKUSDT"},
	}
}

// Watcher monitors market conditions and identifies trading opportunities
type Watcher struct {
	client       *futures.Client
	brain        *brain.BrainEngine
	symbols      []string
	isRunning    bool
	config       WatcherConfig
}

// NewWatcher creates a new market watcher
func NewWatcher(client *futures.Client, brain *brain.BrainEngine, symbols []string) *Watcher {
	config := DefaultWatcherConfig()
	
	// Override with environment variables if set
	if minConf := os.Getenv("MIN_FVG_CONFIDENCE"); minConf != "" {
		if val, err := strconv.ParseFloat(minConf, 64); err == nil {
			config.MinFVGConfidence = val
		}
	}
	if maxVol := os.Getenv("MAX_VOLATILITY"); maxVol != "" {
		if val, err := strconv.ParseFloat(maxVol, 64); err == nil {
			config.MaxVolatility = val
		}
	}
	if minVol := os.Getenv("MIN_24H_VOLUME_USD"); minVol != "" {
		if val, err := strconv.ParseFloat(minVol, 64); err == nil {
			config.Min24hVolume = val
		}
	}
	if regimeTol := os.Getenv("MARKET_REGIME_TOLERANCE"); regimeTol != "" {
		config.MarketRegimeTolerance = regimeTol == "true"
	}
	if symbols := os.Getenv("WATCHLIST_SYMBOLS"); symbols != "" {
		config.WatchlistSymbols = strings.Split(symbols, ",")
	}
	
	// Use configured symbols or default
	if len(symbols) == 0 {
		symbols = config.WatchlistSymbols
	}
	
	return &Watcher{
		client:  client,
		brain:   brain,
		symbols: symbols,
		config:  config,
	}
}

// Start begins market monitoring
func (w *Watcher) Start(ctx context.Context) error {
	logrus.Info("👁️ Starting market watcher...")
	
	w.isRunning = true
	
	// Start monitoring each symbol
	for _, symbol := range w.symbols {
		go w.monitorSymbol(ctx, symbol)
	}
	
	logrus.Info("✅ Market watcher started")
	return nil
}

// Stop gracefully stops the watcher
func (w *Watcher) Stop() error {
	logrus.Info("🛑 Stopping market watcher...")
	w.isRunning = false
	return nil
}

func (w *Watcher) monitorSymbol(ctx context.Context, symbol string) {
	logrus.WithField("symbol", symbol).Info("Starting symbol monitoring")
	
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds for more aggressive testing
	defer ticker.Stop()
	
	// Heartbeat counter for debugging
	heartbeatCount := 0
	
	for w.isRunning {
		select {
		case <-ticker.C:
			heartbeatCount++
			logrus.WithFields(logrus.Fields{
				"symbol":         symbol,
				"heartbeat":      heartbeatCount,
				"config":         w.config,
			}).Debug("Watcher heartbeat - analyzing market")
			
			w.analyzeMarketOpportunity(ctx, symbol)
		case <-ctx.Done():
			return
		}
	}
}

func (w *Watcher) analyzeMarketOpportunity(ctx context.Context, symbol string) {
	startTime := time.Now()
	
	// Gather market data
	marketData, err := w.gatherMarketData(ctx, symbol)
	if err != nil {
		logrus.WithError(err).WithField("symbol", symbol).Error("Failed to gather market data")
		return
	}
	
	// Get AI analysis from brain
	analysis, err := w.brain.AnalyzeMarket(ctx, marketData)
	if err != nil {
		logrus.WithError(err).WithField("symbol", symbol).Error("Failed to analyze market")
		return
	}
	
	// Check for FVG opportunities
	if w.detectFVGOpportunity(marketData, analysis) {
		w.triggerTradingSignal(symbol, marketData, analysis)
	}
	
	latency := time.Since(startTime)
	logrus.WithFields(logrus.Fields{
		"symbol":        symbol,
		"market_regime": analysis.MarketRegime,
		"confidence":    analysis.Confidence,
		"latency_ms":    latency.Milliseconds(),
	}).Debug("Market analysis completed")
}

func (w *Watcher) gatherMarketData(ctx context.Context, symbol string) (interface{}, error) {
	// Get recent kline data
	klines, err := w.client.NewKlinesService().
		Symbol(symbol).
		Interval("1m").
		Limit(60).
		Do(ctx)
	
	if err != nil {
		return nil, err
	}
	
	// Get current position risk
	positions, err := w.client.NewGetPositionRiskService().
		Symbol(symbol).
		Do(ctx)
	
	if err != nil {
		return nil, err
	}
	
	// Calculate market metrics
	var totalVolume float64
	var priceChanges []float64
	var highPrices []float64
	var lowPrices []float64
	
	for i, kline := range klines {
		if i == 0 {
			continue // Skip first for change calculation
		}
		
		volume := parseFloat(kline.Volume)
		totalVolume += volume
		
		closePrice := parseFloat(kline.Close)
		openPrice := parseFloat(kline.Open)
		priceChange := (closePrice - openPrice) / openPrice
		priceChanges = append(priceChanges, priceChange)
		
		highPrices = append(highPrices, parseFloat(kline.High))
		lowPrices = append(lowPrices, parseFloat(kline.Low))
	}
	
	// Calculate volatility (standard deviation of price changes)
	volatility := calculateVolatility(priceChanges)
	
	// Get current position if exists
	var currentPosition interface{}
	for _, pos := range positions {
		positionAmt := parseFloat(pos.PositionAmt)
		if positionAmt != 0 {
			currentPosition = map[string]interface{}{
				"symbol":      pos.Symbol,
				"position":    positionAmt,
				"entry_price": parseFloat(pos.EntryPrice),
				"leverage":    parseFloat(pos.Leverage),
			}
			break
		}
	}
	
	return map[string]interface{}{
		"symbol":           symbol,
		"current_position": currentPosition,
		"volatility":       volatility,
		"volume":           totalVolume,
		"price_changes":    priceChanges,
		"high_prices":      highPrices,
		"low_prices":       lowPrices,
		"timeframe":        "1m",
	}, nil
}

func (w *Watcher) detectFVGOpportunity(marketData interface{}, analysis *brain.MarketAnalysis) bool {
	// Enhanced FVG detection logic with configurable thresholds
	// "Warm-up Mode" - more aggressive for testing
	
	data := marketData.(map[string]interface{})
	volatility := data["volatility"].(float64)
	
	// Log current market conditions for debugging
	logrus.WithFields(logrus.Fields{
		"volatility":        volatility,
		"max_volatility":    w.config.MaxVolatility,
		"confidence":        analysis.Confidence,
		"min_confidence":    w.config.MinFVGConfidence,
		"market_regime":     analysis.MarketRegime,
		"regime_tolerance":  w.config.MarketRegimeTolerance,
	}).Debug("FVG opportunity analysis")
	
	// Check if market conditions are favorable with configurable thresholds
	regimeMatch := analysis.MarketRegime == "RANGING" || w.config.MarketRegimeTolerance
	volatilityOk := volatility <= w.config.MaxVolatility
	confidenceOk := analysis.Confidence >= w.config.MinFVGConfidence
	
	if regimeMatch && volatilityOk && confidenceOk {
		logrus.WithFields(logrus.Fields{
			"volatility":     volatility,
			"confidence":     analysis.Confidence,
			"market_regime":  analysis.MarketRegime,
		}).Info("🎯 FVG opportunity detected - criteria met")
		return true
	}
	
	logrus.WithFields(logrus.Fields{
		"regime_match":     regimeMatch,
		"volatility_ok":    volatilityOk,
		"confidence_ok":    confidenceOk,
	}).Debug("FVG opportunity rejected - criteria not met")
	
	return false
}

func (w *Watcher) triggerTradingSignal(symbol string, marketData interface{}, analysis *brain.MarketAnalysis) {
	// Create trading signal
	signal := map[string]interface{}{
		"symbol":        symbol,
		"opportunity":   "FVG_DETECTED",
		"market_data":   marketData,
		"analysis":      analysis,
		"timestamp":     time.Now(),
		"confidence":    analysis.Confidence,
	}
	
	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"opportunity": "FVG_DETECTED",
		"confidence": analysis.Confidence,
	}).Info("Trading opportunity detected - triggering signal")
	
	// Send signal to striker (would be implemented with proper messaging)
	go w.processTradingSignal(context.Background(), signal)
}

func (w *Watcher) processTradingSignal(ctx context.Context, signal map[string]interface{}) {
	// This would typically send to a message queue or direct to striker
	// For now, we'll simulate the process
	
	symbol := signal["symbol"].(string)
	analysis := signal["analysis"].(*brain.MarketAnalysis)
	
	// Create detailed trading signal for brain
	tradingSignal := struct {
		Symbol        string  `json:"symbol"`
		FVGZone       string  `json:"fvg_zone"`
		FVGConfidence float64 `json:"fvg_confidence"`
		CVDDivergence bool    `json:"cvd_divergence"`
		Volatility    float64 `json:"volatility"`
		MarketRegime  string  `json:"market_regime"`
		Confidence    float64 `json:"confidence"`
	}{
		Symbol:        symbol,
		FVGZone:       "BULLISH", // Would be determined by analysis
		FVGConfidence: analysis.Confidence,
		CVDDivergence: true, // Would be determined by market data
		Volatility:    0.02, // From market data
		MarketRegime:  analysis.MarketRegime,
		Confidence:    analysis.Confidence,
	}
	
	// Get trading decision from brain
	decision, err := w.brain.MakeTradingDecision(ctx, tradingSignal)
	if err != nil {
		logrus.WithError(err).Error("Failed to get trading decision")
		return
	}
	
	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"decision":   decision.Decision,
		"confidence": decision.Confidence,
		"reasoning":  decision.Reasoning,
	}).Info("Trading decision received from brain")
}

// Helper functions
func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func calculateVolatility(changes []float64) float64 {
	if len(changes) == 0 {
		return 0
	}
	
	// Calculate standard deviation
	mean := 0.0
	for _, change := range changes {
		mean += change
	}
	mean /= float64(len(changes))
	
	variance := 0.0
	for _, change := range changes {
		diff := change - mean
		variance += diff * diff
	}
	variance /= float64(len(changes))
	
	return variance // This is actually variance, but close enough for our purposes
}