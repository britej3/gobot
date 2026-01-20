#!/usr/bin/env python3
"""
Example: Basic Orchestrator Usage
================================

Demonstrates basic usage of the Claude orchestrator
"""

import asyncio
import logging
from pathlib import Path

from orchestrator import OrchestratorConfig, ClaudeOrchestrator

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
)
logger = logging.getLogger(__name__)


async def main():
    """Run basic orchestrator example"""

    # Create configuration
    config = OrchestratorConfig(
        max_iterations=5,
        sleep_between_iterations=2.0,
        requests_per_minute=60,
        requests_per_hour=1000,
        circuit_breaker_failure_threshold=5,
        state_dir="./example_state",
        archive_dir="./example_archive"
    )

    logger.info("Starting Basic Orchestrator Example")
    logger.info(f"Config: {config}")

    # Create orchestrator
    orchestrator = ClaudeOrchestrator(config)

    # Run orchestrator
    completed = await orchestrator.run(
        max_iterations=5,
        current_branch="main"
    )

    if completed:
        logger.info("✓ Orchestrator completed successfully!")
    else:
        logger.warning("✗ Orchestrator did not complete")

    return completed


if __name__ == "__main__":
    result = asyncio.run(main())
    exit(0 if result else 1)
