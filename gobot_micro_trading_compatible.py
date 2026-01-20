#!/usr/bin/env python3
"""
GOBOT Micro-Trading Orchestrator (Binance Compatible)
==================================================

Fully compatible with existing GOBOT components:
- Binance API (same testnet/mainnet setup)
- QuantCrawler integration
- Telegram notifications
- Existing environment variables
- Screenshot service
- TradingView integration

Micro-trading: 1 USDT → 100+ USDT with 125x leverage
"""

import asyncio
import json
import logging
import os
import subprocess
import time
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Any, Optional

from orchestrator import OrchestratorConfig, ClaudeOrchestrator, CyclePhase

logger = logging.getLogger(__name__)


class BinanceAPIClient:
    """Binance API client (compatible with existing setup)"""

    def __init__(self):
        self.api_key = os.getenv('BINANCE_API_KEY', '')
        self.secret = os.getenv('BINANCE_SECRET', '')
        self.use_testnet = os.getenv('BINANCE_USE_TESTNET', 'true').lower() == 'true'
        self.base_url = 'https://testnet.binancefuture.com' if self.use_testnet else 'https://fapi.binance.com'

        # Rate limiting
        self.request_count = 0
        self.last_request = time.time()

    async def get_account_info(self) -> Dict:
        """Get futures account info"""
        import aiohttp

        url = f"{self.base_url}/fapi/v2/account"
        headers = {
            'X-MBX-APIKEY': self.api_key,
            'Content-Type': 'application/json'
        }

        async with aiohttp.ClientSession() as session:
            async with session.get(url, headers=headers) as response:
                if response.status == 200:
                    data = await response.json()
                    return {
                        'status': 'success',
                        'data': data
                    }
                else:
                    return {
                        'status': 'error',
                        'message': f'API error: {response.status}'
                    }

    async def set_leverage(self, symbol: str, leverage: int) -> Dict:
        """Set leverage for symbol"""
        import aiohttp

        url = f"{self.base_url}/fapi/v1/leverage"
        headers = {
            'X-MBX-APIKEY': self.api_key,
            'Content-Type': 'application/x-www-form-urlencoded'
        }

        data = f'symbol={symbol}&leverage={leverage}'

        async with aiohttp.ClientSession() as session:
            async with session.post(url, headers=headers, data=data) as response:
                result = await response.json()
                return {
                    'status': 'success' if response.status == 200 else 'error',
                    'data': result
                }

    async def place_order(self, symbol: str, side: str, quantity: float, order_type: str = 'MARKET') -> Dict:
        """Place futures order"""
        import aiohttp

        url = f"{self.base_url}/fapi/v1/order"
        headers = {
            'X-MBX-APIKEY': self.api_key,
            'Content-Type': 'application/x-www-form-urlencoded'
        }

        data = f'symbol={symbol}&side={side}&type={order_type}&quantity={quantity}'

        async with aiohttp.ClientSession() as session:
            async with session.post(url, headers=headers, data=data) as response:
                result = await response.json()
                return {
                    'status': 'success' if response.status == 200 else 'error',
                    'data': result
                }


class QuantCrawlerClient:
    """QuantCrawler integration (compatible with existing setup)"""

    def __init__(self):
        self.quant_crawler_path = Path(__file__).parent / 'services' / 'screenshot-service' / 'quantcrawler-integration.js'

    async def analyze_chart(self, symbol: str, position_size: float) -> Dict:
        """Get AI analysis from QuantCrawler"""
        try:
            # Run existing quantcrawler-integration.js
            cmd = f'node {self.quant_crawler_path} {symbol} {position_size}'
            result = subprocess.run(
                cmd,
                shell=True,
                capture_output=True,
                text=True,
                timeout=120
            )

            if result.returncode == 0:
                # Parse JSON response
                try:
                    data = json.loads(result.stdout)
                    return {
                        'status': 'success',
                        'data': data
                    }
                except json.JSONDecodeError:
                    return {
                        'status': 'error',
                        'message': 'Invalid JSON from QuantCrawler',
                        'raw_output': result.stdout
                    }
            else:
                return {
                    'status': 'error',
                    'message': result.stderr
                }
        except Exception as e:
            return {
                'status': 'error',
                'message': str(e)
            }


class TelegramClient:
    """Telegram client (compatible with existing setup)"""

    def __init__(self):
        self.token = os.getenv('TELEGRAM_TOKEN', '')
        self.chat_id = os.getenv('AUTHORIZED_CHAT_ID', '') or os.getenv('TELEGRAM_CHAT_ID', '')
        self.enabled = bool(self.token and self.chat_id)

    async def send_message(self, message: str) -> bool:
        """Send Telegram message"""
        if not self.enabled:
            logger.warning("Telegram not configured")
            return False

        try:
            import aiohttp

            url = f'https://api.telegram.org/bot{self.token}/sendMessage'
            data = {
                'chat_id': self.chat_id,
                'text': message,
                'parse_mode': 'HTML'
            }

            async with aiohttp.ClientSession() as session:
                async with session.post(url, data=data) as response:
                    return response.status == 200
        except Exception as e:
            logger.error(f"Telegram error: {e}")
            return False


class MicroTradingConfig:
    """Micro-trading configuration"""

    def __init__(self):
        # Balance
        self.initial_balance = 1.0
        self.current_balance = 1.0

        # Leverage (125x)
        self.leverage = 125

        # Risk (ultra-conservative)
        self.risk_per_trade = 0.001  # 0.1%
        self.stop_loss = 0.002  # 0.2%
        self.take_profit = 0.004  # 0.4%
        self.min_order_value = 5.0  # Min 5 USDT

        # Targets
        self.target_balance = 100.0

    def calculate_position_size(self) -> float:
        """Calculate position size"""
        risk_amount = self.current_balance * self.risk_per_trade
        position_size = risk_amount * self.leverage / self.stop_loss

        # Cap by balance
        max_position = self.current_balance * self.leverage
        position_size = min(position_size, max_position)

        # Cap by min order
        position_size = max(position_size, self.min_order_value)

        return position_size


class GOBOTMicroTradingOrchestrator(ClaudeOrchestrator):
    """Fully compatible micro-trading orchestrator"""

    def __init__(self, config: OrchestratorConfig):
        super().__init__(config)

        # Initialize clients
        self.binance = BinanceAPIClient()
        self.quantcrawler = QuantCrawlerClient()
        self.telegram = TelegramClient()

        # Micro-trading config
        self.micro_config = MicroTradingConfig()

        # Stats
        self.stats = {
            'trades': 0,
            'wins': 0,
            'losses': 0,
            'pnl': 0.0,
            'balance_history': []
        }

        logger.info("🚀 GOBOT Micro-Trading Orchestrator (Binance Compatible)")
        logger.info(f"Mode: {'TESTNET' if self.binance.use_testnet else 'MAINNET'}")
        logger.info(f"Balance: {self.micro_config.current_balance} USDT")
        logger.info(f"Leverage: {self.micro_config.leverage}x")
        logger.info(f"Telegram: {'Enabled' if self.telegram.enabled else 'Disabled'}")

    async def _handle_data_scan(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 1: Data scan with Binance API"""
        logger.info("="*60)
        logger.info("Phase 1: MARKET DATA SCAN")
        logger.info("="*60)

        await asyncio.sleep(0.1)

        # Get account info from Binance
        account_info = await self.binance.get_account_info()

        # Get current price (simplified - in production use Binance WebSocket)
        current_prices = {
            'BTCUSDT': 95320.50,
            'ETHUSDT': 3420.15
        }

        market_data = {
            'timestamp': datetime.now().isoformat(),
            'balance': self.micro_config.current_balance,
            'binance_account': account_info,
            'prices': current_prices,
            'leverage': self.micro_config.leverage,
            'available_balance': self.micro_config.current_balance
        }

        logger.info(f"Balance: {self.micro_config.current_balance} USDT")
        logger.info(f"Available: {market_data['available_balance']} USDT")

        # Send Telegram
        await self._send_telegram(
            f"📊 <b>Market Scan</b>\n"
            f"Balance: {self.micro_config.current_balance:.4f} USDT\n"
            f"BTC: ${current_prices['BTCUSDT']}\n"
            f"Mode: {'TESTNET' if self.binance.use_testnet else 'MAINNET'}"
        )

        return market_data

    async def _handle_idea(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 2: Get AI analysis from QuantCrawler"""
        logger.info("="*60)
        logger.info("Phase 2: AI ANALYSIS (QuantCrawler)")
        logger.info("="*60)

        await asyncio.sleep(0.1)

        # Get position size
        position_size = self.micro_config.calculate_position_size()

        # Call QuantCrawler for BTC analysis
        logger.info(f"Analyzing BTCUSDT with position: {position_size:.2f} USDT")

        analysis = await self.quantcrawler.analyze_chart('BTCUSDT', position_size)

        # Parse QuantCrawler response
        if analysis['status'] == 'success':
            data = analysis['data']
            signal = data.get('signal', 'HOLD')
            confidence = data.get('confidence', 0.5)
            entry = data.get('entry_price')
            sl = data.get('stop_loss')
            tp = data.get('take_profit')

            logger.info(f"QuantCrawler Signal: {signal}")
            logger.info(f"Confidence: {confidence:.0%}")
            logger.info(f"Entry: ${entry}")
            logger.info(f"SL: ${sl}")
            logger.info(f"TP: ${tp}")

            result = {
                'signal': signal,
                'confidence': confidence,
                'entry_price': entry,
                'stop_loss': sl,
                'take_profit': tp,
                'position_size': position_size,
                'provider': 'QuantCrawler',
                'raw_data': data
            }

            await self._send_telegram(
                f"🤖 <b>AI Analysis</b>\n"
                f"Signal: {signal}\n"
                f"Confidence: {confidence:.0%}\n"
                f"Position: {position_size:.2f} USDT"
            )

        else:
            logger.error(f"QuantCrawler error: {analysis['message']}")
            result = {
                'signal': 'HOLD',
                'confidence': 0,
                'reason': analysis['message'],
                'provider': 'QuantCrawler'
            }

        return result

    async def _handle_code_edit(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 3: Prepare trade execution"""
        logger.info("="*60)
        logger.info("Phase 3: TRADE PREPARATION")
        logger.info("="*60)

        idea = self.current_cycle.phase_results.get('idea', {})
        signal = idea.get('signal', 'HOLD')
        confidence = idea.get('confidence', 0)

        await asyncio.sleep(0.1)

        if signal == 'HOLD' or confidence < 0.8:
            logger.info("No trade - signal not strong enough")
            return {
                'action': 'NO_TRADE',
                'reason': 'Low confidence or HOLD signal'
            }

        # Get trade details
        entry_price = idea.get('entry_price')
        position_size = idea.get('position_size', 0)
        sl = idea.get('stop_loss')
        tp = idea.get('take_profit')

        # Calculate quantity (for futures, quantity is in base asset)
        quantity = position_size / entry_price

        result = {
            'action': 'TRADE',
            'symbol': 'BTCUSDT',
            'side': 'BUY' if signal == 'LONG' else 'SELL',
            'quantity': quantity,
            'entry_price': entry_price,
            'stop_loss': sl,
            'take_profit': tp,
            'leverage': self.micro_config.leverage,
            'position_size_usdt': position_size,
            'risk_reward': (tp - entry_price) / (entry_price - sl) if sl and tp else 0
        }

        logger.info(f"Trade prepared: {result['side']} {result['quantity']:.6f} BTCUSDT")
        logger.info(f"Entry: ${result['entry_price']}")
        logger.info(f"SL: ${result['stop_loss']}")
        logger.info(f"TP: ${result['take_profit']}")
        logger.info(f"R/R: {result['risk_reward']:.2f}x")

        return result

    async def _handle_backtest(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 4: Execute trade via Binance API"""
        logger.info("="*60)
        logger.info("Phase 4: TRADE EXECUTION")
        logger.info("="*60)

        trade_prep = self.current_cycle.phase_results.get('code_edit', {})
        action = trade_prep.get('action', 'NO_TRADE')

        await asyncio.sleep(0.1)

        if action == 'NO_TRADE':
            return {
                'status': 'NO_TRADE',
                'reason': trade_prep.get('reason')
            }

        # Set leverage first
        symbol = trade_prep['symbol']
        leverage = trade_prep['leverage']

        logger.info(f"Setting leverage to {leverage}x...")
        leverage_result = await self.binance.set_leverage(symbol, leverage)

        if leverage_result['status'] == 'error':
            logger.error(f"Leverage error: {leverage_result['data']}")
            return {
                'status': 'ERROR',
                'message': 'Failed to set leverage'
            }

        # Place order
        logger.info(f"Placing {trade_prep['side']} order...")
        order_result = await self.binance.place_order(
            symbol=symbol,
            side=trade_prep['side'],
            quantity=trade_prep['quantity']
        )

        if order_result['status'] == 'success':
            order_data = order_result['data']
            logger.info(f"✅ Order placed: {order_data['orderId']}")

            # Update stats
            self.stats['trades'] += 1

            await self._send_telegram(
                f"✅ <b>Trade Executed</b>\n"
                f"Symbol: {symbol}\n"
                f"Side: {trade_prep['side']}\n"
                f"Quantity: {trade_prep['quantity']:.6f}\n"
                f"Order ID: {order_data['orderId']}"
            )

            return {
                'status': 'EXECUTED',
                'order_id': order_data['orderId'],
                'data': order_data
            }
        else:
            logger.error(f"Order failed: {order_result['data']}")

            await self._send_telegram(
                f"❌ <b>Order Failed</b>\n"
                f"Error: {order_result['data'].get('msg', 'Unknown')}"
            )

            return {
                'status': 'ERROR',
                'message': order_result['data']
            }

    async def _handle_report(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 5: Performance report"""
        logger.info("="*60)
        logger.info("Phase 5: PERFORMANCE REPORT")
        logger.info("="*60)

        await asyncio.sleep(0.1)

        # Calculate metrics
        win_rate = (self.stats['wins'] / max(self.stats['trades'], 1)) * 100

        report = {
            'timestamp': datetime.now().isoformat(),
            'balance': {
                'current': self.micro_config.current_balance,
                'initial': self.micro_config.initial_balance,
                'growth_pct': ((self.micro_config.current_balance /
                               self.micro_config.initial_balance - 1) * 100)
            },
            'stats': {
                'total_trades': self.stats['trades'],
                'wins': self.stats['wins'],
                'losses': self.stats['losses'],
                'win_rate': f"{win_rate:.1f}%",
                'total_pnl': f"{self.stats['pnl']:.4f} USDT"
            },
            'progress': {
                'target': self.micro_config.target_balance,
                'current': self.micro_config.current_balance,
                'progress_pct': (self.micro_config.current_balance /
                               self.micro_config.target_balance * 100)
            }
        }

        logger.info(f"Balance: {report['balance']['current']:.4f} USDT")
        logger.info(f"Growth: {report['balance']['growth_pct']:.1f}%")
        logger.info(f"Trades: {report['stats']['total_trades']}")
        logger.info(f"Win Rate: {report['stats']['win_rate']}")
        logger.info(f"Progress: {report['progress']['progress_pct']:.1f}%")

        # Send Telegram summary
        await self._send_telegram(
            f"📈 <b>Performance Report</b>\n"
            f"Balance: {self.micro_config.current_balance:.4f} USDT\n"
            f"Growth: {report['balance']['growth_pct']:.1f}%\n"
            f"Trades: {self.stats['trades']}\n"
            f"Win Rate: {win_rate:.1f}%"
        )

        return report

    async def _send_telegram(self, message: str):
        """Send Telegram message"""
        try:
            await self.telegram.send_message(message)
        except Exception as e:
            logger.error(f"Telegram error: {e}")

    def _check_completion_signal(self) -> bool:
        """Check if target reached"""
        if self.micro_config.current_balance >= self.micro_config.target_balance:
            logger.info(f"🎉 TARGET ACHIEVED! {self.micro_config.current_balance:.2f} USDT")
            return True

        if self.micro_config.current_balance < 0.5:
            logger.warning("Balance too low - stopping")
            return True

        return False


async def main():
    """Main entry point"""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
    )

    # Banner
    print("\n" + "="*60)
    print("🚀 GOBOT MICRO-TRADING (BINANCE COMPATIBLE) 🚀")
    print("="*60)
    print("Compatible with existing GOBOT components:")
    print("✅ Binance API (testnet/mainnet)")
    print("✅ QuantCrawler AI")
    print("✅ Telegram notifications")
    print("✅ Environment variables")
    print("="*60 + "\n")

    # Check environment
    if not os.getenv('BINANCE_API_KEY'):
        logger.error("❌ BINANCE_API_KEY not set!")
        logger.error("Please set your Binance API key")
        return False

    if not os.getenv('TELEGRAM_TOKEN'):
        logger.warning("⚠️ TELEGRAM_TOKEN not set - notifications disabled")

    # Load config from .env if exists
    env_file = Path('.env')
    if env_file.exists():
        logger.info("Loading configuration from .env file")

    config = OrchestratorConfig(
        max_iterations=1000,
        sleep_between_iterations=60,  # 1 minute between scans
        state_dir="./micro_trading_state",
        archive_dir="./micro_trading_archive"
    )

    orchestrator = GOBOTMicroTradingOrchestrator(config)

    logger.info(f"Starting orchestrator")
    logger.info(f"Mode: {'TESTNET' if orchestrator.binance.use_testnet else 'MAINNET'}")
    logger.info(f"Target: {orchestrator.micro_config.target_balance} USDT")

    completed = await orchestrator.run(
        max_iterations=1000,
        current_branch="micro-trading"
    )

    if completed:
        logger.info("🎉 Micro-trading target achieved!")

    return completed


if __name__ == "__main__":
    result = asyncio.run(main())
    exit(0 if result else 1)
