package platform

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britebrt/cognee/internal/agent"
	"github.com/britebrt/cognee/internal/platform"
	"github.com/britebrt/cognee/internal/watcher"
	"github.com/britebrt/cognee/pkg/brain"
	"github.com/sirupsen/logrus"
)

// Platform coordinates all GOBOT components
type Platform struct {
	config        *Config
	client        *futures.Client
	brain         *brain.BrainEngine
	feedback      *CogneeFeedbackSystem // Use concrete type instead of interface
	scanner       *watcher.AssetScanner
	striker       *watcher.StrikerExecutor
	stateManager  *StateManager
	reconciler    *agent.Reconciler
	wal           *platform.WAL
	isRunning     bool
	stopChan      chan struct{}
	initialBalance float64 // For Safe-Stop monitoring
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
	
	SafeStop struct {
		Enabled           bool    `json:"enabled"`
		ThresholdPercent  float64 `json:"threshold_percent"`
		MinBalanceUSD     float64 `json:"min_balance_usd"`
		InitialBalance    float64 `json:"initial_balance"`
		CheckInterval     time.Duration `json:"check_interval"`
	} `json:"safe_stop"`
}

// Simple feedback system interface
type CogneeFeedbackSystem struct {
	client   *futures.Client
	botName  string
	enabled  bool
}

// NewCogneeFeedbackSystem creates a new feedback system
func NewCogneeFeedbackSystem(dbPath string, client *futures.Client, botName string) *CogneeFeedbackSystem {
	return &CogneeFeedbackSystem{
		client:  client,
		botName: botName,
		enabled: true,
	}
}

// Start begins the feedback system
func (s *CogneeFeedbackSystem) Start() error {
	logrus.WithField("bot_name", s.botName).Info("GOBOT feedback system started")
	return nil
}

// Stop gracefully stops the feedback system
func (s *CogneeFeedbackSystem) Stop() error {
	logrus.WithField("bot_name", s.botName).Info("GOBOT feedback system stopped")
	return nil
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
	logrus.Info("🏗️ Starting GOBOT platform with LiquidAI LFM2.5...")
	
	// Initialize stop channel for Safe-Stop monitoring
	p.stopChan = make(chan struct{})
	
	// Initialize Binance client
	if err := p.initBinanceClient(); err != nil {
		return fmt.Errorf("failed to initialize Binance client: %w", err)
	}
	
	// Initialize WAL for recovery
	if err := p.initWAL(); err != nil {
		logrus.WithError(err).Warn("Failed to initialize WAL, continuing without it")
	}
	
	// Initialize state manager and restore previous state
	sessionID := fmt.Sprintf("%d", time.Now().Unix())
	p.stateManager = NewStateManager(sessionID)
	
	// Load previous state if exists
	if prevState, err := p.stateManager.Load(); err != nil {
		logrus.WithError(err).Warn("Failed to load previous state, starting fresh")
	} else if prevState != nil {
		logrus.WithFields(logrus.Fields{
			"positions": len(prevState.OpenPositions),
			"balance":   prevState.TotalBalance,
		}).Info("✅ Previous state restored")
	}
	
	// Initialize reconciler for ghost position detection
	if err := p.initReconciler(); err != nil {
		logrus.WithError(err).Warn("Failed to initialize reconciler, continuing without it")
	}
	
	// Run startup reconciliation to catch ghost positions
	if p.reconciler != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		logrus.Info("🔍 Running startup reconciliation for ghost positions...")
		if err := p.reconciler.Reconcile(ctx); err != nil {
			logrus.WithError(err).Error("Startup reconciliation failed")
		}
	}
	
	// Start auto-save goroutine
	p.stateManager.StartAutoSave()
	
	// Initialize feedback system
	if err := p.initFeedbackSystem(); err != nil {
		return fmt.Errorf("failed to initialize feedback system: %w", err)
	}
	
	// Initialize brain engine
	if err := p.initBrainEngine(); err != nil {
		return fmt.Errorf("failed to initialize brain engine: %w", err)
	}
	
	// Initialize asset scanner
	if err := p.initAssetScanner(); err != nil {
		return fmt.Errorf("failed to initialize asset scanner: %w", err)
	}
	
	// Initialize striker workflow
	if err := p.initStrikerWorkflow(); err != nil {
		return fmt.Errorf("failed to initialize striker workflow: %w", err)
	}
	
	// Start Safe-Stop monitoring if enabled
	p.startSafeStopMonitor()
	
	// Start all components
	if err := p.startComponents(); err != nil {
		return fmt.Errorf("failed to start components: %w", err)
	}
	
	p.isRunning = true
	logrus.Info("✅ GOBOT platform with LiquidAI LFM2.5 started successfully")
	
	// Start background tasks
	go p.runBackgroundTasks()
	
	return nil
}

// Stop gracefully shuts down all platform components
func (p *Platform) Stop(ctx context.Context) error {
	logrus.Info("🛑 Stopping GOBOT platform...")
	
	p.isRunning = false
	
	// Signal Safe-Stop monitor to stop
	if p.stopChan != nil {
		close(p.stopChan)
	}
	
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
	
	logrus.Info("✅ GOBOT platform stopped")
	return nil
}

func (p *Platform) initBinanceClient() error {
	logrus.Info("🔗 Initializing Binance client for GOBOT...")
	
	p.client = futures.NewClient(p.config.Binance.APIKey, p.config.Binance.APISecret)
	
	if p.config.Binance.Testnet {
		p.client.BaseURL = "https://testnet.binancefuture.com"
		logrus.Info("🧪 Using Binance testnet for GOBOT")
	}
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := p.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	
	logrus.Info("✅ Binance client initialized for GOBOT")
	return nil
}

func (p *Platform) initFeedbackSystem() error {
	if !p.config.Feedback.Enabled {
		logrus.Info("Feedback system disabled for GOBOT - skipping initialization")
		return nil
	}
	
	logrus.Info("🔄 Initializing GOBOT feedback system...")
	
	system := NewCogneeFeedbackSystem(
		p.config.Feedback.DBPath,
		p.client,
		"GOBOT-LFM25-V1",
	)
	
	p.feedback = system
	logrus.Info("✅ GOBOT feedback system initialized")
	return nil
}

func (p *Platform) initBrainEngine() error {
	logrus.Info("🧠 Initializing GOBOT LiquidAI LFM2.5 brain engine...")
	
	engine, err := brain.NewBrainEngine(p.client, p.feedback, p.config.Brain)
	if err != nil {
		return fmt.Errorf("failed to create brain engine: %w", err)
	}
	
	p.brain = engine
	logrus.Info("✅ GOBOT LiquidAI LFM2.5 brain engine initialized")
	return nil
}

// initAssetScanner initializes the dynamic asset scanner
func (p *Platform) initAssetScanner() error {
	logrus.Info("🔍 Initializing dynamic asset scanner...")
	
	// Create scanner configuration
	config := watcher.DefaultScannerConfig()
	
	// Override with environment variables
	if minVol := os.Getenv("MIN_24H_VOLUME"); minVol != "" {
		if val, err := strconv.ParseFloat(minVol, 64); err == nil {
			config.Min24hVolumeUSD = val
		}
	}
	if minATR := os.Getenv("MIN_ATR_PERCENT"); minATR != "" {
		if val, err := strconv.ParseFloat(minATR, 64); err == nil {
			config.MinATRPercent = val
		}
	}
	if maxAssets := os.Getenv("MAX_ASSETS"); maxAssets != "" {
		if val, err := strconv.Atoi(maxAssets); err == nil {
			config.MaxAssets = val
		}
	}
	
	p.scanner = watcher.NewAssetScanner(p.client, config)
	
	if err := p.scanner.Start(); err != nil {
		return fmt.Errorf("failed to start asset scanner: %w", err)
	}
	
	logrus.Info("✅ Dynamic asset scanner initialized")
	return nil
}

// initStrikerWorkflow initializes the striker trading workflow
func (p *Platform) initStrikerWorkflow() error {
	logrus.Info("🎯 Initializing striker workflow...")
	
	// Pass the Binance client to striker executor for real trade execution
	p.striker = watcher.NewStrikerExecutor(p.brain, p.scanner, p.client)
	
	logrus.Info("✅ Striker workflow initialized with REAL trade execution")
	return nil
}

func (p *Platform) startComponents() error {
	logrus.Info("🚀 Starting GOBOT platform components...")
	
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
	
	// Start asset scanner
	if p.scanner != nil {
		logrus.Info("🔍 Starting asset scanner...")
		// Already started in init, but ensure it's running
	}
	
	// Start striker workflow
	if p.striker != nil {
		logrus.Info("🎯 Starting striker workflow...")
		go p.runStrikerLoop()
	}
	
	return nil
}

func (p *Platform) runBackgroundTasks() {
	logrus.Info("🔄 Starting GOBOT background tasks...")
	
	// System health monitoring
	go p.healthMonitoring()
	
	// Performance reporting
	go p.performanceReporting()
	
	// Soft reconciliation (every 60 minutes)
	go p.softReconciliationLoop()
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
			"base_url":    stats["provider"].(map[string]interface{})["base_url"],
		}).Debug("GOBOT health check completed")
	}
}

// runStrikerLoop continuously executes the striker workflow
func (p *Platform) runStrikerLoop() {
	ticker := time.NewTicker(2 * time.Minute) // Check every 2 minutes for new opportunities
	defer ticker.Stop()
	
	logrus.Info("🎯 Striker workflow loop started - scanning for opportunities")
	
	for p.isRunning {
		select {
		case <-ticker.C:
			logrus.Info("🎯 Striker: Analyzing market for scalping opportunities...")
			
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			
			// Execute striker workflow
			decision, err := p.striker.Execute(ctx)
			if err != nil {
				logrus.WithError(err).Warn("🎯 Striker: No opportunities found this cycle")
			} else {
				// Process the decision
				p.processStrikerDecision(decision)
			}
			
			cancel()
			
		case <-p.stopChan:
			logrus.Info("🎯 Striker workflow stopped")
			return
		}
	}
}

// processStrikerDecision handles the striker decision
func (p *Platform) processStrikerDecision(decision *brain.StrikerDecision) {
	if len(decision.TopTargets) == 0 {
		logrus.Info("🎯 Striker: No actionable targets")
		return
	}
	
	for i, target := range decision.TopTargets {
		logrus.WithFields(logrus.Fields{
			"rank":       i + 1,
			"symbol":     target.Symbol,
			"action":     target.Action,
			"confidence": target.ConfidenceScore,
		}).Info("🎯 STKR: Target identified")
		
		// Queue trade for execution
		go p.executeStrikerTrade(target)
	}
}

// executeStrikerTrade executes a striker-identified trade
func (p *Platform) executeStrikerTrade(target brain.TargetAsset) {
	logrus.WithFields(logrus.Fields{
		"symbol":     target.Symbol,
		"action":     target.Action,
		"entry":      target.EntryZone,
		"stop":       target.StopLoss,
		"tp":         target.TakeProfit,
		"confidence": target.ConfidenceScore,
	}).Info("🎯 Executing striker trade")
	
	// Save state BEFORE execution (WAL pattern)
	if p.stateManager != nil {
		pos := PositionState{
			Symbol:     target.Symbol,
			Side:       target.Action,
			EntryPrice: target.EntryZone,
			Quantity:   target.AllocationMultiplier, // Simplified for now
			StopLoss:   target.StopLoss,
			TakeProfit: target.TakeProfit,
			OpenedAt:   time.Now(),
			Confidence: target.ConfidenceScore,
		}
		p.stateManager.AddPosition(pos)
		logrus.Debug("State saved before trade execution")
	}
	
	// Increment decision counter
	if p.brain != nil {
		p.brain.IncrementDecisions()
	}
	
	// Log to feedback system (simplified for now)
	if p.feedback != nil {
		logrus.WithFields(logrus.Fields{
			"symbol":   target.Symbol,
			"position": target.Action,
			"entry":    target.EntryZone,
			"confidence": target.ConfidenceScore,
		}).Info("Logged to feedback system")
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
		"base_url":       stats["provider"].(map[string]interface{})["base_url"],
	}).Info("GOBOT LFM2.5 performance report generated")
}

func loadConfig() *Config {
	config := &Config{}
	
	// Load from environment variables
	config.Binance.APIKey = os.Getenv("BINANCE_API_KEY")
	config.Binance.APISecret = os.Getenv("BINANCE_API_SECRET")
	config.Binance.Testnet = getEnvBool("BINANCE_USE_TESTNET", true)
	
	// Brain configuration - updated for LiquidAI LFM2.5
	config.Brain = brain.DefaultBrainConfig()
	config.Brain.InferenceMode = getEnvString("INFERENCE_MODE", "LOCAL")
	config.Brain.LocalModel = getEnvString("OLLAMA_MODEL", "LiquidAI/LFM2.5-1.2B-Instruct-GGUF/LFM2.5-1.2B-Instruct-Q8_0.gguf")
	config.Brain.LocalBaseURL = getEnvString("OLLAMA_BASE_URL", "http://localhost:11454")
	config.Brain.CloudAPIKey = os.Getenv("OPENAI_API_KEY")
	config.Brain.CloudProvider = getEnvString("CLOUD_PROVIDER", "openai")
	config.Brain.EnableRecovery = getEnvBool("ENABLE_RECOVERY", true)
	
	// Feedback configuration
	config.Feedback.Enabled = getEnvBool("FEEDBACK_ENABLED", true)
	config.Feedback.DBPath = getEnvString("FEEDBACK_DB_PATH", "gobot_lfm25_production.db")
	
	// Safe-Stop configuration
	config.SafeStop.Enabled = getEnvBool("SAFE_STOP_ENABLED", true)
	config.SafeStop.ThresholdPercent = getEnvFloat("SAFE_STOP_THRESHOLD_PERCENT", 10.0)
	config.SafeStop.MinBalanceUSD = getEnvFloat("SAFE_STOP_MIN_BALANCE_USD", 100.0)
	config.SafeStop.CheckInterval = time.Duration(getEnvInt("SAFE_STOP_CHECK_INTERVAL", 300)) * time.Second // 5 minutes default
	
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

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if val, err := strconv.ParseFloat(value, 64); err == nil {
			return val
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if val, err := strconv.Atoi(value); err == nil {
			return val
		}
	}
	return defaultValue
}

// startSafeStopMonitor begins balance monitoring for automatic stop-loss
func (p *Platform) startSafeStopMonitor() {
	if !p.config.SafeStop.Enabled {
		logrus.Info("🛡️  Safe-Stop protection disabled")
		return
	}
	
	logrus.WithFields(logrus.Fields{
		"threshold_percent": p.config.SafeStop.ThresholdPercent,
		"min_balance_usd":   p.config.SafeStop.MinBalanceUSD,
		"check_interval":    p.config.SafeStop.CheckInterval,
	}).Info("🛡️  Starting Safe-Stop balance monitor")
	
	// Get initial balance for comparison
	ctx := context.Background()
	initialBalance, err := p.getCurrentBalance(ctx)
	if err != nil {
		logrus.WithError(err).Warn("⚠️  Could not fetch initial balance for Safe-Stop")
		return
	}
	
	p.initialBalance = initialBalance
	p.config.SafeStop.InitialBalance = initialBalance
	
	logrus.WithField("initial_balance", initialBalance).Info("🛡️  Safe-Stop baseline established")
	
	// Start monitoring goroutine
	go p.monitorBalance()
}

// monitorBalance continuously checks balance against thresholds
func (p *Platform) monitorBalance() {
	ticker := time.NewTicker(p.config.SafeStop.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if !p.isRunning {
				return
			}
			
			ctx := context.Background()
			currentBalance, err := p.getCurrentBalance(ctx)
			if err != nil {
				logrus.WithError(err).Error("🛡️  Failed to fetch balance for Safe-Stop")
				continue
			}
			
			// Check percentage drop
			balanceDropPercent := ((p.initialBalance - currentBalance) / p.initialBalance) * 100
			
			// Check absolute minimum balance
			if currentBalance < p.config.SafeStop.MinBalanceUSD {
				logrus.WithFields(logrus.Fields{
					"current_balance": currentBalance,
					"min_balance":     p.config.SafeStop.MinBalanceUSD,
				}).Error("🛡️  SAFE-STOP TRIGGERED: Balance below minimum threshold")
				p.triggerSafeStop("minimum balance threshold")
				return
			}
			
			// Check percentage drop threshold
			if balanceDropPercent > p.config.SafeStop.ThresholdPercent {
				logrus.WithFields(logrus.Fields{
					"initial_balance":      p.initialBalance,
					"current_balance":      currentBalance,
					"drop_percent":         balanceDropPercent,
					"threshold_percent":    p.config.SafeStop.ThresholdPercent,
				}).Error("🛡️  SAFE-STOP TRIGGERED: Balance drop exceeded threshold")
				p.triggerSafeStop(fmt.Sprintf("%.1f%% balance drop", balanceDropPercent))
				return
			}
			
			// Log normal status
			if balanceDropPercent > 0 {
				logrus.WithFields(logrus.Fields{
					"current_balance":   currentBalance,
					"drop_percent":      balanceDropPercent,
					"threshold_percent": p.config.SafeStop.ThresholdPercent,
				}).Info("🛡️  Safe-Stop monitoring active")
			}
			
		case <-p.stopChan:
			logrus.Info("🛡️  Safe-Stop monitor stopped")
			return
		}
	}
}

// getCurrentBalance fetches current futures wallet balance
func (p *Platform) getCurrentBalance(ctx context.Context) (float64, error) {
	account, err := p.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch account: %w", err)
	}
	
	return parseFloatSafe(account.TotalWalletBalance), nil
}

// triggerSafeStop initiates emergency shutdown
func (p *Platform) triggerSafeStop(reason string) {
	logrus.WithField("reason", reason).Error("🛡️  EMERGENCY SAFE-STOP ACTIVATED")
	
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Stop all trading activities
	if err := p.Stop(ctx); err != nil {
		logrus.WithError(err).Error("🛡️  Error during Safe-Stop shutdown")
	}
	
	logrus.Info("🛡️  Safe-Stop completed - platform halted for protection")
	os.Exit(1) // Force exit to prevent any further operations
}

// initWAL initializes the Write-Ahead Log
func (p *Platform) initWAL() error {
	logrus.Info("💾 Initializing Write-Ahead Log...")
	
	wal, err := platform.NewWAL("trade.wal")
	if err != nil {
		return fmt.Errorf("failed to create WAL: %w", err)
	}
	
	p.wal = wal
	logrus.Info("✅ WAL initialized")
	return nil
}

// initReconciler initializes the ghost position reconciler
func (p *Platform) initReconciler() error {
	logrus.Info("🔍 Initializing ghost position reconciler...")
	
	if p.wal == nil {
		return fmt.Errorf("WAL not initialized, cannot create reconciler")
	}
	
	p.reconciler = agent.NewReconciler(p.client, p.wal, p.stateManager)
	logrus.Info("✅ Reconciler initialized")
	return nil
}

// softReconciliationLoop runs periodic soft reconciliation
func (p *Platform) softReconciliationLoop() {
	ticker := time.NewTicker(60 * time.Minute) // Every 60 minutes
	defer ticker.Stop()
	
	for p.isRunning {
		select {
		case <-ticker.C:
			if p.reconciler != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := p.reconciler.SoftReconcile(ctx); err != nil {
					logrus.WithError(err).Error("Soft reconciliation failed")
				}
				cancel()
			}
		case <-p.stopChan:
			return
		}
	}
}

// parseFloatSafe safely parses float with default 0
func parseFloatSafe(s string) float64 {
	if s == "" {
		return 0
	}
	var val float64
	fmt.Sscanf(s, "%f", &val)
	return val
}