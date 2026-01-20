package binance

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CircuitBreaker defines the interface for circuit breaker functionality
type CircuitBreaker interface {
	Allow() bool
	RecordSuccess()
	RecordFailure()
	GetState() CircuitState
	Reset()
}

// CircuitState represents the state of the circuit breaker
type CircuitState string

const (
	// StateClosed means the circuit is closed and requests are allowed
	StateClosed CircuitState = "closed"
	// StateOpen means the circuit is open and requests are blocked
	StateOpen CircuitState = "open"
	// StateHalfOpen means the circuit is testing if it should close
	StateHalfOpen CircuitState = "half_open"
)

// AdaptiveCircuitBreaker implements an adaptive circuit breaker
type AdaptiveCircuitBreaker struct {
	// Configuration
	failureThreshold  int
	successThreshold  int
	timeout           time.Duration
	halfOpenRequests  int

	// State
	state             CircuitState
	failures          int
	successes         int
	lastFailureTime   time.Time
	lastStateChange   time.Time
	halfOpenAttempts  int

	// Statistics
	totalRequests     int64
	totalFailures     int64
	totalSuccesses    int64

	mu     sync.RWMutex
	logger *logrus.Logger
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
	HalfOpenRequests int
}

// NewAdaptiveCircuitBreaker creates a new adaptive circuit breaker
func NewAdaptiveCircuitBreaker() CircuitBreaker {
	return NewAdaptiveCircuitBreakerWithConfig(CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		HalfOpenRequests: 3,
	})
}

// NewAdaptiveCircuitBreakerWithConfig creates a circuit breaker with custom config
func NewAdaptiveCircuitBreakerWithConfig(config CircuitBreakerConfig) CircuitBreaker {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	cb := &AdaptiveCircuitBreaker{
		failureThreshold: config.FailureThreshold,
		successThreshold: config.SuccessThreshold,
		timeout:          config.Timeout,
		halfOpenRequests: config.HalfOpenRequests,
		state:            StateClosed,
		logger:           logger,
	}

	logger.WithFields(logrus.Fields{
		"failure_threshold": config.FailureThreshold,
		"success_threshold": config.SuccessThreshold,
		"timeout":           config.Timeout,
		"half_open_requests": config.HalfOpenRequests,
	}).Info("circuit_breaker_initialized")

	return cb
}

// Allow checks if a request is allowed through the circuit breaker
func (cb *AdaptiveCircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastStateChange) >= cb.timeout {
			cb.transitionToHalfOpen()
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.halfOpenAttempts < cb.halfOpenRequests {
			cb.halfOpenAttempts++
			return true
		}
		return false

	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *AdaptiveCircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalSuccesses++
	cb.failures = 0
	cb.successes++

	switch cb.state {
	case StateHalfOpen:
		if cb.successes >= cb.successThreshold {
			cb.transitionToClosed()
		}
	case StateClosed:
		// Already closed, nothing to do
	case StateOpen:
		// Shouldn't happen, but reset if it does
		cb.transitionToClosed()
	}
}

// RecordFailure records a failed request
func (cb *AdaptiveCircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalFailures++
	cb.failures++
	cb.successes = 0
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.failureThreshold {
			cb.transitionToOpen()
		}
	case StateHalfOpen:
		cb.transitionToOpen()
	case StateOpen:
		// Already open, nothing to do
	}
}

// GetState returns the current state of the circuit breaker
func (cb *AdaptiveCircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state
func (cb *AdaptiveCircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.transitionToClosed()
	cb.logger.Info("circuit_breaker_reset")
}

// State transition methods

func (cb *AdaptiveCircuitBreaker) transitionToClosed() {
	oldState := cb.state
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenAttempts = 0
	cb.lastStateChange = time.Now()

	cb.logger.WithFields(logrus.Fields{
		"old_state": oldState,
		"new_state": StateClosed,
	}).Info("circuit_breaker_state_transition")
}

func (cb *AdaptiveCircuitBreaker) transitionToOpen() {
	oldState := cb.state
	cb.state = StateOpen
	cb.successes = 0
	cb.halfOpenAttempts = 0
	cb.lastStateChange = time.Now()

	cb.logger.WithFields(logrus.Fields{
		"old_state": oldState,
		"new_state": StateOpen,
		"failures":  cb.failures,
	}).Warn("circuit_breaker_opened")
}

func (cb *AdaptiveCircuitBreaker) transitionToHalfOpen() {
	oldState := cb.state
	cb.state = StateHalfOpen
	cb.successes = 0
	cb.halfOpenAttempts = 0
	cb.lastStateChange = time.Now()

	cb.logger.WithFields(logrus.Fields{
		"old_state": oldState,
		"new_state": StateHalfOpen,
	}).Info("circuit_breaker_half_open")
}

// GetStats returns statistics about the circuit breaker
func (cb *AdaptiveCircuitBreaker) GetStats() *CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	failureRate := 0.0
	if cb.totalRequests > 0 {
		failureRate = float64(cb.totalFailures) / float64(cb.totalRequests) * 100
	}

	return &CircuitBreakerStats{
		State:            cb.state,
		TotalRequests:    cb.totalRequests,
		TotalFailures:    cb.totalFailures,
		TotalSuccesses:   cb.totalSuccesses,
		CurrentFailures:  cb.failures,
		CurrentSuccesses: cb.successes,
		FailureRate:      failureRate,
		LastStateChange:  cb.lastStateChange,
	}
}

// CircuitBreakerStats represents circuit breaker statistics
type CircuitBreakerStats struct {
	State            CircuitState
	TotalRequests    int64
	TotalFailures    int64
	TotalSuccesses   int64
	CurrentFailures  int
	CurrentSuccesses int
	FailureRate      float64
	LastStateChange  time.Time
}

// String returns a string representation of the stats
func (s *CircuitBreakerStats) String() string {
	return fmt.Sprintf(
		"State: %s, Total: %d, Failures: %d (%.2f%%), Successes: %d, Current Failures: %d, Current Successes: %d",
		s.State,
		s.TotalRequests,
		s.TotalFailures,
		s.FailureRate,
		s.TotalSuccesses,
		s.CurrentFailures,
		s.CurrentSuccesses,
	)
}
