package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/britebrt/cognee/pkg/brain"
	"github.com/britebrt/cognee/pkg/platform"
	internalPlatform "github.com/britebrt/cognee/internal/platform"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		logrus.Warn("⚠️  No .env file found, using system environment variables")
	}

	// Parse command line flags
	var (
		testTrade = flag.Bool("test-trade", false, "Execute a test trade to verify AI connection")
		symbol    = flag.String("symbol", "BTCUSDT", "Symbol for test trade")
		side      = flag.String("side", "BUY", "Side for test trade (BUY/SELL)")
		aggressive = flag.Bool("aggressive", false, "Use aggressive thresholds for testing")
		auditOnly = flag.Bool("audit", false, "Run API audit only and exit")
	)
	flag.Parse()
	
	// Initialize production logging
	setupLogging()
	
	logrus.Info("🚀 COGNEE PRODUCTION SYSTEM - Starting complete integration...")
	logrus.Info("🧠 Brain: AI Engine with Dual Inference")
	logrus.Info("🔄 Feedback: Continuous Improvement Loop")
	logrus.Info("💾 Recovery: Startup Safety Net")
	logrus.Info("📊 Analytics: Performance Tracking")
	
	// Pre-flight audit: Check API connection and balances
	useTestnet := os.Getenv("BINANCE_USE_TESTNET") == "true"
	logrus.Info("🔍 Pre-flight Audit: Checking API and Balances...")
	
	status := internalPlatform.CheckConnection(useTestnet)
	internalPlatform.PrintAuditReport(status)
	
	if !status.IsConnected {
		logrus.Fatal("🚫 CRITICAL: Could not establish API connection. Check your keys and IP whitelist.")
	}
	
	// Handle audit-only mode
	if *auditOnly {
		logrus.Info("✅ Audit complete. Exiting as requested.")
		return
	}
	
	// Handle test trade mode
	if *testTrade {
		logrus.Info("🧪 TEST TRADE MODE - Running AI decision test")
		runTestTrade(*symbol, *side, *aggressive)
		return
	}

	// Initialize platform
	platform := platform.NewPlatform()
	if err := platform.Start(); err != nil {
		logrus.Fatalf("❌ Platform initialization failed: %v", err)
	}

	// Setup graceful shutdown
	setupGracefulShutdown(platform)

	logrus.Info("✅ Cognee production system initialized successfully")
	logrus.Info("🎯 System is ready for high-frequency scalping with AI intelligence")
	
	// Keep main running
	select {}
}

func setupLogging() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logrus.SetLevel(logrus.InfoLevel)
	
	// Add system fields
	logrus.WithFields(logrus.Fields{
		"system":    "cognee",
		"version":   "1.0.0",
		"component": "main",
	}).Info("Production logging configured")
}

func runTestTrade(symbol, side string, aggressive bool) {
	logrus.WithFields(logrus.Fields{
		"symbol":     symbol,
		"side":       side,
		"aggressive": aggressive,
	}).Info("Running test trade to verify AI connection")
	
	// Create a test brain engine
	config := brain.DefaultBrainConfig()
	if aggressive {
		// Use aggressive settings for testing
		config.LocalModel = "qwen3:0.6b"
		config.LocalBaseURL = "http://localhost:11964"
		config.InferenceMode = "LOCAL"
		logrus.Info("Using aggressive test settings")
	}
	
	engine, err := brain.NewBrainEngine(nil, nil, config)
	if err != nil {
		logrus.Fatalf("Failed to create brain engine: %v", err)
	}
	
	// Create test signal
	signal := struct {
		Symbol        string  `json:"symbol"`
		FVGZone       string  `json:"fvg_zone"`
		FVGConfidence float64 `json:"fvg_confidence"`
		CVDDivergence bool    `json:"cvd_divergence"`
		Volatility    float64 `json:"volatility"`
		MarketRegime  string  `json:"market_regime"`
		Confidence    float64 `json:"confidence"`
		Side          string  `json:"side"`
	}{
		Symbol:        symbol,
		FVGZone:       "BULLISH",
		FVGConfidence: 0.65, // Lower confidence for testing
		CVDDivergence: true,
		Volatility:    0.018, // 1.8% volatility
		MarketRegime:  "RANGING",
		Confidence:    0.75,
		Side:          side,
	}
	
	logrus.WithField("signal", signal).Info("Sending test signal to AI brain")
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	// Get trading decision from AI
	decision, err := engine.MakeTradingDecision(ctx, signal)
	if err != nil {
		logrus.WithError(err).Error("Failed to get trading decision from AI")
		logrus.Info("💡 This might be due to JSON parsing. Trying simple prompt...")
		
		// Try simple direct prompt
		prompt := fmt.Sprintf(`You are GOBOT's trading decision AI. Evaluate this signal for %s on %s with FVG confidence 0.65 and CVD divergence. Return ONLY JSON: {"decision": "%s", "confidence": 0.75, "reasoning": "Test signal"}`, side, symbol, side)
		
		// Try direct provider approach
		// Create a simple provider for testing
		testProvider, err := brain.NewOllamaProvider(brain.OllamaConfig{
			Model:       "qwen3:0.6b",
			BaseURL:     "http://localhost:11964",
			Temperature: 0.1,
			Timeout:     10 * time.Second,
		})
		if err != nil {
			logrus.WithError(err).Fatal("Failed to create test provider")
		}
		
		response, err := testProvider.GenerateResponse(ctx, prompt)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to get simple response from AI")
		}
		
		logrus.WithField("response", response).Info("✅ AI responded to simple prompt")
		return
	}
	
	logrus.WithFields(logrus.Fields{
		"decision": decision.Decision,
		"confidence": decision.Confidence,
		"reasoning": decision.Reasoning,
		"risk_level": decision.RiskLevel,
		"recommended_leverage": decision.RecommendedLeverage,
	}).Info("✅ AI Trading Decision Received!")
	
	logrus.Info("🎉 Test trade completed successfully! AI connection verified.")
	logrus.Info("You can now start the full platform: ./cognee")
}

func setupGracefulShutdown(platform *platform.Platform) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		logrus.Info("🛑 Shutdown signal received - initiating graceful shutdown...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := platform.Stop(ctx); err != nil {
			logrus.WithError(err).Error("Failed to stop platform gracefully")
		}
		
		logrus.Info("✅ Graceful shutdown completed")
		os.Exit(0)
	}()
}