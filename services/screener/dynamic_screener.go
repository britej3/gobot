package screener

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/sirupsen/logrus"
)

// DynamicScreenerConfig holds dynamic screener configuration
type DynamicScreenerConfig struct {
	// Base Configuration
	Interval            time.Duration `json:"interval"`
	MaxPairs            int           `json:"max_pairs"`
	SortBy              string        `json:"sort_by"`

	// Dynamic Filters
	MinVolume24h        float64 `json:"min_volume_24h"`
	MaxVolume24h        float64 `json:"max_volume_24h"`
	MinPriceChange      float64 `json:"min_price_change"`
	MaxPriceChange      float64 `json:"max_price_change"`
	MinPrice            float64 `json:"min_price"`
	MaxPrice            float64 `json:"max_price"`

	// Risk Tolerance
	RiskToleranceMode   string  `json:"risk_tolerance_mode"` // "conservative", "moderate", "aggressive", "high"
	MinConfidence       float64 `json:"min_confidence"`
	MaxConfidence       float64 `json:"max_confidence"`

	// Self-Optimization
	EnableSelfOptimization   bool          `json:"enable_self_optimization"`
	OptimizationInterval     time.Duration `json:"optimization_interval"`
	PerformanceWindow        time.Duration `json:"performance_window"`

	// High Risk Mode
	HighRiskMode         bool    `json:"high_risk_mode"`
	VolatilityMultiplier  float64 `json:"volatility_multiplier"`
	VolumeSpikeThreshold float64 `json:"volume_spike_threshold"`

	// Scoring System (Aggressive Autonomous)
	MinScoreThreshold    float64 `json:"min_score_threshold"` // Alert when score > 120
	VolumeSpikeMin       float64 `json:"volume_spike_min"`     // Volume >8x = 40 pts
	DeltaMin             float64 `json:"delta_min"`             // Delta >0.80 = 35 pts
	ATRMultiplier        float64 `json:"atr_multiplier"`        // ATR >5x = 25 pts
	PriceMomentumMin     float64 `json:"price_momentum_min"`    // Price +4-8% = 30 pts
	ADXMin               float64 `json:"adx_min"`               // ADX >40 = 20 pts
	BreakoutBonus        float64 `json:"breakout_bonus"`        // Breaking key level = 25 pts
	FVGBonus             float64 `json:"fvg_bonus"`             // FVG fill + volume = 25 pts
}

// DefaultDynamicScreenerConfig returns default dynamic screener configuration
func DefaultDynamicScreenerConfig() DynamicScreenerConfig {
	return DynamicScreenerConfig{
		Interval:            3 * time.Minute, // More frequent
		MaxPairs:            10,
		SortBy:              "confidence",
		MinVolume24h:        5_000_000, // Lower minimum
		MaxVolume24h:        100_000_000,
		MinPriceChange:      8.0, // Higher minimum move
		MaxPriceChange:      60.0, // Higher max for pump & dumps
		MinPrice:            0.01,
		MaxPrice:            10.0,
		RiskToleranceMode:   "aggressive", // More aggressive
		MinConfidence:       0.65, // Lower threshold
		MaxConfidence:       1.0,
		EnableSelfOptimization: true,
		OptimizationInterval:     30 * time.Minute, // More frequent
		PerformanceWindow:        12 * time.Hour, // Shorter window
		HighRiskMode:         false,
		VolatilityMultiplier:  1.5, // Higher multiplier
		VolumeSpikeThreshold: 1.3, // Lower threshold
	}
}

// HighRiskScreenerConfig returns high risk screener configuration
func HighRiskScreenerConfig() DynamicScreenerConfig {
	return DynamicScreenerConfig{
		Interval:            2 * time.Minute, // Very frequent
		MaxPairs:            20,              // More pairs
		SortBy:              "volatility",    // Sort by volatility
		MinVolume24h:        3_000_000,       // Lower min volume
		MaxVolume24h:        300_000_000,     // Higher max volume
		MinPriceChange:      5.0,             // Lower min change
		MaxPriceChange:      100.0,           // Higher max change
		MinPrice:            0.001,           // Lower min price
		MaxPrice:            50.0,            // Higher max price
		RiskToleranceMode:   "high",
		MinConfidence:       0.55,            // Even lower threshold
		MaxConfidence:       1.0,
		EnableSelfOptimization: true,
		OptimizationInterval:     15 * time.Minute, // Very frequent
		PerformanceWindow:        6 * time.Hour,    // Much shorter window
		HighRiskMode:         true,
		VolatilityMultiplier:  3.0,              // Triple volatility impact
		VolumeSpikeThreshold: 1.5,              // Lower threshold
	}
}

// AggressiveAutonomousConfig returns aggressive autonomous screener configuration with scoring system
func AggressiveAutonomousConfig() DynamicScreenerConfig {
	return DynamicScreenerConfig{
		Interval:            30 * time.Second, // Very frequent - every 30 seconds
		MaxPairs:            5,                // Top 5 ranked signals
		SortBy:              "score",          // Sort by score
		MinVolume24h:        5_000_000,        // Minimum liquidity
		MaxVolume24h:        200_000_000,      // Upper limit
		MinPriceChange:      3.0,              // Minimum 3% move
		MaxPriceChange:      15.0,             // Max 15% move (fresh moves)
		MinPrice:            0.01,
		MaxPrice:            50.0,
		RiskToleranceMode:   "aggressive",
		MinConfidence:       0.60,             // 60% threshold
		MaxConfidence:       1.0,
		EnableSelfOptimization: true,
		OptimizationInterval:     15 * time.Minute, // Frequent optimization
		PerformanceWindow:        4 * time.Hour,    // Short window
		HighRiskMode:         true,
		VolatilityMultiplier:  2.0,
		VolumeSpikeThreshold: 8.0,              // 8x volume spike required
		MinScoreThreshold:    120,              // Alert when score > 120
	}
}

// DynamicAsset represents an asset with dynamic scoring
type DynamicAsset struct {
	Symbol              string
	Volume24h           float64
	PriceChangePct      float64
	CurrentPrice        float64
	Volatility          float64
	VolumeSpike         bool
	VolumeSpikeRatio    float64 // Current volume / average volume
	Delta               float64 // Order book delta
	ATR                 float64 // Average True Range
	ADX                 float64 // Average Directional Index
	PriceMomentum       float64 // Price momentum score
	Confidence          float64
	Score               float64 // Total score (max 200)
	RiskScore           float64
	OpportunityScore    float64
	BreakoutSignal      bool   // Breaking key level
	FVGSignal           bool   // Fair Value Gap fill
	LastUpdated         time.Time
}

// ScreenerPerformance tracks screener performance
type ScreenerPerformance struct {
	TotalScreenings     int
	AssetsSelected      int
	SelectedAssets      []string
	WinRate             float64
	AvgConfidence       float64
	LastOptimization    time.Time
}

// DynamicScreener is an enhanced screener with dynamic adjustment
type DynamicScreener struct {
	client         *futures.Client
	config         DynamicScreenerConfig
	assets         []DynamicAsset
	selectedAssets []string
	performance    ScreenerPerformance
	mu             sync.RWMutex
	running        bool
	stopCh         chan struct{}
	ticker         *time.Ticker
}

// NewDynamicScreener creates a new dynamic screener
func NewDynamicScreener(client *futures.Client, config DynamicScreenerConfig) *DynamicScreener {
	ds := &DynamicScreener{
		client: client,
		config: config,
		stopCh:  make(chan struct{}),
		ticker:  time.NewTicker(config.Interval),
	}
	return ds
}

// Start begins dynamic screening
func (ds *DynamicScreener) Start(ctx context.Context) error {
	logrus.Info("🔍 Starting dynamic screener...")

	ds.mu.Lock()
	ds.running = true
	ds.mu.Unlock()

	// Initial screening
	if err := ds.refresh(ctx); err != nil {
		return err
	}

	// Start screening loop
	go ds.run(ctx)

	// Start self-optimization if enabled
	if ds.config.EnableSelfOptimization {
		go ds.runSelfOptimization(ctx)
	}

	logrus.Info("✅ Dynamic screener started")
	return nil
}

// Stop gracefully stops the screener
func (ds *DynamicScreener) Stop() {
	ds.mu.Lock()
	if ds.running {
		ds.running = false
		close(ds.stopCh)
		ds.ticker.Stop()
	}
	ds.mu.Unlock()
}

// run runs the screening loop
func (ds *DynamicScreener) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ds.stopCh:
			return
		case <-ds.ticker.C:
			ds.refresh(ctx)
		}
	}
}

// refresh refreshes the asset list
func (ds *DynamicScreener) refresh(ctx context.Context) error {
	// Get 24hr ticker statistics
	tickers, err := ds.client.NewListPriceChangeStatsService().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tickers: %w", err)
	}

	// Process tickers
	assets := ds.processTickers(tickers)

	// Apply dynamic filters
	filtered := ds.applyDynamicFilters(assets)

	// Calculate scores
	ds.calculateScores(filtered)

	// Select top assets
	selected := ds.selectTopAssets(filtered)

	ds.mu.Lock()
	ds.assets = filtered
	ds.selectedAssets = selected
	ds.performance.TotalScreenings++
	ds.performance.AssetsSelected = len(selected)
	ds.mu.Unlock()

	logrus.WithFields(logrus.Fields{
		"total_assets": len(assets),
		"filtered":     len(filtered),
		"selected":     len(selected),
	}).Info("🔍 Dynamic screening complete")

	return nil
}

// processTickers processes ticker data into assets
func (ds *DynamicScreener) processTickers(tickers []*futures.PriceChangeStats) []DynamicAsset {
	assets := make([]DynamicAsset, 0, len(tickers))

	for _, ticker := range tickers {
		// Skip non-USDT pairs
		if ticker.Symbol[len(ticker.Symbol)-4:] != "USDT" {
			continue
		}

		// Skip major coins if not in high risk mode
		if !ds.config.HighRiskMode {
			majorCoins := []string{"BTC", "ETH", "BNB", "SOL", "XRP"}
			for _, coin := range majorCoins {
				if ticker.Symbol[:len(coin)] == coin {
					continue
				}
			}
		}

		volume24h := parseFloat(ticker.QuoteVolume)
		priceChangePct := parseFloat(ticker.PriceChangePercent)
		currentPrice := parseFloat(ticker.LastPrice)

		// Calculate volatility
		volatility := ds.calculateVolatility(ticker)

		// Check for volume spike
		volumeSpike := ds.checkVolumeSpike(ticker)

		asset := DynamicAsset{
			Symbol:         ticker.Symbol,
			Volume24h:      volume24h,
			PriceChangePct: priceChangePct,
			CurrentPrice:   currentPrice,
			Volatility:     volatility,
			VolumeSpike:    volumeSpike,
			LastUpdated:    time.Now(),
		}

		assets = append(assets, asset)
	}

	return assets
}

// calculateVolatility calculates volatility for an asset
func (ds *DynamicScreener) calculateVolatility(ticker *futures.PriceChangeStats) float64 {
	priceChangePct := parseFloat(ticker.PriceChangePercent)

	// Use price change as volatility proxy
	volatility := math.Abs(priceChangePct) / 100.0

	// Apply multiplier if in high risk mode
	if ds.config.HighRiskMode {
		volatility *= ds.config.VolatilityMultiplier
	}

	return volatility
}

// checkVolumeSpike checks if there's a volume spike
func (ds *DynamicScreener) checkVolumeSpike(ticker *futures.PriceChangeStats) bool {
	// Use price change as proxy for volume spike
	// In production, this would compare to average volume
	priceChangePct := parseFloat(ticker.PriceChangePercent)

	threshold := ds.config.VolumeSpikeThreshold * 5.0 // 5% change for normal, 10% for high risk
	return math.Abs(priceChangePct) > threshold
}

// applyDynamicFilters applies dynamic filters to assets
func (ds *DynamicScreener) applyDynamicFilters(assets []DynamicAsset) []DynamicAsset {
	filtered := make([]DynamicAsset, 0, len(assets))

	for _, asset := range assets {
		if !ds.matchDynamicFilters(asset) {
			continue
		}
		filtered = append(filtered, asset)
	}

	return filtered
}

// matchDynamicFilters checks if asset matches dynamic filters
func (ds *DynamicScreener) matchDynamicFilters(asset DynamicAsset) bool {
	// Volume filter
	if asset.Volume24h < ds.config.MinVolume24h {
		return false
	}
	if ds.config.MaxVolume24h > 0 && asset.Volume24h > ds.config.MaxVolume24h {
		return false
	}

	// Price change filter
	if math.Abs(asset.PriceChangePct) < ds.config.MinPriceChange {
		return false
	}
	if ds.config.MaxPriceChange > 0 && math.Abs(asset.PriceChangePct) > ds.config.MaxPriceChange {
		return false
	}

	// Price filter
	if asset.CurrentPrice < ds.config.MinPrice {
		return false
	}
	if ds.config.MaxPrice > 0 && asset.CurrentPrice > ds.config.MaxPrice {
		return false
	}

	return true
}

// calculateScores calculates scores for assets
func (ds *DynamicScreener) calculateScores(assets []DynamicAsset) {
	for i := range assets {
		asset := &assets[i]

		// Calculate aggressive score (max 200 points)
		asset.Score = ds.calculateAggressiveScore(asset)

		// Calculate confidence score (0-1)
		asset.Confidence = ds.calculateConfidence(asset)

		// Calculate risk score (0-1)
		asset.RiskScore = ds.calculateRiskScore(asset)

		// Calculate opportunity score (0-1)
		asset.OpportunityScore = ds.calculateOpportunityScore(asset)
	}
}

// calculateAggressiveScore calculates aggressive score (max 200 points)
func (ds *DynamicScreener) calculateAggressiveScore(asset *DynamicAsset) float64 {
	score := 0.0

	// Volume >8x = 40 pts
	if asset.VolumeSpikeRatio >= ds.config.VolumeSpikeMin {
		volumeScore := 40.0
		// Bonus for extreme spikes
		if asset.VolumeSpikeRatio >= 15.0 {
			volumeScore += 10.0 // Extra 10 pts for 15x+
		}
		score += volumeScore
	}

	// Delta >0.80 = 35 pts
	if asset.Delta >= ds.config.DeltaMin {
		deltaScore := 35.0
		if asset.Delta >= 0.90 {
			deltaScore += 10.0 // Extra 10 pts for delta >0.90
		}
		score += deltaScore
	}

	// ATR >5x = 25 pts
	if asset.ATR > 0 {
		atrRatio := asset.ATR / asset.CurrentPrice
		if atrRatio >= 0.05 { // 5%
			atrScore := 25.0
			if atrRatio >= 0.08 { // 8%
				atrScore += 10.0 // Extra 10 pts for 8x+
			}
			score += atrScore
		}
	}

	// Price +4-8% momentum = 30 pts
	if math.Abs(asset.PriceChangePct) >= ds.config.PriceMomentumMin {
		priceMomentumScore := 30.0
		if math.Abs(asset.PriceChangePct) >= 8.0 {
			priceMomentumScore += 10.0 // Extra 10 pts for 8%+
		}
		score += priceMomentumScore
	}

	// ADX >40 = 20 pts
	if asset.ADX >= ds.config.ADXMin {
		adxScore := 20.0
		if asset.ADX >= 50.0 {
			adxScore += 5.0 // Extra 5 pts for ADX >50
		}
		score += adxScore
	}

	// Breaking key level = 25 pts
	if asset.BreakoutSignal {
		score += ds.config.BreakoutBonus
	}

	// FVG fill + volume = 25 pts
	if asset.FVGSignal {
		score += ds.config.FVGBonus
	}

	// Volatility bonus (up to 15 pts)
	if asset.Volatility > 0.05 {
		volatilityBonus := math.Min(asset.Volatility*100*3, 15.0)
		score += volatilityBonus
	}

	return score
}

// calculateConfidence calculates confidence score for an asset
func (ds *DynamicScreener) calculateConfidence(asset *DynamicAsset) float64 {
	confidence := 0.0

	// Volume score (0-30)
	if asset.Volume24h >= ds.config.MaxVolume24h {
		confidence += 30.0
	} else {
		volumeRatio := asset.Volume24h / ds.config.MaxVolume24h
		confidence += volumeRatio * 30.0
	}

	// Price change score (0-30)
	priceChangeRatio := math.Min(math.Abs(asset.PriceChangePct)/ds.config.MaxPriceChange, 1.0)
	confidence += priceChangeRatio * 30.0

	// Volatility score (0-20)
	if ds.config.HighRiskMode {
		// High volatility is good in high risk mode
		volatilityScore := math.Min(asset.Volatility*100, 20.0)
		confidence += volatilityScore
	} else {
		// Moderate volatility is better
		if asset.Volatility > 0.01 && asset.Volatility < 0.03 {
			confidence += 20.0
		} else if asset.Volatility >= 0.03 {
			confidence += 10.0
		} else {
			confidence += 5.0
		}
	}

	// Volume spike bonus (0-20)
	if asset.VolumeSpike {
		confidence += 20.0
	}

	// Normalize to 0-1
	confidence /= 100.0

	// Apply risk tolerance adjustment
	switch ds.config.RiskToleranceMode {
	case "conservative":
		confidence *= 0.8
	case "aggressive":
		confidence *= 1.2
	case "high":
		confidence *= 1.5
	}

	// Ensure within bounds
	if confidence < ds.config.MinConfidence {
		confidence = ds.config.MinConfidence
	}
	if confidence > ds.config.MaxConfidence {
		confidence = ds.config.MaxConfidence
	}

	return confidence
}

// calculateRiskScore calculates risk score for an asset
func (ds *DynamicScreener) calculateRiskScore(asset *DynamicAsset) float64 {
	riskScore := 0.0

	// Volatility risk (0-40)
	volatilityRisk := math.Min(asset.Volatility*100*2, 40.0)
	riskScore += volatilityRisk

	// Price change risk (0-30)
	priceChangeRisk := math.Min(math.Abs(asset.PriceChangePct)/2, 30.0)
	riskScore += priceChangeRisk

	// Volume risk (0-30)
	if asset.Volume24h < ds.config.MinVolume24h*2 {
		riskScore += 30.0
	} else if asset.Volume24h < ds.config.MinVolume24h*5 {
		riskScore += 15.0
	}

	// Normalize to 0-1
	return riskScore / 100.0
}

// calculateOpportunityScore calculates opportunity score for an asset
func (ds *DynamicScreener) calculateOpportunityScore(asset *DynamicAsset) float64 {
	opportunityScore := 0.0

	// Price change opportunity (0-40)
	priceChangeOpportunity := math.Min(math.Abs(asset.PriceChangePct), 40.0)
	opportunityScore += priceChangeOpportunity

	// Volume spike opportunity (0-30)
	if asset.VolumeSpike {
		opportunityScore += 30.0
	}

	// Volatility opportunity (0-30)
	if ds.config.HighRiskMode {
		volatilityOpportunity := math.Min(asset.Volatility*100*3, 30.0)
		opportunityScore += volatilityOpportunity
	} else {
		if asset.Volatility > 0.01 && asset.Volatility < 0.03 {
			opportunityScore += 20.0
		}
	}

	// Normalize to 0-1
	return opportunityScore / 100.0
}

// calculateOverallScore calculates overall score for an asset
func (ds *DynamicScreener) calculateOverallScore(asset *DynamicAsset) float64 {
	// Weight factors
	weightConfidence := 0.4
	weightOpportunity := 0.4
	weightRisk := 0.2

	// Calculate weighted score
	overallScore := (asset.Confidence * weightConfidence) +
		(asset.OpportunityScore * weightOpportunity) +
		((1.0 - asset.RiskScore) * weightRisk)

	return overallScore
}

// selectTopAssets selects top assets based on score
func (ds *DynamicScreener) selectTopAssets(assets []DynamicAsset) []string {
	// Sort by score
	sort.Slice(assets, func(i, j int) bool {
		if ds.config.SortBy == "confidence" {
			return assets[i].Confidence > assets[j].Confidence
		} else if ds.config.SortBy == "volatility" {
			return assets[i].Volatility > assets[j].Volatility
		} else if ds.config.SortBy == "opportunity" {
			return assets[i].OpportunityScore > assets[j].OpportunityScore
		}
		return assets[i].Score > assets[j].Score
	})

	// Select top assets
	maxPairs := ds.config.MaxPairs
	if len(assets) < maxPairs {
		maxPairs = len(assets)
	}

	selected := make([]string, maxPairs)
	for i := 0; i < maxPairs; i++ {
		selected[i] = assets[i].Symbol
	}

	return selected
}

// runSelfOptimization runs periodic self-optimization
func (ds *DynamicScreener) runSelfOptimization(ctx context.Context) {
	ticker := time.NewTicker(ds.config.OptimizationInterval)
	defer ticker.Stop()

	logrus.Info("🔍 Screener self-optimization loop started")

	for ds.running {
		select {
		case <-ticker.C:
			if err := ds.optimizeParameters(ctx); err != nil {
				logrus.WithError(err).Error("Failed to optimize screener parameters")
			}

		case <-ds.stopCh:
			return

		case <-ctx.Done():
			return
		}
	}
}

// optimizeParameters optimizes screener parameters based on performance
func (ds *DynamicScreener) optimizeParameters(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	logrus.Info("🔍 Running screener self-optimization...")

	originalConfig := ds.config

	// Adjust filters based on performance
	if ds.performance.AssetsSelected < 5 {
		// Too few assets: relax filters
		ds.config.MinVolume24h *= 0.8
		ds.config.MinPriceChange *= 0.8
		ds.config.MinConfidence *= 0.9
		logrus.Info("🔍 Relaxed filters due to low asset selection")
	} else if float64(ds.performance.AssetsSelected) > float64(ds.config.MaxPairs)*0.8 {
		// Too many assets: tighten filters
		ds.config.MinVolume24h *= 1.1
		ds.config.MinPriceChange *= 1.1
		ds.config.MinConfidence *= 1.05
		logrus.Info("🔍 Tightened filters due to high asset selection")
	}

	// Adjust interval based on market conditions
	avgVolatility := ds.calculateAverageVolatility()
	if avgVolatility > 0.03 {
		// High volatility: more frequent screening
		ds.config.Interval = time.Duration(float64(ds.config.Interval) * 0.8)
		logrus.Info("🔍 Increased screening frequency due to high volatility")
	} else if avgVolatility < 0.01 {
		// Low volatility: less frequent screening
		ds.config.Interval = time.Duration(float64(ds.config.Interval) * 1.2)
		logrus.Info("🔍 Decreased screening frequency due to low volatility")
	}

	ds.performance.LastOptimization = time.Now()

	logrus.WithFields(logrus.Fields{
		"original_min_volume": originalConfig.MinVolume24h,
		"new_min_volume":      ds.config.MinVolume24h,
		"original_min_change": originalConfig.MinPriceChange,
		"new_min_change":      ds.config.MinPriceChange,
		"original_interval":   originalConfig.Interval,
		"new_interval":        ds.config.Interval,
	}).Info("🔍 Screener self-optimization complete")

	return nil
}

// calculateAverageVolatility calculates average volatility across assets
func (ds *DynamicScreener) calculateAverageVolatility() float64 {
	if len(ds.assets) == 0 {
		return 0
	}

	totalVolatility := 0.0
	for _, asset := range ds.assets {
		totalVolatility += asset.Volatility
	}

	return totalVolatility / float64(len(ds.assets))
}

// GetSelectedAssets returns selected assets
func (ds *DynamicScreener) GetSelectedAssets() []string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	result := make([]string, len(ds.selectedAssets))
	copy(result, ds.selectedAssets)
	return result
}

// GetAssets returns all assets with scores
func (ds *DynamicScreener) GetAssets() []DynamicAsset {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	result := make([]DynamicAsset, len(ds.assets))
	copy(result, ds.assets)
	return result
}

// GetPerformance returns screener performance
func (ds *DynamicScreener) GetPerformance() ScreenerPerformance {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.performance
}

// GetConfig returns current configuration
func (ds *DynamicScreener) GetConfig() DynamicScreenerConfig {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.config
}

// parseFloat safely parses a string to float64
func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}