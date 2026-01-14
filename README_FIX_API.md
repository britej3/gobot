# 🚨 URGENT: Fix Binance Testnet API Permissions

## Problem
Your bot is crashing with: `code=-2015, msg=Invalid API-key, IP, or permissions for action`

## Root Cause
Your Binance Testnet API key does **NOT** have Futures trading permission enabled.

## Solution (3 minutes)

### Step 1: Generate NEW Testnet API Keys
1. Visit: https://testnet.binancefuture.com/en/futures
2. Click **User Center** (top right) → **API Key**
3. Click **Generate API Key**
4. **CRITICAL**: Check the box **"Enable Futures"**
5. Set IP whitelist to: `0.0.0.0/0` (for testing)
6. Click **Create**
7. **SAVE** both API Key and Secret Key

### Step 2: Update .env File
Edit `/Users/britebrt/GOBOT/.env`:

```bash
# Replace with your NEW testnet keys
BINANCE_TESTNET_API=your_new_api_key_here
BINANCE_TESTNET_SECRET=your_new_secret_here

# Ensure this is set
cd /Users/britebrt/GOBOT
export BINANCE_USE_TESTNET=true
```

### Step 3: Restart Bot
```bash
./run_gobot.sh
```

## Verification
Check if API works:
```bash
curl -H "X-MBX-APIKEY: your_new_api_key" \
  https://testnet.binancefuture.com/fapi/v2/account
```

Should return account info, NOT an error.

## Still Having Issues?
- Clear IP whitelist: Set to `0.0.0.0/0`
- Regenerate keys if they don't work
- Ensure you're using **testnet** keys, not mainnet
- Check that `BINANCE_USE_TESTNET=true` in .env

## Alternative: Skip API Checks (Not Recommended)
If you want to test AI logic without trades:
```bash
./cognee --audit  # Just checks connectivity
```

---
**Once API is fixed, your bot will trade automatically!**