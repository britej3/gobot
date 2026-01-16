package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britebrt/cognee/config"
	"github.com/britebrt/cognee/internal/position"
	"github.com/britebrt/cognee/internal/striker"
	"github.com/britebrt/cognee/pkg/brain"
	"github.com/britebrt/cognee/services/screener"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// AutonomousBot is the fully autonomous trading bot with all advanced features
type AutonomousBot struct {
	binanceClient      *futures.Client
	brain              *brain.BrainEngine
	dynamicManager     *position.DynamicManager
	trailingManager    *position.TrailingManager
	enhancedStriker    *striker.EnhancedStriker
	dynamicScreener    *screener.DynamicScreener
	prodConfig         *config.ProductionConfig
	stopChan           chan struct{}
	running            bool
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Load environment variables from .env file
	if err := godotenv.Load("/Users/britebrt/GOBOT/.env"); err != nil {
		logrus.WithError(err).Warn("Failed to load .env file, using system environment variables")
	}

	logrus.Info("🚀 Starting GOBOT Autonomous Trading Bot v3.0")
	logrus.Info("🎯 Balance: 26 USDT | Aggressive Mid-Cap Trading")

	ctx := context.Background()

	// Load production configuration
	prodConfig, err := config.LoadProductionConfig(ctx, "/Users/britebrt/GOBOT/config/config.yaml")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load production config")
	}

	logrus.WithFields(logrus.Fields{
		"initial_capital": prodConfig.Trading.InitialCapitalUSD,
		"min_position":    prodConfig.Trading.MaxPositionUSD * 0.6,
		"max_position":    prodConfig.Trading.MaxPositionUSD,
		"stop_loss":       prodConfig.Trading.StopLossPercent,
		"take_profit":     prodConfig.Trading.TakeProfitPercent,
	}).Info("Configuration loaded")

	// Initialize Binance client
	binanceClient := futures.NewClient(
		prodConfig.Binance.APIKey,
		prodConfig.Binance.APISecret,
	)
	if prodConfig.Binance.UseTestnet {
		binanceClient.BaseURL = "https://testnet.binance.vision"
	}

	// Initialize brain engine
	brainEngine, err := brain.NewBrainEngine(
		binanceClient,
		nil,
		brain.DefaultBrainConfig(),
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize brain engine")
	}

	// Initialize dynamic manager
	dynamicManager := position.NewDynamicManager(
		binanceClient,
		position.DefaultDynamicConfig(),
	)

	// Initialize trailing manager with aggressive autonomous configuration
	trailingManager := position.NewTrailingManager(
		binanceClient,
		position.AggressiveAutonomousConfig(),
	)

	// Initialize enhanced striker
	enhancedStriker := striker.NewEnhancedStriker(
		binanceClient,
		brainEngine,
		dynamicManager,
		trailingManager,
		true, // high risk mode
	)

	// Initialize dynamic screener with aggressive autonomous configuration
	dynamicScreener := screener.NewDynamicScreener(
		binanceClient,
		screener.AggressiveAutonomousConfig(),
	)

	// Create autonomous bot
	bot := &AutonomousBot{
		binanceClient:      binanceClient,
		brain:              brainEngine,
		dynamicManager:     dynamicManager,
		trailingManager:    trailingManager,
		enhancedStriker:    enhancedStriker,
		dynamicScreener:    dynamicScreener,
		prodConfig:         prodConfig,
		stopChan:           make(chan struct{}),
		running:            false,
	}

	// Start the bot
	if err := bot.Start(); err != nil {
		logrus.WithError(err).Fatal("Failed to start bot")
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logrus.Info("🛑 Shutting down...")
	bot.Stop()
	logrus.Info("✅ Shutdown complete")
}

// Start begins autonomous trading
func (ab *AutonomousBot) Start() error {
	logrus.Info("🎯 Initializing autonomous trading mode...")

	// Start all components
	if err := ab.dynamicScreener.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start dynamic screener: %w", err)
	}

	if err := ab.dynamicManager.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start dynamic manager: %w", err)
	}

	if err := ab.trailingManager.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start trailing manager: %w", err)
	}

	if err := ab.enhancedStriker.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start enhanced striker: %w", err)
	}

	ab.running = true

	// Start main trading loop
	go ab.tradingLoop()

	// Start monitoring loop
	go ab.monitoringLoop()

	logrus.Info("✅ Autonomous trading bot started successfully")
	logrus.Info("📊 Trading Loop: Every 3 minutes")
	logrus.Info("🎯 Position Sizing: Dynamic based on volatility & confidence")
	logrus.Info("🛡️ Risk Management: 1.5% SL, 5% TP, 1% trailing")
	logrus.Info("📈 Target: 15-25% monthly returns with <13 USDT drawdown")

	return nil
}

// Stop gracefully shuts down the bot
func (ab *AutonomousBot) Stop() {
	logrus.Info("🛑 Stopping autonomous trading bot...")
	ab.running = false
	close(ab.stopChan)

	// Stop all components
	ab.enhancedStriker.Stop()
	ab.trailingManager.Stop()
	ab.dynamicManager.Stop()
	ab.dynamicScreener.Stop()
}

// tradingLoop runs the main autonomous trading logic
func (ab *AutonomousBot) tradingLoop() {
	ticker := time.NewTicker(3 * time.Minute) // Trade every 3 minutes
	defer ticker.Stop()

	logrus.Info("🎯 Trading loop started (every 3 minutes)")

	// Give screener time to initialize
	time.Sleep(5 * time.Second)

	for ab.running {
		select {
		case <-ticker.C:
			ab.executeTradingCycle()
		case <-ab.stopChan:
			return
		}
	}
}

// executeTradingCycle executes one complete trading cycle with rotation logic
func (ab *AutonomousBot) executeTradingCycle() {
	logrus.Info("🔄 Starting trading cycle...")

	ctx := context.Background()

	// Step 1: Get current positions
	positions, err := ab.binanceClient.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get positions")
		return
	}

	// Count open positions
	openPositions := 0
	var positionSymbols []string
	for _, pos := range positions {
		positionAmt, _ := parseFloat(pos.PositionAmt)
		if positionAmt != 0 {
			openPositions++
			positionSymbols = append(positionSymbols, pos.Symbol)
		}
	}

	logrus.WithFields(logrus.Fields{
		"open_positions": openPositions,
		"symbols":        positionSymbols,
	}).Info("Current positions")

	// Step 2: Get top assets from dynamic screener
	tickers, err := ab.binanceClient.NewListPriceChangeStatsService().Do(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get price change stats")
		return
	}

	// Filter for high volatility mid-caps with aggressive criteria
	var topAssets []interface{}
	for _, ticker := range tickers {
		priceChangePercent, _ := parseFloat(ticker.PriceChangePercent)
		quoteVolume, _ := parseFloat(ticker.QuoteVolume)

		// Aggressive filters: 3-15% move, 5M+ volume
		if priceChangePercent > 3.0 && priceChangePercent < 15.0 && quoteVolume > 5000000 {
			topAssets = append(topAssets, ticker)
			if len(topAssets) >= 5 {
				break
			}
		}
	}

	if len(topAssets) == 0 {
		logrus.Warn("No assets selected by screener")
		return
	}

	logrus.WithField("count", len(topAssets)).Info("Assets selected by screener")

	// Step 3: Use enhanced striker to analyze and score assets
	decision, err := ab.enhancedStriker.Execute(ctx, topAssets)
	if err != nil {
		logrus.WithError(err).Error("Failed to execute enhanced striker")
		return
	}

	if decision != nil && len(decision.TopTargets) > 0 {
		topTarget := decision.TopTargets[0]

		// Check if score meets threshold (120+ points)
		if topTarget.ConfidenceScore > 120.0 {
			// Rotation logic: Check if we should replace a weak position
			if openPositions >= 3 {
				// Find weakest position (lowest PnL or dying momentum)
				weakestPos := ab.findWeakestPosition(ctx, positions)
				if weakestPos != nil {
					entryPrice, _ := parseFloat(weakestPos.EntryPrice)
					currentPrice, _ := parseFloat(weakestPos.MarkPrice)

					// Calculate PnL percentage
					pnlPct := ((currentPrice - entryPrice) / entryPrice) * 100

					// Replace if weak performance (<5% gain or losing)
					if pnlPct < 5.0 {
						logrus.WithFields(logrus.Fields{
							"symbol":         weakestPos.Symbol,
							"pnl_percent":    pnlPct,
							"replacement":    topTarget.Symbol,
							"replacement_score": topTarget.ConfidenceScore,
						}).Info("Rotating out weak position")

						// Close weak position
						ab.closePosition(ctx, weakestPos.Symbol)

						// Enter new position
						ab.enterPosition(ctx, topTarget)
					} else {
						logrus.WithFields(logrus.Fields{
							"symbol":     topTarget.Symbol,
							"action":     topTarget.Action,
							"confidence": topTarget.ConfidenceScore,
							"reason":     "Max positions reached with good performers",
						}).Info("High confidence signal - skipping (rotation not needed)")
					}
				}
			} else {
				// Enter new position (space available)
				ab.enterPosition(ctx, topTarget)
			}
		} else {
			logrus.WithFields(logrus.Fields{
				"symbol":     topTarget.Symbol,
				"score":      topTarget.ConfidenceScore,
				"threshold":  120.0,
				"reason":     "Score below threshold",
			}).Info("Signal below threshold - skipping")
		}
	} else {
		logrus.Info("No high confidence signals found")
	}
}

// findWeakestPosition finds the weakest position to rotate out
func (ab *AutonomousBot) findWeakestPosition(ctx context.Context, positions []*futures.PositionRisk) *futures.PositionRisk {
	var weakest *futures.PositionRisk
	weakestPnl := 999999.0

	for _, pos := range positions {
		positionAmt, _ := parseFloat(pos.PositionAmt)
		if positionAmt == 0 {
			continue
		}

		unrealizedPnl, _ := parseFloat(pos.UnRealizedProfit)
		if unrealizedPnl < weakestPnl {
			weakestPnl = unrealizedPnl
			weakest = pos
		}
	}

	return weakest
}

// enterPosition enters a new position with aggressive parameters
func (ab *AutonomousBot) enterPosition(ctx context.Context, target brain.TargetAsset) {
	logrus.WithFields(logrus.Fields{
		"symbol":     target.Symbol,
		"action":     target.Action,
		"confidence": target.ConfidenceScore,
		"entry":      target.EntryZone,
		"tp":         target.TakeProfit,
		"sl":         target.StopLoss,
	}).Info("Entering position")

	// Calculate position size (90% of max for aggressive)
	positionSize := ab.prodConfig.Trading.MaxPositionUSD * 0.90

	// Calculate leverage based on confidence (10-30x)
	leverage := 10
	if target.ConfidenceScore > 150 {
		leverage = 25
	} else if target.ConfidenceScore > 130 {
		leverage = 20
	} else if target.ConfidenceScore > 120 {
		leverage = 15
	}

	// Execute trade via enhanced striker
	ab.enhancedStriker.ExecuteEnhancedTrade(
		ctx,
		target.Symbol,
		target.Action,
		positionSize,
		leverage,
		target.EntryZone,
		target.TakeProfit,
		target.StopLoss,
	)
}

// closePosition closes a position
func (ab *AutonomousBot) closePosition(ctx context.Context, symbol string) {
	logrus.WithField("symbol", symbol).Info("Closing position")

	// Get current position
	positions, err := ab.binanceClient.NewGetPositionRiskService().Symbol(symbol).Do(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get position")
		return
	}

	if len(positions) == 0 {
		return
	}

	positionAmt, _ := parseFloat(positions[0].PositionAmt)
	if positionAmt == 0 {
		return
	}

	// Determine side
	side := "SELL"
	if positionAmt < 0 {
		side = "BUY"
	}

	// Close position
	_, err = ab.binanceClient.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideType(side)).
		Type(futures.OrderTypeMarket).
		Quantity(fmt.Sprintf("%.6f", math.Abs(positionAmt))).
		Do(ctx)

	if err != nil {
		logrus.WithError(err).Error("Failed to close position")
		return
	}

	logrus.WithField("symbol", symbol).Info("Position closed successfully")
}

// monitoringLoop monitors positions and manages risk
func (ab *AutonomousBot) monitoringLoop() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	logrus.Info("🛡️ Monitoring loop started (every 30 seconds)")

	for ab.running {
		select {
		case <-ticker.C:
			ab.checkAndManagePositions()
		case <-ab.stopChan:
			return
		}
	}
}

// checkAndManagePositions checks all positions and applies risk management
func (ab *AutonomousBot) checkAndManagePositions() {
	ctx := context.Background()

	// Get current positions
	positions, err := ab.binanceClient.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get positions")
		return
	}

	openPositions := 0
	for _, pos := range positions {
		positionAmt, _ := parseFloat(pos.PositionAmt)
		if positionAmt != 0 {
			openPositions++
			logrus.WithFields(logrus.Fields{
				"symbol":          pos.Symbol,
				"position_amt":    positionAmt,
				"entry_price":     pos.EntryPrice,
				"unrealized_pnl":  pos.UnRealizedProfit,
			}).Info("Position found")
		}
	}

	if openPositions > 0 {
		logrus.WithField("count", openPositions).Info("Managing open positions")
		// Trailing manager will automatically update SL/TP
	}
}

// parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}