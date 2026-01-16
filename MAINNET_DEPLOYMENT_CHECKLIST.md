# Mainnet Deployment Checklist

## ⚠️ CRITICAL WARNINGS

- **This is REAL MONEY trading**
- **Start with 10 USDT minimum position**
- **Testnet testing required first**
- **Never commit .env file**
- **Enable IP whitelisting on Binance**

---

## Pre-Deployment Requirements

### 1. API Keys Configuration

- [ ] Copy `.env.example` to `.env`
- [ ] Fill in `BINANCE_API_KEY` (mainnet)
- [ ] Fill in `BINANCE_API_SECRET` (mainnet)
- [ ] Fill in `GEMINI_API_KEY` (free tier)
- [ ] Fill in `OPENROUTER_API_KEY` (free tier)
- [ ] Fill in `GROQ_API_KEY` (free tier)
- [ ] Set `BINANCE_USE_TESTNET=false`
- [ ] Set `KILL_SWITCH_PASSWORD` for emergency stop

### 2. Binance Account Setup

- [ ] Enable IP whitelisting on Binance
- [ ] Set API key permissions:
  - [ ] Enable Futures Trading
  - [ ] Enable Reading
  - [ ] Disable Withdrawals (for safety)
- [ ] Verify account balance >= 30 USDT
- [ ] Check margin mode: Cross or Isolated
- [ ] Set leverage limits if needed

### 3. Telegram Notifications (Optional but Recommended)

- [ ] Create Telegram bot via @BotFather
- [ ] Get bot token
- [ ] Get chat ID
- [ ] Test notifications with `/status` command
- [ ] Configure kill switch commands

### 4. System Preparation

- [ ] Create logs directory: `mkdir -p logs`
- [ ] Set file permissions: `chmod 600 .env`
- [ ] Verify Go 1.25.4 installed: `go version`
- [ ] Build executable: `go build ./cmd/cobot`
- [ ] Test executable: `./cobot --help`

### 5. Configuration Verification

- [ ] Review `config/config.yaml`
- [ ] Verify `initial_capital_usd: 30`
- [ ] Verify `max_position_usd: 20`
- [ ] Verify `min_position_usd: 10`
- [ ] Verify `stop_loss_percent: 2.0`
- [ ] Verify `take_profit_percent: 4.0`
- [ ] Verify `use_testnet: false`

---

## Testnet Validation (REQUIRED)

### Before Mainnet: Complete These Steps

1. **Run on Testnet for 24-48 Hours**
   ```bash
   export BINANCE_USE_TESTNET=true
   ./cobot
   ```

2. **Verify All Functions Work**
   - [ ] Trading decisions generated
   - [ ] Orders placed correctly
   - [ ] Stop loss orders set
   - [ ] Take profit orders set
   - [ ] Position monitoring active
   - [ ] Trailing stop works
   - [ ] Emergency stop works

3. **Check Logs**
   - [ ] No critical errors
   - [ ] All API calls successful
   - [ ] Rate limits respected
   - [ ] No ghost positions

4. **Test Emergency Procedures**
   - [ ] Kill switch works
   - [ ] Manual stop works
   - [ ] Position closure works
   - [ ] Telegram alerts work

---

## Mainnet Deployment Steps

### Phase 1: Deployment

1. **Stop Testnet** (if running)
   ```bash
   # Kill the process
   kill $(pgrep cobot)
   ```

2. **Configure Mainnet**
   ```bash
   cp .env.example .env
   nano .env  # Edit with your API keys
   chmod 600 .env
   ```

3. **Verify Configuration**
   ```bash
   grep "BINANCE_USE_TESTNET" .env  # Should be false
   grep "initial_capital_usd" config/config.yaml  # Should be 30
   ```

4. **Start Mainnet Bot**
   ```bash
   ./cobot
   ```

### Phase 2: Initial Monitoring (First Hour)

- [ ] Watch logs: `tail -f logs/gobot.log`
- [ ] Check first trade execution
- [ ] Verify position size (should be 10-20 USDT)
- [ ] Verify stop loss set
- [ ] Verify take profit set
- [ ] Check Telegram notifications

### Phase 3: First Day Monitoring

- [ ] Monitor every trade
- [ ] Verify P&L calculations
- [ ] Check rate limit usage
- [ ] Review AI decisions
- [ ] Verify no ghost positions
- [ ] Check daily loss limit (15 USDT)

---

## Emergency Procedures

### If Something Goes Wrong

1. **Immediate Stop**
   ```bash
   # Kill the bot
   kill $(pgrep cobot)
   
   # Or use kill switch via Telegram
   /panic
   ```

2. **Close All Positions**
   ```bash
   # Manual position closure via Binance UI
   # Or use emergency script
   ```

3. **Check Logs**
   ```bash
   tail -100 logs/gobot.log
   tail -100 logs/error.log
   ```

4. **Verify Account**
   ```bash
   # Check Binance account
   # Verify no unexpected positions
   # Check balance
   ```

---

## Monitoring Commands

### Real-Time Monitoring

```bash
# View main log
tail -f logs/gobot.log

# View error log
tail -f logs/error.log

# View trading log
tail -f logs/trades_mainnet.log

# Check process
ps aux | grep cobot

# Check memory usage
top -pid $(pgrep cobot)
```

### Health Checks

```bash
# Check API health
curl http://localhost:8080/health

# Check metrics
curl http://localhost:8080/metrics

# Check positions
curl http://localhost:8080/positions
```

---

## Risk Limits (30 USDT Balance)

| Parameter | Value | Description |
|-----------|-------|-------------|
| Opening Balance | 30 USDT | Total capital |
| Min Position | 10 USDT | Minimum trade size |
| Max Position | 20 USDT | Maximum single position |
| Max Positions | 3 | Concurrent positions |
| Stop Loss | 2% | Automatic stop loss |
| Take Profit | 4% | Automatic take profit |
| Trailing Stop | 1.5% | Trailing stop distance |
| Daily Loss Limit | 15 USDT | Stop trading if loss > 15 USDT |
| Weekly Loss Limit | 30 USDT | Stop trading if loss > 30 USDT |

---

## Success Criteria

### First Week Targets

- [ ] No critical errors
- [ ] Win rate > 60%
- [ ] Daily loss limit never exceeded
- [ ] No ghost positions
- [ ] All stop losses honored
- [ ] All take profits honored
- [ ] AI decisions reasonable
- [ ] Rate limits respected
- [ ] Telegram alerts working
- [ ] P&L positive or minimal loss

---

## Post-Deployment Actions

### Daily

- [ ] Review all trades
- [ ] Check P&L
- [ ] Review AI decisions
- [ ] Check error logs
- [ ] Verify no ghost positions

### Weekly

- [ ] Review overall performance
- [ ] Check weekly loss limit
- [ ] Review AI provider usage
- [ ] Optimize parameters if needed
- [ ] Backup logs

### Monthly

- [ ] Comprehensive performance review
- [ ] Update strategies if needed
- [ ] Review and rotate API keys
- [ ] Update dependencies
- [ ] Security audit

---

## Troubleshooting

### Common Issues

**Issue**: Bot won't start
- Check .env file permissions (chmod 600)
- Verify API keys are correct
- Check if port 8080 is available

**Issue**: No trades executing
- Check AI provider API keys
- Check confidence threshold
- Check balance
- Check error logs

**Issue**: Orders failing
- Verify API key permissions
- Check account balance
- Check order size (min 10 USDT)
- Check symbol availability

**Issue**: High latency
- Check internet connection
- Check AI provider response time
- Check Binance API status

---

## Support Resources

- **Documentation**: `IFLOW.md`
- **Configuration**: `config/config.yaml`
- **Logs**: `logs/` directory
- **Telegram**: Use `/status` command
- **Emergency**: Kill switch or manual stop

---

## Final Confirmation

Before deploying to mainnet, confirm:

- [ ] I understand this is REAL money trading
- [ ] I have tested on testnet for 24-48 hours
- [ ] I have configured all API keys correctly
- [ ] I have enabled IP whitelisting on Binance
- [ ] I have setup Telegram notifications
- [ ] I have tested emergency procedures
- [ ] I will start with 10 USDT minimum positions
- [ ] I will monitor closely for the first 24 hours
- [ ] I understand the risk limits (15 USDT daily loss)
- [ ] I am ready to proceed with mainnet deployment

---

**Last Updated**: 2026-01-16
**Version**: 1.0.0
**Status**: Ready for Deployment