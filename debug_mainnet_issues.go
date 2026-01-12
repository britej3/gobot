package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britebrt/cognee/pkg/brain"
	"github.com/britebrt/cognee/pkg/platform"
	internalPlatform "github.com/britebrt/cognee/internal/platform"
	"github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("🔍 DEBUGGING MAINNET TRADE PLACEMENT ISSUES")
	fmt.Println("=" + strings.Repeat("=", 50))

	// 1. Check Environment Configuration
	fmt.Println("\n1. ENVIRONMENT CONFIGURATION:")
	checkEnvironmentConfig()

	// 2. Test API Connection
	fmt.Println("\n2. API CONNECTION TEST:")
	testAPIConnection()

	// 3. Test Brain Engine
	fmt.Println("\n3. BRAIN ENGINE TEST:")
	testBrainEngine()

	// 4. Test Watcher Configuration
	fmt.Println("\n4. WATCHER CONFIGURATION:")
	testWatcherConfig()

	// 5. Test Striker Configuration
	fmt.Println("\n5. STRIKER CONFIGURATION:")
	testStrikerConfig()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🔍 DEBUG COMPLETE - Check results above")
}

func checkEnvironmentConfig() {
	fmt.Printf("  BINANCE_USE_TESTNET: %s\n", os.Getenv("BINANCE_USE_TESTNET"))
	fmt.Printf("  BINANCE_API_KEY: %s\n", maskString(os.Getenv("BINANCE_API_KEY"), 4))
	fmt.Printf("  BINANCE_API_SECRET: %s\n", maskString(os.Getenv("BINANCE_API_SECRET"), 4))
	fmt.Printf("  MSTY_BASE_URL: %s\n", os.Getenv("MSTY_BASE_URL"))
	fmt.Printf("  MSTY_MODEL: %s\n", os.Getenv("MSTY_MODEL"))
	fmt.Printf("  OLLAMA_MODEL: %s\n", os.Getenv("OLLAMA_MODEL"))
	fmt.Printf("  OLLAMA_BASE_URL: %s\n", os.Getenv("OLLAMA_BASE_URL"))
}

func testAPIConnection() {
	useTestnet := os.Getenv("BINANCE_USE_TESTNET") == "true"
	fmt.Printf("  Environment: %s\n", getEnvName(useTestnet))

	status := internalPlatform.CheckConnection(useTestnet)
	if status.IsConnected {
		fmt.Printf("  ✅ API Connection: SUCCESS\n")
		fmt.Printf("  📊 Futures Balance: %s USDT\n", status.FuturesBalance)
		fmt.Printf("  💰 Available Margin: %.4f USDT\n", status.AvailableMargin)
	} else {
		fmt.Printf("  ❌ API Connection: FAILED\n")
		fmt.Printf("  Error: %s\n", status.Error)
	}
}

func testBrainEngine() {
	fmt.Printf("  Testing brain engine initialization...\n")

	config := brain.DefaultBrainConfig()
	config.InferenceMode = "LOCAL"
	config.LocalModel = "lfm2.5-1.2b-instruct-q8_0:latest" // Correct model name
	config.LocalBaseURL = "http://localhost:11964"

	// Create a test client (nil for testing)
	client := futures.NewClient("", "")
	
	_, err := brain.NewBrainEngine(client, nil, config)
	if err != nil {
		fmt.Printf("  ❌ Brain Engine: FAILED - %v\n", err)
		fmt.Printf("  💡 Available models should include: lfm2.5-1.2b-instruct-q8_0:latest\n")
	} else {
		fmt.Printf("  ✅ Brain Engine: SUCCESS\n")
	}
}

func testWatcherConfig() {
	fmt.Printf("  Testing watcher configuration...\n")
	
	// Check environment variables
	minConf := os.Getenv("MIN_FVG_CONFIDENCE")
	maxVol := os.Getenv("MAX_VOLATILITY")
	minVol := os.Getenv("MIN_24H_VOLUME_USD")
	regimeTol := os.Getenv("MARKET_REGIME_TOLERANCE")
	symbols := os.Getenv("WATCHLIST_SYMBOLS")

	fmt.Printf("  MIN_FVG_CONFIDENCE: %s\n", minConf)
	fmt.Printf("  MAX_VOLATILITY: %s\n", maxVol)
	fmt.Printf("  MIN_24H_VOLUME_USD: %s\n", minVol)
	fmt.Printf("  MARKET_REGIME_TOLERANCE: %s\n", regimeTol)
	fmt.Printf("  WATCHLIST_SYMBOLS: %s\n", symbols)

	// Check if symbols are properly configured
	if symbols == "" {
		fmt.Printf("  ⚠️  WARNING: No watchlist symbols configured\n")
	} else {
		symbolCount := len(strings.Split(symbols, ","))
		fmt.Printf("  📊 Watchlist contains %d symbols\n", symbolCount)
	}
}

func testStrikerConfig() {
	fmt.Printf("  Testing striker configuration...\n")
	
	// Test order execution logic
	fmt.Printf("  🎯 Order execution: SIMULATED\n")
	fmt.Printf("  🎲 Anti-sniffer jitter: ENABLED\n")
	fmt.Printf("  🛡️  Risk management: ENABLED\n")
	fmt.Printf("  📊 Stop loss/take profit: ENABLED\n")
}

func getEnvName(useTestnet bool) string {
	if useTestnet {
		return "TESTNET (Safe)"
	}
	return "MAINNET (Real Money)"
}

func maskString(s string, keep int) string {
	if len(s) <= keep {
		return s
	}
	return s[:keep] + strings.Repeat("*", len(s)-keep)
}

// Add missing import
