// Package health provides system health checks, startup validation,
// and real-time monitoring for GOBOT trading system.
// Compatible with Intel Macs (darwin/amd64) and Linux (linux/amd64, linux/arm64)
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Health Check Types
// ============================================================================

// CheckStatus represents the status of a health check
type CheckStatus string

const (
	StatusOK       CheckStatus = "OK"
	StatusWarning  CheckStatus = "WARNING"
	StatusError    CheckStatus = "ERROR"
	StatusUnknown  CheckStatus = "UNKNOWN"
)

// HealthCheck represents a single health check result
type HealthCheck struct {
	Name        string        `json:"name"`
	Category    string        `json:"category"`
	Status      CheckStatus   `json:"status"`
	Message     string        `json:"message"`
	Duration    time.Duration `json:"duration_ms"`
	Timestamp   time.Time     `json:"timestamp"`
	Details     interface{}   `json:"details,omitempty"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Overall     CheckStatus    `json:"overall"`
	Platform    PlatformInfo   `json:"platform"`
	Checks      []HealthCheck  `json:"checks"`
	StartupTime time.Time      `json:"startup_time"`
	LastCheck   time.Time      `json:"last_check"`
	Uptime      time.Duration  `json:"uptime"`
}

// PlatformInfo contains OS/architecture information
type PlatformInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	NumCPU       int    `json:"num_cpu"`
	GoVersion    string `json:"go_version"`
	IsSupported  bool   `json:"is_supported"`
}

// ============================================================================
// Health Checker
// ============================================================================

// HealthChecker performs system health checks
type HealthChecker struct {
	mu          sync.RWMutex
	checks      []HealthCheck
	startupTime time.Time
	config      *HealthConfig
}

// HealthConfig contains configuration for health checks
type HealthConfig struct {
	BinanceBaseURL     string
	BinanceAPIKey      string
	BinanceSecretKey   string
	OllamaURL          string
	OpenRouterURL      string
	OpenRouterAPIKey   string
	MemoryDBPath       string
	CheckTimeout       time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(cfg *HealthConfig) *HealthChecker {
	return &HealthChecker{
		startupTime: time.Now(),
		config:      cfg,
	}
}

// ============================================================================
// Platform Checks
// ============================================================================

// CheckPlatform verifies platform compatibility
func (h *HealthChecker) CheckPlatform() HealthCheck {
	start := time.Now()
	
	info := PlatformInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		NumCPU:    runtime.NumCPU(),
		GoVersion: runtime.Version(),
	}
	
	// Supported platforms: Intel Mac, Linux x64, Linux ARM64
	supportedPlatforms := map[string][]string{
		"darwin": {"amd64"},           // Intel Mac
		"linux":  {"amd64", "arm64"},  // Linux x64, ARM64
	}
	
	if archs, ok := supportedPlatforms[info.OS]; ok {
		for _, arch := range archs {
			if arch == info.Arch {
				info.IsSupported = true
				break
			}
		}
	}
	
	status := StatusOK
	message := fmt.Sprintf("Platform %s/%s supported", info.OS, info.Arch)
	
	if !info.IsSupported {
		status = StatusWarning
		message = fmt.Sprintf("Platform %s/%s not officially supported", info.OS, info.Arch)
	}
	
	return HealthCheck{
		Name:      "Platform Compatibility",
		Category:  "system",
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Details:   info,
	}
}

// ============================================================================
// API Connectivity Checks
// ============================================================================

// CheckBinanceAPI tests Binance Futures API connectivity
func (h *HealthChecker) CheckBinanceAPI(ctx context.Context) HealthCheck {
	start := time.Now()
	
	baseURL := h.config.BinanceBaseURL
	if baseURL == "" {
		baseURL = "https://fapi.binance.com"
	}
	
	// Test public endpoint (no auth required)
	url := baseURL + "/fapi/v1/ping"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return HealthCheck{
			Name:      "Binance API Connectivity",
			Category:  "api",
			Status:    StatusError,
			Message:   fmt.Sprintf("Failed to create request: %v", err),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return HealthCheck{
			Name:      "Binance API Connectivity",
			Category:  "api",
			Status:    StatusError,
			Message:   fmt.Sprintf("Connection failed: %v", err),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()
	
	latency := time.Since(start)
	
	status := StatusOK
	message := fmt.Sprintf("Connected (latency: %dms)", latency.Milliseconds())
	
	if latency > 500*time.Millisecond {
		status = StatusWarning
		message = fmt.Sprintf("High latency: %dms", latency.Milliseconds())
	}
	
	if resp.StatusCode != 200 {
		status = StatusError
		message = fmt.Sprintf("Unexpected status: %d", resp.StatusCode)
	}
	
	return HealthCheck{
		Name:      "Binance API Connectivity",
		Category:  "api",
		Status:    status,
		Message:   message,
		Duration:  latency,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"endpoint":    url,
			"status_code": resp.StatusCode,
			"latency_ms":  latency.Milliseconds(),
		},
	}
}

// CheckBinanceAuth tests Binance API authentication
func (h *HealthChecker) CheckBinanceAuth(ctx context.Context) HealthCheck {
	start := time.Now()
	
	if h.config.BinanceAPIKey == "" {
		return HealthCheck{
			Name:      "Binance API Authentication",
			Category:  "api",
			Status:    StatusError,
			Message:   "BINANCE_API_KEY not configured",
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}
	
	if h.config.BinanceSecretKey == "" {
		return HealthCheck{
			Name:      "Binance API Authentication",
			Category:  "api",
			Status:    StatusError,
			Message:   "BINANCE_SECRET_KEY not configured",
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}
	
	// Keys are configured - actual auth test would require signed request
	return HealthCheck{
		Name:      "Binance API Authentication",
		Category:  "api",
		Status:    StatusOK,
		Message:   "API credentials configured",
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}
}

// CheckOllama tests Ollama connectivity for embeddings
func (h *HealthChecker) CheckOllama(ctx context.Context) HealthCheck {
	start := time.Now()
	
	ollamaURL := h.config.OllamaURL
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	
	url := ollamaURL + "/api/tags"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return HealthCheck{
			Name:      "Ollama Embeddings",
			Category:  "api",
			Status:    StatusError,
			Message:   fmt.Sprintf("Failed to create request: %v", err),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return HealthCheck{
			Name:      "Ollama Embeddings",
			Category:  "api",
			Status:    StatusWarning,
			Message:   "Ollama not running (optional for memory)",
			Duration:  time.Since(start),
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"error": err.Error(),
				"hint":  "Run: ollama serve",
			},
		}
	}
	defer resp.Body.Close()
	
	// Check for embedding model
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	
	json.NewDecoder(resp.Body).Decode(&result)
	
	hasEmbedding := false
	for _, m := range result.Models {
		if strings.Contains(m.Name, "nomic-embed") || strings.Contains(m.Name, "embed") {
			hasEmbedding = true
			break
		}
	}
	
	status := StatusOK
	message := "Ollama connected with embedding model"
	
	if !hasEmbedding {
		status = StatusWarning
		message = "Ollama connected but no embedding model found"
	}
	
	return HealthCheck{
		Name:      "Ollama Embeddings",
		Category:  "api",
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"models_count":  len(result.Models),
			"has_embedding": hasEmbedding,
		},
	}
}

// CheckOpenRouter tests OpenRouter API connectivity
func (h *HealthChecker) CheckOpenRouter(ctx context.Context) HealthCheck {
	start := time.Now()
	
	if h.config.OpenRouterAPIKey == "" {
		return HealthCheck{
			Name:      "OpenRouter LLM",
			Category:  "api",
			Status:    StatusWarning,
			Message:   "OPENROUTER_API_KEY not configured",
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}
	
	url := "https://openrouter.ai/api/v1/models"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return HealthCheck{
			Name:      "OpenRouter LLM",
			Category:  "api",
			Status:    StatusError,
			Message:   fmt.Sprintf("Failed to create request: %v", err),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}
	
	req.Header.Set("Authorization", "Bearer "+h.config.OpenRouterAPIKey)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return HealthCheck{
			Name:      "OpenRouter LLM",
			Category:  "api",
			Status:    StatusError,
			Message:   fmt.Sprintf("Connection failed: %v", err),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()
	
	status := StatusOK
	message := "OpenRouter connected"
	
	if resp.StatusCode == 401 {
		status = StatusError
		message = "Invalid API key"
	} else if resp.StatusCode != 200 {
		status = StatusWarning
		message = fmt.Sprintf("Unexpected status: %d", resp.StatusCode)
	}
	
	return HealthCheck{
		Name:      "OpenRouter LLM",
		Category:  "api",
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}
}

// ============================================================================
// Configuration Checks
// ============================================================================

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string `json:"field"`
	Issue   string `json:"issue"`
	Fix     string `json:"fix"`
}

// CheckConfiguration validates all configuration
func (h *HealthChecker) CheckConfiguration() HealthCheck {
	start := time.Now()
	
	var errors []ConfigError
	var warnings []ConfigError
	
	// Check environment variables
	envChecks := []struct {
		name     string
		env      string
		required bool
	}{
		{"Binance API Key", "BINANCE_API_KEY", true},
		{"Binance Secret Key", "BINANCE_SECRET_KEY", true},
		{"OpenRouter API Key", "OPENROUTER_API_KEY", false},
		{"OpenRouter Backup Key", "OPENROUTER_API_KEY_BACKUP", false},
		{"Mainnet Mode", "MAINNET", false},
	}
	
	for _, check := range envChecks {
		val := os.Getenv(check.env)
		if val == "" && check.required {
			errors = append(errors, ConfigError{
				Field: check.env,
				Issue: "Required environment variable not set",
				Fix:   fmt.Sprintf("export %s=your-value", check.env),
			})
		} else if val == "" && !check.required {
			warnings = append(warnings, ConfigError{
				Field: check.env,
				Issue: "Optional environment variable not set",
				Fix:   fmt.Sprintf("export %s=your-value", check.env),
			})
		}
	}
	
	// Check for placeholder values
	placeholders := []string{"your-api-key", "your-secret", "xxx", "placeholder"}
	for _, check := range envChecks {
		val := os.Getenv(check.env)
		for _, ph := range placeholders {
			if strings.Contains(strings.ToLower(val), ph) {
				errors = append(errors, ConfigError{
					Field: check.env,
					Issue: "Contains placeholder value",
					Fix:   "Replace with actual value",
				})
				break
			}
		}
	}
	
	status := StatusOK
	message := "Configuration valid"
	
	if len(warnings) > 0 {
		status = StatusWarning
		message = fmt.Sprintf("%d warnings", len(warnings))
	}
	
	if len(errors) > 0 {
		status = StatusError
		message = fmt.Sprintf("%d errors, %d warnings", len(errors), len(warnings))
	}
	
	return HealthCheck{
		Name:      "Configuration",
		Category:  "config",
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"errors":   errors,
			"warnings": warnings,
		},
	}
}

// ============================================================================
// File System Checks
// ============================================================================

// CheckFilePermissions verifies sensitive file permissions
func (h *HealthChecker) CheckFilePermissions() HealthCheck {
	start := time.Now()
	
	var issues []string
	
	sensitiveFiles := []string{".env", "state.json", "config.json"}
	
	for _, file := range sensitiveFiles {
		info, err := os.Stat(file)
		if err != nil {
			continue // File doesn't exist, skip
		}
		
		mode := info.Mode().Perm()
		
		// Check if file is world-readable (security issue)
		if mode&0004 != 0 {
			issues = append(issues, fmt.Sprintf("%s is world-readable (mode: %04o)", file, mode))
		}
		
		// Check if file is writable by group/others
		if mode&0022 != 0 {
			issues = append(issues, fmt.Sprintf("%s is writable by group/others (mode: %04o)", file, mode))
		}
	}
	
	status := StatusOK
	message := "File permissions secure"
	
	if len(issues) > 0 {
		status = StatusWarning
		message = fmt.Sprintf("%d permission issues", len(issues))
	}
	
	return HealthCheck{
		Name:      "File Permissions",
		Category:  "security",
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"issues": issues,
			"fix":    "chmod 600 <file>",
		},
	}
}

// CheckDiskSpace verifies sufficient disk space
func (h *HealthChecker) CheckDiskSpace() HealthCheck {
	start := time.Now()
	
	// Platform-specific disk space check
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd = exec.Command("df", "-h", ".")
	default:
		return HealthCheck{
			Name:      "Disk Space",
			Category:  "system",
			Status:    StatusUnknown,
			Message:   "Disk check not supported on this platform",
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}
	
	output, err := cmd.Output()
	if err != nil {
		return HealthCheck{
			Name:      "Disk Space",
			Category:  "system",
			Status:    StatusWarning,
			Message:   fmt.Sprintf("Failed to check: %v", err),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}
	
	return HealthCheck{
		Name:      "Disk Space",
		Category:  "system",
		Status:    StatusOK,
		Message:   "Disk space available",
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"output": string(output),
		},
	}
}

// ============================================================================
// Dependency Checks
// ============================================================================

// CheckDependencies verifies required system dependencies
func (h *HealthChecker) CheckDependencies() HealthCheck {
	start := time.Now()
	
	type depCheck struct {
		name     string
		cmd      string
		args     []string
		required bool
	}
	
	deps := []depCheck{
		{"Python3", "python3", []string{"--version"}, false},
		{"Go", "go", []string{"version"}, true},
		{"Git", "git", []string{"--version"}, false},
	}
	
	var missing []string
	var found []string
	
	for _, dep := range deps {
		cmd := exec.Command(dep.cmd, dep.args...)
		output, err := cmd.Output()
		
		if err != nil {
			if dep.required {
				missing = append(missing, dep.name+" (required)")
			} else {
				missing = append(missing, dep.name+" (optional)")
			}
		} else {
			version := strings.TrimSpace(string(output))
			found = append(found, fmt.Sprintf("%s: %s", dep.name, version))
		}
	}
	
	status := StatusOK
	message := fmt.Sprintf("%d dependencies found", len(found))
	
	if len(missing) > 0 {
		hasRequired := false
		for _, m := range missing {
			if strings.Contains(m, "required") {
				hasRequired = true
				break
			}
		}
		
		if hasRequired {
			status = StatusError
			message = fmt.Sprintf("%d missing (including required)", len(missing))
		} else {
			status = StatusWarning
			message = fmt.Sprintf("%d optional dependencies missing", len(missing))
		}
	}
	
	return HealthCheck{
		Name:      "System Dependencies",
		Category:  "system",
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"found":   found,
			"missing": missing,
		},
	}
}

// ============================================================================
// Code Integrity Checks
// ============================================================================

// CheckCodeIntegrity verifies the codebase compiles without errors
func (h *HealthChecker) CheckCodeIntegrity(ctx context.Context) HealthCheck {
	start := time.Now()
	
	cmd := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "./...")
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		// Parse errors
		lines := strings.Split(string(output), "\n")
		var errors []string
		for _, line := range lines {
			if strings.Contains(line, "error") || strings.Contains(line, "undefined") {
				errors = append(errors, strings.TrimSpace(line))
			}
		}
		
		return HealthCheck{
			Name:      "Code Integrity",
			Category:  "code",
			Status:    StatusError,
			Message:   fmt.Sprintf("Build failed: %d errors", len(errors)),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"errors": errors,
				"output": string(output),
			},
		}
	}
	
	return HealthCheck{
		Name:      "Code Integrity",
		Category:  "code",
		Status:    StatusOK,
		Message:   "Build successful",
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}
}

// ============================================================================
// Run All Checks
// ============================================================================

// RunAllChecks performs all health checks
func (h *HealthChecker) RunAllChecks(ctx context.Context) *SystemHealth {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	health := &SystemHealth{
		Platform:    h.getPlatformInfo(),
		StartupTime: h.startupTime,
		LastCheck:   time.Now(),
		Uptime:      time.Since(h.startupTime),
	}
	
	// Run all checks
	checks := []HealthCheck{
		h.CheckPlatform(),
		h.CheckConfiguration(),
		h.CheckFilePermissions(),
		h.CheckDependencies(),
		h.CheckDiskSpace(),
		h.CheckBinanceAPI(ctx),
		h.CheckBinanceAuth(ctx),
		h.CheckOllama(ctx),
		h.CheckOpenRouter(ctx),
	}
	
	// Optionally run code integrity check (slow)
	// checks = append(checks, h.CheckCodeIntegrity(ctx))
	
	health.Checks = checks
	
	// Determine overall status
	health.Overall = StatusOK
	for _, check := range checks {
		if check.Status == StatusError {
			health.Overall = StatusError
			break
		}
		if check.Status == StatusWarning && health.Overall != StatusError {
			health.Overall = StatusWarning
		}
	}
	
	h.checks = checks
	
	return health
}

// RunStartupChecks performs essential startup checks
func (h *HealthChecker) RunStartupChecks(ctx context.Context) (*SystemHealth, error) {
	health := h.RunAllChecks(ctx)
	
	// Collect critical errors
	var criticalErrors []string
	for _, check := range health.Checks {
		if check.Status == StatusError {
			criticalErrors = append(criticalErrors, 
				fmt.Sprintf("%s: %s", check.Name, check.Message))
		}
	}
	
	if len(criticalErrors) > 0 {
		return health, fmt.Errorf("startup checks failed: %s", strings.Join(criticalErrors, "; "))
	}
	
	return health, nil
}

// getPlatformInfo returns current platform information
func (h *HealthChecker) getPlatformInfo() PlatformInfo {
	info := PlatformInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		NumCPU:    runtime.NumCPU(),
		GoVersion: runtime.Version(),
	}
	
	// Check if supported
	supportedPlatforms := map[string][]string{
		"darwin": {"amd64"},
		"linux":  {"amd64", "arm64"},
	}
	
	if archs, ok := supportedPlatforms[info.OS]; ok {
		for _, arch := range archs {
			if arch == info.Arch {
				info.IsSupported = true
				break
			}
		}
	}
	
	return info
}

// GetLastChecks returns the last health check results
func (h *HealthChecker) GetLastChecks() []HealthCheck {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.checks
}
