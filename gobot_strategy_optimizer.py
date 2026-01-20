#!/usr/bin/env python3
"""
GOBOT Strategy Optimizer
=======================

Ralph-style iterative improvement for trading strategies

Cycle:
1. Data Scan → Analyze current strategy performance
2. Idea → Generate optimization ideas
3. Code Edit → Adjust parameters
4. Backtest → Validate changes
5. Report → Document improvements

Uses Ralph patterns:
- Exit on completion (strategy meets targets)
- Max iterations (prevent infinite loops)
- State persistence (track improvements)
- Pattern discovery (learn what works)
"""

import asyncio
import json
import logging
from datetime import datetime
from typing import Dict, List, Any, Optional
from pathlib import Path
import random

from orchestrator import OrchestratorConfig, ClaudeOrchestrator, CyclePhase

logger = logging.getLogger(__name__)


class GOBOTStrategyOptimizer(ClaudeOrchestrator):
    """
    Ralph-inspired strategy optimizer
    Iteratively improves trading strategies
    """

    def __init__(self, config: OrchestratorConfig):
        super().__init__(config)
        self.target_metrics = {
            "min_win_rate": 60.0,  # %
            "min_profit_factor": 1.5,
            "max_drawdown": 10.0,  # %
            "min_trades": 100  # minimum sample size
        }
        self.optimization_history = []

    async def _handle_data_scan(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 1: Strategy Performance Analysis"""
        logger.info("="*60)
        logger.info("Phase 1: STRATEGY PERFORMANCE SCAN")
        logger.info("="*60)

        # In real implementation, analyze actual strategy performance
        # For demo, simulate current performance

        await asyncio.sleep(0.5)

        current_performance = {
            "strategy_name": "BTC Scalping v2.1",
            "total_trades": 87,
            "win_rate": 58.6,  # %
            "profit_factor": 1.34,
            "max_drawdown": 7.2,  # %
            "total_pnl": 245.67,  # USD
            "sharpe_ratio": 1.15,
            "avg_trade_duration": 23.5,  # minutes
            "best_symbol": "BTCUSDT",
            "worst_symbol": "XRPUSDT",
            "best_timeframe": "5m",
            "worst_timeframe": "1m",
            "parameters": {
                "rsi_period": 14,
                "rsi_oversold": 30,
                "rsi_overbought": 70,
                "stop_loss_pct": 2.0,
                "take_profit_pct": 4.0,
                "position_size_pct": 2.0
            }
        }

        # Analyze gaps vs targets
        gaps = []
        if current_performance["win_rate"] < self.target_metrics["min_win_rate"]:
            gaps.append({
                "metric": "win_rate",
                "current": current_performance["win_rate"],
                "target": self.target_metrics["min_win_rate"],
                "gap": self.target_metrics["min_win_rate"] - current_performance["win_rate"]
            })

        if current_performance["profit_factor"] < self.target_metrics["min_profit_factor"]:
            gaps.append({
                "metric": "profit_factor",
                "current": current_performance["profit_factor"],
                "target": self.target_metrics["min_profit_factor"],
                "gap": self.target_metrics["min_profit_factor"] - current_performance["profit_factor"]
            })

        if current_performance["max_drawdown"] > self.target_metrics["max_drawdown"]:
            gaps.append({
                "metric": "max_drawdown",
                "current": current_performance["max_drawdown"],
                "target": self.target_metrics["max_drawdown"],
                "gap": current_performance["max_drawdown"] - self.target_metrics["max_drawdown"]
            })

        result = {
            "performance": current_performance,
            "target_metrics": self.target_metrics,
            "gaps": gaps,
            "strengths": [
                "Good Sharpe ratio",
                "Reasonable drawdown",
                "Profitable overall"
            ],
            "weaknesses": [
                f"Win rate below target ({current_performance['win_rate']:.1f}% vs {self.target_metrics['min_win_rate']:.1f}%)",
                f"Profit factor below target ({current_performance['profit_factor']:.2f} vs {self.target_metrics['min_profit_factor']:.2f})"
            ]
        }

        # Add pattern
        self.state_manager.add_pattern(
            f"Strategy scan: Win rate {current_performance['win_rate']:.1f}%, needs improvement"
        )

        logger.info(f"Strategy: {current_performance['strategy_name']}")
        logger.info(f"Win rate: {current_performance['win_rate']:.1f}% (target: {self.target_metrics['min_win_rate']:.1f}%)")
        logger.info(f"Profit factor: {current_performance['profit_factor']:.2f} (target: {self.target_metrics['min_profit_factor']:.2f})")
        logger.info(f"Drawdown: {current_performance['max_drawdown']:.1f}% (target: <{self.target_metrics['max_drawdown']:.1f}%)")
        logger.info(f"Found {len(gaps)} gaps to optimize")

        return result

    async def _handle_idea(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 2: Optimization Ideas"""
        logger.info("="*60)
        logger.info("Phase 2: OPTIMIZATION IDEAS")
        logger.info("="*60)

        # Get scan results
        scan = self.current_cycle.phase_results.get("data_scan", {})
        gaps = scan.get("gaps", [])
        params = scan.get("performance", {}).get("parameters", {})

        await asyncio.sleep(0.5)

        # Generate optimization ideas based on gaps
        ideas = []

        # Idea 1: Adjust RSI parameters
        ideas.append({
            "title": "Optimize RSI Parameters",
            "priority": "HIGH",
            "effort": "LOW",
            "reasoning": "Current RSI (14, 30/70) may be too sensitive. Try (21, 25/75) to reduce false signals",
            "expected_impact": {
                "win_rate": "+3-5%",
                "profit_factor": "+0.15-0.25"
            },
            "parameter_changes": {
                "rsi_period": {"from": 14, "to": 21},
                "rsi_oversold": {"from": 30, "to": 25},
                "rsi_overbought": {"from": 70, "to": 75}
            }
        })

        # Idea 2: Improve stop loss
        ideas.append({
            "title": "Tighten Stop Loss",
            "priority": "HIGH",
            "effort": "LOW",
            "reasoning": "Reduce losses by tightening SL from 2% to 1.5%, offset by reducing TP from 4% to 3%",
            "expected_impact": {
                "max_drawdown": "-2-3%",
                "profit_factor": "+0.10-0.20"
            },
            "parameter_changes": {
                "stop_loss_pct": {"from": 2.0, "to": 1.5},
                "take_profit_pct": {"from": 4.0, "to": 3.0}
            }
        })

        # Idea 3: Dynamic position sizing
        ideas.append({
            "title": "Implement Dynamic Position Sizing",
            "priority": "MEDIUM",
            "effort": "MEDIUM",
            "reasoning": "Reduce position size during high volatility, increase during low volatility",
            "expected_impact": {
                "max_drawdown": "-1-2%",
                "sharpe_ratio": "+0.10-0.20"
            },
            "parameter_changes": {
                "position_size_pct": {
                    "from": 2.0,
                    "to": "dynamic(1-3%)"
                }
            }
        })

        # Idea 4: Add filter
        ideas.append({
            "title": "Add Volume Filter",
            "priority": "MEDIUM",
            "effort": "MEDIUM",
            "reasoning": "Only trade when volume is above 24h average to avoid false breakouts",
            "expected_impact": {
                "win_rate": "+2-4%",
                "total_trades": "-15-20%"
            },
            "parameter_changes": {
                "volume_filter": {"from": "none", "to": "above_24h_avg"}
            }
        })

        # Select best idea (highest priority, lowest effort, highest impact)
        def score_idea(idea):
            priority_score = {"HIGH": 3, "MEDIUM": 2, "LOW": 1}[idea["priority"]]
            effort_score = {"LOW": 3, "MEDIUM": 2, "HIGH": 1}[idea["effort"]]
            impact_score = sum([
                float(idea["expected_impact"]["win_rate"].replace("+", "").replace("%", "").split("-")[0]),
                float(idea["expected_impact"]["profit_factor"].replace("+", "").split("-")[0])
            ])
            return priority_score * 0.4 + effort_score * 0.3 + impact_score * 0.3

        best_idea = max(ideas, key=score_idea)

        result = {
            "all_ideas": ideas,
            "selected_idea": best_idea,
            "selection_reasoning": f"Highest score: {score_idea(best_idea):.2f}",
            "expected_improvement": best_idea["expected_impact"]
        }

        logger.info(f"Generated {len(ideas)} optimization ideas")
        logger.info(f"Selected: {best_idea['title']}")
        logger.info(f"Reasoning: {best_idea['reasoning']}")
        logger.info(f"Expected impact: {best_idea['expected_impact']}")

        return result

    async def _handle_code_edit(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 3: Implement Strategy Changes"""
        logger.info("="*60)
        logger.info("Phase 3: IMPLEMENT CHANGES")
        logger.info("="*60)

        # Get idea from previous phase
        idea = self.current_cycle.phase_results.get("idea", {}).get("selected_idea", {})
        changes = idea.get("parameter_changes", {})

        await asyncio.sleep(0.5)

        # In real implementation, this would:
        # - Update strategy configuration files
        # - Modify code parameters
        # - Save backup of old parameters

        old_params = {
            "rsi_period": 14,
            "rsi_oversold": 30,
            "rsi_overbought": 70,
            "stop_loss_pct": 2.0,
            "take_profit_pct": 4.0,
            "position_size_pct": 2.0
        }

        new_params = old_params.copy()
        for param, change in changes.items():
            if isinstance(change, dict) and "to" in change:
                new_params[param] = change["to"]

        files_changed = [
            "/Users/britebrt/GOBOT/services/screenshot-service/config/strategy-btc-scalping-v2.2.json",
            "/Users/britebrt/GOBOT/services/screenshot-service/strategies/basic-rsi.js"
        ]

        result = {
            "idea_title": idea.get("title"),
            "old_parameters": old_params,
            "new_parameters": new_params,
            "parameter_changes": changes,
            "files_changed": files_changed,
            "backup_created": True,
            "changes_summary": f"Applied {len(changes)} parameter changes",
            "rollback_plan": "Revert to backup if backtest fails"
        }

        logger.info(f"Idea: {idea.get('title')}")
        logger.info(f"Changed {len(changes)} parameters:")
        for param, change in changes.items():
            if isinstance(change, dict):
                logger.info(f"  {param}: {change['from']} → {change['to']}")
        logger.info(f"Modified {len(files_changed)} files")

        return result

    async def _handle_backtest(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 4: Backtest Strategy Changes"""
        logger.info("="*60)
        logger.info("Phase 4: BACKTEST CHANGES")
        logger.info("="*60)

        # Get changes from previous phase
        changes = self.current_cycle.phase_results.get("code_edit", {})
        new_params = changes.get("new_parameters", {})

        await asyncio.sleep(0.5)

        # Simulate backtest results
        # In real implementation, run actual backtest on historical data

        backtest_results = {
            "period": "Last 30 days (1,440 30-min candles)",
            "sample_size": 156,  # trades
            "baseline": {
                "win_rate": 58.6,
                "profit_factor": 1.34,
                "max_drawdown": 7.2,
                "total_trades": 87
            },
            "optimized": {
                "win_rate": 62.8,  # +4.2%
                "profit_factor": 1.52,  # +0.18
                "max_drawdown": 6.1,  # -1.1%
                "total_trades": 156,
                "sharpe_ratio": 1.34,  # +0.19
                "avg_trade_pnl": 2.87,
                "total_pnl": 447.72
            },
            "improvement": {
                "win_rate": "+4.2%",
                "profit_factor": "+0.18",
                "max_drawdown": "-1.1%",
                "total_trades": "+79.3%"
            },
            "target_achieved": {
                "win_rate": True,  # 62.8% > 60%
                "profit_factor": True,  # 1.52 > 1.5
                "max_drawdown": True,  # 6.1% < 10%
                "min_trades": True  # 156 > 100
            }
        }

        all_targets_met = all(backtest_results["target_achieved"].values())

        result = {
            "status": "PASSED" if all_targets_met else "FAILED",
            "backtest_results": backtest_results,
            "all_targets_met": all_targets_met,
            "ready_for_live": all_targets_met,
            "validation_details": {
                "win_rate": f"{backtest_results['optimized']['win_rate']:.1f}% (target: {self.target_metrics['min_win_rate']:.1f}%)",
                "profit_factor": f"{backtest_results['optimized']['profit_factor']:.2f} (target: {self.target_metrics['min_profit_factor']:.2f})",
                "max_drawdown": f"{backtest_results['optimized']['max_drawdown']:.1f}% (target: <{self.target_metrics['max_drawdown']:.1f}%)",
                "sample_size": f"{backtest_results['sample_size']} trades (target: >{self.target_metrics['min_trades']})"
            }
        }

        logger.info(f"Backtest status: {result['status']}")
        logger.info(f"Win rate: {backtest_results['optimized']['win_rate']:.1f}% (+{backtest_results['improvement']['win_rate']})")
        logger.info(f"Profit factor: {backtest_results['optimized']['profit_factor']:.2f} (+{backtest_results['improvement']['profit_factor']})")
        logger.info(f"Drawdown: {backtest_results['optimized']['max_drawdown']:.1f}% ({backtest_results['improvement']['max_drawdown']})")
        logger.info(f"All targets met: {all_targets_met}")

        return result

    async def _handle_report(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 5: Optimization Report"""
        logger.info("="*60)
        logger.info("Phase 5: OPTIMIZATION REPORT")
        logger.info("="*60)

        # Aggregate all results
        scan = self.current_cycle.phase_results.get("data_scan", {})
        idea = self.current_cycle.phase_results.get("idea", {})
        changes = self.current_cycle.phase_results.get("code_edit", {})
        backtest = self.current_cycle.phase_results.get("backtest", {})

        await asyncio.sleep(0.5)

        report = {
            "timestamp": datetime.now().isoformat(),
            "optimization_id": f"opt_{datetime.now().strftime('%Y%m%d_%H%M%S')}",
            "cycle_summary": {
                "phase_1_scan": "Strategy performance analyzed",
                "phase_2_idea": f"{idea.get('selected_idea', {}).get('title', 'Unknown')} selected",
                "phase_3_changes": f"{len(changes.get('parameter_changes', {}))} parameters modified",
                "phase_4_backtest": f"{backtest.get('status', 'UNKNOWN')} - targets met: {backtest.get('all_targets_met', False)}"
            },
            "before": scan.get("performance", {}),
            "optimization": {
                "idea": idea.get("selected_idea", {}),
                "changes_applied": changes.get("parameter_changes", {}),
                "files_modified": changes.get("files_changed", [])
            },
            "after": backtest.get("backtest_results", {}),
            "improvements": backtest.get("backtest_results", {}).get("improvement", {}),
            "targets_status": backtest.get("target_achieved", {}),
            "next_steps": [
                "Deploy optimized strategy to live trading" if backtest.get("all_targets_met") else "Revert changes and try different idea",
                "Monitor live performance for 24 hours",
                "Schedule next optimization cycle",
                "Document learnings for future iterations"
            ],
            "learnings": [
                f"{idea.get('selected_idea', {}).get('title', 'Unknown')} improved win rate by {backtest.get('backtest_results', {}).get('improvement', {}).get('win_rate', 'N/A')}",
                f"Profit factor increased from {scan.get('performance', {}).get('profit_factor', 0):.2f} to {backtest.get('backtest_results', {}).get('optimized', {}).get('profit_factor', 0):.2f}",
                "Larger sample size (156 trades) provides better confidence"
            ],
            "quality_score": 9.5 if backtest.get("all_targets_met") else 7.0
        }

        # Store in optimization history
        self.optimization_history.append(report)

        logger.info(f"Optimization {report['optimization_id']} completed")
        logger.info(f"Quality score: {report['quality_score']}/10")
        logger.info(f"Status: {'SUCCESS' if backtest.get('all_targets_met') else 'FAILED'}")
        logger.info(f"Next steps: {len(report['next_steps'])} items")

        return report

    def _check_completion_signal(self) -> bool:
        """
        Check for completion (Ralph pattern)
        Strategy optimization complete when:
        - All target metrics achieved
        - OR max iterations reached
        - OR diminishing returns (no improvement in 3 iterations)
        """
        # Check if we have successful optimizations
        recent_optimizations = self.optimization_history[-3:] if self.optimization_history else []

        if recent_optimizations:
            # Check if all recent optimizations met targets
            if all(opt.get("targets_status", {}).get("win_rate", False) for opt in recent_optimizations):
                logger.info("✓ All recent optimizations successful - strategy optimized")
                return True

        # Check if no improvement in last 3 iterations
        if len(recent_optimizations) >= 3:
            improvements = [opt.get("improvements", {}).get("profit_factor", "0") for opt in recent_optimizations]
            # If no significant improvement, consider complete
            if all(imp in ["0", "+0.00", "+0.01", "+0.02"] for imp in improvements):
                logger.info("✓ Diminishing returns detected - optimization complete")
                return True

        return False


async def main():
    """Run GOBOT Strategy Optimizer"""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
    )

    config = OrchestratorConfig(
        max_iterations=10,  # Max optimization iterations
        sleep_between_iterations=1.0,  # Fast iteration for optimization
        state_dir="./gobot_optimizer_state",
        archive_dir="./gobot_optimizer_archive"
    )

    logger.info("🚀 Starting GOBOT Strategy Optimizer")
    logger.info(f"Target win rate: {config.target_metrics['min_win_rate']:.1f}%")
    logger.info(f"Target profit factor: {config.target_metrics['min_profit_factor']:.2f}")

    optimizer = GOBOTStrategyOptimizer(config)
    completed = await optimizer.run(
        max_iterations=10,
        current_branch="strategy-optimization"
    )

    if completed:
        logger.info("✓ Strategy optimization completed successfully")
        logger.info(f"Completed {len(optimizer.optimization_history)} optimization cycles")
    else:
        logger.warning("✗ Strategy optimization terminated")

    return completed


if __name__ == "__main__":
    result = asyncio.run(main())
    exit(0 if result else 1)
