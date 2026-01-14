const http = require('http');
const url = require('url');
const querystring = require('querystring');

const PORT = 5678;

// Simple router
const routes = {
  'GET': {
    '/healthz': handleHealth,
    '/workflows': handleWorkflows
  },
  'POST': {
    '/webhook/quantcrawler-analysis': handleQuantCrawler,
    '/webhook/trade-signal': handleTradeSignal,
    '/webhook/risk-alert': handleRiskAlert,
    '/workflows/toggle': handleToggle
  }
};

// Parse body
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

// Send JSON response
function sendJSON(res, statusCode, data) {
  res.writeHead(statusCode, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(data));
}

// Health check
async function handleHealth(req, res) {
  sendJSON(res, 200, {
    status: 'healthy',
    service: 'gobot-n8n-alt',
    timestamp: new Date().toISOString()
  });
}

// List workflows
async function handleWorkflows(req, res) {
  sendJSON(res, 200, {
    workflows: [{
      id: 'quantcrawler-analysis',
      name: 'QuantCrawler Analysis',
      enabled: true,
      steps: ['Webhook', 'Parse', 'Screenshots', 'QuantCrawler', 'Format', 'Send to Go Bot']
    }]
  });
}

// QuantCrawler Analysis workflow
async function handleQuantCrawler(req, res) {
  const body = await parseBody(req);
  const { symbol, account_balance, current_price } = body;
  
  if (!symbol) {
    return sendJSON(res, 400, { error: 'Missing symbol' });
  }
  
  console.log('\n==============================================');
  console.log('  QuantCrawler Analysis Workflow');
  console.log('==============================================');
  console.log(`Symbol:    ${symbol}`);
  console.log(`Balance:   ${account_balance || 1000}`);
  console.log(`Price:     ${current_price || 0}`);
  
  const startTime = Date.now();
  
  // Mock QuantCrawler response
  const directions = ['LONG', 'SHORT', 'STAY AWAY'];
  const direction = directions[Math.floor(Math.random() * 3)];
  const confidence = Math.floor(Math.random() * 50) + 50;
  const price = current_price || 0.00001;
  
  const response = {
    success: true,
    symbol,
    ticker: symbol.replace('USDT', ''),
    current_price: price,
    entry: price,
    confidence,
    direction,
    recommendation: `${direction} signal at ${confidence}% confidence`,
    options: [{
      name: 'Standard Position',
      contracts: 1,
      risk_per_contract: 100,
      stop_price: price * 0.995,
      target_price: price * 1.015,
      risk_reward_ratio: 1.5,
      recommended: true
    }],
    timeframes: {
      '15m': 'Bullish momentum building',
      '5m': 'Consolidating near support',
      '1m': 'Short-term volatility high'
    },
    key_levels: {
      support: price * 0.99,
      resistance: price * 1.01
    },
    confluence: '2/3 timeframes agree',
    request_id: `req_${Date.now()}`,
    processed_at: new Date().toISOString(),
    workflow_duration_ms: Date.now() - startTime
  };
  
  console.log(`Direction: ${response.direction}`);
  console.log(`Confidence: ${response.confidence}%`);
  console.log(`Duration: ${response.workflow_duration_ms}ms`);
  console.log('==============================================\n');
  
  sendJSON(res, 200, response);
}

// Trade signal
async function handleTradeSignal(req, res) {
  const body = await parseBody(req);
  console.log('\n==============================================');
  console.log('  Trade Signal Received');
  console.log('==============================================');
  console.log(`Symbol:    ${body.symbol}`);
  console.log(`Action:    ${body.action}`);
  console.log(`Confidence: ${(body.confidence * 100).toFixed(0)}%`);
  console.log(`Entry:     ${body.entry_price}`);
  console.log(`Stop:      ${body.stop_loss}`);
  console.log(`Target:    ${body.take_profit}`);
  console.log('==============================================\n');
  
  sendJSON(res, 200, { received: true, symbol: body.symbol });
}

// Risk alert
async function handleRiskAlert(req, res) {
  const body = await parseBody(req);
  console.log('\n⚠️  Risk Alert:', body);
  sendJSON(res, 200, { received: true });
}

// Toggle workflow
async function handleToggle(req, res) {
  sendJSON(res, 200, { id: 'quantcrawler-analysis', enabled: true });
}

// Main server
const server = http.createServer(async (req, res) => {
  const parsedUrl = url.parse(req.url, true);
  const pathname = parsedUrl.pathname;
  const method = req.method;
  
  console.log(`[${new Date().toISOString()}] ${method} ${pathname}`);
  
  const methodRoutes = routes[method] || {};
  const handler = methodRoutes[pathname];
  
  if (handler) {
    await handler(req, res);
  } else {
    sendJSON(res, 404, {
      error: 'Not found',
      endpoints: [
        'POST /webhook/quantcrawler-analysis',
        'POST /webhook/trade-signal',
        'POST /webhook/risk-alert',
        'GET /healthz',
        'GET /workflows'
      ]
    });
  }
});

server.listen(PORT, () => {
  console.log('');
  console.log('==============================================');
  console.log('  GOBOT - N8N Alternative Server');
  console.log('==============================================');
  console.log(`  Server: http://localhost:${PORT}`);
  console.log('');
  console.log('Endpoints:');
  console.log(`  POST /webhook/quantcrawler-analysis`);
  console.log(`  POST /webhook/trade-signal`);
  console.log(`  POST /webhook/risk-alert`);
  console.log(`  GET  /healthz`);
  console.log(`  GET  /workflows`);
  console.log('');
  console.log('Test:');
  console.log(`  curl -X POST http://localhost:${PORT}/webhook/quantcrawler-analysis \\`);
  console.log(`    -H "Content-Type: application/json" \\`);
  console.log(`    -d '{"symbol":"1000PEPEUSDT","account_balance":1000}'`);
  console.log('==============================================');
});
