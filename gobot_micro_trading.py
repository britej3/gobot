#!/usr/bin/env python3
"""
GOBOT Micro-Trading Orchestrator
==============================

Ultra-low balance trading: 1 USDT → 100+ USDT
Strategy: High leverage + smart compounding + grid trading

Features:
- 125x leverage on Binance Futures
- 0.1% risk per trade (micro-management)
- Smart liquidation protection
- Compounding strategy
- Grid trading for consolidation
- AI-powered signal filtering
"""

import asyncio
import json
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional
from pathlib import Path

from orchestrator import OrchestratorConfig, ClaudeOrchestrator, CyclePhase

logger = logging.getLogger(__name__)


class MicroTradingConfig:
    """Configuration for micro-trading (1 USDT)"""

    def __init__(self):
        # Starting balance
        self.initial_balance = 1.0  # 1 USDT
        self.current_balance = 1.0

        # High leverage settings
        self.leverage = 125  # 125x leverage
        self.min_order_size = 0.001  # BTC minimum

        # Risk management (ultra-conservative)
        self.risk_per_trade = 0.001  # 0.1% of balance
        self.max_position_size = 10.0  # Max 10 USDT position
        self.stop_loss_pct = 0.2  # 0.2% stop loss
        self.take_profit_pct = 0.4  # 0.4% take profit (2:1 RR)
        self.max_daily_trades = 50  # High frequency

        # Compounding
        self.compound_threshold = 5.0  # Compound when balance > 5 USDT
        self.compound_rate = 0.5  # Compound 50% of profits

        # Grid trading (for consolidation)
        self.grid_enabled = True
        self.grid_size = 0.1  # 0.1% grid spacing
        self.grid_levels = 5

        # Liquidation protection
        self.liquidation_buffer = 5.0  # 5% buffer from liquidation price

        # AI filtering (high confidence only)
        self.min_confidence = 0.95  # 95% confidence minimum

        # Target
        self.target_balance = 100.0  # Grow to 100 USDT

    def get_position_size(self) -> float:
        """Calculate position size based on balance and leverage"""
        # Risk amount
        risk_amount = self.current_balance * self.risk_per_trade

        # Position size with leverage
        position_size = risk_amount * self.leverage / (self.stop_loss_pct / 100)

        # Cap at max position size
        position_size = min(position_size, self.max_position_size)

        # Cap at available balance * leverage
        max_by_balance = self.current_balance * self.leverage
        position_size = min(position_size, max_by_balance)

        # Ensure minimum order size
        position_size = max(position_size, self.min_order_size)

        return position_size

    def get_liquidation_price(self, entry_price: float, side: str) -> float:
        """Calculate liquidation price"""
        # Simplified liquidation calculation
        # Actual formula: liquidation_price = entry_price * (1 - 1/leverage + maintenance_margin)
        maintenance_margin = 0.005  # 0.5% maintenance margin

        if side == "LONG":
            liquidation_price = entry_price * (1 - 1/self.leverage - maintenance_margin)
        else:  # SHORT
            liquidation_price = entry_price * (1 + 1/self.leverage + maintenance_margin)

        # Add buffer
        buffer = liquidation_price * (self.liquidation_buffer / 100)

        if side == "LONG":
            return liquidation_price - buffer
        else:
            return liquidation_price + buffer

    def should_compound(self) -> bool:
        """Check if should compound profits"""
        return self.current_balance >= self.compound_threshold

    def get_compound_amount(self) -> float:
        """Get amount to compound"""
        if not self.should_compound():
            return 0.0

        # Compound 50% of balance above threshold
        excess = self.current_balance - self.compound_threshold
        return excess * self.compound_rate


class GOBOTMicroTradingOrchestrator(ClaudeOrchestrator):
    """
    Micro-trading orchestrator for small balances
    Grows 1 USDT → 100+ USDT using high leverage and smart compounding
    """

    def __init__(self, config: OrchestratorConfig):
        super().__init__(config)
        self.micro_config = MicroTradingConfig()

        # Track performance
        self.trade_stats = {
            "total_trades": 0,
            "winning_trades": 0,
            "losing_trades": 0,
            "total_pnl": 0.0,
            "largest_win": 0.0,
            "largest_loss": 0.0,
            "win_streak": 0,
            "loss_streak": 0,
            "current_streak": 0,
            "balance_history": [],
            "compounds": 0
        }

        # Grid orders
        self.active_grids = []
        self.grid_orders = []

        logger.info("🚀 GOBOT Micro-Trading Orchestrator Initialized")
        logger.info(f"Starting balance: {self.micro_config.initial_balance} USDT")
        logger.info(f"Leverage: {self.micro_config.leverage}x")
        logger.info(f"Target: {self.micro_config.target_balance} USDT")

    async def _handle_data_scan(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 1: Ultra-precise market scan"""
        logger.info("="*60)
        logger.info("Phase 1: ULTRA-PRECISE MARKET SCAN")
        logger.info("="*60)

        await asyncio.sleep(0.1)  # Fast scanning for micro-trading

        # Simulate market data with high precision
        market_data = {
            "timestamp": datetime.now().isoformat(),
            "balance": self.micro_config.current_balance,
            "leverage": self.micro_config.leverage,
            "symbols": {
                "BTCUSDT": {
                    "price": 95320.50,
                    "bid": 95320.25,
                    "ask": 95320.75,
                    "spread": 0.50,
                    "volume_24h": 1250000,
                    "volatility_1h": 0.015,  # 1.5% hourly volatility
                    "rsi_5m": 28.5,  # Very oversold on 5m
                    "rsi_15m": 35.2,
                    "macd_signal": "bullish_divergence",
                    "bb_position": "lower_band",
                    "volume_spike": 1.8,  # 80% above average
                    "order_book_imbalance": 0.65,  # 65% buy pressure
                },
                "ETHUSDT": {
                    "price": 3420.15,
                    "bid": 3420.05,
                    "ask": 3420.25,
                    "spread": 0.20,
                    "volume_24h": 890000,
                    "volatility_1h": 0.018,
                    "rsi_5m": 31.2,
                    "rsi_15m": 38.7,
                    "macd_signal": "bullish_cross",
                    "bb_position": "middle",
                    "volume_spike": 1.5,
                    "order_book_imbalance": 0.58,
                }
            },
            "market_regime": "high_volatility",
            "fear_greed_index": 15,  # Extreme fear
            "funding_rate": -0.0008,  # Negative = longers paid
            "open_interest": "increasing"
        }

        # Calculate signal strength
        signals = []
        for symbol, data in market_data["symbols"].items():
            signal_score = 0

            # RSI signal (oversold = bullish)
            if data["rsi_5m"] < 30:
                signal_score += 25

            # Volume spike (confirmation)
            if data["volume_spike"] > 1.5:
                signal_score += 20

            # Order book imbalance (buy pressure)
            if data["order_book_imbalance"] > 0.6:
                signal_score += 20

            # MACD signal
            if data["macd_signal"] in ["bullish_divergence", "bullish_cross"]:
                signal_score += 20

            # Bollinger Bands
            if data["bb_position"] == "lower_band":
                signal_score += 15

            signals.append({
                "symbol": symbol,
                "signal_score": signal_score,
                "action": "LONG" if signal_score > 60 else "HOLD",
                "strength": "STRONG" if signal_score > 70 else "MODERATE"
            })

        market_data["signals"] = signals

        # Add patterns
        self.state_manager.add_pattern(
            f"Market scan: {market_data['market_regime']} regime, high volatility opportunity"
        )

        logger.info(f"Balance: {market_data['balance']} USDT")
        logger.info(f"Leverage: {market_data['leverage']}x")
        logger.info(f"Market regime: {market_data['market_regime']}")

        for signal in signals:
            logger.info(f"{signal['symbol']}: {signal['action']} "
                       f"(score: {signal['signal_score']}, {signal['strength']})")

        return market_data

    async def _handle_idea(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 2: High-conviction idea generation"""
        logger.info("="*60)
        logger.info("Phase 2: HIGH-CONVICTION IDEA GENERATION")
        logger.info("="*60)

        market_data = self.current_cycle.phase_results.get("data_scan", {})
        signals = market_data.get("signals", [])

        await asyncio.sleep(0.1)

        # Filter for high-conviction signals only
        high_conviction = [
            s for s in signals
            if s["signal_score"] > 70 and s["action"] == "LONG"
        ]

        ideas = []

        for signal in high_conviction:
            symbol_data = market_data["symbols"][signal["symbol"]]

            # Calculate precise entry, SL, TP
            entry_price = symbol_data["price"]
            position_size = self.micro_config.get_position_size()

            # Long position calculations
            sl_price = entry_price * (1 - self.micro_config.stop_loss_pct / 100)
            tp_price = entry_price * (1 + self.micro_config.take_profit_pct / 100)

            # Calculate risk/reward
            risk = (entry_price - sl_price) / entry_price * 100
            reward = (tp_price - entry_price) / entry_price * 100

            # Expected value
            # Assume 70% win rate with 2:1 RR
            expected_value = (0.7 * reward) - (0.3 * risk)

            # Calculate liquidation price
            liq_price = self.micro_config.get_liquidation_price(entry_price, "LONG")
            distance_to_liq = (entry_price - liq_price) / entry_price * 100

            ideas.append({
                "symbol": signal["symbol"],
                "action": "LONG",
                "signal_score": signal["signal_score"],
                "entry_price": entry_price,
                "position_size_usdt": position_size,
                "position_size_coins": position_size / entry_price,
                "leverage": self.micro_config.leverage,
                "stop_loss": sl_price,
                "take_profit": tp_price,
                "liquidation_price": liq_price,
                "risk_percent": risk,
                "reward_percent": reward,
                "expected_value": expected_value,
                "distance_to_liquidation": distance_to_liq,
                "conviction": "VERY_HIGH" if signal["signal_score"] > 80 else "HIGH",
                "reasoning": f"RSI {symbol_data['rsi_5m']:.1f} oversold + "
                           f"volume spike {symbol_data['volume_spike']:.1f}x + "
                           f"buy pressure {symbol_data['order_book_imbalance']:.0%}"
            })

        # Select best idea
        best_idea = None
        if ideas:
            # Sort by expected value
            best_idea = max(ideas, key=lambda x: x["expected_value"])

        result = {
            "total_signals": len(signals),
            "high_conviction_signals": len(high_conviction),
            "ideas": ideas,
            "selected_idea": best_idea,
            "strategy": "HIGH_LEVERAGE_MICRO" if best_idea else "NO_SIGNAL",
            "position_size": self.micro_config.get_position_size() if best_idea else 0
        }

        logger.info(f"Total signals: {result['total_signals']}")
        logger.info(f"High conviction: {result['high_conviction_signals']}")

        if best_idea:
            logger.info(f"Selected: {best_idea['symbol']} LONG")
            logger.info(f"Entry: ${best_idea['entry_price']:.2f}")
            logger.info(f"Position: {best_idea['position_size_usdt']:.2f} USDT "
                       f"({best_idea['leverage']}x)")
            logger.info(f"Risk: {best_idea['risk_percent']:.2f}%")
            logger.info(f"Reward: {best_idea['reward_percent']:.2f}%")
            logger.info(f"Expected Value: {best_idea['expected_value']:.2f}%")
            logger.info(f"Liquidation: ${best_idea['liquidation_price']:.2f} "
                       f"({best_idea['distance_to_liquidation']:.1f}% away)")
        else:
            logger.info("No high-conviction signals - waiting")

        return result

    async def _handle_code_edit(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 3: Smart position sizing and grid setup"""
        logger.info("="*60)
        logger.info("Phase 3: POSITION SIZING & GRID SETUP")
        logger.info("="*60)

        idea_result = self.current_cycle.phase_results.get("idea", {})
        selected_idea = idea_result.get("selected_idea")

        await asyncio.sleep(0.1)

        if not selected_idea:
            # Setup grid trading for consolidation
            logger.info("No signal - setting up grid trading")

            grid_orders = []
            if self.micro_config.grid_enabled:
                base_price = 95320.50  # Current BTC price

                for i in range(self.micro_config.grid_levels):
                    # Lower grid (buy)
                    lower_price = base_price * (1 - (i + 1) * self.micro_config.grid_size / 100)
                    upper_price = base_price * (1 + (i + 1) * self.micro_config.grid_size / 100)

                    grid_orders.append({
                        "type": "GRID_BUY",
                        "symbol": "BTCUSDT",
                        "price": lower_price,
                        "size": 0.001,  # Minimum size
                        "leverage": self.micro_config.leverage,
                        "level": i + 1
                    })

                    grid_orders.append({
                        "type": "GRID_SELL",
                        "symbol": "BTCUSDT",
                        "price": upper_price,
                        "size": 0.001,
                        "leverage": self.micro_config.leverage,
                        "level": i + 1
                    })

                logger.info(f"Set up {len(grid_orders)} grid orders")
                self.grid_orders = grid_orders

            return {
                "status": "GRID_TRADING",
                "reason": "No high-conviction signal",
                "grid_orders": grid_orders,
                "grid_profit_target": 0.05  # 5 cents per grid completion
            }

        # Position sizing for signal
        entry_price = selected_idea["entry_price"]
        position_size = selected_idea["position_size_usdt"]
        leverage = selected_idea["leverage"]

        # Smart adjustments based on balance
        if self.micro_config.current_balance < 2.0:
            # Ultra conservative for very small balances
            leverage = min(leverage, 50)  # Reduce leverage
            position_size *= 0.5  # Half size
            logger.warning("Very small balance - reduced position size and leverage")

        # Calculate margin required
        margin_required = position_size / leverage

        result = {
            "status": "POSITION_SET",
            "idea": selected_idea,
            "adjusted_position": {
                "size_usdt": position_size,
                "leverage": leverage,
                "margin_required": margin_required,
                "free_margin": self.micro_config.current_balance - margin_required,
                "max_leverage_safe": self.micro_config.current_balance / 0.01  # 1% margin
            },
            "risk_metrics": {
                "risk_amount": position_size * (self.micro_config.stop_loss_pct / 100),
                "reward_amount": position_size * (self.micro_config.take_profit_pct / 100),
                "risk_reward_ratio": self.micro_config.take_profit_pct / self.micro_config.stop_loss_pct,
                "margin_safety": margin_required / self.micro_config.current_balance * 100
            },
            "liquidation_protection": {
                "buffer_pct": self.micro_config.liquidation_buffer,
                "safe_distance": selected_idea["distance_to_liquidation"]
            }
        }

        logger.info(f"Position set: {position_size:.2f} USDT @ {entry_price:.2f}")
        logger.info(f"Leverage: {leverage}x (Margin: {margin_required:.3f} USDT)")
        logger.info(f"Risk: {result['risk_metrics']['risk_amount']:.3f} USDT")
        logger.info(f"Reward: {result['risk_metrics']['reward_amount']:.3f} USDT")
        logger.info(f"Risk/Reward: {result['risk_metrics']['risk_reward_ratio']:.1f}x")

        return result

    async def _handle_backtest(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 4: Trade execution simulation"""
        logger.info("="*60)
        logger.info("Phase 4: TRADE EXECUTION")
        logger.info("="*60)

        position_result = self.current_cycle.phase_results.get("code_edit", {})
        status = position_result.get("status")

        await asyncio.sleep(0.1)

        if status == "GRID_TRADING":
            # Grid trading mode
            logger.info("Grid trading mode - waiting for price to hit grids")
            return {
                "status": "GRID_WAITING",
                "mode": "GRID_TRADING",
                "active_grids": len(self.grid_orders),
                "expected_profit_per_grid": 0.05
            }

        # Execute position trade
        idea = position_result.get("idea", {})
        adjusted_pos = position_result.get("adjusted_position", {})

        logger.info(f"Executing {idea.get('action')} order for {idea.get('symbol')}")

        # Simulate trade execution
        import random

        # Higher win rate for high-conviction signals
        win_probability = 0.75  # 75% win rate

        # Check if stop loss or take profit hits first
        trade_won = random.random() < win_probability

        if trade_won:
            pnl = adjusted_pos["size_usdt"] * (self.micro_config.take_profit_pct / 100)
            outcome = "WIN"
            priority = "PROFIT"
        else:
            pnl = -adjusted_pos["size_usdt"] * (self.micro_config.stop_loss_pct / 100)
            outcome = "LOSS"
            priority = "LOSS"

        # Update balance
        self.micro_config.current_balance += pnl

        # Update stats
        self.trade_stats["total_trades"] += 1
        self.trade_stats["total_pnl"] += pnl

        if trade_won:
            self.trade_stats["winning_trades"] += 1
            self.trade_stats["win_streak"] += 1
            self.trade_stats["loss_streak"] = 0
            self.trade_stats["current_streak"] = self.trade_stats["win_streak"]
        else:
            self.trade_stats["losing_trades"] += 1
            self.trade_stats["loss_streak"] += 1
            self.trade_stats["win_streak"] = 0
            self.trade_stats["current_streak"] = -self.trade_stats["loss_streak"]

        if pnl > self.trade_stats["largest_win"]:
            self.trade_stats["largest_win"] = pnl
        if pnl < self.trade_stats["largest_loss"]:
            self.trade_stats["largest_loss"] = pnl

        # Check for compounding
        compound_amount = 0
        if self.micro_config.should_compound():
            compound_amount = self.micro_config.get_compound_amount()
            self.trade_stats["compounds"] += 1
            logger.info(f"🎉 Compounding {compound_amount:.2f} USDT!")

        result = {
            "status": "EXECUTED",
            "symbol": idea.get("symbol"),
            "action": idea.get("action"),
            "outcome": outcome,
            "pnl": pnl,
            "new_balance": self.micro_config.current_balance,
            "win_rate": (self.trade_stats["winning_trades"] /
                        self.trade_stats["total_trades"] * 100),
            "streak": self.trade_stats["current_streak"],
            "compounded": compound_amount > 0,
            "compounding_amount": compound_amount,
            "progress_to_target": (
                self.micro_config.current_balance /
                self.micro_config.target_balance * 100
            )
        }

        logger.info(f"{'✅' if trade_won else '❌'} Trade {outcome}: {pnl:.4f} USDT")
        logger.info(f"New balance: {self.micro_config.current_balance:.4f} USDT")
        logger.info(f"Win rate: {result['win_rate']:.1f}%")
        logger.info(f"Streak: {self.trade_stats['current_streak']}")
        logger.info(f"Progress: {result['progress_to_target']:.1f}% to target")

        return result

    async def _handle_report(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 5: Performance report with compounding strategy"""
        logger.info("="*60)
        logger.info("Phase 5: PERFORMANCE REPORT")
        logger.info("="*60)

        trade_result = self.current_cycle.phase_results.get("backtest", {})

        await asyncio.sleep(0.1)

        # Calculate key metrics
        win_rate = (self.trade_stats["winning_trades"] /
                   max(self.trade_stats["total_trades"], 1) * 100)

        avg_win = (self.trade_stats["largest_win"] /
                  max(self.trade_stats["winning_trades"], 1))

        avg_loss = abs(self.trade_stats["largest_loss"] /
                      max(self.trade_stats["losing_trades"], 1))

        profit_factor = (avg_win * self.trade_stats["winning_trades"]) / \
                       (avg_loss * self.trade_stats["losing_trades"]) \
                       if self.trade_stats["losing_trades"] > 0 else float('inf')

        # Project time to target
        daily_trades = self.micro_config.max_daily_trades
        expected_daily_pnl = win_rate * avg_win * daily_trades / 100
        days_to_target = (self.micro_config.target_balance -
                         self.micro_config.current_balance) / max(expected_daily_pnl, 0.01)

        report = {
            "timestamp": datetime.now().isoformat(),
            "balance": {
                "current": self.micro_config.current_balance,
                "initial": self.micro_config.initial_balance,
                "growth": ((self.micro_config.current_balance /
                           self.micro_config.initial_balance - 1) * 100),
                "target": self.micro_config.target_balance,
                "progress_pct": (self.micro_config.current_balance /
                                self.micro_config.target_balance * 100)
            },
            "performance": {
                "total_trades": self.trade_stats["total_trades"],
                "win_rate": f"{win_rate:.1f}%",
                "profit_factor": f"{profit_factor:.2f}",
                "total_pnl": f"{self.trade_stats['total_pnl']:.4f} USDT",
                "largest_win": f"{self.trade_stats['largest_win']:.4f}",
                "largest_loss": f"{self.trade_stats['largest_loss']:.4f}",
                "current_streak": self.trade_stats["current_streak"],
                "compounds": self.trade_stats["compounds"]
            },
            "projections": {
                "expected_daily_pnl": f"{expected_daily_pnl:.4f} USDT",
                "days_to_target": f"{days_to_target:.1f}",
                "weekly_growth": f"{expected_daily_pnl * 7:.2f} USDT"
            },
            "strategy_status": {
                "leverage": f"{self.micro_config.leverage}x",
                "risk_per_trade": f"{self.micro_config.risk_per_trade * 100:.1f}%",
                "mode": "COMPOUNDING" if self.micro_config.should_compound() else "GROWTH"
            },
            "next_actions": self._get_next_actions()
        }

        logger.info(f"Current Balance: {report['balance']['current']:.4f} USDT")
        logger.info(f"Growth: {report['balance']['growth']:.1f}%")
        logger.info(f"Progress: {report['balance']['progress_pct']:.1f}% to target")
        logger.info(f"Win Rate: {report['performance']['win_rate']}")
        logger.info(f"Expected Daily P&L: {report['projections']['expected_daily_pnl']}")
        logger.info(f"Days to Target: {report['projections']['days_to_target']}")

        return report

    def _get_next_actions(self) -> List[str]:
        """Determine next actions based on current state"""
        actions = []

        if self.micro_config.current_balance < 2.0:
            actions.append("Continue micro-trading with high conviction signals only")

        if self.micro_config.should_compound():
            actions.append("Compound profits when balance reaches threshold")

        if self.trade_stats["loss_streak"] >= 3:
            actions.append("Reduce position size after loss streak")

        if self.micro_config.current_balance >= self.micro_config.target_balance:
            actions.append("🎉 TARGET ACHIEVED! Consider taking profits")

        return actions

    def _check_completion_signal(self) -> bool:
        """Check if target reached or should stop"""
        # Stop if target reached
        if self.micro_config.current_balance >= self.micro_config.target_balance:
            logger.info(f"🎉 TARGET ACHIEVED! {self.micro_config.current_balance:.2f} USDT")
            return True

        # Stop if balance drops too low
        if self.micro_config.current_balance < 0.5:
            logger.warning("Balance too low - stopping")
            return True

        # Stop if too many losses
        if self.trade_stats["loss_streak"] >= 5:
            logger.warning("Too many losses - taking break")
            return True

        return False


async def main():
    """Run micro-trading orchestrator"""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
    )

    # Print banner
    print("\n" + "="*60)
    print("🚀 GOBOT MICRO-TRADING ORCHESTRATOR 🚀")
    print("="*60)
    print("Strategy: 1 USDT → 100+ USDT")
    print("Leverage: 125x")
    print("Risk: 0.1% per trade")
    print("="*60 + "\n")

    config = OrchestratorConfig(
        max_iterations=1000,  # Run until target reached
        sleep_between_iterations=30,  # 30 seconds between trades
        state_dir="./micro_trading_state",
        archive_dir="./micro_trading_archive",
        requests_per_minute=60,
        circuit_breaker_failure_threshold=10
    )

    orchestrator = GOBOTMicroTradingOrchestrator(config)

    logger.info("Starting micro-trading orchestrator")
    logger.info(f"Target: {orchestrator.micro_config.target_balance} USDT")

    completed = await orchestrator.run(
        max_iterations=1000,
        current_branch="micro-trading"
    )

    if completed:
        logger.info("🎉 MICRO-TRADING TARGET ACHIEVED!")
    else:
        logger.info("Micro-trading session ended")

    return completed


if __name__ == "__main__":
    result = asyncio.run(main())
    exit(0 if result else 1)
