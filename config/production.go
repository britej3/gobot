package config

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ProductionConfig struct {
	Binance        BinanceAPIConfig     `yaml:"binance"`
	Trading        TradingConfig        `yaml:"trading"`
	Execution      ExecutionConfig      `yaml:"execution"`
	Stealth        StealthConfig        `yaml:"stealth"`
	AI             AIConfig             `yaml:"ai"`
	Watchlist      WatchlistConfig      `yaml:"watchlist"`
	Risk           RiskConfig           `yaml:"risk"`
	Emergency      EmergencyConfig      `yaml:"emergency"`
	Monitoring     MonitoringConfig     `yaml:"monitoring"`
	State          StateConfig          `yaml:"state"`
	Performance    PerformanceConfig    `yaml:"performance"`
	TradingView    TradingViewConfig    `yaml:"tradingview"`
	N8NIntegration N8NConfig            `yaml:"n8n"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
}

type BinanceAPIConfig struct {
	APIKey         string `yaml:"api_key"`
	APISecret      string `yaml:"api_secret"`
	UseTestnet     bool   `yaml:"use_testnet"`
	RateLimitRPS   int    `yaml:"rate_limit_rps"`
	RateLimitBurst int    `yaml:"rate_limit_burst"`
	RecvWindowMS   int    `yaml:"recv_window_ms"`
}

func (c BinanceAPIConfig) Endpoint() string {
	if c.UseTestnet {
		return "https://testnet.binancefuture.com"
	}
	return "https://fapi.binance.com"
}

type TradingConfig struct {
	InitialCapitalUSD   float64 `yaml:"initial_capital_usd"`
	MaxPositionUSD      float64 `yaml:"max_position_usd"`
	DailyTradeLimit     float64 `yaml:"daily_trade_limit"`
	WeeklyLossLimit     float64 `yaml:"weekly_loss_limit"`
	StopLossPercent     float64 `yaml:"stop_loss_percent"`
	TakeProfitPercent   float64 `yaml:"take_profit_percent"`
	TrailingStopEnabled bool    `yaml:"trailing_stop_enabled"`
	TrailingStopPercent float64 `yaml:"trailing_stop_percent"`
	MaxDailyDrawdown    float64 `yaml:"max_daily_drawdown"`
	KellyFraction       float64 `yaml:"kelly_fraction"`
	MaxRiskPerTrade     float64 `yaml:"max_risk_per_trade"`
	TradingIntervalMin  int     `yaml:"trading_interval_minutes"`
	MaxTradesPerDay     int     `yaml:"max_trades_per_day"`
	SymbolCooldownMin   int     `yaml:"symbol_cooldown_minutes"`
	MinConfidence       float64 `yaml:"min_confidence_threshold"`
	MinRiskRewardRatio  float64 `yaml:"min_risk_reward_ratio"`
	MaxSpreadPercent    float64 `yaml:"max_spread_percent"`
	MinVolume24HUSD     float64 `yaml:"min_volume_24h_usd"`
}

type ExecutionConfig struct {
	AutoExecute         bool    `yaml:"auto_execute"`
	MinConfidence       float64 `yaml:"min_confidence"`
	MaxDailyTrades      int     `yaml:"max_daily_trades"`
	RequireTrendConfirm bool    `yaml:"require_trend_confirmation"`
	RequireVolumeSpike  bool    `yaml:"require_volume_spike"`
}

type StealthConfig struct {
	Enabled              bool    `yaml:"enabled"`
	JitterEnabled        bool    `yaml:"jitter_enabled"`
	JitterRangeMS        int     `yaml:"jitter_range_ms"`
	UserAgentRotation    bool    `yaml:"user_agent_rotation"`
	RequestDelayMinMS    int     `yaml:"request_delay_min_ms"`
	RequestDelayMaxMS    int     `yaml:"request_delay_max_ms"`
	APIRateLimitRPS      int     `yaml:"api_rate_limit_rps"`
	SignatureVariance    float64 `yaml:"signature_variance"`
	FingerprintRandomize bool    `yaml:"fingerprint_randomization"`
}

type AIConfig struct {
	Enabled           bool    `yaml:"enabled"`
	Model             string  `yaml:"model"`
	APIKey            string  `yaml:"api_key"`
	VisionMaxTokens   int     `yaml:"vision_max_tokens"`
	VisionTemperature float64 `yaml:"vision_temperature"`
	MaxImageSizeKB    int     `yaml:"max_image_size_kb"`
}

type WatchlistConfig struct {
	Symbols []string `yaml:"symbols"`
}

type RiskConfig struct {
	MaxAPIErrorsPerHour  int     `yaml:"max_api_errors_per_hour"`
	MaxAPIErrorPercent   float64 `yaml:"max_api_error_percent"`
	MaxLatencyMS         int     `yaml:"max_acceptable_latency_ms"`
	DailyLossAlert       float64 `yaml:"daily_loss_alert"`
	ConsecutiveLossAlert int     `yaml:"consecutive_loss_alert"`
	PositionSizeAlert    float64 `yaml:"position_size_alert"`
}

type EmergencyConfig struct {
	KillSwitchEnabled     bool   `yaml:"kill_switch_enabled"`
	KillSwitchPassword    string `yaml:"kill_switch_password"`
	KillSwitchFile        string `yaml:"kill_switch_file"`
	EnableRecovery        bool   `yaml:"enable_recovery"`
	RecoveryMode          string `yaml:"recovery_mode"`
	MaxRecoveryAttempts   int    `yaml:"max_recovery_attempts"`
	RecoveryCooldownHours int    `yaml:"recovery_cooldown_hours"`
}

type MonitoringConfig struct {
	TelegramEnabled     bool   `yaml:"telegram_enabled"`
	TelegramToken       string `yaml:"telegram_token"`
	TelegramChatID      string `yaml:"telegram_chat_id"`
	AlertOnTrade        bool   `yaml:"alert_on_trade"`
	AlertOnPNLMilestone bool   `yaml:"alert_on_pnl_milestone"`
	AlertOnRiskBreach   bool   `yaml:"alert_on_risk_breach"`
	AlertOnSystemError  bool   `yaml:"alert_on_system_error"`
	AuditLogEnabled     bool   `yaml:"audit_log_enabled"`
	AuditLogPath        string `yaml:"audit_log_path"`
	TradeLogPath        string `yaml:"trade_log_path"`
	DetailedTradeLog    bool   `yaml:"detailed_trade_log"`
	LogLevel            string `yaml:"log_level"`
}

type StateConfig struct {
	PersistenceEnabled  bool   `yaml:"persistence_enabled"`
	StateDir            string `yaml:"state_dir"`
	StateFile           string `yaml:"state_file"`
	SaveIntervalSeconds int    `yaml:"save_interval_seconds"`
}

type PerformanceConfig struct {
	MaxMemoryMB           int `yaml:"max_memory_mb"`
	RestartIntervalHours  int `yaml:"restart_interval_hours"`
	CacheKlinesMinutes    int `yaml:"cache_klines_minutes"`
	CachePriceSeconds     int `yaml:"cache_price_seconds"`
	MaxConcurrentRequests int `yaml:"max_concurrent_requests"`
}

type TradingViewConfig struct {
	BaseURL       string   `yaml:"base_url"`
	Intervals     []string `yaml:"intervals"`
	ScreenshotDir string   `yaml:"screenshot_dir"`
}

type N8NIntegrationConfig struct {
	Enabled      bool   `yaml:"enabled"`
	BaseURL      string `yaml:"base_url"`
	WebhookUser  string `yaml:"webhook_user"`
	WebhookPass  string `yaml:"webhook_pass"`
	TradeWebhook string `yaml:"trade_webhook"`
	AlertWebhook string `yaml:"alert_webhook"`
}

type CircuitBreakerConfig struct {
	Enabled              bool `yaml:"enabled"`
	FailureThreshold     int  `yaml:"failure_threshold"`
	FailureWindowSeconds int  `yaml:"failure_window_seconds"`
	RecoveryTimeoutSecs  int  `yaml:"recovery_timeout_seconds"`
	HalfOpenRequests     int  `yaml:"half_open_requests"`
}

func LoadProductionConfig(ctx context.Context, configPath string) (*ProductionConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg ProductionConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg = cfg.applyEnvironmentOverrides()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func (c ProductionConfig) applyEnvironmentOverrides() ProductionConfig {
	if apiKey := os.Getenv("BINANCE_API_KEY"); apiKey != "" {
		c.Binance.APIKey = apiKey
	}
	if apiSecret := os.Getenv("BINANCE_API_SECRET"); apiSecret != "" {
		c.Binance.APISecret = apiSecret
	}
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		c.AI.APIKey = openaiKey
	}
	if tgToken := os.Getenv("TELEGRAM_TOKEN"); tgToken != "" {
		c.Monitoring.TelegramToken = tgToken
	}
	if tgChat := os.Getenv("TELEGRAM_CHAT_ID"); tgChat != "" {
		c.Monitoring.TelegramChatID = tgChat
	}
	if useTestnet := os.Getenv("BINANCE_USE_TESTNET"); useTestnet == "true" {
		c.Binance.UseTestnet = true
	}
	if killSwitch := os.Getenv("KILL_SWITCH_PASSWORD"); killSwitch != "" {
		c.Emergency.KillSwitchPassword = killSwitch
	}
	return c
}

func expandEnvVars(value string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		key := match[2 : len(match)-1]
		if val := os.Getenv(key); val != "" {
			return val
		}
		return match
	})
}

func (c ProductionConfig) Validate() error {
	var errors []string

	if c.Binance.APIKey == "" || strings.HasPrefix(c.Binance.APIKey, "YOUR_") {
		errors = append(errors, "BINANCE_API_KEY must be set and not be a placeholder")
	}
	if c.Binance.APISecret == "" || strings.HasPrefix(c.Binance.APISecret, "YOUR_") {
		errors = append(errors, "BINANCE_API_SECRET must be set and not be a placeholder")
	}
	if c.Trading.InitialCapitalUSD <= 0 {
		errors = append(errors, "trading.initial_capital_usd must be positive")
	}
	if c.Trading.MaxPositionUSD <= 0 {
		errors = append(errors, "trading.max_position_usd must be positive")
	}
	if c.Trading.StopLossPercent <= 0 {
		errors = append(errors, "trading.stop_loss_percent must be positive")
	}
	if c.Trading.TakeProfitPercent <= c.Trading.StopLossPercent {
		errors = append(errors, "trading.take_profit_percent must be greater than stop_loss_percent")
	}
	if c.Trading.MinConfidence < 0 || c.Trading.MinConfidence > 1 {
		errors = append(errors, "trading.min_confidence_threshold must be between 0 and 1")
	}
	if c.Emergency.KillSwitchPassword == "" {
		errors = append(errors, "emergency.kill_switch_password must be set")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}
	return nil
}

func (c ProductionConfig) GetStateFilePath() string {
	return fmt.Sprintf("%s/%s", c.State.StateDir, c.State.StateFile)
}

func (c ProductionConfig) GetKillSwitchFilePath() string {
	return c.Emergency.KillSwitchFile
}

func (c ProductionConfig) GetTradeLogPath() string {
	return c.Monitoring.TradeLogPath
}

func (c ProductionConfig) GetAuditLogPath() string {
	return c.Monitoring.AuditLogPath
}

func (c ProductionConfig) GetScreenshotDir() string {
	return c.TradingView.ScreenshotDir
}

func (c ProductionConfig) ShouldUseTestnet() bool {
	return c.Binance.UseTestnet
}

func (c ProductionConfig) GetMaxPositionSize() float64 {
	return c.Trading.MaxPositionUSD
}

func (c ProductionConfig) GetStopLossPercent() float64 {
	return c.Trading.StopLossPercent / 100.0
}

func (c ProductionConfig) GetTakeProfitPercent() float64 {
	return c.Trading.TakeProfitPercent / 100.0
}

func (c TradingConfig) GetTradingInterval() time.Duration {
	return time.Duration(c.TradingIntervalMin) * time.Minute
}

func (c TradingConfig) GetSymbolCooldown() time.Duration {
	return time.Duration(c.SymbolCooldownMin) * time.Minute
}

func (c CircuitBreakerConfig) GetFailureWindow() time.Duration {
	return time.Duration(c.FailureWindowSeconds) * time.Second
}

func (c CircuitBreakerConfig) GetRecoveryTimeout() time.Duration {
	return time.Duration(c.RecoveryTimeoutSecs) * time.Second
}

func (c StateConfig) GetSaveInterval() time.Duration {
	return time.Duration(c.SaveIntervalSeconds) * time.Second
}

func (c PerformanceConfig) GetRestartInterval() time.Duration {
	return time.Duration(c.RestartIntervalHours) * time.Hour
}

func (c PerformanceConfig) GetCacheKlinesDuration() time.Duration {
	return time.Duration(c.CacheKlinesMinutes) * time.Minute
}

func (c PerformanceConfig) GetCachePriceDuration() time.Duration {
	return time.Duration(c.CachePriceSeconds) * time.Second
}
