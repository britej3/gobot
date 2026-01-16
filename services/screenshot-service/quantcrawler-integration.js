#!/usr/bin/env node

/**
 * QuantCrawler Integration - Screenshot Upload & Report Retrieval
 * 
 * Flow:
 * 1. Login to TradingView with Google OAuth (britej3@gmail.com)
 * 2. Capture 3 screenshots (1m, 5m, 15m) using agent-browser
 * 3. Upload screenshots to QuantCrawler with ticker and position amount
 * 4. Get QuantCrawler report (entry, exit, SL, TP, confidence)
 * 5. Return structured trading signal
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const https = require('https');
const http = require('http');

const CONFIG = {
  quantCrawlerURL: 'https://quantcrawler.com/api/analyze',
  tradingViewURL: 'https://www.tradingview.com',
  screenshotDir: path.join(__dirname, 'screenshots'),
  
  // Google OAuth credentials
  googleEmail: process.env.QUANTCRAWLER_EMAIL || '',
  googlePassword: process.env.QUANTCRAWLER_PASSWORD || '',
  
  // Request timeout
  timeout: 120000, // 2 minutes
};

const C = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  red: '\x1b[31m',
  cyan: '\x1b[36m',
};

function log(msg, color = 'reset') {
  console.log(`${C[color]}${msg}${C.reset}`);
}

function logSection(title) {
  console.log('');
  log('═══════════════════════════════════════════════════════════════', 'blue');
  log(`  ${title}`, 'blue');
  log('═══════════════════════════════════════════════════════════════', 'blue');
}

function loadEnv() {
  const envPath = path.join(__dirname, '..', '..', '.env');
  if (fs.existsSync(envPath)) {
    const envContent = fs.readFileSync(envPath, 'utf8');
    envContent.split('\n').forEach(line => {
      const trimmed = line.trim();
      if (trimmed && !trimmed.startsWith('#') && trimmed.includes('=')) {
        const [key, ...vals] = trimmed.split('=');
        const value = vals.join('=').trim();
        if (key && value && !process.env[key.trim()]) {
          process.env[key.trim()] = value;
        }
      }
    });
  }
}

function httpRequest(url, method = 'GET', data = null, headers = {}) {
  return new Promise((resolve, reject) => {
    const parsed = new URL(url);
    const client = parsed.protocol === 'https:' ? https : http;
    
    const options = {
      hostname: parsed.hostname,
      port: parsed.port || (parsed.protocol === 'https:' ? 443 : 80),
      path: parsed.pathname + parsed.search,
      method,
      headers: {
        'Content-Type': 'application/json',
        ...headers,
      },
      timeout: CONFIG.timeout,
    };

    const req = client.request(options, (res) => {
      let body = '';
      res.on('data', chunk => body += chunk);
      res.on('end', () => {
        try {
          resolve({ status: res.statusCode, data: JSON.parse(body) });
        } catch (e) {
          resolve({ status: res.statusCode, data: body });
        }
      });
    });

    req.on('error', reject);
    req.on('timeout', () => { req.destroy(); reject(new Error('Request timeout')); });
    if (data) req.write(JSON.stringify(data));
    req.end();
  });
}

/**
 * Login to TradingView with Google OAuth
 */
async function loginToTradingView() {
  logSection('TRADINGVIEW LOGIN');
  
  if (!CONFIG.googleEmail || !CONFIG.googlePassword) {
    throw new Error('Google credentials not configured. Set QUANTCRAWLER_EMAIL and QUANTCRAWLER_PASSWORD in .env');
  }
  
  log(`Logging in as ${CONFIG.googleEmail}...`, 'blue');
  
  try {
    // Use agent-browser to login to TradingView with Google OAuth
    const loginURL = 'https://www.tradingview.com';
    
    execSync(`agent-browser open "${loginURL}" --headers '{"User-Agent":"Mozilla/5.0"}'`, {
      encoding: 'utf8',
      timeout: 15000,
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    
    await new Promise(r => setTimeout(r, 3000));
    
    // Execute Google OAuth login
    execSync(`agent-browser execute --script "document.querySelector('button[data-google]').click()"`, {
      encoding: 'utf8',
      timeout: 10000,
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    
    await new Promise(r => setTimeout(r, 2000));
    
    // Enter email
    execSync(`agent-browser type --selector 'input[type="email"]' --text "${CONFIG.googleEmail}"`, {
      encoding: 'utf8',
      timeout: 10000,
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    
    await new Promise(r => setTimeout(r, 1000));
    
    // Click next
    execSync(`agent-browser click --selector '#identifierNext'`, {
      encoding: 'utf8',
      timeout: 10000,
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    
    await new Promise(r => setTimeout(r, 2000));
    
    // Enter password (Google App Password)
    execSync(`agent-browser type --selector 'input[type="password"]' --text "${CONFIG.googlePassword}"`, {
      encoding: 'utf8',
      timeout: 10000,
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    
    await new Promise(r => setTimeout(r, 1000));
    
    // Click next/submit
    execSync(`agent-browser click --selector '#passwordNext'`, {
      encoding: 'utf8',
      timeout: 10000,
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    
    await new Promise(r => setTimeout(r, 5000));
    
    log('✅ Successfully logged in to TradingView', 'green');
    return true;
    
  } catch (e) {
    log(`❌ Login failed: ${e.message}`, 'red');
    throw e;
  }
}

/**
 * Capture 3 screenshots (1m, 5m, 15m) using agent-browser
 */
async function captureScreenshots(symbol) {
  logSection('CAPTURING SCREENSHOTS');
  
  const screenshotDir = CONFIG.screenshotDir;
  if (!fs.existsSync(screenshotDir)) {
    fs.mkdirSync(screenshotDir, { recursive: true });
  }
  
  const intervals = ['1m', '5m', '15m'];
  const capturedPaths = [];
  
  for (const interval of intervals) {
    const url = `https://www.tradingview.com/chart/?symbol=BINANCE:${symbol}&interval=${interval}`;
    const filename = `quantcrawler_${symbol}_${interval}_${Date.now()}.png`;
    const filepath = path.join(screenshotDir, filename);
    
    log(`Capturing ${symbol} - ${interval}...`, 'blue');
    
    try {
      execSync(`agent-browser open "${url}" --headers '{"User-Agent":"Mozilla/5.0"}'`, {
        encoding: 'utf8',
        timeout: 15000,
        stdio: ['pipe', 'pipe', 'pipe'],
      });
      
      await new Promise(r => setTimeout(r, 4000));
      
      execSync(`agent-browser screenshot "${filepath}" --full`, {
        encoding: 'utf8',
        timeout: 10000,
        stdio: ['pipe', 'pipe', 'pipe'],
      });
      
      if (fs.existsSync(filepath)) {
        log(`  ✅ ${filename}`, 'green');
        capturedPaths.push(filepath);
      }
    } catch (e) {
      log(`  ❌ Failed: ${e.message.substring(0, 50)}`, 'red');
    }
    
    await new Promise(r => setTimeout(r, 1000));
  }
  
  execSync('agent-browser close', { encoding: 'utf8', timeout: 5000 });
  
  return capturedPaths;
}

/**
 * Upload screenshots to QuantCrawler and get report
 */
async function uploadToQuantCrawler(symbol, positionAmount, screenshotPaths) {
  logSection('UPLOADING TO QUANTCRAWLER');
  
  log(`Symbol: ${symbol}`, 'blue');
  log(`Position Amount: $${positionAmount}`, 'blue');
  log(`Screenshots: ${screenshotPaths.length}`, 'blue');
  
  // Read screenshots as base64
  const screenshots = screenshotPaths.map(filepath => {
    const buffer = fs.readFileSync(filepath);
    return buffer.toString('base64');
  });
  
  const payload = {
    ticker: symbol,
    position_amount: positionAmount,
    screenshots: screenshots,
    intervals: ['1m', '5m', '15m'],
    timestamp: new Date().toISOString(),
  };
  
  try {
    log('Uploading to QuantCrawler...', 'blue');
    const response = await httpRequest(CONFIG.quantCrawlerURL, 'POST', payload);
    
    if (response.status === 200) {
      log('✅ Successfully uploaded to QuantCrawler', 'green');
      return response.data;
    } else {
      log(`❌ Upload failed: ${response.status}`, 'red');
      throw new Error(`QuantCrawler upload failed with status ${response.status}`);
    }
  } catch (e) {
    log(`❌ Upload error: ${e.message}`, 'red');
    throw e;
  }
}

/**
 * Parse QuantCrawler report into structured signal
 */
function parseQuantCrawlerReport(report) {
  logSection('PARSING QUANTCRAWLER REPORT');
  
  const signal = {
    symbol: report.ticker || '',
    direction: report.direction || 'HOLD',
    confidence: report.confidence || 50,
    entry: report.entry || report.current_price || 0,
    stop_loss: report.stop_loss || 0,
    take_profit: report.take_profit || 0,
    reasoning: report.recommendation || report.reasoning || '',
    timeframe_analysis: report.timeframes || {},
    key_levels: report.key_levels || {},
    timestamp: new Date().toISOString(),
    source: 'quantcrawler',
  };
  
  log(`Direction: ${signal.direction}`, signal.direction === 'LONG' ? 'green' : 'red');
  log(`Confidence: ${signal.confidence}%`, 'green');
  log(`Entry: ${signal.entry}`, 'blue');
  log(`Stop Loss: ${signal.stop_loss}`, 'blue');
  log(`Take Profit: ${signal.take_profit}`, 'blue');
  
  return signal;
}

/**
 * Main function - Complete QuantCrawler flow
 */
async function analyzeWithQuantCrawler(symbol, positionAmount = 100) {
  logSection('QUANTCRAWLER ANALYSIS');
  log(`Symbol: ${symbol}`);
  log(`Position Amount: $${positionAmount}`);
  
  loadEnv();
  
  try {
    // Step 1: Login to TradingView
    await loginToTradingView();
    
    // Step 2: Capture screenshots
    const screenshotPaths = await captureScreenshots(symbol);
    
    if (screenshotPaths.length === 0) {
      throw new Error('No screenshots captured');
    }
    
    log(`Captured ${screenshotPaths.length}/3 screenshots`, screenshotPaths.length === 3 ? 'green' : 'yellow');
    
    // Step 3: Upload to QuantCrawler
    const report = await uploadToQuantCrawler(symbol, positionAmount, screenshotPaths);
    
    // Step 4: Parse report
    const signal = parseQuantCrawlerReport(report);
    
    log('');
    log('═══════════════════════════════════════════════════════════════', 'green');
    log('  QUANTCRAWLER ANALYSIS COMPLETE', 'green');
    log('═══════════════════════════════════════════════════════════════', 'green');
    
    return signal;
    
  } catch (e) {
    log(`❌ Analysis failed: ${e.message}`, 'red');
    throw e;
  }
}

// Export for use in other modules
module.exports = {
  analyzeWithQuantCrawler,
  loginToTradingView,
  captureScreenshots,
  uploadToQuantCrawler,
  parseQuantCrawlerReport,
};

// If run directly
if (require.main === module) {
  const args = process.argv.slice(2);
  
  if (args.length < 1) {
    console.log(`
QuantCrawler Integration - Screenshot Upload & Report Retrieval
================================================================

Usage: node quantcrawler-integration.js <symbol> [position_amount]

Example:
  node quantcrawler-integration.js BTCUSDT 1000
  node quantcrawler-integration.js 1000PEPEUSDT 500

Environment Variables:
  QUANTCRAWLER_EMAIL      - Google email for TradingView login
  QUANTCRAWLER_PASSWORD   - Google App Password for OAuth

Features:
  - Login to TradingView with Google OAuth
  - Capture 3 screenshots (1m, 5m, 15m)
  - Upload to QuantCrawler
  - Get detailed trading report (entry, exit, SL, TP)
`);
    process.exit(1);
  }
  
  const symbol = args[0];
  const positionAmount = parseFloat(args[1]) || 100;
  
  analyzeWithQuantCrawler(symbol, positionAmount)
    .then(signal => {
      console.log('');
      console.log(JSON.stringify(signal, null, 2));
    })
    .catch(e => {
      console.error('Failed:', e.message);
      process.exit(1);
    });
}
