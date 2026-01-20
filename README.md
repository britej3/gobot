# GOBOT - Autonomous Trading Bot

**GOBOT** is an advanced autonomous cryptocurrency trading bot built in Go, featuring AI/LLM integration, memory systems, and autonomous agent capabilities for intelligent trading decisions.

## Overview

GOBOT is a production-ready trading system that combines:
- **Binance API Integration** for real-time trading
- **AI/LLM Decision Making** with multiple provider support (OpenAI, Claude, Gemini, Ollama)
- **Memory System** (SimpleMem) for learning from past trades
- **Autonomous Agent** (Ralph) for self-improvement and feature development
- **Risk Management** with circuit breakers and position management
- **Real-time Monitoring** via Telegram alerts and audit logging

## Architecture

```
gobot/
├── cmd/                    # Entry points for different bot modes
│   ├── gobot-engine/      # Main trading engine
│   ├── cobot/             # Alternative bot implementation
│   └── cognee/            # Cognitive engine
├── config/                 # Configuration management
├── domain/                 # Core business logic
│   ├── trade/             # Trading domain
│   ├── market/            # Market data
│   ├── strategy/          # Trading strategies
│   └── llm/               # LLM integration
├── infra/                  # Infrastructure layer
│   ├── binance/           # Binance client
│   ├── llm/               # LLM providers
│   ├── storage/           # Data persistence
│   └── monitoring/        # Observability
├── internal/               # Internal services
│   ├── agent/             # Autonomous agent logic
│   ├── brain/             # Decision-making core
│   ├── risk/              # Risk management
│   └── position/          # Position management
├── services/               # Application services
│   ├── executor/          # Trade execution
│   ├── screener/          # Asset screening
│   └── quantcrawler/      # Market data crawler
├── memory/                 # SimpleMem - Trading memory system
└── scripts/                # Utility scripts
    └── ralph/             # Ralph - Autonomous development agent
```

## Quick Start

### Prerequisites

- Go 1.18 or higher
- Python 3.8+ (for memory system)
- Ollama (for local LLM)
- Redis (optional, for caching)
- Binance API keys

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/britej3/gobot.git
cd gobot
```

2. **Install Go dependencies**
```bash
go mod download
```

3. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env with your API keys
```

4. **Set up memory system**
```bash
cd memory
./setup.sh
source venv/bin/activate
```

5. **Build the bot**
```bash
go build -o gobot cmd/gobot-engine/main.go
```

### Running the Bot

**Testnet Mode (Recommended for first run):**
```bash
./run-testnet-final.sh
```

**Production Mode:**
```bash
./start-autonomous-trading.sh
```

**With specific configuration:**
```bash
./gobot --config config/production.yaml
```

## Key Features

### 1. AI-Powered Trading
- Multiple LLM provider support (OpenAI, Claude, Gemini, Ollama)
- Context-aware decision making
- Learning from historical trades via SimpleMem

### 2. Risk Management
- Circuit breakers for rapid loss prevention
- Position size limits
- Daily loss limits
- Per-symbol cooldown periods
- Kill switch for emergency stops

### 3. Memory System (SimpleMem)
- Stores trading experiences with semantic search
- Retrieves relevant context before trades
- Learns from wins and losses
- Python-based with Go bridge

### 4. Autonomous Agent (Ralph)
- Reads PRD (Product Requirements Document)
- Implements features autonomously
- Commits code and updates progress
- Self-improving capabilities

### 5. Monitoring & Alerts
- Real-time Telegram notifications
- Comprehensive audit logging
- Trade history tracking
- Performance metrics

## Configuration

### Main Configuration File
Edit `config/config.yaml`:

```yaml
binance:
  api_key: "your-api-key"
  api_secret: "your-api-secret"
  use_testnet: true

risk:
  max_position_size: 100.0
  max_daily_loss: 50.0
  max_trades_per_day: 10

llm:
  provider: "ollama"
  model: "llama3"
  temperature: 0.7

monitoring:
  telegram_enabled: true
  telegram_token: "your-bot-token"
  telegram_chat_id: "your-chat-id"
```

## Usage Examples

### Basic Trading
```go
import "github.com/britej3/gobot/domain/trade"

// Initialize trading engine
engine := trade.NewEngine(config)

// Start trading
engine.Start(context.Background())
```

### Using Memory System
```python
from trading_memory import TradingMemory

memory = TradingMemory()

# Store a trade
memory.add_trade(
    symbol="BTCUSDT",
    side="long",
    entry_price=43500.50,
    exit_price=43800.00,
    pnl_percent=0.68,
    outcome="win",
    lessons_learned="Wait for confirmation"
)

# Query before trading
context = memory.get_trading_context("BTCUSDT", "long")
```

### Using Ralph Agent
```bash
cd scripts/ralph

# Create PRD
cat > prd.json << EOF
{
  "branchName": "ralph/add-feature",
  "userStories": [
    {
      "id": "US-001",
      "title": "Add logging to trades",
      "priority": 1,
      "passes": false,
      "description": "Log entry and exit prices"
    }
  ]
}
EOF

# Run Ralph
./ralph.sh 10
```

## Testing

### Run Tests
```bash
go test ./...
```

### Verify Setup
```bash
./verify_repositories.sh
```

### Test Binance Connection
```bash
./test-binance-testnet.sh
```

## Documentation

- **[Quick Start Guide](QUICK_START.md)** - Get started in 5 minutes
- **[Repository Overview](REPOSITORY_OVERVIEW.md)** - Detailed component overview
- **[Trading Strategy](docs/TRADING_STRATEGY.md)** - Strategy documentation
- **[LLM Workflow](docs/LLM_WORKFLOW.md)** - AI integration details
- **[Mainnet Deployment](MAINNET_DEPLOYMENT_GUIDE.md)** - Production deployment

## Project Status

✅ **Core Trading Engine** - Fully operational  
✅ **Binance Integration** - Production ready  
✅ **LLM Integration** - Multiple providers supported  
✅ **Memory System** - SimpleMem integrated  
✅ **Autonomous Agent** - Ralph operational  
✅ **Risk Management** - Circuit breakers active  
✅ **Monitoring** - Telegram alerts working  

## Safety & Disclaimers

⚠️ **Important Notes:**
- Always test on Binance Testnet first
- Start with small position sizes
- Monitor the bot actively during initial runs
- Use kill switches and circuit breakers
- Cryptocurrency trading carries significant risk
- This software is provided as-is without warranties

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

This project is licensed under the MIT License - see LICENSE file for details.

## Support

For issues, questions, or feature requests:
- Open an issue on GitHub
- Check existing documentation
- Review the troubleshooting guides

## Acknowledgments

- **SimpleMem** - AI-powered memory system
- **Ralph** - Autonomous development agent
- **Binance** - Trading platform
- **Ollama** - Local LLM support

---

**Built with ❤️ for autonomous trading**
