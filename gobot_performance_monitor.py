#!/usr/bin/env python3
"""
GOBOT Performance Monitor
========================

State management with Ralph patterns for GOBOT

Features:
- Track trading performance across branches/sessions
- Archive old runs when switching strategies
- Pattern discovery: which strategies work best
- Progress tracking with learnings
"""

import json
import logging
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Any, Optional
from dataclasses import dataclass, asdict

logger = logging.getLogger(__name__)


@dataclass
class TradingCycle:
    """Single trading cycle result"""
    cycle_id: str
    timestamp: datetime
    symbol: str
    action: str  # BUY, SELL, HOLD
    confidence: float
    entry_price: Optional[float]
    exit_price: Optional[float]
    pnl: Optional[float]
    duration_minutes: Optional[float]
    success: bool
    learnings: List[str]


@dataclass
class StrategyPerformance:
    """Strategy performance metrics"""
    strategy_name: str
    total_cycles: int
    win_rate: float
    profit_factor: float
    avg_win: float
    avg_loss: float
    max_drawdown: float
    sharpe_ratio: float
    total_pnl: float


class GOBOTPerformanceMonitor:
    """
    GOBOT Performance Monitor with Ralph patterns
    - Branch tracking: archive when switching strategies
    - Pattern discovery: which strategies work best
    - Progress tracking with learnings
    - State persistence
    """

    def __init__(self, state_dir: str = "./gobot_performance"):
        self.state_dir = Path(state_dir)
        self.state_dir.mkdir(exist_ok=True)

        self.branch_file = self.state_dir / "current_branch.json"
        self.progress_file = self.state_dir / "progress.json"
        self.strategies_file = self.state_dir / "strategies.json"
        self.archive_dir = self.state_dir / "archive"

        self.archive_dir.mkdir(exist_ok=True)

        # Track current strategy
        self.current_strategy = "default"
        self.session_start = datetime.now()

    def get_current_branch(self) -> Optional[str]:
        """Get current branch/strategy"""
        if self.branch_file.exists():
            try:
                data = json.loads(self.branch_file.read_text())
                return data.get("strategy")
            except:
                pass
        return None

    def set_current_strategy(self, strategy: str):
        """Set current strategy"""
        self.branch_file.write_text(json.dumps({
            "strategy": strategy,
            "timestamp": datetime.now().isoformat()
        }, indent=2))
        self.current_strategy = strategy

    def should_archive(self, new_strategy: str) -> bool:
        """Check if we should archive (Ralph pattern)"""
        last_strategy = self.get_current_branch()
        return last_strategy is not None and last_strategy != new_strategy

    def archive_current_strategy(self, new_strategy: str):
        """Archive strategy performance when switching (Ralph pattern)"""
        if not self.should_archive(new_strategy):
            return

        last_strategy = self.get_current_branch()
        date_str = datetime.now().strftime("%Y-%m-%d")
        archive_folder = self.archive_dir / f"{date_str}-{last_strategy}"

        logger.info(f"Archiving strategy {last_strategy} -> {archive_folder}")

        archive_folder.mkdir(exist_ok=True)

        # Copy strategy data
        for file_path in [self.progress_file, self.strategies_file]:
            if file_path.exists():
                dest = archive_folder / file_path.name
                dest.write_text(file_path.read_text())

        # Update current strategy
        self.set_current_strategy(new_strategy)

        logger.info(f"Strategy {last_strategy} archived, switched to {new_strategy}")

    def record_trading_cycle(self, cycle: TradingCycle):
        """Record a single trading cycle"""
        progress = self.load_progress()
        if "cycles" not in progress:
            progress["cycles"] = []

        progress["cycles"].append(asdict(cycle))
        self.save_progress(progress)

        # Also save to current strategy file
        self._update_strategy_metrics(cycle)

        logger.info(f"Recorded cycle {cycle.cycle_id}: {cycle.action} {cycle.symbol} "
                   f"(P&L: ${cycle.pnl:.2f})")

    def load_progress(self) -> Dict:
        """Load progress from state"""
        if self.progress_file.exists():
            try:
                return json.loads(self.progress_file.read_text())
            except:
                pass
        return {
            "strategy": self.current_strategy,
            "session_start": self.session_start.isoformat(),
            "cycles": [],
            "patterns": [],
            "learnings": []
        }

    def save_progress(self, progress: Dict):
        """Save progress to state"""
        self.progress_file.write_text(json.dumps(progress, indent=2))

    def add_pattern(self, pattern: str):
        """Add reusable pattern (Ralph pattern)"""
        progress = self.load_progress()
        if "patterns" not in progress:
            progress["patterns"] = []
        if pattern not in progress["patterns"]:
            progress["patterns"].append(pattern)
            self.save_progress(progress)
            logger.info(f"Pattern added: {pattern}")

    def add_learning(self, learning: str):
        """Add learning from trading session"""
        progress = self.load_progress()
        if "learnings" not in progress:
            progress["learnings"] = []
        progress["learnings"].append({
            "timestamp": datetime.now().isoformat(),
            "strategy": self.current_strategy,
            "learning": learning
        })
        self.save_progress(progress)
        logger.info(f"Learning: {learning}")

    def _update_strategy_metrics(self, cycle: TradingCycle):
        """Update strategy performance metrics"""
        strategies = self.load_strategies()

        if self.current_strategy not in strategies:
            strategies[self.current_strategy] = {
                "name": self.current_strategy,
                "cycles": [],
                "total_cycles": 0,
                "wins": 0,
                "losses": 0,
                "total_pnl": 0.0,
                "wins_list": [],
                "losses_list": []
            }

        strategy_data = strategies[self.current_strategy]
        strategy_data["cycles"].append(asdict(cycle))
        strategy_data["total_cycles"] += 1
        strategy_data["total_pnl"] += cycle.pnl or 0

        if cycle.pnl and cycle.pnl > 0:
            strategy_data["wins"] += 1
            strategy_data["wins_list"].append(cycle.pnl)
        elif cycle.pnl:
            strategy_data["losses"] += 1
            strategy_data["losses_list"].append(abs(cycle.pnl))

        # Calculate metrics
        total = strategy_data["total_cycles"]
        wins = strategy_data["wins"]
        losses = strategy_data["losses"]

        strategy_data["win_rate"] = (wins / total * 100) if total > 0 else 0
        strategy_data["profit_factor"] = (
            sum(strategy_data["wins_list"]) / sum(strategy_data["losses_list"])
            if strategy_data["losses_list"] and sum(strategy_data["losses_list"]) > 0
            else 0
        )
        strategy_data["avg_win"] = (
            sum(strategy_data["wins_list"]) / len(strategy_data["wins_list"])
            if strategy_data["wins_list"] else 0
        )
        strategy_data["avg_loss"] = (
            sum(strategy_data["losses_list"]) / len(strategy_data["losses_list"])
            if strategy_data["losses_list"] else 0
        )

        self.save_strategies(strategies)

    def load_strategies(self) -> Dict:
        """Load strategy performance data"""
        if self.strategies_file.exists():
            try:
                return json.loads(self.strategies_file.read_text())
            except:
                pass
        return {}

    def save_strategies(self, strategies: Dict):
        """Save strategy performance data"""
        self.strategies_file.write_text(json.dumps(strategies, indent=2))

    def get_strategy_performance(self, strategy: Optional[str] = None) -> Dict:
        """Get performance for a specific strategy"""
        strategy = strategy or self.current_strategy
        strategies = self.load_strategies()

        if strategy not in strategies:
            return {"error": f"Strategy {strategy} not found"}

        data = strategies[strategy]

        return {
            "strategy": strategy,
            "total_cycles": data["total_cycles"],
            "win_rate": f"{data['win_rate']:.1f}%",
            "profit_factor": f"{data['profit_factor']:.2f}",
            "total_pnl": f"${data['total_pnl']:.2f}",
            "avg_win": f"${data['avg_win']:.2f}",
            "avg_loss": f"${data['avg_loss']:.2f}",
            "wins": data["wins"],
            "losses": data["losses"],
            "wins_list": data["wins_list"][-5:],  # Last 5 wins
            "losses_list": data["losses_list"][-5:]  # Last 5 losses
        }

    def get_best_strategies(self, limit: int = 5) -> List[Dict]:
        """Get top performing strategies"""
        strategies = self.load_strategies()

        # Calculate score for each strategy
        scored = []
        for name, data in strategies.items():
            if data["total_cycles"] >= 5:  # Minimum sample size
                score = (
                    data["win_rate"] * 0.4 +  # 40% weight on win rate
                    data["profit_factor"] * 20 * 0.3 +  # 30% weight on profit factor
                    (100 - data.get("max_drawdown", 0)) * 0.3  # 30% weight on drawdown
                )
                scored.append({
                    "strategy": name,
                    "score": score,
                    "win_rate": data["win_rate"],
                    "profit_factor": data["profit_factor"],
                    "total_pnl": data["total_pnl"]
                })

        # Sort by score
        scored.sort(key=lambda x: x["score"], reverse=True)

        return scored[:limit]

    def discover_patterns(self):
        """Discover patterns from trading data"""
        progress = self.load_progress()
        strategies = self.load_strategies()

        patterns = []

        # Pattern 1: High confidence trades perform better
        high_conf_cycles = [
            c for c in progress.get("cycles", [])
            if c.get("confidence", 0) >= 0.8
        ]
        if high_conf_cycles:
            win_rate = sum(1 for c in high_conf_cycles if c.get("success", False)) / len(high_conf_cycles)
            patterns.append(f"High confidence (80%+) trades: {win_rate:.1%} win rate")

        # Pattern 2: Best performing strategy
        best = self.get_best_strategies(1)
        if best:
            patterns.append(f"Best strategy: {best[0]['strategy']} "
                          f"(Win rate: {best[0]['win_rate']:.1f}%, "
                          f"PF: {best[0]['profit_factor']:.2f})")

        # Pattern 3: Time-based patterns
        hour_performance = {}
        for cycle_data in progress.get("cycles", []):
            if cycle_data.get("timestamp"):
                hour = datetime.fromisoformat(cycle_data["timestamp"]).hour
                if hour not in hour_performance:
                    hour_performance[hour] = {"wins": 0, "total": 0}
                hour_performance[hour]["total"] += 1
                if cycle_data.get("success"):
                    hour_performance[hour]["wins"] += 1

        if hour_performance:
            best_hour = max(hour_performance.items(),
                          key=lambda x: x[1]["wins"] / x[1]["total"] if x[1]["total"] > 0 else 0)
            patterns.append(f"Best trading hour: {best_hour[0]}:00 "
                          f"({best_hour[1]['wins']/best_hour[1]['total']:.1%} win rate)")

        # Save patterns
        for pattern in patterns:
            self.add_pattern(pattern)

        return patterns

    def generate_report(self) -> Dict:
        """Generate comprehensive performance report"""
        progress = self.load_progress()
        strategies = self.load_strategies()

        report = {
            "timestamp": datetime.now().isoformat(),
            "current_strategy": self.current_strategy,
            "session_duration_hours": (
                datetime.now() - self.session_start
            ).total_seconds() / 3600,
            "total_cycles": len(progress.get("cycles", [])),
            "patterns": progress.get("patterns", []),
            "learnings": progress.get("learnings", []),
            "strategy_performance": {
                name: {
                    "cycles": data["total_cycles"],
                    "win_rate": f"{data['win_rate']:.1f}%",
                    "pnl": f"${data['total_pnl']:.2f}"
                }
                for name, data in strategies.items()
            },
            "best_strategies": self.get_best_strategies(3),
            "recent_cycles": progress.get("cycles", [])[-10:]  # Last 10 cycles
        }

        return report


async def main():
    """Test GOBOT Performance Monitor"""
    logging.basicConfig(level=logging.INFO)

    monitor = GOBOTPerformanceMonitor("./test_performance")

    # Simulate trading sessions with different strategies
    strategies = ["conservative", "aggressive", "scalping"]

    for strategy in strategies:
        logger.info(f"\n{'='*60}")
        logger.info(f"Testing Strategy: {strategy.upper()}")
        logger.info('='*60)

        monitor.set_current_strategy(strategy)

        # Simulate 10 cycles per strategy
        import random
        for i in range(10):
            cycle = TradingCycle(
                cycle_id=f"{strategy}_{i}",
                timestamp=datetime.now(),
                symbol="BTCUSDT",
                action=random.choice(["BUY", "SELL"]),
                confidence=random.uniform(0.5, 0.95),
                entry_price=random.uniform(90000, 100000),
                exit_price=random.uniform(90000, 100000),
                pnl=random.uniform(-5, 15),
                duration_minutes=random.uniform(5, 60),
                success=random.choice([True, False]),
                learnings=[
                    f"Learning from {strategy} cycle {i}",
                    f"Market condition observed"
                ]
            )

            monitor.record_trading_cycle(cycle)

        # Add strategy-specific learning
        monitor.add_learning(
            f"Strategy {strategy} performs well in volatile markets"
        )

    # Switch strategies (should trigger archive)
    logger.info(f"\n{'='*60}")
    logger.info("SWITCHING STRATEGIES (Archive Test)")
    logger.info('='*60)

    monitor.archive_current_strategy("new-hybrid-strategy")

    # Discover patterns
    logger.info(f"\n{'='*60}")
    logger.info("PATTERN DISCOVERY")
    logger.info('='*60)

    patterns = monitor.discover_patterns()
    for pattern in patterns:
        logger.info(f"Pattern: {pattern}")

    # Generate final report
    logger.info(f"\n{'='*60}")
    logger.info("FINAL PERFORMANCE REPORT")
    logger.info('='*60)

    report = monitor.generate_report()
    print(json.dumps(report, indent=2))

    # Save report
    report_file = Path("gobot_performance_report.json")
    report_file.write_text(json.dumps(report, indent=2))
    logger.info(f"\nReport saved to: {report_file}")


if __name__ == "__main__":
    asyncio.run(main())
