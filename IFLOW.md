# GOBOT Project Documentation

## Project Overview

GOBOT is an automated cryptocurrency trading bot system developed in Go, focusing on meme coin trading on the Binance platform. The project features a modular architecture, integrates multiple AI models for market analysis and trading decisions, and supports extensibility through N8N workflow automation.

### Core Features

- **Automated Trading**: Supports automated trading on both Binance mainnet and testnet
- **AI-Driven Analysis**: Integrates multiple free AI providers (Groq/Kimi-K2, Llama 3.3 70B, Gemini 1.5 Flash)
- **Modular Architecture**: Pluggable trading strategies, coin selectors, and executors
- **Risk Management**: Built-in stop-loss, take-profit, position management, and other risk controls
- **Anti-Detection Mode**: Supports request jitter, user agent rotation, and other anti-detection features
- **N8N Integration**: Complex trading logic through N8N workflow automation
- **QuantCrawler Integration**: Automated market analysis data retrieval from QuantCrawler using Puppeteer
- **Telegram Notifications**: Real-time trading signals, risk alerts, and system status updates
- **SimpleMem Memory System**: Python-based long-term memory management with dialogue history and semantic retrieval
- **Ollama Model Support**: Supports CogneeBrain and LiquidAIBrain Ollama models

### Tech Stack

- **Backend**: Go 1.25.4
- **Frontend**: Next.js (agents.md directory)
- **Workflow Automation**: N8N
- **Browser Automation**: Puppeteer 21.0+
- **Containerization**: Docker & Docker Compose
- **Database**: Write-Ahead Log (WAL) + In-memory cache + Vector storage
- **Messaging**: Telegram Bot API
- **AI/LLM**: Groq API (Kimi-K2, Llama 3.3), Gemini API, Ollama (local models)
- **Python Memory System**: SimpleMem (semantic lossless compression, hybrid retrieval)

### Project Architecture

```
GOBOT/
├── cmd/                    # Command-line tool entry points
│   ├── analyzer/           # Market analyzer
│   ├── cobot/              # Main trading bot
│   ├── cognee/             # Cognitive engine
│   ├── gobot-engine/       # Trading engine
│   ├── screener_assets/    # Asset screener
│   ├── screener_demo/      # Screener demo
│   ├── test_jitter/        # Jitter testing
│   └── tester/             # Testing tools
├── domain/                # Domain models and interface definitions
│   ├── asset/              # Asset models
│   ├── automation/         # Automation interfaces
│   ├── errors/             # Error type definitions
│   ├── executor/           # Executor interfaces
│   ├── llm/                # LLM integration interfaces
│   ├── market/             # Market data models
│   ├── platform/           # Platform engine
│   ├── selector/           # Coin selector interfaces
│   ├── strategy/           # Trading strategy interfaces
│   └── trade/              # Trading models
├── infra/                 # Infrastructure
│   ├── binance/            # Binance API client
│   ├── cache/              # Cache layer
│   ├── llm/                # LLM services
│   ├── notify/             # Notification services
│   └── storage/            # Storage layer (WAL)
├── internal/              # Internal components
│   ├── agent/              # AI agent
│   ├── alerting/           # Alerting system
│   ├── auditor/            # Audit system
│   ├── brain/              # Decision engine
│   ├── health/             # Health checks
│   ├── market/             # Market data
│   ├── memory/             # Memory management
│   ├── monitoring/         # Monitoring
│   ├── platform/           # Platform management
│   ├── position/           # Position management
│   ├── risk/               # Risk management
│   ├── startup/            # Startup process
│   ├── striker/            # Executor
│   └── ui/                 # User interface
├── memory/                # Python memory management system (SimpleMem)
│   ├── core/               # Core logic
│   ├── database/           # Vector storage
│   ├── models/             # Data models
│   └── utils/              # Utility functions
├── pkg/                   # Reusable packages
├── config/                # Configuration files
│   ├── config.go           # Configuration loading
│   ├── config.yaml         # Production configuration
│   ├── llm.go              # LLM configuration
│   └── production.go       # Production environment configuration
├── n8n/                   # N8N workflows and scripts
│   ├── workflows/          # Workflow definitions
│   ├── scripts/            # Puppeteer scripts
│   └── SETUP_GUIDE.md      # N8N setup guide
├── agents.md/             # Next.js agent interface
├── docs/                  # Documentation
├── logs/                  # Log directory
├── scripts/               # Script tools
├── state/                 # State persistence
├── tasks/                 # Task management
└── test/                  # Test files
```

## Build and Run

### Requirements

- Go 1.25+
- Python 3.14+ (for SimpleMem memory system)
- Docker & Docker Compose
- Node.js 20+ (for N8N and Puppeteer)
- Binance API credentials
- Ollama 0.13.3+ (optional, for local models)

### Quick Start

#### Method 1: Using Startup Scripts (Recommended)

```bash
# 1. Configure environment variables
cp .env.example .env
nano .env  # Add API keys

# 2. Start all services (N8N + QuantCrawler)
./start-all.sh

# 3. Start GOBOT
./gobot.sh start

# Or use mainnet deployment script
./mainnet-production.sh start
```

#### Method 2: Using Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

#### Method 3: Manual Build

```bash
# 1. Build GOBOT
go build -o gobot ./cmd/cobot

# 2. Run
./gobot
```

### Configuration

Main configuration is in `config/config.yaml`, environment variables are set in `.env`.

#### Required Environment Variables

```bash
# Binance API
BINANCE_API_KEY=your_api_key
BINANCE_API_SECRET=your_secret
BINANCE_USE_TESTNET=false  # Recommended to test on testnet first

# AI Providers
GROQ_API_KEY=your_groq_key
GEMINI_API_KEY=your_gemini_key

# N8N
N8N_BASE_URL=http://localhost:5678
N8N_WEBHOOK_USER=gobot
N8N_WEBHOOK_PASS=secure_password

# Telegram
TELEGRAM_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id

# QuantCrawler
QUANTCRAWLER_EMAIL=your_email@gmail.com
QUANTCRAWLER_PASSWORD=your_16_char_app_password
```

### Service Ports

| Service | Port | Description |
|---------|------|-------------|
| GOBOT | 8080 | Main trading bot API |
| N8N | 5678 | Workflow automation platform |
| QuantCrawler | 3456 | Puppeteer automation service |
| Chrome | 3000 | Headless browser service |

### Common Commands

```bash
# Start GOBOT
./gobot.sh start

# Stop all services
./gobot.sh stop

# Check status
./gobot.sh status

# Test webhooks
./gobot.sh test

# View logs
./gobot.sh logs

# Import N8N workflows
./gobot.sh n8n-import

# Mainnet deployment
./mainnet-production.sh setup   # Configure
./mainnet-production.sh start   # Start
./mainnet-production.sh status  # Status
./mainnet-production.sh monitor # Monitor
./mainnet-production.sh stop    # Stop
```

### Testing

#### Test GOBOT Webhooks

```bash
# Trading signal test
curl -X POST http://localhost:8080/webhook/trade_signal \
  -H "Content-Type: application/json" \
  -d '{"symbol":"BTCUSDT","action":"buy","confidence":0.85,"price":65000,"reason":"RSI oversold"}'

# Risk alert test
curl -X POST http://localhost:8080/webhook/risk-alert \
  -H "Content-Type: application/json" \
  -d '{"position":"BTCUSDT","pnl_percent":-5.5,"health_score":35,"reason":"Large drawdown"}'

# Market analysis test
curl -X POST http://localhost:8080/webhook/market-analysis \
  -H "Content-Type: application/json" \
  -d '{"symbol":"BTCUSDT","timeframe":"1h"}'
```

#### Test QuantCrawler

```bash
curl -X POST http://localhost:3456/webhook \
  -H "Content-Type: application/json" \
  -d '{"symbol":"1000PEPEUSDT","account_balance":1000}'
```

#### Test N8N Workflows

```bash
curl -X POST http://localhost:5678/webhook/quantcrawler-analysis \
  -H "Content-Type: application/json" \
  -d '{"symbol":"1000PEPEUSDT","account_balance":1000}'
```

## Development Guidelines

### Code Structure

- **Domain-Driven Design**: Use `domain/` directory for interface and type definitions
- **Dependency Injection**: Inject dependencies through constructors
- **Interface-First**: Define clear interfaces for easy testing and replacement
- **Error Handling**: Use error types defined in `domain/errors/`

### Trading Strategy Development

To add a new trading strategy:

1. Create a new directory under `services/strategy/`
2. Implement the `Strategy` interface defined in `domain/strategy/strategy.go`
3. Register the strategy in the main program

```go
// Example: Implement custom strategy
type MyStrategy struct {
    cfg strategy.StrategyConfig
}

func (s *MyStrategy) Type() strategy.StrategyType {
    return strategy.StrategyCustom
}

func (s *MyStrategy) ShouldEnter(ctx context.Context, market trade.MarketData) (bool, string, error) {
    // Implement entry logic
    return true, "buy signal", nil
}
```

### Coin Selector Development

To add a new coin selector:

1. Create a new directory under `services/selector/`
2. Implement the `Selector` interface defined in `domain/selector/selector.go`

### Executor Development

To add a new executor:

1. Create a new directory under `services/executor/`
2. Implement the `Executor` interface defined in `domain/executor/executor.go`

### N8N Workflow Integration

1. Create workflow in N8N interface
2. Configure Webhook node to receive GOBOT events
3. Export workflow as JSON file
4. Place file in `n8n/workflows/` directory
5. Use `./gobot.sh n8n-import` to import workflow

### Testing Guidelines

- Use Go standard testing framework
- Write unit tests for each component
- Use table-driven tests
- Test file naming: `*_test.go`

```go
func TestMyStrategy(t *testing.T) {
    tests := []struct {
        name     string
        market   trade.MarketData
        want     bool
        wantErr  bool
    }{
        {
            name: "bullish signal",
            market: trade.MarketData{RSI: 30},
            want: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s := &MyStrategy{}
            got, _, err := s.ShouldEnter(context.Background(), tt.market)
            if (err != nil) != tt.wantErr {
                t.Errorf("ShouldEnter() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ShouldEnter() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Logging Guidelines

- Use `logrus` for logging
- Log levels: `debug`, `info`, `warn`, `error`
- Structured logging with context

```go
log.WithFields(logrus.Fields{
    "symbol": symbol,
    "action": action,
    "price":  price,
}).Info("Executing trade")
```

### Security Guidelines

- Never commit API keys to repository
- Use environment variables for sensitive information
- Add `.env` to `.gitignore`
- Set API key file permissions to 600

```bash
chmod 600 .env
```

## AI Provider Configuration

The project supports multiple free AI providers, used in priority order:

### Priority Order

1. **Groq - Kimi-K2 (Moonshot AI)** (Primary) - Free, fast, suitable for crypto analysis
2. **Groq - Llama 3.3 70B** (Fallback) - Free large context model
3. **Google - Gemini 1.5 Flash** (Last fallback) - Free tier available
4. **Ollama - CogneeBrain** (Local) - Local model, requires Ollama
5. **Ollama - LiquidAIBrain** (Local) - Local model, requires Ollama

### Token Usage

- Per request: ~400 prompt tokens + ~150 response tokens = 550 tokens
- Hourly limit: 20 requests (safety margin)
- Hourly total: ~11,000 tokens
- Daily: ~264,000 tokens
- Monthly: ~7.9M tokens (fully within free tier)

### Configuration Example

```yaml
ai:
  enabled: true
  primary_model: "moonshotai/kimi-k2-instruct"
  fallback_model: "llama-3.3-70b-versatile"
  gemini_model: "gemini-1.5-flash"
  max_requests_per_hour: 20
  max_tokens_per_minute: 10000
```

### Ollama Configuration

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull models
ollama pull cogneebrain
ollama pull liquidaibrain

# Or use modelfile
ollama create cogneebrain -f CogneeBrain.modelfile
ollama create liquidaibrain -f LiquidAIBrain.modelfile
```

## SimpleMem Memory System

SimpleMem is an efficient long-term memory management system based on semantic lossless compression principles.

### Core Features

- **Semantic Structured Compression**: Convert dialogues to structured memory entries
- **Hybrid Retrieval**: Combine keyword search and semantic retrieval
- **Adaptive Query-Aware Retrieval**: Dynamically adjust retrieval based on queries
- **Vector Storage**: Use vector database for semantic search
- **Dialogue History Management**: Support context maintenance for multi-turn conversations

### Usage

```python
from memory.main import SimpleMemSystem

# Initialize system
mem = SimpleMemSystem(
    api_key="your_api_key",
    model="gpt-4",
    db_path="./memory.db"
)

# Add dialogue
mem.add_dialogue(
    user="User message",
    assistant="Assistant response",
    metadata={"source": "trading"}
)

# Query memory
result = mem.ask("How to set stop loss?")
print(result.answer)
```

### Configuration Options

```python
mem = SimpleMemSystem(
    api_key="your_api_key",
    model="gpt-4",
    base_url="https://api.openai.com/v1",  # Optional: custom API endpoint
    db_path="./memory.db",
    table_name="memory",
    clear_db=False,
    enable_thinking=True,      # Enable thinking mode
    use_streaming=False,       # Enable streaming output
    enable_planning=True,      # Enable planning
    enable_reflection=True,    # Enable reflection
    max_reflection_rounds=3,   # Max reflection rounds
    enable_parallel_processing=True,  # Enable parallel processing
    max_parallel_workers=4,    # Max parallel workers
)
```

## Risk Management

### Built-in Risk Controls

- **Stop Loss**: Default 2%
- **Take Profit**: Default 4%
- **Trailing Stop**: Enabled, default 1.5%
- **Max Position**: Default 10 USD
- **Daily Trade Limit**: Default 30 trades
- **Weekly Loss Limit**: Default 50 USD
- **Max Daily Drawdown**: Default 5%
- **Kelly Fraction**: Default 0.25
- **Max Risk Per Trade**: Default 2%

### Emergency Controls

```bash
# Immediately stop all trading (panic mode)
./gobot.sh stop

# Or send in Telegram
/panic

# Create kill switch file
touch /tmp/gobot_kill_switch
```

### Circuit Breaker

```yaml
circuit_breaker:
  enabled: true
  failure_threshold: 5
  failure_window_seconds: 60
  recovery_timeout_seconds: 300
  half_open_requests: 3
```

## Monitoring and Logging

### Log Files

- **Main Log**: `logs/gobot.log`
- **Trading Log**: `logs/trades_mainnet.log`
- **Audit Log**: `logs/mainnet_audit.log`
- **Error Log**: `logs/error.log`
- **P&L Log**: `logs/pnl_YYYYMMDD.csv`
- **Alert Log**: `logs/mainnet_alerts.log`

### Real-time Monitoring

```bash
# View real-time logs
tail -f logs/gobot.log

# View trading logs
tail -f logs/trades_mainnet.log

# Use watch for continuous monitoring
watch -n 5 'tail -20 logs/gobot.log'

# Mainnet monitoring
./mainnet-production.sh monitor
```

### Health Checks

```bash
# Check GOBOT status
curl http://localhost:8080/health

# Check service status
./gobot.sh status

# Mainnet status check
./mainnet-production.sh status
```

## Troubleshooting

### Common Issues

1. **GOBOT won't start**
   - Check if `.env` file is properly configured
   - Check if port 8080 is in use
   - View logs: `./gobot.sh logs`

2. **N8N won't start**
   - Check if Docker is running
   - Check if port 5678 is in use
   - View logs: `docker-compose logs n8n`

3. **Webhooks not working**
   - Test health check: `curl http://localhost:8080/health`
   - Check firewall settings
   - Verify N8N workflows are active

4. **AI requests failing**
   - Check if API keys are correct
   - Check rate limits
   - View error messages in logs

5. **Trade execution failing**
   - Check Binance API credentials
   - Check account balance
   - Check if trading pair is available

6. **SimpleMem system errors**
   - Check Python version (requires 3.14+)
   - Check dependency installation: `pip install -r memory/requirements.txt`
   - Check database permissions

### Debug Mode

```bash
# Enable debug logging
export GOBOT_LOG_LEVEL=debug
./gobot

# Test jitter
go run ./cmd/test_jitter

# Verify API connection
./test_binance_account.sh
```

## Deployment Guide

### Testnet Deployment

```bash
# 1. Set testnet mode
export BINANCE_USE_TESTNET=true

# 2. Get testnet credentials
# Visit https://testnet.binance.vision/

# 3. Start
./gobot.sh start

# Or run testnet test
./run-testnet-final.sh
```

### Mainnet Deployment

⚠️ **Warning**: Before deploying to mainnet, make sure to:
1. Run on testnet for at least 24-48 hours
2. Verify all functions are working
3. Test emergency stop functionality
4. Start with small amounts

```bash
# 1. Configure mainnet
./mainnet-production.sh setup

# 2. Verify configuration
./mainnet-production.sh check

# 3. Test connection
./mainnet-production.sh connect

# 4. Start trading
./mainnet-production.sh start

# 5. Monitor
./mainnet-production.sh monitor
```

### System Service Deployment (Linux)

```bash
# Use systemd
sudo ./scripts/setup_systemd.sh

# Start service
sudo systemctl start gobot

# Check status
sudo systemctl status gobot

# View logs
journalctl -u gobot -f
```

### System Service Deployment (macOS)

```bash
# Use launchd
cp docs/com.gobot.mainnet.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.gobot.mainnet.plist
launchctl start com.gobot.mainnet
```

## Related Documentation

- **Complete Implementation Report**: `COMPLETE_IMPLEMENTATION_REPORT.md`
- **Modular Architecture**: `MODULAR_ARCHITECTURE.md`
- **N8N Integration**: `N8N_INTEGRATION_PLAN.md`
- **AI Routing**: `LLM_ROUTING_N8N_INTEGRATION.md`
- **Mainnet Deployment**: `MAINNET_DEPLOYMENT_GUIDE.md`
- **Quick Start**: `QUICK_START.md`
- **Local Setup**: `README_LOCAL.md`
- **N8N Setup Guide**: `n8n/SETUP_GUIDE.md`
- **Preflight Validation**: `docs/PREFLIGHT_VALIDATION.md`

## Telegram Commands

- `/status` - View current P&L and positions
- `/panic` - Emergency stop (close all positions)
- `/halt` - Stop new entries only
- `/reconcile` - Force reconciliation check

## Key Metrics

| Metric | Target | Check Command |
|--------|--------|---------------|
| Time Offset | < 500μs | `chronyc tracking \| grep offset` |
| WebSocket Latency | < 50ms | `grep "WS" logs/gobot.log` |
| WAL Flush | < 1ms | Automatic (buffered) |
| Reconciliation | < 1s | `grep RECON logs/gobot.log` |
| CPU Usage | < 15% | `top -pid $(pgrep gobot)` |
| Memory Usage | < 4GB | `ps aux \| grep gobot` |
| Ghost Positions | 0/day | `grep GHOST logs/gobot.log \| wc -l` |

## Project Version Info

- **Go Version**: 1.25.4
- **Project Version**: 2.0.0
- **Status**: ✅ Production Ready
- **Last Updated**: 2026-01-16

## Important Reminders

1. ⚠️ **Always test on testnet first** for 24-48 hours
2. 🔒 **Never share API keys** or .env files
3. 📁 **Keep .env chmod 600** (secure permissions)
4. 🧪 **Test /panic command before mainnet**
5. 📊 **Monitor logs daily** for ghost positions
6. 🔄 **Update dependencies monthly**
7. 🔄 **Rotate API keys every 90 days**
8. 🌐 **Use IP whitelisting on Binance**
9. 💰 **Start small** (position size)
10. 🆘 **Have emergency funds ready**

## Contributing

1. Fork the project
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

MIT License

---

**Last Updated**: 2026-01-16
**Version**: 2.0.0
**Status**: ✅ Production Ready