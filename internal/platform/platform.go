package platform

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britebrt/cognee/pkg/brain"
	"github.com/britebrt/cognee/pkg/feedback"
	"github.com/britebrt/cognee/internal/monitoring"
	"github.com/britebrt/cognee/internal/risk"
	"github.com/britebrt/cognee/internal/alerting"
	"github.com/sirupsen/logrus"
)

// Platform coordinates all Cognee components
type Platform struct {
	config       *Config
	client       *futures.Client
	brain        *brain.BrainEngine
	feedback     *feedback.CogneeFeedbackSystem
	dashboard    *monitoring.DashboardServer
	riskManager  *risk.RiskManager
	alerting     *alerting.AlertingSystem
	isRunning    bool
}

// Config holds platform configuration
type Config struct {
	Binance struct {
		APIKey    string `json:"api_key"`
		APISecret string `json:"api_secret"`
		Testnet   bool   `json:"testnet"`
	} `json:"binance"`
	
	Brain brain.BrainConfig `json:"brain"`
	
	Feedback struct {
		Enabled bool   `json:"enabled"`
		DBPath  string `json:"db_path"`
	} `json:"feedback"`
	
	WatchlistSymbols []string `json:"watchlist_symbols"`
}

// NewPlatform creates a new platform instance
func NewPlatform() *Platform {
	config := loadConfig()
	
	return &Platform{
		config: config,
	}
}

// Start initializes and starts all platform components
func (p *Platform) Start() error {
	logrus.Info("🏗️ Starting Cognee platform...")
	
	// Initialize Binance client
	if err := p.initBinanceClient(); err != nil {
		return fmt.Errorf("failed to initialize Binance client: %w", err)
	}
	
	// Initialize feedback system
	if err := p.initFeedbackSystem(); err != nil {
		return fmt.Errorf("failed to initialize feedback system: %w", err)
	}
	
	// Initialize brain engine
	if err := p.initBrainEngine(); err != nil {
		return fmt.Errorf("failed to initialize brain engine: %w", err)
	}
	
	// Initialize new components
	if err := p.initNewComponents(); err != nil {
		return fmt.Errorf("failed to initialize new components: %w", err)
	}
	
	// Start all components
	if err := p.startComponents(); err != nil {
		return fmt.Errorf("failed to start components: %w", err)
	}
	
	p.isRunning = true
	logrus.Info("✅ Cognee platform started successfully")
	
	// Start background tasks
	go p.runBackgroundTasks()
	
	return nil
}

// Stop gracefully shuts down all platform components
func (p *Platform) Stop(ctx context.Context) error {
	logrus.Info("🛑 Stopping Cognee platform...")
	
	p.isRunning = false
	
	// Stop brain engine
	if p.brain != nil {
		if err := p.brain.Stop(); err != nil {
			logrus.WithError(err).Error("Failed to stop brain engine")
		}
	}
	
	// Stop feedback system
	if p.feedback != nil {
		if err := p.feedback.Stop(); err != nil {
			logrus.WithError(err).Error("Failed to stop feedback system")
		}
	}
	
	logrus.Info("✅ Cognee platform stopped")
	return nil
}

func (p *Platform) initBinanceClient() error {
	logrus.Info("🔗 Initializing Binance client...")
	
	// Use testnet credentials if testnet mode is enabled
	apiKey := p.config.Binance.APIKey
	apiSecret := p.config.Binance.APISecret
	
	if p.config.Binance.Testnet {
		// Override with testnet credentials from environment
		testnetKey := os.Getenv("BINANCE_TESTNET_API")
		testnetSecret := os.Getenv("BINANCE_TESTNET_SECRET")
		
		if testnetKey != "" && testnetSecret != "" {
			apiKey = testnetKey
			apiSecret = testnetSecret
			logrus.Info("🧪 Using Binance testnet credentials")
		} else {
			logrus.Warn("⚠️  Testnet enabled but BINANCE_TESTNET_API/SECRET not set, using mainnet keys")
		}
		
		p.client = futures.NewClient(apiKey, apiSecret)
		p.client.BaseURL = "https://testnet.binancefuture.com"
		logrus.Info("🧪 Using Binance testnet URL")
	} else {
		p.client = futures.NewClient(apiKey, apiSecret)
	}
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := p.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	
	logrus.Info("✅ Binance client initialized")
	return nil
}

func (p *Platform) initFeedbackSystem() error {
	if !p.config.Feedback.Enabled {
		logrus.Info("Feedback system disabled - skipping initialization")
		return nil
	}
	
	logrus.Info("🔄 Initializing feedback system...")
	
	system, err := feedback.NewCogneeFeedbackSystem(
		p.config.Feedback.DBPath,
		p.client,
		"Cognee-Platform-V1",
	)
	if err != nil {
		return fmt.Errorf("failed to create feedback system: %w", err)
	}
	
	p.feedback = system
	logrus.Info("✅ Feedback system initialized")
	return nil
}

func (p *Platform) initBrainEngine() error {
	logrus.Info("🧠 Initializing brain engine...")
	
	engine, err := brain.NewBrainEngine(p.client, p.feedback, p.config.Brain)
	if err != nil {
		return fmt.Errorf("failed to create brain engine: %w", err)
	}
	
	p.brain = engine
	logrus.Info("✅ Brain engine initialized")
	return nil
}

func (p *Platform) initNewComponents() error {
	logrus.Info("🔧 Initializing new performance components...")
	
	// Initialize dashboard
	p.dashboard = monitoring.NewDashboardServer(p.client, p.feedback, p.brain, p.config.WatchlistSymbols)
	
	// Initialize risk manager
	p.riskManager = risk.NewRiskManager(p.client, p.feedback, p.config.WatchlistSymbols)
	
	// Initialize alerting system
	p.alerting = alerting.NewAlertingSystem(p.client, p.feedback, p.brain, p)
	
	return nil
}

func (p *Platform) startComponents() error {
	logrus.Info("🚀 Starting platform components...")
	
	// Start feedback system
	if p.feedback != nil {
		if err := p.feedback.Start(); err != nil {
			return fmt.Errorf("failed to start feedback system: %w", err)
		}
	}
	
	// Start brain engine
	if err := p.brain.Start(); err != nil {
		return fmt.Errorf("failed to start brain engine: %w", err)
	}
	
	// Start new components
	if p.dashboard != nil {
		if err := p.dashboard.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start dashboard: %w", err)
		}
	}
	
	if p.alerting != nil {
		if err := p.alerting.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start alerting system: %w", err)
		}
	}
	
	return nil
}

func (p *Platform) runBackgroundTasks() {
	logrus.Info("🔄 Starting background tasks...")
	
	// System health monitoring
	go p.healthMonitoring()
	
	// Performance reporting
	go p.performanceReporting()
}

func (p *Platform) healthMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for p.isRunning {
		select {
		case <-ticker.C:
			p.performHealthCheck()
		}
	}
}

func (p *Platform) performHealthCheck() {
	// Check brain engine health
	if p.brain != nil {
		stats := p.brain.GetEngineStats()
		logrus.WithFields(logrus.Fields{
			"uptime":      stats["uptime"],
			"decisions":   stats["decisions_made"],
			"provider":    stats["provider"].(map[string]interface{})["model"],
			"healthy":     stats["provider"].(map[string]interface{})["healthy"],
		}).Debug("Health check completed")
	}
}

func (p *Platform) performanceReporting() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for p.isRunning {
		select {
		case <-ticker.C:
			p.generatePerformanceReport()
		}
	}
}

func (p *Platform) generatePerformanceReport() {
	if p.brain == nil {
		return
	}
	
	stats := p.brain.GetEngineStats()
	
	logrus.WithFields(logrus.Fields{
		"uptime":         stats["uptime"],
		"total_decisions": stats["decisions_made"],
		"recoveries":     stats["recoveries"],
		"provider_model": stats["provider"].(map[string]interface{})["model"],
		"provider_healthy": stats["provider"].(map[string]interface{})["healthy"],
	}).Info("Performance report generated")
}

func loadConfig() *Config {
	config := &Config{}
	
	// Load from environment variables
	config.Binance.APIKey = os.Getenv("BINANCE_API_KEY")
	config.Binance.APISecret = os.Getenv("BINANCE_API_SECRET")
	config.Binance.Testnet = getEnvBool("BINANCE_USE_TESTNET", false)
	
	// Load watchlist symbols
	watchlistStr := os.Getenv("WATCHLIST_SYMBOLS")
	if watchlistStr != "" {
		config.WatchlistSymbols = strings.Split(watchlistStr, ",")
	} else {
		// Default watchlist
		config.WatchlistSymbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	}
	
	// Brain configuration
	config.Brain = brain.DefaultBrainConfig()
	config.Brain.InferenceMode = getEnvString("INFERENCE_MODE", "LOCAL")
	config.Brain.LocalModel = "cognee-brain"  // Use our custom model
	config.Brain.CloudAPIKey = os.Getenv("OPENAI_API_KEY")
	config.Brain.CloudProvider = getEnvString("CLOUD_PROVIDER", "openai")
	config.Brain.EnableRecovery = getEnvBool("ENABLE_RECOVERY", true)
	
	// Feedback configuration
	config.Feedback.Enabled = getEnvBool("FEEDBACK_ENABLED", true)
	config.Feedback.DBPath = getEnvString("FEEDBACK_DB_PATH", "cognee_production.db")
	
	return config
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}