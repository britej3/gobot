package platform

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/sirupsen/logrus"
)

// AccountStatus holds the financial health check results
type AccountStatus struct {
	SpotBalance       string  `json:"spot_balance"`
	FuturesBalance    string  `json:"futures_balance"`
	IsConnected       bool    `json:"is_connected"`
	Environment       string  `json:"environment"`
	TotalWalletValue  float64 `json:"total_wallet_value"`
	AvailableMargin   float64 `json:"available_margin"`
	PositionInitialMargin float64 `json:"position_initial_margin"`
	UnrealizedProfit  float64 `json:"unrealized_profit"`
	Error             string  `json:"error,omitempty"`
}

// CheckConnection performs a comprehensive API audit for both Testnet and Mainnet
func CheckConnection(useTestnet bool) *AccountStatus {
	// Use testnet keys when in testnet mode, otherwise use mainnet keys
	var apiKey, secretKey string
	if useTestnet {
		apiKey = os.Getenv("BINANCE_TESTNET_API")
		secretKey = os.Getenv("BINANCE_TESTNET_SECRET")
	} else {
		apiKey = os.Getenv("BINANCE_API_KEY")
		secretKey = os.Getenv("BINANCE_API_SECRET")
	}
	
	status := &AccountStatus{
		IsConnected: false,
		Environment: getEnvName(useTestnet),
	}
	
	logrus.WithFields(logrus.Fields{
		"environment": status.Environment,
		"has_api_key": len(apiKey) > 0,
	}).Info("🔍 Starting pre-flight API audit")
	
	// Validate API keys exist
	if apiKey == "" || secretKey == "" {
		status.Error = "BINANCE_API_KEY or BINANCE_API_SECRET not set"
		logrus.Error("❌ API keys not configured")
		return status
	}
	
	// Configure testnet if needed
	if useTestnet {
		futures.UseTestnet = true
		binance.UseTestnet = true
		logrus.Info("🧪 Using Binance Testnet for audit")
	} else {
		logrus.Info("💰 Using Binance Mainnet for audit")
		logrus.Warn("⚠️  Mainnet detected - real money trading environment")
	}
	
	// Create clients
	fClient := futures.NewClient(apiKey, secretKey)
	sClient := binance.NewClient(apiKey, secretKey)
	
	ctx := context.Background()
	
	// 1. Ping Futures Server (Primary connection test)
	logrus.Info("📡 Pinging Binance Futures API...")
	if err := fClient.NewPingService().Do(ctx); err != nil {
		status.Error = fmt.Sprintf("Futures API ping failed: %v", err)
		logrus.WithError(err).Error("❌ Futures API connection failed")
		return status
	}
	logrus.Info("✅ Futures API connection established")
	
	// 2. Ping Spot Server (Secondary connection test)
	logrus.Info("📡 Pinging Binance Spot API...")
	if err := sClient.NewPingService().Do(ctx); err != nil {
		logrus.WithError(err).Warn("⚠️  Spot API connection failed (non-critical)")
		// Continue - spot is not required for futures trading
	} else {
		logrus.Info("✅ Spot API connection established")
	}
	
	// 3. Fetch Futures Account Details (Primary balance check)
	logrus.Info("💰 Fetching Futures account details...")
	fAcc, err := fClient.NewGetAccountService().Do(ctx)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to fetch futures account: %v", err)
		logrus.WithError(err).Error("❌ Failed to fetch futures account")
		return status
	}
	
	// Parse futures balance
	if fAcc.TotalWalletBalance != "" {
		status.FuturesBalance = fAcc.TotalWalletBalance
		if val, err := parseFloatString(fAcc.TotalWalletBalance); err == nil {
			status.TotalWalletValue = val
		}
	}
	
	if fAcc.AvailableBalance != "" {
		if val, err := parseFloatString(fAcc.AvailableBalance); err == nil {
			status.AvailableMargin = val
		}
	}
	
	status.PositionInitialMargin = parseFloatSafe(fAcc.TotalPositionInitialMargin)
	status.UnrealizedProfit = parseFloatSafe(fAcc.TotalUnrealizedProfit)
	
	logrus.WithFields(logrus.Fields{
		"total_wallet_balance": status.FuturesBalance,
		"available_margin":     status.AvailableMargin,
		"unrealized_pnl":       status.UnrealizedProfit,
	}).Info("📊 Futures account details retrieved")
	
	// 4. Fetch Spot Account (Mainnet only, for comprehensive overview)
	if !useTestnet {
		logrus.Info("💰 Fetching Spot account details...")
		acc, err := sClient.NewGetAccountService().Do(ctx)
		if err != nil {
			logrus.WithError(err).Warn("⚠️  Failed to fetch spot account (non-critical)")
		} else {
			// Find USDT balance
			for _, bal := range acc.Balances {
				if bal.Asset == "USDT" && (bal.Free != "" || bal.Locked != "") {
					totalUSDT := parseFloatSafe(bal.Free) + parseFloatSafe(bal.Locked)
					status.SpotBalance = fmt.Sprintf("%.6f", totalUSDT)
					logrus.WithField("usdt_balance", status.SpotBalance).Info("📊 Spot USDT balance retrieved")
					break
				}
			}
		}
	} else {
		status.SpotBalance = "N/A (Testnet)"
		logrus.Info("📊 Spot balance skipped (Testnet environment)")
	}
	
	// 5. Check API permissions by testing a lightweight endpoint
	logrus.Info("🔑 Verifying API permissions...")
	serverTime, err := fClient.NewServerTimeService().Do(ctx)
	if err != nil {
		logrus.WithError(err).Warn("⚠️  API permissions may be limited")
	} else {
		logrus.WithField("server_time", serverTime).Info("✅ API permissions verified")
	}
	
	status.IsConnected = true
	
	// 6. Safety checks for mainnet
	if !useTestnet {
		logrus.Warn("🚨 MAINNET SAFETY CHECKS:")
		logrus.Warn("- Ensure API key has 'Enable Futures' permission")
		logrus.Warn("- Ensure API key has 'Reading' permission") 
		logrus.Warn("- Ensure 'Enable Withdrawals' is DISABLED")
		logrus.Warn("- Consider IP whitelist for security")
		
		if status.TotalWalletValue > 0 {
			logrus.WithField("balance", status.TotalWalletValue).Warn("💰 Real money detected - trade carefully!")
		}
	}
	
	return status
}

// parseFloatString safely parses a float string
func parseFloatString(s string) (float64, error) {
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	return val, err
}

// parseFloatSafe safely parses a float string with default 0
func parseFloatSafe(s string) float64 {
	if s == "" {
		return 0
	}
	val, err := parseFloatString(s)
	if err != nil {
		return 0
	}
	return val
}

// getEnvName returns friendly environment name
func getEnvName(useTestnet bool) string {
	if useTestnet {
		return "TESTNET (Safe)"
	}
	return "MAINNET (Real Money)"
}

// PrintAuditReport displays a formatted audit report
func PrintAuditReport(status *AccountStatus) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🏦 COGNEE SYSTEM AUDIT REPORT")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Environment:     %s\n", status.Environment)
	fmt.Printf("API Connection:  %s\n", getStatusIcon(status.IsConnected))
	
	if status.IsConnected {
		fmt.Printf("Spot USDT:       %s\n", status.SpotBalance)
		fmt.Printf("Futures Wallet:  %s\n", status.FuturesBalance)
		fmt.Printf("Available Margin: %.4f USDT\n", status.AvailableMargin)
		fmt.Printf("Total Wallet:    %.4f USDT\n", status.TotalWalletValue)
		
		if status.UnrealizedProfit != 0 {
			fmt.Printf("Unrealized PnL:  %.4f USDT\n", status.UnrealizedProfit)
		}
		
		// Safety warnings
		if !strings.Contains(status.Environment, "Testnet") && status.TotalWalletValue > 0 {
			fmt.Println("\n⚠️  SAFETY REMINDERS:")
			fmt.Println("- Ensure 'Enable Withdrawals' is DISABLED")
			fmt.Println("- Monitor API usage regularly")
			fmt.Println("- Consider IP whitelisting for security")
		}
	} else {
		fmt.Printf("\n❌ ERROR: %s\n", status.Error)
		fmt.Println("\nTroubleshooting:")
		fmt.Println("- Verify BINANCE_API_KEY and BINANCE_API_SECRET are set")
		fmt.Println("- Check IP whitelist settings in Binance")
		fmt.Println("- Ensure API key has 'Reading' and 'Enable Futures' permissions")
	}
	
	fmt.Println(strings.Repeat("=", 60))
}

// getStatusIcon returns a status emoji
func getStatusIcon(connected bool) string {
	if connected {
		return "✅ ONLINE"
	}
	return "❌ OFFLINE"
}