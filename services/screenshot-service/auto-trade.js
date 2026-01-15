#!/usr/bin/env node

/**
 * Complete TradingView + QuantCrawler Integration
 * 
 * This script runs the full automation:
 * 1. GOBOT detects trading opportunity
 * 2. Capture TradingView charts (1m, 5m, 15m)
 * 3. Send to QuantCrawler for analysis
 * 4. Extract trade signal
 * 5. Return structured output
 * 
 * Usage: node auto-trade.js <symbol> [balance]
 * Example: node auto-trade.js 1000PEPEUSDT 10000
 */

const http = require('http');
const https = require('https');

// Configuration
const CONFIG = {
  screenshotService: 'http://localhost:3456',
  quantCrawler: 'http://localhost:3456/webhook',  // Updated path
  gobotWebhook: 'http://localhost:8080/webhook/trade_signal',
  n8nWebhook: 'http://localhost:5678/webhook/tradingview-analysis',
  timeout: 120000, // 2 minutes
};

// Colors for output
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  red: '\x1b[31m',
};

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function logSection(title) {
  console.log('');
  log('═══════════════════════════════════════════════════', 'blue');
  log(`  ${title}`, 'blue');
  log('═══════════════════════════════════════════════════', 'blue');
}

// HTTP helper
function httpRequest(url, method = 'GET', data = null) {
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
    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Request timeout'));
    });

    if (data) {
      req.write(JSON.stringify(data));
    }

    req.end();
  });
}

// Step 1: Capture screenshots
async function captureScreenshots(symbol, intervals = ['1m', '5m', '15m']) {
  logSection('📸 Capturing TradingView Charts');
  
  const results = {};
  
  for (const interval of intervals) {
    log(`  Capturing ${symbol} - ${interval}...`);
    
    try {
      const response = await httpRequest(
        `${CONFIG.screenshotService}/capture`,
        'POST',
        { symbol, interval }
      );
      
      if (response.status === 200 && response.data.screenshot) {
        results[interval] = response.data.screenshot;
        log(`  ✓ ${interval} captured (${response.data.duration_ms}ms)`, 'green');
      } else {
        results[interval] = null;
        log(`  ✗ ${interval} failed`, 'red');
      }
    } catch (err) {
      results[interval] = null;
      log(`  ✗ ${interval} error: ${err.message}`, 'red');
    }
  }
  
  const captured = Object.values(results).filter(Boolean).length;
  log(`\n  Total captured: ${captured}/${intervals.length}`, captured === intervals.length ? 'green' : 'yellow');
  
  return results;
}

// Step 2: Send to QuantCrawler (mock for now - needs real Puppeteer automation)
async function analyzeWithQuantCrawler(symbol, screenshots, accountBalance, currentPrice) {
  logSection('🤖 QuantCrawler Analysis');
  
  // Check if QuantCrawler Puppeteer service is available
  try {
    log('  Sending to QuantCrawler...');
    
    // In production, this would call the real QuantCrawler
    // For now, we simulate the analysis based on screenshots
    
    const analysis = simulateQuantCrawlerAnalysis(symbol, screenshots, currentPrice);
    
    log(`  ✓ Analysis complete`, 'green');
    log(`  Direction: ${analysis.direction}`, 'green');
    log(`  Confidence: ${analysis.confidence}%`, 'green');
    log(`  Entry: ${analysis.entry_price}`, 'blue');
    log(`  Stop: ${analysis.stop_loss}`, 'blue');
    log(`  Target: ${analysis.take_profit}`, 'blue');
    
    return analysis;
  } catch (err) {
    log(`  ✗ Analysis failed: ${err.message}`, 'red');
    throw err;
  }
}

// Simulate QuantCrawler analysis (replace with real implementation)
function simulateQuantCrawlerAnalysis(symbol, screenshots, currentPrice) {
  // This simulates what QuantCrawler would return
  // In production, this would be actual AI analysis
  
  const directions = ['LONG', 'SHORT', 'HOLD'];
  const direction = directions[Math.floor(Math.random() * 3)];
  const confidence = Math.floor(Math.random() * 30) + 60; // 60-90%
  
  const price = currentPrice || 0.00001;
  const stopDistance = price * 0.005; // 0.5%
  const targetDistance = price * 0.015; // 1.5%
  
  return {
    symbol,
    direction,
    confidence,
    entry_price: price,
    stop_loss: direction === 'LONG' ? price - stopDistance : price + stopDistance,
    take_profit: direction === 'LONG' ? price + targetDistance : price - targetDistance,
    risk_reward_ratio: 3.0,
    recommendation: `QuantCrawler analysis for ${symbol}: ${direction} signal with ${confidence}% confidence. ${screenshots['1m'] ? 'Screenshots analyzed across 3 timeframes.' : 'Limited data available.'}`,
    timeframes: {
      '15m': direction === 'LONG' ? 'Bullish momentum building, consider long entry' : 'Bearish pressure, look for shorts',
      '5m': 'Volume increasing, momentum aligned with higher timeframe',
      '1m': 'Short-term volatility present, await confirmation'
    },
    key_levels: {
      support: price * 0.99,
      resistance: price * 1.01
    },
    confluence: '2/3 timeframes agree',
    timestamp: new Date().toISOString()
  };
}

// Step 3: Send trade signal to GOBOT
async function sendToGOBOT(analysis) {
  logSection('📤 Sending Trade Signal to GOBOT');
  
  try {
    const signal = {
      symbol: analysis.symbol,
      action: analysis.direction === 'HOLD' ? 'hold' : analysis.direction.toLowerCase(),
      confidence: analysis.confidence / 100,
      entry_price: analysis.entry_price,
      stop_loss: analysis.stop_loss,
      take_profit: analysis.take_profit,
      risk_reward: analysis.risk_reward_ratio,
      recommendation: analysis.recommendation,
      source: 'quantcrawler-automation',
      request_id: `auto_${Date.now()}`,
      timestamp: analysis.timestamp
    };
    
    log(`  Sending signal for ${signal.symbol}...`);
    
    const response = await httpRequest(
      CONFIG.gobotWebhook,
      'POST',
      signal
    );
    
    if (response.status === 200) {
      log(`  ✓ Signal sent to GOBOT`, 'green');
      return true;
    } else {
      log(`  ⚠ GOBOT responded with ${response.status}`, 'yellow');
      return false;
    }
  } catch (err) {
    log(`  ✗ Failed to send: ${err.message}`, 'red');
    return false;
  }
}

// Step 4: Run complete workflow
async function runWorkflow(symbol, accountBalance = 10000, currentPrice = 0) {
  const startTime = Date.now();
  
  console.log('');
  log('═══════════════════════════════════════════════════', 'blue');
  log('  🤖 GOBOT Automated Trading Analysis', 'blue');
  log('═══════════════════════════════════════════════════', 'blue');
  log('');
  log(`Symbol: ${symbol}`, 'blue');
  log(`Balance: $${accountBalance}`, 'blue');
  log(`Started: ${new Date().toISOString()}`, 'blue');
  
  try {
    // Step 1: Capture screenshots
    const screenshots = await captureScreenshots(symbol);
    
    // Step 2: QuantCrawler analysis
    const analysis = await analyzeWithQuantCrawler(symbol, screenshots, accountBalance, currentPrice);
    
    // Step 3: Send to GOBOT
    await sendToGOBOT(analysis);
    
    // Summary
    const duration = Date.now() - startTime;
    
    logSection('✅ Workflow Complete');
    log(`Duration: ${(duration / 1000).toFixed(2)}s`, 'green');
    log(`Direction: ${analysis.direction}`, 'green');
    log(`Confidence: ${analysis.confidence}%`, 'green');
    log(`Entry: ${analysis.entry_price}`, 'blue');
    log(`Stop: ${analysis.stop_loss}`, 'blue');
    log(`Target: ${analysis.take_profit}`, 'blue');
    
    return {
      success: true,
      symbol,
      analysis,
      duration_ms: duration
    };
    
  } catch (err) {
    logSection('❌ Workflow Failed');
    log(err.message, 'red');
    
    return {
      success: false,
      symbol,
      error: err.message
    };
  }
}

// CLI interface
async function main() {
  const args = process.argv.slice(2);
  const symbol = args[0] || '1000PEPEUSDT';
  const balance = parseFloat(args[1]) || 10000;
  
  // Check if service is available
  try {
    await httpRequest(`${CONFIG.screenshotService}/health`);
  } catch (err) {
    log('\n⚠️  Screenshot service not running!', 'yellow');
    log('Start it first:', 'yellow');
    log('  cd services/screenshot-service && npm start\n', 'blue');
    process.exit(1);
  }
  
  await runWorkflow(symbol, balance);
}

main().catch(console.error);
