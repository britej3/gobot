#!/usr/bin/env node

/**
 * GOBOT Live Trading Dashboard
 * - Real-time TUI display
 * - Position monitoring
 * - Telegram live updates
 * - Auto-refresh every 5 seconds
 */

const fs = require('fs');
const path = require('path');
const http = require('http');
const https = require('https');
const crypto = require('crypto');

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

const CONFIG = {
  apiKey: process.env.BINANCE_API_KEY || '',
  secret: process.env.BINANCE_API_SECRET || '',
  useTestnet: process.env.BINANCE_USE_TESTNET === 'true',
  
  minBalance: 10,
  
  telegramToken: process.env.TELEGRAM_TOKEN || '',
  telegramChatId: process.env.AUTHORIZED_CHAT_ID || process.env.TELEGRAM_CHAT_ID || '',
  telegramEnabled: (process.env.TELEGRAM_NOTIFICATIONS === 'true' || process.env.TELEGRAM_TOKEN) && process.env.TELEGRAM_TOKEN && (process.env.AUTHORIZED_CHAT_ID || process.env.TELEGRAM_CHAT_ID),
  
  refreshInterval: 5000,
  
  getBaseURL() {
    return this.useTestnet 
      ? 'https://testnet.binancefuture.com' 
      : 'https://fapi.binance.com';
  },
};

const C = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  dim: '\x1b[2m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  red: '\x1b[31m',
  cyan: '\x1b[36m',
  magenta: '\x1b[35m',
  white: '\x1b[37m',
  bgRed: '\x1b[41m',
  bgGreen: '\x1b[42m',
};

const clear = '\x1Bc';
const cursorHide = '\x1B[?25l';
const cursorShow = '\x1B[?25h';

function log(msg, color = 'reset') {
  process.stdout.write(`${C[color]}${msg}${C.reset}`);
}

function signRequest(queryString) {
  return crypto.createHmac('sha256', CONFIG.secret)
    .update(queryString)
    .digest('hex');
}

async function httpRequest(url, method = 'GET', data = null, headers = {}) {
  return new Promise((resolve, reject) => {
    const parsed = new URL(url);
    const client = parsed.protocol === 'https:' ? https : http;
    
    const options = {
      hostname: parsed.hostname,
      port: parsed.port || 443,
      path: parsed.pathname + parsed.search,
      method,
      headers: { 'Content-Type': 'application/json', ...headers },
      timeout: 10000,
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
    req.on('timeout', () => { req.destroy(); reject(new Error('Timeout')); });
    if (data) req.write(JSON.stringify(data));
    req.end();
  });
}

async function getWalletBalance() {
  try {
    const timestamp = Date.now();
    const queryString = `timestamp=${timestamp}`;
    const signature = signRequest(queryString);
    const url = `${CONFIG.getBaseURL()}/fapi/v2/balance?${queryString}&signature=${signature}`;
    
    const response = await httpRequest(url, 'GET', null, {
      'X-MBX-APIKEY': CONFIG.apiKey,
    });
    
    if (response.data && Array.isArray(response.data)) {
      const usdt = response.data.find(b => b.asset === 'USDT');
      return parseFloat(usdt?.availableBalance || usdt?.balance || 0);
    }
    return 0;
  } catch (e) {
    return 0;
  }
}

async function getPositions() {
  try {
    const timestamp = Date.now();
    const queryString = `timestamp=${timestamp}`;
    const signature = signRequest(queryString);
    const url = `${CONFIG.getBaseURL()}/fapi/v2/account?${queryString}&signature=${signature}`;
    
    const response = await httpRequest(url, 'GET', null, {
      'X-MBX-APIKEY': CONFIG.apiKey,
    });
    
    if (!response.data?.positions) return [];
    
    return response.data.positions
      .filter(p => parseFloat(p.positionAmt) !== 0)
      .map(p => ({
        symbol: p.symbol,
        positionAmt: Math.abs(parseFloat(p.positionAmt)),
        entryPrice: parseFloat(p.entryPrice),
        markPrice: parseFloat(p.markPrice) || 0,
        unrealizedPnL: parseFloat(p.unrealizedPnl) || 0,
        roe: parseFloat(p.roe) || 0,
        positionSide: p.positionSide,
      }));
  } catch (e) {
    return [];
  }
}

async function getPrice(symbol) {
  try {
    const url = `https://api.binance.com/api/v3/ticker/24hr?symbol=${symbol}`;
    const response = await httpRequest(url);
    return parseFloat(response.data?.lastPrice || 0);
  } catch (e) {
    return 0;
  }
}

let lastTelegramUpdate = 0;
let lastPnL = 0;

async function sendTelegramLiveUpdate(status) {
  if (!CONFIG.telegramEnabled || !CONFIG.telegramToken || !CONFIG.telegramChatId) {
    return;
  }

  const now = Date.now();
  const pnlChanged = Math.abs(status.totalPnL - lastPnL) > 1;
  const shouldUpdate = pnlChanged || (now - lastTelegramUpdate > 60000);
  
  if (!shouldUpdate && status.positions.length > 0) return;
  
  lastTelegramUpdate = now;
  lastPnL = status.totalPnL;

  let message = `*🔄 GOBOT LIVE UPDATE*\n\n`;
  message += `\`══════════════════════\`\n`;
  message += `Mode: ${CONFIG.useTestnet ? '🧪 TESTNET' : '💰 MAINNET'}\n`;
  message += `Balance: \`$${status.balance.toFixed(2)}\`\n`;
  message += `\`══════════════════════\`\n`;
  
  if (status.positions.length > 0) {
    message += `\n📊 *POSITIONS (${status.positions.length})*\n`;
    
    for (const p of status.positions) {
      const emoji = p.pnl >= 0 ? '🟢' : '🔴';
      const side = p.side === 'LONG' ? '📈' : '📉';
      message += `\n${emoji} *${p.symbol}* ${side}\n`;
      message += `   PnL: \`$${p.pnl.toFixed(2)} (${p.pnlPercent.toFixed(2)}%)\`\n`;
      message += `   Entry: \`$${p.entryPrice.toFixed(4)}\`\n`;
      message += `   Size: \`${p.size.toFixed(2)}\`\n`;
    }
    
    message += `\n📈 *TOTAL PNL: \`$${status.totalPnL.toFixed(2)}\`*\n`;
  } else {
    message += `\n📭 *No open positions*\n`;
    message += `\n🚀 *Ready for new trades*\n`;
  }
  
  message += `\n⏰ _${new Date().toLocaleTimeString()}_`;

  const url = `https://api.telegram.org/bot${CONFIG.telegramToken}/sendMessage`;
  
  try {
    await httpRequest(url, 'POST', {
      chat_id: CONFIG.telegramChatId,
      text: message,
      parse_mode: 'Markdown',
    });
  } catch (e) {}
}

function drawDashboard(status) {
  process.stdout.write(clear);
  process.stdout.write(cursorHide);
  
  const modeColor = CONFIG.useTestnet ? 'yellow' : 'green';
  const modeIcon = CONFIG.useTestnet ? '🧪 TESTNET' : '💰 MAINNET';
  
  log('╔══════════════════════════════════════════════════════════════╗\n', 'magenta');
  log('║                    🚀 GOBOT LIVE DASHBOARD                   ║\n', 'magenta');
  log('╚══════════════════════════════════════════════════════════════╝\n', 'magenta');
  
  log(`\n┌─────────────────────────────────────────────────────────────┐\n`, modeColor);
  log(`│  ${modeIcon}  │  ${new Date().toLocaleString()}                    │\n`, modeColor);
  log(`└─────────────────────────────────────────────────────────────┘\n`, modeColor);
  
  const balanceColor = status.balance >= CONFIG.minBalance ? 'green' : status.balance > 1 ? 'yellow' : 'red';
  log(`\n┌─────────────────────────────────────────────────────────────┐\n`, 'cyan');
  log(`│  💰 WALLET BALANCE: $${status.balance.toFixed(2).padEnd(42)}│\n`, balanceColor);
  log(`│  🎯 MIN REQUIRED:   $${CONFIG.minBalance}.00${' '.repeat(39)}│\n`, 'cyan');
  log(`│  📊 STATUS:         ${status.canTrade ? '✅ READY TO TRADE'.padEnd(42) : '⏸️ WAITING'.padEnd(42)}│\n`, status.canTrade ? 'green' : 'yellow');
  log(`└─────────────────────────────────────────────────────────────┘\n`, 'cyan');
  
  log(`\n┌─────────────────────────────────────────────────────────────┐\n`, 'blue');
  log(`│  📊 OPEN POSITIONS: ${status.positions.length}                                        │\n`, 'blue');
  log(`├─────────────────────────────────────────────────────────────┤\n`, 'blue');
  
  if (status.positions.length === 0) {
    log(`│                                                             │\n`, 'cyan');
    log(`│          📭 NO ACTIVE POSITIONS                             │\n`, 'cyan');
    log(`│          Waiting for trading signals...                     │\n`, 'cyan');
    log(`│                                                             │\n`, 'cyan');
  } else {
    for (const p of status.positions) {
      const pnlColor = p.pnl >= 0 ? 'green' : 'red';
      const sideIcon = p.side === 'LONG' ? '📈 LONG' : '📉 SHORT';
      const pnlIcon = p.pnl >= 0 ? '🟢' : '🔴';
      
      log(`│  ${pnlIcon} ${p.symbol.padEnd(10)} │ ${sideIcon.padEnd(12)} │ Size: ${p.size.toFixed(2).padEnd(12)}│\n`, 'white');
      log(`│  Entry: $${p.entryPrice.toFixed(4)} │ Current: $${p.currentPrice.toFixed(4).padEnd(10)}│\n`, 'cyan');
      log(`│  ${pnlIcon} PnL: $${p.pnl.toFixed(2).padEnd(10)} (${p.pnlPercent >= 0 ? '+' : ''}${p.pnlPercent.toFixed(2)}%)                          │\n`, pnlColor);
      log(`├─────────────────────────────────────────────────────────────┤\n`, 'blue');
    }
    
    const totalColor = status.totalPnL >= 0 ? 'green' : 'red';
    log(`│  📈 TOTAL PnL: $${status.totalPnL.toFixed(2).padEnd(45)}│\n`, totalColor);
  }
  
  log(`└─────────────────────────────────────────────────────────────┘\n`, 'blue');
  
  log(`\n┌─────────────────────────────────────────────────────────────┐\n`, 'magenta');
  log(`│  📱 TELEGRAM: ${CONFIG.telegramEnabled ? '✅ CONNECTED' : '❌ DISABLED'.padEnd(41)}│\n`, CONFIG.telegramEnabled ? 'green' : 'red');
  log(`│  🔄 AUTO-REFRESH: ${(CONFIG.refreshInterval/1000)}s${' '.repeat(41)}│\n`, 'cyan');
  log(`│  ⏹️  Press Ctrl+C to exit                                    │\n`, 'yellow');
  log(`└─────────────────────────────────────────────────────────────┘\n`, 'magenta');
  
  process.stdout.write(cursorShow);
}

async function getStatus() {
  const balance = await getWalletBalance();
  const positions = await getPositions();
  
  const status = {
    timestamp: new Date().toISOString(),
    mode: CONFIG.useTestnet ? 'TESTNET' : 'MAINNET',
    balance,
    positions: [],
    totalPnL: 0,
    canTrade: false,
    needsWait: false,
    waitReason: '',
  };
  
  for (const pos of positions) {
    const currentPrice = await getPrice(pos.symbol);
    const pnl = currentPrice > 0 
      ? (currentPrice - pos.entryPrice) * pos.positionAmt * (pos.positionSide === 'LONG' ? 1 : -1)
      : pos.unrealizedPnL;
    const pnlPercent = pos.entryPrice > 0 ? (pnl / (pos.entryPrice * pos.positionAmt)) * 100 : 0;
    
    status.positions.push({
      symbol: pos.symbol,
      side: pos.positionSide,
      size: pos.positionAmt,
      entryPrice: pos.entryPrice,
      currentPrice,
      pnl,
      pnlPercent,
    });
    
    status.totalPnL += pnl;
  }
  
  if (status.positions.length > 0) {
    status.needsWait = true;
    status.waitReason = 'Active positions';
  }
  
  if (balance < CONFIG.minBalance) {
    status.needsWait = true;
    status.waitReason = balance < 1 
      ? 'Critical: Balance too low' 
      : `Insufficient balance ($${balance.toFixed(2)})`;
  } else if (status.positions.length === 0) {
    status.canTrade = true;
  }
  
  return status;
}

async function main() {
  console.clear();
  
  if (!CONFIG.apiKey || !CONFIG.secret) {
    console.log('ERROR: BINANCE_API_KEY and BINANCE_API_SECRET required');
    console.log('Add to .env file');
    process.exit(1);
  }
  
  log('🔄 Connecting to Binance...\n', 'cyan');
  
  let checkCount = 0;
  let lastStatus = null;
  
  try {
    while (true) {
      checkCount++;
      const status = await getStatus();
      drawDashboard(status);
      
      if (!lastStatus || JSON.stringify(status.positions) !== JSON.stringify(lastStatus.positions)) {
        await sendTelegramLiveUpdate(status);
      }
      
      lastStatus = status;
      
      await new Promise(r => setTimeout(r, CONFIG.refreshInterval));
    }
  } catch (e) {
    process.stdout.write(cursorShow);
    console.log(`\n\nError: ${e.message}`);
    console.log('Make sure Binance API keys are correct');
    process.exit(1);
  }
}

process.on('SIGINT', () => {
  process.stdout.write(cursorShow);
  console.clear();
  console.log('\n👋 Dashboard closed\n');
  process.exit(0);
});

if (require.main === module) {
  main();
}

module.exports = { getStatus, drawDashboard };
