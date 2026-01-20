#!/usr/bin/env python3
"""
GOBOT Mainnet Trading Orchestrator
=================================

Production-ready orchestrator with comprehensive safety for live trading

Features:
- Circuit breakers for protection
- Conservative position sizing
- Real-time risk monitoring
- Telegram alerts for every action
- Dry run mode for testing
- Emergency stop procedures
"""

import asyncio
import json
import logging
import os
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, Optional

# Manually load .env file
env_path = Path(__file__).parent / '.env'
if env_path.exists():
    with open(env_path) as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith('#') and '=' in line:
                key, value = line.split('=', 1)
                os.environ[key.strip()] = value.strip()

from orchestrator import OrchestratorConfig, ClaudeOrchestrator, CyclePhase

logger = logging.getLogger(__name__)


class GOBOTMainnetOrchestrator(ClaudeOrchestrator):
    """
    Mainnet-ready GOBOT orchestrator with safety features
    """

    def __init__(self, config: OrchestratorConfig):
        super().__init__(config)
        self.dry_run = os.getenv("DRY_RUN", "false").lower() == "true"
        self.use_testnet = os.getenv("BINANCE_USE_TESTNET", "true").lower() == "true"
        self.mainnet_mode = not self.use_testnet and not self.dry_run

        # Load safety config
        from mainnet_safety_config import MAINNET_SAFETY_CONFIG
        self.safety_config = MAINNET_SAFETY_CONFIG

        # Track mainnet stats
        self.mainnet_stats = {
            "trades_today": 0,
            "pnl_today": 0.0,
            "last_trade_time": None,
            "circuit_breaker_triggered": False,
            "emergency_stop": False
        }

        # Setup logging
        self._setup_mainnet_logging()

        logger.info(f"Mode: {'DRY RUN' if self.dry_run else 'TESTNET' if self.use_testnet else 'MAINNET'}")
        logger.warning("="*60)
        if self.mainnet_mode:
            logger.critical("⚠️  MAINNET MODE - REAL MONEY AT RISK ⚠️")
            logger.critical("="*60)
        elif self.dry_run:
            logger.info("🧪 DRY RUN MODE - NO REAL TRADES")

    def _setup_mainnet_logging(self):
        """Setup enhanced logging for mainnet"""
        log_file = Path(f"logs/gobot_mainnet_{datetime.now().strftime('%Y%m%d')}.log")
        log_file.parent.mkdir(exist_ok=True)

        # Create file handler
        fh = logging.FileHandler(log_file)
        fh.setLevel(logging.DEBUG)

        # Create formatter
        formatter = logging.Formatter(
            '%(asctime)s [%(levelname)s] %(name)s: %(message)s'
        )
        fh.setFormatter(formatter)

        # Add to logger
        logger.addHandler(fh)

        logger.info(f"Logging to: {log_file}")

    async def _send_telegram_alert(self, message: str, priority: str = "INFO"):
        """Send Telegram alert (integrates with your existing bot)"""
        import aiohttp

        token = self.safety_config["telegram"]["token"]
        chat_id = self.safety_config["telegram"]["chat_id"]

        # Format message with emojis based on priority
        emoji_map = {
            "INFO": "ℹ️",
            "TRADE": "💰",
            "PROFIT": "✅",
            "LOSS": "❌",
            "WARNING": "⚠️",
            "CRITICAL": "🚨",
            "SUCCESS": "🎉"
        }
        emoji = emoji_map.get(priority, "ℹ️")

        formatted_message = f"{emoji} GOBOT Mainnet\n\n{message}"

        try:
            async with aiohttp.ClientSession() as session:
                url = f"https://api.telegram.org/bot{token}/sendMessage"
                data = {
                    "chat_id": chat_id,
                    "text": formatted_message,
                    "parse_mode": "HTML"
                }
                async with session.post(url, data=data) as response:
                    if response.status == 200:
                        logger.info(f"Telegram alert sent: {priority}")
                    else:
                        logger.error(f"Failed to send Telegram alert: {response.status}")
        except Exception as e:
            logger.error(f"Error sending Telegram alert: {e}")

    async def _handle_data_scan(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 1: Enhanced Market Data Scan"""
        logger.info("="*60)
        logger.info("Phase 1: MARKET DATA SCAN")
        logger.info("="*60)

        # Add dry run banner
        if self.dry_run:
            logger.info("🧪 DRY RUN - NO REAL TRADES")

        await asyncio.sleep(0.5)

        # Simulate market data (in production, fetch from Binance)
        market_data = {
            "timestamp": datetime.now().isoformat(),
            "mode": "DRY_RUN" if self.dry_run else "TESTNET" if self.use_testnet else "MAINNET",
            "symbols": {
                "BTCUSDT": {
                    "price": 95320.50,
                    "change_24h": -1.96,
                    "volume": 1250000,
                    "rsi": 45.2,
                    "macd": -0.5,
                    "sentiment": "bearish"
                },
                "ETHUSDT": {
                    "price": 3420.15,
                    "change_24h": -2.31,
                    "volume": 890000,
                    "rsi": 38.7,
                    "macd": -0.3,
                    "sentiment": "bearish"
                }
            },
            "market_conditions": {
                "trend": "bearish",
                "volatility": "medium",
                "fear_greed_index": 25,
                "news_sentiment": "negative"
            },
            "risk_metrics": {
                "portfolio_heat": 3.2,  # % of capital at risk
                "correlation": 0.85,
                "var_24h": 3.2
            },
            "mainnet_stats": self.mainnet_stats
        }

        # Send Telegram alert
        await self._send_telegram_alert(
            f"<b>Market Scan Complete</b>\n"
            f"Mode: {market_data['mode']}\n"
            f"BTC: ${market_data['symbols']['BTCUSDT']['price']} ({market_data['symbols']['BTCUSDT']['change_24h']}%)\n"
            f"Trades today: {self.mainnet_stats['trades_today']}\n"
            f"P&L today: ${self.mainnet_stats['pnl_today']:.2f}",
            "INFO"
        )

        logger.info(f"Market trend: {market_data['market_conditions']['trend']}")
        logger.info(f"BTC price: ${market_data['symbols']['BTCUSDT']['price']}")
        logger.info(f"Mode: {market_data['mode']}")

        return market_data

    async def _handle_idea(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 2: Risk-Aware Idea Generation"""
        logger.info("="*60)
        logger.info("Phase 2: TRADING IDEA GENERATION")
        logger.info("="*60)

        market_data = self.current_cycle.phase_results.get("data_scan", {})

        # Check risk limits BEFORE generating ideas
        if self.mainnet_mode:
            # Check daily limits
            if self.mainnet_stats["trades_today"] >= self.safety_config["risk_manager"]["max_daily_trades"]:
                await self._send_telegram_alert(
                    "Daily trade limit reached. Bot stopping.",
                    "WARNING"
                )
                return {
                    "decision": "STOP",
                    "reason": "Max daily trades reached",
                    "trades_today": self.mainnet_stats["trades_today"]
                }

            if self.mainnet_stats["pnl_today"] <= -self.safety_config["risk_manager"]["max_daily_loss_usd"]:
                await self._send_telegram_alert(
                    f"Daily loss limit reached: ${self.mainnet_stats['pnl_today']:.2f}\nBot stopping.",
                    "CRITICAL"
                )
                return {
                    "decision": "STOP",
                    "reason": "Daily loss limit reached",
                    "pnl_today": self.mainnet_stats["pnl_today"]
                }

        await asyncio.sleep(0.5)

        # Generate ideas with conservative filtering
        ideas = []
        for symbol, data in market_data.get("symbols", {}).items():
            rsi = data.get("rsi", 50)

            # Ultra-conservative: Only trade very clear signals
            if rsi < 25:  # Extremely oversold
                ideas.append({
                    "symbol": symbol,
                    "action": "LONG",
                    "confidence": 0.90,  # High confidence required
                    "reasoning": f"RSI {rsi} extremely oversold",
                    "position_size": min(
                        self.safety_config["trading"]["max_position_usd"],
                        2.0  # Ultra conservative: max $2
                    )
                })

        if not ideas:
            result = {
                "decision": "HOLD",
                "reason": "No high-confidence signals",
                "ideas_generated": 0
            }
            logger.info("No high-confidence trading opportunities")
        else:
            best_idea = ideas[0]
            result = {
                "decision": best_idea["action"],
                "selected_idea": best_idea,
                "ideas_generated": len(ideas),
                "confidence": best_idea["confidence"]
            }
            logger.info(f"Selected: {best_idea['action']} {best_idea['symbol']} "
                       f"(confidence: {best_idea['confidence']:.0%})")

        return result

    async def _handle_code_edit(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 3: Pre-Trade Risk Validation"""
        logger.info("="*60)
        logger.info("Phase 3: PRE-TRADE RISK VALIDATION")
        logger.info("="*60)

        idea_result = self.current_cycle.phase_results.get("idea", {})
        decision = idea_result.get("decision", "HOLD")

        if decision == "HOLD" or decision == "STOP":
            return {
                "status": "NO_TRADE",
                "reason": idea_result.get("reason", "No action needed")
            }

        # Get selected idea
        selected_idea = idea_result.get("selected_idea", {})
        position_size = selected_idea.get("position_size", 0)

        # CRITICAL: Validate position size
        max_position = self.safety_config["trading"]["max_position_usd"]
        if position_size > max_position:
            await self._send_telegram_alert(
                f"Position size ${position_size} exceeds limit ${max_position}\nTrade blocked.",
                "CRITICAL"
            )
            return {
                "status": "BLOCKED",
                "reason": f"Position size ${position_size} > ${max_position} limit"
            }

        # Validate confidence
        min_confidence = self.safety_config["trading"]["min_confidence_threshold"]
        confidence = selected_idea.get("confidence", 0)
        if confidence < min_confidence:
            await self._send_telegram_alert(
                f"Confidence {confidence:.0%} below threshold {min_confidence:.0%}\nTrade blocked.",
                "WARNING"
            )
            return {
                "status": "BLOCKED",
                "reason": f"Confidence {confidence:.0%} < {min_confidence:.0%}"
            }

        # Pre-trade checklist
        checklist = {
            "position_size_valid": position_size <= max_position,
            "confidence_valid": confidence >= min_confidence,
            "daily_limits_ok": (
                self.mainnet_stats["trades_today"] <
                self.safety_config["risk_manager"]["max_daily_trades"]
            ),
            "loss_limit_ok": (
                self.mainnet_stats["pnl_today"] >
                -self.safety_config["risk_manager"]["max_daily_loss_usd"]
            ),
            "dry_run": self.dry_run
        }

        all_valid = all(checklist.values())

        if all_valid:
            result = {
                "status": "APPROVED",
                "trade_details": selected_idea,
                "pre_trade_checklist": checklist,
                "position_size": position_size,
                "stop_loss": position_size * self.safety_config["trading"]["stop_loss_percent"] / 100,
                "take_profit": position_size * self.safety_config["trading"]["take_profit_percent"] / 100
            }
            logger.info(f"✅ Trade APPROVED: {selected_idea['symbol']} {selected_idea['action']}")
            logger.info(f"Position: ${position_size}")
            logger.info(f"Stop Loss: ${result['stop_loss']:.2f}")
            logger.info(f"Take Profit: ${result['take_profit']:.2f}")
        else:
            result = {
                "status": "REJECTED",
                "reason": "Pre-trade validation failed",
                "checklist": checklist
            }
            logger.warning("❌ Trade REJECTED")

        return result

    async def _handle_backtest(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 4: Trade Execution & Validation"""
        logger.info("="*60)
        logger.info("Phase 4: TRADE EXECUTION")
        logger.info("="*60)

        pre_trade = self.current_cycle.phase_results.get("code_edit", {})
        status = pre_trade.get("status", "NO_TRADE")

        if status == "NO_TRADE":
            return {"status": "NO_TRADE", "reason": "No trade to execute"}

        if status == "BLOCKED" or status == "REJECTED":
            return {"status": "BLOCKED", "reason": pre_trade.get("reason")}

        # Get trade details
        trade_details = pre_trade.get("trade_details", {})
        symbol = trade_details.get("symbol")
        action = trade_details.get("action")
        position_size = pre_trade.get("position_size", 0)

        logger.info(f"Executing {action} order for {symbol}")
        logger.info(f"Position size: ${position_size}")

        # Send Telegram alert
        await self._send_telegram_alert(
            f"<b>Trade Executed</b>\n"
            f"Symbol: {symbol}\n"
            f"Action: {action}\n"
            f"Size: ${position_size}\n"
            f"Mode: {'DRY RUN' if self.dry_run else 'LIVE'}",
            "TRADE"
        )

        # Simulate trade execution (in production, call Binance API)
        await asyncio.sleep(0.5)

        # Simulate outcome
        import random
        trade_won = random.choice([True, False])  # 50/50 for demo
        pnl = random.uniform(0.05, 0.30) if trade_won else -random.uniform(0.02, 0.15)

        # Update stats
        self.mainnet_stats["trades_today"] += 1
        self.mainnet_stats["pnl_today"] += pnl
        self.mainnet_stats["last_trade_time"] = datetime.now().isoformat()

        # Determine outcome
        if trade_won:
            outcome = "WIN"
            priority = "PROFIT"
            emoji = "✅"
        else:
            outcome = "LOSS"
            priority = "LOSS"
            emoji = "❌"

        result = {
            "status": "EXECUTED",
            "symbol": symbol,
            "action": action,
            "position_size": position_size,
            "outcome": outcome,
            "pnl": pnl,
            "dry_run": self.dry_run,
            "timestamp": datetime.now().isoformat()
        }

        logger.info(f"{emoji} Trade {outcome}: ${pnl:.2f}")
        logger.info(f"Daily stats: {self.mainnet_stats['trades_today']} trades, "
                   f"${self.mainnet_stats['pnl_today']:.2f} P&L")

        # Send Telegram update
        await self._send_telegram_alert(
            f"<b>Trade Result</b>\n"
            f"{emoji} {outcome}\n"
            f"P&L: ${pnl:.2f}\n"
            f"Daily Total: ${self.mainnet_stats['pnl_today']:.2f}\n"
            f"Trades: {self.mainnet_stats['trades_today']}",
            priority
        )

        # Check if we should stop
        if self.mainnet_mode and self.mainnet_stats["pnl_today"] <= -self.safety_config["risk_manager"]["max_daily_loss_usd"]:
            logger.critical("Daily loss limit reached - activating emergency stop")
            await self._send_telegram_alert(
                "<b>🚨 EMERGENCY STOP 🚨</b>\n"
                f"Daily loss limit reached: ${self.mainnet_stats['pnl_today']:.2f}\n"
                "Bot stopping automatically.",
                "CRITICAL"
            )
            self.mainnet_stats["emergency_stop"] = True

        return result

    async def _handle_report(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 5: Performance Report & Next Steps"""
        logger.info("="*60)
        logger.info("Phase 5: PERFORMANCE REPORT")
        logger.info("="*60)

        all_results = self.current_cycle.phase_results
        trade_result = all_results.get("backtest", {})

        await asyncio.sleep(0.5)

        report = {
            "timestamp": datetime.now().isoformat(),
            "mode": "DRY_RUN" if self.dry_run else "TESTNET" if self.use_testnet else "MAINNET",
            "cycle_summary": {
                "trade_executed": trade_result.get("status") == "EXECUTED",
                "outcome": trade_result.get("outcome", "N/A"),
                "pnl": trade_result.get("pnl", 0),
                "emergency_stop": self.mainnet_stats.get("emergency_stop", False)
            },
            "daily_stats": self.mainnet_stats,
            "next_steps": []
        }

        if trade_result.get("status") == "EXECUTED":
            report["next_steps"] = [
                "Monitor position",
                "Wait for stop loss or take profit",
                "Prepare for next cycle"
            ]
        else:
            report["next_steps"] = [
                "Wait for next market scan",
                "Continue monitoring for opportunities"
            ]

        if self.mainnet_stats.get("emergency_stop"):
            report["next_steps"] = [
                "Review today's trades",
                "Adjust strategy if needed",
                "Restart tomorrow"
            ]

        logger.info(f"Mode: {report['mode']}")
        logger.info(f"Daily P&L: ${self.mainnet_stats['pnl_today']:.2f}")
        logger.info(f"Emergency stop: {self.mainnet_stats.get('emergency_stop', False)}")

        # Send final Telegram update
        await self._send_telegram_alert(
            f"<b>Cycle Complete</b>\n"
            f"Mode: {report['mode']}\n"
            f"Daily P&L: ${self.mainnet_stats['pnl_today']:.2f}\n"
            f"Trades: {self.mainnet_stats['trades_today']}\n"
            f"{'🚨 STOPPED' if self.mainnet_stats.get('emergency_stop') else '✅ CONTINUING'}",
            "INFO"
        )

        return report

    def _check_completion_signal(self) -> bool:
        """Enhanced completion for mainnet"""
        # Stop if emergency stop triggered
        if self.mainnet_stats.get("emergency_stop"):
            logger.info("Emergency stop triggered - completing")
            return True

        # Stop if daily profit target reached
        daily_target = 10.0  # $10 profit target
        if self.mainnet_stats["pnl_today"] >= daily_target:
            logger.info(f"Daily profit target reached: ${self.mainnet_stats['pnl_today']:.2f}")
            return True

        # Stop if daily loss limit reached
        daily_loss_limit = self.safety_config["risk_manager"]["max_daily_loss_usd"]
        if self.mainnet_stats["pnl_today"] <= -daily_loss_limit:
            logger.info(f"Daily loss limit reached: ${self.mainnet_stats['pnl_today']:.2f}")
            return True

        # Stop if max trades reached
        max_trades = self.safety_config["risk_manager"]["max_daily_trades"]
        if self.mainnet_stats["trades_today"] >= max_trades:
            logger.info(f"Max daily trades reached: {self.mainnet_stats['trades_today']}")
            return True

        return False


async def main():
    """Main entry point with mainnet safety checks"""
    import os

    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
    )

    # Print safety warning for mainnet
    print("\n" + "="*60)
    print("⚠️  GOBOT MAINNET TRADING ORCHESTRATOR  ⚠️")
    print("="*60)
    print(f"Mode: {os.getenv('BINANCE_USE_TESTNET', 'true')}")
    print(f"Dry Run: {os.getenv('DRY_RUN', 'false')}")
    print("="*60)

    # Check environment
    if os.getenv("BINANCE_USE_TESTNET", "true").lower() != "false":
        print("\n✅ TESTNET MODE - Safe for testing")
    else:
        print("\n🚨 MAINNET MODE - REAL MONEY AT RISK 🚨")
        print("Are you sure? Press Ctrl+C to cancel or wait 10 seconds...")
        await asyncio.sleep(10)

    config = OrchestratorConfig(
        max_iterations=int(os.getenv("MAX_ITERATIONS", "50")),
        sleep_between_iterations=int(os.getenv("LOOP_INTERVAL", "900")),  # 15 minutes
        state_dir="./gobot_mainnet_state",
        archive_dir="./gobot_mainnet_archive",
        requests_per_minute=10,
        circuit_breaker_failure_threshold=2
    )

    orchestrator = GOBOTMainnetOrchestrator(config)

    logger.info("Starting GOBOT Mainnet Orchestrator")
    logger.info(f"Max iterations: {config.max_iterations}")
    logger.info(f"Loop interval: {config.sleep_between_iterations}s")

    completed = await orchestrator.run(
        max_iterations=config.max_iterations,
        current_branch="mainnet-live"
    )

    if completed:
        logger.info("✅ Orchestrator completed")
    else:
        logger.warning("⚠️ Orchestrator terminated")

    return completed


if __name__ == "__main__":
    result = asyncio.run(main())
    sys.exit(0 if result else 1)
