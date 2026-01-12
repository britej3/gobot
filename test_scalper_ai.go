package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/britebrt/cognee/pkg/brain"
)

func main() {
	fmt.Println("🚀 Testing GOBOT Aggressive Scalper AI...")
	
	// Create a test brain engine
	config := brain.DefaultBrainConfig()
	config.LocalModel = "LiquidAI/LFM2.5-1.2B-Instruct-GGUF/LFM2.5-1.2B-Instruct-Q8_0.gguf"
	config.LocalBaseURL = "http://localhost:11454"
	config.InferenceMode = "LOCAL"
	
	engine, err := brain.NewBrainEngine(nil, nil, config)
	if err != nil {
		log.Fatal("Failed to create brain engine:", err)
	}
	
	// Test trading signal
	signal := struct {
		Symbol        string  `json:"symbol"`
		FVGZone       string  `json:"fvg_zone"`
		FVGConfidence float64 `json:"fvg_confidence"`
		CVDDivergence bool    `json:"cvd_divergence"`
		Volatility    float64 `json:"volatility"`
		MarketRegime  string  `json:"market_regime"`
		Confidence    float64 `json:"confidence"`
	}{
		Symbol:        "BTCUSDT",
		FVGZone:       "BULLISH",
		FVGConfidence: 0.82,
		CVDDivergence: true,
		Volatility:    0.018,
		MarketRegime:  "RANGING",
		Confidence:    0.85,
	}
	
	fmt.Println("📊 Sending trading signal to AI...")
	signalJSON, _ := json.MarshalIndent(signal, "", "  ")
	fmt.Println(string(signalJSON))
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	decision, err := engine.MakeTradingDecision(ctx, signal)
	if err != nil {
		log.Fatal("Failed to get trading decision:", err)
	}
	
	fmt.Println("\n🎯 AI TRADING DECISION:")
	decisionJSON, _ := json.MarshalIndent(decision, "", "  ")
	fmt.Println(string(decisionJSON))
	
	// Test market analysis
	marketData := struct {
		Price       float64 `json:"price"`
		Volume      float64 `json:"volume"`
		Volatility  float64 `json:"volatility"`
		Spread      float64 `json:"spread"`
		Liquidity   float64 `json:"liquidity"`
	}{
		Price:      49400,
		Volume:     1250000,
		Volatility: 0.018,
		Spread:     0.0005,
		Liquidity:  85000000,
	}
	
	fmt.Println("\n📈 Requesting market analysis...")
	analysis, err := engine.AnalyzeMarket(ctx, marketData)
	if err != nil {
		log.Fatal("Failed to get market analysis:", err)
	}
	
	fmt.Println("🧠 AI MARKET ANALYSIS:")
	analysisJSON, _ := json.MarshalIndent(analysis, "", "  ")
	fmt.Println(string(analysisJSON))
}