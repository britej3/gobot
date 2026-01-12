#!/bin/bash

# One-liner to properly call Binance /fapi/v2/account endpoint
# This shows the correct format for including timestamp and signature

# Load API credentials from .env
export $(cat .env | grep -v '^#' | xargs)

# Settings
USE_TESTNET=${BINANCE_USE_TESTNET:-false}
TIMESTAMP=$(date +%s000)

# Determine endpoint
if [ "$USE_TESTNET" = "true" ]; then
    ENDPOINT="https://testnet.binancefuture.com/fapi/v2/account"
else
    ENDPOINT="https://fapi.binance.com/fapi/v2/account"
fi

# Generate signature
QUERY_STRING="timestamp=$TIMESTAMP"
SIGNATURE=$(echo -n "$QUERY_STRING" | openssl dgst -sha256 -hmac "$BINANCE_API_SECRET" | sed 's/^.* //')

# Make the API call (with proper authentication)
curl -H "X-MBX-APIKEY: $BINANCE_API_KEY" "${ENDPOINT}?${QUERY_STRING}&signature=${SIGNATURE}"
