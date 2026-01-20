#!/usr/bin/env python3
"""
GOBOT Aggressive Micro-Trading Config
==================================

Ultra-aggressive settings for maximum growth from 1 USDT

WARNING: High risk, high reward strategy
"""

from dataclasses import dataclass


@dataclass
class AggressiveMicroConfig:
    """Ultra-aggressive micro-trading configuration"""

    # Starting balance
    initial_balance: float = 1.0
    current_balance: float = 1.0

    # Leverage settings (MAXIMUM)
    leverage: int = 125  # 125x leverage
    min_order_size: float = 0.001

    # Risk management (AGGRESSIVE)
    risk_per_trade: float = 0.005  # 0.5% risk per trade (was 0.1%)
    max_position_size: float = 50.0  # Allow larger positions
    stop_loss_pct: float = 0.15  # 0.15% stop loss (was 0.2%)
    take_profit_pct: float = 0.45  # 0.45% take profit (3:1 RR)
    max_daily_trades: int = 100  # Very high frequency

    # Compounding (AGGRESSIVE)
    compound_threshold: float = 3.0  # Lower threshold (was 5.0)
    compound_rate: float = 0.7  # Compound 70% (was 50%)

    # Grid trading (AGGRESSIVE)
    grid_enabled: bool = True
    grid_size: float = 0.05  # Tighter grids (was 0.1%)
    grid_levels: int = 10  # More levels (was 5)

    # Liquidation protection
    liquidation_buffer: float = 3.0  # Smaller buffer (was 5%)

    # Signal filtering (HIGH CONFIDENCE)
    min_confidence: float = 0.92  # Slightly lower than 95% (was 0.95)

    # Targets
    target_balance: float = 100.0
    stretch_target: float = 500.0  # Even higher target

    def get_position_size(self) -> float:
        """Calculate aggressive position size"""
        # More aggressive risk calculation
        risk_amount = self.current_balance * self.risk_per_trade

        # Position with leverage
        position_size = risk_amount * self.leverage / (self.stop_loss_pct / 100)

        # Cap at max
        position_size = min(position_size, self.max_position_size)

        # Cap by balance
        max_by_balance = self.current_balance * self.leverage
        position_size = min(position_size, max_by_balance)

        # Ensure minimum
        position_size = max(position_size, self.min_order_size)

        return position_size

    def should_compound(self) -> bool:
        """Check if should compound (lower threshold)"""
        return self.current_balance >= self.compound_threshold

    def get_compound_amount(self) -> float:
        """Get compound amount (more aggressive)"""
        if not self.should_compound():
            return 0.0

        # Compound 70% of excess
        excess = self.current_balance - self.compound_threshold
        return excess * self.compound_rate

    def get_liquidation_price(self, entry_price: float, side: str) -> float:
        """Calculate liquidation price with minimal buffer"""
        maintenance_margin = 0.003  # 0.3%

        if side == "LONG":
            liquidation_price = entry_price * (1 - 1/self.leverage - maintenance_margin)
        else:
            liquidation_price = entry_price * (1 + 1/self.leverage + maintenance_margin)

        # Minimal buffer
        buffer = liquidation_price * (self.liquidation_buffer / 100)

        if side == "LONG":
            return liquidation_price - buffer
        else:
            return liquidation_price + buffer


# Example usage
if __name__ == "__main__":
    config = AggressiveMicroConfig()

    print("🚀 AGGRESSIVE MICRO-TRADING CONFIG")
    print("="*60)
    print(f"Starting Balance: {config.initial_balance} USDT")
    print(f"Target: {config.target_balance} USDT")
    print(f"Leverage: {config.leverage}x")
    print(f"Risk per trade: {config.risk_per_trade*100}%")
    print(f"Stop Loss: {config.stop_loss_pct}%")
    print(f"Take Profit: {config.take_profit_pct}%")
    print(f"Compounding threshold: {config.compound_threshold} USDT")
    print(f"Grid levels: {config.grid_levels}")
    print("="*60)
    print("\n⚠️  WARNING: ULTRA-HIGH RISK STRATEGY ⚠️")
    print("This config can grow fast BUT can also lose fast!")
    print("Only use with money you can afford to lose completely.")
    print("="*60)
