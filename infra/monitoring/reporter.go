package monitoring

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Reporter provides real-time monitoring and reporting
type Reporter struct {
	// Metrics storage
	metrics     map[string]*Metric
	events      []Event
	maxEvents   int
	mu          sync.RWMutex

	// Reporting channels
	metricsChan chan *Metric
	eventsChan  chan Event
	alertsChan  chan Alert

	// Configuration
	reportInterval time.Duration
	logger         *logrus.Logger

	// Control
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// Metric represents a single metric measurement
type Metric struct {
	Name      string
	Value     float64
	Unit      string
	Timestamp time.Time
	Tags      map[string]string
}

// Event represents a system event
type Event struct {
	Type      EventType
	Message   string
	Severity  Severity
	Timestamp time.Time
	Data      map[string]interface{}
}

// Alert represents a system alert
type Alert struct {
	Level     AlertLevel
	Title     string
	Message   string
	Timestamp time.Time
	Data      map[string]interface{}
}

// EventType defines types of events
type EventType string

const (
	EventTypeOrder       EventType = "order"
	EventTypePosition    EventType = "position"
	EventTypeRisk        EventType = "risk"
	EventTypeSystem      EventType = "system"
	EventTypePerformance EventType = "performance"
	EventTypeError       EventType = "error"
)

// Severity defines event severity levels
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// AlertLevel defines alert levels
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// ReporterConfig holds configuration for the reporter
type ReporterConfig struct {
	ReportInterval time.Duration
	MaxEvents      int
}

// NewReporter creates a new monitoring reporter
func NewReporter(config ReporterConfig) *Reporter {
	if config.ReportInterval == 0 {
		config.ReportInterval = 10 * time.Second
	}
	if config.MaxEvents == 0 {
		config.MaxEvents = 1000
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	reporter := &Reporter{
		metrics:        make(map[string]*Metric),
		events:         make([]Event, 0, config.MaxEvents),
		maxEvents:      config.MaxEvents,
		metricsChan:    make(chan *Metric, 100),
		eventsChan:     make(chan Event, 100),
		alertsChan:     make(chan Alert, 50),
		reportInterval: config.ReportInterval,
		logger:         logger,
		stopChan:       make(chan struct{}),
	}

	// Start background workers
	reporter.wg.Add(3)
	go reporter.metricsWorker()
	go reporter.eventsWorker()
	go reporter.reportWorker()

	logger.WithFields(logrus.Fields{
		"report_interval": config.ReportInterval,
		"max_events":      config.MaxEvents,
	}).Info("reporter_initialized")

	return reporter
}

// RecordMetric records a metric measurement
func (r *Reporter) RecordMetric(name string, value float64, unit string, tags map[string]string) {
	metric := &Metric{
		Name:      name,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
		Tags:      tags,
	}

	select {
	case r.metricsChan <- metric:
	default:
		r.logger.Warn("metrics_channel_full")
	}
}

// RecordEvent records a system event
func (r *Reporter) RecordEvent(eventType EventType, message string, severity Severity, data map[string]interface{}) {
	event := Event{
		Type:      eventType,
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Data:      data,
	}

	select {
	case r.eventsChan <- event:
	default:
		r.logger.Warn("events_channel_full")
	}
}

// SendAlert sends an alert
func (r *Reporter) SendAlert(level AlertLevel, title, message string, data map[string]interface{}) {
	alert := Alert{
		Level:     level,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Data:      data,
	}

	select {
	case r.alertsChan <- alert:
	default:
		r.logger.Warn("alerts_channel_full")
	}

	// Also log the alert
	r.logger.WithFields(logrus.Fields{
		"level":   level,
		"title":   title,
		"message": message,
	}).Warn("alert_sent")
}

// GetMetrics returns all current metrics
func (r *Reporter) GetMetrics() map[string]*Metric {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a copy
	metrics := make(map[string]*Metric, len(r.metrics))
	for k, v := range r.metrics {
		metricCopy := *v
		metrics[k] = &metricCopy
	}

	return metrics
}

// GetEvents returns recent events
func (r *Reporter) GetEvents(limit int) []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || limit > len(r.events) {
		limit = len(r.events)
	}

	// Return most recent events
	start := len(r.events) - limit
	if start < 0 {
		start = 0
	}

	events := make([]Event, limit)
	copy(events, r.events[start:])

	return events
}

// GetReport generates a comprehensive report
func (r *Reporter) GetReport() *Report {
	r.mu.RLock()
	defer r.mu.RUnlock()

	report := &Report{
		Timestamp:    time.Now(),
		Metrics:      make(map[string]*Metric),
		RecentEvents: make([]Event, 0),
	}

	// Copy metrics
	for k, v := range r.metrics {
		metricCopy := *v
		report.Metrics[k] = &metricCopy
	}

	// Copy recent events (last 50)
	eventCount := len(r.events)
	if eventCount > 50 {
		eventCount = 50
	}
	if eventCount > 0 {
		start := len(r.events) - eventCount
		report.RecentEvents = make([]Event, eventCount)
		copy(report.RecentEvents, r.events[start:])
	}

	return report
}

// Report represents a comprehensive system report
type Report struct {
	Timestamp    time.Time
	Metrics      map[string]*Metric
	RecentEvents []Event
}

// ToJSON converts the report to JSON
func (r *Report) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Background workers

func (r *Reporter) metricsWorker() {
	defer r.wg.Done()

	for {
		select {
		case <-r.stopChan:
			return
		case metric := <-r.metricsChan:
			r.mu.Lock()
			r.metrics[metric.Name] = metric
			r.mu.Unlock()

			r.logger.WithFields(logrus.Fields{
				"metric": metric.Name,
				"value":  metric.Value,
				"unit":   metric.Unit,
			}).Debug("metric_recorded")
		}
	}
}

func (r *Reporter) eventsWorker() {
	defer r.wg.Done()

	for {
		select {
		case <-r.stopChan:
			return
		case event := <-r.eventsChan:
			r.mu.Lock()
			r.events = append(r.events, event)
			// Trim events if exceeding max
			if len(r.events) > r.maxEvents {
				r.events = r.events[len(r.events)-r.maxEvents:]
			}
			r.mu.Unlock()

			logLevel := logrus.InfoLevel
			switch event.Severity {
			case SeverityWarning:
				logLevel = logrus.WarnLevel
			case SeverityError, SeverityCritical:
				logLevel = logrus.ErrorLevel
			}

			r.logger.WithFields(logrus.Fields{
				"type":     event.Type,
				"severity": event.Severity,
				"message":  event.Message,
			}).Log(logLevel, "event_recorded")
		}
	}
}

func (r *Reporter) reportWorker() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.reportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			report := r.GetReport()
			r.logger.WithFields(logrus.Fields{
				"metrics_count": len(report.Metrics),
				"events_count":  len(report.RecentEvents),
			}).Info("periodic_report")
		case alert := <-r.alertsChan:
			r.handleAlert(alert)
		}
	}
}

func (r *Reporter) handleAlert(alert Alert) {
	logLevel := logrus.InfoLevel
	switch alert.Level {
	case AlertLevelWarning:
		logLevel = logrus.WarnLevel
	case AlertLevelCritical:
		logLevel = logrus.ErrorLevel
	}

	r.logger.WithFields(logrus.Fields{
		"level":   alert.Level,
		"title":   alert.Title,
		"message": alert.Message,
	}).Log(logLevel, "alert_handled")
}

// Close stops the reporter and all background workers
func (r *Reporter) Close() {
	close(r.stopChan)
	r.wg.Wait()
	r.logger.Info("reporter_closed")
}

// Helper methods for common metrics

// RecordOrderMetric records an order-related metric
func (r *Reporter) RecordOrderMetric(symbol string, side string, quantity float64, price float64) {
	r.RecordMetric("order_placed", 1, "count", map[string]string{
		"symbol": symbol,
		"side":   side,
	})
	r.RecordMetric("order_quantity", quantity, symbol, map[string]string{
		"symbol": symbol,
		"side":   side,
	})
	r.RecordMetric("order_price", price, "USDT", map[string]string{
		"symbol": symbol,
		"side":   side,
	})
}

// RecordPositionMetric records a position-related metric
func (r *Reporter) RecordPositionMetric(symbol string, size float64, pnl float64) {
	r.RecordMetric("position_size", size, symbol, map[string]string{
		"symbol": symbol,
	})
	r.RecordMetric("position_pnl", pnl, "USDT", map[string]string{
		"symbol": symbol,
	})
}

// RecordPerformanceMetric records a performance metric
func (r *Reporter) RecordPerformanceMetric(operation string, duration time.Duration) {
	r.RecordMetric(fmt.Sprintf("performance_%s", operation), float64(duration.Milliseconds()), "ms", map[string]string{
		"operation": operation,
	})
}

// RecordErrorMetric records an error metric
func (r *Reporter) RecordErrorMetric(errorType string) {
	r.RecordMetric("error_count", 1, "count", map[string]string{
		"type": errorType,
	})
}

// Helper methods for common events

// RecordOrderEvent records an order event
func (r *Reporter) RecordOrderEvent(message string, severity Severity, data map[string]interface{}) {
	r.RecordEvent(EventTypeOrder, message, severity, data)
}

// RecordPositionEvent records a position event
func (r *Reporter) RecordPositionEvent(message string, severity Severity, data map[string]interface{}) {
	r.RecordEvent(EventTypePosition, message, severity, data)
}

// RecordRiskEvent records a risk event
func (r *Reporter) RecordRiskEvent(message string, severity Severity, data map[string]interface{}) {
	r.RecordEvent(EventTypeRisk, message, severity, data)
}

// RecordSystemEvent records a system event
func (r *Reporter) RecordSystemEvent(message string, severity Severity, data map[string]interface{}) {
	r.RecordEvent(EventTypeSystem, message, severity, data)
}

// RecordErrorEvent records an error event
func (r *Reporter) RecordErrorEvent(message string, err error, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["error"] = err.Error()
	r.RecordEvent(EventTypeError, message, SeverityError, data)
}
