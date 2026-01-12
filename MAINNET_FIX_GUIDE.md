# Mainnet API Fix Guide

## Problem Identified

Your Binance Mainnet API calls are failing with **`-1022 "Signature for this request is not valid."`**

### Root Cause

Your `.env` file contains **identical values** for `BINANCE_API_KEY` and `BINANCE_API_SECRET`:

```bash
BINANCE_API_KEY=mR0qYeuJGgFdSyEQOjxJ52KIX16xCjeCEswnPRkIVvE02a6b1STdSvgvW0ez0zUi
BINANCE_API_SECRET=mR0qYeuJGgFdSyEQOjxJ52KIX16xCjeCEswnPRkIVvE02a6b1STdSvgvW0ez0zUi
                                      ↑↑↑ THE SAME VALUE! ↑↑↑
```

**API Key and Secret must be DIFFERENT strings!**

---

## Fix Steps

### 1. Get Correct API Credentials from Binance

Go to: https://www.binance.com/en/my/settings/api-management

**Option A: Create New API Key (Recommended)**
1. Click "Create API"
2. Label it: "Cognee-Mainnet"
3. **Enable Permissions:**
   - ✅ Enable Reading
   - ✅ Enable Futures
4. Set IP Restrictions (optional but recommended)
5. Complete 2FA verification
6. **Save both values** - the secret is shown only once!

**Option B: Verify Existing Key**
1. Find your existing API key
2. Verify permissions are enabled (Reading + Futures)
3. If needed, regenerate the secret

### 2. Update Your `.env` File

Replace the values with your **correct, different** API credentials:

```bash
# CORRECT - Different values:
BINANCE_API_KEY=LpV3kD3f9TqR8sJ7nM4pW2xZ6qV1aB9eC5fH8jK0uN2o (example)
BINANCE_API_SECRET=sY9mN2vB5xK7pQ1rT4wE8aD6zC3fG0hJ2nM5 (different!)
```

### 3. Test the Fix

Run the test script:

```bash
./test_mainnet_quick.sh
```

**Expected success output:**
```json
{"totalWalletBalance":"123.45","availableBalance":"100.00","totalUnrealizedProfit":"0.00"}
```

### 4. Run Your Bot

Once the test works, start Cognee:

```bash
./cognee
```

---

## Troubleshooting

### If you still get errors:

**Error -1022 (Signature Invalid)**
- API key and secret still don't match
- Secret has a typo
- Using testnet keys on mainnet

**Error -2015 (Invalid API Key)**
- API key doesn't exist in Binance
- API key is expired
- Futures permissions not enabled

**Error -1022 (IP Restriction)**
- Your IP is not whitelisted
- Get your IP: `curl ifconfig.me`
- Add to API settings in Binance

---

## Quick Test Commands

**Test without authentication:**
```bash
curl "https://fapi.binance.com/fapi/v1/time"
```

**Test with authentication:**
```bash
export $(awk -F'=' '/^BINANCE_API_KEY=/ {print $0}' .env) && \
export $(awk -F'=' '/^BINANCE_API_SECRET=/ {print $0}' .env) && \
TIMESTAMP=$(date +%s000) && \
SIGNATURE=$(echo -n "timestamp=$TIMESTAMP" | openssl dgst -sha256 -hmac "$BINANCE_API_SECRET" | sed 's/^.* //') && \
curl -H "X-MBX-APIKEY: $BINANCE_API_KEY" \
  "https://fapi.binance.com/fapi/v2/account?timestamp=$TIMESTAMP&signature=$SIGNATURE"
```

---

## Scripts Provided

- `./fix_mainnet_keys.sh` - Diagnoses and explains the problem
- `./test_mainnet_quick.sh` - Quick test of API connectivity
- `./test_mainnet_simple.sh` - Simple formatted test (if env parsing works)
- `verify_mainnet_credentials.sh` - Verification script (created by fix script)

---

## Key Takeaway

**API Key ≠ API Secret**

These must be two **different** strings from Binance. The secret is used to generate cryptographic signatures, and if it's identical to the key, the signature validation will always fail.

Testnet worked because you likely had the correct separate key/secret values for testnet.