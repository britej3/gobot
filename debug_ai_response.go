package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// MSTYRequest represents a request to MSTY API (OpenAI-compatible)
type MSTYRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Temperature float64 `json:"temperature"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MSTYResponse represents a response from MSTY API (OpenAI-compatible)
type MSTYResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func main() {
	fmt.Println("🔍 Debugging AI Response from MSTY...")
	
	// Test the exact prompt that would be sent
	prompt := `You are GOBOT's trading decision AI powered by LiquidAI LFM2.5. Evaluate this trading signal for ultra-high-frequency scalping:

Signal Data: {"symbol": "BTCUSDT", "fvg_confidence": 0.82, "cvd_divergence": true, "volatility": 0.018, "market_regime": "RANGING"}

Decision criteria for LFM2.5:
- FVG confidence > 0.75
- CVD divergence present and strong
- Volatility within optimal range (0.5-2.0%)
- No high-impact news events
- Market microstructure favorable

Provide your decision in JSON format only, no additional text or explanation:
{
  "decision": "BUY/SELL/HOLD",
  "confidence": 0.85,
  "reasoning": "Strong bullish FVG at $49,400 with CVD divergence",
  "risk_level": "LOW/MEDIUM/HIGH",
  "recommended_leverage": 20,
  "fvg_confidence": 0.82,
  "cvd_divergence": true
}`

	// Create MSTY-compatible request
	request := MSTYRequest{
		Model: "LiquidAI/LFM2.5-1.2B-Instruct-GGUF/LFM2.5-1.2B-Instruct-Q8_0.gguf",
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.05,
		Stream: false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		log.Fatal("Failed to marshal request:", err)
	}

	fmt.Println("📤 Sending request to MSTY...")
	fmt.Printf("URL: http://localhost:11454/v1/chat/completions\n")
	fmt.Printf("Model: %s\n", request.Model)
	fmt.Printf("Temperature: %.2f\n", request.Temperature)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), "POST", "http://localhost:11454/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal("Failed to create request:", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Request failed:", err)
	}
	defer resp.Body.Close()

	fmt.Printf("\n📥 Response Status: %d\n", resp.StatusCode)
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read response:", err)
	}

	fmt.Println("Raw Response Body:")
	fmt.Println(string(body))

	// Try to parse as MSTY response
	var mstyResp MSTYResponse
	if err := json.Unmarshal(body, &mstyResp); err != nil {
		fmt.Printf("\n❌ Failed to parse as MSTY response: %v\n", err)
		fmt.Println("Trying to parse as regular JSON...")
		
		// Try to parse as generic JSON
		var generic interface{}
		if err := json.Unmarshal(body, &generic); err != nil {
			fmt.Printf("❌ Failed to parse as JSON: %v\n", err)
		} else {
			fmt.Println("✅ Parsed as generic JSON:")
			prettyJSON, _ := json.MarshalIndent(generic, "", "  ")
			fmt.Println(string(prettyJSON))
		}
		return
	}

	if len(mstyResp.Choices) == 0 {
		fmt.Println("❌ No choices returned")
		return
	}

	fmt.Println("\n✅ Parsed MSTY Response:")
	fmt.Println("AI Content:")
	fmt.Println(mstyResp.Choices[0].Message.Content)
}