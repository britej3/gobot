// Package startup provides preflight checks before bot execution
// Compatible with Intel Macs (darwin/amd64) and Linux
package startup

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/britebrt/cognee/internal/health"
	"github.com/britebrt/cognee/internal/ui"
)

// PreflightResult contains the result of preflight checks
type PreflightResult struct {
	Passed       bool
	CriticalFail bool
	Health       *health.SystemHealth
	Errors       []string
	Warnings     []string
	StartupTime  time.Duration
}

// PreflightConfig contains preflight check configuration
type PreflightConfig struct {
	BinanceAPIKey      string
	BinanceSecretKey   string
	OpenRouterAPIKey   string
	OpenRouterBackupKey string
	OllamaURL          string
	MainnetMode        bool
	SkipCodeCheck      bool
	Timeout            time.Duration
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() *PreflightConfig {
	return &PreflightConfig{
		BinanceAPIKey:       os.Getenv("BINANCE_API_KEY"),
		BinanceSecretKey:    os.Getenv("BINANCE_SECRET_KEY"),
		OpenRouterAPIKey:    os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterBackupKey: os.Getenv("OPENROUTER_API_KEY_BACKUP"),
		OllamaURL:           os.Getenv("OLLAMA_BASE_URL"),
		MainnetMode:         os.Getenv("MAINNET") == "true",
		SkipCodeCheck:       os.Getenv("SKIP_CODE_CHECK") == "true",
		Timeout:             30 * time.Second,
	}
}

// RunPreflight executes all preflight checks
func RunPreflight(ctx context.Context, cfg *PreflightConfig) *PreflightResult {
	start := time.Now()
	result := &PreflightResult{
		Passed: true,
	}

	// Print startup banner
	fmt.Print(ui.RenderStartupBanner())
	fmt.Println("Running preflight checks...")
	fmt.Println(strings.Repeat("─", 60))

	// Create health checker
	healthCfg := &health.HealthConfig{
		BinanceAPIKey:    cfg.BinanceAPIKey,
		BinanceSecretKey: cfg.BinanceSecretKey,
		OpenRouterAPIKey: cfg.OpenRouterAPIKey,
		OllamaURL:        cfg.OllamaURL,
	}

	checker := health.NewHealthChecker(healthCfg)

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	// Run health checks
	systemHealth, err := checker.RunStartupChecks(checkCtx)
	result.Health = systemHealth

	if err != nil {
		result.Passed = false
		result.CriticalFail = true
		result.Errors = append(result.Errors, err.Error())
	}

	// Additional preflight validations
	result.validateConfig(cfg)
	result.validateTradingReadiness(cfg)

	// Display results
	fmt.Print(ui.RenderHealthChecks(systemHealth))

	// Summary
	result.StartupTime = time.Since(start)
	result.printSummary(cfg)

	return result
}

// validateConfig validates configuration values
func (r *PreflightResult) validateConfig(cfg *PreflightConfig) {
	// Check for placeholder values
	placeholders := []string{
		"your-api-key",
		"your-secret",
		"placeholder",
		"xxx",
		"YOUR_",
	}

	checkValue := func(name, value string, required bool) {
		if value == "" {
			if required {
				r.Errors = append(r.Errors, fmt.Sprintf("%s is required but not set", name))
				r.Passed = false
			} else {
				r.Warnings = append(r.Warnings, fmt.Sprintf("%s not set (optional)", name))
			}
			return
		}

		for _, ph := range placeholders {
			if strings.Contains(strings.ToLower(value), strings.ToLower(ph)) {
				r.Errors = append(r.Errors, fmt.Sprintf("%s contains placeholder value", name))
				r.Passed = false
				return
			}
		}
	}

	// Required
	checkValue("BINANCE_API_KEY", cfg.BinanceAPIKey, true)
	checkValue("BINANCE_SECRET_KEY", cfg.BinanceSecretKey, true)

	// Optional
	checkValue("OPENROUTER_API_KEY", cfg.OpenRouterAPIKey, false)
	checkValue("OPENROUTER_API_KEY_BACKUP", cfg.OpenRouterBackupKey, false)

	// Validate API key formats
	if cfg.BinanceAPIKey != "" && len(cfg.BinanceAPIKey) < 20 {
		r.Warnings = append(r.Warnings, "BINANCE_API_KEY seems too short")
	}
}

// validateTradingReadiness checks if ready for trading
func (r *PreflightResult) validateTradingReadiness(cfg *PreflightConfig) {
	// Mainnet warning
	if cfg.MainnetMode {
		r.Warnings = append(r.Warnings, "⚠️  MAINNET MODE ENABLED - Real money at risk!")
	} else {
		fmt.Println("📋 Running in TESTNET mode")
	}

	// Check if we have LLM configured
	if cfg.OpenRouterAPIKey == "" {
		r.Warnings = append(r.Warnings, "No LLM configured - Brain will use fallback logic")
	} else if cfg.OpenRouterBackupKey != "" {
		fmt.Println("✓ LLM configured with backup key for resilience")
	}
}

// printSummary prints the preflight summary
func (r *PreflightResult) printSummary(cfg *PreflightConfig) {
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))

	// Print errors
	if len(r.Errors) > 0 {
		fmt.Printf("\n%s❌ ERRORS (%d):%s\n", "\033[31m", len(r.Errors), "\033[0m")
		for _, err := range r.Errors {
			fmt.Printf("   • %s\n", err)
		}
	}

	// Print warnings
	if len(r.Warnings) > 0 {
		fmt.Printf("\n%s⚠️  WARNINGS (%d):%s\n", "\033[33m", len(r.Warnings), "\033[0m")
		for _, warn := range r.Warnings {
			fmt.Printf("   • %s\n", warn)
		}
	}

	// Final status
	fmt.Println()
	if r.Passed {
		fmt.Printf("%s✅ PREFLIGHT PASSED%s (%.2fs)\n", "\033[32m", "\033[0m", r.StartupTime.Seconds())
		if cfg.MainnetMode {
			fmt.Printf("\n%s🚀 Ready for MAINNET trading%s\n", "\033[1m", "\033[0m")
		} else {
			fmt.Printf("\n🧪 Ready for TESTNET trading\n")
		}
	} else {
		fmt.Printf("%s❌ PREFLIGHT FAILED%s\n", "\033[31m", "\033[0m")
		if r.CriticalFail {
			fmt.Println("\n⛔ Critical errors must be fixed before trading")
		}
	}

	fmt.Println(strings.Repeat("═", 60))
}

// MustPass panics if preflight did not pass
func (r *PreflightResult) MustPass() {
	if !r.Passed {
		fmt.Println("\n⛔ Cannot start: Preflight checks failed")
		os.Exit(1)
	}
}

// ============================================================================
// Quick Checks (for runtime)
// ============================================================================

// QuickHealthCheck performs a fast health check during runtime
func QuickHealthCheck(ctx context.Context, checker *health.HealthChecker) bool {
	// Only check critical services
	binanceCheck := checker.CheckBinanceAPI(ctx)
	
	if binanceCheck.Status == health.StatusError {
		return false
	}
	
	return true
}

// CheckBeforeTrade performs checks before each trade
func CheckBeforeTrade(ctx context.Context, checker *health.HealthChecker) (bool, string) {
	// Quick API check
	binanceCheck := checker.CheckBinanceAPI(ctx)
	if binanceCheck.Status == health.StatusError {
		return false, "Binance API unavailable"
	}
	
	// Check high latency
	if binanceCheck.Duration > 500*time.Millisecond {
		return false, fmt.Sprintf("High API latency: %dms", binanceCheck.Duration.Milliseconds())
	}
	
	return true, ""
}
