#!/usr/bin/env node

/**
 * GOBOT Aggressive Trading Mode
 * 
 * Features:
 * - Dynamic position sizing (smaller)
 * - Dynamic averaging (higher)
 * - Trailing stop loss and take profit
 * - One trade at a time
 * - Select highest confidence signal
 * - R:R ratio tiebreaker
 */

const http = require('http');
const https = require('https');

// ═══════════════════════════════════════════════════════════════════════
// AGGRESSIVE MODE CONFIGURATION
// ═══════════════════════════════════════════════════════════════════════

const AGGRESSIVE_CONFIG = {
  // Position sizing (smaller for aggressive)
  basePositionSize: 0.02,      // 2% of capital (conservative base)
  minPositionSize: 0.01,       // 1% minimum
  maxPositionSize: 0.03,       // 3% maximum
  
  // Averaging settings
  maxAveragingLevels: 3,       // Max 3 entries
  averagingIncrement: 0.50,    // Add 50% more on each averaging
  
  // Trailing SL/TP
  trailingSLEnabled: true,
  trailingSLActivation: 1.5,   // Activate after 1.5% gain
  trailingSLDistance: 1.0,     // Trail by 1%
  
  trailingTPEnabled: true,
  trailingTPStep: 0.5,         // Move TP by 0.5% increments
  
  // One trade at a time
  maxConcurrentTrades: 1,
  
  // Selection criteria
  minConfidence: 0.70,         // Only trade signals >70%
  minRRRatio: 1.5,             // Minimum 1.5:1 R:R
};

// ═══════════════════════════════════════════════════════════════════════
// DYNAMIC POSITION SIZING
// ═══════════════════════════════════════════════════════════════════════

function calculateDynamicPositionSize(capital, confidence, volatility = 0.02) {
  /**
   * Calculate position size based on:
   * - Capital available
   * - Signal confidence (higher confidence = larger position)
   * - Market volatility (higher volatility = smaller position)
   */
  
  // Base size from confidence
  const confidenceMultiplier = Math.max(confidence - 0.5, 0.5); // 0.5 to 0.5+
  
  // Volatility adjustment (inverse)
  const volatilityMultiplier = Math.max(1 - (volatility * 10), 0.5); // Reduce for high vol
  
  // Calculate position
  let positionPct = AGGRESSIVE_CONFIG.basePositionSize * confidenceMultiplier * volatilityMultiplier;
  
  // Clamp to limits
  positionPct = Math.max(positionPct, AGGRESSIVE_CONFIG.minPositionSize);
  positionPct = Math.min(positionPct, AGGRESSIVE_CONFIG.maxPositionSize);
  
  const positionSize = capital * positionPct;
  
  return {
    positionPct: (positionPct * 100).toFixed(2) + '%',
    positionSize: positionSize.toFixed(2),
    confidenceMultiplier: confidenceMultiplier.toFixed(2),
    volatilityMultiplier: volatilityMultiplier.toFixed(2),
  };
}

// ═══════════════════════════════════════════════════════════════════════
// DYNAMIC AVERAGING SYSTEM
// ═══════════════════════════════════════════════════════════════════════

function calculateAveragingLevels(entryPrice, direction, confidence) {
  /**
   * Calculate dynamic averaging levels
   * Higher confidence = fewer averaging levels
   */
  
  const levels = [];
  const maxLevels = AGGRESSIVE_CONFIG.maxAveragingLevels;
  const increment = AGGRESSIVE_CONFIG.averagingIncrement;
  
  // Adjust levels based on confidence
  const actualLevels = confidence > 0.85 ? 2 : confidence > 0.75 ? 3 : maxLevels;
  
  for (let i = 1; i <= actualLevels; i++) {
    const priceMove = entryPrice * (0.01 * i * increment); // 0.5%, 1%, 1.5% moves
    
    let levelPrice;
    let action;
    
    if (direction === 'LONG') {
      levelPrice = entryPrice - priceMove; // Add below on dips
      action = 'ADD';
    } else {
      levelPrice = entryPrice + priceMove; // Add above on pumps
      action = 'ADD';
    }
    
    // Position size increases for each level
    const positionPct = (AGGRESSIVE_CONFIG.basePositionSize * (1 + increment * i) * 0.5).toFixed(2);
    
    levels.push({
      level: i,
      action,
      price: levelPrice.toFixed(8),
      positionPct: positionPct + '%',
    });
  }
  
  return levels;
}

// ═══════════════════════════════════════════════════════════════════════
// TRAILING STOP LOSS & TAKE PROFIT
// ═══════════════════════════════════════════════════════════════════════

function calculateTrailingSLTP(entryPrice, direction, confidence) {
  /**
   * Calculate trailing SL and TP levels
   */
  
  const stopDistance = entryPrice * (AGGRESSIVE_CONFIG.trailingSLDistance / 100);
  const tpStep = entryPrice * (AGGRESSIVE_CONFIG.trailingTPStep / 100);
  
  let initialSL, initialTP, trailingSL, trailingTP;
  
  if (direction === 'LONG') {
    initialSL = entryPrice * (1 - AGGRESSIVE_CONFIG.trailingSLActivation / 100);
    initialTP = entryPrice * (1 + AGGRESSIVE_CONFIG.trailingSLActivation / 100);
    
    trailingSL = entryPrice - stopDistance;
    trailingTP = entryPrice + tpStep;
  } else {
    initialSL = entryPrice * (1 + AGGRESSIVE_CONFIG.trailingSLActivation / 100);
    initialTP = entryPrice * (1 - AGGRESSIVE_CONFIG.trailingSLActivation / 100);
    
    trailingSL = entryPrice + stopDistance;
    trailingTP = entryPrice - tpStep;
  }
  
  return {
    initial: {
      stop: initialSL.toFixed(8),
      target: initialTP.toFixed(8),
    },
    trailing: {
      stop: trailingSL.toFixed(8),
      target: trailingTP.toFixed(8),
      activation: AGGRESSIVE_CONFIG.trailingSLActivation + '%',
      trailDistance: (AGGRESSIVE_CONFIG.trailingSLDistance) + '%',
    },
  };
}

// ═══════════════════════════════════════════════════════════════════════
// SIGNAL SELECTION (Highest Confidence + R:R Tiebreaker)
// ═══════════════════════════════════════════════════════════════════════

function selectBestSignal(signals) {
  /**
   * Select the best signal from multiple candidates
   * Criteria:
   * 1. Highest confidence
   * 2. If tie, highest R:R ratio
   */
  
  if (!signals || signals.length === 0) {
    return null;
  }
  
  if (signals.length === 1) {
    return signals[0];
  }
  
  // Filter by minimum confidence
  const validSignals = signals.filter(s => s.confidence >= AGGRESSIVE_CONFIG.minConfidence);
  
  if (validSignals.length === 0) {
    return null;
  }
  
  // Sort by confidence (descending)
  validSignals.sort((a, b) => b.confidence - a.confidence);
  
  // Get highest confidence
  const highestConfidence = validSignals[0].confidence;
  
  // Get all signals with highest confidence
  const topSignals = validSignals.filter(s => s.confidence === highestConfidence);
  
  // If tie, use R:R ratio
  if (topSignals.length > 1) {
    topSignals.sort((a, b) => (b.risk_reward || 2) - (a.risk_reward || 2));
  }
  
  const selected = topSignals[0];
  
  // Mark as selected
  selected.selectionReason = 'highest_confidence';
  if (topSignals.length > 1) {
    selected.selectionReason = 'tiebreaker_rr';
  }
  
  return selected;
}

// ═══════════════════════════════════════════════════════════════════════
// TRADE EXECUTION STATE
// ═══════════════════════════════════════════════════════════════════════

class AggressiveTradeManager {
  constructor() {
    this.activeTrade = null;
    this.tradeHistory = [];
    this.entryOrders = [];
    this.currentPositionSize = 0;
  }
  
  canTrade() {
    return this.activeTrade === null;
  }
  
  openTrade(signal, capital) {
    if (!this.canTrade()) {
      return { success: false, reason: 'Trade already in progress' };
    }
    
    // Calculate position size
    const position = calculateDynamicPositionSize(
      capital, 
      signal.confidence,
      signal.volatility || 0.02
    );
    
    // Calculate averaging levels
    const averaging = calculateAveragingLevels(
      parseFloat(signal.entry_price),
      signal.action.toLowerCase(),
      signal.confidence
    );
    
    // Calculate trailing SL/TP
    const trailing = calculateTrailingSLTP(
      parseFloat(signal.entry_price),
      signal.action.toLowerCase(),
      signal.confidence
    );
    
    // Create trade
    this.activeTrade = {
      symbol: signal.symbol,
      action: signal.action,
      entryPrice: parseFloat(signal.entry_price),
      confidence: signal.confidence,
      riskReward: signal.risk_reward || 2,
      selectionReason: signal.selectionReason,
      position: position,
      averaging: averaging,
      trailing: trailing,
      status: 'OPEN',
      entryTime: new Date().toISOString(),
      entries: [],
      partialExits: [],
    };
    
    return {
      success: true,
      trade: this.activeTrade,
    };
  }
  
  addEntry(price, quantity) {
    if (!this.activeTrade) {
      return { success: false, reason: 'No active trade' };
    }
    
    this.activeTrade.entries.push({
      price,
      quantity,
      time: new Date().toISOString(),
    });
    
    this.currentPositionSize += quantity;
    
    return { success: true };
  }
  
  updateTrailing(currentPrice) {
    if (!this.activeTrade || !AGGRESSIVE_CONFIG.trailingSLEnabled) {
      return null;
    }
    
    const trade = this.activeTrade;
    const direction = trade.action.toLowerCase();
    
    // Check if trailing should activate
    const gain = direction === 'LONG'
      ? (currentPrice - trade.entryPrice) / trade.entryPrice * 100
      : (trade.entryPrice - currentPrice) / trade.entryPrice * 100;
    
    if (gain >= AGGRESSIVE_CONFIG.trailingSLActivation) {
      // Update trailing levels
      const distance = currentPrice * (AGGRESSIVE_CONFIG.trailingSLDistance / 100);
      
      if (direction === 'LONG') {
        trade.trailing.currentSL = Math.max(
          trade.trailing.currentSL || trade.trailing.initial.stop,
          currentPrice - distance
        );
        trade.trailing.currentTP = trade.trailing.initial.target + 
          (currentPrice - trade.entryPrice) * 0.5; // Move TP half of gain
      } else {
        trade.trailing.currentSL = Math.min(
          trade.trailing.currentSL || trade.trailing.initial.stop,
          currentPrice + distance
        );
        trade.trailing.currentTP = trade.trailing.initial.target - 
          (trade.entryPrice - currentPrice) * 0.5;
      }
      
      trade.trailing.lastUpdate = new Date().toISOString();
      
      return {
        sl: trade.trailing.currentSL.toFixed(8),
        tp: trade.trailing.currentTP.toFixed(8),
      };
    }
    
    return null;
  }
  
  closeTrade(reason, exitPrice) {
    if (!this.activeTrade) {
      return { success: false, reason: 'No active trade' };
    }
    
    const trade = this.activeTrade;
    
    // Calculate P&L
    const avgEntry = trade.entries.reduce((sum, e) => sum + e.price * e.quantity, 0) /
                     trade.entries.reduce((sum, e) => sum + e.quantity, 0);
    
    let pnl, pnlPct;
    if (trade.action === 'LONG') {
      pnl = (exitPrice - avgEntry) * this.currentPositionSize;
      pnlPct = (exitPrice - avgEntry) / avgEntry * 100;
    } else {
      pnl = (avgEntry - exitPrice) * this.currentPositionSize;
      pnlPct = (avgEntry - exitPrice) / avgEntry * 100;
    }
    
    // Record trade
    const closedTrade = {
      ...trade,
      status: 'CLOSED',
      closeReason: reason,
      exitPrice,
      exitTime: new Date().toISOString(),
      avgEntry: avgEntry.toFixed(8),
      exitPrice: exitPrice.toFixed(8),
      pnl: pnl.toFixed(2),
      pnlPct: pnlPct.toFixed(2),
      duration: new Date(trade.entryTime) - new Date(trade.exitTime),
    };
    
    this.tradeHistory.push(closedTrade);
    this.activeTrade = null;
    this.currentPositionSize = 0;
    
    return {
      success: true,
      trade: closedTrade,
    };
  }
  
  getStats() {
    const wins = this.tradeHistory.filter(t => parseFloat(t.pnl) > 0).length;
    const losses = this.tradeHistory.filter(t => parseFloat(t.pnl) <= 0).length;
    const total = this.tradeHistory.length;
    
    const totalPnl = this.tradeHistory.reduce((sum, t) => sum + parseFloat(t.pnl), 0);
    const avgPnl = total > 0 ? totalPnl / total : 0;
    
    const winRate = total > 0 ? (wins / total * 100).toFixed(1) : 0;
    
    return {
      totalTrades: total,
      wins,
      losses,
      winRate,
      totalPnl: totalPnl.toFixed(2),
      avgPnl: avgPnl.toFixed(2),
      activeTrade: this.activeTrade ? 'YES' : 'NO',
    };
  }
}

// ═══════════════════════════════════════════════════════════════════════
// MAIN AGGRESSIVE TRADING FUNCTION
// ═══════════════════════════════════════════════════════════════════════

async function runAggressiveTradingMode(symbols, capital) {
  const manager = new AggressiveTradeManager();
  
  console.log('\n╔════════════════════════════════════════════════════════════════════╗');
  console.log('║       GOBOT AGGRESSIVE TRADING MODE                          ║');
  console.log('╚════════════════════════════════════════════════════════════════════╝\n');
  
  console.log('Configuration:');
  console.log('  Position Size: 1-3% (dynamic based on confidence)');
  console.log('  Averaging: Up to 3 levels');
  console.log('  Trailing SL/TP: Enabled');
  console.log('  One trade at a time');
  console.log('  Selection: Highest confidence + R:R tiebreaker');
  console.log('');
  
  // Fetch signals for all symbols
  console.log('Fetching signals for all symbols...');
  
  // In production, this would call QuantCrawler for each symbol
  const signals = await fetchSignalsForSymbols(symbols);
  
  console.log(`Received ${signals.length} signals\n`);
  
  // Select best signal
  const selectedSignal = selectBestSignal(signals);
  
  if (!selectedSignal) {
    console.log('❌ No valid signals (confidence < 70% or no signals)');
    return manager.getStats();
  }
  
  console.log('✅ Selected Signal:');
  console.log(`   Symbol:    ${selectedSignal.symbol}`);
  console.log(`   Action:    ${selectedSignal.action}`);
  console.log(`   Confidence: ${(selectedSignal.confidence * 100).toFixed(0)}%`);
  console.log(`   Entry:     ${selectedSignal.entry_price}`);
  console.log(`   Stop:      ${selectedSignal.stop_loss}`);
  console.log(`   Target:    ${selectedSignal.take_profit}`);
  console.log(`   R:R:       ${selectedSignal.risk_reward || 2}:1`);
  console.log(`   Reason:    ${selectedSignal.selectionReason}`);
  console.log('');
  
  // Calculate position size
  const position = calculateDynamicPositionSize(
    capital,
    selectedSignal.confidence,
    selectedSignal.volatility || 0.02
  );
  
  console.log('Position Sizing:');
  console.log(`   Size:      ${position.positionSize} (${position.positionPct})`);
  console.log(`   Confidence Adj: ${position.confidenceMultiplier}x`);
  console.log(`   Volatility Adj: ${position.volatilityMultiplier}x`);
  console.log('');
  
  // Calculate averaging levels
  const averaging = calculateAveragingLevels(
    parseFloat(selectedSignal.entry_price),
    selectedSignal.action.toLowerCase(),
    selectedSignal.confidence
  );
  
  console.log('Averaging Levels:');
  averaging.forEach(level => {
    console.log(`   Level ${level.level}: ${level.action} @ ${level.price} (${level.positionPct})`);
  });
  console.log('');
  
  // Calculate trailing SL/TP
  const trailing = calculateTrailingSLTP(
    parseFloat(selectedSignal.entry_price),
    selectedSignal.action.toLowerCase(),
    selectedSignal.confidence
  );
  
  console.log('Trailing SL/TP:');
  console.log(`   Initial SL: ${trailing.initial.stop}`);
  console.log(`   Initial TP: ${trailing.initial.target}`);
  console.log(`   Activation: ${trailing.trailing.activation} gain`);
  console.log(`   Trail: ${trailing.trailing.trailDistance}`);
  console.log('');
  
  // Open trade
  const openResult = manager.openTrade(selectedSignal, capital);
  
  if (openResult.success) {
    console.log('✅ Trade opened successfully');
    console.log(`   Status: ${manager.activeTrade.status}`);
    console.log('');
    
    // Simulate trade monitoring (in production, this would be real-time)
    console.log('Trade Management:');
    console.log('   Monitoring for trailing SL/TP updates...');
    console.log('   Waiting for exit signal...\n');
  } else {
    console.log(`❌ Failed to open trade: ${openResult.reason}`);
  }
  
  return manager.getStats();
}

// Mock function to fetch signals (replace with real QuantCrawler calls)
async function fetchSignalsForSymbols(symbols) {
  const signals = [];
  
  for (const symbol of symbols) {
    // Simulate signal generation
    const confidence = 0.65 + Math.random() * 0.30;
    const roll = Math.random();
    
    let action;
    if (roll < 0.25) {
      action = 'HOLD';
    } else if (roll < 0.60) {
      action = 'LONG';
    } else {
      action = 'SHORT';
    }
    
    const entry = 0.00001 + Math.random() * 0.00001;
    
    signals.push({
      symbol,
      action,
      confidence,
      entry_price: entry.toFixed(8),
      stop_loss: (entry * 0.98).toFixed(8),
      take_profit: (entry * 1.04).toFixed(8),
      risk_reward: 2,
      volatility: 0.02,
    });
  }
  
  return signals;
}

// Export
module.exports = {
  AGGRESSIVE_CONFIG,
  calculateDynamicPositionSize,
  calculateAveragingLevels,
  calculateTrailingSLTP,
  selectBestSignal,
  AggressiveTradeManager,
  runAggressiveTradingMode,
};

// CLI
if (require.main === module) {
  const symbols = process.argv.slice(2) || ['1000PEPEUSDT', '1000BONKUSDT', '1000FLOKIUSDT'];
  const capital = 5000;
  
  runAggressiveTradingMode(symbols, capital).then(stats => {
    console.log('\n╔════════════════════════════════════════════════════════════════════╗');
    console.log('║                      STATISTICS                              ║');
    console.log('╚════════════════════════════════════════════════════════════════════╝\n');
    
    console.log(`  Total Trades:  ${stats.totalTrades}`);
    console.log(`  Wins:          ${stats.wins}`);
    console.log(`  Losses:        ${stats.losses}`);
    console.log(`  Win Rate:      ${stats.winRate}%`);
    console.log(`  Total P&L:     $${stats.totalPnl}`);
    console.log(`  Avg P&L:       $${stats.avgPnl}`);
    console.log(`  Active Trade:  ${stats.activeTrade}`);
  });
}
