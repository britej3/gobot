# GOBOT Mainnet Deployment Guide

## 🚨 CRITICAL: Safety-First Approach

### Phase 1: Pre-Deployment Checklist

#### ✅ Testnet Validation (COMPLETE)
- [x] 60-minute testnet run
- [x] 4 cycles completed successfully
- [x] 100% success rate
- [x] Telegram alerts working
- [x] AI analysis (Llama 3.3 70B) functional

#### ✅ Safety Configuration (REQUIRED)
- [x] Risk manager configured
- [x] Circuit breakers enabled
- [x] Conservative position sizes
- [x] Daily loss limits
- [x] Stop loss at 1.5%

### Phase 2: Mainnet API Setup

#### Step 1: Get Mainnet API Keys
```bash
# 1. Go to: https://www.binance.com/en/my/settings/api-management
# 2. Create new API key
# 3. Enable: Futures Trading
# 4. Set IP restrictions (recommended)
# 5. Save keys securely
```

#### Step 2: Configure Environment
```bash
# Update .env file
BINANCE_USE_TESTNET=false
BINANCE_API_KEY=<your_mainnet_api_key>
BINANCE_SECRET=<your_mainnet_secret>
```

### Phase 3: Ultra-Conservative Settings

#### Conservative Mainnet Config (FIRST WEEK)
```yaml
trading:
  initial_capital_usd: 100        # Start small!
  max_position_usd: 2             # 2% of capital per trade
  stop_loss_percent: 1.5          # Tight SL
  take_profit_percent: 3.0         # 2:1 RR
  min_confidence_threshold: 0.90   # Only high-confidence trades

risk_manager:
  max_daily_loss_usd: 25          # Stop after $25 loss
  max_daily_trades: 3             # Max 3 trades/day
  max_consecutive_losses: 2        # Stop after 2 losses
  max_drawdown_percent: 3         # Emergency stop
```

#### After 1 Week Success (Scale Up)
```yaml
trading:
  initial_capital_usd: 250
  max_position_usd: 5
  stop_loss_percent: 1.5
  take_profit_percent: 3.0
  min_confidence_threshold: 0.85

risk_manager:
  max_daily_loss_usd: 50
  max_daily_trades: 5
  max_consecutive_losses: 3
  max_drawdown_percent: 5
```

### Phase 4: Deployment Commands

#### Command 1: Verify Configuration
```bash
python mainnet_safety_config.py
# Check all safety settings
```

#### Command 2: Test with Dry Run
```bash
# Set dry run mode (no actual trades)
export DRY_RUN=true
python gobot_trading_orchestrator.py
```

#### Command 3: Go Live (When Ready)
```bash
# Set mainnet mode
export BINANCE_USE_TESTNET=false
export DRY_RUN=false

# Start with 15-minute test
python gobot_trading_orchestrator.py --max-iterations=1

# Monitor first trade carefully
# Check Telegram for alerts
# Verify execution
```

#### Command 4: Full Deployment
```bash
# Full autonomous run (15-minute cycles)
python gobot_trading_orchestrator.py --max-iterations=96  # 24 hours
```

### Phase 5: Monitoring & Alerts

#### Telegram Alerts (Critical)
```python
# You'll receive notifications for:
# ✅ Trade executed: LONG BTCUSDT $5 @ 95320.50
# ✅ Stop loss triggered: -$0.08
# ✅ Take profit hit: +$0.15
# ⚠️ Risk limit reached: Daily loss limit hit
# 🚨 Emergency stop: Circuit breaker opened
# 📊 Daily summary: 3 trades, +$2.45
```

#### Monitoring Dashboard
```bash
# Check progress
tail -f gobot_state/progress.json

# View performance
cat gobot_state/progress.json | jq '.cycles[-1]'

# Check patterns
cat gobot_state/progress.json | jq '.patterns'
```

### Phase 6: Scaling Strategy

#### Week 1: Ultra-Conservative
- Capital: $100
- Max position: $2
- Daily trades: 3
- Daily loss limit: $25

#### Week 2-4: Proven Success
- Capital: $250
- Max position: $5
- Daily trades: 5
- Daily loss limit: $50

#### Month 2+: Confidence Building
- Capital: $500
- Max position: $10
- Daily trades: 10
- Daily loss limit: $100

### Emergency Procedures

#### If Something Goes Wrong
```bash
# 1. IMMEDIATELY stop the bot
# Press Ctrl+C or kill the process

# 2. Check your positions
# Login to Binance Futures manually

# 3. Close positions if needed
# Manual intervention required

# 4. Review logs
# cat gobot_state/progress.json

# 5. Adjust settings
# Edit mainnet_safety_config.py

# 6. Restart with dry run
# export DRY_RUN=true
```

#### Circuit Breaker Activation
```
If circuit breaker opens:
1. Bot stops automatically
2. Telegram alert sent
3. All positions monitored
4. Manual restart required
5. Review logs before restart
```

### Success Metrics (First Week)

#### Target Performance
- Daily trades: 3-5
- Win rate: >60%
- Daily P&L: +$1-5
- Max daily loss: <$25
- No emergency stops

#### Green Flags (Continue)
- Consistent profits
- No circuit breaker activations
- All trades within risk limits
- Telegram alerts working

#### Red Flags (Pause & Review)
- 2+ consecutive losses
- Daily loss >$20
- Circuit breaker activation
- API errors
- Unusual volatility

### Quick Start Commands

```bash
# 1. Safety check
python mainnet_safety_config.py

# 2. Dry run test
export DRY_RUN=true
python gobot_trading_orchestrator.py

# 3. First real trade
export DRY_RUN=false
export BINANCE_USE_TESTNET=false
python gobot_trading_orchestrator.py --max-iterations=1

# 4. Monitor closely
tail -f gobot_state/progress.json

# 5. Check Telegram
# Look for: Trade executed, Stop loss, Take profit
```

### Financial Safety Net

#### Starting Capital: $100
- Max per trade: $2
- Max daily loss: $25 (25% of capital)
- Emergency stop: 3% drawdown ($3)
- **Worst case: Lose $100 in 4 days**

#### Recommended: Start with $100, scale up after 1 week of success

### Legal & Compliance

#### Important Notes
- Binance Futures trading may be restricted in your jurisdiction
- Ensure you understand tax implications
- Keep records of all trades
- Never invest more than you can afford to lose
- This is for educational purposes

### Support & Troubleshooting

#### Common Issues
1. **API key error**: Check .env file
2. **Network timeout**: Check internet connection
3. **Telegram not working**: Verify token/chat_id
4. **Orders rejected**: Check margin balance
5. **High latency**: Use closer server

#### Logs Location
- Progress: `gobot_state/progress.json`
- Archive: `gobot_archive/`
- Errors: Check console output

### Final Checklist Before Go-Live

- [ ] Testnet validated (✅ Done)
- [ ] Mainnet API keys obtained
- [ ] Safety config reviewed
- [ ] Telegram alerts tested
- [ ] Position sizes conservative
- [ ] Loss limits set
- [ ] Emergency procedures documented
- [ ] Ready to monitor closely
- [ ] **Comfortable losing $100**

### Go-Live Decision

**Only proceed if:**
- ✅ All checklist items complete
- ✅ You're comfortable with the risk
- ✅ You can monitor the first few trades
- ✅ You have emergency stop plan
- ✅ You've read and understood all risks

**Remember:** Better to start small and scale up than to lose big!
