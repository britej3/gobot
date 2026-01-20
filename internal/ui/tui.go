// Package ui provides the Terminal User Interface for GOBOT
// Compatible with Intel Macs (darwin/amd64) and Linux (linux/amd64, linux/arm64)
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/britej3/gobot/internal/health"
)

// ============================================================================
// TUI Layout Constants
// ============================================================================

const (
	// Box drawing characters (UTF-8)
	BoxHorizontal = "─"
	BoxVertical   = "│"
	BoxTopLeft    = "┌"
	BoxTopRight   = "┐"
	BoxBottomLeft = "└"
	BoxBottomRight = "┘"
	BoxTeeLeft    = "├"
	BoxTeeRight   = "┤"
	BoxTeeTop     = "┬"
	BoxTeeBottom  = "┴"
	BoxCross      = "┼"
	
	// Colors (ANSI)
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
)

// ============================================================================
// TUI State
// ============================================================================

// TUIState holds all data for TUI rendering
type TUIState struct {
	// System
	SystemHealth   *health.SystemHealth
	Metrics        *health.SystemMetrics
	
	// Wallet
	Wallet         *health.WalletInfo
	
	// Trading
	Positions      []health.PositionInfo
	RecentTrades   []health.TradeInfo
	TopMovers      []health.TopMoverInfo
	
	// Session
	SessionStart   time.Time
	IsFirstTrade   bool
	TradingStatus  string
	
	// Brain
	BrainLogs      []string
	LastDecision   string
	Confidence     float64
	
	// Errors
	LastError      string
	Warnings       []string
}

// ============================================================================
// TUI Renderer
// ============================================================================

// RenderDashboard renders the full dashboard
func RenderDashboard(state *TUIState) string {
	var sb strings.Builder
	
	// Clear screen
	sb.WriteString("\033[2J\033[H")
	
	// Header
	sb.WriteString(renderHeader(state))
	sb.WriteString("\n")
	
	// Main content (3 columns)
	sb.WriteString(renderMainContent(state))
	sb.WriteString("\n")
	
	// Brain log
	sb.WriteString(renderBrainLog(state))
	
	return sb.String()
}

// renderHeader renders the top status bar
func renderHeader(state *TUIState) string {
	var sb strings.Builder
	
	// Status indicator
	statusColor := ColorGreen
	statusText := "● ACTIVE"
	if state.TradingStatus == "PAUSED" {
		statusColor = ColorYellow
		statusText = "◐ PAUSED"
	} else if state.TradingStatus == "STOPPED" {
		statusColor = ColorRed
		statusText = "○ STOPPED"
	} else if state.TradingStatus == "FIRST_TRADE" {
		statusColor = ColorCyan
		statusText = "◉ FIRST TRADE"
	}
	
	// Build header
	sb.WriteString(fmt.Sprintf("%s%s GOBOT %s", ColorBold, ColorCyan, ColorReset))
	sb.WriteString(fmt.Sprintf(" %s%s%s", statusColor, statusText, ColorReset))
	
	// Uptime
	if state.Metrics != nil {
		sb.WriteString(fmt.Sprintf(" │ Uptime: %s", formatDuration(state.Metrics.Uptime)))
	}
	
	// API Status
	if state.Metrics != nil && state.Metrics.APILatency > 0 {
		latencyColor := ColorGreen
		if state.Metrics.APILatency > 200*time.Millisecond {
			latencyColor = ColorYellow
		}
		if state.Metrics.APILatency > 500*time.Millisecond {
			latencyColor = ColorRed
		}
		sb.WriteString(fmt.Sprintf(" │ API: %s%dms%s", latencyColor, state.Metrics.APILatency.Milliseconds(), ColorReset))
	}
	
	// Memory
	if state.Metrics != nil {
		sb.WriteString(fmt.Sprintf(" │ Mem: %dMB", state.Metrics.MemoryAlloc))
	}
	
	// Time
	sb.WriteString(fmt.Sprintf(" │ %s", time.Now().Format("15:04:05")))
	
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", 100))
	
	return sb.String()
}

// renderMainContent renders the main 3-column layout
func renderMainContent(state *TUIState) string {
	var sb strings.Builder
	
	// Wallet section
	sb.WriteString(renderWalletSection(state))
	sb.WriteString("\n")
	
	// Positions section
	sb.WriteString(renderPositionsSection(state))
	sb.WriteString("\n")
	
	// Top Movers section
	sb.WriteString(renderTopMoversSection(state))
	sb.WriteString("\n")
	
	// Recent Trades section
	sb.WriteString(renderRecentTradesSection(state))
	
	return sb.String()
}

// renderWalletSection renders wallet/account info
func renderWalletSection(state *TUIState) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("\n%s%s WALLET %s\n", ColorBold, ColorCyan, ColorReset))
	sb.WriteString(strings.Repeat("─", 50) + "\n")
	
	if state.Wallet == nil {
		sb.WriteString(fmt.Sprintf("%sNo wallet data%s\n", ColorDim, ColorReset))
		return sb.String()
	}
	
	w := state.Wallet
	
	// Balance
	sb.WriteString(fmt.Sprintf("Balance:      %s$%.2f%s\n", ColorBold, w.TotalBalance, ColorReset))
	sb.WriteString(fmt.Sprintf("Available:    $%.2f\n", w.AvailableBalance))
	sb.WriteString(fmt.Sprintf("Unrealized:   %s\n", formatPnL(w.UnrealizedPnL)))
	
	// Daily stats
	dailyColor := ColorGreen
	if w.DailyPnL < 0 {
		dailyColor = ColorRed
	}
	sb.WriteString(fmt.Sprintf("Daily PnL:    %s$%.2f (%.2f%%)%s\n", dailyColor, w.DailyPnL, w.DailyPnLPercent, ColorReset))
	
	// Margin
	marginColor := ColorGreen
	if w.MarginRatio > 0.5 {
		marginColor = ColorYellow
	}
	if w.MarginRatio > 0.8 {
		marginColor = ColorRed
	}
	sb.WriteString(fmt.Sprintf("Margin Used:  %s%.1f%%%s\n", marginColor, w.MarginRatio*100, ColorReset))
	
	// Daily trades
	sb.WriteString(fmt.Sprintf("Trades Today: %d (W:%d L:%d)\n", w.DailyTrades, w.DailyWins, w.DailyLosses))
	
	return sb.String()
}

// renderPositionsSection renders open positions
func renderPositionsSection(state *TUIState) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("\n%s%s POSITIONS (%d) %s\n", ColorBold, ColorCyan, len(state.Positions), ColorReset))
	sb.WriteString(strings.Repeat("─", 80) + "\n")
	
	if len(state.Positions) == 0 {
		sb.WriteString(fmt.Sprintf("%sNo open positions%s\n", ColorDim, ColorReset))
		return sb.String()
	}
	
	// Header
	sb.WriteString(fmt.Sprintf("%-12s %-6s %10s %10s %10s %8s %8s\n",
		"SYMBOL", "SIDE", "SIZE", "ENTRY", "MARK", "PNL", "LIQ%"))
	sb.WriteString(strings.Repeat("─", 80) + "\n")
	
	for _, p := range state.Positions {
		sideColor := ColorGreen
		if p.Side == "SHORT" {
			sideColor = ColorRed
		}
		
		pnlColor := ColorGreen
		if p.UnrealizedPnL < 0 {
			pnlColor = ColorRed
		}
		
		liqColor := ColorGreen
		if p.LiquidationDist < 10 {
			liqColor = ColorYellow
		}
		if p.LiquidationDist < 5 {
			liqColor = ColorRed
		}
		
		sb.WriteString(fmt.Sprintf("%-12s %s%-6s%s %10.4f %10.4f %10.4f %s%+8.2f%s %s%7.1f%%%s\n",
			p.Symbol,
			sideColor, p.Side, ColorReset,
			p.Size,
			p.EntryPrice,
			p.MarkPrice,
			pnlColor, p.UnrealizedPnL, ColorReset,
			liqColor, p.LiquidationDist, ColorReset,
		))
	}
	
	return sb.String()
}

// renderTopMoversSection renders top movers
func renderTopMoversSection(state *TUIState) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("\n%s%s TOP MOVERS %s\n", ColorBold, ColorCyan, ColorReset))
	sb.WriteString(strings.Repeat("─", 70) + "\n")
	
	if len(state.TopMovers) == 0 {
		sb.WriteString(fmt.Sprintf("%sNo top movers%s\n", ColorDim, ColorReset))
		return sb.String()
	}
	
	// Header
	sb.WriteString(fmt.Sprintf("%-12s %-15s %10s %12s %8s\n",
		"SYMBOL", "CATEGORY", "CHANGE%", "VOLUME", "SCORE"))
	sb.WriteString(strings.Repeat("─", 70) + "\n")
	
	for i, m := range state.TopMovers {
		if i >= 10 {
			break
		}
		
		changeColor := ColorGreen
		if m.PriceChange < 0 {
			changeColor = ColorRed
		}
		
		selected := ""
		if m.IsSelected {
			selected = fmt.Sprintf("%s►%s ", ColorYellow, ColorReset)
		}
		
		sb.WriteString(fmt.Sprintf("%s%-12s %-15s %s%+9.2f%%%s %11.0f %8.1f\n",
			selected,
			m.Symbol,
			m.Category,
			changeColor, m.PriceChange, ColorReset,
			m.Volume24h,
			m.MomentumScore,
		))
	}
	
	return sb.String()
}

// renderRecentTradesSection renders recent trades
func renderRecentTradesSection(state *TUIState) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("\n%s%s RECENT TRADES %s\n", ColorBold, ColorCyan, ColorReset))
	sb.WriteString(strings.Repeat("─", 90) + "\n")
	
	if len(state.RecentTrades) == 0 {
		sb.WriteString(fmt.Sprintf("%sNo recent trades%s\n", ColorDim, ColorReset))
		return sb.String()
	}
	
	// Header
	sb.WriteString(fmt.Sprintf("%-10s %-12s %-6s %10s %10s %10s %10s\n",
		"TIME", "SYMBOL", "SIDE", "ENTRY", "EXIT", "PNL", "REASON"))
	sb.WriteString(strings.Repeat("─", 90) + "\n")
	
	for i, t := range state.RecentTrades {
		if i >= 10 {
			break
		}
		
		pnlColor := ColorGreen
		if t.PnL < 0 {
			pnlColor = ColorRed
		}
		
		sideColor := ColorGreen
		if t.Side == "SHORT" {
			sideColor = ColorRed
		}
		
		sb.WriteString(fmt.Sprintf("%-10s %-12s %s%-6s%s %10.4f %10.4f %s%+10.2f%s %-10s\n",
			t.ClosedAt.Format("15:04:05"),
			t.Symbol,
			sideColor, t.Side, ColorReset,
			t.EntryPrice,
			t.ExitPrice,
			pnlColor, t.PnL, ColorReset,
			truncateString(t.ExitReason, 10),
		))
	}
	
	return sb.String()
}

// renderBrainLog renders the brain activity log
func renderBrainLog(state *TUIState) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("\n%s%s BRAIN LOG %s", ColorBold, ColorMagenta, ColorReset))
	if state.LastDecision != "" {
		sb.WriteString(fmt.Sprintf(" │ Last: %s (%.0f%% confidence)", state.LastDecision, state.Confidence*100))
	}
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", 100) + "\n")
	
	if len(state.BrainLogs) == 0 {
		sb.WriteString(fmt.Sprintf("%sNo brain activity%s\n", ColorDim, ColorReset))
		return sb.String()
	}
	
	// Show last 5 logs
	start := 0
	if len(state.BrainLogs) > 5 {
		start = len(state.BrainLogs) - 5
	}
	
	for _, log := range state.BrainLogs[start:] {
		sb.WriteString(fmt.Sprintf("%s%s%s\n", ColorDim, log, ColorReset))
	}
	
	return sb.String()
}

// ============================================================================
// Helper Functions
// ============================================================================

func formatPnL(pnl float64) string {
	if pnl >= 0 {
		return fmt.Sprintf("%s+$%.2f%s", ColorGreen, pnl, ColorReset)
	}
	return fmt.Sprintf("%s-$%.2f%s", ColorRed, -pnl, ColorReset)
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// ============================================================================
// Health Check Display
// ============================================================================

// RenderHealthChecks renders health check results
func RenderHealthChecks(health *health.SystemHealth) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("\n%s%s SYSTEM HEALTH CHECK %s\n", ColorBold, ColorCyan, ColorReset))
	sb.WriteString(strings.Repeat("═", 60) + "\n\n")
	
	// Platform info
	sb.WriteString(fmt.Sprintf("Platform: %s/%s (%d CPUs)\n", 
		health.Platform.OS, health.Platform.Arch, health.Platform.NumCPU))
	sb.WriteString(fmt.Sprintf("Go: %s\n\n", health.Platform.GoVersion))
	
	// Overall status
	overallColor := ColorGreen
	if health.Overall == "WARNING" {
		overallColor = ColorYellow
	}
	if health.Overall == "ERROR" {
		overallColor = ColorRed
	}
	sb.WriteString(fmt.Sprintf("Overall Status: %s%s%s\n\n", overallColor, health.Overall, ColorReset))
	
	// Individual checks
	for _, check := range health.Checks {
		statusIcon := "✓"
		statusColor := ColorGreen
		
		if check.Status == "WARNING" {
			statusIcon = "⚠"
			statusColor = ColorYellow
		}
		if check.Status == "ERROR" {
			statusIcon = "✗"
			statusColor = ColorRed
		}
		if check.Status == "UNKNOWN" {
			statusIcon = "?"
			statusColor = ColorDim
		}
		
		sb.WriteString(fmt.Sprintf("%s%s%s %-30s %s (%dms)\n",
			statusColor, statusIcon, ColorReset,
			check.Name,
			check.Message,
			check.Duration.Milliseconds(),
		))
	}
	
	sb.WriteString("\n" + strings.Repeat("═", 60) + "\n")
	
	return sb.String()
}

// RenderStartupBanner renders the startup banner
func RenderStartupBanner() string {
	return fmt.Sprintf(`
%s%s
   ██████╗  ██████╗ ██████╗  ██████╗ ████████╗
  ██╔════╝ ██╔═══██╗██╔══██╗██╔═══██╗╚══██╔══╝
  ██║  ███╗██║   ██║██████╔╝██║   ██║   ██║   
  ██║   ██║██║   ██║██╔══██╗██║   ██║   ██║   
  ╚██████╔╝╚██████╔╝██████╔╝╚██████╔╝   ██║   
   ╚═════╝  ╚═════╝ ╚═════╝  ╚═════╝    ╚═╝   
%s
  %sBinance Futures Perpetual Trading Bot%s
  %sTop Movers • Dynamic Position • Trailing TP%s
  
`, ColorCyan, ColorBold, ColorReset, ColorWhite, ColorReset, ColorDim, ColorReset)
}
