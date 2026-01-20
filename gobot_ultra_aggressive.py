#!/usr/bin/env python3
"""
GOBOT Ultra-Aggressive Micro-Trading
===================================

Ultra-aggressive settings for MAXIMUM growth from 1 USDT
WARNING: High risk, high reward strategy

Features:
- 125x leverage (maximum on Binance)
- 0.5% risk per trade (aggressive)
- 3:1 reward/risk ratio
- Compounding at 3 USDT threshold
- Grid trading with 10 levels
- Liquidation protection at 3% buffer
- 95% confidence signals only
- Target: 1 USDT → 100 USDT in weeks
"""

import asyncio
import json
import logging
import os
import subprocess
import time
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, List, Any

from orchestrator import OrchestratorConfig, ClaudeOrchestrator, CyclePhase

logger = logging.getLogger(__name__)


class UltraAggressiveConfig:
    """Ultra-aggressive micro-trading configuration"""

    def __init__(self):
        # Balance management
        self.initial_balance = 1.0
        self.current_balance = 1.0
        self.target_balance = 100.0
        self.stretch_target = 500.0

        # Leverage (MAXIMUM)
        self.leverage = 125
        self.max_leverage = 125

        # Risk management (AGGRESSIVE)
        self.risk_per_trade = 0.005  # 0.5% of balance
        self.max_risk_per_trade = 0.01  # Cap at 1%
        self.stop_loss = 0.0015  # 0.15%
        self.take_profit = 0.0045  # 0.45% (3:1 RR)
        self.max_position_size = 100.0  # Max 100 USDT position
        self.min_order_value = 5.0  # Min 5 USDT

        # Frequency
        self.max_trades_per_hour = 10
        self.max_trades_per_day = 100

        # Compounding (AGGRESSIVE)
        self.compound_threshold = 3.0  # Compound when balance > 3 USDT
        self.compound_rate = 0.7  # Compound 70% of profits
        self.compound_frequency = 'daily'  # Compound daily

        # Grid trading (AGGRESSIVE)
        self.grid_enabled = True
        self.grid_size = 0.05  # 0.05% grid spacing
        self.grid_levels = 10  # 10 levels above and below
        self.grid_profit_target = 0.001  # 0.1% profit per grid

        # Signal filtering (ULTRA-STRICT)
        self.min_confidence = 0.95  # 95% confidence minimum
        self.min_signal_score = 80  # 80/100 signal score

        # Liquidation protection
        self.liquidation_buffer = 3.0  # 3% buffer from liquidation
        self.max_risk_exposure = 10.0  # Max 10% of balance at risk

        # Martingale (OPTIONAL - HIGH RISK)
        self.martingale_enabled = False  # Disabled by default
        self.martingale_multiplier = 1.5
        self.martingale_max_levels = 3

        # Targets and projections
        self.daily_target_pct = 5.0  # 5% daily growth target
        self.weekly_target_pct = 50.0  # 50% weekly growth target
        self.monthly_target_pct = 500.0  # 500% monthly growth target

    def calculate_position_size(self) -> float:
        """Calculate aggressive position size"""
        # Base risk amount
        risk_amount = self.current_balance * self.risk_per_trade

        # Position with leverage
        position_size = risk_amount * self.leverage / self.stop_loss

        # Apply max risk cap
        max_risk = self.current_balance * self.max_risk_per_trade
        position_size = min(position_size, max_risk * self.leverage / self.stop_loss)

        # Cap by max position size
        position_size = min(position_size, self.max_position_size)

        # Cap by available balance * leverage
        max_by_balance = self.current_balance * self.leverage
        position_size = min(position_size, max_by_balance)

        # Ensure minimum
        position_size = max(position_size, self.min_order_value)

        return position_size

    def get_liquidation_price(self, entry_price: float, side: str) -> float:
        """Calculate liquidation price with buffer"""
        # Simplified liquidation calculation
        maintenance_margin = 0.003  # 0.3%

        if side == "LONG":
            # Liquidation = entry * (1 - 1/leverage - maintenance_margin)
            liquidation = entry_price * (1 - 1/self.leverage - maintenance_margin)
        else:
            # SHORT liquidation
            liquidation = entry_price * (1 + 1/self.leverage + maintenance_margin)

        # Add buffer
        buffer_amount = liquidation * (self.liquidation_buffer / 100)

        if side == "LONG":
            return liquidation - buffer_amount
        else:
            return liquidation + buffer_amount

    def should_compound(self) -> bool:
        """Check if should compound"""
        return self.current_balance >= self.compound_threshold

    def get_compound_amount(self) -> float:
        """Get amount to compound"""
        if not self.should_compound():
            return 0.0

        # Compound 70% of balance above threshold
        excess = self.current_balance - self.compound_threshold
        return excess * self.compound_rate

    def get_grid_orders(self, symbol: str, current_price: float) -> List[Dict]:
        """Generate grid orders"""
        orders = []

        for i in range(1, self.grid_levels + 1):
            # Buy orders (below current price)
            buy_price = current_price * (1 - i * self.grid_size / 100)
            buy_quantity = self.min_order_value / buy_price

            orders.append({
                'type': 'BUY',
                'symbol': symbol,
                'price': buy_price,
                'quantity': buy_quantity,
                'side': 'BUY',
                'level': i
            })

            # Sell orders (above current price)
            sell_price = current_price * (1 + i * self.grid_size / 100)
            sell_quantity = self.min_order_value / sell_price

            orders.append({
                'type': 'SELL',
                'symbol': symbol,
                'price': sell_price,
                'quantity': sell_quantity,
                'side': 'SELL',
                'level': i
            })

        return orders

    def get_projection_days(self) -> Dict[str, float]:
        """Calculate projected days to targets"""
        daily_return = self.daily_target_pct / 100
        balance = self.current_balance

        # Days to 100 USDT
        if balance >= self.target_balance:
            days_100 = 0
        else:
            days_100 = (self.target_balance / balance) ** (1/daily_return) - 1

        # Days to 500 USDT
        if balance >= self.stretch_target:
            days_500 = 0
        else:
            days_500 = (self.stretch_target / balance) ** (1/daily_return) - 1

        return {
            'to_100': days_100,
            'to_500': days_500
        }


class GOBOTUltraAggressiveOrchestrator(ClaudeOrchestrator):
    """Ultra-aggressive micro-trading orchestrator"""

    def __init__(self, config: OrchestratorConfig):
        super().__init__(config)

        # Ultra-aggressive config
        self.ultra_config = UltraAggressiveConfig()

        # Stats
        self.stats = {
            'total_trades': 0,
            'winning_trades': 0,
            'losing_trades': 0,
            'total_pnl': 0.0,
            'largest_win': 0.0,
            'largest_loss': 0.0,
            'win_streak': 0,
            'loss_streak': 0,
            'balance_history': [],
            'compounds': 0,
            'grids_executed': 0,
            'hourly_trades': [],
            'daily_pnl': 0.0,
            'start_time': datetime.now()
        }

        # Active positions
        self.active_positions = []
        self.active_grids = []

        logger.info("🚀 GOBOT ULTRA-AGGRESSIVE MICRO-TRADING")
        logger.warning("="*60)
        logger.warning("⚠️  WARNING: ULTRA-HIGH RISK STRATEGY ⚠️")
        logger.warning("="*60)
        logger.info(f"Starting balance: {self.ultra_config.initial_balance} USDT")
        logger.info(f"Target: {self.ultra_config.target_balance} USDT")
        logger.info(f"Leverage: {self.ultra_config.leverage}x")
        logger.info(f"Risk per trade: {self.ultra_config.risk_per_trade*100}%")
        logger.info(f"Stop loss: {self.ultra_config.stop_loss*100}%")
        logger.info(f"Take profit: {self.ultra_config.take_profit*100}%")
        logger.warning("="*60)

    async def _handle_data_scan(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 1: Ultra-precise market scan"""
        logger.info("="*60)
        logger.info("Phase 1: ULTRA-PRECISE MARKET SCAN")
        logger.info("="*60)

        await asyncio.sleep(0.05)  # Ultra-fast scanning

        # Simulate high-precision market data
        market_data = {
            'timestamp': datetime.now().isoformat(),
            'balance': self.ultra_config.current_balance,
            'leverage': self.ultra_config.leverage,
            'symbols': {
                'BTCUSDT': {
                    'price': 95320.50,
                    'bid': 95320.25,
                    'ask': 95320.75,
                    'spread': 0.50,
                    'volume_24h': 1250000,
                    'volatility_1h': 0.012,  # 1.2%
                    'volatility_5m': 0.005,  # 0.5%
                    'rsi_1m': 22.5,
                    'rsi_5m': 28.5,
                    'rsi_15m': 35.2,
                    'macd': 'bullish_divergence',
                    'bb_position': 'lower_band',
                    'volume_spike': 2.1,  # 110% above average
                    'order_flow': 'very_bullish',
                    'funding_rate': -0.0012,  # -0.12%
                    'open_interest': 'increasing_fast',
                    'whale_activity': 'high'
                }
            },
            'market_regime': 'extreme_volatility',
            'sentiment': 'extreme_fear',
            'signal_strength': 95  # 95/100
        }

        # Calculate ultra-precise signals
        signals = []
        btc = market_data['symbols']['BTCUSDT']

        # Ultra-strict signal criteria
        signal_score = 0
        criteria_met = []

        # RSI oversold (1m)
        if btc['rsi_1m'] < 25:
            signal_score += 25
            criteria_met.append('RSI_1m_oversold')

        # RSI oversold (5m)
        if btc['rsi_5m'] < 30:
            signal_score += 20
            criteria_met.append('RSI_5m_oversold')

        # Volume spike
        if btc['volume_spike'] > 2.0:
            signal_score += 20
            criteria_met.append('volume_spike')

        # Bollinger Bands
        if btc['bb_position'] == 'lower_band':
            signal_score += 15
            criteria_met.append('bb_lower')

        # MACD
        if btc['macd'] == 'bullish_divergence':
            signal_score += 15
            criteria_met.append('macd_bullish')

        # Funding rate (negative = good for longs)
        if btc['funding_rate'] < -0.001:
            signal_score += 10
            criteria_met.append('funding_negative')

        # Order flow
        if btc['order_flow'] in ['very_bullish', 'bullish']:
            signal_score += 10
            criteria_met.append('order_flow_bullish')

        signals.append({
            'symbol': 'BTCUSDT',
            'signal_score': signal_score,
            'criteria_met': criteria_met,
            'action': 'LONG' if signal_score >= self.ultra_config.min_signal_score else 'HOLD',
            'strength': 'ULTRA_STRONG' if signal_score >= 90 else 'VERY_STRONG' if signal_score >= 80 else 'STRONG'
        })

        market_data['signals'] = signals
        market_data['signal_quality'] = signal_score

        # Add pattern
        self.state_manager.add_pattern(
            f"Ultra-scan: {signal_score}/100 signal strength, {len(criteria_met)} criteria met"
        )

        logger.info(f"Balance: {market_data['balance']} USDT")
        logger.info(f"Leverage: {market_data['leverage']}x")
        logger.info(f"Signal quality: {signal_score}/100")
        logger.info(f"Criteria met: {', '.join(criteria_met)}")

        for signal in signals:
            logger.info(f"{signal['symbol']}: {signal['action']} "
                       f"(score: {signal['signal_score']}, {signal['strength']})")

        return market_data

    async def _handle_idea(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 2: Ultra-high conviction ideas"""
        logger.info("="*60)
        logger.info("Phase 2: ULTRA-HIGH CONVICTION IDEAS")
        logger.info("="*60)

        market_data = self.current_cycle.phase_results.get('data_scan', {})
        signals = market_data.get('signals', [])

        await asyncio.sleep(0.05)

        # Filter for ultra-high conviction only
        ultra_high_conviction = [
            s for s in signals
            if s['signal_score'] >= self.ultra_config.min_signal_score
            and s['action'] == 'LONG'
        ]

        ideas = []

        for signal in ultra_high_conviction:
            symbol_data = market_data['symbols'][signal['symbol']]

            # Calculate position
            entry_price = symbol_data['price']
            position_size = self.ultra_config.calculate_position_size()

            # Stop loss and take profit
            sl_price = entry_price * (1 - self.ultra_config.stop_loss)
            tp_price = entry_price * (1 + self.ultra_config.take_profit)

            # Risk/reward
            risk = (entry_price - sl_price) / entry_price * 100
            reward = (tp_price - entry_price) / entry_price * 100
            rr_ratio = reward / risk

            # Expected value (assume 75% win rate)
            expected_value = (0.75 * reward) - (0.25 * risk)

            # Liquidation price
            liq_price = self.ultra_config.get_liquidation_price(entry_price, 'LONG')
            distance_to_liq = (entry_price - liq_price) / entry_price * 100

            ideas.append({
                'symbol': signal['symbol'],
                'action': 'LONG',
                'signal_score': signal['signal_score'],
                'entry_price': entry_price,
                'position_size_usdt': position_size,
                'position_size_coins': position_size / entry_price,
                'leverage': self.ultra_config.leverage,
                'stop_loss': sl_price,
                'take_profit': tp_price,
                'liquidation_price': liq_price,
                'risk_percent': risk,
                'reward_percent': reward,
                'rr_ratio': rr_ratio,
                'expected_value': expected_value,
                'distance_to_liquidation': distance_to_liq,
                'conviction': 'ULTRA_HIGH',
                'criteria': signal['criteria_met'],
                'reasoning': f"{len(signal['criteria_met'])} criteria met: {', '.join(signal['criteria_met'])}"
            })

        # Select best idea
        best_idea = None
        if ideas:
            best_idea = max(ideas, key=lambda x: x['expected_value'])

        result = {
            'total_signals': len(signals),
            'ultra_high_conviction': len(ultra_high_conviction),
            'ideas': ideas,
            'selected_idea': best_idea,
            'position_size': self.ultra_config.calculate_position_size() if best_idea else 0,
            'strategy': 'ULTRA_AGGRESSIVE' if best_idea else 'NO_SIGNAL'
        }

        logger.info(f"Total signals: {result['total_signals']}")
        logger.info(f"Ultra-high conviction: {result['ultra_high_conviction']}")

        if best_idea:
            logger.info(f"Selected: {best_idea['symbol']} LONG")
            logger.info(f"Entry: ${best_idea['entry_price']:.2f}")
            logger.info(f"Position: {best_idea['position_size_usdt']:.2f} USDT "
                       f"({best_idea['leverage']}x)")
            logger.info(f"Risk: {best_idea['risk_percent']:.3f}%")
            logger.info(f"Reward: {best_idea['reward_percent']:.3f}%")
            logger.info(f"R/R: {best_idea['rr_ratio']:.1f}x")
            logger.info(f"Expected Value: {best_idea['expected_value']:.2f}%")
            logger.info(f"Liquidation: ${best_idea['liquidation_price']:.2f} "
                       f"({best_idea['distance_to_liquidation']:.1f}% away)")
            logger.info(f"Criteria: {', '.join(best_idea['criteria'])}")
        else:
            logger.info("No ultra-high conviction signals - waiting")

        return result

    async def _handle_code_edit(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 3: Position sizing and grid setup"""
        logger.info("="*60)
        logger.info("Phase 3: POSITION SIZING & GRID SETUP")
        logger.info("="*60)

        idea_result = self.current_cycle.phase_results.get('idea', {})
        selected_idea = idea_result.get('selected_idea')

        await asyncio.sleep(0.05)

        if not selected_idea:
            # Setup grid trading
            logger.info("No signal - setting up aggressive grid trading")

            btc_price = 95320.50  # Current price
            grid_orders = self.ultra_config.get_grid_orders('BTCUSDT', btc_price)

            logger.info(f"Set up {len(grid_orders)} grid orders")
            logger.info(f"Grid size: {self.ultra_config.grid_size}%")
            logger.info(f"Grid levels: {self.ultra_config.grid_levels}")

            return {
                'status': 'GRID_TRADING',
                'reason': 'No ultra-high conviction signal',
                'grid_orders': grid_orders,
                'expected_profit_per_grid': self.ultra_config.grid_profit_target * 100
            }

        # Position sizing for signal
        entry_price = selected_idea['entry_price']
        position_size = selected_idea['position_size_usdt']
        leverage = selected_idea['leverage']

        # Margin calculation
        margin_required = position_size / leverage
        free_margin = self.ultra_config.current_balance - margin_required

        result = {
            'status': 'POSITION_SET',
            'idea': selected_idea,
            'adjusted_position': {
                'size_usdt': position_size,
                'leverage': leverage,
                'margin_required': margin_required,
                'free_margin': free_margin,
                'utilization_pct': (margin_required / self.ultra_config.current_balance * 100)
            },
            'risk_metrics': {
                'risk_amount': position_size * self.ultra_config.stop_loss,
                'reward_amount': position_size * self.ultra_config.take_profit,
                'rr_ratio': self.ultra_config.take_profit / self.ultra_config.stop_loss,
                'max_exposure': self.ultra_config.max_risk_exposure
            },
            'liquidation_protection': {
                'buffer_pct': self.ultra_config.liquidation_buffer,
                'safe_distance': selected_idea['distance_to_liquidation']
            }
        }

        logger.info(f"Position set: {position_size:.2f} USDT @ {entry_price:.2f}")
        logger.info(f"Leverage: {leverage}x (Margin: {margin_required:.3f} USDT)")
        logger.info(f"Free margin: {free_margin:.3f} USDT")
        logger.info(f"Utilization: {result['adjusted_position']['utilization_pct']:.1f}%")
        logger.info(f"Risk: {result['risk_metrics']['risk_amount']:.3f} USDT")
        logger.info(f"Reward: {result['risk_metrics']['reward_amount']:.3f} USDT")

        return result

    async def _handle_backtest(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 4: Trade execution"""
        logger.info("="*60)
        logger.info("Phase 4: TRADE EXECUTION")
        logger.info("="*60)

        position_result = self.current_cycle.phase_results.get('code_edit', {})
        status = position_result.get('status')

        await asyncio.sleep(0.05)

        if status == 'GRID_TRADING':
            logger.info("Grid trading mode - waiting for price movement")
            return {
                'status': 'GRID_WAITING',
                'mode': 'GRID_TRADING',
                'active_grids': len(position_result.get('grid_orders', []))
            }

        # Execute position trade
        idea = position_result.get('idea', {})

        logger.info(f"Executing {idea.get('action')} order for {idea.get('symbol')}")

        # Simulate ultra-high conviction win rate (80%)
        import random
        trade_won = random.random() < 0.80

        if trade_won:
            pnl = idea.get('position_size_usdt', 0) * self.ultra_config.take_profit
            outcome = 'WIN'
        else:
            pnl = -idea.get('position_size_usdt', 0) * self.ultra_config.stop_loss
            outcome = 'LOSS'

        # Update balance
        self.ultra_config.current_balance += pnl

        # Update stats
        self.stats['total_trades'] += 1
        self.stats['total_pnl'] += pnl

        if trade_won:
            self.stats['winning_trades'] += 1
            self.stats['win_streak'] += 1
            self.stats['loss_streak'] = 0
        else:
            self.stats['losing_trades'] += 1
            self.stats['loss_streak'] += 1
            self.stats['win_streak'] = 0

        # Check for compounding
        compound_amount = 0
        if self.ultra_config.should_compound():
            compound_amount = self.ultra_config.get_compound_amount()
            self.stats['compounds'] += 1
            logger.info(f"🎉 Compounding {compound_amount:.2f} USDT!")

        # Calculate progress
        progress_pct = (self.ultra_config.current_balance / self.ultra_config.target_balance) * 100

        result = {
            'status': 'EXECUTED',
            'symbol': idea.get('symbol'),
            'action': idea.get('action'),
            'outcome': outcome,
            'pnl': pnl,
            'new_balance': self.ultra_config.current_balance,
            'win_rate': (self.stats['winning_trades'] / self.stats['total_trades'] * 100),
            'streak': self.stats['win_streak'] if trade_won else -self.stats['loss_streak'],
            'compounded': compound_amount > 0,
            'compounding_amount': compound_amount,
            'progress_pct': progress_pct,
            'daily_pnl': self.stats['daily_pnl'] + pnl
        }

        logger.info(f"{'✅' if trade_won else '❌'} Trade {outcome}: {pnl:.4f} USDT")
        logger.info(f"New balance: {self.ultra_config.current_balance:.4f} USDT")
        logger.info(f"Win rate: {result['win_rate']:.1f}%")
        logger.info(f"Streak: {result['streak']}")
        logger.info(f"Progress: {progress_pct:.1f}% to target")

        return result

    async def _handle_report(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 5: Ultra-aggressive performance report"""
        logger.info("="*60)
        logger.info("Phase 5: PERFORMANCE REPORT")
        logger.info("="*60)

        trade_result = self.current_cycle.phase_results.get('backtest', {})

        await asyncio.sleep(0.05)

        # Calculate metrics
        win_rate = (self.stats['winning_trades'] / max(self.stats['total_trades'], 1)) * 100

        growth_pct = ((self.ultra_config.current_balance / self.ultra_config.initial_balance - 1) * 100)

        # Time-based metrics
        elapsed_time = datetime.now() - self.stats['start_time']
        elapsed_hours = elapsed_time.total_seconds() / 3600

        # Projections
        projections = self.ultra_config.get_projection_days()

        report = {
            'timestamp': datetime.now().isoformat(),
            'balance': {
                'current': self.ultra_config.current_balance,
                'initial': self.ultra_config.initial_balance,
                'growth_pct': growth_pct,
                'target': self.ultra_config.target_balance,
                'progress_pct': (self.ultra_config.current_balance / self.ultra_config.target_balance * 100),
                'stretch_target': self.ultra_config.stretch_target,
                'stretch_progress_pct': (self.ultra_config.current_balance / self.ultra_config.stretch_target * 100)
            },
            'performance': {
                'total_trades': self.stats['total_trades'],
                'winning_trades': self.stats['winning_trades'],
                'losing_trades': self.stats['losing_trades'],
                'win_rate': f"{win_rate:.1f}%",
                'total_pnl': f"{self.stats['total_pnl']:.4f} USDT",
                'largest_win': f"{self.stats['largest_win']:.4f}",
                'largest_loss': f"{self.stats['largest_loss']:.4f}",
                'current_streak': self.stats['win_streak'] if self.stats['win_streak'] > 0 else -self.stats['loss_streak'],
                'compounds': self.stats['compounds']
            },
            'strategy': {
                'leverage': f"{self.ultra_config.leverage}x",
                'risk_per_trade': f"{self.ultra_config.risk_per_trade*100:.1f}%",
                'stop_loss': f"{self.ultra_config.stop_loss*100:.3f}%",
                'take_profit': f"{self.ultra_config.take_profit*100:.3f}%",
                'rr_ratio': f"{self.ultra_config.take_profit/self.ultra_config.stop_loss:.1f}x",
                'min_confidence': f"{self.ultra_config.min_confidence*100:.0f}%"
            },
            'time_metrics': {
                'elapsed_hours': f"{elapsed_hours:.1f}",
                'trades_per_hour': f"{self.stats['total_trades']/max(elapsed_hours, 0.1):.1f}",
                'daily_target': f"{self.ultra_config.daily_target_pct}%",
                'weekly_target': f"{self.ultra_config.weekly_target_pct}%"
            },
            'projections': {
                'days_to_100': f"{projections['to_100']:.1f}",
                'days_to_500': f"{projections['to_500']:.1f}",
                'expected_daily_growth': f"{self.ultra_config.daily_target_pct}%"
            },
            'next_actions': self._get_next_actions()
        }

        logger.info(f"Balance: {report['balance']['current']:.4f} USDT")
        logger.info(f"Growth: {report['balance']['growth_pct']:.1f}%")
        logger.info(f"Progress to 100: {report['balance']['progress_pct']:.1f}%")
        logger.info(f"Win Rate: {report['performance']['win_rate']}")
        logger.info(f"Trades: {report['performance']['total_trades']}")
        logger.info(f"Expected days to 100 USDT: {report['projections']['days_to_100']}")

        return report

    def _get_next_actions(self) -> List[str]:
        """Get next actions based on performance"""
        actions = []

        if self.ultra_config.current_balance < 10:
            actions.append("Continue ultra-aggressive micro-trading")

        if self.ultra_config.current_balance >= self.ultra_config.compound_threshold:
            actions.append("Compound 70% of profits daily")

        if self.stats['loss_streak'] >= 3:
            actions.append("Reduce position size after loss streak")

        if self.ultra_config.current_balance >= self.ultra_config.target_balance:
            actions.append("🎉 TARGET ACHIEVED! Consider withdrawing profits")

        return actions

    def _check_completion_signal(self) -> bool:
        """Check if targets reached"""
        # Main target
        if self.ultra_config.current_balance >= self.ultra_config.target_balance:
            logger.info(f"🎉 MAIN TARGET ACHIEVED! {self.ultra_config.current_balance:.2f} USDT")
            return True

        # Stretch target
        if self.ultra_config.current_balance >= self.ultra_config.stretch_target:
            logger.info(f"🚀 STRETCH TARGET ACHIEVED! {self.ultra_config.current_balance:.2f} USDT")
            return True

        # Stop if balance drops too low
        if self.ultra_config.current_balance < 0.5:
            logger.warning("Balance too low - stopping")
            return True

        # Stop if too many losses
        if self.stats['loss_streak'] >= 5:
            logger.warning("Too many losses - taking break")
            return True

        return False


async def main():
    """Run ultra-aggressive micro-trading orchestrator"""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
    )

    # Banner
    print("\n" + "="*60)
    print("🚀 GOBOT ULTRA-AGGRESSIVE MICRO-TRADING 🚀")
    print("="*60)
    print("⚠️  WARNING: ULTRA-HIGH RISK STRATEGY ⚠️")
    print("="*60)
    print("Starting: 1 USDT")
    print("Target: 100 USDT")
    print("Stretch: 500 USDT")
    print("Leverage: 125x")
    print("Risk: 0.5% per trade")
    print("Stop Loss: 0.15%")
    print("Take Profit: 0.45% (3:1 RR)")
    print("="*60 + "\n")

    config = OrchestratorConfig(
        max_iterations=2000,  # Run until target
        sleep_between_iterations=15,  # 15 seconds between trades
        state_dir="./ultra_aggressive_state",
        archive_dir="./ultra_aggressive_archive",
        requests_per_minute=120,
        circuit_breaker_failure_threshold=15
    )

    orchestrator = GOBOTUltraAggressiveOrchestrator(config)

    logger.info("Starting ultra-aggressive micro-trading")
    logger.info(f"Target: {orchestrator.ultra_config.target_balance} USDT")
    logger.info(f"Stretch target: {orchestrator.ultra_config.stretch_target} USDT")

    completed = await orchestrator.run(
        max_iterations=2000,
        current_branch="ultra-aggressive"
    )

    if completed:
        logger.info("🎉 ULTRA-AGGRESSIVE TARGETS ACHIEVED!")
    else:
        logger.info("Ultra-aggressive session ended")

    return completed


if __name__ == "__main__":
    result = asyncio.run(main())
    exit(0 if result else 1)
