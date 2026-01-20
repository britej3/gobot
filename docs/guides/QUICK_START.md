# COGNEE QUICK START GUIDE
## 5-Minute Deployment Reference

**Status:** ✅ Production Ready  
**Platforms:** Linux / macOS Intel  
**Version:** 1.0.0

---

## 🚀 LINUX DEPLOYMENT (2 MINUTES)

```bash
# 1. Install Go (if not installed)
sudo apt update && sudo apt install -y golang-go

# 2. Clone and build
git clone <cognee-repo> ~/cognee
cd ~/cognee
go build -ldflags="-s -w" -o cognee ./cmd/cognee

# 3. Configure .env
cp .env.example .env
nano .env  # Add your keys

# 4. Secure files
chmod 600 .env state.json trade.wal

# 5. Install systemd service
sudo ./scripts/setup_systemd.sh

# 6. Start bot
sudo systemctl start cognee

# 7. Monitor
journalctl -u cognee -f --since "5 minutes ago"
```

---

## 🍎 macOS INTEL DEPLOYMENT (3 MINUTES)

```bash
# 1. Install dependencies
xcode-select --install
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install go chrony

# 2. Build for Intel
cd ~/cognee
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o cognee ./cmd/cognee

# 3. Configure .env
nano .env  # Add your keys
chmod 600 .env state.json trade.wal

# 4. Setup Chrony
sudo chronyd
sudo brew services start chrony

# 5. Create launchd plist
cp docs/com.cognee.mainnet.plist ~/Library/LaunchAgents/
sed -i '' 's/YOUR_USER/'$(whoami)'/g' ~/Library/LaunchAgents/com.cognee.mainnet.plist

# 6. Start service
launchctl load ~/Library/LaunchAgents/com.cognee.mainnet.plist
launchctl start com.cognee.mainnet

# 7. Verify time sync
chronyc tracking | grep "Last offset"  # Should be < 0.0005s

# 8. Monitor logs
tail -f ~/cognee/logs/cognee.log
```

---

## 🔧 .env CONFIGURATION

```bash
# Required
cat > .env << EOF
BINANCE_API_KEY=your_api_key_here
BINANCE_API_SECRET=your_secret_here
TELEGRAM_TOKEN=your_bot_token_here
AUTHORIZED_CHAT_ID=your_telegram_user_id_here
BINANCE_USE_TESTNET=true  # Start on testnet!

# Risk Management
SAFE_STOP_THRESHOLD_PERCENT=10.0
SAFE_STOP_MIN_BALANCE_USD=100.0

# Optional
MAX_ASSETS=15
MIN_24H_VOLUME=1000000
EOF

chmod 600 .env
```

---

## 📊 VERIFICATION COMMANDS

### Quick Health Check
```bash
# Linux
sudo systemctl status cognee
journalctl -u cognee -n 20

# macOS
launchctl list | grep cognee
tail -n 20 ~/cognee/logs/cognee.log
```

### Performance Check
```bash
# Time sync (both platforms)
chronyc tracking | grep "Last offset"  # Must be < 0.0005s

# Check ghost positions
grep "GHOST" logs/cognee.log

# Verify jitter applied
grep "anti-sniffer" logs/cognee.log
```

### Test Telegram
```bash
# Send test message
curl -X POST "https://api.telegram.org/bot$TELEGRAM_TOKEN/sendMessage?chat_id=$AUTHORIZED_CHAT_ID&text=Cognee+is+running"
```

---

## 🚨 EMERGENCY COMMANDS

### Immediate Stop (Panic)
```bash
# Linux
sudo systemctl stop cognee

# macOS
launchctl stop com.cognee.mainnet

# Manual panic (any platform)
cd ~/cognee && ./cognee --panic
```

### Restart Service
```bash
# Linux
sudo systemctl restart cognee

# macOS
launchctl kickstart -k gui/$(id -u)/com.cognee.mainnet
```

### View Recent Errors
```bash
# Linux
journalctl -u cognee -f | grep -i error

# macOS
tail -f ~/cognee/logs/error.log
```

---

## 📈 MONITORING DASHBOARD

### Real-Time Metrics (Linux)
```bash
# Combined monitoring script
watch -n 5 '
echo "=== Cognee Status ===" &&
systemctl is-active cognee &&
echo "" &&
echo "Time Offset:" &&
chronyc tracking | grep "Last offset" &&
echo "" &&
echo "Recent GHOSTs:" &&
journalctl -u cognee -n 10 | grep GHOST | tail -3 &&
echo "" &&
echo "Recent Errors:" &&
journalctl -u cognee -n 10 | grep -i error | tail -3
'
```

### Real-Time Metrics (macOS)
```bash
# Combined monitoring script
watch -n 5 '
echo "=== Cognee Status ===" &&
launchctl list | grep cognee &&
echo "" &&
echo "Time Offset:" &&
chronyc tracking | grep "Last offset" &&
echo "" &&
echo "Recent GHOSTs:" &&
tail -n 10 ~/cognee/logs/cognee.log | grep GHOST | tail -3 &&
echo "" &&
echo "Recent Errors:" &&
tail -n 10 ~/cognee/logs/cognee.log | grep -i error | tail -3
'
```

---

## 📱 TELEGRAM COMMAND QUICK REFERENCE

- `/status` - Current P&L and open positions
- `/panic` - Emergency stop (closes all positions)
- `/halt` - Stop new entries only
- `/reconcile` - Force reconciliation check

---

## 🎯 CRITICAL METRICS

| Metric | Target | Check Command |
|--------|--------|---------------|
| Time Offset | < 500μs | `chronyc tracking \| grep offset` |
| WebSocket Latency | < 50ms | `grep "WS" logs/cognee.log` |
| WAL Flush | < 1ms | Buffered (automatic) |
| Reconciliation | < 1s | `grep RECON logs/cognee.log` |
| CPU Usage | < 15% | `top -pid $(pgrep cognee)` |
| Memory | < 4GB | `ps aux \| grep cognee` |
| Ghost Positions | 0/day | `grep GHOST logs/cognee.log \| wc -l` |

---

## 🔔 DAILY CHECKLIST

```bash
# Quick 30-second health check
#!/bin/bash
echo "=== Daily Cognee Check ==="
echo "Time: $(date)"
echo ""

# Check service
if systemctl is-active cognee &>/dev/null; then
    echo "✅ Service running (Linux)"
elif launchctl list | grep -q cognee; then
    echo "✅ Service running (macOS)"
else
    echo "❌ Service NOT running"
fi

# Check time
OFFSET=$(chronyc tracking | awk '/Last offset/ {print $4}')
if (( $(echo "$OFFSET < 0.0005" | bc -l) )); then
    echo "✅ Time sync good: $OFFSET"
else
    echo "❌ Time sync BAD: $OFFSET"
fi

# Check ghosts
GHOSTS=$(grep -c "GHOST" logs/cognee.log 2>/dev/null || echo "0")
echo "Ghost positions today: $GHOSTS"

# Check errors
ERRORS=$(grep -c "ERROR" logs/error.log 2>/dev/null || echo "0")
echo "Errors today: $ERRORS"

echo ""
echo "Status: $(if [ "$GHOSTS" -lt 3 ] && [ "$ERRORS" -lt 5 ]; then echo "✅ HEALTHY"; else echo "⚠️  REVIEW"; fi)"
```

---

## 📚 DOCUMENTATION LINKS

- **Full Guide:** `COMPLETE_IMPLEMENTATION_REPORT.md`
- **macOS Setup:** `docs/MACOS_INTEL_INSTALL.md`
- **Linux Setup:** `FINAL_SETUP_GUIDE.md`
- **Component Details:** `INTEGRATION_SUMMARY.md`
- **Spec Mapping:** `TECHNICAL_SPECS_IMPLEMENTATION.md`

---

## ⚠️ CRITICAL REMINDERS

1. **Always start on testnet** for 24-48 hours
2. **Never share API keys** or .env file
3. **Keep .env chmod 600** (secure permissions)
4. **Test /panic command** before mainnet
5. **Monitor logs daily** for ghost positions
6. **Update dependencies** monthly
7. **Rotate API keys** every 90 days
8. **Use IP whitelisting** on Binance
9. **Start small** (position sizes)
10. **Have emergency funds** ready

---

## 🚀 FROM ZERO TO TRADING IN 5 MINUTES

**Fastest deployment path:**

```bash
# 1. Setup (2 minutes)
cd ~/cognee
nano .env  # Add keys
chmod 600 .env

# 2. Build (1 minute)
go build -o cognee ./cmd/cognee

# 3. Start service (1 minute)
# Linux:
sudo ./scripts/setup_systemd.sh
sudo systemctl start cognee

# macOS:
launchctl load ~/Library/LaunchAgents/com.cognee.mainnet.plist

# 4. Monitor (1 minute)
tail -f logs/cognee.log | grep -E "(✅|⚠️|🚨)"
```

**Status:** ✅ Ready for trading  
**Time to deploy:** ~5 minutes  
**Learning curve:** Low (comprehensive docs)  
**Risk level:** Managed (multiple safety systems)

---

**🎊 COGNEE IS READY - HAPPY TRADING! 🎊**
