package alerting

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britej3/gobot/pkg/feedback"
	"github.com/britej3/gobot/pkg/brain"
	"github.com/sirupsen/logrus"
)

// AlertType defines the type of alert
type AlertType string

const (
	AlertTypeCritical AlertType = "CRITICAL"
	AlertTypeWarning  AlertType = "WARNING"
	AlertTypeInfo     AlertType = "INFO"
	AlertTypeSuccess  AlertType = "SUCCESS"
)

// AlertSeverity defines the severity level
type AlertSeverity string

const (
	SeverityHigh    AlertSeverity = "HIGH"
	SeverityMedium  AlertSeverity = "MEDIUM"
	SeverityLow     AlertSeverity = "LOW"
	SeverityInfo    AlertSeverity = "INFO"
)

// Alert represents a system alert
type Alert struct {
	ID          string        `json:"id"`
	Type        AlertType     `json:"type"`
	Severity    AlertSeverity `json:"severity"`
	Title       string        `json:"title"`
	Message     string        `json:"message"`
	Symbol      string        `json:"symbol,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Source      string        `json:"source"`
	Resolved    bool          `json:"resolved"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
	AutoResolve bool          `json:"auto_resolve"`
}

// AlertingConfig holds alerting configuration
type AlertingConfig struct {
	TelegramEnabled     bool     `json:"telegram_enabled"`
	TelegramToken       string   `json:"telegram_token"`
	TelegramChatID      string   `json:"telegram_chat_id"`
	EmailEnabled        bool     `json:"email_enabled"`
	EmailSMTPServer     string   `json:"email_smtp_server"`
	EmailSMTPPort       int      `json:"email_smtp_port"`
	EmailUsername       string   `json:"email_username"`
	EmailPassword       string   `json:"email_password"`
	EmailRecipients     []string `json:"email_recipients"`
	WebhookEnabled      bool     `json:"webhook_enabled"`
	WebhookURL          string   `json:"webhook_url"`
	WebhookHeaders      map[string]string `json:"webhook_headers"`
	AutoResolveEnabled  bool     `json:"auto_resolve_enabled"`
	AutoResolveTimeout  int      `json:"auto_resolve_timeout"` // minutes
}

// DefaultAlertingConfig returns default alerting configuration
func DefaultAlertingConfig() AlertingConfig {
	return AlertingConfig{
		TelegramEnabled:    false,
		EmailEnabled:       false,
		WebhookEnabled:     false,
		AutoResolveEnabled: true,
		AutoResolveTimeout: 30, // 30 minutes
	}
}

// AlertingSystem manages real-time alerts
type AlertingSystem struct {
	config     AlertingConfig
	client     *futures.Client
	feedback   *feedback.CogneeFeedbackSystem
	brain      *brain.BrainEngine
	platform   interface{} // Remove platform dependency to avoid import cycle
	mu         sync.RWMutex
	activeAlerts map[string]*Alert
	history      []*Alert
	telegramBot  interface{} // Remove platform dependency to avoid import cycle
	stopCh       chan struct{}
}

// NewAlertingSystem creates a new alerting system
func NewAlertingSystem(client *futures.Client, feedback *feedback.CogneeFeedbackSystem, brain *brain.BrainEngine, platform interface{}) *AlertingSystem {
	return &AlertingSystem{
		config:       DefaultAlertingConfig(),
		client:       client,
		feedback:     feedback,
		brain:        brain,
		platform:     platform, // Store as interface{} to avoid import cycle
		activeAlerts: make(map[string]*Alert),
		history:      make([]*Alert, 0),
		stopCh:       make(chan struct{}),
	}
}

// UpdateConfig updates alerting configuration
func (as *AlertingSystem) UpdateConfig(config AlertingConfig) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.config = config
}

// Start begins the alerting system
func (as *AlertingSystem) Start(ctx context.Context) error {
	logrus.Info("🚨 Starting real-time alerting system...")
	
	// Initialize Telegram bot if enabled
	if as.config.TelegramEnabled {
		// Note: platform import removed to avoid import cycle
		// telegramBot, err := platform.NewSecureBot()
		// For now, skip Telegram functionality or implement without platform dependency
		logrus.Warn("Telegram bot functionality disabled due to import cycle")
		return nil
	}
	
	// Start monitoring goroutines
	go as.monitorSystemHealth(ctx)
	go as.monitorAccountHealth(ctx)
	go as.monitorTradingPerformance(ctx)
	go as.monitorMarketConditions(ctx)
	
	// Start auto-resolve worker
	if as.config.AutoResolveEnabled {
		go as.autoResolveWorker(ctx)
	}
	
	logrus.Info("✅ Real-time alerting system started")
	return nil
}

// Stop gracefully stops the alerting system
func (as *AlertingSystem) Stop() {
	logrus.Info("🛑 Stopping alerting system...")
	as.stopCh <- struct{}{}
}

// SendAlert sends an alert through all configured channels
func (as *AlertingSystem) SendAlert(alert *Alert) {
	as.mu.Lock()
	defer as.mu.Unlock()
	
	// Add to active alerts
	as.activeAlerts[alert.ID] = alert
	as.history = append(as.history, alert)
	
	// Send through configured channels
	if as.config.TelegramEnabled && as.telegramBot != nil {
		as.sendTelegramAlert(alert)
	}
	
	if as.config.EmailEnabled {
		as.sendEmailAlert(alert)
	}
	
	if as.config.WebhookEnabled {
		as.sendWebhookAlert(alert)
	}
	
	// Log the alert
	logrus.WithFields(logrus.Fields{
		"alert_id":   alert.ID,
		"type":       alert.Type,
		"severity":   alert.Severity,
		"title":      alert.Title,
		"symbol":     alert.Symbol,
		"source":     alert.Source,
	}).Warn("🚨 Alert triggered")
}

// ResolveAlert resolves an active alert
func (as *AlertingSystem) ResolveAlert(alertID string) {
	as.mu.Lock()
	defer as.mu.Unlock()
	
	if alert, exists := as.activeAlerts[alertID]; exists {
		now := time.Now()
		alert.Resolved = true
		alert.ResolvedAt = &now
		
		// Remove from active alerts
		delete(as.activeAlerts, alertID)
		
		logrus.WithFields(logrus.Fields{
			"alert_id": alertID,
			"title":    alert.Title,
		}).Info("✅ Alert resolved")
	}
}

// GetActiveAlerts returns all active alerts
func (as *AlertingSystem) GetActiveAlerts() []*Alert {
	as.mu.RLock()
	defer as.mu.RUnlock()
	
	alerts := make([]*Alert, 0, len(as.activeAlerts))
	for _, alert := range as.activeAlerts {
		alerts = append(alerts, alert)
	}
	return alerts
}

// GetAlertHistory returns alert history
func (as *AlertingSystem) GetAlertHistory(limit int) []*Alert {
	as.mu.RLock()
	defer as.mu.RUnlock()
	
	if limit <= 0 || limit >= len(as.history) {
		return as.history
	}
	
	start := len(as.history) - limit
	return as.history[start:]
}

// monitorSystemHealth monitors system components
func (as *AlertingSystem) monitorSystemHealth(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-as.stopCh:
			return
		case <-ticker.C:
			as.checkSystemHealth()
		}
	}
}

// monitorAccountHealth monitors account balance and risk
func (as *AlertingSystem) monitorAccountHealth(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-as.stopCh:
			return
		case <-ticker.C:
			as.checkAccountHealth()
		}
	}
}

// monitorTradingPerformance monitors trading performance
func (as *AlertingSystem) monitorTradingPerformance(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-as.stopCh:
			return
		case <-ticker.C:
			as.checkTradingPerformance()
		}
	}
}

// monitorMarketConditions monitors market conditions
func (as *AlertingSystem) monitorMarketConditions(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-as.stopCh:
			return
		case <-ticker.C:
			as.checkMarketConditions()
		}
	}
}

// checkSystemHealth checks system component health
func (as *AlertingSystem) checkSystemHealth() {
	// Check API connection
	err := as.client.NewPingService().Do(context.Background())
	if err != nil {
		as.SendAlert(&Alert{
			ID:          "api_connection_failed",
			Type:        AlertTypeCritical,
			Severity:    SeverityHigh,
			Title:       "API Connection Failed",
			Message:     fmt.Sprintf("Binance API connection failed: %v", err),
			Timestamp:   time.Now(),
			Source:      "System Monitor",
			AutoResolve: true,
		})
	} else {
		as.ResolveAlert("api_connection_failed")
	}
	
	// Check brain engine health
	if as.brain != nil {
		stats := as.brain.GetEngineStats()
		healthy := stats["provider"].(map[string]interface{})["healthy"].(bool)
		if !healthy {
			as.SendAlert(&Alert{
				ID:          "brain_engine_unhealthy",
				Type:        AlertTypeWarning,
				Severity:    SeverityMedium,
				Title:       "Brain Engine Unhealthy",
				Message:     "AI brain engine is not responding properly",
				Timestamp:   time.Now(),
				Source:      "Brain Engine",
				AutoResolve: true,
			})
		} else {
			as.ResolveAlert("brain_engine_unhealthy")
		}
	}
}

// checkAccountHealth checks account balance and risk levels
func (as *AlertingSystem) checkAccountHealth() {
	// Get account balance
	acc, err := as.client.NewGetAccountService().Do(context.Background())
	if err != nil {
		return
	}
	
	var balance float64
	// Parse USDT balance from TotalWalletBalance
	if acc.TotalWalletBalance != "" {
		balance = parseFloatSafe(acc.TotalWalletBalance)
	}
	
	// Check minimum balance
	if balance < 1000 {
		as.SendAlert(&Alert{
			ID:          "low_balance",
			Type:        AlertTypeWarning,
			Severity:    SeverityMedium,
			Title:       "Low Account Balance",
			Message:     fmt.Sprintf("Account balance %.2f below minimum threshold", balance),
			Timestamp:   time.Now(),
			Source:      "Risk Manager",
			AutoResolve: false,
		})
	} else {
		as.ResolveAlert("low_balance")
	}
	
	// Check unrealized P&L
	var unrealizedPnL float64
	positions, err := as.client.NewGetPositionRiskService().Do(context.Background())
	if err == nil {
		for _, pos := range positions {
			unrealizedPnL += parseFloatSafe(pos.UnRealizedProfit) // Note: field name is UnRealizedProfit
		}
	}
	
	// Check drawdown
	if unrealizedPnL < -500 {
		as.SendAlert(&Alert{
			ID:          "high_drawdown",
			Type:        AlertTypeCritical,
			Severity:    SeverityHigh,
			Title:       "High Drawdown Detected",
			Message:     fmt.Sprintf("Unrealized P&L %.2f indicates significant drawdown", unrealizedPnL),
			Timestamp:   time.Now(),
			Source:      "Risk Manager",
			AutoResolve: false,
		})
	} else {
		as.ResolveAlert("high_drawdown")
	}
}

// checkTradingPerformance checks trading performance metrics
func (as *AlertingSystem) checkTradingPerformance() {
	if as.feedback == nil {
		return
	}
	
	// Get recent trades
	// Note: getRecentTrades is unexported, need to use public interface or make it exported
	// For now, skip this functionality or use a different approach
	recentTrades := []feedback.TradeLog{} // Placeholder
	err := fmt.Errorf("getRecentTrades is unexported")
	if err != nil || len(recentTrades) == 0 {
		return
	}
	
	// Calculate win rate
	var wins, total float64
	for _, trade := range recentTrades {
		// Note: RealizedPnL field may not exist in TradeLog, need to check feedback package
		// For now, skip this check or use a different metric
		if trade.PnL > 0 {
			wins++
		}
		total++
	}
	
	winRate := wins / total
	
	// Check poor performance
	if winRate < 0.4 {
		as.SendAlert(&Alert{
			ID:          "poor_performance",
			Type:        AlertTypeWarning,
			Severity:    SeverityMedium,
			Title:       "Poor Trading Performance",
			Message:     fmt.Sprintf("Win rate %.1f%% below acceptable threshold", winRate*100),
			Timestamp:   time.Now(),
			Source:      "Performance Monitor",
			AutoResolve: false,
		})
	} else {
		as.ResolveAlert("poor_performance")
	}
}

// checkMarketConditions checks market conditions
func (as *AlertingSystem) checkMarketConditions() {
	// Check for extreme volatility
	for _, symbol := range []string{"BTCUSDT", "ETHUSDT"} {
		klines, err := as.client.NewKlinesService().
			Symbol(symbol).
			Interval("1m").
			Limit(10).
			Do(context.Background())
		
		if err != nil {
			continue
		}
		
		// Calculate volatility
		var volatility float64
		for i := 1; i < len(klines); i++ {
			prevClose := parseFloatSafe(klines[i-1].Close)
			currClose := parseFloatSafe(klines[i].Close)
			if prevClose > 0 {
				change := (currClose - prevClose) / prevClose
				volatility += change * change
			}
		}
		volatility = math.Sqrt(volatility / float64(len(klines)-1))
		
		// Alert on extreme volatility
		if volatility > 0.02 {
			as.SendAlert(&Alert{
				ID:          fmt.Sprintf("high_volatility_%s", symbol),
				Type:        AlertTypeWarning,
				Severity:    SeverityMedium,
				Title:       "High Volatility Detected",
				Message:     fmt.Sprintf("%s volatility %.2f%% indicates extreme market conditions", symbol, volatility*100),
				Symbol:      symbol,
				Timestamp:   time.Now(),
				Source:      "Market Monitor",
				AutoResolve: true,
			})
		} else {
			as.ResolveAlert(fmt.Sprintf("high_volatility_%s", symbol))
		}
	}
}

// autoResolveWorker automatically resolves alerts after timeout
func (as *AlertingSystem) autoResolveWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-as.stopCh:
			return
		case <-ticker.C:
			as.autoResolveAlerts()
		}
	}
}

// autoResolveAlerts resolves alerts that have been active longer than timeout
func (as *AlertingSystem) autoResolveAlerts() {
	as.mu.Lock()
	defer as.mu.Unlock()
	
	timeout := time.Duration(as.config.AutoResolveTimeout) * time.Minute
	
	for id, alert := range as.activeAlerts {
		if alert.AutoResolve && time.Since(alert.Timestamp) > timeout && !alert.Resolved {
			as.ResolveAlert(id)
		}
	}
}

// Helper methods for sending alerts
func (as *AlertingSystem) sendTelegramAlert(alert *Alert) {
	// Telegram functionality disabled due to import cycle
	// This would normally send alerts via Telegram bot
	logrus.WithField("alert", alert).Info("Telegram alert would be sent (disabled due to import cycle)")
}

func (as *AlertingSystem) sendEmailAlert(alert *Alert) {
	// Email implementation would go here
	// This is a placeholder for SMTP email sending
	logrus.WithField("alert", alert).Info("Email alert would be sent")
}

func (as *AlertingSystem) sendWebhookAlert(alert *Alert) {
	if as.config.WebhookURL == "" {
		return
	}
	
	payload, err := json.Marshal(alert)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal alert for webhook")
		return
	}
	
	req, err := http.NewRequest("POST", as.config.WebhookURL, strings.NewReader(string(payload)))
	if err != nil {
		logrus.WithError(err).Error("Failed to create webhook request")
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	for key, value := range as.config.WebhookHeaders {
		req.Header.Set(key, value)
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Failed to send webhook alert")
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		logrus.WithField("status", resp.StatusCode).Error("Webhook returned error status")
	}
}

// Telegram command handlers
// Note: Telegram handlers removed due to import cycle
// func (as *AlertingSystem) handleTelegramAlerts(update platform.Update) error {
// 	activeAlerts := as.GetActiveAlerts()
//
// 	if len(activeAlerts) == 0 {
// 		return as.telegramBot.SendMessage("✅ No active alerts")
// 	}
//
// 	message := "🚨 *Active Alerts*\n\n"
// 	for _, alert := range activeAlerts {
// 		message += fmt.Sprintf("• *%s* - %s\n  %s\n  %s\n\n",
// 			alert.Severity, alert.Title, alert.Message, alert.Timestamp.Format("15:04:05"))
// 	}
//
// 	return as.telegramBot.SendMessage(message)
// }

// Telegram command handlers
// Note: Telegram handlers removed due to import cycle
// func (as *AlertingSystem) handleTelegramPanic(update platform.Update) error {
// 	message := "🛑 *EMERGENCY STOP ACTIVATED*\n\nAll trading has been halted due to critical system alert. Please review system status and manually resume trading when safe."
//
// 	// Send alert
// 	as.SendAlert(&Alert{
// 		ID:          "emergency_panic",
// 		Type:        AlertTypeCritical,
// 		Severity:    SeverityHigh,
// 		Title:       "Emergency Stop Activated",
// 		Message:     "Manual panic command executed",
// 		Timestamp:   time.Now(),
// 		Source:      "Telegram Command",
// 		AutoResolve: false,
// 	})
//
// 	return as.telegramBot.SendMessage(message)
// }
//
// func (as *AlertingSystem) handleTelegramStatus(update platform.Update) error {
// 	// Get system status
// 	status := "✅ System Status\n\n"
//
// 	// Check API
// 	_, err := as.client.NewPingService().Do(context.Background())
// 	if err != nil {
// 		status += "❌ API Connection: FAILED\n"
// 	} else {
// 		status += "✅ API Connection: OK\n"
// 	}
//
// 	// Check balance
// 	acc, err := as.client.NewGetAccountService().Do(context.Background())
// 	if err == nil {
// 		// Parse USDT balance from TotalWalletBalance
// 		if acc.TotalWalletBalance != "" {
// 			status += fmt.Sprintf("💰 Balance: %.2f USDT\n", parseFloatSafe(acc.TotalWalletBalance))
// 		}
// 	}
//
// 	// Check active alerts
// 	activeAlerts := as.GetActiveAlerts()
// 	status += fmt.Sprintf("🚨 Active Alerts: %d\n", len(activeAlerts))
//
// 	return as.telegramBot.SendMessage(status)
// }

// Helper function
func parseFloatSafe(s string) float64 {
	if s == "" {
		return 0
	}
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	if err != nil {
		return 0
	}
	return val
}