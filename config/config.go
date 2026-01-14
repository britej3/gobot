package config

import (
	"context"
	"os"
	"time"
)

type Config struct {
	Binance   BinanceConfig
	Scanner   ScannerConfig
	Executor  ExecutorConfig
	Monitor   MonitorConfig
	Scheduler SchedulerConfig
}

type BinanceConfig struct {
	APIKey     string
	APISecret  string
	Testnet    bool
	Timeout    time.Duration
	MaxRetries int
}

func (c BinanceConfig) Endpoint() string {
	if c.Testnet {
		return "https://testnet.binancefuture.com"
	}
	return "https://fapi.binance.com"
}

type ScannerConfig struct {
	Interval      time.Duration
	MinVolume     float64
	MaxAssets     int
	MinConfidence float64
}

type ExecutorConfig struct {
	DefaultSize  float64
	StopLoss     float64
	TakeProfit   float64
	MaxPositions int
}

type MonitorConfig struct {
	CheckInterval   time.Duration
	HealthThreshold float64
}

type SchedulerConfig struct {
	Workers   int
	QueueSize int
}

func Load(ctx context.Context) (*Config, error) {
	cfg := &Config{
		Binance: BinanceConfig{
			APIKey:     os.Getenv("BINANCE_API_KEY"),
			APISecret:  os.Getenv("BINANCE_API_SECRET"),
			Testnet:    os.Getenv("BINANCE_USE_TESTNET") == "true",
			Timeout:    10 * time.Second,
			MaxRetries: 3,
		},
		Scanner: ScannerConfig{
			Interval:      2 * time.Minute,
			MinVolume:     1000000,
			MaxAssets:     15,
			MinConfidence: 0.65,
		},
		Executor: ExecutorConfig{
			DefaultSize:  0.001,
			StopLoss:     0.005,
			TakeProfit:   0.015,
			MaxPositions: 5,
		},
		Monitor: MonitorConfig{
			CheckInterval:   30 * time.Second,
			HealthThreshold: 45,
		},
		Scheduler: SchedulerConfig{
			Workers:   4,
			QueueSize: 100,
		},
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Binance.APIKey == "" {
		return ErrMissingAPIKey
	}
	if c.Binance.APISecret == "" {
		return ErrMissingAPISecret
	}
	if c.Scanner.MaxAssets <= 0 {
		return ErrInvalidMaxAssets
	}
	if c.Executor.DefaultSize <= 0 {
		return ErrInvalidDefaultSize
	}
	return nil
}

type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}

var (
	ErrMissingAPIKey      = &ConfigError{Field: "BINANCE_API_KEY", Message: "required but not set"}
	ErrMissingAPISecret   = &ConfigError{Field: "BINANCE_API_SECRET", Message: "required but not set"}
	ErrInvalidMaxAssets   = &ConfigError{Field: "Scanner.MaxAssets", Message: "must be positive"}
	ErrInvalidDefaultSize = &ConfigError{Field: "Executor.DefaultSize", Message: "must be positive"}
)
