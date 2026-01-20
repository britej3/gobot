#!/usr/bin/env python3
"""
Steve-GOBOT Integration
====================

Steve CLI integration for smoother GOBOT workflow on Intel Mac

Features:
- Native Mac browser control via Steve CLI
- Automated TradingView screenshots
- UI automation for trading
- Screenshot automation
- Window management
- Element detection and interaction

Compatible with:
- GOBOT screenshot service
- TradingView automation
- Micro-trading orchestrator
- Existing GOBOT components
"""

import subprocess
import json
import asyncio
import logging
from typing import Dict, List, Optional, Tuple
from pathlib import Path
import time

logger = logging.getLogger(__name__)


class SteveClient:
    """Steve CLI client for Mac application control"""

    def __init__(self):
        self.steve_path = "/usr/local/bin/steve"
        self.default_timeout = 5

    def run_command(self, command: List[str], timeout: int = None) -> Dict:
        """Run Steve command"""
        timeout = timeout or self.default_timeout

        try:
            result = subprocess.run(
                [self.steve_path] + command,
                capture_output=True,
                text=True,
                timeout=timeout
            )

            return {
                'ok': result.returncode == 0,
                'exit_code': result.returncode,
                'stdout': result.stdout,
                'stderr': result.stderr,
                'output': result.stdout + result.stderr
            }
        except subprocess.TimeoutExpired:
            return {
                'ok': False,
                'error': 'timeout',
                'message': f'Command timed out after {timeout}s'
            }
        except Exception as e:
            return {
                'ok': False,
                'error': 'exception',
                'message': str(e)
            }

    def get_apps(self) -> List[str]:
        """Get list of running applications"""
        result = self.run_command(['apps'])
        if result['ok']:
            # Parse output to extract app names
            lines = result['stdout'].strip().split('\n')
            apps = [line.strip('- ').strip() for line in lines if line.strip()]
            return apps
        return []

    def focus_app(self, app_name: str) -> bool:
        """Focus an application"""
        result = self.run_command(['focus', app_name])
        return result['ok']

    def take_screenshot(self, app: str = None, output_path: str = None) -> bool:
        """Take screenshot via Steve"""
        command = ['screenshot']
        if app:
            command.extend(['--app', app])
        if output_path:
            command.extend(['-o', output_path])
        else:
            # Output to stdout
            pass

        result = self.run_command(command)
        return result['ok']

    def find_element(self, text: str = None, title: str = None, role: str = None) -> Optional[str]:
        """Find element by text/title/role"""
        command = ['find']
        if text:
            command.extend(['--text', text])
        if title:
            command.extend(['--title', title])
        if role:
            command.extend(['--role', role])

        result = self.run_command(command)
        if result['ok'] and result['stdout'].strip():
            # Extract element ID from output
            lines = result['stdout'].strip().split('\n')
            for line in lines:
                if 'ax://' in line:
                    return line.strip()
        return None

    def click_element(self, element_id: str) -> bool:
        """Click element by ID"""
        result = self.run_command(['click', element_id])
        return result['ok']

    def click_text(self, text: str, window: str = None) -> bool:
        """Click element by text"""
        command = ['click', '--text', text]
        if window:
            command.extend(['--window', window])

        result = self.run_command(command)
        return result['ok']

    def type_text(self, text: str, delay: int = 0) -> bool:
        """Type text"""
        command = ['type', text]
        if delay > 0:
            command.extend(['--delay', str(delay)])

        result = self.run_command(command)
        return result['ok']

    def press_key(self, key_combo: str) -> bool:
        """Press key combination"""
        result = self.run_command(['key', key_combo])
        return result['ok']

    def exists(self, text: str = None, title: str = None) -> bool:
        """Check if element exists"""
        command = ['exists']
        if text:
            command.extend(['--text', text])
        if title:
            command.extend(['--title', title])

        result = self.run_command(command)
        return result['ok']

    def wait_for(self, text: str = None, title: str = None, timeout: int = 5) -> bool:
        """Wait for element to appear"""
        command = ['wait', '--timeout', str(timeout)]
        if text:
            command.extend(['--text', text])
        if title:
            command.extend(['--title', title])

        result = self.run_command(command, timeout=timeout + 5)
        return result['ok']

    def get_windows(self, app: str = None) -> List[Dict]:
        """Get list of windows"""
        result = self.run_command(['windows'])
        if result['ok']:
            # Parse window info
            windows = []
            # This would need proper parsing based on Steve's output format
            return windows
        return []


class SteveGOBOTIntegrator:
    """Steve integration for GOBOT"""

    def __init__(self):
        self.steve = SteveClient()
        self.screenshot_dir = Path('/Users/britebrt/GOBOT/services/screenshot-service/screenshots')
        self.screenshot_dir.mkdir(exist_ok=True)

    async def capture_tradingview_chart(self, symbol: str, timeframe: str = '1m') -> Tuple[bool, str]:
        """
        Capture TradingView chart via Steve automation
        """
        logger.info(f"Capturing {symbol} chart on {timeframe}")

        try:
            # 1. Check if browser is running
            apps = self.steve.get_apps()
            browser_found = any('safari' in app.lower() or 'chrome' in app.lower() for app in apps)

            if not browser_found:
                logger.info("No browser found, launching Safari...")
                # Steve can launch apps
                result = self.steve.run_command(['launch', 'com.apple.Safari'])
                if not result['ok']:
                    logger.error("Failed to launch Safari")
                    return False, "Failed to launch browser"

            # 2. Focus Safari
            self.steve.focus_app('Safari')

            # 3. Wait for browser to be ready
            await asyncio.sleep(2)

            # 4. Navigate to TradingView
            tradingview_url = f"https://www.tradingview.com/symbols/{symbol}/"
            self.steve.type_text(tradingview_url)
            self.steve.press_key('return')

            # 5. Wait for page to load
            await asyncio.sleep(5)

            # 6. Wait for chart to appear
            if not self.steve.wait_for('chart', timeout=10):
                logger.warning("Chart not detected, taking screenshot anyway")

            # 7. Take screenshot
            timestamp = int(time.time())
            screenshot_path = self.screenshot_dir / f"{symbol}_{timeframe}_{timestamp}.png"

            if self.steve.take_screenshot(output_path=str(screenshot_path)):
                logger.info(f"Screenshot saved: {screenshot_path}")
                return True, str(screenshot_path)
            else:
                return False, "Failed to take screenshot"

        except Exception as e:
            logger.error(f"Error capturing chart: {e}")
            return False, str(e)

    async def automate_tradingview_analysis(self, symbol: str) -> Dict:
        """
        Automated TradingView analysis workflow
        """
        logger.info(f"Starting automated analysis for {symbol}")

        result = {
            'symbol': symbol,
            'screenshots': [],
            'signals': [],
            'success': False
        }

        try:
            # Capture multiple timeframes
            timeframes = ['1m', '5m', '15m']

            for tf in timeframes:
                success, path = await self.capture_tradingview_chart(symbol, tf)
                if success:
                    result['screenshots'].append({
                        'timeframe': tf,
                        'path': path,
                        'timestamp': int(time.time())
                    })
                    logger.info(f"✓ Captured {symbol} {tf} chart")
                else:
                    logger.warning(f"✗ Failed to capture {symbol} {tf}")

            if result['screenshots']:
                # Use existing GOBOT components to analyze
                from quantcrawler_integration import analyze_with_quantcrawler
                analysis = await analyze_with_quantcrawler(
                    symbol=symbol,
                    screenshots=result['screenshots']
                )
                result['signals'] = analysis
                result['success'] = True

            return result

        except Exception as e:
            logger.error(f"Error in automated analysis: {e}")
            result['error'] = str(e)
            return result

    async def control_browser_for_trading(self, action: str, symbol: str) -> bool:
        """
        Control browser actions for trading
        """
        logger.info(f"Performing browser action: {action} for {symbol}")

        try:
            # Focus browser
            self.steve.focus_app('Safari')

            # Perform action
            if action == 'refresh':
                self.steve.press_key('cmd+r')
            elif action == 'fullscreen':
                self.steve.press_key('cmd+shift+f')
            elif action == 'screenshot':
                screenshot_path = self.screenshot_dir / f"{symbol}_{int(time.time())}.png"
                self.steve.take_screenshot(output_path=str(screenshot_path))
                logger.info(f"Screenshot saved: {screenshot_path}")
            elif action == 'zoom_in':
                self.steve.press_key('cmd+plus')
            elif action == 'zoom_out':
                self.steve.press_key('cmd+minus')

            await asyncio.sleep(1)
            return True

        except Exception as e:
            logger.error(f"Error controlling browser: {e}")
            return False

    async def detect_trading_signals(self, screenshot_path: str) -> Dict:
        """
        Detect trading signals from screenshot
        This would integrate with your existing AI analysis
        """
        logger.info(f"Analyzing screenshot: {screenshot_path}")

        # For now, return mock analysis
        # In production, this would call your QuantCrawler integration
        return {
            'symbol': 'BTCUSDT',
            'signal': 'LONG',
            'confidence': 0.85,
            'entry': 95320.50,
            'stop_loss': 93420.50,
            'take_profit': 97220.50,
            'source': 'Steve + GOBOT',
            'screenshot': screenshot_path
        }

    async def run_steve_enhanced_gobot(self, symbol: str, cycles: int = 5) -> Dict:
        """
        Run GOBOT with Steve enhancement
        """
        logger.info(f"Running Steve-enhanced GOBOT for {symbol} ({cycles} cycles)")

        results = {
            'symbol': symbol,
            'cycles': cycles,
            'completed': 0,
            'screenshots': [],
            'signals': [],
            'errors': []
        }

        for i in range(cycles):
            logger.info(f"\n--- Cycle {i+1}/{cycles} ---")

            try:
                # 1. Capture chart
                success, path = await self.capture_tradingview_chart(symbol)
                if success:
                    results['screenshots'].append(path)

                # 2. Analyze
                signal = await self.detect_trading_signals(path if success else None)
                if signal:
                    results['signals'].append(signal)

                results['completed'] += 1

                # 3. Wait before next cycle
                await asyncio.sleep(5)

            except Exception as e:
                error_msg = f"Cycle {i+1} failed: {e}"
                logger.error(error_msg)
                results['errors'].append(error_msg)

        return results


# Steve-GOBOT Orchestrator Integration
class SteveGOBOTOrchestrator:
    """
    Orchestrator that uses Steve for browser automation
    Integrates with micro-trading system
    """

    def __init__(self):
        self.steve_integrator = SteveGOBOTIntegrator()
        self.steve = SteveClient()

    async def run_steve_micro_trading(self, symbol: str = 'BTCUSDT') -> Dict:
        """
        Run Steve-enhanced micro-trading
        """
        logger.info("Starting Steve-enhanced micro-trading")

        # 1. Setup browser
        logger.info("Setting up browser...")
        self.steve.focus_app('Safari')

        # 2. Capture initial chart
        success, screenshot = await self.steve_integrator.capture_tradingview_chart(symbol)
        if not success:
            return {'success': False, 'error': 'Failed to capture chart'}

        # 3. Analyze with GOBOT
        logger.info("Analyzing with GOBOT...")
        from gobot_micro_trading_compatible import GOBOTMicroTradingOrchestrator

        # This would integrate the screenshot with the micro-trading system
        # For now, return the screenshot path
        return {
            'success': True,
            'screenshot': screenshot,
            'symbol': symbol,
            'ready_for_trading': True
        }


async def main():
    """Test Steve-GOBOT integration"""
    logging.basicConfig(level=logging.INFO)

    integrator = SteveGOBOTIntegrator()

    # Test 1: Check Steve installation
    logger.info("Test 1: Checking Steve installation")
    apps = integrator.steve.get_apps()
    logger.info(f"Running apps: {apps[:5]}")  # Show first 5

    # Test 2: Take screenshot
    logger.info("\nTest 2: Taking screenshot")
    screenshot_path = integrator.screenshot_dir / f"test_{int(time.time())}.png"
    success = integrator.steve.take_screenshot(output_path=str(screenshot_path))
    logger.info(f"Screenshot {'success' if success else 'failed'}: {screenshot_path}")

    # Test 3: Browser control
    logger.info("\nTest 3: Browser control")
    await integrator.control_browser_for_trading('screenshot', 'BTCUSDT')

    # Test 4: Automated analysis
    logger.info("\nTest 4: Automated TradingView analysis")
    result = await integrator.automate_tradingview_analysis('BTCUSDT')
    logger.info(f"Analysis result: {result['success']}")
    logger.info(f"Screenshots captured: {len(result['screenshots'])}")

    logger.info("\n✓ Steve-GOBOT integration tests complete")


if __name__ == "__main__":
    asyncio.run(main())
