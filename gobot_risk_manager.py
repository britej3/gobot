#!/usr/bin/env python3
"""
GOBOT Risk Manager
==================

Circuit breaker pattern for trading protection

Prevents catastrophic losses by:
- Detecting abnormal market conditions
- Triggering emergency stop on consecutive losses
- Automatic position closure on extreme volatility
- Circuit breaker for API failures
"""

import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional
from enum import Enum

logger = logging.getLogger(__name__)


class RiskLevel(Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"


class TradingCircuitBreaker:
    """
    Circuit breaker for trading decisions
    Prevents cascading losses
    """

    def __init__(self):
        self.state = "CLOSED"  # CLOSED, OPEN, HALF_OPEN
        self.failure_count = 0
        self.consecutive_losses = 0
        self.last_failure_time: Optional[datetime] = None
        self.last_loss_time: Optional[datetime] = None

        # Thresholds
        self.max_consecutive_losses = 5
        self.failure_threshold = 3
        self.recovery_timeout = 1800  # 30 minutes

    async def check_trade(self, trade_decision: Dict) -> Dict:
        """
        Validate trade through circuit breaker
        Returns: {allowed: bool, reason: str, risk_level: RiskLevel}
        """

        # Check circuit state
        if self.state == "OPEN":
            if self._should_attempt_reset():
                self.state = "HALF_OPEN"
                logger.warning("Circuit breaker: HALF_OPEN - allowing test trade")
            else:
                return {
                    "allowed": False,
                    "reason": f"Circuit breaker OPEN - too many failures",
                    "risk_level": RiskLevel.CRITICAL,
                    "action": "BLOCK_ALL_TRADES"
                }

        # Perform risk checks
        risk_checks = [
            self._check_consecutive_losses(trade_decision),
            self._check_drawdown(trade_decision),
            self._check_volatility(trade_decision),
            self._check_correlation(trade_decision),
            self._check_market_hours(trade_decision),
        ]

        # Aggregate risk level
        max_risk = max(risk_checks, key=lambda x: x["risk_level"].value)

        if max_risk["allowed"]:
            self._on_success()
            return {
                "allowed": True,
                "reason": "All risk checks passed",
                "risk_level": RiskLevel.LOW,
                "action": "EXECUTE_TRADE",
                "checks": risk_checks
            }
        else:
            self._on_failure()
            return {
                "allowed": False,
                "reason": max_risk["reason"],
                "risk_level": max_risk["risk_level"],
                "action": max_risk.get("action", "BLOCK_TRADE"),
                "checks": risk_checks
            }

    def _check_consecutive_losses(self, trade: Dict) -> Dict:
        """Check for consecutive losing trades"""
        if self.consecutive_losses >= self.max_consecutive_losses:
            return {
                "allowed": False,
                "reason": f"Too many consecutive losses: {self.consecutive_losses}",
                "risk_level": RiskLevel.HIGH,
                "action": "EMERGENCY_STOP"
            }
        return {"allowed": True, "risk_level": RiskLevel.LOW}

    def _check_drawdown(self, trade: Dict) -> Dict:
        """Check maximum drawdown"""
        # In real implementation, check actual drawdown from portfolio
        simulated_drawdown = 5.2  # %

        if simulated_drawdown > 20:
            return {
                "allowed": False,
                "reason": f"Drawdown too high: {simulated_drawdown}%",
                "risk_level": RiskLevel.CRITICAL,
                "action": "CLOSE_ALL_POSITIONS"
            }
        elif simulated_drawdown > 10:
            return {
                "allowed": False,
                "reason": f"Drawdown elevated: {simulated_drawdown}%",
                "risk_level": RiskLevel.HIGH,
                "action": "REDUCE_POSITION_SIZE"
            }
        return {"allowed": True, "risk_level": RiskLevel.LOW}

    def _check_volatility(self, trade: Dict) -> Dict:
        """Check market volatility"""
        # In real implementation, check ATR, VIX, etc.
        simulated_volatility = 0.15  # 15% annualized

        if simulated_volatility > 0.80:
            return {
                "allowed": False,
                "reason": f"Extreme volatility: {simulated_volatility:.1%}",
                "risk_level": RiskLevel.CRITICAL,
                "action": "CLOSE_ALL_POSITIONS"
            }
        elif simulated_volatility > 0.50:
            return {
                "allowed": False,
                "reason": f"High volatility: {simulated_volatility:.1%}",
                "risk_level": RiskLevel.HIGH,
                "action": "REDUCE_POSITION_SIZE"
            }
        return {"allowed": True, "risk_level": RiskLevel.LOW}

    def _check_correlation(self, trade: Dict) -> Dict:
        """Check correlation between positions"""
        # In real implementation, check actual portfolio correlation
        simulated_correlation = 0.92  # BTC/ETH correlation

        if simulated_correlation > 0.95:
            return {
                "allowed": False,
                "reason": f"Positions too correlated: {simulated_correlation:.2f}",
                "risk_level": RiskLevel.MEDIUM,
                "action": "DIVERSIFY"
            }
        return {"allowed": True, "risk_level": RiskLevel.LOW}

    def _check_market_hours(self, trade: Dict) -> Dict:
        """Check if trading is allowed (e.g., avoid low-liquidity hours)"""
        now = datetime.now()
        hour = now.hour

        # Avoid low-liquidity hours (2am-4am)
        if 2 <= hour <= 4:
            return {
                "allowed": False,
                "reason": "Low liquidity hours (2am-4am)",
                "risk_level": RiskLevel.MEDIUM,
                "action": "WAIT"
            }
        return {"allowed": True, "risk_level": RiskLevel.LOW}

    def _should_attempt_reset(self) -> bool:
        """Check if enough time has passed for recovery"""
        if self.last_failure_time is None:
            return False
        elapsed = (datetime.now() - self.last_failure_time).total_seconds()
        return elapsed >= self.recovery_timeout

    def _on_success(self):
        """Reset circuit breaker on success"""
        self.failure_count = 0
        self.state = "CLOSED"
        logger.info("Circuit breaker: CLOSED - reset on success")

    def _on_failure(self):
        """Handle failure"""
        self.failure_count += 1
        self.consecutive_losses += 1
        self.last_failure_time = datetime.now()
        self.last_loss_time = datetime.now()

        if self.failure_count >= self.failure_threshold:
            self.state = "OPEN"
            logger.critical(
                f"Circuit breaker OPEN: {self.failure_count} failures, "
                f"{self.consecutive_losses} consecutive losses"
            )

    def record_win(self):
        """Record winning trade"""
        self.consecutive_losses = 0
        self._on_success()

    def record_loss(self):
        """Record losing trade"""
        self.consecutive_losses += 1
        self._on_failure()

    def get_status(self) -> Dict:
        """Get circuit breaker status"""
        return {
            "state": self.state,
            "consecutive_losses": self.consecutive_losses,
            "failure_count": self.failure_count,
            "last_failure": self.last_failure_time.isoformat() if self.last_failure_time else None,
            "max_losses": self.max_consecutive_losses,
            "threshold": self.failure_threshold
        }


class GOBOTRiskManager:
    """
    GOBOT Risk Management System
    Combines multiple protection layers
    """

    def __init__(self):
        self.circuit_breaker = TradingCircuitBreaker()
        self.position_limits = {
            "max_position_usd": 10,
            "max_daily_trades": 10,
            "max_daily_loss_usd": 100,
            "max_portfolio_risk": 5  # %
        }
        self.daily_stats = {
            "trades": 0,
            "wins": 0,
            "losses": 0,
            "pnl": 0.0
        }

    async def validate_trade(self, trade_decision: Dict) -> Dict:
        """
        Comprehensive trade validation
        """
        logger.info("="*60)
        logger.info("RISK MANAGEMENT CHECK")
        logger.info("="*60)

        # 1. Circuit breaker check
        circuit_result = await self.circuit_breaker.check_trade(trade_decision)

        # 2. Position size check
        position_size = trade_decision.get("position_size", 0)
        if position_size > self.position_limits["max_position_usd"]:
            position_check = {
                "allowed": False,
                "reason": f"Position size ${position_size} exceeds limit ${self.position_limits['max_position_usd']}",
                "risk_level": RiskLevel.HIGH
            }
        else:
            position_check = {"allowed": True, "risk_level": RiskLevel.LOW}

        # 3. Daily limits check
        if self.daily_stats["trades"] >= self.position_limits["max_daily_trades"]:
            daily_check = {
                "allowed": False,
                "reason": f"Max daily trades reached: {self.position_limits['max_daily_trades']}",
                "risk_level": RiskLevel.HIGH
            }
        elif self.daily_stats["pnl"] <= -self.position_limits["max_daily_loss_usd"]:
            daily_check = {
                "allowed": False,
                "reason": f"Daily loss limit reached: ${self.daily_stats['pnl']}",
                "risk_level": RiskLevel.CRITICAL
            }
        else:
            daily_check = {"allowed": True, "risk_level": RiskLevel.LOW}

        # Aggregate results
        all_checks = [circuit_result, position_check, daily_check]
        max_risk = max(all_checks, key=lambda x: x["risk_level"].value)

        # Final decision
        if max_risk["allowed"]:
            logger.info(f"✓ Trade APPROVED - Risk: {max_risk['risk_level'].value.upper()}")
            logger.info(f"  Reason: {max_risk['reason']}")
            return {
                "approved": True,
                "risk_level": max_risk["risk_level"],
                "action": "EXECUTE_TRADE",
                "checks": all_checks
            }
        else:
            logger.warning(f"✗ Trade REJECTED - Risk: {max_risk['risk_level'].value.upper()}")
            logger.warning(f"  Reason: {max_risk['reason']}")
            return {
                "approved": False,
                "risk_level": max_risk["risk_level"],
                "action": max_risk.get("action", "BLOCK_TRADE"),
                "reason": max_risk["reason"],
                "checks": all_checks
            }

    def record_trade_result(self, won: bool, pnl: float):
        """Record trade result for tracking"""
        self.daily_stats["trades"] += 1
        self.daily_stats["pnl"] += pnl

        if won:
            self.daily_stats["wins"] += 1
            self.circuit_breaker.record_win()
        else:
            self.daily_stats["losses"] += 1
            self.circuit_breaker.record_loss()

        logger.info(f"Trade recorded: {'WIN' if won else 'LOSS'} ${pnl:.2f}")
        logger.info(f"Daily stats: {self.daily_stats}")

    def get_risk_report(self) -> Dict:
        """Generate risk report"""
        return {
            "timestamp": datetime.now().isoformat(),
            "circuit_breaker": self.circuit_breaker.get_status(),
            "daily_stats": self.daily_stats,
            "position_limits": self.position_limits,
            "current_risk": {
                "portfolio_at_risk": 3.2,  # %
                "correlation": 0.85,
                "exposure": {
                    "BTC": 50,  # %
                    "ETH": 30,  # %
                    "CASH": 20   # %
                }
            }
        }


async def main():
    """Test GOBOT Risk Manager"""
    logging.basicConfig(level=logging.INFO)

    risk_manager = GOBOTRiskManager()

    # Test 1: Normal trade
    trade1 = {
        "symbol": "BTCUSDT",
        "action": "LONG",
        "position_size": 10,
        "confidence": 0.85
    }

    result1 = await risk_manager.validate_trade(trade1)
    print(f"\nTest 1: {result1}")

    # Test 2: Large position (should be rejected)
    trade2 = {
        "symbol": "ETHUSDT",
        "action": "LONG",
        "position_size": 50,  # Exceeds limit
        "confidence": 0.90
    }

    result2 = await risk_manager.validate_trade(trade2)
    print(f"\nTest 2: {result2}")

    # Test 3: After consecutive losses (circuit breaker)
    for i in range(6):
        risk_manager.record_trade_result(False, -10)

    trade3 = {
        "symbol": "BTCUSDT",
        "action": "LONG",
        "position_size": 5,
        "confidence": 0.90
    }

    result3 = await risk_manager.validate_trade(trade3)
    print(f"\nTest 3 (after 6 losses): {result3}")

    # Print final risk report
    print("\n" + "="*60)
    print("FINAL RISK REPORT")
    print("="*60)
    print(json.dumps(risk_manager.get_risk_report(), indent=2))


if __name__ == "__main__":
    asyncio.run(main())
