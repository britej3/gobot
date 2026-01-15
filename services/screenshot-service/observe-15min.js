#!/usr/bin/env node

/**
 * GOBOT 15-Minute Observation Cycle
 * - Quick analysis of all symbols
 * - Optimization recommendations
 * - Metrics export
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const CONFIG = {
  symbols: ['BTCUSDT', 'ETHUSDT', 'XRPUSDT'],
  initialBalance: 100,
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

function runAnalysisSync(symbol) {
  try {
    const scriptDir = path.dirname(__filename);
    const fullResult = execSync(
      `OPENROUTER_API_KEY=${process.env.OPENROUTER_API_KEY} GROQ_API_KEY=${process.env.GROQ_API_KEY} node ai-analyzer.js ${symbol} ${CONFIG.initialBalance}`,
      { encoding: 'utf8', timeout: 90000, cwd: scriptDir }
    );
    return fullResult;
  } catch (e) {
    return e.stdout || `Error: ${e.message}`;
  }
}

async function runObservation() {
  logSection('GOBOT 15-MINUTE OBSERVATION CYCLE');
  log(`Started: ${new Date().toISOString().split('T')[1].split('.')[0]}`, 'cyan');

  const results = {};
  const startTime = Date.now();

  for (const symbol of CONFIG.symbols) {
    log(`\n📊 Analyzing ${symbol}...`, 'blue');
    const result = runAnalysisSync(symbol);
    results[symbol] = result;

    try {
      const lines = result.split('\n');
      const startIdx = lines.findIndex(l => l.trim().startsWith('{'));
      if (startIdx >= 0) {
        const jsonText = lines.slice(startIdx).join('\n');
        const parsed = JSON.parse(jsonText);

        const action = parsed.action || 'N/A';
        const confidence = parsed.confidence || 0;
        const reasoning = parsed.reasoning || '';
        const source = parsed.source || 'N/A';

        const color = action === 'LONG' ? 'green' : action === 'SHORT' ? 'red' : 'yellow';
        log(`  → ${action} @ ${(confidence * 100).toFixed(0)}%`, color);
        log(`  └─ ${reasoning.substring(0, 50)}...`, 'cyan');
        log(`  └─ Source: ${source}`, 'blue');
      } else {
        log(`  └─ Could not parse response`, 'yellow');
      }
    } catch (e) {
      log(`  └─ Parse error: ${e.message.substring(0, 30)}`, 'red');
    }

    require('child_process').execSync('sleep 3', { encoding: 'utf8' });
  }

  const duration = Math.round((Date.now() - startTime) / 1000);

  logSection('OBSERVATION RESULTS');
  log(`Duration: ${duration}s`, 'cyan');

  for (const [symbol, result] of Object.entries(results)) {
    try {
      const lines = result.split('\n');
      const startIdx = lines.findIndex(l => l.trim().startsWith('{'));
      if (startIdx >= 0) {
        const parsed = JSON.parse(lines.slice(startIdx).join('\n'));
        const action = parsed.action || 'N/A';
        const confidence = (parsed.confidence || 0) * 100;
        const color = action === 'LONG' ? 'green' : action === 'SHORT' ? 'red' : 'yellow';
        log(`  ${symbol}: ${action} (${confidence.toFixed(0)}%)`, color);
      }
    } catch (e) {}
  }

  logSection('OPTIMIZATION RECOMMENDATIONS');

  const actions = [];
  for (const [symbol, result] of Object.entries(results)) {
    try {
      const lines = result.split('\n');
      const startIdx = lines.findIndex(l => l.trim().startsWith('{'));
      if (startIdx >= 0) {
        const parsed = JSON.parse(lines.slice(startIdx).join('\n'));
        actions.push({ symbol, action: parsed.action, confidence: parsed.confidence });
      }
    } catch (e) {}
  }

  const longCount = actions.filter(a => a.action === 'LONG').length;
  const shortCount = actions.filter(a => a.action === 'SHORT').length;
  const holdCount = actions.filter(a => a.action === 'HOLD').length;

  log(`Market Sentiment: ${longCount} LONG | ${shortCount} SHORT | ${holdCount} HOLD`, 'cyan');
  log('');

  if (longCount > shortCount) {
    log('📈 BULLISH BIAS DETECTED', 'green');
    log('  → Consider increasing LONG positions');
    log('  → Reduce SHORT exposure');
  } else if (shortCount > longCount) {
    log('📉 BEARISH BIAS DETECTED', 'red');
    log('  → Consider increasing SHORT positions');
    log('  → Reduce LONG exposure');
  } else {
    log('⚖️ MARKET NEUTRAL', 'yellow');
    log('  → Hold current positions');
    log('  → Wait for clearer signal');
  }

  log('');
  log('Active Optimizations:', 'cyan');
  log('  1. Confidence threshold: 0.75', 'blue');
  log('  2. Position sizing: 10% per trade', 'blue');
  log('  3. Stop-loss: 2% | Take-profit: 4% (2:1 RR)', 'blue');
  log('  4. Rate limiting: 20 req/hr (free tier safe)', 'blue');

  const metrics = {
    timestamp: new Date().toISOString(),
    duration_seconds: duration,
    symbols_analyzed: CONFIG.symbols.length,
    results,
    sentiment: { long: longCount, short: shortCount, hold: holdCount },
    optimizations: ['confidence:0.75', 'position:10%', 'sl:2%', 'tp:4%'],
  };

  fs.writeFileSync(path.join(__dirname, 'observation_report.json'), JSON.stringify(metrics, null, 2));
  log('\n✅ Report saved: observation_report.json', 'green');

  logSection('NEXT STEPS');
  log('  1. Review observation results above', 'cyan');
  log('  2. Apply manual optimizations if needed', 'cyan');
  log('  3. Run next 15-min cycle: node observe-15min.js', 'cyan');
  log('  4. After 4 cycles: Full 60-min test complete', 'cyan');
}

async function main() {
  if (!process.env.OPENROUTER_API_KEY) {
    log('ERROR: OPENROUTER_API_KEY required', 'red');
    log('Set: export OPENROUTER_API_KEY=...', 'yellow');
    process.exit(1);
  }

  try {
    await runObservation();
  } catch (error) {
    log(`Failed: ${error.message}`, 'red');
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { runObservation, runAnalysisSync };
