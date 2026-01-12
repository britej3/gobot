#!/usr/bin/env python3
"""
Trading Memory System for GOBOT
Wraps SimpleMem for trading-specific memory operations
"""
import sys
import os
import json
from datetime import datetime
from typing import Optional, List, Dict, Any

# Add memory directory to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from main import SimpleMemSystem


class TradingMemory:
    """
    Trading-specific memory system built on SimpleMem
    """
    
    def __init__(self, clear_db: bool = False):
        """Initialize trading memory system"""
        self.system = SimpleMemSystem(
            clear_db=clear_db,
            enable_parallel_processing=True,
            max_parallel_workers=4,
            enable_parallel_retrieval=True,
            max_retrieval_workers=2
        )
    
    def add_trade(
        self,
        symbol: str,
        side: str,
        entry_price: float,
        exit_price: float,
        pnl: float,
        pnl_percent: float,
        leverage: int,
        confidence: float,
        reason: str,
        outcome: str,
        lessons_learned: Optional[str] = None
    ):
        """
        Store a completed trade in memory
        """
        content = (
            f"Executed {side.upper()} trade on {symbol}: "
            f"Entry ${entry_price:.4f}, Exit ${exit_price:.4f}, "
            f"PnL {pnl_percent:+.2f}% ({outcome}). "
            f"Used {leverage}x leverage with {confidence*100:.0f}% confidence. "
            f"Trade reason: {reason}."
        )
        
        if lessons_learned:
            content += f" Lesson learned: {lessons_learned}"
        
        timestamp = datetime.now().isoformat()
        self.system.add_dialogue("TradeExecutor", content, timestamp)
        self.system.finalize()
    
    def add_market_insight(
        self,
        symbol: str,
        timeframe: str,
        observation: str,
        pattern: Optional[str] = None,
        indicators: Optional[Dict[str, float]] = None
    ):
        """
        Store a market observation/insight
        """
        content = f"Market analysis for {symbol} on {timeframe} timeframe: {observation}"
        
        if pattern:
            content += f" Detected pattern: {pattern}."
        
        if indicators:
            indicator_str = ", ".join([f"{k}={v:.2f}" for k, v in indicators.items()])
            content += f" Indicators: {indicator_str}."
        
        timestamp = datetime.now().isoformat()
        self.system.add_dialogue("MarketAnalyzer", content, timestamp)
        self.system.finalize()
    
    def add_strategy_learning(self, learning: str, context: Optional[str] = None):
        """
        Store a strategy performance learning
        """
        content = f"Strategy optimization insight: {learning}"
        if context:
            content += f" Context: {context}"
        
        timestamp = datetime.now().isoformat()
        self.system.add_dialogue("StrategyOptimizer", content, timestamp)
        self.system.finalize()
    
    def add_risk_event(self, event: str, severity: str = "medium"):
        """
        Store a risk event (circuit breaker, drawdown, etc.)
        """
        content = f"Risk event ({severity.upper()}): {event}"
        
        timestamp = datetime.now().isoformat()
        self.system.add_dialogue("RiskManager", content, timestamp)
        self.system.finalize()
    
    def query_similar_trades(self, symbol: str, side: str) -> str:
        """
        Find similar past trades and their outcomes
        """
        question = (
            f"What were the outcomes of previous {side} trades on {symbol}? "
            f"What patterns or conditions led to winning vs losing trades?"
        )
        return self.system.ask(question)
    
    def query_market_patterns(self, symbol: str) -> str:
        """
        Find relevant market patterns for a symbol
        """
        question = (
            f"What market patterns have been observed for {symbol}? "
            f"What conditions typically preceded successful trading opportunities?"
        )
        return self.system.ask(question)
    
    def query_strategy_learnings(self, strategy_type: str = None) -> str:
        """
        Get strategy optimization insights
        """
        if strategy_type:
            question = f"What optimizations and learnings have been recorded for {strategy_type} strategies?"
        else:
            question = "What are the key strategy optimization insights and learnings?"
        return self.system.ask(question)
    
    def query_risk_events(self) -> str:
        """
        Get recent risk events and responses
        """
        question = "What risk events have occurred recently? What triggered them and how were they handled?"
        return self.system.ask(question)
    
    def get_trading_context(self, symbol: str, side: str) -> str:
        """
        Get comprehensive trading context for a potential trade
        """
        question = (
            f"I'm considering a {side} trade on {symbol}. "
            f"What relevant information do you have about: "
            f"1) Previous {side} trades on {symbol} and their outcomes, "
            f"2) Recent market patterns for {symbol}, "
            f"3) Any risk events or warnings related to {symbol}?"
        )
        return self.system.ask(question)
    
    def ask(self, question: str) -> str:
        """
        General-purpose memory query
        """
        return self.system.ask(question)
    
    def get_all_memories(self) -> List[Dict[str, Any]]:
        """
        Get all stored memories (for debugging)
        """
        memories = self.system.get_all_memories()
        return [
            {
                "id": mem.entry_id,
                "content": mem.lossless_restatement,
                "timestamp": mem.timestamp,
                "persons": mem.persons,
                "entities": mem.entities,
                "keywords": mem.keywords,
            }
            for mem in memories
        ]


def main():
    """CLI interface for trading memory"""
    import argparse
    
    parser = argparse.ArgumentParser(description="GOBOT Trading Memory System")
    parser.add_argument("action", choices=[
        "add_trade", "add_insight", "add_learning", "add_risk",
        "query_trades", "query_patterns", "query_learnings", "query_risks",
        "context", "ask", "list"
    ])
    parser.add_argument("--symbol", help="Trading symbol (e.g., BTCUSDT)")
    parser.add_argument("--side", help="Trade side (long/short)")
    parser.add_argument("--entry", type=float, help="Entry price")
    parser.add_argument("--exit", type=float, help="Exit price")
    parser.add_argument("--pnl", type=float, help="PnL amount")
    parser.add_argument("--pnl-pct", type=float, help="PnL percentage")
    parser.add_argument("--leverage", type=int, default=1, help="Leverage used")
    parser.add_argument("--confidence", type=float, default=0.5, help="Confidence (0-1)")
    parser.add_argument("--reason", help="Trade reason")
    parser.add_argument("--outcome", choices=["win", "loss", "breakeven"])
    parser.add_argument("--lesson", help="Lesson learned")
    parser.add_argument("--observation", help="Market observation")
    parser.add_argument("--pattern", help="Detected pattern")
    parser.add_argument("--timeframe", default="1m", help="Timeframe")
    parser.add_argument("--learning", help="Strategy learning")
    parser.add_argument("--event", help="Risk event description")
    parser.add_argument("--severity", default="medium", help="Event severity")
    parser.add_argument("--question", help="Question to ask")
    parser.add_argument("--clear", action="store_true", help="Clear database")
    
    args = parser.parse_args()
    
    memory = TradingMemory(clear_db=args.clear)
    
    if args.action == "add_trade":
        memory.add_trade(
            symbol=args.symbol,
            side=args.side,
            entry_price=args.entry,
            exit_price=args.exit,
            pnl=args.pnl,
            pnl_percent=args.pnl_pct,
            leverage=args.leverage,
            confidence=args.confidence,
            reason=args.reason,
            outcome=args.outcome,
            lessons_learned=args.lesson
        )
        print("Trade added to memory")
    
    elif args.action == "add_insight":
        memory.add_market_insight(
            symbol=args.symbol,
            timeframe=args.timeframe,
            observation=args.observation,
            pattern=args.pattern
        )
        print("Market insight added to memory")
    
    elif args.action == "add_learning":
        memory.add_strategy_learning(args.learning)
        print("Strategy learning added to memory")
    
    elif args.action == "add_risk":
        memory.add_risk_event(args.event, args.severity)
        print("Risk event added to memory")
    
    elif args.action == "query_trades":
        result = memory.query_similar_trades(args.symbol, args.side)
        print(result)
    
    elif args.action == "query_patterns":
        result = memory.query_market_patterns(args.symbol)
        print(result)
    
    elif args.action == "query_learnings":
        result = memory.query_strategy_learnings()
        print(result)
    
    elif args.action == "query_risks":
        result = memory.query_risk_events()
        print(result)
    
    elif args.action == "context":
        result = memory.get_trading_context(args.symbol, args.side)
        print(result)
    
    elif args.action == "ask":
        result = memory.ask(args.question)
        print(result)
    
    elif args.action == "list":
        memories = memory.get_all_memories()
        print(json.dumps(memories, indent=2, default=str))


if __name__ == "__main__":
    main()
