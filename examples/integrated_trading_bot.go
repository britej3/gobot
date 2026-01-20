package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britej3/gobot/infra/binance"
	"github.com/britej3/gobot/infra/errors"
	"github.com/britej3/gobot/infra/monitoring"
	"github.com/britej3/gobot/infra/ratelimit"
	"github.com/sirupsen/logrus"
)

// IntegratedTradingBot demonstrates all components working together
type IntegratedTradingBot struct {
	// Core components
	futuresClient  *binance.FuturesClient
	reporter       *monitoring.Reporter
	healthChecker  *monitoring.HealthChecker
	errorHandler   *errors.ErrorHandler

	// Configuration
	config         *BotConfig
	logger         *logrus.Logger

	// Control
	ctx            context.Context
	cancel         context.CancelFunc
}

// BotConfig holds bot configuration
type BotConfig struct {
	BinanceAPIKey    string
	BinanceAPISecret string
	Testnet          bool
	RedisAddr        string
	RedisPassword    string
	HealthCheckPort  string
	TradingSymbol    string
	InitialCapital   float64
}

// NewIntegratedTradingBot creates a new integrated trading bot
func NewIntegratedTradingBot(config *BotConfig) *IntegratedTradingBot {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	ctx, cancel := context.WithCancel(context.Background())

	bot := &IntegratedTradingBot{
		config: config,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize components
	bot.initializeComponents()

	return bot
}

// initializeComponents initializes all bot components
func (bot *IntegratedTradingBot) initializeComponents() {
	bot.logger.Info("initializing_components")

	// Initialize error handler
	bot.errorHandler = errors.NewErrorHandler(errors.ErrorHandlerConfig{
		MaxErrors: 1000,
		PanicMode: false,
	})

	// Register error callbacks
	bot.errorHandler.RegisterErrorCallback(func(record *errors.ErrorRecord) {
		bot.logger.WithFields(logrus.Fields{
			"error": record.Error.Error(),
			"type":  record.Type,
		}).Error("error_callback_triggered")
	})

	// Initialize reporter
	bot.reporter = monitoring.NewReporter(monitoring.ReporterConfig{
		ReportInterval: 10 * time.Second,
		MaxEvents:      1000,
	})

	// Initialize health checker
	bot.healthChecker = monitoring.NewHealthChecker()

	// Initialize Futures client
	bot.futuresClient = binance.NewFuturesClient(binance.FuturesConfig{
		APIKey:    bot.config.BinanceAPIKey,
		APISecret: bot.config.BinanceAPISecret,
		Testnet:   bot.config.Testnet,
		PoolSize:  10,
		Redis: binance.RedisConfig{
			Addr:     bot.config.RedisAddr,
			Password: bot.config.RedisPassword,
			DB:       0,
		},
	})

	// Register health checks
	bot.registerHealthChecks()

	bot.logger.Info("components_initialized")
}

// registerHealthChecks registers all health checks
func (bot *IntegratedTradingBot) registerHealthChecks() {
	// Binance API health check
	bot.healthChecker.RegisterCheck(monitoring.NewBinanceHealthCheck(func(ctx context.Context) error {
		_, err := bot.futuresClient.GetExchangeInfo(ctx)
		return err
	}))

	// Circuit breaker health check
	bot.healthChecker.RegisterCheck(monitoring.NewCircuitBreakerHealthCheck(func() string {
		return string(bot.futuresClient.circuitBreaker.GetState())
	}))

	// Rate limiter health check
	bot.healthChecker.RegisterCheck(monitoring.NewRateLimiterHealthCheck(func() (float64, error) {
		stats := bot.futuresClient.rateLimiter.GetStats()
		return stats.MaxUsagePercentage, nil
	}))

	bot.logger.Info("health_checks_registered")
}

// Start starts the trading bot
func (bot *IntegratedTradingBot) Start() error {
	bot.logger.Info("starting_trading_bot")

	// Record startup event
	bot.reporter.RecordSystemEvent("Trading bot started", monitoring.SeverityInfo, map[string]interface{}{
		"config": bot.config,
	})

	// Start health check server
	go func() {
		if err := bot.healthChecker.StartHTTPServer(bot.config.HealthCheckPort); err != nil {
			bot.logger.WithError(err).Error("health_check_server_failed")
		}
	}()

	// Initialize trading session
	if err := bot.initializeTradingSession(); err != nil {
		return bot.errorHandler.Handle(bot.ctx, err, errors.ErrorTypeSystem, map[string]interface{}{
			"operation": "initialize_trading_session",
		})
	}

	// Start trading loop
	go bot.tradingLoop()

	// Start monitoring loop
	go bot.monitoringLoop()

	// Wait for shutdown signal
	bot.waitForShutdown()

	return nil
}

// initializeTradingSession initializes the trading session
func (bot *IntegratedTradingBot) initializeTradingSession() error {
	bot.logger.Info("initializing_trading_session")

	ctx, cancel := context.WithTimeout(bot.ctx, 30*time.Second)
	defer cancel()

	// Get account information
	account, err := bot.futuresClient.GetAccount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}

	bot.logger.WithFields(logrus.Fields{
		"available_balance": account.AvailableBalance,
		"total_balance":     account.TotalWalletBalance,
	}).Info("account_info_retrieved")

	// Record account metrics
	bot.reporter.RecordMetric("account_balance", account.AvailableBalance, "USDT", nil)
	bot.reporter.RecordMetric("total_wallet_balance", account.TotalWalletBalance, "USDT", nil)

	// Set position mode to one-way
	if err := bot.futuresClient.SetPositionMode(ctx, false); err != nil {
		bot.logger.WithError(err).Warn("failed_to_set_position_mode")
	}

	// Set margin type to isolated for the trading symbol
	if err := bot.futuresClient.SetMarginType(ctx, bot.config.TradingSymbol, futures.MarginTypeIsolated); err != nil {
		bot.logger.WithError(err).Warn("failed_to_set_margin_type")
	}

	// Set initial leverage
	if err := bot.futuresClient.SetLeverage(ctx, bot.config.TradingSymbol, 10); err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	bot.logger.Info("trading_session_initialized")
	return nil
}

// tradingLoop is the main trading loop
func (bot *IntegratedTradingBot) tradingLoop() {
	bot.logger.Info("trading_loop_started")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bot.ctx.Done():
			bot.logger.Info("trading_loop_stopped")
			return
		case <-ticker.C:
			if err := bot.executeTradingCycle(); err != nil {
				bot.errorHandler.Handle(bot.ctx, err, errors.ErrorTypeExecution, map[string]interface{}{
					"operation": "trading_cycle",
				})
			}
		}
	}
}

// executeTradingCycle executes one trading cycle
func (bot *IntegratedTradingBot) executeTradingCycle() error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		bot.reporter.RecordPerformanceMetric("trading_cycle", duration)
	}()

	ctx, cancel := context.WithTimeout(bot.ctx, 10*time.Second)
	defer cancel()

	// Get current positions
	positions, err := bot.futuresClient.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	bot.logger.WithFields(logrus.Fields{
		"position_count": len(positions),
	}).Info("positions_retrieved")

	// Record position metrics
	for _, position := range positions {
		bot.reporter.RecordPositionMetric(position.Symbol, position.PositionAmt, position.UnrealizedProfit)
		
		bot.reporter.RecordPositionEvent(
			fmt.Sprintf("Position update: %s", position.Symbol),
			monitoring.SeverityInfo,
			map[string]interface{}{
				"symbol":            position.Symbol,
				"size":              position.PositionAmt,
				"unrealized_profit": position.UnrealizedProfit,
				"entry_price":       position.EntryPrice,
				"mark_price":        position.MarkPrice,
			},
		)
	}

	// Get mark price for trading symbol
	markPrice, err := bot.futuresClient.GetMarkPrice(ctx, bot.config.TradingSymbol)
	if err != nil {
		return fmt.Errorf("failed to get mark price: %w", err)
	}

	bot.reporter.RecordMetric("mark_price", markPrice, "USDT", map[string]string{
		"symbol": bot.config.TradingSymbol,
	})

	bot.logger.WithFields(logrus.Fields{
		"symbol":     bot.config.TradingSymbol,
		"mark_price": markPrice,
	}).Info("mark_price_retrieved")

	return nil
}

// monitoringLoop monitors system health and performance
func (bot *IntegratedTradingBot) monitoringLoop() {
	bot.logger.Info("monitoring_loop_started")

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bot.ctx.Done():
			bot.logger.Info("monitoring_loop_stopped")
			return
		case <-ticker.C:
			bot.performHealthCheck()
			bot.reportStatistics()
		}
	}
}

// performHealthCheck performs a health check
func (bot *IntegratedTradingBot) performHealthCheck() {
	ctx, cancel := context.WithTimeout(bot.ctx, 30*time.Second)
	defer cancel()

	status := bot.healthChecker.CheckHealth(ctx)

	bot.logger.WithFields(logrus.Fields{
		"status":       status.Status,
		"check_count":  len(status.Checks),
		"uptime":       status.Uptime,
	}).Info("health_check_completed")

	// Send alert if unhealthy
	if status.Status != "healthy" {
		bot.reporter.SendAlert(
			monitoring.AlertLevelWarning,
			"System Health Warning",
			"One or more health checks failed",
			map[string]interface{}{
				"status": status,
			},
		)
	}
}

// reportStatistics reports system statistics
func (bot *IntegratedTradingBot) reportStatistics() {
	// Get error statistics
	errorStats := bot.errorHandler.GetErrorStats()

	bot.logger.WithFields(logrus.Fields{
		"total_errors":   errorStats.Total,
		"recovered":      errorStats.Recovered,
		"recovery_rate":  errorStats.RecoveryRate,
	}).Info("error_statistics")

	// Get report
	report := bot.reporter.GetReport()

	bot.logger.WithFields(logrus.Fields{
		"metrics_count": len(report.Metrics),
		"events_count":  len(report.RecentEvents),
	}).Info("system_report")
}

// waitForShutdown waits for shutdown signal
func (bot *IntegratedTradingBot) waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	bot.logger.WithFields(logrus.Fields{
		"signal": sig.String(),
	}).Info("shutdown_signal_received")

	bot.Shutdown()
}

// Shutdown gracefully shuts down the bot
func (bot *IntegratedTradingBot) Shutdown() {
	bot.logger.Info("shutting_down")

	// Cancel context
	bot.cancel()

	// Close positions if configured
	// bot.closeAllPositions()

	// Stop health check server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := bot.healthChecker.Stop(ctx); err != nil {
		bot.logger.WithError(err).Error("failed_to_stop_health_checker")
	}

	// Close reporter
	bot.reporter.Close()

	// Record shutdown event
	bot.reporter.RecordSystemEvent("Trading bot shutdown", monitoring.SeverityInfo, nil)

	bot.logger.Info("shutdown_complete")
}

// Main function
func main() {
	// Load configuration from environment
	config := &BotConfig{
		BinanceAPIKey:    os.Getenv("BINANCE_API_KEY"),
		BinanceAPISecret: os.Getenv("BINANCE_API_SECRET"),
		Testnet:          os.Getenv("BINANCE_TESTNET") == "true",
		RedisAddr:        getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		HealthCheckPort:  getEnvOrDefault("HEALTH_CHECK_PORT", ":8080"),
		TradingSymbol:    getEnvOrDefault("TRADING_SYMBOL", "BTCUSDT"),
		InitialCapital:   100.0,
	}

	// Validate configuration
	if config.BinanceAPIKey == "" || config.BinanceAPISecret == "" {
		fmt.Println("Error: BINANCE_API_KEY and BINANCE_API_SECRET must be set")
		os.Exit(1)
	}

	// Create and start bot
	bot := NewIntegratedTradingBot(config)
	
	if err := bot.Start(); err != nil {
		fmt.Printf("Error starting bot: %v\n", err)
		os.Exit(1)
	}
}

// Helper function
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
