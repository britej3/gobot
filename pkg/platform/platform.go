package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britej3/gobot/internal/agent"
	"github.com/britej3/gobot/internal/platform"
	"github.com/britej3/gobot/internal/position"
	"github.com/britej3/gobot/pkg/brain"
	"github.com/britej3/gobot/services/screener"
	"github.com/sirupsen/logrus"
)

type Platform struct {
	config         *Config
	client         *futures.Client
	brain          *brain.BrainEngine
	feedback       *CogneeFeedbackSystem
	screener       *screener.Screener
	positionMgr    *position.PositionManager
	stateManager   *StateManager
	reconciler     *agent.Reconciler
	wal            *platform.WAL
	isRunning      bool
	stopChan       chan struct{}
	initialBalance float64
}

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
		Enabled          bool          `json:"enabled"`
		ThresholdPercent float64       `json:"threshold_percent"`
		MinBalanceUSD    float64       `json:"min_balance_usd"`
		InitialBalance   float64       `json:"initial_balance"`
		CheckInterval    time.Duration `json:"check_interval"`
	} `json:"safe_stop"`

	Screener struct {
		Enabled        bool     `json:"enabled"`
		Interval       duration `json:"interval"`
		MaxPairs       int      `json:"max_pairs"`
		MinVolume24h   float64  `json:"min_volume_24h"`
		MinPriceChange float64  `json:"min_price_change"`
		IncludeSymbols []string `json:"include_symbols"`
		ExcludeSymbols []string `json:"exclude_symbols"`
	} `json:"screener"`
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

type CogneeFeedbackSystem struct {
	client  *futures.Client
	botName string
	enabled bool
}

func NewCogneeFeedbackSystem(dbPath string, client *futures.Client, botName string) *CogneeFeedbackSystem {
	return &CogneeFeedbackSystem{
		client:  client,
		botName: botName,
		enabled: true,
	}
}

func (s *CogneeFeedbackSystem) Start() error {
	logrus.WithField("bot_name", s.botName).Info("GOBOT feedback system started")
	return nil
}

func (s *CogneeFeedbackSystem) Stop() error {
	logrus.WithField("bot_name", s.botName).Info("GOBOT feedback system stopped")
	return nil
}

func NewPlatform() *Platform {
	config := loadConfig()
	return &Platform{
		config: config,
	}
}

func (p *Platform) Start() error {
	logrus.Info("Starting GOBOT platform with Meme Coin Screener...")

	p.stopChan = make(chan struct{})

	if err := p.initBinanceClient(); err != nil {
		return fmt.Errorf("failed to initialize Binance client: %w", err)
	}

	if err := p.initWAL(); err != nil {
		logrus.WithError(err).Warn("Failed to initialize WAL, continuing without it")
	}

	sessionID := fmt.Sprintf("%d", time.Now().Unix())
	p.stateManager = NewStateManager(sessionID)

	if prevState, err := p.stateManager.Load(); err != nil {
		logrus.WithError(err).Warn("Failed to load previous state, starting fresh")
	} else if prevState != nil {
		logrus.WithFields(logrus.Fields{
			"positions": len(prevState.OpenPositions),
			"balance":   prevState.TotalBalance,
		}).Info("Previous state restored")
	}

	if err := p.initReconciler(); err != nil {
		logrus.WithError(err).Warn("Failed to initialize reconciler, continuing without it")
	}

	if p.reconciler != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logrus.Info("Running startup reconciliation for ghost positions...")
		if err := p.reconciler.Reconcile(ctx); err != nil {
			logrus.WithError(err).Error("Startup reconciliation failed")
		}
	}

	p.stateManager.StartAutoSave()

	if err := p.initFeedbackSystem(); err != nil {
		return fmt.Errorf("failed to initialize feedback system: %w", err)
	}

	if err := p.initBrainEngine(); err != nil {
		return fmt.Errorf("failed to initialize brain engine: %w", err)
	}

	if err := p.initScreener(); err != nil {
		return fmt.Errorf("failed to initialize screener: %w", err)
	}

	if err := p.initPositionManager(); err != nil {
		logrus.WithError(err).Warn("Failed to initialize position manager, continuing without it")
	}

	p.startSafeStopMonitor()

	if err := p.startComponents(); err != nil {
		return fmt.Errorf("failed to start components: %w", err)
	}

	p.isRunning = true
	logrus.Info("GOBOT platform started successfully")

	go p.runBackgroundTasks()

	return nil
}

func (p *Platform) Stop(ctx context.Context) error {
	logrus.Info("Stopping GOBOT platform...")

	p.isRunning = false

	if p.stopChan != nil {
		close(p.stopChan)
	}

	if p.screener != nil {
		p.screener.Stop()
	}

	if p.brain != nil {
		if err := p.brain.Stop(); err != nil {
			logrus.WithError(err).Error("Failed to stop brain engine")
		}
	}

	if p.feedback != nil {
		if err := p.feedback.Stop(); err != nil {
			logrus.WithError(err).Error("Failed to stop feedback system")
		}
	}

	if p.positionMgr != nil {
		if err := p.positionMgr.Stop(); err != nil {
			logrus.WithError(err).Error("Failed to stop position manager")
		}
	}

	logrus.Info("GOBOT platform stopped")
	return nil
}

func (p *Platform) initBinanceClient() error {
	logrus.Info("Initializing Binance client...")

	apiKey := p.config.Binance.APIKey
	apiSecret := p.config.Binance.APISecret

	if p.config.Binance.Testnet {
		testnetKey := os.Getenv("BINANCE_TESTNET_API")
		testnetSecret := os.Getenv("BINANCE_TESTNET_SECRET")

		if testnetKey != "" && testnetSecret != "" {
			apiKey = testnetKey
			apiSecret = testnetSecret
			logrus.Info("Using Binance testnet credentials")
		}

		p.client = futures.NewClient(apiKey, apiSecret)
		p.client.BaseURL = "https://testnet.binancefuture.com"
		logrus.Info("Using Binance testnet URL")
	} else {
		p.client = futures.NewClient(apiKey, apiSecret)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := p.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	logrus.Info("Binance client initialized")
	return nil
}

func (p *Platform) initFeedbackSystem() error {
	if !p.config.Feedback.Enabled {
		logrus.Info("Feedback system disabled - skipping initialization")
		return nil
	}

	logrus.Info("Initializing feedback system...")

	system := NewCogneeFeedbackSystem(
		p.config.Feedback.DBPath,
		p.client,
		"GOBOT-SCREENER-V1",
	)

	p.feedback = system
	logrus.Info("Feedback system initialized")
	return nil
}

func (p *Platform) initBrainEngine() error {
	logrus.Info("Initializing brain engine...")

	engine, err := brain.NewBrainEngine(p.client, p.feedback, p.config.Brain)
	if err != nil {
		return fmt.Errorf("failed to create brain engine: %w", err)
	}

	p.brain = engine
	logrus.Info("Brain engine initialized")
	return nil
}

func (p *Platform) initScreener() error {
	if !p.config.Screener.Enabled {
		logrus.Info("Screener disabled - skipping initialization")
		return nil
	}

	logrus.Info("Initializing meme coin screener...")

	filter := screener.AssetFilter{
		ContractType:   "PERPETUAL",
		QuoteAsset:     "USDT",
		MinVolume24h:   p.config.Screener.MinVolume24h,
		MinPriceChange: p.config.Screener.MinPriceChange,
		Status:         "TRADING",
		IncludeSymbols: p.config.Screener.IncludeSymbols,
		ExcludeSymbols: p.config.Screener.ExcludeSymbols,
	}

	p.screener = screener.NewScreener(nil,
		screener.WithAssetFilter(filter),
		screener.WithInterval(p.config.Screener.Interval.Duration),
		screener.WithMaxPairs(p.config.Screener.MaxPairs),
		screener.WithSortBy("volatility"),
	)

	logrus.Info("Meme coin screener initialized")
	return nil
}

func (p *Platform) GetScreener() *screener.Screener {
	return p.screener
}

func (p *Platform) initPositionManager() error {
	logrus.Info("Initializing position manager...")

	p.positionMgr = position.NewPositionManager(p.client, p.brain)

	logrus.Info("Position manager initialized")
	return nil
}

func (p *Platform) startComponents() error {
	logrus.Info("Starting platform components...")

	if p.feedback != nil {
		if err := p.feedback.Start(); err != nil {
			return fmt.Errorf("failed to start feedback system: %w", err)
		}
	}

	if err := p.brain.Start(); err != nil {
		return fmt.Errorf("failed to start brain engine: %w", err)
	}

	if p.screener != nil {
		logrus.Info("Starting screener...")
	}

	return nil
}

func (p *Platform) runBackgroundTasks() {
	logrus.Info("Starting background tasks...")

	go p.healthMonitoring()
	go p.performanceReporting()
	go p.softReconciliationLoop()
	go p.screenerStatsLoop()
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
	if p.brain != nil {
		stats := p.brain.GetEngineStats()
		logrus.WithFields(logrus.Fields{
			"uptime":    stats["uptime"],
			"decisions": stats["decisions_made"],
			"provider":  stats["provider"].(map[string]interface{})["model"],
			"healthy":   stats["provider"].(map[string]interface{})["healthy"],
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
		"uptime":          stats["uptime"],
		"total_decisions": stats["decisions_made"],
		"recoveries":      stats["recoveries"],
		"provider_model":  stats["provider"].(map[string]interface{})["model"],
	}).Info("Performance report generated")
}

func (p *Platform) screenerStatsLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for p.isRunning {
		select {
		case <-ticker.C:
			p.logScreenerStats()
		case <-p.stopChan:
			return
		}
	}
}

func (p *Platform) logScreenerStats() {
	if p.screener != nil {
		stats := p.screener.Stats()
		pairs := p.screener.GetActivePairs()

		logrus.WithFields(logrus.Fields{
			"total_pairs":  stats.TotalPairs,
			"active_pairs": stats.ActivePairs,
			"avg_volume":   stats.AvgVolume,
			"avg_change":   stats.AvgChange,
			"pairs":        pairs,
		}).Debug("Screener stats")
	}
}

func loadConfig() *Config {
	config := &Config{}

	config.Binance.APIKey = os.Getenv("BINANCE_API_KEY")
	config.Binance.APISecret = os.Getenv("BINANCE_API_SECRET")
	config.Binance.Testnet = getEnvBool("BINANCE_USE_TESTNET", true)

	config.Brain = brain.DefaultBrainConfig()
	config.Brain.InferenceMode = getEnvString("INFERENCE_MODE", "CLOUD")
	config.Brain.LocalModel = getEnvString("OLLAMA_MODEL", "qwen3:0.6b")
	config.Brain.LocalBaseURL = getEnvString("OLLAMA_BASE_URL", "http://localhost:11964")
	config.Brain.CloudAPIKey = os.Getenv("GEMINI_API_KEY")
	config.Brain.CloudProvider = getEnvString("CLOUD_PROVIDER", "gemini")
	config.Brain.EnableRecovery = getEnvBool("ENABLE_RECOVERY", true)

	config.Feedback.Enabled = getEnvBool("FEEDBACK_ENABLED", true)
	config.Feedback.DBPath = getEnvString("FEEDBACK_DB_PATH", "gobot_production.db")

	config.SafeStop.Enabled = getEnvBool("SAFE_STOP_ENABLED", true)
	config.SafeStop.ThresholdPercent = getEnvFloat("SAFE_STOP_THRESHOLD_PERCENT", 10.0)
	config.SafeStop.MinBalanceUSD = getEnvFloat("SAFE_STOP_MIN_BALANCE_USD", 100.0)
	config.SafeStop.CheckInterval = time.Duration(getEnvInt("SAFE_STOP_CHECK_INTERVAL", 300)) * time.Second

	config.Screener.Enabled = getEnvBool("SCREENER_ENABLED", true)
	config.Screener.Interval.Duration = time.Duration(getEnvInt("SCREENER_INTERVAL_SECONDS", 300)) * time.Second
	config.Screener.MaxPairs = getEnvInt("SCREENER_MAX_PAIRS", 5)
	config.Screener.MinVolume24h = getEnvFloat("SCREENER_MIN_VOLUME_24H", 5000000)
	config.Screener.MinPriceChange = getEnvFloat("SCREENER_MIN_PRICE_CHANGE", 5.0)
	config.Screener.IncludeSymbols = getEnvSlice("SCREENER_INCLUDE_SYMBOLS")
	config.Screener.ExcludeSymbols = getEnvSlice("SCREENER_EXCLUDE_SYMBOLS")

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

func getEnvSlice(key string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return nil
}

func (p *Platform) startSafeStopMonitor() {
	if !p.config.SafeStop.Enabled {
		logrus.Info("Safe-Stop protection disabled")
		return
	}

	logrus.WithFields(logrus.Fields{
		"threshold_percent": p.config.SafeStop.ThresholdPercent,
		"min_balance_usd":   p.config.SafeStop.MinBalanceUSD,
		"check_interval":    p.config.SafeStop.CheckInterval,
	}).Info("Starting Safe-Stop balance monitor")

	ctx := context.Background()
	initialBalance, err := p.getCurrentBalance(ctx)
	if err != nil {
		logrus.WithError(err).Warn("Could not fetch initial balance for Safe-Stop")
		return
	}

	p.initialBalance = initialBalance
	p.config.SafeStop.InitialBalance = initialBalance

	logrus.WithField("initial_balance", initialBalance).Info("Safe-Stop baseline established")

	go p.monitorBalance()
}

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
				logrus.WithError(err).Error("Failed to fetch balance for Safe-Stop")
				continue
			}

			balanceDropPercent := ((p.initialBalance - currentBalance) / p.initialBalance) * 100

			if currentBalance < p.config.SafeStop.MinBalanceUSD {
				logrus.WithFields(logrus.Fields{
					"current_balance": currentBalance,
					"min_balance":     p.config.SafeStop.MinBalanceUSD,
				}).Error("SAFE-STOP TRIGGERED: Balance below minimum threshold")
				p.triggerSafeStop("minimum balance threshold")
				return
			}

			if balanceDropPercent > p.config.SafeStop.ThresholdPercent {
				logrus.WithFields(logrus.Fields{
					"initial_balance":   p.initialBalance,
					"current_balance":   currentBalance,
					"drop_percent":      balanceDropPercent,
					"threshold_percent": p.config.SafeStop.ThresholdPercent,
				}).Error("SAFE-STOP TRIGGERED: Balance drop exceeded threshold")
				p.triggerSafeStop(fmt.Sprintf("%.1f%% balance drop", balanceDropPercent))
				return
			}

			if balanceDropPercent > 0 {
				logrus.WithFields(logrus.Fields{
					"current_balance":   currentBalance,
					"drop_percent":      balanceDropPercent,
					"threshold_percent": p.config.SafeStop.ThresholdPercent,
				}).Info("Safe-Stop monitoring active")
			}

		case <-p.stopChan:
			logrus.Info("Safe-Stop monitor stopped")
			return
		}
	}
}

func (p *Platform) getCurrentBalance(ctx context.Context) (float64, error) {
	account, err := p.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch account: %w", err)
	}

	return parseFloatSafe(account.TotalWalletBalance), nil
}

func (p *Platform) triggerSafeStop(reason string) {
	logrus.WithField("reason", reason).Error("EMERGENCY SAFE-STOP ACTIVATED")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := p.Stop(ctx); err != nil {
		logrus.WithError(err).Error("Error during Safe-Stop shutdown")
	}

	logrus.Info("Safe-Stop completed - platform halted for protection")
	os.Exit(1)
}

func (p *Platform) initWAL() error {
	logrus.Info("Initializing Write-Ahead Log...")

	wal, err := platform.NewWAL("trade.wal")
	if err != nil {
		return fmt.Errorf("failed to create WAL: %w", err)
	}

	p.wal = wal
	logrus.Info("WAL initialized")
	return nil
}

func (p *Platform) initReconciler() error {
	logrus.Info("Initializing ghost position reconciler...")

	if p.wal == nil {
		return fmt.Errorf("WAL not initialized, cannot create reconciler")
	}

	p.reconciler = agent.NewReconciler(p.client, p.wal, p.stateManager)
	logrus.Info("Reconciler initialized")
	return nil
}

func (p *Platform) softReconciliationLoop() {
	ticker := time.NewTicker(60 * time.Minute)
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

func parseFloatSafe(s string) float64 {
	if s == "" {
		return 0
	}
	var val float64
	fmt.Sscanf(s, "%f", &val)
	return val
}
