const puppeteer = require('puppeteer');
const fs = require('fs');
const path = require('path');

const PORT = 3456;
const SESSION_DIR = path.join(__dirname, '..', 'tradingview-sessions');

// Ensure session directory exists
if (!fs.existsSync(SESSION_DIR)) {
  fs.mkdirSync(SESSION_DIR, { recursive: true });
}

const CONFIG = {
  tradingview: {
    baseUrl: 'https://www.tradingview.com',
    chartUrl: 'https://www.tradingview.com/chart/',
    sessionDir: SESSION_DIR,
    timeout: 60000,
  },
  selectors: {
    chartContainer: '.chart-container, .tv-chart-container, [class*="chart-root"]',
    candle: '.bar, .tv-candle, [class*="candle"]',
    searchInput: '[placeholder*="Search" i], [placeholder*="Symbol" i], input[data-name="search-symbols-input"]',
    submitButton: 'button:has-text("Submit"), button:has-text("Analyze"), [class*="submit"]',
  }
};

// ═══════════════════════════════════════════════════════════════════════════
// MAIN SERVER - HTTP ENDPOINT
// ═══════════════════════════════════════════════════════════════════════════

const http = require('http');

const server = http.createServer(async (req, res) => {
  const startTime = Date.now();
  console.log(`\n[${new Date().toISOString()}] ${req.method} ${req.url}`);

  // CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

  if (req.method === 'OPTIONS') {
    return res.writeHead(204), res.end();
  }

  const parsedUrl = new URL(req.url, `http://localhost:${PORT}`);
  const pathname = parsedUrl.pathname;

  try {
    // Routes
    if (pathname === '/health' && req.method === 'GET') {
      return sendJSON(res, 200, { status: 'healthy', service: 'tradingview-screenshots' });
    }

    if (pathname === '/capture' && req.method === 'POST') {
      const body = await parseBody(req);
      const { symbol, interval = '1m' } = body;
      if (!symbol) return sendJSON(res, 400, { error: 'Missing symbol' });

      console.log(`📸 Capturing ${symbol} (${interval})`);
      const screenshot = await captureTradingView(symbol, interval);
      
      sendJSON(res, 200, {
        symbol,
        interval,
        timeframe: interval,
        screenshot,
        timestamp: new Date().toISOString(),
        duration_ms: Date.now() - startTime
      });
    }

    if (pathname === '/capture-multi' && req.method === 'POST') {
      const body = await parseBody(req);
      const { symbol, intervals = ['1m', '5m', '15m'] } = body;
      if (!symbol) return sendJSON(res, 400, { error: 'Missing symbol' });

      console.log(`📸 Capturing ${symbol} at ${intervals.join(', ')}`);
      const results = {};
      
      for (const interval of intervals) {
        try {
          results[interval] = await captureTradingView(symbol, interval);
        } catch (err) {
          console.log(`  Failed ${interval}: ${err.message}`);
          results[interval] = null;
        }
      }

      sendJSON(res, 200, {
        symbol,
        intervals,
        results,
        timestamp: new Date().toISOString(),
        duration_ms: Date.now() - startTime
      });
    }

    if (pathname === '/capture-all' && req.method === 'POST') {
      const body = await parseBody(req);
      const { symbol } = body;
      if (!symbol) return sendJSON(res, 400, { error: 'Missing symbol' });

      console.log(`📸 Capturing ${symbol} at ALL timeframes`);
      
      const results = {};
      const allIntervals = ['1m', '5m', '15m', '1h', '4h'];
      
      for (const interval of allIntervals) {
        try {
          results[interval] = await captureTradingView(symbol, interval);
        } catch (err) {
          results[interval] = null;
        }
      }

      sendJSON(res, 200, {
        symbol,
        intervals: allIntervals,
        results,
        timestamp: new Date().toISOString(),
        duration_ms: Date.now() - startTime
      });
    }

    sendJSON(res, 404, { error: 'Not found', endpoints: ['/capture', '/capture-multi', '/capture-all', '/health'] });

  } catch (error) {
    console.error(`Error: ${error.message}`);
    sendJSON(res, 500, { error: error.message });
  }
});

// ═══════════════════════════════════════════════════════════════════════════
// PUPPETEER CHART CAPTURE
// ═══════════════════════════════════════════════════════════════════════════

let browser = null;

async function getBrowser() {
  if (browser) return browser;
  
  console.log('🌐 Launching browser...');
  browser = await puppeteer.launch({
    headless: 'new',
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-dev-shm-usage',
      '--disable-gpu',
      '--window-size=1920,1080',
    ],
  });

  browser.on('disconnected', () => {
    console.log('Browser disconnected');
    browser = null;
  });

  console.log('✅ Browser ready');
  return browser;
}

async function captureTradingView(symbol, interval) {
  const br = await getBrowser();
  const page = await br.newPage();
  
  // Set viewport
  await page.setViewport({ width: 1920, height: 1080 });

  // Map interval to TradingView format
  const intervalMap = {
    '1m': '1', '3m': '3', '5m': '5', '15m': '15', '30m': '30',
    '1h': '60', '2h': '120', '4h': '240', '1d': 'D', '1w': 'W', '1M': 'M'
  };
  const tvInterval = intervalMap[interval] || interval;
  const tvSymbol = symbol.replace('PERP', '').replace('USDT', '').toUpperCase();

  // Build TradingView URL
  const chartUrl = `https://www.tradingview.com/chart/?symbol=BINANCE:${tvSymbol}&interval=${tvInterval}`;
  
  console.log(`  Loading: ${chartUrl}`);
  
  await page.goto(chartUrl, { waitUntil: 'networkidle2', timeout: 30000 });
  
  // Wait for chart to load
  try {
    await page.waitForSelector('body', { timeout: 10000 });
  } catch (e) {
    console.log('  Warning: Page load timeout');
  }
  
  // Additional wait for chart rendering
  await new Promise(r => setTimeout(r, 3000));
  
  // Capture screenshot of the chart area
  const screenshot = await page.screenshot({
    type: 'png',
    fullPage: false,
    clip: { x: 0, y: 60, width: 1920, height: 800 }
  });
  
  await page.close();
  
  // Return base64
  return screenshot.toString('base64');
}

// ═══════════════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════════════

function parseBody(req) {
  return new Promise((resolve, reject) => {
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', () => {
      try {
        resolve(body ? JSON.parse(body) : {});
      } catch (e) {
        resolve({});
      }
    });
    req.on('error', reject);
  });
}

function sendJSON(res, statusCode, data) {
  res.writeHead(statusCode, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(data));
}

// ═══════════════════════════════════════════════════════════════════════════
// START SERVER
// ═══════════════════════════════════════════════════════════════════════════

server.listen(PORT, async () => {
  console.log('');
  console.log('==============================================');
  console.log('  TradingView Screenshot Service');
  console.log('==============================================');
  console.log(`  Server: http://localhost:${PORT}`);
  console.log('');
  console.log('Endpoints:');
  console.log(`  POST /capture        - Single timeframe`);
  console.log(`  POST /capture-multi  - Multiple timeframes`);
  console.log(`  POST /capture-all    - All timeframes (1m,5m,15m,1h,4h)`);
  console.log(`  GET  /health         - Health check`);
  console.log('');
  console.log('Examples:');
  console.log(`  curl -X POST http://localhost:${PORT}/capture \\`);
  console.log(`    -H "Content-Type: application/json" \\`);
  console.log(`    -d '{"symbol":"BTCUSDT","interval":"1m"}'`);
  console.log('');
  console.log(`  curl -X POST http://localhost:${PORT}/capture-multi \\`);
  console.log(`    -H "Content-Type: application/json" \\`);
  console.log(`    -d '{"symbol":"1000PEPEUSDT","intervals":["1m","5m","15m"]}'`);
  console.log('==============================================');
  
  // Pre-launch browser for faster first request
  getBrowser().catch(console.error);
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('\nShutting down...');
  if (browser) await browser.close();
  process.exit(0);
});

process.on('SIGINT', async () => {
  console.log('\nShutting down...');
  if (browser) await browser.close();
  process.exit(0);
});
