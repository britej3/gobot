# N8N + GOBOT Automation Plan

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                    N8N WORKFLOW                       │
└─────────────────────────────────────────────────────┘

1. GOBOT Scanner (Source)
   ↓
   Scans market, finds top assets
   ↓
   Writes to: /tmp/gobot_targets.json
   {"symbol": "BTCUSDT", "confidence": 0.85, ...}
   ↓
2. N8N Trigger (File Watch)
   ↓
   Detects new file: gobot_targets.json
   ↓
3. N8N - Screenshot Capture
   ├─ Opens chart page (TradingView/Binance)
   ├─ Selects 1m timeframe → capture
   ├─ Selects 5m timeframe → capture
   └─ Selects 15m timeframe → capture
   ↓
4. N8N - Financial Analyzer
   ├─ HTTP POST to analyzer website
   ├─ Uploads: 3 screenshots + ticker name
   └─ Receives: Trade recommendation
   ↓
5. N8N - Parse Response
   {
     "action": "BUY",
     "symbol": "BTCUSDT",
     "position_size": 0.001,
     "entry_price": 98000,
     "exit_price": 99000,
     "stop_loss": 97500,
     "take_profit": 99500,
     "confidence": 0.85
   }
   ↓
6. N8N - Send to GOBOT
   ↓
   Writes to: /tmp/gobot_trades.json
   ↓
7. GOBOT - Trade Executor (Consumer)
   ↓
   Monitors: /tmp/gobot_trades.json
   ↓
   Reads trade command
   ↓
   Executes order on Binance
   ↓
   Writes result: /tmp/gobot_trade_results.json
   ↓
8. N8N - Verify & Log
   ↓
   Reads: gobot_trade_results.json
   ↓
   Logs to database/file
   ↓
   Send notification (optional)
```

## Communication Protocol

### 1. Scanner → N8N

**File:** `/tmp/gobot_targets.json`

```json
{
  "timestamp": "2026-01-13T07:30:00Z",
  "targets": [
    {
      "symbol": "BTCUSDT",
      "current_price": 98000.50,
      "confidence": 0.85,
      "volatility": 1.5,
      "volume_spike": true
    },
    {
      "symbol": "ETHUSDT",
      "current_price": 3200.75,
      "confidence": 0.78,
      "volatility": 1.2,
      "volume_spike": false
    }
  ]
}
```

### 2. N8N → GOBOT (Trade Command)

**File:** `/tmp/gobot_trades.json`

```json
{
  "trade_id": "trade_20260113_073000",
  "source": "financial_analyzer",
  "action": "BUY",
  "symbol": "BTCUSDT",
  "position_size": 0.001,
  "entry_price": 98000.00,
  "stop_loss": 97510.00,
  "take_profit": 98500.00,
  "confidence": 0.85,
  "reasoning": "Bullish FVG with strong volume",
  "timestamp": "2026-01-13T07:30:15Z"
}
```

### 3. GOBOT → N8N (Execution Result)

**File:** `/tmp/gobot_trade_results.json`

```json
{
  "trade_id": "trade_20260113_073000",
  "status": "EXECUTED",
  "order_id": "1234567890",
  "executed_price": 98001.25,
  "executed_quantity": 0.001,
  "timestamp": "2026-01-13T07:30:16Z",
  "error": null
}
```

## GOBOT Changes Required

### 1. File-Based Target Output (Scanner)

**File:** `internal/watcher/scanner.go`

Add function to write targets to file:

```go
func (s *AssetScanner) writeTargetsToFile(targets []ScoredAsset) error {
    data := map[string]interface{}{
        "timestamp": time.Now().Format(time.RFC3339),
        "targets":   targets,
    }
    
    jsonData, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile("/tmp/gobot_targets.json", jsonData, 0644)
}
```

### 2. Trade Command Consumer (New File)

**Create:** `internal/executor/file_consumer.go`

```go
package executor

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"

    "github.com/adshao/go-binance/v2/futures"
    "github.com/sirupsen/logrus"
)

type FileConsumer struct {
    client     *futures.Client
    targetFile string
    resultFile string
    stopChan   chan struct{}
}

type TradeCommand struct {
    TradeID      string  `json:"trade_id"`
    Source       string  `json:"source"`
    Action       string  `json:"action"`
    Symbol       string  `json:"symbol"`
    PositionSize float64 `json:"position_size"`
    EntryPrice   float64 `json:"entry_price"`
    StopLoss     float64 `json:"stop_loss"`
    TakeProfit   float64 `json:"take_profit"`
    Confidence   float64 `json:"confidence"`
    Reasoning    string  `json:"reasoning"`
    Timestamp    string  `json:"timestamp"`
}

type TradeResult struct {
    TradeID         string  `json:"trade_id"`
    Status          string  `json:"status"`
    OrderID         string  `json:"order_id"`
    ExecutedPrice   float64 `json:"executed_price"`
    ExecutedQty     float64 `json:"executed_quantity"`
    Timestamp       string  `json:"timestamp"`
    Error           string  `json:"error"`
}

func NewFileConsumer(client *futures.Client) *FileConsumer {
    return &FileConsumer{
        client:     client,
        targetFile: "/tmp/gobot_trades.json",
        resultFile: "/tmp/gobot_trade_results.json",
        stopChan:   make(chan struct{}),
    }
}

func (fc *FileConsumer) Start() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    logrus.Info("Trade command consumer started")
    
    for {
        select {
        case <-ticker.C:
            fc.checkAndExecuteTrades()
        case <-fc.stopChan:
            return
        }
    }
}

func (fc *FileConsumer) checkAndExecuteTrades() {
    // Check if trade command file exists and has content
    if _, err := os.Stat(fc.targetFile); os.IsNotExist(err) {
        return // No trade to execute
    }
    
    // Read trade command
    data, err := os.ReadFile(fc.targetFile)
    if err != nil {
        logrus.WithError(err).Error("Failed to read trade command")
        return
    }
    
    var command TradeCommand
    if err := json.Unmarshal(data, &command); err != nil {
        logrus.WithError(err).Error("Failed to parse trade command")
        return
    }
    
    // Check if already executed (by checking results file)
    if fc.isTradeExecuted(command.TradeID) {
        return // Already processed
    }
    
    // Execute trade
    result := fc.executeTrade(command)
    
    // Write result
    fc.writeResult(result)
    
    // Archive command file
    fc.archiveCommand(command.TradeID)
}

func (fc *FileConsumer) executeTrade(cmd TradeCommand) TradeResult {
    logrus.WithFields(logrus.Fields{
        "trade_id": cmd.TradeID,
        "symbol":    cmd.Symbol,
        "action":    cmd.Action,
        "source":    cmd.Source,
    }).Info("Executing trade command from file")
    
    var side futures.SideType
    if cmd.Action == "BUY" {
        side = futures.SideTypeBuy
    } else if cmd.Action == "SELL" {
        side = futures.SideTypeSell
    } else {
        return TradeResult{
            TradeID: cmd.TradeID,
            Status:  "FAILED",
            Error:   fmt.Sprintf("Invalid action: %s", cmd.Action),
            Timestamp: time.Now().Format(time.RFC3339),
        }
    }
    
    // Execute market order
    order, err := fc.client.NewCreateOrderService().
        Symbol(cmd.Symbol).
        Side(side).
        Type(futures.OrderTypeMarket).
        Quantity(fmt.Sprintf("%.6f", cmd.PositionSize)).
        Do(context.Background())
    
    if err != nil {
        return TradeResult{
            TradeID: cmd.TradeID,
            Status:  "FAILED",
            Error:   err.Error(),
            Timestamp: time.Now().Format(time.RFC3339),
        }
    }
    
    logrus.WithFields(logrus.Fields{
        "trade_id":     cmd.TradeID,
        "symbol":       cmd.Symbol,
        "order_id":     order.OrderID,
        "executed_price": parsePrice(order.AvgPrice),
    }).Info("Trade executed successfully")
    
    return TradeResult{
        TradeID:       cmd.TradeID,
        Status:        "EXECUTED",
        OrderID:       fmt.Sprintf("%d", order.OrderID),
        ExecutedPrice: parsePrice(order.AvgPrice),
        ExecutedQty:   cmd.PositionSize,
        Timestamp:     time.Now().Format(time.RFC3339),
        Error:         nil,
    }
}

func (fc *FileConsumer) writeResult(result TradeResult) {
    // Read existing results
    var results []TradeResult
    if data, err := os.ReadFile(fc.resultFile); err == nil {
        json.Unmarshal(data, &results)
    }
    
    // Append new result
    results = append(results, result)
    
    // Write back
    data, _ := json.MarshalIndent(results, "", "  ")
    os.WriteFile(fc.resultFile, data, 0644)
    
    logrus.WithField("trade_id", result.TradeID).Info("Trade result written")
}

func (fc *FileConsumer) isTradeExecuted(tradeID string) bool {
    data, err := os.ReadFile(fc.resultFile)
    if err != nil {
        return false
    }
    
    var results []TradeResult
    if err := json.Unmarshal(data, &results); err != nil {
        return false
    }
    
    for _, result := range results {
        if result.TradeID == tradeID {
            return true
        }
    }
    return false
}

func (fc *FileConsumer) archiveCommand(tradeID string) {
    archivePath := filepath.Join("/tmp/gobot_trades_archive", tradeID+".json")
    os.MkdirAll(filepath.Dir(archivePath), 0755)
    os.Rename(fc.targetFile, archivePath)
}

func (fc *FileConsumer) Stop() {
    close(fc.stopChan)
}

func parsePrice(s string) float64 {
    var f float64
    fmt.Sscanf(s, "%f", &f)
    return f
}
```

### 3. Integration into Platform

**File:** `pkg/platform/platform.go`

```go
import (
    "github.com/britebrt/cognee/internal/executor"
)

type Platform struct {
    ...
    tradeConsumer *executor.FileConsumer
    ...
}

func (p *Platform) initTradeConsumer() error {
    p.tradeConsumer = executor.NewFileConsumer(p.client)
    go p.tradeConsumer.Start()
    logrus.Info("Trade command consumer started")
    return nil
}

func (p *Platform) Start() error {
    ...
    // Initialize trade consumer
    if err := p.initTradeConsumer(); err != nil {
        logrus.WithError(err).Warn("Failed to init trade consumer")
    }
    ...
}
```

## N8N Workflow Design

### Node 1: Trigger (File Watch)

- **Type:** File Watch (n8n-node-file-watch)
- **Settings:**
  - Path: `/tmp/gobot_targets.json`
  - Watch for: Create/Modify
  - Poll interval: 5 seconds

### Node 2: Read File

- **Type:** Read Binary File
- **Input:** File path from Node 1
- **Output:** File content (JSON)

### Node 3: Parse JSON

- **Type:** Code
- **Code:**
```javascript
// Parse targets JSON
const data = $input.item;
const targets = JSON.parse(data).targets;

return targets.map(t => ({
  json: t,
  symbol: t.symbol,
  current_price: t.current_price
}));
```

### Node 4: Screenshot - 1m

- **Type:** HTTP Request (Puppeteer/Playwright)
- **URL:** TradingView/Binance chart
- **Settings:**
  - Symbol: `{{ $json.symbol }}`
  - Timeframe: 1m
  - Screenshot: Save to `/tmp/screenshots/{{ $json.symbol }}_1m.png`

### Node 5: Screenshot - 5m

- **Type:** HTTP Request (Puppeteer/Playwright)
- **URL:** TradingView/Binance chart
- **Settings:**
  - Symbol: `{{ $json.symbol }}`
  - Timeframe: 5m
  - Screenshot: Save to `/tmp/screenshots/{{ $json.symbol }}_5m.png`

### Node 6: Screenshot - 15m

- **Type:** HTTP Request (Puppeteer/Playwright)
- **URL:** TradingView/Binance chart
- **Settings:**
  - Symbol: `{{ $json.symbol }}`
  - Timeframe: 15m
  - Screenshot: Save to `/tmp/screenshots/{{ $json.symbol }}_15m.png`

### Node 7: Read Screenshots (Binary)

- **Type:** Read Binary File (3 nodes in parallel)
- **Output:** Base64 encoded images

### Node 8: Prepare Analyzer Request

- **Type:** Code
- **Code:**
```javascript
// Prepare multipart form data for analyzer
const request = {
  ticker: $input.item.symbol,
  screenshots: {
    "1m": $binary1,
    "5m": $binary2,
    "15m": $binary3
  },
  timestamp: new Date().toISOString()
};

return { json: request };
```

### Node 9: Send to Financial Analyzer

- **Type:** HTTP Request
- **Method:** POST
- **URL:** Your Financial Analyzer API endpoint
- **Body Type:** Form-Data (multipart)
- **Fields:**
  - ticker: `{{ $json.ticker }}`
  - screenshot_1m: `{{ $json.screenshots["1m"] }}`
  - screenshot_5m: `{{ $json.screenshots["5m"] }}`
  - screenshot_15m: `{{ $json.screenshots["15m"] }}`

### Node 10: Parse Analyzer Response

- **Type:** Code
- **Code:**
```javascript
// Parse analyzer response
const response = $input.item;
const trade = {
  trade_id: "trade_" + Date.now(),
  source: "financial_analyzer",
  action: response.action,
  symbol: response.ticker,
  position_size: response.position_size,
  entry_price: response.entry,
  stop_loss: response.stop_loss,
  take_profit: response.take_profit,
  confidence: response.confidence,
  reasoning: response.reasoning,
  timestamp: new Date().toISOString()
};

return { json: trade };
```

### Node 11: Write Trade Command

- **Type:** Write Binary File
- **Path:** `/tmp/gobot_trades.json`
- **Content:** `{{ JSON.stringify($json) }}`

### Node 12: Wait for Result (Loop)

- **Type:** Wait
- **Duration:** 30 seconds

### Node 13: Read Execution Result

- **Type:** Read Binary File
- **Path:** `/tmp/gobot_trade_results.json`

### Node 14: Parse & Log Result

- **Type:** Code
- **Code:**
```javascript
// Find latest result
const results = JSON.parse($input.item);
const latest = results[results.length - 1];

if (latest.status === 'EXECUTED') {
  console.log('Trade executed:', latest.trade_id);
} else {
  console.log('Trade failed:', latest.error);
}

return { json: latest };
```

### Node 15: Send Notification (Optional)

- **Type:** Telegram/Discord/Email
- **Content:** Trade execution summary

## N8N Setup Instructions

### Prerequisites

1. **Install n8n:**
```bash
# Using npm
npm install -g n8n

# Or using Docker
docker run -it --rm \
  --name n8n \
  -p 5678:5678 \
  n8nio/n8n
```

2. **Install Browser Automation Node:**
```bash
cd ~/.n8n
npm install n8n-nodes-puppeteer
# OR
npm install n8n-nodes-playwright
```

3. **Start n8n:**
```bash
n8n start
```

Access: http://localhost:5678

### Create Workflow

1. **Import Workflow:**
   - Copy the JSON workflow template
   - Import in n8n UI
   - Adjust file paths to match your system

2. **Configure Credentials:**
   - Financial Analyzer API (if needed)
   - Notification credentials (Telegram/Discord)

3. **Test Nodes:**
   - Test screenshot capture
   - Test analyzer connection
   - Test file writing

4. **Activate Workflow:**
   - Set to "Active"
   - Monitor execution logs

## Testing & Validation

### 1. Test Scanner Output

```bash
# Start GOBOT (scanner only writes to file)
./gobot-production

# Check if file is created
cat /tmp/gobot_targets.json
```

### 2. Test N8N Workflow

```bash
# Manually create test file
cat > /tmp/gobot_targets.json << 'EOF'
{
  "timestamp": "2026-01-13T07:30:00Z",
  "targets": [
    {
      "symbol": "BTCUSDT",
      "current_price": 98000,
      "confidence": 0.85
    }
  ]
}
EOF

# Trigger N8N manually or wait for file watch
# Monitor n8n execution logs
```

### 3. Test Screenshot Capture

- Verify screenshots are saved to `/tmp/screenshots/`
- Check image quality and content
- Ensure correct timeframe is captured

### 4. Test Financial Analyzer

- Manually send test request to analyzer
- Verify response format matches expected
- Check confidence and trade parameters

### 5. Test Trade Execution

```bash
# Manually create trade command
cat > /tmp/gobot_trades.json << 'EOF'
{
  "trade_id": "test_trade_001",
  "source": "test",
  "action": "BUY",
  "symbol": "BTCUSDT",
  "position_size": 0.001,
  "entry_price": 98000,
  "stop_loss": 97500,
  "take_profit": 98500,
  "confidence": 0.85,
  "reasoning": "Test trade",
  "timestamp": "2026-01-13T07:30:00Z"
}
EOF

# Check result
cat /tmp/gobot_trade_results.json
```

## Deployment

### 1. Development (Local)

- GOBOT running locally
- n8n running locally (localhost:5678)
- File-based communication

### 2. Production

- GOBOT: Docker container
- n8n: Docker container
- Shared volume: `/tmp` for file communication
- Database: PostgreSQL/MySQL for trade history

## Alternative: REST API Communication

If file-based is not preferred, can use REST API:

### GOBOT API Endpoint

```go
// cmd/cognee/api.go
package main

import (
    "encoding/json"
    "net/http"
)

func main() {
    http.HandleFunc("/api/trade", handleTradeCommand)
    http.ListenAndServe(":8080", nil)
}

func handleTradeCommand(w http.ResponseWriter, r *http.Request) {
    var cmd executor.TradeCommand
    json.NewDecoder(r.Body).Decode(&cmd)
    
    // Execute trade
    result := executeTrade(cmd)
    
    json.NewEncoder(w).Encode(result)
}
```

### N8N HTTP Request Node

- **URL:** http://localhost:8080/api/trade
- **Method:** POST
- **Body:** JSON trade command

## Security Considerations

1. **File Permissions:**
   - Ensure only GOBOT and n8n can access `/tmp` files
   - Use appropriate umask (0755)

2. **API Keys:**
   - Store Financial Analyzer API key in n8n credentials
   - Store Binance API key in GOBOT environment
   - Never commit keys to git

3. **Network:**
   - Use HTTPS for analyzer API
   - Consider VPN for security
   - Rate limiting on analyzer API

4. **Validation:**
   - Validate trade parameters before execution
   - Check position size limits
   - Verify symbol is valid

## Monitoring & Logging

### GOBOT Logs

```
Trade command consumer started
Executing trade command from file: trade_20260113_073000
Trade executed successfully: BTCUSDT BUY
Trade result written
```

### N8N Logs

```
[File Watch Trigger] New file detected: gobot_targets.json
[Screenshot 1m] Captured: BTCUSDT_1m.png
[Screenshot 5m] Captured: BTCUSDT_5m.png
[Screenshot 15m] Captured: BTCUSDT_15m.png
[Financial Analyzer] Response received
[Parse Response] Trade command prepared
[Write Trade Command] Written: gobot_trades.json
```

## Troubleshooting

### Issue: N8N not detecting file changes

**Solution:** Check file path permissions and use absolute paths

### Issue: Screenshots not captured

**Solution:** 
- Install puppeteer/playwright nodes
- Test screenshot URL in browser first
- Check for anti-bot protections

### Issue: Financial Analyzer not responding

**Solution:**
- Verify API endpoint is accessible
- Check request format (multipart/form-data)
- Check rate limits

### Issue: GOBOT not executing trades

**Solution:**
- Check trade command file format
- Verify file path matches
- Check GOBOT logs for errors

## Next Steps

1. **Install n8n** locally
2. **Create workflow** using template
3. **Test each node** individually
4. **Integrate with GOBOT** (file-based or API)
5. **End-to-end testing**
6. **Deploy to production**

## Estimated Time

- n8n Installation: 15 minutes
- Workflow Creation: 30-45 minutes
- Testing: 30-60 minutes
- Total: 1.5 - 2 hours

## Benefits

- **Separation of Concerns**
   - GOBOT: Trade execution only
   - n8n: Workflow automation
   - Analyzer: Technical analysis

- **Flexibility**
   - Easy to modify workflow
   - Can add/replace analyzer
   - Multiple analyzers possible

- **Visual Interface**
   - Drag-and-drop workflow design
   - Easy debugging
   - Visual logs

- **Scalability**
   - Can process multiple symbols
   - Parallel screenshot capture
   - Queue management
