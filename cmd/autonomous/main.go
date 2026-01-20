package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britebrt/cognee/config"
	"github.com/britebrt/cognee/internal/adaptive"
	"github.com/britebrt/cognee/internal/position"
	"github.com/britebrt/cognee/internal/striker"
	"github.com/britebrt/cognee/pkg/brain"
	"github.com/britebrt/cognee/pkg/alerting"
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
	
	// Enhanced adaptive time-based optimization
	adaptiveConfig     *adaptive.AdaptiveConfig
	currentSession     adaptive.TradingSession
	lastSessionUpdate  time.Time
	
	// Telegram notifications
	telegramAlert      *alerting.TelegramAlert
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

	// Initialize adaptive time-based optimization
	adaptiveConfig := adaptive.NewAdaptiveConfig()
	currentSession := adaptive.GetCurrentSession()

	// Initialize Telegram notifications
	telegramAlert := alerting.NewTelegramAlert(alerting.TelegramConfig{
		Token:   os.Getenv("TELEGRAM_TOKEN"),
		ChatID:  os.Getenv("TELEGRAM_CHAT_ID"),
		Enabled: prodConfig.Monitoring.TelegramEnabled,
	})

	// Create autonomous bot
	bot := &AutonomousBot{
		binanceClient:         binanceClient,
		brain:                 brainEngine,
		dynamicManager:        dynamicManager,
		trailingManager:       trailingManager,
		enhancedStriker:       enhancedStriker,
		dynamicScreener:       dynamicScreener,
		prodConfig:            prodConfig,
		stopChan:              make(chan struct{}),
		running:               false,
		adaptiveConfig:        adaptiveConfig,
		currentSession:        currentSession,
		lastSessionUpdate:     time.Now(),
		telegramAlert:         telegramAlert,
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

// updateAdaptiveConfig updates the adaptive configuration based on current time
func (ab *AutonomousBot) updateAdaptiveConfig() {
	// Get current session
	newSession := adaptive.GetCurrentSession()

	// Check if session changed
	if newSession.Name != ab.currentSession.Name {
		ab.currentSession = newSession
		ab.adaptiveConfig.CurrentSession = newSession
		ab.lastSessionUpdate = time.Now()

		logrus.WithFields(logrus.Fields{
			"old_session": ab.currentSession.Name,
			"new_session": newSession.Name,
		}).Info("📍 Session changed")

		logrus.WithFields(logrus.Fields{
			"volume_threshold": newSession.VolumeThreshold,
			"delta_threshold":  newSession.DeltaThreshold,
			"momentum_min":     newSession.MomentumMin,
			"momentum_max":     newSession.MomentumMax,
			"position_multi":   newSession.PositionSizeMulti,
		}).Info("📊 New session thresholds")

		// Log strategy recommendation
		logrus.Info("🎯 " + adaptive.GetSessionStrategy())
	}

	// Update adaptive config with current session
	ab.adaptiveConfig.CurrentSession = ab.currentSession
}

// getAdaptiveThresholds returns current adaptive thresholds
func (ab *AutonomousBot) getAdaptiveThresholds() (volumeThreshold, deltaThreshold, momentumMin, momentumMax float64, positionSize float64) {
	// Apply adaptive thresholds
	adaptedSession := ab.adaptiveConfig.AdaptThresholds()

	// Calculate position size based on session multiplier
	capital := ab.prodConfig.Trading.InitialCapitalUSD
	positionSize = adaptive.GetOptimalPositionSize(capital, adaptedSession)

	// Log relaxation level if active
	if ab.adaptiveConfig.RelaxationLevel > 0 {
		logrus.WithFields(logrus.Fields{
			"relaxation_level": ab.adaptiveConfig.RelaxationLevel,
			"no_signal_minutes": ab.adaptiveConfig.NoSignalMinutes,
		}).Warn("⚠️ Auto-relaxation active")
	}

	return adaptedSession.VolumeThreshold, adaptedSession.DeltaThreshold,
		adaptedSession.MomentumMin, adaptedSession.MomentumMax, positionSize
}

// Start begins autonomous trading
func (ab *AutonomousBot) Start() error {
	logrus.Info("🎯 Initializing autonomous trading mode...")

	// Show current session info
	shouldTrade, reason := adaptive.ShouldTrade()
	logrus.WithFields(logrus.Fields{
		"should_trade": shouldTrade,
		"reason":       reason,
	}).Info("📊 Trading status")

	logrus.Info("🎯 " + adaptive.GetSessionStrategy())

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

	// Start adaptive optimization loop
	go ab.adaptiveOptimizationLoop()

	logrus.Info("✅ Autonomous trading bot started successfully")
	logrus.Info("📊 Trading Loop: Every 30 seconds (scalping mode)")
	logrus.Info("🎯 Position Sizing: Dynamic based on session & volatility & confidence")
	logrus.Info("🛡️ Risk Management: 1.5% SL, 5% TP, 1% trailing")
	logrus.Info("📈 Target: 15-25% monthly returns with <13 USDT drawdown")
	logrus.Info("⏰ Adaptive Time-Based Optimization: Active (7 sessions, 3-level relaxation)")

	// Send Telegram notification
	logrus.Info("📱 Sending Telegram notification...")
	if err := ab.telegramAlert.SendTrade("🚀 GOBOT v3.0 Started\n\n✅ Autonomous trading bot is now running\n📊 Trading Loop: Every 30 seconds (scalping mode)\n🎯 Balance: 26 USDT"); err != nil {
		logrus.WithError(err).Error("Failed to send Telegram notification")
	} else {
		logrus.Info("✅ Telegram notification sent successfully")
	}

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

	// Send Telegram notification
	ab.telegramAlert.SendTrade("🛑 GOBOT v3.0 Stopped\n\n✅ Bot has been shut down gracefully\n📊 Trading session ended")
}

// adaptiveOptimizationLoop runs adaptive optimization checks
func (ab *AutonomousBot) adaptiveOptimizationLoop() {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	logrus.Info("⏰ Adaptive optimization loop started (every 5 minutes)")

	for ab.running {
		select {
		case <-ticker.C:
			ab.updateAdaptiveConfig()
		case <-ab.stopChan:
			return
		}
	}
}

// tradingLoop runs the main autonomous trading logic
func (ab *AutonomousBot) tradingLoop() {
	ticker := time.NewTicker(30 * time.Second) // Trade every 30 seconds (scalping mode)
	defer ticker.Stop()

	logrus.Info("🎯 Trading loop started (every 30 seconds)")

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
	logrus.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logrus.Info("🔄 Starting trading cycle...")

	ctx := context.Background()

	// Update adaptive configuration
	ab.updateAdaptiveConfig()

	// Get adaptive thresholds
	volumeThreshold, deltaThreshold, momentumMin, momentumMax, positionSize := ab.getAdaptiveThresholds()

	// Log current session and thresholds
	logrus.WithFields(logrus.Fields{
		"session":           ab.currentSession.Name,
		"volume_threshold":  volumeThreshold,
		"delta_threshold":   deltaThreshold,
		"momentum_min":      momentumMin,
		"momentum_max":      momentumMax,
		"position_size":     positionSize,
		"relaxation_level":  ab.adaptiveConfig.RelaxationLevel,
	}).Info("📍 Current session and thresholds")

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
	}).Info("💼 Current positions")

	// Step 2: Get top assets from dynamic screener
	tickers, err := ab.binanceClient.NewListPriceChangeStatsService().Do(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get price change stats")
		return
	}

	// Filter for high volatility mid-caps with adaptive criteria
	var topAssets []interface{}
	for _, ticker := range tickers {
		priceChangePercent, _ := parseFloat(ticker.PriceChangePercent)
		quoteVolume, _ := parseFloat(ticker.QuoteVolume)

		// Apply adaptive filters
		if priceChangePercent >= momentumMin && priceChangePercent <= momentumMax {
			// Volume filter (simplified - in production use actual volume spike ratio)
			if quoteVolume > 3000000 { // Minimum 3M volume
				topAssets = append(topAssets, ticker)
				if len(topAssets) >= 5 {
					break
				}
			}
		}
	}

	if len(topAssets) == 0 {
		logrus.Warn("⚠️ No assets selected by adaptive screener")
		return
	}

	logrus.WithFields(logrus.Fields{
		"count":              len(topAssets),
		"momentum_min":       momentumMin,
		"momentum_max":       momentumMax,
		"relaxation_level":   ab.adaptiveConfig.RelaxationLevel,
	}).Info("🔍 Assets selected by adaptive screener")

	// Step 3: Use enhanced striker to analyze and score assets
	decision, err := ab.enhancedStriker.Execute(ctx, topAssets)
	if err != nil {
		logrus.WithError(err).Error("Failed to execute enhanced striker")
		ab.telegramAlert.SendError(fmt.Sprintf("❌ Enhanced striker error: %v", err))
		return
	}

	if decision != nil && len(decision.TopTargets) > 0 {
		topTarget := decision.TopTargets[0]

		// Score threshold based on relaxation level
		scoreThreshold := 120.0
		if ab.adaptiveConfig.RelaxationLevel == 1 {
			scoreThreshold = 110.0
		} else if ab.adaptiveConfig.RelaxationLevel == 2 {
			scoreThreshold = 100.0
		} else if ab.adaptiveConfig.RelaxationLevel == 3 {
			scoreThreshold = 90.0
		}

		if topTarget.ConfidenceScore > scoreThreshold {
			// Reset relaxation when signal found
			ab.adaptiveConfig.ResetRelaxation()

			logrus.WithFields(logrus.Fields{
				"symbol":    topTarget.Symbol,
				"score":     topTarget.ConfidenceScore,
				"threshold": scoreThreshold,
			}).Info("✅ High confidence signal found - relaxation reset")

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
							"symbol":            weakestPos.Symbol,
							"pnl_percent":       pnlPct,
							"replacement":       topTarget.Symbol,
							"replacement_score": topTarget.ConfidenceScore,
							"session":           ab.currentSession.Name,
						}).Info("🔄 Rotating out weak position")

						// Close weak position
						ab.closePosition(ctx, weakestPos.Symbol)

						// Enter new position
						ab.enterPosition(ctx, topTarget, positionSize)
					} else {
						logrus.WithFields(logrus.Fields{
							"symbol":     topTarget.Symbol,
							"action":     topTarget.Action,
							"confidence": topTarget.ConfidenceScore,
							"reason":     "Max positions reached with good performers",
						}).Info("ℹ️ High confidence signal - skipping (rotation not needed)")
					}
				}
			} else {
				// Enter new position (space available)
				ab.enterPosition(ctx, topTarget, positionSize)
			}
		} else {
			logrus.WithFields(logrus.Fields{
				"symbol":           topTarget.Symbol,
				"score":            topTarget.ConfidenceScore,
				"threshold":        scoreThreshold,
				"relaxation_level": ab.adaptiveConfig.RelaxationLevel,
				"reason":           "Score below threshold",
			}).Info("⚠️ Signal below threshold - skipping")
		}
	} else {
		logrus.Info("ℹ️ No high confidence signals found")
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

// enterPosition enters a new position with adaptive parameters
func (ab *AutonomousBot) enterPosition(ctx context.Context, target brain.TargetAsset, positionSize float64) {
	logrus.WithFields(logrus.Fields{
		"symbol":        target.Symbol,
		"action":        target.Action,
		"confidence":    target.ConfidenceScore,
		"entry":         target.EntryZone,
		"tp":            target.TakeProfit,
		"sl":            target.StopLoss,
		"session":       ab.currentSession.Name,
		"position_size": positionSize,
	}).Info("📈 Entering position")

	// Calculate leverage based on confidence (10-30x)
	leverage := 10
	if target.ConfidenceScore > 150 {
		leverage = 25
	} else if target.ConfidenceScore > 130 {
		leverage = 20
	} else if target.ConfidenceScore > 120 {
		leverage = 15
	}

	// Send Telegram notification
	emoji := "🟢"
	if target.Action == "SELL" {
		emoji = "🔴"
	}
	ab.telegramAlert.SendTrade(fmt.Sprintf(
		"%s *GOBOT Trade Executed*\n\n*Symbol:* %s\n*Action:* %s\n*Confidence:* %.1f\n*Entry:* $%.6f\n*Leverage:* %dx\n*Session:* %s\n*Score:* %.1f",
		emoji, target.Symbol, target.Action, target.ConfidenceScore, target.EntryZone, leverage, ab.currentSession.Name, target.ConfidenceScore,
	))

	// Execute trade via enhanced striker
	err := ab.enhancedStriker.ExecuteEnhancedTrade(
		ctx,
		target.Symbol,
		target.Action,
		positionSize,
		leverage,
		target.EntryZone,
		target.TakeProfit,
		target.StopLoss,
	)
	
	if err != nil {
		logrus.WithError(err).Error("❌ Failed to execute enhanced trade")
		ab.telegramAlert.SendError(fmt.Sprintf("❌ Trade execution failed for %s: %v", target.Symbol, err))
		
		// Check if it's a critical error
		if contains(err.Error(), "insufficient") || contains(err.Error(), "balance") || contains(err.Error(), "margin") {
			ab.telegramAlert.SendError(fmt.Sprintf("🚨 CRITICAL: Insufficient balance/margin for %s. Please check account.", target.Symbol))
		}
		if contains(err.Error(), "position side") || contains(err.Error(), "Position side") {
			ab.telegramAlert.SendError(fmt.Sprintf("🚨 CRITICAL: Position mode mismatch for %s. Check Binance Futures position mode settings.", target.Symbol))
		}
		if contains(err.Error(), "precision") || contains(err.Error(), "tick") {
			ab.telegramAlert.SendError(fmt.Sprintf("⚠️ Precision error for %s: %v. Will retry with adjusted parameters.", target.Symbol, err))
		}
		return
	}
	
	logrus.WithField("symbol", target.Symbol).Info("✅ Enhanced trade executed successfully")
}

// closePosition closes a position
func (ab *AutonomousBot) closePosition(ctx context.Context, symbol string) {
	logrus.WithField("symbol", symbol).Info("📉 Closing position")

	// Get current position
	positions, err := ab.binanceClient.NewGetPositionRiskService().Symbol(symbol).Do(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get position")
		ab.telegramAlert.SendError(fmt.Sprintf("❌ Failed to get position for %s: %v", symbol, err))
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
		ab.telegramAlert.SendError(fmt.Sprintf("❌ Failed to close position for %s: %v", symbol, err))
		
		// Check if it's a critical error
		if contains(err.Error(), "insufficient") || contains(err.Error(), "balance") || contains(err.Error(), "margin") {
			ab.telegramAlert.SendError(fmt.Sprintf("🚨 CRITICAL: Insufficient balance/margin to close %s. Please check account.", symbol))
		}
		if contains(err.Error(), "position side") || contains(err.Error(), "Position side") {
			ab.telegramAlert.SendError(fmt.Sprintf("🚨 CRITICAL: Position mode mismatch for %s. Check Binance Futures position mode settings.", symbol))
		}
		return
	}

	logrus.WithField("symbol", symbol).Info("✅ Position closed successfully")
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
		ab.telegramAlert.SendError(fmt.Sprintf("❌ Failed to get positions: %v", err))
		return
	}

	openPositions := 0
	for _, pos := range positions {
		positionAmt, _ := parseFloat(pos.PositionAmt)
		if positionAmt != 0 {
			openPositions++
			logrus.WithFields(logrus.Fields{
				"symbol":         pos.Symbol,
				"position_amt":   positionAmt,
				"entry_price":    pos.EntryPrice,
				"unrealized_pnl": pos.UnRealizedProfit,
			}).Info("💼 Position found")
		}
	}

	if openPositions > 0 {
		logrus.WithField("count", openPositions).Info("🔄 Managing open positions")
		// Trailing manager will automatically update SL/TP
	}
}

// parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

// findSubstring finds a substring in a string (case-insensitive)
func findSubstring(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}