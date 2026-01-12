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
	fmt.Println("🚀 Testing GOBOT Aggressive Scalper AI - Direct Engine Test...")
	
	// Create a test brain engine
	config := brain.DefaultBrainConfig()
	config.LocalModel = "LiquidAI/LFM2.5-1.2B-Instruct-GGUF/LFM2.5-1.2B-Instruct-Q8_0.gguf"
	config.LocalBaseURL = "http://localhost:11454"
	config.InferenceMode = "LOCAL"
	
	engine, err := brain.NewBrainEngine(nil, nil, config)
	if err != nil {
		log.Fatal("Failed to create brain engine:", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Test 1: Trading Decision
	fmt.Println("🎯 TEST 1: Trading Decision")
	tradingSignal := struct {
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
	
	signalJSON, _ := json.MarshalIndent(tradingSignal, "", "  ")
	fmt.Println("Input Signal:")
	fmt.Println(string(signalJSON))
	
	fmt.Println("\n🤖 AI ANALYZING...")
	decision, err := engine.MakeTradingDecision(ctx, tradingSignal)
	if err != nil {
		fmt.Printf("❌ Trading decision error: %v\n", err)
		fmt.Println("This might be due to JSON parsing - checking raw response...")
	} else {
		fmt.Println("✅ TRADING DECISION:")
		decisionJSON, _ := json.MarshalIndent(decision, "", "  ")
		fmt.Println(string(decisionJSON))
	}
	
	// Test 2: Market Analysis
	fmt.Println("\n📈 TEST 2: Market Analysis")
	marketData := struct {
		Price      float64 `json:"price"`
		Volume     float64 `json:"volume"`
		Volatility float64 `json:"volatility"`
		Spread     float64 `json:"spread"`
		Liquidity  float64 `json:"liquidity"`
	}{
		Price:      49400,
		Volume:     1250000,
		Volatility: 0.018,
		Spread:     0.0005,
		Liquidity:  85000000,
	}
	
	marketJSON, _ := json.MarshalIndent(marketData, "", "  ")
	fmt.Println("Market Data:")
	fmt.Println(string(marketJSON))
	
	fmt.Println("\n🧠 AI ANALYZING MARKET...")
	analysis, err := engine.AnalyzeMarket(ctx, marketData)
	if err != nil {
		fmt.Printf("❌ Market analysis error: %v\n", err)
	} else {
		fmt.Println("✅ MARKET ANALYSIS:")
		analysisJSON, _ := json.MarshalIndent(analysis, "", "  ")
		fmt.Println(string(analysisJSON))
	}
	
	fmt.Println("\n🎉 Test completed!")
}