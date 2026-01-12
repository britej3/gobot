package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/britebrt/cognee/pkg/brain"
)

func main() {
	fmt.Println("🚀 Testing GOBOT Aggressive Scalper AI - Simple Test...")
	
	// Create a test brain engine
	config := brain.DefaultBrainConfig()
	config.LocalModel = "LiquidAI/LFM2.5-1.2B-Instruct-GGUF/LFM2.5-1.2B-Instruct-Q8_0.gguf"
	config.LocalBaseURL = "http://localhost:11454"
	config.InferenceMode = "LOCAL"
	
	engine, err := brain.NewBrainEngine(nil, nil, config)
	if err != nil {
		log.Fatal("Failed to create brain engine:", err)
	}
	
	// Test simple prompt to see raw AI response
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	// Simple test prompt
	prompt := `You are GOBOT's trading decision AI powered by LiquidAI LFM2.5. 
Evaluate this trading signal for ultra-high-frequency scalping:

Signal Data: {"symbol": "BTCUSDT", "fvg_confidence": 0.82, "cvd_divergence": true, "volatility": 0.018}

Provide your decision in JSON format only, no additional text.`
	
	fmt.Println("🎯 Sending prompt to AI...")
	fmt.Println("Prompt:", prompt)
	
	response, err := engine.GetProvider().GenerateResponse(ctx, prompt)
	if err != nil {
		log.Fatal("Failed to get response:", err)
	}
	
	fmt.Println("\n🤖 RAW AI RESPONSE:")
	fmt.Println(response)
	
	// Test another prompt for market analysis
	marketPrompt := `You are GOBOT's market analysis AI. Analyze BTCUSDT market with volatility 0.018 and provide JSON analysis.`
	
	fmt.Println("\n📈 Testing market analysis...")
	marketResponse, err := engine.GetProvider().GenerateResponse(ctx, marketPrompt)
	if err != nil {
		log.Fatal("Failed to get market response:", err)
	}
	
	fmt.Println("\n📊 MARKET ANALYSIS RESPONSE:")
	fmt.Println(marketResponse)
}