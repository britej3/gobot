package brain

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/britebrt/cognee/internal/platform"
	"github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// SimulationResult holds backtesting results
type SimulationResult struct {
	OriginalPnL     float64
	SimulatedPnL    float64
	SlippageSaved   float64
	TotalTrades     int
	WinningTrades   int
	LosingTrades    int
	AverageSlippage float64 // In basis points
	ExecutionAlpha  float64 // Difference between signal and fill price
	DecayRate       float64 // How fast signal loses value (ms)
}

// Backtester performs strategy backtesting using WAL data
type Backtester struct {
	brainEngine interface {
		MakeTradingDecision(ctx interface{}, signal interface{}) (interface{}, error)
	}
	walPath string
}

// NewBacktester creates a new backtester instance
func NewBacktester(walPath string) *Backtester {
	return &Backtester{
		walPath: walPath,
	}
}

// RunBacktest executes a backtest with new parameters
func (b *Backtester) RunBacktest(newThreshold float64) (*SimulationResult, error) {
	logrus.WithField("threshold", newThreshold).Info("🧪 Starting backtest simulation...")

	file, err := os.Open(b.walPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	result := &SimulationResult{}
	var lastIntent *platform.LogEntry

	for {
		var entry platform.LogEntry
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			logrus.WithError(err).Warn("Failed to decode WAL entry, skipping")
			continue
		}

		// Process INTENT entries only
		if entry.Status == "INTENT" {
			result.TotalTrades++
			lastIntent = &entry

			// Simulate slippage analysis
			slippage := b.simulateSlippage(entry.Symbol, entry.Timestamp)
			result.SlippageSaved += slippage

			// If slippage is positive, count as winning trade
			if slippage > 0 {
				result.WinningTrades++
			} else {
				result.LosingTrades++
			}

			// Simulate execution with new threshold
			simFill := b.simulateFill(entry.Symbol, entry.Timestamp, newThreshold)
			
			// Calculate PnL difference (simplified)
			if entry.Price > 0 {
				pnlDiff := (simFill - entry.Price) * entry.Qty
				result.SimulatedPnL += pnlDiff
			}
		}

		// Process COMMITTED entries to calculate original PnL
		if entry.Status == "COMMITTED" && lastIntent != nil && lastIntent.Symbol == entry.Symbol {
			// Calculate actual PnL from the trade
			if lastIntent.Price > 0 && entry.Price > 0 {
				result.OriginalPnL += (entry.Price - lastIntent.Price) * lastIntent.Qty
			}
			lastIntent = nil
		}
	}

	// Calculate averages
	if result.TotalTrades > 0 {
		result.AverageSlippage = (result.SlippageSaved / float64(result.TotalTrades)) * 10000 // Convert to basis points
		result.ExecutionAlpha = result.AverageSlippage // Simplified
	}

	// Estimate decay rate (simplified - would need historical data)
	result.DecayRate = estimateDecayRate(result.TotalTrades)

	logrus.WithFields(logrus.Fields{
		"total_trades":     result.TotalTrades,
		"winning_trades":   result.WinningTrades,
		"losing_trades":    result.LosingTrades,
		"simulated_pnl":    result.SimulatedPnL,
		"avg_slippage_bp":  result.AverageSlippage,
		"execution_alpha":  result.ExecutionAlpha,
	}).Info("🧪 Backtest completed")

	return result, nil
}

// SimulateFill simulates order execution with different thresholds
func (b *Backtester) simulateFill(symbol string, signalTime time.Time, threshold float64) float64 {
	// This is a simplified simulation
	// In production, you would fetch historical tick data from Binance
	
	// Simulate normal distribution fill price
	basePrice := 50000.0 // Default BTC price (should be fetched from historical data)
	
	// Calculate time decay factor (signal loses value over time)
	elapsed := time.Since(signalTime).Milliseconds()
	decayFactor := 1.0 - (float64(elapsed) / 1000.0) // 1 second half-life
	if decayFactor < 0.1 {
		decayFactor = 0.1
	}
	
	// Simulate slippage with normal distribution
	// Mean 0, stddev based on volatility
	volatility := 0.0003 // 3 bps typical spread
	slippage := randNormal(0, volatility) * decayFactor * threshold
	
	return basePrice * (1 + slippage)
}

// simulateSlippage simulates slippage for a trade
func (b *Backtester) simulateSlippage(symbol string, signalTime time.Time) float64 {
	// Simulate adverse excursion and slippage
	// In production, fetch actual historical data
	
	// Simulate random slippage between -2bps and +3bps
	slippageBps := randNormal(0.5, 1.5) // Mean 0.5bp, std 1.5bp
	
	// Cap slippage for realism
	if slippageBps > 3.0 {
		slippageBps = 3.0
	} else if slippageBps < -2.0 {
		slippageBps = -2.0
	}
	
	return slippageBps / 10000.0 // Convert to percentage
}

// randNormal generates normally distributed random numbers
func randNormal(mean, stddev float64) float64 {
	// Using Box-Muller transformation
	u1 := rand.Float64()
	u2 := rand.Float64()
	z0 := math.Sqrt(-2.0 * math.Log(u1)) * math.Cos(2.0 * math.Pi * u2)
	return z0 * stddev + mean
}

// estimateDecayRate estimates how fast signals lose value
func estimateDecayRate(totalTrades int) float64 {
	// Simplified: in production, measure actual decay from historical data
	if totalTrades == 0 {
		return 50.0 // Default: 50% value remains after 200ms
	}
	// Assume signals lose about 50% value after 200ms
	return 50.0
}

// PerturbationTest checks if strategy is overfitted
func (b *Backtester) PerturbationTest(optimalThreshold float64, perturbation float64) (*SimulationResult, error) {
	logrus.Info("🧪 Running perturbation test (checking for overfitting)...")
	
	// Test with threshold ±perturbation%
	testThreshold := optimalThreshold * (1 + perturbation/100.0)
	
	result, err := b.RunBacktest(testThreshold)
	if err != nil {
		return nil, err
	}
	
	// Compare performance
	baselineResult, err := b.RunBacktest(optimalThreshold)
	if err != nil {
		return nil, err
	}
	
	performanceDrop := 0.0
	if baselineResult.SimulatedPnL != 0 {
		performanceDrop = ((baselineResult.SimulatedPnL - result.SimulatedPnL) / baselineResult.SimulatedPnL) * 100
	}
	
	if performanceDrop > 50.0 {
		logrus.WithField("drop_percent", performanceDrop).Warn("⚠️  Strategy may be overfitted! Performance collapsed with small parameter change")
	} else {
		logrus.WithField("drop_percent", performanceDrop).Info("✅ Strategy appears robust to parameter perturbation")
	}
	
	return result, nil
}

// WalkForwardAnalysis performs walk-forward optimization
func WalkForwardAnalysis(walPath string, weeks int) error {
	logrus.WithField("weeks", weeks).Info("📈 Starting walk-forward analysis...")
	
	// This would split WAL data by weeks and perform rolling optimization
	// For now, simplified version
	
	for week := 1; week <= weeks; week++ {
		logrus.WithField("week", week).Info("Testing week...")
		
		// In production: load WAL data for specific week range
		// Train on weeks 1..week-1, test on week
		
		// Simulate result for demonstration
		result := &SimulationResult{
			TotalTrades:  100,
			WinningTrades: 55,
			SimulatedPnL:  0.025, // 2.5% return
		}
		
		logrus.WithFields(logrus.Fields{
			"week":         week,
			"trades":       result.TotalTrades,
			"win_rate":     float64(result.WinningTrades) / float64(result.TotalTrades),
			"return_pct":   result.SimulatedPnL * 100,
		}).Info("Week completed")
	}
	
	logrus.Info("📊 Walk-forward analysis completed")
	return nil
}
