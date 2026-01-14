# TradingView Screenshot Service

Microservice for capturing TradingView chart screenshots using Puppeteer.

## Features

- Express.js server on localhost:3456
- Single screenshot capture (`POST /capture`)
- Multiple timeframe capture (`POST /capture-multi`)
- Automatic browser management
- Error handling and logging
- Health check endpoint

## Requirements

- Node.js 18+
- npm or yarn
- Chromium (installed by Puppeteer)

## Installation

```bash
# Navigate to service directory
cd services/screenshot-service

# Install dependencies
npm install
```

## Usage

### Start the Server

```bash
npm start
```

### Development Mode (auto-restart)

```bash
npm run dev
```

## API Endpoints

### POST /capture

Capture a single TradingView chart screenshot.

**Request:**
```json
{
  "symbol": "BTCUSDT",
  "interval": "1m"
}
```

**Parameters:**
- `symbol` (required): Trading pair (e.g., "BTCUSDT", "1000PEPEUSDT")
- `interval` (optional): Chart timeframe. Default: "1m"

**Valid Intervals:**
`1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `1d`, `1w`, `1M`

**Response:**
```json
{
  "symbol": "BTCUSDT",
  "interval": "1m",
  "timeframe": "1m",
  "screenshot": "base64_encoded_image...",
  "timestamp": "2026-01-14T10:30:00.000Z",
  "duration_ms": 5420
}
```

---

### POST /capture-multi

Capture multiple timeframe screenshots in one request.

**Request:**
```json
{
  "symbol": "BTCUSDT",
  "intervals": ["1m", "5m", "15m"]
}
```

**Response:**
```json
{
  "symbol": "BTCUSDT",
  "intervals": ["1m", "5m", "15m"],
  "results": {
    "1m": "base64_image_1...",
    "5m": "base64_image_2...",
    "15m": "base64_image_3..."
  },
  "timestamp": "2026-01-14T10:30:00.000Z",
  "duration_ms": 15200
}
```

---

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-14T10:30:00.000Z",
  "browser": "ready"
}
```

---

### POST /browser/restart

Restart the browser instance (useful if hung).

## Testing

### Single Screenshot

```bash
curl -X POST http://localhost:3456/capture \
  -H "Content-Type: application/json" \
  -d '{"symbol":"BTCUSDT","interval":"1m"}'
```

### Multiple Timeframes

```bash
curl -X POST http://localhost:3456/capture-multi \
  -H "Content-Type: application/json" \
  -d '{"symbol":"1000PEPEUSDT","intervals":["1m","5m","15m"]}'
```

### Health Check

```bash
curl http://localhost:3456/health
```

### Save Screenshot to File

```bash
curl -s -X POST http://localhost:3456/capture \
  -H "Content-Type: application/json" \
  -d '{"symbol":"BTCUSDT","interval":"1m"}' | \
  jq -r '.screenshot' | \
  base64 -d > screenshot.png
```

## Integration with N8N

Use the **HTTP Request** node in N8N:

1. Method: `POST`
2. URL: `http://localhost:3456/capture`
3. Body (JSON):
   ```json
   {
     "symbol": "{{ $json.symbol }}",
     "interval": "1m"
   }
   ```
4. Extract image from `{{ $json.screenshot }}`

## Troubleshooting

### "Module not found: puppeteer"

```bash
npm install
```

### Browser won't launch

The browser needs ~500MB disk space. Ensure you have enough disk space.

### Screenshot is blank

- TradingView might be blocked (ad blocker, etc.)
- Check browser logs
- Try increasing wait time in the code

### Port already in use

Change the port in `server.js`:
```javascript
const PORT = 3457;  // Change this
```

## Project Structure

```
screenshot-service/
├── package.json
├── server.js
└── README.md
```

## License

MIT
