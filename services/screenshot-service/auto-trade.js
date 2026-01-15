#!/usr/bin/env node

/**
 * GOBOT Automated Trading - Agent-Browser + AI Integration
 * 
 * Full automation pipeline:
 * 1. agent-browser captures TradingView charts (1m, 5m, 15m)
 * 2. GPT-4o Vision for AI analysis (or fallback)
 * 3. Extract structured trade signal
 * 4. Send to GOBOT webhook
 * 
 * Usage: node auto-trade.js <symbol> [balance]
 * Example: node auto-trade.js 1000PEPEUSDT 10000
 */

const http = require('http');
const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const envPath = path.join(__dirname, '..', '..', '.env');
if (fs.existsSync(envPath)) {
  const envContent = fs.readFileSync(envPath, 'utf8');
  envContent.split('\n').forEach(line => {
    const trimmed = line.trim();
    if (trimmed && !trimmed.startsWith('#') && trimmed.includes('=')) {
      const [key, ...vals] = trimmed.split('=');
      if (key && vals.join('=').trim()) {
        process.env[key.trim()] = vals.join('=').trim();
      }
    }
  });
}

const aiAnalyzer = require('./ai-analyzer.js');

const CONFIG = {
  useTestnet: process.env.BINANCE_USE_TESTNET === 'false' ? false : true,
  screenshotService: 'http://localhost:3456',
  gobotWebhook: 'http://localhost:8080/webhook/trade_signal',
  
  telegramToken: process.env.TELEGRAM_TOKEN || '',
  telegramChatId: process.env.AUTHORIZED_CHAT_ID || process.env.TELEGRAM_CHAT_ID || '',
  telegramEnabled: (process.env.TELEGRAM_NOTIFICATIONS === 'true' || process.env.TELEGRAM_TOKEN) && process.env.TELEGRAM_TOKEN,
  
  getBinanceBaseURL() {
    return this.useTestnet 
      ? 'https://testnet.binancefuture.com' 
      : 'https://fapi.binance.com';
  },
  
  timeout: 120000,
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

function httpRequest(url, method = 'GET', data = null) {
  return new Promise((resolve, reject) => {
    const parsed = new URL(url);
    const client = parsed.protocol === 'https:' ? https : http;
    
    const options = {
      hostname: parsed.hostname,
      port: parsed.port || (parsed.protocol === 'https:' ? 443 : 80),
      path: parsed.pathname + parsed.search,
      method,
      headers: { 'Content-Type': 'application/json' },
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

async function sendTelegramNotification(analysis, marketData, mode) {
  if (!CONFIG.telegramEnabled || !CONFIG.telegramToken || !CONFIG.telegramChatId) {
    log('Telegram notifications disabled or not configured', 'yellow');
    return false;
  }

  const emoji = analysis.action === 'LONG' ? '🟢' : analysis.action === 'SHORT' ? '🔴' : '⚪';
  const modeEmoji = mode === 'MAINNET' ? '💰' : '🧪';

  const message = `
${emoji} *GOBOT TRADING SIGNAL*
${modeEmoji} *${mode}*

📊 *Symbol:* \`${analysis.symbol}\`
🎯 *Action:* \`${analysis.action}\`
📈 *Confidence:* \`${(analysis.confidence * 100).toFixed(0)}%\`
💰 *Price:* \`$${marketData.price?.toLocaleString() || 'N/A'}\`
📉 *24h Change:* \`${marketData.change24h?.toFixed(2) || 0}%\`

💡 *Reasoning:*
${analysis.reasoning.substring(0, 100)}...

🔒 *Risk:* 2% SL | 4% TP (2:1 RR)
⏰ *Time:* ${new Date().toISOString()}
`.trim();

  const url = `https://api.telegram.org/bot${CONFIG.telegramToken}/sendMessage`;
  const data = {
    chat_id: CONFIG.telegramChatId,
    text: message,
    parse_mode: 'Markdown',
    disable_web_page_preview: true,
  };

  return new Promise((resolve) => {
    const parsed = new URL(url);
    const client = parsed.protocol === 'https:' ? https : http;

    const options = {
      hostname: parsed.hostname,
      port: 443,
      path: '/bot' + CONFIG.telegramToken + '/sendMessage',
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      timeout: 10000,
    };

    const req = client.request(options, (res) => {
      let body = '';
      res.on('data', chunk => body += chunk);
      res.on('end', () => {
        try {
          const response = JSON.parse(body);
          if (response.ok) {
            log('✅ Telegram notification sent', 'green');
            resolve(true);
          } else {
            log(`❌ Telegram failed: ${response.description}`, 'red');
            resolve(false);
          }
        } catch (e) {
          log(`❌ Telegram error: ${e.message}`, 'red');
          resolve(false);
        }
      });
    });

    req.on('error', (e) => {
      log(`❌ Telegram network error: ${e.message}`, 'red');
      resolve(false);
    });
    req.write(JSON.stringify(data));
    req.end();
  });
}

async function captureWithAgentBrowser(symbol, intervals = ['1m', '5m', '15m']) {
  logSection('AGENT-BROWSER CHART CAPTURE');
  
  const screenshotDir = path.join(__dirname, 'screenshots');
  if (!fs.existsSync(screenshotDir)) {
    fs.mkdirSync(screenshotDir, { recursive: true });
  }
  
  const capturedPaths = [];
  
  for (const interval of intervals) {
    const url = `https://www.tradingview.com/chart/?symbol=BINANCE:${symbol}&interval=${interval}`;
    const filename = `ab_${symbol}_${interval}_${Date.now()}.png`;
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

async function fetchMarketData(symbol) {
  const url = `${CONFIG.getBinanceBaseURL()}/fapi/v1/ticker/24hr?symbol=${symbol.toUpperCase()}`;
  
  try {
    const response = await httpRequest(url);
    return {
      price: parseFloat(response.data.lastPrice),
      change24h: parseFloat(response.data.priceChangePercent),
    };
  } catch (e) {
    log(`Market data unavailable: ${e.message}`, 'yellow');
    return { price: 0, change24h: 0 };
  }
}

async function sendToGobot(signal, marketData) {
  logSection('SENDING TRADE SIGNAL TO GOBOT');
  
  const payload = {
    symbol: signal.symbol,
    action: signal.action,
    confidence: signal.confidence,
    entry_price: signal.entry_price || marketData.price?.toString() || '0',
    stop_loss: signal.stop_loss || '0',
    take_profit: signal.take_profit || '0',
    risk_reward: signal.risk_reward || 2,
    reasoning: signal.reasoning,
    analysis_id: signal.analysis_id,
    timestamp: signal.timestamp,
    source: signal.source || 'ai-analyzer',
  };
  
  try {
    const response = await httpRequest(CONFIG.gobotWebhook, 'POST', payload);
    
    if (response.status === 200) {
      log(`✅ Signal sent to GOBOT`, 'green');
      return true;
    } else {
      log(`❌ Failed: ${response.status}`, 'red');
      return false;
    }
  } catch (e) {
    log(`❌ Webhook error: ${e.message}`, 'red');
    return false;
  }
}

async function runTradingCycle(symbol, balance) {
  logSection('GOBOT AUTOMATED TRADING ANALYSIS');
  log(`Symbol: ${symbol}`);
  log(`Balance: $${balance}`);
  log(`Mode: ${CONFIG.useTestnet ? 'TESTNET 🧪' : 'MAINNET 💰'}`);
  
  const chartPaths = await captureWithAgentBrowser(symbol);
  
  if (chartPaths.length === 0) {
    log('No charts captured, aborting', 'red');
    return { success: false, reason: 'No charts captured' };
  }
  
  log(`Captured ${chartPaths.length}/3 charts`, chartPaths.length === 3 ? 'green' : 'yellow');
  
  const marketData = await fetchMarketData(symbol);
  if (marketData.price > 0) {
    log(`Price: $${marketData.price.toLocaleString()}`, 'blue');
    log(`24h Change: ${marketData.change24h.toFixed(2)}%`, marketData.change24h >= 0 ? 'green' : 'red');
  }
  
  logSection('AI ANALYSIS');
  const analysis = await aiAnalyzer.analyzeSymbol(symbol, balance);
  
  log('');
  log('📊 AI ANALYSIS RESULTS:', 'cyan');
  log(`  Direction:    ${analysis.action}`, analysis.action === 'LONG' ? 'green' : analysis.action === 'SHORT' ? 'red' : 'yellow');
  log(`  Confidence:   ${(analysis.confidence * 100).toFixed(0)}%`, 'green');
  if (analysis.entry_price !== '0.00000000') {
    log(`  Entry Price:  ${analysis.entry_price}`, 'blue');
    log(`  Stop Loss:    ${analysis.stop_loss}`, 'blue');
    log(`  Take Profit:  ${analysis.take_profit}`, 'blue');
  }
  log(`  Reasoning:    ${analysis.reasoning.substring(0, 60)}...`, 'yellow');
  
  const mode = CONFIG.useTestnet ? 'TESTNET' : 'MAINNET';
  
  logSection('TELEGRAM ALERTS');
  await sendTelegramNotification(analysis, marketData, mode);
  
  await sendToGobot(analysis, marketData);
  
  return { success: true, analysis, marketData };
}

async function main() {
  const args = process.argv.slice(2);
  
  if (args.length < 1) {
    console.log(`
GOBOT Automated Trading - Agent-Browser + AI
=============================================

Usage: node auto-trade.js <symbol> [balance]

Example:
  node auto-trade.js BTCUSDT 1000
  node auto-trade.js 1000PEPEUSDT 500

Environment Variables:
  OPENAI_API_KEY       - For GPT-4o Vision analysis
  GOOGLE_EMAIL         - For authenticated TradingView
  GOOGLE_APP_PASSWORD  - App password for Google Auth

Features:
  - agent-browser for TradingView chart capture
  - GPT-4o Vision AI analysis (when API key set)
  - Structured trading signals
  - GOBOT webhook integration
`);
    process.exit(1);
  }
  
  const symbol = args[0];
  const balance = parseFloat(args[1]) || 100;
  
  try {
    const result = await runTradingCycle(symbol, balance);
    
    console.log('');
    if (result.success) {
      log('✅ WORKFLOW COMPLETE', 'green');
    } else {
      log('⚠️ WORKFLOW INCOMPLETE', 'yellow');
    }
  } catch (error) {
    log(`Failed: ${error.message}`, 'red');
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { runTradingCycle, captureWithAgentBrowser, sendToGobot };
