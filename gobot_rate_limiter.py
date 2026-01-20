#!/usr/bin/env python3
"""
GOBOT Adaptive Rate Limiter
===========================

Token bucket rate limiter with backoff for Binance API

Features:
- Requests per minute/hour limits
- Adaptive backoff on 429 errors
- Automatic recovery after outages
- Priority-based request queuing
"""

import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Callable, Any
from enum import Enum
import json

logger = logging.getLogger(__name__)


class RequestPriority(Enum):
    CRITICAL = 1  # Close positions, emergency stops
    HIGH = 2      # Place orders, cancel orders
    MEDIUM = 3    # Check balance, get positions
    LOW = 4       # Historical data, market info


class GOBOTRateLimiter:
    """
    Adaptive rate limiter for GOBOT
    Protects Binance API limits with smart backoff
    """

    def __init__(self):
        # Binance Futures limits (from docs)
        self.requests_per_minute = 1200
        self.requests_per_hour = 100000

        # Token buckets
        self.minute_bucket = self.requests_per_minute
        self.hour_bucket = self.requests_per_hour
        self.last_refill = datetime.now()

        # Request tracking
        self.request_history: List[datetime] = []
        self.error_count = 0
        self.last_error_time: Optional[datetime] = None
        self.backoff_multiplier = 1.0

        # Priority queue
        self.request_queue: List[Dict] = []
        self.max_queue_size = 100

    async def acquire(
        self,
        priority: RequestPriority = RequestPriority.MEDIUM,
        endpoint: str = ""
    ) -> str:
        """
        Acquire permission to make API request
        Returns: request_id for tracking
        """
        request_id = f"{endpoint}_{datetime.now().strftime('%Y%m%d_%H%M%S_%f')}"

        # Add to priority queue
        request = {
            "id": request_id,
            "priority": priority,
            "endpoint": endpoint,
            "timestamp": datetime.now(),
            "retries": 0
        }

        self.request_queue.append(request)
        self.request_queue.sort(key=lambda x: x["priority"].value)

        # Wait for slot
        await self._wait_for_slot(request_id)

        return request_id

    async def _wait_for_slot(self, request_id: str):
        """Wait for available rate limit slot"""
        max_wait = 30  # seconds

        start_time = datetime.now()

        while (datetime.now() - start_time).total_seconds() < max_wait:
            self._refill_buckets()

            # Check if request is at front of queue
            if not self.request_queue or self.request_queue[0]["id"] != request_id:
                await asyncio.sleep(0.1)
                continue

            # Check buckets
            if self.minute_bucket > 0 and self.hour_bucket > 0:
                # Consume tokens
                self.minute_bucket -= 1
                self.hour_bucket -= 1
                self.request_history.append(datetime.now())

                # Remove from queue
                self.request_queue = [r for r in self.request_queue if r["id"] != request_id]

                logger.debug(f"Request {request_id} approved")
                return

            # Calculate wait time
            wait_time = 1.0
            if self.minute_bucket <= 0:
                wait_time = max(wait_time, 60 - (datetime.now() - self.last_refill).total_seconds())
            if self.hour_bucket <= 0:
                wait_time = max(wait_time, 3600 - (datetime.now() - self.last_refill).total_seconds())

            # Apply backoff multiplier
            wait_time *= self.backoff_multiplier

            logger.debug(f"Rate limited, waiting {wait_time:.2f}s")
            await asyncio.sleep(wait_time)

        raise TimeoutError(f"Request {request_id} timed out after {max_wait}s")

    def _refill_buckets(self):
        """Refill token buckets based on elapsed time"""
        now = datetime.now()
        elapsed = (now - self.last_refill).total_seconds()

        if elapsed > 0:
            # Refill based on elapsed time
            self.minute_bucket = min(
                self.requests_per_minute,
                self.minute_bucket + (elapsed / 60.0) * self.requests_per_minute
            )
            self.hour_bucket = min(
                self.requests_per_hour,
                self.hour_bucket + (elapsed / 3600.0) * self.requests_per_hour
            )
            self.last_refill = now

    async def handle_error(
        self,
        request_id: str,
        status_code: int,
        error_message: str
    ):
        """Handle API errors with adaptive backoff"""
        self.error_count += 1
        self.last_error_time = datetime.now()

        if status_code == 429:  # Rate limit exceeded
            logger.warning(f"Rate limit exceeded for {request_id}: {error_message}")
            self._increase_backoff()
        elif status_code >= 500:  # Server error
            logger.warning(f"Server error for {request_id}: {error_message}")
            self._increase_backoff()
        else:  # Client error
            logger.error(f"Client error for {request_id}: {error_message}")
            self.backoff_multiplier = 1.0  # Reset

    def _increase_backoff(self):
        """Increase backoff multiplier"""
        self.backoff_multiplier = min(
            10.0,  # Max 10x backoff
            self.backoff_multiplier * 1.5
        )
        logger.warning(f"Backoff increased to {self.backoff_multiplier}x")

    async def handle_success(self, request_id: str):
        """Handle successful request"""
        if self.backoff_multiplier > 1.0:
            self.backoff_multiplier = max(
                1.0,
                self.backoff_multiplier * 0.9
            )
            logger.info(f"Backoff decreased to {self.backoff_multiplier}x")

    def get_status(self) -> Dict:
        """Get rate limiter status"""
        return {
            "buckets": {
                "minute": {
                    "available": self.minute_bucket,
                    "limit": self.requests_per_minute,
                    "usage": f"{((self.requests_per_minute - self.minute_bucket) / self.requests_per_minute * 100):.1f}%"
                },
                "hour": {
                    "available": self.hour_bucket,
                    "limit": self.requests_per_hour,
                    "usage": f"{((self.requests_per_hour - self.hour_bucket) / self.requests_per_hour * 100):.1f}%"
                }
            },
            "backoff_multiplier": self.backoff_multiplier,
            "error_count": self.error_count,
            "queue_size": len(self.request_queue),
            "recent_requests": len([
                t for t in self.request_history
                if (datetime.now() - t).total_seconds() < 60
            ])
        }


# Example: Simulating GOBOT trading loop
async def simulate_gobot_trading():
    """Simulate GOBOT trading with rate limiting"""
    logging.basicConfig(level=logging.INFO)

    rate_limiter = GOBOTRateLimiter()

    logger.info("="*60)
    logger.info("GOBOT Trading Loop Simulation")
    logger.info("="*60)

    # Simulate trading decisions
    trading_actions = [
        ("get_account_info", RequestPriority.HIGH),
        ("place_order", RequestPriority.HIGH),
        ("get_order_status", RequestPriority.MEDIUM),
        ("get_market_data", RequestPriority.LOW),
        ("get_balance", RequestPriority.MEDIUM),
        ("cancel_order", RequestPriority.CRITICAL),
        ("get_historical_klines", RequestPriority.LOW),
        ("get_position_info", RequestPriority.HIGH),
    ]

    results = []

    for action, priority in trading_actions:
        logger.info(f"\nAction: {action} (priority: {priority.name})")

        try:
            # Acquire rate limit slot
            request_id = await rate_limiter.acquire(
                priority=priority,
                endpoint=action
            )

            # Simulate API call
            await asyncio.sleep(0.1)  # Simulate network delay

            # Simulate response
            success = True  # or False for errors
            status_code = 200 if success else 429

            if success:
                await rate_limiter.handle_success(request_id)
                logger.info(f"✓ {action} succeeded")
                results.append({"action": action, "status": "success"})
            else:
                await rate_limiter.handle_error(request_id, status_code, "Rate limit exceeded")
                logger.warning(f"✗ {action} failed: rate limited")
                results.append({"action": action, "status": "rate_limited"})

        except Exception as e:
            logger.error(f"✗ {action} error: {e}")
            results.append({"action": action, "status": "error"})

    # Print final status
    logger.info("\n" + "="*60)
    logger.info("RATE LIMITER STATUS")
    logger.info("="*60)
    status = rate_limiter.get_status()
    print(json.dumps(status, indent=2))

    logger.info(f"\nProcessed {len(results)} actions")
    for result in results:
        logger.info(f"  {result['action']}: {result['status']}")


if __name__ == "__main__":
    asyncio.run(simulate_gobot_trading())
