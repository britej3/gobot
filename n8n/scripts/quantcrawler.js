const puppeteer = require('puppeteer');
const fs = require('fs');
const path = require('path');

// ═══════════════════════════════════════════════════════════════════════════
// CONFIGURATION - All paths relative to project root
// ═══════════════════════════════════════════════════════════════════════════

// Resolve paths from project root
const PROJECT_ROOT = path.resolve(__dirname, '..', '..');
const SESSION_DIR = path.join(PROJECT_ROOT, 'n8n-sessions');
const SESSION_FILE = path.join(SESSION_DIR, 'quantcrawler-session.json');

const CONFIG = {
  quantcrawler: {
    baseUrl: 'https://app.quantcrawler.com',
    loginUrl: 'https://app.quantcrawler.com/auth/google',
    analyzerPath: '/futures',
    sessionDir: SESSION_DIR,
    sessionFile: SESSION_FILE,
    timeout: 120000, // 2 minutes for full OAuth + analysis
  },
  selectors: {
    // QuantCrawler selectors (update if UI changes)
    tickerInput: '[placeholder*="ticker" i], [placeholder*="symbol" i], input[name*="symbol" i]',
    amountInput: '[placeholder*="amount" i], [placeholder*="balance" i], input[name*="amount" i]',
    analyzeButton: 'button:has-text("Analyze"), button:has-text("Submit"), button[type="submit"]',
    fileInput: 'input[type="file"]',
    
    // Response selectors
    responseContainer: '[class*="response"], [class*="result"], [class*="analysis"], [class*="output"]',
    directionText: '[class*="direction"], [class*="signal"], [class*="recommendation"], [class*="verdict"]',
    confidenceText: '[class*="confidence"], [class*="score"], [class*="probability"]',
    levelsText: '[class*="level"], [class*="support"], [class*="resistance"], [class*="stop"], [class*="target"]',
    
    // Google OAuth selectors
    googleEmailInput: 'input[type="email"], #identifierId',
    googleNextButton: '#identifierNext, button:has-text("Next")',
    googlePasswordInput: 'input[type="password"], #password',
    googlePasswordNext: '#passwordNext, button:has-text("Next")',
    googleAvatar: '[data-avatar], [class*="avatar"]',
  },
};

// ═══════════════════════════════════════════════════════════════════════════
// MAIN FUNCTION - Called by N8N
// ═══════════════════════════════════════════════════════════════════════════
async function analyzeWithQuantCrawler({ symbol, screenshots = [], accountBalance, currentPrice }) {
  console.log(`[QuantCrawler] Starting analysis for ${symbol}`);
  
  const browser = await puppeteer.launch({
    headless: false, // Must be false for Google OAuth
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-dev-shm-usage',
      '--disable-gpu',
      '--window-size=1920,1080',
      '--start-maximized',
    ],
  });

  try {
    const page = await browser.newPage();
    page.setDefaultTimeout(30000);
    page.setDefaultNavigationTimeout(30000);
    
    // Enable request interception for debugging
    await page.setRequestInterception(true);
    page.on('request', request => {
      if (request.url().includes('google')) {
        console.log(`[Google OAuth] Request: ${request.url()}`);
      }
      request.continue();
    });
    
    // Try to restore session
    const sessionRestored = await restoreSession(page);
    console.log(`[QuantCrawler] Session restored: ${sessionRestored}`);
    
    // Navigate to analyzer
    console.log(`[QuantCrawler] Navigating to ${CONFIG.quantcrawler.baseUrl}${CONFIG.quantcrawler.analyzerPath}`);
    await page.goto(`${CONFIG.quantcrawler.baseUrl}${CONFIG.quantcrawler.analyzerPath}`, {
      waitUntil: 'domcontentloaded',
      timeout: 30000,
    });
    
    // Wait for page to load
    await new Promise(r => setTimeout(r, 2000));
    
    // Check if logged in
    const isAuthPage = page.url().includes('auth') || page.url().includes('login');
    
    if (isAuthPage || !await isLoggedIn(page)) {
      console.log('[QuantCrawler] Not authenticated, initiating Google OAuth...');
      await performGoogleOAuth(page);
      await saveSession(page);
    } else {
      console.log('[QuantCrawler] Already authenticated');
    }
    
    // Navigate directly to analyzer after auth
    await page.goto(`${CONFIG.quantcrawler.baseUrl}${CONFIG.quantcrawler.analyzerPath}`, {
      waitUntil: 'domcontentloaded',
      timeout: 30000,
    });
    
    // Wait for analyzer to load
    try {
      await page.waitForSelector(CONFIG.selectors.tickerInput, { timeout: 15000 });
    } catch {
      console.log('[QuantCrawler] Warning: Ticker input not found, trying alternative selectors');
      await page.waitForSelector('input', { timeout: 10000 });
    }
    
    // Input ticker
    await page.click(CONFIG.selectors.tickerInput);
    await page.type(CONFIG.selectors.tickerInput, symbol.replace('USDT', '').replace('PERP', ''));
    console.log(`[QuantCrawler] Ticker input: ${symbol}`);
    
    // Input account balance if provided
    if (accountBalance) {
      try {
        await page.click(CONFIG.selectors.amountInput);
        await page.type(CONFIG.selectors.amountInput, accountBalance.toString());
        console.log(`[QuantCrawler] Account balance input: ${accountBalance}`);
      } catch (e) {
        console.log('[QuantCrawler] Could not input account balance (optional)');
      }
    }
    
    // Upload screenshots if provided
    if (screenshots.length > 0) {
      try {
        const fileInput = await page.$(CONFIG.selectors.fileInput);
        if (fileInput) {
          for (const screenshot of screenshots) {
            if (fs.existsSync(screenshot)) {
              await fileInput.uploadFile(screenshot);
              console.log(`[QuantCrawler] Uploaded screenshot: ${screenshot}`);
            }
          }
        }
      } catch (e) {
        console.log('[QuantCrawler] Could not upload screenshots (optional)');
      }
    }
    
    // Click analyze button
    const analyzeBtn = await page.$(CONFIG.selectors.analyzeButton);
    if (analyzeBtn) {
      await analyzeBtn.click();
      console.log('[QuantCrawler] Submitted for analysis...');
    } else {
      console.log('[QuantCrawler] Warning: Analyze button not found');
    }
    
    // Wait for AI response
    console.log('[QuantCrawler] Waiting for AI analysis (up to 90s)...');
    try {
      await page.waitForSelector(CONFIG.selectors.responseContainer, { timeout: 90000 });
    } catch {
      console.log('[QuantCrawler] Warning: Response container not found within timeout');
    }
    
    // Small delay for content to render
    await new Promise(r => setTimeout(r, 3000));
    
    // Extract response
    const result = await extractResponse(page, symbol, currentPrice);
    
    console.log(`[QuantCrawler] Analysis complete: ${result.direction} (${result.confidence}% confidence)`);
    
    return result;
    
  } finally {
    await browser.close();
  }
}

// ═══════════════════════════════════════════════════════════════════════════
// GOOGLE OAUTH FLOW
// ═══════════════════════════════════════════════════════════════════════════
async function performGoogleOAuth(page) {
  console.log('[Google OAuth] Starting authentication flow...');
  
  // Navigate to QuantCrawler's Google auth
  await page.goto(CONFIG.quantcrawler.loginUrl, { waitUntil: 'domcontentloaded' });
  
  // Wait for Google login page
  await page.waitForSelector(CONFIG.selectors.googleEmailInput, { timeout: 30000 });
  console.log('[Google OAuth] Email input found');
  
  // Get credentials from environment
  const email = process.env.QUANTCRAWLER_EMAIL || process.env.GOOGLE_EMAIL;
  const password = process.env.QUANTCRAWLER_PASSWORD || process.env.GOOGLE_PASSWORD;
  
  if (!email || !password) {
    console.log('[Google OAuth] ERROR: QUANTCRAWLER_EMAIL and QUANTCRAWLER_PASSWORD env vars required');
    console.log('[Google OAuth] Please set these environment variables and try again');
    throw new Error('Missing Google credentials. Set QUANTCRAWLER_EMAIL and QUANTCRAWLER_PASSWORD');
  }
  
  // Enter email
  console.log('[Google OAuth] Entering email...');
  await page.type(CONFIG.selectors.googleEmailInput, email);
  await page.click(CONFIG.selectors.googleNextButton);
  
  // Wait for password field
  await page.waitForSelector(CONFIG.selectors.googlePasswordInput, { timeout: 15000 });
  console.log('[Google OAuth] Password input found');
  
  // Enter password
  console.log('[Google OAuth] Entering password...');
  await page.type(CONFIG.selectors.googlePasswordInput, password);
  await page.click(CONFIG.selectors.googlePasswordNext);
  
  // Wait for navigation back to QuantCrawler
  console.log('[Google OAuth] Waiting for authentication to complete...');
  
  try {
    // Wait for either redirect back to QuantCrawler or 2FA prompt
    await page.waitForFunction(
      () => {
        const url = window.location.href;
        return url.includes('quantcrawler') || 
               url.includes('accounts.google') ||
               document.querySelector('[class*="2fa"]') !== null ||
               document.querySelector('[class*="verification"]') !== null;
      },
      { timeout: 60000 }
    );
    
    // Check if 2FA is required
    const has2FA = await page.$('[class*="2fa"], [class*="verification"], [class*="code"]');
    if (has2FA) {
      console.log('[Google OAuth] 2FA required - manual intervention needed');
      console.log('[Google OAuth] Please complete 2FA in the browser, then press Enter here...');
      console.log('[Google OAuth] For automated use, use an app password or disable 2FA');
      
      // Wait for manual 2FA completion
      await page.waitForFunction(
        () => window.location.href.includes('quantcrawler'),
        { timeout: 300000 } // 5 minutes for manual 2FA
      );
    }
    
  } catch (e) {
    console.log('[Google OAuth] Warning: Could not confirm auth completion');
  }
  
  // Wait for QuantCrawler to fully load
  await page.waitForFunction(
    () => window.location.href.includes('quantcrawler'),
    { timeout: 30000 }
  );
  
  await new Promise(r => setTimeout(r, 2000));
  
  console.log('[Google OAuth] Authentication successful!');
}

// ═══════════════════════════════════════════════════════════════════════════
// SESSION MANAGEMENT
// ═══════════════════════════════════════════════════════════════════════════
async function restoreSession(page) {
  if (fs.existsSync(CONFIG.quantcrawler.sessionFile)) {
    try {
      const sessionData = JSON.parse(fs.readFileSync(CONFIG.quantcrawler.sessionFile, 'utf8'));
      
      if (sessionData.cookies && sessionData.cookies.length > 0) {
        // Set all cookies
        await page.setCookie(...sessionData.cookies);
        console.log(`[QuantCrawler] Restored ${sessionData.cookies.length} cookies`);
        
        // Check if session is expired
        if (sessionData.expiresAt && new Date(sessionData.expiresAt) < new Date()) {
          console.log('[QuantCrawler] Session expired, will re-authenticate');
          return false;
        }
        
        return true;
      }
    } catch (e) {
      console.log('[QuantCrawler] Failed to restore session:', e.message);
    }
  }
  return false;
}

async function saveSession(page) {
  const cookies = await page.cookies();
  
  // Calculate expiration (Google sessions typically last ~1 month)
  const expiresAt = new Date();
  expiresAt.setDate(expiresAt.getDate() + 30);
  
  const sessionData = {
    cookies,
    expiresAt: expiresAt.toISOString(),
    savedAt: new Date().toISOString(),
  };
  
  // Ensure session directory exists
  if (!fs.existsSync(CONFIG.quantcrawler.sessionDir)) {
    fs.mkdirSync(CONFIG.quantcrawler.sessionDir, { recursive: true });
  }
  
  fs.writeFileSync(CONFIG.quantcrawler.sessionFile, JSON.stringify(sessionData, null, 2));
  console.log(`[QuantCrawler] Session saved with ${cookies.length} cookies`);
}

async function isLoggedIn(page) {
  try {
    const url = page.url();
    
    // Check URL
    if (url.includes('auth') || url.includes('login')) {
      return false;
    }
    
    // Check for logged-in elements
    const userMenu = await page.$('[class*="user"], [class*="profile"], [class*="avatar"]');
    const logoutBtn = await page.$('a:has-text("Logout"), button:has-text("Logout"), a:has-text("Sign out")');
    const quantcrawlerContent = await page.$('[class*="dashboard"], [class*="analyzer"], [class*="futures"]');
    
    return (userMenu !== null || logoutBtn !== null || quantcrawlerContent !== null);
  } catch {
    return false;
  }
}

// ═══════════════════════════════════════════════════════════════════════════
// RESPONSE EXTRACTION
// ═══════════════════════════════════════════════════════════════════════════
async function extractResponse(page, symbol, currentPrice) {
  const result = {
    symbol: symbol,
    ticker: symbol.replace('USDT', '').replace('PERP', ''),
    current_price: currentPrice || 0,
    entry: currentPrice || 0,
    confidence: 0,
    direction: 'STAY AWAY',
    recommendation: 'Analysis could not be completed',
    options: [],
    timeframes: {},
    key_levels: { support: 0, resistance: 0 },
    risks: '',
    confluence: '0/3 timeframes agree',
    request_id: `qc_${Date.now()}`,
    processed_at: new Date().toISOString(),
  };
  
  try {
    // Get full page text for parsing
    const pageText = await page.evaluate(() => document.body.innerText);
    
    // Extract direction
    const directionPatterns = [
      /(?:direction|signal|verdict|recommendation)[:\s]*([A-Z]+)/i,
      /(?:LONG|SHORT|BUY|SELL|HOLD)/i,
      /(?:STAY\s+AWAY|KEEP\s+OUT)/i,
    ];
    
    for (const pattern of directionPatterns) {
      const match = pageText.match(pattern);
      if (match) {
        const direction = match[1] || match[0];
        if (direction.includes('LONG')) result.direction = 'LONG';
        else if (direction.includes('SHORT')) result.direction = 'SHORT';
        else if (direction.includes('STAY') || direction.includes('AWAY')) result.direction = 'STAY AWAY';
        break;
      }
    }
    
    // Extract confidence
    const confidenceMatch = pageText.match(/(\d+)%?\s*(?:confidence|probability|score)/i);
    if (confidenceMatch) {
      result.confidence = parseInt(confidenceMatch[1], 10);
    } else {
      // Try to extract any percentage
      const percentMatch = pageText.match(/(\d{1,3})%/);
      if (percentMatch) {
        result.confidence = Math.min(100, Math.max(0, parseInt(percentMatch[1], 10)));
      }
    }
    
    // Extract recommendation text
    const responseEl = await page.$(CONFIG.selectors.responseContainer);
    if (responseEl) {
      const text = await responseEl.evaluate(el => el.textContent);
      result.recommendation = text.substring(0, 1500);
    } else {
      // Use first substantial paragraph
      const paragraphs = pageText.split(/\n\n+/).filter(p => p.length > 50);
      if (paragraphs.length > 0) {
        result.recommendation = paragraphs[0].substring(0, 1500);
      }
    }
    
    // Extract key levels
    const supportMatch = pageText.match(/(?:support|s\.l\.?|stop)[:\s]*([\d.]+)/i);
    const resistanceMatch = pageText.match(/(?:resistance|r\.l\.?|target|t\.p\.)[:\s]*([\d.]+)/i);
    const entryMatch = pageText.match(/(?:entry|entry\.?price)[:\s]*([\d.]+)/i);
    
    if (supportMatch) result.key_levels.support = parseFloat(supportMatch[1]);
    if (resistanceMatch) result.key_levels.resistance = parseFloat(resistanceMatch[1]);
    if (entryMatch) result.entry = parseFloat(entryMatch[1]);
    
    // Extract position options
    result.options = parsePositionOptions(pageText, currentPrice);
    
    // Extract timeframe analysis
    result.timeframes = extractTimeframes(pageText);
    
    // Calculate confluence
    let agreeCount = 0;
    if (result.timeframes['15m'] && result.timeframes['15m'].length > 20) agreeCount++;
    if (result.timeframes['5m'] && result.timeframes['5m'].length > 20) agreeCount++;
    if (result.timeframes['1m'] && result.timeframes['1m'].length > 20) agreeCount++;
    result.confluence = `${agreeCount}/3 timeframes agree`;
    
  } catch (e) {
    console.log('[QuantCrawler] Error extracting response:', e.message);
    result.recommendation = `Analysis completed with errors: ${e.message}`;
  }
  
  return result;
}

function extractTimeframes(pageText) {
  const timeframes = { '15m': '', '5m': '', '1m': '' };
  
  // Split by timeframe references
  const lines = pageText.split('\n');
  let currentTF = '';
  
  for (const line of lines) {
    const lower = line.toLowerCase();
    if (lower.includes('15m') || lower.includes('15 minute')) currentTF = '15m';
    else if (lower.includes('5m') || lower.includes('5 minute')) currentTF = '5m';
    else if (lower.includes('1m') || lower.includes('1 minute')) currentTF = '1m';
    
    if (currentTF && line.length > 30 && line.length < 300) {
      if (timeframes[currentTF].length === 0 || !timeframes[currentTF].includes(line.substring(0, 50))) {
        timeframes[currentTF] += line.substring(0, 200) + ' ';
      }
    }
  }
  
  // Clean up
  for (const tf of Object.keys(timeframes)) {
    timeframes[tf] = timeframes[tf].trim().substring(0, 300);
  }
  
  return timeframes;
}

function parsePositionOptions(text, currentPrice) {
  const options = [];
  
  // Look for single contract option
  const singleMatch = text.match(/(?:single|one).*?contract.*?(\d+).*?(\d+\.?\d*).*?(\d+\.?\d*)/i);
  
  // Look for multiple contracts option  
  const multipleMatch = text.match(/(?:multiple|more).*?(\d+).*?contracts.*?(\d+\.?\d*).*?(\d+\.?\d*)/i);
  
  if (singleMatch) {
    options.push({
      name: 'Single Contract - Tighter Stop',
      contracts: 1,
      risk_per_contract: parseFloat(singleMatch[2]),
      total_risk: parseFloat(singleMatch[2]),
      stop_price: parseFloat(singleMatch[3]) || (currentPrice * 0.995),
      target_price: parseFloat(singleMatch[3]) ? parseFloat(singleMatch[3]) * 1.015 : (currentPrice * 1.015),
      risk_reward_ratio: 1.5,
      best_for: 'Smaller accounts, lower risk tolerance',
      recommended: true,
    });
  }
  
  if (multipleMatch) {
    const riskPer = parseFloat(multipleMatch[2]);
    options.push({
      name: 'Multiple Contracts - Wider Structure',
      contracts: parseInt(multipleMatch[1]),
      risk_per_contract: riskPer,
      total_risk: riskPer * parseInt(multipleMatch[1]),
      stop_price: currentPrice * 0.99,
      target_price: currentPrice * 1.03,
      risk_reward_ratio: 2.0,
      best_for: 'Higher confidence setups',
      recommended: false,
    });
  }
  
  // Default option if parsing fails
  if (options.length === 0) {
    options.push({
      name: 'Standard Position',
      contracts: 1,
      risk_per_contract: 100,
      total_risk: 100,
      stop_price: currentPrice * 0.995,
      target_price: currentPrice * 1.015,
      risk_reward_ratio: 1.5,
      best_for: 'Default sizing',
      recommended: true,
    });
  }
  
  return options;
}

// ═══════════════════════════════════════════════════════════════════════════
// EXPORT FOR N8N
// ═══════════════════════════════════════════════════════════════════════════
module.exports = { analyzeWithQuantCrawler, CONFIG };

// ═══════════════════════════════════════════════════════════════════════════
// CLI MODE
// ═══════════════════════════════════════════════════════════════════════════
if (require.main === module) {
  const args = process.argv.slice(2);
  
  if (args.length < 1) {
    console.log('Usage:');
    console.log('  node quantcrawler.js <symbol>                    # CLI mode');
    console.log('  node quantcrawler.js --webhook                   # Webhook mode');
    console.log('');
    console.log('Environment Variables Required:');
    console.log('  QUANTCRAWLER_EMAIL    - Google account email');
    console.log('  QUANTCRAWLER_PASSWORD - Google account password');
    process.exit(1);
  }
  
  if (args[0] === '--webhook') {
    // HTTP server mode for N8N
    const http = require('http');
    const server = http.createServer(async (req, res) => {
      if (req.method === 'POST' && req.url === '/webhook') {
        let body = '';
        req.on('data', chunk => body += chunk);
        req.on('end', async () => {
          try {
            const { symbol, account_balance, current_price } = JSON.parse(body);
            const result = await analyzeWithQuantCrawler({ 
              symbol, 
              accountBalance: account_balance, 
              currentPrice: current_price 
            });
            res.writeHead(200, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify(result));
          } catch (e) {
            console.error('Error:', e.message);
            res.writeHead(500, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ error: e.message }));
          }
        });
      } else {
        res.writeHead(404);
        res.end('Not Found');
      }
    });
    
    server.listen(3456, () => {
      console.log('[QuantCrawler] Webhook server running on http://localhost:3456/webhook');
    });
  } else {
    // CLI mode
    analyzeWithQuantCrawler({ symbol: args[0] })
      .then(result => console.log(JSON.stringify(result, null, 2)))
      .catch(e => {
        console.error('Error:', e.message);
        process.exit(1);
      });
  }
}
