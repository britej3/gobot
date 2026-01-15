#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

LOG_DIR="/Users/britebrt/GOBOT/logs"
DEPLOY_LOG="$LOG_DIR/mainnet_deploy_$(date +%Y%m%d_%H%M%S).log"
TRADES_LOG="$LOG_DIR/trades_mainnet_$(date +%Y%m%d).log"
PNL_LOG="$LOG_DIR/pnl_$(date +%Y%m%d).csv"

mkdir -p "$LOG_DIR"

log() { echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$DEPLOY_LOG"; }
log_info() { echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$DEPLOY_LOG"; }
log_warn() { echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}" | tee -a "$DEPLOY_LOG"; }
log_error() { echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$DEPLOY_LOG"; }
log_success() { echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$DEPLOY_LOG"; }
log_trade() { echo "[$(date '+%Y-%m-%d %H:%M:%S')],$1" >> "$TRADES_LOG"; }
log_pnl() { echo "[$(date '+%Y-%m-%d %H:%M:%S')],$1" >> "$PNL_LOG"; }

audit() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] [AUDIT] $1" >> "$AUDIT_LOG"; }

header() {
    echo ""
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                  GOBOT MAINNET PRODUCTION DEPLOYER                  ║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

usage() {
    header
    echo "Usage: $0 <command>"
    echo ""
    echo "Commands:"
    echo "  setup       - Configure API keys and settings"
    echo "  check       - Validate configuration"
    echo "  connect     - Test mainnet connectivity"
    echo "  status      - Check system status"
    echo "  start       - Start live trading"
    echo "  stop        - Stop all trading"
    echo "  monitor     - Live P&L monitoring"
    echo "  report      - Generate P&L report"
    echo ""
    exit 1
}

header

# ============================================================================
# SETUP - Configure API Keys
# ============================================================================
setup() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                         CONFIGURATION SETUP                         ║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    echo "Enter your Binance Mainnet API credentials:"
    echo ""
    read -p "BINANCE API KEY: " api_key
    read -s -p "BINANCE API SECRET: " api_secret
    echo ""
    
    echo ""
    echo "Enter Telegram credentials (optional):"
    read -p "TELEGRAM BOT TOKEN: " tg_token
    read -p "TELEGRAM CHAT ID: " tg_chat
    echo ""
    
    echo "Settings:"
    read -p "Initial Capital (default: 100): " capital
    capital=${capital:-100}
    read -p "Max Position Size (default: 10): " max_pos
    max_pos=${max_pos:-10}
    echo ""
    
    # Update .env.mainnet.production
    sed -i '' "s|YOUR_BINANCE_MAINNET_API_KEY_HERE|$api_key|g" .env.mainnet.production
    sed -i '' "s|YOUR_BINANCE_MAINNET_API_SECRET_HERE|$api_secret|g" .env.mainnet.production
    sed -i '' "s|YOUR_TELEGRAM_BOT_TOKEN_HERE|$tg_token|g" .env.mainnet.production
    sed -i '' "s|YOUR_TELEGRAM_CHAT_ID_HERE|$tg_chat|g" .env.mainnet.production
    sed -i '' "s|^INITIAL_CAPITAL_USD=100|INITIAL_CAPITAL_USD=$capital|g" .env.mainnet.production
    sed -i '' "s|^MAX_POSITION_SIZE_USD=10|MAX_POSITION_SIZE_USD=$max_pos|g" .env.mainnet.production
    
    log_success "Configuration saved to .env.mainnet.production"
    
    echo ""
    echo "To validate: $0 check"
    echo "To start trading: $0 start"
}

# ============================================================================
# CHECK - Validate Configuration
# ============================================================================
check_config() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                    CONFIGURATION VALIDATION                         ║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    local config_file="/Users/britebrt/GOBOT/.env.mainnet.production"
    local errors=0
    
    # Check file exists
    if [ ! -f "$config_file" ]; then
        log_error "Config file not found: $config_file"
        echo "Run: $0 setup first"
        return 1
    fi
    
    # Check API keys
    if grep -q "YOUR_BINANCE_MAINNET_API_KEY_HERE" "$config_file"; then
        log_error "BINANCE_API_KEY not configured"
        errors=$((errors + 1))
    else
        log_success "BINANCE_API_KEY: configured"
    fi
    
    if grep -q "YOUR_BINANCE_MAINNET_API_SECRET_HERE" "$config_file"; then
        log_error "BINANCE_API_SECRET not configured"
        errors=$((errors + 1))
    else
        log_success "BINANCE_API_SECRET: configured"
    fi
    
    # Check mainnet mode
    if grep -q "BINANCE_USE_TESTNET=true" "$config_file"; then
        log_error "Must set BINANCE_USE_TESTNET=false for mainnet"
        errors=$((errors + 1))
    else
        log_success "BINANCE_USE_TESTNET=false (mainnet mode)"
    fi
    
    # Check auto-execution
    if grep -q "AUTO_EXECUTE_TRADES=true" "$config_file"; then
        log_success "AUTO_EXECUTE_TRADES=true (will execute real trades)"
    else
        log_warn "AUTO_EXECUTE_TRADES=false (read-only mode)"
    fi
    
    # Check capital settings
    capital=$(grep "INITIAL_CAPITAL_USD" "$config_file" | cut -d= -f2 | tr -d ' ')
    max_pos=$(grep "MAX_POSITION_SIZE_USD" "$config_file" | cut -d= -f2 | tr -d ' ')
    
    echo ""
    echo "Capital Settings:"
    echo "  Initial Capital: \$$capital"
    echo "  Max Position: \$$max_pos"
    
    if [ "$capital" -le 50 ]; then
        log_warn "Initial capital < $50 - very conservative"
    elif [ "$capital" -le 200 ]; then
        log_info "Initial capital \$$capital - conservative"
    else
        log_warn "Initial capital \$$capital - consider starting smaller"
    fi
    
    echo ""
    if [ $errors -eq 0 ]; then
        log_success "Configuration validation: PASSED"
        return 0
    else
        log_error "Configuration validation: FAILED ($errors errors)"
        return 1
    fi
}

# ============================================================================
# CONNECT - Test Mainnet Connectivity
# ============================================================================
test_connectivity() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                    MAINNET CONNECTIVITY TEST                        ║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    log_info "Testing Binance API connectivity..."
    
    # Test ping
    if curl -s -m 10 "https://api.binance.com/api/v3/ping" | grep -q "{}"; then
        log_success "Binance API ping: OK"
    else
        log_error "Binance API ping: FAILED"
        return 1
    fi
    
    # Test time sync
    server_time=$(curl -s -m 10 "https://api.binance.com/api/v3/time" | grep -o '"serverTime":[0-9]*' | cut -d: -f2)
    if [ -n "$server_time" ]; then
        local_time=$(date +%s)000
        diff=$((server_time - local_time))
        log_info "Time sync diff: ${diff}ms"
        if [ ${diff#-} -lt 5000 ]; then
            log_success "Time sync: OK"
        else
            log_warn "Time diff: ${diff}ms - ensure clock is accurate"
        fi
    fi
    
    # Test with API key (limited test)
    api_key=$(grep "BINANCE_API_KEY=" .env.mainnet.production | cut -d= -f2 | tr -d ' ')
    if [ -n "$api_key" ] && [ "$api_key" != "YOUR_BINANCE_MAINNET_API_KEY_HERE" ]; then
        log_info "Testing account endpoint..."
        account_test=$(curl -s -m 5 -H "X-MBX-APIKEY: $api_key" "https://api.binance.com/api/v3/account" 2>&1)
        if echo "$account_test" | grep -q "balances\|commission"; then
            log_success "API authentication: OK"
        elif echo "$account_test" | grep -q "API-key"; then
            log_warn "API authentication: Invalid key format"
        else
            log_info "API endpoint reachable"
        fi
    fi
    
    log_success "Connectivity test complete"
}

# ============================================================================
# START - Deploy to Mainnet with Auto-Execution
# ============================================================================
start_mainnet() {
    header
    echo -e "${RED}╔════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║           ⚠️  LIVE TRADING - REAL MONEY AT RISK ⚠️                  ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "This will start GOBOT with ${GREEN}REAL TRADE EXECUTION${NC} on mainnet."
    echo ""
    echo "Configuration:"
    grep "INITIAL_CAPITAL_USD\|MAX_POSITION_SIZE_USD\|AUTO_EXECUTE_TRADES" .env.mainnet.production | head -3
    echo ""
    
    echo -e "${YELLOW}Type 'YES' to confirm live trading:${NC} "
    read -r confirmation
    if [ "$confirmation" != "YES" ]; then
        echo "Cancelled."
        exit 0
    fi
    
    # Stop existing
    log_info "Stopping any existing services..."
    pkill -f "./gobot" 2>/dev/null || true
    pkill -f "mainnet_monitor" 2>/dev/null || true
    sleep 2
    
    # Backup current config
    if [ -f ".env" ]; then
        cp .env .env.backup_$(date +%Y%m%d_%H%M%S)
    fi
    
    # Apply mainnet config
    cp .env.mainnet.production .env
    log_success "Production configuration applied"
    
    # Initialize P&L log
    echo "timestamp,symbol,action,entry_price,exit_price,quantity,pnl,pnl_percent,status" > "$TRADES_LOG"
    echo "timestamp,capital,pnl_daily,pnl_total,win_count,loss_count" > "$PNL_LOG"
    
    # Build
    log_info "Building GOBOT..."
    go build -o gobot ./cmd/cobot/
    
    # Start GOBOT
    log_info "Starting GOBOT (live trading)..."
    ./gobot > /tmp/gobot_mainnet.log 2>&1 &
    pid=$!
    echo $pid > /tmp/gobot.pid
    
    sleep 3
    
    if curl -s http://localhost:8080/health > /dev/null; then
        log_success "GOBOT started (PID: $pid)"
        audit "MAINNET TRADING STARTED - PID: $pid"
    else
        log_error "GOBOT failed to start"
        tail -20 /tmp/gobot_mainnet.log
        return 1
    fi
    
    # Start monitoring
    start_monitor
    
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║               ✅ LIVE TRADING IS NOW ACTIVE                         ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Live Trading Status:"
    echo "  PID: $pid"
    echo "  Capital: $(grep INITIAL_CAPITAL .env | cut -d= -f2)"
    echo "  Max Position: $(grep MAX_POSITION .env | cut -d= -f2)"
    echo ""
    echo "Commands:"
    echo "  Check status: $0 status"
    echo "  View P&L: $0 monitor"
    echo "  Stop trading: $0 stop"
    echo ""
    echo "Log files:"
    echo "  Trading: $TRADES_LOG"
    echo "  P&L: $PNL_LOG"
    echo "  System: /tmp/gobot_mainnet.log"
}

# ============================================================================
# START MONITORING
# ============================================================================
start_monitor() {
    cat > /tmp/mainnet_monitor.sh << 'MONITOR'
#!/bin/bash
TRADES_LOG="/Users/britebrt/GOBOT/logs/trades_mainnet_$(date +%Y%m%d).log"
PNL_LOG="/Users/britebrt/GOBOT/logs/pnl_$(date +%Y%m%d).csv"
ALERT_LOG="/Users/britebrt/GOBOT/logs/mainnet_alerts.log"
AUDIT_LOG="/Users/britebrt/GOBOT/logs/mainnet_audit.log"

capital=$(grep "INITIAL_CAPITAL_USD" /Users/britebrt/GOBOT/.env | cut -d= -f2 | tr -d ' ')
total_pnl=0
win_count=0
loss_count=0

while true; do
    # Check GOBOT health
    if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [CRITICAL] GOBOT not responding" | tee -a "$ALERT_LOG"
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [AUDIT] GOBOT_UNHEALTHY" >> "$AUDIT_LOG"
    fi
    
    # Check for new trades
    if [ -f "$TRADES_LOG" ]; then
        new_trades=$(tail -5 "$TRADES_LOG" | grep -v "timestamp" | wc -l)
        if [ "$new_trades" -gt 0 ]; then
            # Calculate P&L
            total_pnl=0
            win_count=0
            loss_count=0
            while IFS= read -r line; do
                if echo "$line" | grep -q "WIN"; then
                    win_count=$((win_count + 1))
                elif echo "$line" | grep -q "LOSS"; then
                    loss_count=$((loss_count + 1))
                fi
            done < <(tail -20 "$TRADES_LOG")
        fi
    fi
    
    # Check kill switch
    if [ -f "/tmp/gobot_kill_switch" ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [EMERGENCY] Kill switch triggered" | tee -a "$ALERT_LOG"
        pkill -f "./gobot" 2>/dev/null || true
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [AUDIT] KILL_SWITCH_TRIGGERED" >> "$AUDIT_LOG"
        exit 1
    fi
    
    # Daily P&L summary
    pnl_daily=$(tail -100 "$TRADES_LOG" 2>/dev/null | grep -c "WIN" || echo "0")
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [STATUS] W:$win_count L:$loss_count" >> "$ALERT_LOG" 2>/dev/null || true
    
    sleep 60
done
MONITOR
    
    chmod +x /tmp/mainnet_monitor.sh
    nohup /tmp/mainnet_monitor.sh > /dev/null 2>&1 &
    log_info "Monitoring started"
}

# ============================================================================
# STATUS - Check Current Status
# ============================================================================
show_status() {
    header
    echo "GOBOT MAINNET STATUS"
    echo ""
    
    # Check if running
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        status="🟢 RUNNING"
        if [ -f /tmp/gobot.pid ]; then
            pid=$(cat /tmp/gobot.pid)
            echo "  PID: $pid"
        fi
    else
        status="🔴 NOT RUNNING"
    fi
    
    echo -e "Status: ${status}"
    echo ""
    
    # Capital info
    capital=$(grep "INITIAL_CAPITAL_USD" .env 2>/dev/null | cut -d= -f2 || echo "N/A")
    max_pos=$(grep "MAX_POSITION_SIZE_USD" .env 2>/dev/null | cut -d= -f2 || echo "N/A")
    
    echo "Capital Configuration:"
    echo "  Initial Capital: \$$capital"
    echo "  Max Position: \$$max_pos"
    echo ""
    
    # P&L summary
    if [ -f "$TRADES_LOG" ]; then
        wins=$(grep -c "WIN\|profit" "$TRADES_LOG" 2>/dev/null || echo "0")
        losses=$(grep -c "LOSS\|loss" "$TRADES_LOG" 2>/dev/null || echo "0")
        total=$((wins + losses))
        if [ $total -gt 0 ]; then
            win_rate=$((wins * 100 / total))
            echo "Trading Summary:"
            echo "  Total Trades: $total"
            echo "  Wins: $wins"
            echo "  Losses: $losses"
            echo "  Win Rate: ${win_rate}%"
        fi
    fi
    
    echo ""
    echo "Recent Activity:"
    tail -10 /tmp/gobot_mainnet.log 2>/dev/null | grep -v "^202" | head -5 || echo "No recent activity"
    
    echo ""
    echo "Kill Switch:"
    if [ -f "/tmp/gobot_kill_switch" ]; then
        echo -e "  ${RED}ACTIVE - TRADING HALTED${NC}"
    else
        echo -e "  ${GREEN}INACTIVE - TRADING ENABLED${NC}"
    fi
}

# ============================================================================
# MONITOR - Live P&L Monitoring
# ============================================================================
monitor() {
    header
    echo "LIVE P&L MONITORING"
    echo "Press Ctrl+C to exit"
    echo ""
    
    while true; do
        clear
        echo -e "${CYAN}╔════════════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${CYAN}║                       GOBOT LIVE MONITOR                           ║${NC}"
        echo -e "${CYAN}╚════════════════════════════════════════════════════════════════════╝${NC}"
        echo ""
        
        if [ -f "$TRADES_LOG" ]; then
            echo "Recent Trades:"
            tail -10 "$TRADES_LOG" | grep -v "timestamp" | tail -5
            echo ""
            
            wins=$(grep -c "WIN" "$TRADES_LOG" 2>/dev/null || echo "0")
            losses=$(grep -c "LOSS" "$TRADES_LOG" 2>/dev/null || echo "0")
            total=$((wins + losses))
            
            if [ $total -gt 0 ]; then
                win_rate=$((wins * 100 / total))
                echo "Statistics:"
                echo "  Total Trades: $total"
                echo "  Wins: $wins | Losses: $losses"
                echo "  Win Rate: ${win_rate}%"
            fi
        else
            echo "No trades yet..."
        fi
        
        echo ""
        echo "System Status:"
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            echo -e "  GOBOT: ${GREEN}RUNNING${NC}"
        else
            echo -e "  GOBOT: ${RED}STOPPED${NC}"
        fi
        
        sleep 5
    done
}

# ============================================================================
# STOP - Stop All Trading
# ============================================================================
stop_all() {
    header
    log_info "Stopping all trading..."
    
    pkill -f "./gobot" 2>/dev/null || true
    pkill -f "mainnet_monitor" 2>/dev/null || true
    rm -f /tmp/gobot.pid
    
    log_success "All trading stopped"
    echo ""
    echo "To restart: $0 start"
}

# ============================================================================
# REPORT - Generate P&L Report
# ============================================================================
report() {
    header
    echo "P&L REPORT - $(date)"
    echo ""
    
    if [ -f "$TRADES_LOG" ]; then
        echo "Trade History:"
        cat "$TRADES_LOG"
        echo ""
        
        wins=$(grep -c "WIN" "$TRADES_LOG" 2>/dev/null || echo "0")
        losses=$(grep -c "LOSS" "$TRADES_LOG" 2>/dev/null || echo "0")
        total=$((wins + losses))
        
        if [ $total -gt 0 ]; then
            win_rate=$((wins * 100 / total))
            echo "Summary:"
            echo "  Total Trades: $total"
            echo "  Wins: $wins ($win_rate%)"
            echo "  Losses: $losses"
        fi
    else
        echo "No trade history found."
    fi
}

# Main
case "${1:-}" in
    setup) setup ;;
    check) check_config ;;
    connect) test_connectivity ;;
    start) start_mainnet ;;
    status) show_status ;;
    monitor) monitor ;;
    stop) stop_all ;;
    report) report ;;
    *) usage ;;
esac
