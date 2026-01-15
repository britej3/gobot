package circuitbreaker

import (
	"sync"
	"time"
)

type CircuitBreaker struct {
	name              string
	failureCount      int
	successCount      int
	lastFailureTime   time.Time
	state             State
	mu                sync.RWMutex
	failureThreshold  int
	recoveryTimeout   time.Duration
	failureWindow     time.Duration
	halfOpenRequests  int
	halfOpenSuccesses int
	onStateChange     func(name string, from State, to State)
}

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold int
	RecoveryTimeout  time.Duration
	FailureWindow    time.Duration
	HalfOpenRequests int
	OnStateChange    func(name string, from State, to State)
}

func New(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.RecoveryTimeout == 0 {
		cfg.RecoveryTimeout = 30 * time.Second
	}
	if cfg.FailureWindow == 0 {
		cfg.FailureWindow = 60 * time.Second
	}
	if cfg.HalfOpenRequests == 0 {
		cfg.HalfOpenRequests = 3
	}

	return &CircuitBreaker{
		name:             cfg.Name,
		state:            StateClosed,
		failureThreshold: cfg.FailureThreshold,
		recoveryTimeout:  cfg.RecoveryTimeout,
		failureWindow:    cfg.FailureWindow,
		halfOpenRequests: cfg.HalfOpenRequests,
		onStateChange:    cfg.OnStateChange,
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		cb.halfOpenSuccesses++
		if cb.halfOpenSuccesses >= cb.halfOpenRequests {
			cb.transitionTo(StateClosed)
		}
	case StateClosed:
		cb.successCount++
		if cb.successCount >= cb.failureThreshold {
			cb.successCount = 0
		}
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()
	cb.failureCount++
	cb.successCount = 0

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.failureThreshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	}
}

func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) >= cb.recoveryTimeout {
			cb.transitionTo(StateHalfOpen)
			cb.halfOpenSuccesses = 0
			return true
		}
		return false
	case StateHalfOpen:
		return cb.halfOpenRequests > 0
	}
	return false
}

func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return Stats{
		State:             cb.state.String(),
		FailureCount:      cb.failureCount,
		SuccessCount:      cb.successCount,
		LastFailureTime:   cb.lastFailureTime,
		HalfOpenRemaining: cb.halfOpenRequests,
	}
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenSuccesses = 0
}

func (cb *CircuitBreaker) transitionTo(newState State) {
	if cb.state == newState {
		return
	}
	oldState := cb.state
	cb.state = newState
	cb.failureCount = 0
	cb.successCount = 0
	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, oldState, newState)
	}
}

type Stats struct {
	State             string
	FailureCount      int
	SuccessCount      int
	LastFailureTime   time.Time
	HalfOpenRemaining int
}

func Execute[T any](cb *CircuitBreaker, fn func() (T, error)) (T, error) {
	if !cb.AllowRequest() {
		var zero T
		return zero, &CircuitOpenError{Message: "circuit breaker is open"}
	}

	result, err := fn()

	if err != nil {
		cb.RecordFailure()
		var zero T
		return zero, err
	}

	cb.RecordSuccess()
	return result, nil
}

type CircuitOpenError struct {
	Message string
}

func (e *CircuitOpenError) Error() string {
	return e.Message
}
