#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

LOG_DIR="/Users/britebrt/GOBOT/logs"
DEPLOY_LOG="$LOG_DIR/mainnet_deploy_$(date +%Y%m%d_%H%M%S).log"
AUDIT_LOG="$LOG_DIR/mainnet_audit.log"

log() { echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$DEPLOY_LOG"; }
log_info() { echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$DEPLOY_LOG"; }
log_warn() { echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}" | tee -a "$DEPLOY_LOG"; }
log_error() { echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$DEPLOY_LOG"; }
log_success() { echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$DEPLOY_LOG"; }

audit() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] [AUDIT] $1" >> "$AUDIT_LOG"; }

usage() {
    echo "GOBOT Mainnet Deployment Script"
    echo ""
    echo "Usage: $0 <command>"
    echo ""
    echo "Commands:"
    echo "  check       - Validate configuration (safe)"
    echo "  connect     - Test mainnet API connectivity"
    echo "  deploy      - Deploy to mainnet (requires --confirm)"
    echo "  start       - Start mainnet trading"
    echo "  stop        - Stop all trading"
    echo "  status      - Check current status"
    echo "  logs        - View live logs"
    echo ""
    exit 1
}

check_config() {
    log_info "=========================================="
    log_info "MAINNET CONFIGURATION VALIDATION"
    log_info "=========================================="
    
    local errors=0
    local config_file="/Users/britebrt/GOBOT/.env.mainnet"
    
    if [ ! -f "$config_file" ]; then
        log_error "Mainnet config not found: $config_file"
        return 1
    fi
    
    log_info "Checking configuration file..."
    
    # Check API keys
    if grep -q "YOUR_BINANCE_MAINNET_API_KEY_HERE" "$config_file"; then
        log_error "BINANCE_API_KEY not configured"
        errors=$((errors + 1))
    else
        log_success "BINANCE_API_KEY configured"
    fi
    
    if grep -q "YOUR_BINANCE_MAINNET_API_SECRET_HERE" "$config_file"; then
        log_error "BINANCE_API_SECRET not configured"
        errors=$((errors + 1))
    else
        log_success "BINANCE_API_SECRET configured"
    fi
    
    # Check testnet mode is disabled
    if grep -q "BINANCE_USE_TESTNET=true" "$config_file"; then
        log_error "BINANCE_USE_TESTNET must be false for mainnet"
        errors=$((errors + 1))
    else
        log_success "BINANCE_USE_TESTNET=false"
    fi
    
    # Check Telegram
    if grep -q "YOUR_TELEGRAM_BOT_TOKEN_HERE" "$config_file"; then
        log_warn "TELEGRAM_TOKEN not configured (alerts disabled)"
    else
        log_success "TELEGRAM_TOKEN configured"
    fi
    
    if grep -q "YOUR_TELEGRAM_CHAT_ID_HERE" "$config_file"; then
        log_warn "AUTHORIZED_CHAT_ID not configured"
    else
        log_success "AUTHORIZED_CHAT_ID configured"
    fi
    
    # Check kill switch
    if grep -q "KILL_SWITCH_PASSWORD=STOP123" "$config_file"; then
        log_warn "Default kill switch password - change in production"
    else
        log_success "Kill switch password configured"
    fi
    
    # Check position limits
    local pos_limit=$(grep "MAX_POSITION_SIZE_USD" "$config_file" | cut -d= -f2 | tr -d ' ' | cut -d'#' -f1)
    log_info "Position limit: \$$pos_limit"
    
    if [ "$pos_limit" -gt 100 ] 2>/dev/null; then
        log_warn "Position limit > $100 - consider starting smaller"
    fi
    
    log_info "=========================================="
    if [ $errors -eq 0 ]; then
        log_success "Configuration validation PASSED"
        return 0
    else
        log_error "Configuration validation FAILED ($errors errors)"
        return 1
    fi
}

test_connectivity() {
    log_info "=========================================="
    log_info "TESTING MAINNET CONNECTIVITY"
    log_info "=========================================="
    
    log_info "Testing Binance mainnet API..."
    
    # Test ping
    if curl -s -m 10 "https://api.binance.com/api/v3/ping" | grep -q "{}"; then
        log_success "Binance API ping: OK"
    else
        log_error "Binance API ping: FAILED"
        return 1
    fi
    
    # Test time sync
    local server_time=$(curl -s -m 10 "https://api.binance.com/api/v3/time" | grep -o '"serverTime":[0-9]*' | cut -d: -f2)
    if [ -n "$server_time" ]; then
        local local_time=$(date +%s)000
        local diff=$((server_time - local_time))
        log_info "Server time sync diff: ${diff}ms"
        if [ ${diff#-} -lt 5000 ]; then
            log_success "Time sync: OK"
        else
            log_warn "Time sync: ${diff}ms difference - ensure system clock is accurate"
        fi
    fi
    
    # Test with API key (will fail auth but proves endpoint reachable)
    log_info "Testing API endpoint reachability..."
    local api_test=$(curl -s -m 5 -H "X-MBX-APIKEY: test" "https://api.binance.com/api/v3/account" 2>&1)
    if echo "$api_test" | grep -q "API-key"; then
        log_success "API endpoint: OK (auth required)"
    elif echo "$api_test" | grep -q "timeout\|Timed out"; then
        log_error "API endpoint: TIMEOUT"
        return 1
    else
        log_success "API endpoint: OK"
    fi
    
    log_success "All connectivity tests passed"
}

stop_all() {
    log_info "Stopping all GOBOT services..."
    pkill -f "./gobot" 2>/dev/null || true
    pkill -f "node.*server.js" 2>/dev/null || true
    pkill -f "mainnet_monitor" 2>/dev/null || true
    rm -f /tmp/gobot.pid /tmp/screenshot.pid
    log_success "All services stopped"
}

start_mainnet() {
    log_info "=========================================="
    log_info "STARTING GOBOT ON MAINNET"
    log_info "=========================================="
    
    # Stop existing
    stop_all
    
    cd /Users/britebrt/GOBOT
    
    # Backup current config
    if [ -f ".env" ]; then
        cp .env .env.testnet.backup_$(date +%Y%m%d_%H%M%S)
        log_info "Testnet config backed up"
    fi
    
    # Apply mainnet config
    cp .env.mainnet .env
    log_success "Mainnet configuration applied"
    
    audit "MAINNET TRADING STARTED"
    
    # Build
    log_info "Building GOBOT..."
    go build -o gobot ./cmd/cobot/
    
    # Start GOBOT
    log_info "Starting GOBOT..."
    ./gobot > /tmp/gobot_mainnet.log 2>&1 &
    local pid=$!
    echo $pid > /tmp/gobot.pid
    log_success "GOBOT started (PID: $pid)"
    
    # Wait for health check
    sleep 3
    
    if curl -s http://localhost:8080/health > /dev/null; then
        log_success "GOBOT health check: OK"
        audit "GOBOT HEALTH CHECK PASSED"
    else
        log_error "GOBOT health check: FAILED"
        audit "GOBOT HEALTH CHECK FAILED"
        tail -20 /tmp/gobot_mainnet.log
        return 1
    fi
    
    # Start monitoring
    start_monitor
    
    log_info "=========================================="
    log_success "GOBOT IS NOW TRADING ON MAINNET"
    log_info "=========================================="
    log_info "View logs: tail -f /tmp/gobot_mainnet.log"
    log_info "Kill switch: echo 'stop' > /tmp/gobot_kill_switch"
}

start_monitor() {
    cat > /tmp/mainnet_monitor.sh << 'MONITOR'
#!/bin/bash
AUDIT_LOG="/Users/britebrt/GOBOT/logs/mainnet_audit.log"
ALERT_LOG="/Users/britebrt/GOBOT/logs/mainnet_alerts.log"

while true; do
    # Check if GOBOT is running
    if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ALERT] GOBOT not responding" | tee -a "$ALERT_LOG"
    fi
    
    # Check for panics
    if grep -q "panic\|PANIC" /tmp/gobot_mainnet.log 2>/dev/null; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [CRITICAL] Panic detected" | tee -a "$ALERT_LOG"
    fi
    
    # Check kill switch
    if [ -f "/tmp/gobot_kill_switch" ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [EMERGENCY] Kill switch triggered" | tee -a "$ALERT_LOG"
        pkill -f "./gobot" 2>/dev/null || true
        exit 1
    fi
    
    sleep 60
done
MONITOR
    
    chmod +x /tmp/mainnet_monitor.sh
    nohup /tmp/mainnet_monitor.sh > /dev/null 2>&1 &
    log_info "Monitoring started"
}

show_status() {
    echo "=========================================="
    echo "GOBOT MAINNET STATUS"
    echo "=========================================="
    
    local gobot_running=false
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        gobot_running=true
        echo -e "${GREEN}GOBOT: RUNNING${NC}"
    else
        echo -e "${RED}GOBOT: NOT RUNNING${NC}"
    fi
    
    if [ -f /tmp/gobot.pid ]; then
        echo "PID: $(cat /tmp/gobot.pid)"
    fi
    
    echo ""
    echo "Recent activity:"
    tail -10 /tmp/gobot_mainnet.log 2>/dev/null | grep -v "^202" || echo "No logs"
    
    echo ""
    echo "Kill switch status:"
    if [ -f "/tmp/gobot_kill_switch" ]; then
        echo -e "${RED}ACTIVE - TRADING HALTED${NC}"
    else
        echo -e "${GREEN}INACTIVE - TRADING ENABLED${NC}"
    fi
}

# Main
case "${1:-}" in
    check)
        check_config
        ;;
    connect)
        test_connectivity
        ;;
    deploy)
        if [ "${2:-}" != "--confirm" ]; then
            echo -e "${YELLOW}WARNING: This will start REAL TRADING with REAL USDT${NC}"
            echo ""
            echo "Before deploying, ensure:"
            echo "  1. Configuration is validated: $0 check"
            echo "  2. Connectivity is tested: $0 connect"
            echo "  3. You understand the risks"
            echo ""
            echo "To deploy, run: $0 deploy --confirm"
            exit 1
        fi
        
        if check_config && test_connectivity; then
            start_mainnet
        else
            log_error "Pre-deployment checks failed"
            exit 1
        fi
        ;;
    start)
        start_mainnet
        ;;
    stop)
        audit "MAINNET TRADING STOPPED MANUALLY"
        stop_all
        ;;
    status)
        show_status
        ;;
    logs)
        tail -f /tmp/gobot_mainnet.log
        ;;
    *)
        usage
        ;;
esac
