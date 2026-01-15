package alerting

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type TelegramConfig struct {
	Token      string
	ChatID     string
	Enabled    bool
	HTTPClient *http.Client
}

type TelegramAlert struct {
	config TelegramConfig
}

type AlertType string

const (
	AlertTradeExecution AlertType = "TRADE"
	AlertPnLPositive    AlertType = "PnL+"
	AlertPnLNegative    AlertType = "PnL-"
	AlertRiskBreach     AlertType = "RISK"
	AlertSystemError    AlertType = "ERROR"
	AlertDailySummary   AlertType = "SUMMARY"
	AlertKillSwitch     AlertType = "KILL"
)

func NewTelegramAlert(cfg TelegramConfig) *TelegramAlert {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &TelegramAlert{config: cfg}
}

func (t *TelegramAlert) Send(alertType AlertType, message string) error {
	if !t.config.Enabled {
		return nil
	}

	if t.config.Token == "" || t.config.ChatID == "" {
		return nil
	}

	emoji := ""
	switch alertType {
	case AlertTradeExecution:
		emoji = "📊"
	case AlertPnLPositive:
		emoji = "💰"
	case AlertPnLNegative:
		emoji = "📉"
	case AlertRiskBreach:
		emoji = "⚠️"
	case AlertSystemError:
		emoji = "❌"
	case AlertDailySummary:
		emoji = "📋"
	case AlertKillSwitch:
		emoji = "🛑"
	}

	url := fmt.Sprintf(
		"https://api.telegram.org/bot%s/sendMessage",
		t.config.Token,
	)

	payload := fmt.Sprintf(
		`{"chat_id":"%s","text":"%s %s","parse_mode":"Markdown"}`,
		t.config.ChatID,
		emoji,
		message,
	)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(payload))

	resp, err := t.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

func (t *TelegramAlert) SendTrade(tradeInfo string) error {
	return t.Send(AlertTradeExecution, tradeInfo)
}

func (t *TelegramAlert) SendPnL(pnl float64, symbol string) error {
	sign := "+"
	if pnl < 0 {
		sign = ""
	}
	msg := fmt.Sprintf("%s%s on %s", sign, formatPnL(pnl), symbol)
	if pnl >= 0 {
		return t.Send(AlertPnLPositive, msg)
	}
	return t.Send(AlertPnLNegative, msg)
}

func (t *TelegramAlert) SendRiskAlert(reason string) error {
	return t.Send(AlertRiskBreach, reason)
}

func (t *TelegramAlert) SendError(err string) error {
	return t.Send(AlertSystemError, err)
}

func (t *TelegramAlert) SendKillSwitch() error {
	return t.Send(AlertKillSwitch, "🛑 KILL SWITCH ACTIVATED - TRADING HALTED")
}

func formatPnL(pnl float64) string {
	return fmt.Sprintf("$%.2f", pnl)
}

type AuditLogger struct {
	auditPath string
	tradePath string
	enabled   bool
}

type AuditConfig struct {
	AuditLogPath   string
	TradeLogPath   string
	Enabled        bool
	DetailedTrades bool
}

func NewAuditLogger(cfg AuditConfig) *AuditLogger {
	if cfg.AuditLogPath == "" {
		cfg.AuditLogPath = "/Users/britebrt/GOBOT/logs/mainnet_audit.log"
	}
	if cfg.TradeLogPath == "" {
		cfg.TradeLogPath = "/Users/britebrt/GOBOT/logs/trades_mainnet.log"
	}

	logger := &AuditLogger{
		auditPath: cfg.AuditLogPath,
		tradePath: cfg.TradeLogPath,
		enabled:   cfg.Enabled,
	}

	if cfg.Enabled {
		logger.ensureFileExists(cfg.AuditLogPath)
		logger.ensureFileExists(cfg.TradeLogPath)
	}

	return logger
}

func (l *AuditLogger) ensureFileExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.WriteFile(path, []byte(""), 0644)
	}
}

func (l *AuditLogger) Log(event string, data map[string]interface{}) {
	if !l.enabled {
		return
	}

	entry := fmt.Sprintf("[%s] %s | %v\n", time.Now().Format(time.RFC3339), event, data)
	l.appendToFile(l.auditPath, entry)
}

func (l *AuditLogger) LogTrade(trade map[string]interface{}) {
	if !l.enabled {
		return
	}

	entry := fmt.Sprintf(
		"[%s] TRADE | Symbol:%s | Side:%s | PnL:%s | Status:%s\n",
		time.Now().Format(time.RFC3339),
		trade["symbol"],
		trade["side"],
		formatTradePnL(trade["pnl"]),
		trade["status"],
	)
	l.appendToFile(l.tradePath, entry)
}

func (l *AuditLogger) appendToFile(path, entry string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error writing to log file: %v\n", err)
		return
	}
	defer f.Close()
	f.WriteString(entry)
}

func formatTradePnL(pnl interface{}) string {
	switch v := pnl.(type) {
	case float64:
		return fmt.Sprintf("$%.2f", v)
	case string:
		return v
	default:
		return "N/A"
	}
}
