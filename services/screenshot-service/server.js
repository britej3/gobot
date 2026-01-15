const express = require('express');
const puppeteer = require('puppeteer');

const app = express();
const PORT = 3456;

// Middleware
app.use(express.json());

// Request logging
app.use((req, res, next) => {
  const timestamp = new Date().toISOString();
  console.log(`[${timestamp}] ${req.method} ${req.path}`);
  next();
});

// Browser instance
let browser = null;

// Initialize browser
async function initBrowser() {
  if (browser) return browser;
  
  console.log('[Screenshot] Launching browser...');
  
  browser = await puppeteer.launch({
    headless: 'new',
    executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-dev-shm-usage',
      '--disable-gpu',
      '--window-size=1920,1080',
    ],
  });
  
  browser.on('disconnected', () => {
    console.log('[Screenshot] Browser disconnected');
    browser = null;
  });
  
  console.log('[Screenshot] Browser ready');
  return browser;
}

// Clean browser on exit
async function cleanup() {
  if (browser) {
    await browser.close();
    browser = null;
    console.log('[Screenshot] Browser closed');
  }
}

process.on('SIGTERM', cleanup);
process.on('SIGINT', cleanup);

// Health check
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'tradingview-screenshots',
    timestamp: new Date().toISOString(),
  });
});

// Capture screenshot
app.post('/capture', async (req, res) => {
  const startTime = Date.now();
  
  try {
    const { symbol, interval = '1m' } = req.body;
    
    // Validate input
    if (!symbol) {
      return res.status(400).json({
        error: 'Missing required field: symbol',
        example: { symbol: 'BTCUSDT', interval: '1m' },
      });
    }
    
    const validIntervals = ['1m', '3m', '5m', '15m', '30m', '1h', '2h', '4h', '1d', '1w', '1M'];
    if (!validIntervals.includes(interval)) {
      return res.status(400).json({
        error: 'Invalid interval',
        valid: validIntervals,
      });
    }
    
    console.log(`[Screenshot] Capturing ${symbol} ${interval}...`);
    
    // Get browser
    const br = await initBrowser();
    const page = await br.newPage();
    
    // Set viewport
    await page.setViewport({ width: 1920, height: 1080 });
    
    // Map interval to TradingView format
    const intervalMap = {
      '1m': '1', '3m': '3', '5m': '5', '15m': '15', '30m': '30',
      '1h': '60', '2h': '120', '4h': '240', '1d': 'D', '1w': 'W', '1M': 'M'
    };
    const tvInterval = intervalMap[interval];
    const tvSymbol = symbol.replace('PERP', '').replace('USDT', '').toUpperCase();
    
    // Build TradingView URL
    const url = `https://www.tradingview.com/chart/?symbol=BINANCE:${tvSymbol}&interval=${tvInterval}`;
    
    console.log(`  Loading: ${url}`);
    
    // Navigate to TradingView
    await page.goto(url, { waitUntil: 'networkidle2', timeout: 30000 });
    
    // Wait for chart to load
    await page.waitForSelector('body', { timeout: 10000 });
    
    // Wait for chart to render
    await new Promise(r => setTimeout(r, 3000));
    
    // Take screenshot of the chart area
    const screenshotBuffer = await page.screenshot({
      type: 'png',
      fullPage: false,
      clip: { x: 0, y: 60, width: 1920, height: 800 },
    });
    
    // Close page
    await page.close();
    
    // Convert to base64
    const base64 = screenshotBuffer.toString('base64');
    
    const duration = Date.now() - startTime;
    
    console.log(`[Screenshot] Captured ${symbol} ${interval} in ${duration}ms`);
    
    // Return response
    res.json({
      symbol,
      interval,
      timeframe: interval,
      screenshot: base64,
      timestamp: new Date().toISOString(),
      duration_ms: duration,
    });
    
  } catch (error) {
    console.error(`[Screenshot] Error: ${error.message}`);
    
    res.status(500).json({
      error: 'Failed to capture screenshot',
      message: error.message,
    });
  }
});

// Capture multiple timeframes
app.post('/capture-multi', async (req, res) => {
  const startTime = Date.now();
  
  try {
    const { symbol, intervals = ['1m', '5m', '15m'] } = req.body;
    
    if (!symbol) {
      return res.status(400).json({
        error: 'Missing required field: symbol',
      });
    }
    
    console.log(`[Screenshot] Capturing ${symbol} at ${intervals.join(', ')}...`);
    
    const results = {};
    
    for (const interval of intervals) {
      try {
        const br = await initBrowser();
        const page = await br.newPage();
        await page.setViewport({ width: 1920, height: 1080 });
        
        const intervalMap = {
          '1m': '1', '3m': '3', '5m': '5', '15m': '15', '30m': '30',
          '1h': '60', '2h': '120', '4h': '240', '1d': 'D', '1w': 'W', '1M': 'M'
        };
        const tvInterval = intervalMap[interval];
        const tvSymbol = symbol.replace('PERP', '').replace('USDT', '').toUpperCase();
        
        const url = `https://www.tradingview.com/chart/?symbol=BINANCE:${tvSymbol}&interval=${tvInterval}`;
        await page.goto(url, { waitUntil: 'networkidle2', timeout: 30000 });
        await page.waitForSelector('body', { timeout: 10000 });
        await new Promise(r => setTimeout(r, 3000));
        
        const buffer = await page.screenshot({
          type: 'png',
          fullPage: false,
          clip: { x: 0, y: 60, width: 1920, height: 800 },
        });
        
        await page.close();
        results[interval] = buffer.toString('base64');
      } catch (err) {
        console.error(`  Failed ${interval}: ${err.message}`);
        results[interval] = null;
      }
    }
    
    const duration = Date.now() - startTime;
    
    console.log(`[Screenshot] Multi-capture completed in ${duration}ms`);
    
    res.json({
      symbol,
      intervals,
      results,
      timestamp: new Date().toISOString(),
      duration_ms: duration,
    });
    
  } catch (error) {
    console.error(`[Screenshot] Error: ${error.message}`);
    
    res.status(500).json({
      error: 'Failed to capture screenshots',
      message: error.message,
    });
  }
});

// Browser status
app.get('/browser/status', async (req, res) => {
  try {
    await initBrowser();
    res.json({
      status: 'ready',
      browser: browser ? 'running' : 'stopped',
    });
  } catch (error) {
    res.status(500).json({
      status: 'error',
      message: error.message,
    });
  }
});

// Restart browser
app.post('/browser/restart', async (req, res) => {
  try {
    if (browser) {
      await browser.close();
      browser = null;
    }
    await initBrowser();
    res.json({ status: 'restarted' });
  } catch (error) {
    res.status(500).json({
      error: 'Failed to restart browser',
      message: error.message,
    });
  }
});

// Error handler
app.use((err, req, res, next) => {
  console.error(`[Error] ${err.message}`);
  res.status(500).json({
    error: 'Internal server error',
    message: err.message,
  });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({
    error: 'Not found',
    available: ['POST /capture', 'POST /capture-multi', 'GET /health', 'GET /browser/status'],
  });
});

// Start server
app.listen(PORT, async () => {
  console.log('==============================================');
  console.log('  TradingView Screenshot Service');
  console.log('==============================================');
  console.log(`  Server:      http://localhost:${PORT}`);
  console.log(`  Endpoints:`);
  console.log(`    POST /capture        - Capture single timeframe`);
  console.log(`    POST /capture-multi  - Capture multiple timeframes`);
  console.log(`    GET  /health         - Health check`);
  console.log(`    GET  /browser/status - Browser status`);
  console.log(`    POST /browser/restart - Restart browser`);
  console.log('');
  console.log(`  Example:`);
  console.log(`    curl -X POST http://localhost:${PORT}/capture \\`);
  console.log(`      -H "Content-Type: application/json" \\`);
  console.log(`      -d '{"symbol":"BTCUSDT","interval":"1m"}'`);
  console.log('==============================================');
  
  // Initialize browser
  await initBrowser();
});
