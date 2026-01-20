package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthChecker provides health checking functionality
type HealthChecker struct {
	checks  map[string]HealthCheck
	mu      sync.RWMutex
	logger  *logrus.Logger
	server  *http.Server
}

// HealthCheck represents a single health check
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
	Timeout() time.Duration
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    string                   `json:"status"`
	Timestamp time.Time                `json:"timestamp"`
	Checks    map[string]CheckResult   `json:"checks"`
	Uptime    time.Duration            `json:"uptime"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Status    string        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &HealthChecker{
		checks: make(map[string]HealthCheck),
		logger: logger,
	}
}

// RegisterCheck registers a health check
func (hc *HealthChecker) RegisterCheck(check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[check.Name()] = check
	hc.logger.WithFields(logrus.Fields{
		"check": check.Name(),
	}).Info("health_check_registered")
}

// UnregisterCheck unregisters a health check
func (hc *HealthChecker) UnregisterCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.checks, name)
	hc.logger.WithFields(logrus.Fields{
		"check": name,
	}).Info("health_check_unregistered")
}

// CheckHealth performs all health checks and returns the overall status
func (hc *HealthChecker) CheckHealth(ctx context.Context) *HealthStatus {
	hc.mu.RLock()
	checks := make(map[string]HealthCheck, len(hc.checks))
	for k, v := range hc.checks {
		checks[k] = v
	}
	hc.mu.RUnlock()

	results := make(map[string]CheckResult)
	overallStatus := "healthy"

	for name, check := range checks {
		checkCtx, cancel := context.WithTimeout(ctx, check.Timeout())
		
		startTime := time.Now()
		err := check.Check(checkCtx)
		duration := time.Since(startTime)
		cancel()

		result := CheckResult{
			Status:    "healthy",
			Duration:  duration,
			Timestamp: time.Now(),
		}

		if err != nil {
			result.Status = "unhealthy"
			result.Message = err.Error()
			overallStatus = "unhealthy"

			hc.logger.WithFields(logrus.Fields{
				"check":    name,
				"error":    err.Error(),
				"duration": duration,
			}).Warn("health_check_failed")
		}

		results[name] = result
	}

	return &HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    results,
		Uptime:    time.Since(startupTime),
	}
}

// StartHTTPServer starts an HTTP server for health checks
func (hc *HealthChecker) StartHTTPServer(addr string) error {
	mux := http.NewServeMux()

	// Liveness probe
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Readiness probe
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		status := hc.CheckHealth(ctx)
		
		w.Header().Set("Content-Type", "application/json")
		
		if status.Status == "healthy" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(status)
	})

	// Detailed health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		status := hc.CheckHealth(ctx)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(status)
	})

	hc.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	hc.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Info("health_check_server_starting")

	return hc.server.ListenAndServe()
}

// Stop stops the HTTP server
func (hc *HealthChecker) Stop(ctx context.Context) error {
	if hc.server != nil {
		return hc.server.Shutdown(ctx)
	}
	return nil
}

// Built-in health checks

var startupTime = time.Now()

// SimpleHealthCheck is a basic health check implementation
type SimpleHealthCheck struct {
	name    string
	timeout time.Duration
	checkFn func(context.Context) error
}

// NewSimpleHealthCheck creates a new simple health check
func NewSimpleHealthCheck(name string, timeout time.Duration, checkFn func(context.Context) error) HealthCheck {
	return &SimpleHealthCheck{
		name:    name,
		timeout: timeout,
		checkFn: checkFn,
	}
}

func (shc *SimpleHealthCheck) Name() string {
	return shc.name
}

func (shc *SimpleHealthCheck) Check(ctx context.Context) error {
	return shc.checkFn(ctx)
}

func (shc *SimpleHealthCheck) Timeout() time.Duration {
	return shc.timeout
}

// RedisHealthCheck checks Redis connectivity
type RedisHealthCheck struct {
	name    string
	timeout time.Duration
	pingFn  func(context.Context) error
}

// NewRedisHealthCheck creates a Redis health check
func NewRedisHealthCheck(pingFn func(context.Context) error) HealthCheck {
	return &RedisHealthCheck{
		name:    "redis",
		timeout: 5 * time.Second,
		pingFn:  pingFn,
	}
}

func (rhc *RedisHealthCheck) Name() string {
	return rhc.name
}

func (rhc *RedisHealthCheck) Check(ctx context.Context) error {
	return rhc.pingFn(ctx)
}

func (rhc *RedisHealthCheck) Timeout() time.Duration {
	return rhc.timeout
}

// BinanceHealthCheck checks Binance API connectivity
type BinanceHealthCheck struct {
	name    string
	timeout time.Duration
	pingFn  func(context.Context) error
}

// NewBinanceHealthCheck creates a Binance health check
func NewBinanceHealthCheck(pingFn func(context.Context) error) HealthCheck {
	return &BinanceHealthCheck{
		name:    "binance",
		timeout: 10 * time.Second,
		pingFn:  pingFn,
	}
}

func (bhc *BinanceHealthCheck) Name() string {
	return bhc.name
}

func (bhc *BinanceHealthCheck) Check(ctx context.Context) error {
	return bhc.pingFn(ctx)
}

func (bhc *BinanceHealthCheck) Timeout() time.Duration {
	return bhc.timeout
}

// CircuitBreakerHealthCheck checks circuit breaker status
type CircuitBreakerHealthCheck struct {
	name    string
	timeout time.Duration
	stateFn func() string
}

// NewCircuitBreakerHealthCheck creates a circuit breaker health check
func NewCircuitBreakerHealthCheck(stateFn func() string) HealthCheck {
	return &CircuitBreakerHealthCheck{
		name:    "circuit_breaker",
		timeout: 1 * time.Second,
		stateFn: stateFn,
	}
}

func (cbhc *CircuitBreakerHealthCheck) Name() string {
	return cbhc.name
}

func (cbhc *CircuitBreakerHealthCheck) Check(ctx context.Context) error {
	state := cbhc.stateFn()
	if state == "open" {
		return fmt.Errorf("circuit breaker is open")
	}
	return nil
}

func (cbhc *CircuitBreakerHealthCheck) Timeout() time.Duration {
	return cbhc.timeout
}

// RateLimiterHealthCheck checks rate limiter status
type RateLimiterHealthCheck struct {
	name    string
	timeout time.Duration
	usageFn func() (float64, error)
}

// NewRateLimiterHealthCheck creates a rate limiter health check
func NewRateLimiterHealthCheck(usageFn func() (float64, error)) HealthCheck {
	return &RateLimiterHealthCheck{
		name:    "rate_limiter",
		timeout: 1 * time.Second,
		usageFn: usageFn,
	}
}

func (rlhc *RateLimiterHealthCheck) Name() string {
	return rlhc.name
}

func (rlhc *RateLimiterHealthCheck) Check(ctx context.Context) error {
	usage, err := rlhc.usageFn()
	if err != nil {
		return err
	}
	
	// Alert if usage is above 80%
	if usage > 80 {
		return fmt.Errorf("rate limit usage too high: %.2f%%", usage)
	}
	
	return nil
}

func (rlhc *RateLimiterHealthCheck) Timeout() time.Duration {
	return rlhc.timeout
}
