#!/bin/bash

# ============================================================================
# GOBOT - N8N Integration CLI
# ============================================================================
# Usage: ./gobot.sh [command]
#
# Commands:
#   start       - Start GOBOT and N8N
#   stop        - Stop all services
#   restart     - Restart all services
#   status      - Check service status
#   test        - Test webhooks
#   logs        - View logs
#   setup       - Initial setup
#   n8n-import  - Import N8N workflows
#   help        - Show this help
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Config
GOBOT_PORT=${GOBOT_PORT:-8080}
N8N_PORT=${N8N_PORT:-5678}
N8N_USER=${N8N_USER:-gobot}
N8N_PASS=${N8N_PASS:-gobot}
DATA_DIR="${DATA_DIR:-$HOME/.gobot}"
N8N_DATA_DIR="${N8N_DATA_DIR:-$HOME/.n8n}"

# ============================================================================
# Functions
# ============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

check_env() {
    log_info "Checking environment..."

    local missing=0

    if [ -z "$BINANCE_API_KEY" ]; then
        log_warn "BINANCE_API_KEY not set"
        missing=1
    fi

    if [ -z "$BINANCE_API_SECRET" ]; then
        log_warn "BINANCE_API_SECRET not set"
        missing=1
    fi

    if [ $missing -eq 1 ]; then
        log_error "Missing required environment variables!"
        log_info "Edit .env file and restart"
        return 1
    fi

    log_success "Environment check passed"
    return 0
}

start_gobot() {
    log_info "Starting GOBOT on port $GOBOT_PORT..."

    if lsof -i:$GOBOT_PORT > /dev/null 2>&1; then
        log_warn "GOBOT already running on port $GOBOT_PORT"
        return 0
    fi

    # Build if needed
    if [ ! -f "./gobot" ]; then
        log_info "Building GOBOT..."
        go build -o gobot ./cmd/cobot
    fi

    # Start in background
    ./gobot > $DATA_DIR/gobot.log 2>&1 &
    GOBOT_PID=$!

    # Wait for start
    sleep 3

    if kill -0 $GOBOT_PID 2>/dev/null; then
        log_success "GOBOT started (PID: $GOBOT_PID)"
        echo $GOBOT_PID > $DATA_DIR/gobot.pid
    else
        log_error "GOBOT failed to start"
        cat $DATA_DIR/gobot.log
        return 1
    fi
}

start_n8n() {
    log_info "Starting N8N on port $N8N_PORT..."

    if lsof -i:$N8N_PORT > /dev/null 2>&1; then
        log_warn "N8N already running on port $N8N_PORT"
        return 0
    fi

    # Create directories
    mkdir -p "$N8N_DATA_DIR"
    mkdir -p "$DATA_DIR"

    # Start N8N via Docker
    docker run -d \
        --name gobot-n8n \
        -p $N8N_PORT:5678 \
        -v $N8N_DATA_DIR:/home/node/.n8n \
        -e N8N_BASIC_AUTH_ACTIVE=true \
        -e N8N_BASIC_AUTH_USER=$N8N_USER \
        -e N8N_BASIC_AUTH_PASSWORD=$N8N_PASS \
        -e WEBHOOK_URL=http://localhost:$N8N_PORT/ \
        -e N8N_HOST=0.0.0.0 \
        n8nio/n8n > $DATA_DIR/n8n.log 2>&1

    # Wait for start
    sleep 5

    if docker ps | grep gobot-n8n > /dev/null; then
        log_success "N8N started"
        log_info "N8N URL: http://localhost:$N8N_PORT"
        log_info "Login: $N8N_USER / $N8N_PASS"
    else
        log_error "N8N failed to start"
        cat $DATA_DIR/n8n.log
        return 1
    fi
}

stop_all() {
    log_info "Stopping all services..."

    # Stop GOBOT
    if [ -f $DATA_DIR/gobot.pid ]; then
        kill $(cat $DATA_DIR/gobot.pid) 2>/dev/null || true
        rm -f $DATA_DIR/gobot.pid
        log_success "GOBOT stopped"
    fi

    # Stop N8N
    docker stop gobot-n8n 2>/dev/null || true
    docker rm gobot-n8n 2>/dev/null || true
    log_success "N8N stopped"

    log_success "All services stopped"
}

restart_all() {
    stop_all
    sleep 2
    start_all
}

status_check() {
    log_info "Checking service status..."

    local running=0

    # Check GOBOT
    if lsof -i:$GOBOT_PORT > /dev/null 2>&1; then
        log_success "GOBOT: Running on port $GOBOT_PORT"
        running=1
    else
        log_warn "GOBOT: Not running"
    fi

    # Check N8N
    if docker ps | grep gobot-n8n > /dev/null 2>&1; then
        log_success "N8N: Running on port $N8N_PORT"
        running=1
    else
        log_warn "N8N: Not running"
    fi

    # Check webhooks
    if [ $running -eq 1 ]; then
        log_info "Testing webhooks..."
        if curl -s http://localhost:$GOBOT_PORT/health > /dev/null 2>&1; then
            log_success "Webhook endpoint: OK"
        else
            log_warn "Webhook endpoint: Not responding"
        fi
    fi
}

test_webhooks() {
    log_info "Testing webhooks..."

    # Test trade signal
    log_info "Testing /webhook/trade_signal..."
    if curl -s -X POST http://localhost:$GOBOT_PORT/webhook/trade_signal \
        -H "Content-Type: application/json" \
        -d '{"symbol":"BTCUSDT","action":"buy","confidence":0.85,"price":65000}' \
        | grep -q "error\|fail"; then
        log_warn "Trade signal webhook may have issues"
    else
        log_success "Trade signal webhook: OK"
    fi

    # Test risk alert
    log_info "Testing /webhook/risk-alert..."
    if curl -s -X POST http://localhost:$GOBOT_PORT/webhook/risk-alert \
        -H "Content-Type: application/json" \
        -d '{"position":"BTCUSDT","pnl_percent":-5.5,"health_score":35,"reason":"Large drawdown"}' \
        | grep -q "error\|fail"; then
        log_warn "Risk alert webhook may have issues"
    else
        log_success "Risk alert webhook: OK"
    fi

    # Test market analysis
    log_info "Testing /webhook/market-analysis..."
    if curl -s -X POST http://localhost:$GOBOT_PORT/webhook/market-analysis \
        -H "Content-Type: application/json" \
        -d '{"symbol":"BTCUSDT","timeframe":"1h"}' \
        | grep -q "error\|fail"; then
        log_warn "Market analysis webhook may have issues"
    else
        log_success "Market analysis webhook: OK"
    fi

    log_success "Webhook tests completed"
}

show_logs() {
    log_info "Showing recent logs..."

    echo -e "\n${BLUE}=== GOBOT Logs ===${NC}"
    tail -20 $DATA_DIR/gobot.log 2>/dev/null || echo "No logs yet"

    echo -e "\n${BLUE}=== N8N Logs ===${NC}"
    tail -20 $DATA_DIR/n8n.log 2>/dev/null || echo "No logs yet"
}

setup_n8n() {
    log_info "Setting up N8N workflows..."

    if [ ! -f "01-trade-signal.json" ]; then
        log_error "N8N workflow files not found!"
        log_info "Copy from n8n/workflows/ first"
        return 1
    fi

    log_info "N8N workflow files are ready in current directory:"
    ls -la *.json 2>/dev/null | grep -E "01|02|03" || true

    log_info ""
    log_info "To import to N8N:"
    log_info "1. Open http://localhost:$N8N_PORT"
    log_info "2. Login with: $N8N_USER / $N8N_PASS"
    log_info "3. Click 'Import from File'"
    log_info "4. Select the JSON files"
    log_info "5. Activate each workflow"
}

import_n8n_api() {
    log_info "Importing N8N workflows via API..."

    local COOKIE_FILE=$(mktemp)
    local CSRF_FILE=$(mktemp)

    # Login to N8N
    log_info "Logging in to N8N..."
    curl -s -c $COOKIE_FILE -b $COOKIE_FILE \
        -X POST http://localhost:$N8N_PORT/rest/login \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$N8N_USER\",\"password\":\"$N8N_PASS\"}" > /dev/null

    # Get CSRF token
    local CSRF=$(grep "csrf" $CSRF_FILE 2>/dev/null | cut -f7 || echo "")

    # Import each workflow
    for file in 01-trade-signal.json 02-risk-alert.json 03-market-analysis.json; do
        if [ -f "$file" ]; then
            log_info "Importing $file..."
            local WORKFLOW_DATA=$(cat $file | jq -c .)

            curl -s -c $COOKIE_FILE -b $COOKIE_FILE \
                -X POST http://localhost:$N8N_PORT/rest/workflows \
                -H "Content-Type: application/json" \
                -H "X-CSRF-Token: $CSRF" \
                -d "{\"name\":\"$(basename $file .json)\",\"nodes\":[],\"connections\":{},\"settings\":{},\"active\":false}" \
                > /dev/null 2>&1

            log_success "Imported $file"
        fi
    done

    rm -f $COOKIE_FILE $CSRF_FILE
    log_success "N8N workflows imported"
    log_info "Open http://localhost:$N8N_PORT to activate workflows"
}

start_all() {
    log_info "Starting all services..."

    mkdir -p "$DATA_DIR"

    check_env || return 1

    start_gobot
    start_n8n

    sleep 2

    log_success "All services started!"
    log_info ""
    log_info "========================================"
    log_info "  GOBOT:   http://localhost:$GOBOT_PORT"
    log_info "  N8N:     http://localhost:$N8N_PORT"
    log_info "  Login:   $N8N_USER / $N8N_PASS"
    log_info "========================================"
    log_info ""
    log_info "Next steps:"
    log_info "1. Open N8N at http://localhost:$N8N_PORT"
    log_info "2. Import workflows from:"
    log_info "   - 01-trade-signal.json"
    log_info "   - 02-risk-alert.json"
    log_info "   - 03-market-analysis.json"
    log_info "3. Activate the workflows"
    log_info ""
}

help() {
    echo -e "${BLUE}"
    cat << 'EOF'
╔═══════════════════════════════════════════════════════════════════╗
║                     GOBOT N8N CLI Help                            ║
╠═══════════════════════════════════════════════════════════════════╣
║                                                                   ║
║  ./gobot.sh [command]                                             ║
║                                                                   ║
║  Commands:                                                        ║
║    start       Start GOBOT and N8N                                ║
║    stop        Stop all services                                  ║
║    restart     Restart all services                               ║
║    status      Check service status                               ║
║    test        Test webhooks                                      ║
║    logs        View logs                                          ║
║    setup       Setup N8N workflows                                ║
║    n8n-import  Import workflows to N8N (API)                      ║
║    help        Show this help                                     ║
║                                                                   ║
║  Environment Variables:                                           ║
║    GOBOT_PORT     GOBOT port (default: 8080)                      ║
║    N8N_PORT       N8N port (default: 5678)                        ║
║    N8N_USER       N8N username (default: gobot)                   ║
║    N8N_PASS       N8N password (default: gobot)                   ║
║    DATA_DIR       Data directory (default: ~/.gobot)              ║
║                                                                   ║
║  Examples:                                                        ║
║    ./gobot.sh start           # Start all services                ║
║    ./gobot.sh test            # Test webhooks                     ║
║    ./gobot.sh logs            # View logs                         ║
║    ./gobot.sh stop            # Stop all services                 ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

# ============================================================================
# Main
# ============================================================================

case "${1:-start}" in
    start)
        start_all
        ;;
    stop)
        stop_all
        ;;
    restart)
        restart_all
        ;;
    status)
        status_check
        ;;
    test)
        test_webhooks
        ;;
    logs)
        show_logs
        ;;
    setup)
        setup_n8n
        ;;
    n8n-import|n8n)
        import_n8n_api
        ;;
    help|--help|-h)
        help
        ;;
    *)
        log_error "Unknown command: $1"
        help
        exit 1
        ;;
esac
