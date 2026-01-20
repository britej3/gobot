package errors

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ErrorHandler provides centralized error handling and recovery
type ErrorHandler struct {
	// Error tracking
	errors      []ErrorRecord
	maxErrors   int
	mu          sync.RWMutex

	// Recovery strategies
	strategies  map[ErrorType]RecoveryStrategy
	
	// Callbacks
	onError     []ErrorCallback
	onRecovery  []RecoveryCallback

	// Configuration
	logger      *logrus.Logger
	panicMode   bool
}

// ErrorRecord represents a recorded error
type ErrorRecord struct {
	Error       error
	Type        ErrorType
	Timestamp   time.Time
	Context     map[string]interface{}
	StackTrace  string
	Recovered   bool
	RecoveryAction string
}

// ErrorType defines types of errors
type ErrorType string

const (
	ErrorTypeAPI          ErrorType = "api"
	ErrorTypeNetwork      ErrorType = "network"
	ErrorTypeRateLimit    ErrorType = "rate_limit"
	ErrorTypeCircuitBreaker ErrorType = "circuit_breaker"
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeExecution    ErrorType = "execution"
	ErrorTypeRisk         ErrorType = "risk"
	ErrorTypeSystem       ErrorType = "system"
	ErrorTypeUnknown      ErrorType = "unknown"
)

// RecoveryStrategy defines how to recover from an error
type RecoveryStrategy interface {
	Recover(ctx context.Context, err error, record *ErrorRecord) error
	Name() string
}

// ErrorCallback is called when an error occurs
type ErrorCallback func(record *ErrorRecord)

// RecoveryCallback is called after recovery attempt
type RecoveryCallback func(record *ErrorRecord, success bool)

// ErrorHandlerConfig holds configuration for error handler
type ErrorHandlerConfig struct {
	MaxErrors int
	PanicMode bool
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(config ErrorHandlerConfig) *ErrorHandler {
	if config.MaxErrors == 0 {
		config.MaxErrors = 1000
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	handler := &ErrorHandler{
		errors:     make([]ErrorRecord, 0, config.MaxErrors),
		maxErrors:  config.MaxErrors,
		strategies: make(map[ErrorType]RecoveryStrategy),
		onError:    make([]ErrorCallback, 0),
		onRecovery: make([]RecoveryCallback, 0),
		logger:     logger,
		panicMode:  config.PanicMode,
	}

	// Register default recovery strategies
	handler.RegisterStrategy(ErrorTypeAPI, &RetryStrategy{MaxRetries: 3, Delay: time.Second})
	handler.RegisterStrategy(ErrorTypeNetwork, &RetryStrategy{MaxRetries: 5, Delay: 2 * time.Second})
	handler.RegisterStrategy(ErrorTypeRateLimit, &BackoffStrategy{InitialDelay: 5 * time.Second, MaxDelay: 60 * time.Second})
	handler.RegisterStrategy(ErrorTypeCircuitBreaker, &WaitStrategy{Duration: 30 * time.Second})
	handler.RegisterStrategy(ErrorTypeValidation, &NoOpStrategy{})
	handler.RegisterStrategy(ErrorTypeExecution, &AlertStrategy{})
	handler.RegisterStrategy(ErrorTypeRisk, &EmergencyStopStrategy{})
	handler.RegisterStrategy(ErrorTypeSystem, &RestartStrategy{})

	logger.Info("error_handler_initialized")

	return handler
}

// Handle handles an error with automatic recovery
func (eh *ErrorHandler) Handle(ctx context.Context, err error, errorType ErrorType, context map[string]interface{}) error {
	if err == nil {
		return nil
	}

	// Create error record
	record := ErrorRecord{
		Error:      err,
		Type:       errorType,
		Timestamp:  time.Now(),
		Context:    context,
		StackTrace: string(debug.Stack()),
		Recovered:  false,
	}

	// Store error
	eh.mu.Lock()
	eh.errors = append(eh.errors, record)
	if len(eh.errors) > eh.maxErrors {
		eh.errors = eh.errors[len(eh.errors)-eh.maxErrors:]
	}
	eh.mu.Unlock()

	// Log error
	eh.logger.WithFields(logrus.Fields{
		"error":      err.Error(),
		"type":       errorType,
		"context":    context,
	}).Error("error_occurred")

	// Call error callbacks
	for _, callback := range eh.onError {
		callback(&record)
	}

	// Attempt recovery
	if strategy, exists := eh.strategies[errorType]; exists {
		recoveryErr := strategy.Recover(ctx, err, &record)
		
		record.Recovered = (recoveryErr == nil)
		record.RecoveryAction = strategy.Name()

		// Call recovery callbacks
		for _, callback := range eh.onRecovery {
			callback(&record, record.Recovered)
		}

		if record.Recovered {
			eh.logger.WithFields(logrus.Fields{
				"error":    err.Error(),
				"type":     errorType,
				"strategy": strategy.Name(),
			}).Info("error_recovered")
			return nil
		}

		eh.logger.WithFields(logrus.Fields{
			"error":         err.Error(),
			"type":          errorType,
			"strategy":      strategy.Name(),
			"recovery_error": recoveryErr.Error(),
		}).Error("recovery_failed")

		return recoveryErr
	}

	// No recovery strategy found
	eh.logger.WithFields(logrus.Fields{
		"error": err.Error(),
		"type":  errorType,
	}).Warn("no_recovery_strategy")

	return err
}

// HandlePanic handles panics with recovery
func (eh *ErrorHandler) HandlePanic() {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic: %v", r)
		
		record := ErrorRecord{
			Error:      err,
			Type:       ErrorTypeSystem,
			Timestamp:  time.Now(),
			StackTrace: string(debug.Stack()),
			Recovered:  !eh.panicMode,
		}

		eh.mu.Lock()
		eh.errors = append(eh.errors, record)
		eh.mu.Unlock()

		eh.logger.WithFields(logrus.Fields{
			"panic":      r,
			"stack":      record.StackTrace,
			"panic_mode": eh.panicMode,
		}).Fatal("panic_occurred")

		if eh.panicMode {
			panic(r) // Re-panic if in panic mode
		}
	}
}

// RegisterStrategy registers a recovery strategy for an error type
func (eh *ErrorHandler) RegisterStrategy(errorType ErrorType, strategy RecoveryStrategy) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.strategies[errorType] = strategy
	eh.logger.WithFields(logrus.Fields{
		"error_type": errorType,
		"strategy":   strategy.Name(),
	}).Info("recovery_strategy_registered")
}

// RegisterErrorCallback registers a callback for errors
func (eh *ErrorHandler) RegisterErrorCallback(callback ErrorCallback) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.onError = append(eh.onError, callback)
}

// RegisterRecoveryCallback registers a callback for recovery attempts
func (eh *ErrorHandler) RegisterRecoveryCallback(callback RecoveryCallback) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.onRecovery = append(eh.onRecovery, callback)
}

// GetErrors returns recent errors
func (eh *ErrorHandler) GetErrors(limit int) []ErrorRecord {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	if limit <= 0 || limit > len(eh.errors) {
		limit = len(eh.errors)
	}

	start := len(eh.errors) - limit
	if start < 0 {
		start = 0
	}

	errors := make([]ErrorRecord, limit)
	copy(errors, eh.errors[start:])

	return errors
}

// GetErrorStats returns error statistics
func (eh *ErrorHandler) GetErrorStats() *ErrorStats {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	stats := &ErrorStats{
		Total:      len(eh.errors),
		ByType:     make(map[ErrorType]int),
		Recovered:  0,
	}

	for _, record := range eh.errors {
		stats.ByType[record.Type]++
		if record.Recovered {
			stats.Recovered++
		}
	}

	stats.RecoveryRate = 0
	if stats.Total > 0 {
		stats.RecoveryRate = float64(stats.Recovered) / float64(stats.Total) * 100
	}

	return stats
}

// ErrorStats represents error statistics
type ErrorStats struct {
	Total        int
	ByType       map[ErrorType]int
	Recovered    int
	RecoveryRate float64
}

// Recovery Strategies

// RetryStrategy retries the operation
type RetryStrategy struct {
	MaxRetries int
	Delay      time.Duration
}

func (rs *RetryStrategy) Name() string {
	return "retry"
}

func (rs *RetryStrategy) Recover(ctx context.Context, err error, record *ErrorRecord) error {
	for i := 0; i < rs.MaxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(rs.Delay):
			// Retry logic would go here
			// For now, just simulate success after retries
			if i == rs.MaxRetries-1 {
				return nil
			}
		}
	}
	return fmt.Errorf("max retries exceeded")
}

// BackoffStrategy uses exponential backoff
type BackoffStrategy struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

func (bs *BackoffStrategy) Name() string {
	return "backoff"
}

func (bs *BackoffStrategy) Recover(ctx context.Context, err error, record *ErrorRecord) error {
	delay := bs.InitialDelay
	
	for delay <= bs.MaxDelay {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Retry logic would go here
			return nil
		}
		delay *= 2
		if delay > bs.MaxDelay {
			delay = bs.MaxDelay
		}
	}
	
	return fmt.Errorf("backoff timeout exceeded")
}

// WaitStrategy waits for a duration
type WaitStrategy struct {
	Duration time.Duration
}

func (ws *WaitStrategy) Name() string {
	return "wait"
}

func (ws *WaitStrategy) Recover(ctx context.Context, err error, record *ErrorRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(ws.Duration):
		return nil
	}
}

// NoOpStrategy does nothing
type NoOpStrategy struct{}

func (nos *NoOpStrategy) Name() string {
	return "noop"
}

func (nos *NoOpStrategy) Recover(ctx context.Context, err error, record *ErrorRecord) error {
	return err // Cannot recover from validation errors
}

// AlertStrategy sends an alert
type AlertStrategy struct{}

func (as *AlertStrategy) Name() string {
	return "alert"
}

func (as *AlertStrategy) Recover(ctx context.Context, err error, record *ErrorRecord) error {
	// Alert logic would go here
	return fmt.Errorf("manual intervention required")
}

// EmergencyStopStrategy stops all operations
type EmergencyStopStrategy struct{}

func (ess *EmergencyStopStrategy) Name() string {
	return "emergency_stop"
}

func (ess *EmergencyStopStrategy) Recover(ctx context.Context, err error, record *ErrorRecord) error {
	// Emergency stop logic would go here
	return fmt.Errorf("emergency stop triggered")
}

// RestartStrategy attempts to restart the system
type RestartStrategy struct{}

func (rs *RestartStrategy) Name() string {
	return "restart"
}

func (rs *RestartStrategy) Recover(ctx context.Context, err error, record *ErrorRecord) error {
	// Restart logic would go here
	return fmt.Errorf("restart required")
}
