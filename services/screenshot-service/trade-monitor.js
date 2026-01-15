#!/usr/bin/env node

/**
 * GOBOT Live Trade Monitor v2
 * - Monitors open positions on Binance
 * - Checks wallet balance
 * - Blocks new trades if balance too low
 * - Sends Telegram alerts
 * - Auto-restarts when safe
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
  minPositionSize: 5,
  
  telegramToken: process.env.TELEGRAM_TOKEN || '',
  telegramChatId: process.env.AUTHORIZED_CHAT_ID || process.env.TELEGRAM_CHAT_ID || '',
  telegramEnabled: (process.env.TELEGRAM_NOTIFICATIONS === 'true' || process.env.TELEGRAM_TOKEN) && process.env.TELEGRAM_TOKEN && (process.env.AUTHORIZED_CHAT_ID || process.env.TELEGRAM_CHAT_ID),
  
  checkInterval: 60000,
  
  getBaseURL() {
    return this.useTestnet 
      ? 'https://testnet.binancefuture.com' 
      : 'https://fapi.binance.com';
  },
};

const C = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  red: '\x1b[31m',
  cyan: '\x1b[36m',
  magenta: '\x1b[35m',
};

function log(msg, color = 'reset') {
  console.log(`${C[color]}${msg}${C.reset}`);
}

function logSection(title) {
  console.log('');
  log('═══════════════════════════════════════════════════════════════', 'magenta');
  log(`  ${title}`, 'magenta');
  log('═══════════════════════════════════════════════════════════════', 'magenta');
}

function httpRequest(url, method = 'GET', data = null, headers = {}) {
  return new Promise((resolve, reject) => {
    const parsed = new URL(url);
    const client = parsed.protocol === 'https:' ? https : http;
    
    const options = {
      hostname: parsed.hostname,
      port: parsed.port || 443,
      path: parsed.pathname + parsed.search,
      method,
      headers: { 'Content-Type': 'application/json', ...headers },
      timeout: 15000,
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

function signRequest(queryString) {
  return crypto.createHmac('sha256', CONFIG.secret)
    .update(queryString)
    .digest('hex');
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
    log(`Balance fetch error: ${e.message}`, 'red');
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
        isolatedMargin: parseFloat(p.isolatedMargin) || 0,
      }));
  } catch (e) {
    log(`Positions fetch error: ${e.message}`, 'red');
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

async function sendTelegram(message) {
  if (!CONFIG.telegramEnabled || !CONFIG.telegramToken || !CONFIG.telegramChatId) {
    log('Telegram not configured', 'yellow');
    return false;
  }

  const url = `https://api.telegram.org/bot${CONFIG.telegramToken}/sendMessage`;
  
  try {
    const response = await httpRequest(url, 'POST', {
      chat_id: CONFIG.telegramChatId,
      text: message,
      parse_mode: 'Markdown',
    });
    return response.data?.ok;
  } catch (e) {
    log(`Telegram error: ${e.message}`, 'red');
    return false;
  }
}

async function checkAndMonitor() {
  logSection('🔍 GOBOT STATUS CHECK');
  log(`Mode: ${CONFIG.useTestnet ? 'TESTNET 🧪' : 'MAINNET 💰'}`, 'cyan');
  log(`Time: ${new Date().toLocaleString()}`, 'cyan');
  
  const balance = await getWalletBalance();
  const positions = await getPositions();
  
  log(`\n💰 Wallet Balance: $${balance.toFixed(2)}`, 
      balance >= CONFIG.minBalance ? 'green' : balance > 1 ? 'yellow' : 'red');
  
  const status = {
    timestamp: new Date().toISOString(),
    mode: CONFIG.useTestnet ? 'TESTNET' : 'MAINNET',
    balance,
    positions: [],
    canTrade: false,
    needsWait: false,
    waitReason: '',
  };
  
  if (positions.length > 0) {
    log(`\n📊 OPEN POSITIONS: ${positions.length}`, 'yellow');
    
    for (const pos of positions) {
      const currentPrice = await getPrice(pos.symbol);
      const pnl = currentPrice > 0 
        ? (currentPrice - pos.entryPrice) * pos.positionAmt * (pos.positionSide === 'LONG' ? 1 : -1)
        : pos.unrealizedPnL;
      const pnlPercent = pos.entryPrice > 0 ? (pnl / (pos.entryPrice * pos.positionAmt)) * 100 : 0;
      
      const pnlEmoji = pnl >= 0 ? '🟢' : '🔴';
      const pnlColor = pnl >= 0 ? 'green' : 'red';
      
      log(`\n${pnlEmoji} ${pos.symbol}`, 'cyan');
      log(`   Side: ${pos.positionSide}`);
      log(`   Size: ${pos.positionAmt.toFixed(4)}`);
      log(`   Entry: $${pos.entryPrice.toFixed(4)}`);
      log(`   Current: $${currentPrice.toFixed(4) || 'N/A'}`);
      log(`   PnL: $${pnl.toFixed(2)} (${pnlPercent.toFixed(2)}%)`, pnlColor);
      
      status.positions.push({
        symbol: pos.symbol,
        side: pos.positionSide,
        size: pos.positionAmt,
        entryPrice: pos.entryPrice,
        currentPrice,
        pnl,
        pnlPercent,
      });
    }
    
    status.needsWait = true;
    status.waitReason = 'Active positions detected - monitoring';
  } else {
    log('\n📭 No open positions', 'green');
  }
  
  if (balance < CONFIG.minBalance) {
    log(`\n⚠️ BALANCE TOO LOW: $${balance.toFixed(2)} < $${CONFIG.minBalance}`, 'red');
    log('Cannot open new trades', 'yellow');
    
    status.needsWait = true;
    status.waitReason = `Insufficient balance ($${balance.toFixed(2)})`;
    
    if (balance < 1) {
      status.waitReason = 'Critical: Balance near zero - waiting for recovery';
    }
  } else if (positions.length === 0) {
    status.canTrade = true;
    log('\n✅ READY FOR NEW TRADES', 'green');
    log(`   Available: $${balance.toFixed(2)}`, 'green');
  }
  
  const totalPnL = status.positions.reduce((sum, p) => sum + p.pnl, 0);
  log(`\n📈 Total PnL: $${totalPnL.toFixed(2)}`, totalPnL >= 0 ? 'green' : 'red');
  
  return status;
}

async function sendStatusAlert(status) {
  let message = `*GOBOT STATUS UPDATE*\n\n`;
  message += `Mode: \`${status.mode}\`\n`;
  message += `Balance: \`$${status.balance.toFixed(2)}\`\n`;
  message += `Positions: \`${status.positions.length}\`\n`;
  
  if (status.positions.length > 0) {
    message += `\n📊 *OPEN POSITIONS*\n`;
    for (const p of status.positions) {
      const emoji = p.pnl >= 0 ? '🟢' : '🔴';
      message += `${emoji} \`${p.symbol}\`: \`$${p.pnl.toFixed(2)} (${p.pnlPercent.toFixed(2)}%)\`\n`;
    }
  }
  
  if (status.canTrade) {
    message += `\n🚀 *READY FOR NEW TRADES*`;
  } else if (status.needsWait) {
    message += `\n⏸️ *WAITING*: ${status.waitReason}`;
  }
  
  message += `\n⏰ ${new Date().toLocaleTimeString()}`;
  
  await sendTelegram(message);
}

async function continuousMonitor() {
  logSection('🔄 GOBOT LIVE MONITOR');
  log('Monitoring active trades and balance...', 'cyan');
  log(`Check interval: ${CONFIG.checkInterval / 1000}s`, 'cyan');
  log('Press Ctrl+C to stop\n', 'yellow');
  
  let checkCount = 0;
  let lastStatus = null;
  
  while (true) {
    checkCount++;
    const time = new Date().toLocaleTimeString();
    log(`\n--- Check #${checkCount} [${time}] ---`, 'cyan');
    
    const status = await checkAndMonitor();
    
    if (lastStatus && JSON.stringify(status.positions) !== JSON.stringify(lastStatus.positions)) {
      log('\n🔔 Position change detected!', 'magenta');
      await sendStatusAlert(status);
    } else if (checkCount % 5 === 0) {
      await sendStatusAlert(status);
    }
    
    lastStatus = status;
    
    if (status.canTrade) {
      log('\n✅ System ready - new trades can be opened', 'green');
      log('Run: node auto-trade.js <symbol> <balance>', 'cyan');
    } else if (status.needsWait) {
      log(`\n⏳ Status: WAITING - ${status.waitReason}`, 'yellow');
    }
    
    await new Promise(r => setTimeout(r, CONFIG.checkInterval));
  }
}

async function main() {
  const args = process.argv.slice(2);
  
  if (args.includes('--help') || args.includes('-h')) {
    console.log(`
GOBOT Live Trade Monitor v2
============================

Usage: node trade-monitor.js [options]

Options:
  --once       Run single check then exit (default)
  --continuous Run continuous monitoring
  --balance    Show balance only
  --positions  Show positions only

Features:
  - Checks Binance wallet balance
  - Monitors all open positions
  - Calculates real-time PnL
  - Blocks new trades if balance low
  - Sends Telegram alerts

Thresholds:
  - Min balance for trading: $${CONFIG.minBalance}
  - Min position size: $${CONFIG.minPositionSize}
`);
    process.exit(0);
  }
  
  if (!CONFIG.apiKey || !CONFIG.secret) {
    log('ERROR: BINANCE_API_KEY and BINANCE_API_SECRET required', 'red');
    process.exit(1);
  }
  
  if (args.includes('--continuous')) {
    try {
      await continuousMonitor();
    } catch (e) {
      log(`Monitor error: ${e.message}`, 'red');
    }
  } else {
    const status = await checkAndMonitor();
    await sendStatusAlert(status);
  }
}

process.on('SIGINT', () => {
  log('\n\n👋 Monitor stopped', 'yellow');
  process.exit(0);
});

if (require.main === module) {
  main();
}

module.exports = { checkAndMonitor, getWalletBalance, getPositions };
