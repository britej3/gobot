package watcher

import (
	"fmt"
	"strings"
)

// StrikerPrompt generates the aggressive scalping prompt for LFM2.5
func StrikerPrompt(topAssets []ScoredAsset) string {
	if len(topAssets) == 0 {
		return ""
	}
	
	// Generate market data JSON for top 15 assets
	marketData := generateMarketDataJSON(topAssets)
	
	// Build the prompt
	prompt := fmt.Sprintf(`You are Cognee, an Aggressive Autonomous Scalper specialized in Binance Futures Mid-Cap assets. Your objective is to identify high-probability (Scalp) opportunities in the nearest future (1-15 minutes). You ignore long-term trends unless they create immediate momentum. You are decisive, cold, and mathematically driven.

**Market Context & Constraints**
- Timeframe: 1m and 5m (High-Frequency).
- Strategy: Breakout detection, Volatility Expansion, and Mean Reversion.
- Leverage: 20x (Slippage and fees are critical).
- Logic: If Volatility (ATR) is expanding and Volume is surging, look for a "Strike."

**Input Data (Top 15 Ranked by Volatility)**
%s

**Task Instructions**
1. Analyze the provided candidates for "Nearest Future" potential.
2. Filter out any asset where the spread or slippage would kill a 20x scalp.
3. Assign a confidence_score (0-100) based on technical confluence (e.g., RSI Divergence + Volume Spike + EMA Breakout).
4. Rank the Top 3 targets from the list.

**RULES:**
- Output ONLY JSON. No conversational filler.
- Only include assets with confidence_score >= 85
- Consider 20x leverage impact on slippage and fees
- Focus on 1-15 minute timeframe ONLY
- Be decisive and mathematically precise

**Expected JSON Schema:**
{
  "timestamp": "2026-01-09T07:45:00Z",
  "top_targets": [
    {
      "symbol": "ZECUSDT",
      "action": "LONG",
      "confidence_score": 92,
      "probability_reason": "Volatility expansion (1m > 5m) + Bullish FVG on 1m chart.",
      "entry_zone": 42.55,
      "take_profit": 42.98,
      "stop_loss": 42.40,
      "allocation_multiplier": 1.0
    }
  ],
  "market_regime": "VOLATILE_EXPANSION"
}`, marketData)
	
	return prompt
}

// MinimumConfidenceThreshold is the minimum confidence required to execute
const MinimumConfidenceThreshold = 85.0

// generateMarketDataJSON creates the market data portion of the prompt
func generateMarketDataJSON(assets []ScoredAsset) string {
	if len(assets) > 15 {
		assets = assets[:15] // Take top 15
	}
	
	var sb strings.Builder
	sb.WriteString("```json\n")
	sb.WriteString("[\n")
	
	for i, asset := range assets {
		// Format: Symbol, Price, ATR%, 1m Vol Spike, RSI, EMA-9 Distance, BTC Correlation
		sb.WriteString(fmt.Sprintf("  {\n"))
		sb.WriteString(fmt.Sprintf("    \"symbol\": \"%s\",\n", asset.Symbol))
		sb.WriteString(fmt.Sprintf("    \"price\": %.4f,\n", asset.CurrentPrice))
		sb.WriteString(fmt.Sprintf("    \"atr_percent\": %.2f,\n", asset.ATRPercent))
		sb.WriteString(fmt.Sprintf("    \"vol_spike_1m\": %.2f,\n", asset.VolumeLastMinute/asset.AvgVolume5Min))
		sb.WriteString(fmt.Sprintf("    \"rsi\": %.2f,\n", asset.RSI))
		sb.WriteString(fmt.Sprintf("    \"ema_9_distance\": %.4f,\n", (asset.CurrentPrice-asset.EMACurrent)/asset.EMACurrent*100))
		sb.WriteString(fmt.Sprintf("    \"btc_correlation\": %.2f,\n", 0.85)) // Placeholder - would fetch real correlation
		sb.WriteString(fmt.Sprintf("    \"volume_24h_usd\": %.0f,\n", asset.Volume24hUSD))
		sb.WriteString(fmt.Sprintf("    \"confidence\": %.2f\n", asset.Confidence))
		sb.WriteString(fmt.Sprintf("  }%s\n", map[bool]string{true: ",", false: ""}[i < len(assets)-1]))
	}
	
	sb.WriteString("]\n")
	sb.WriteString("```\n")
	
	return sb.String()
}