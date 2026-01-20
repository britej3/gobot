#!/usr/bin/env python3
"""
Ralph-Inspired Claude Orchestrator
==================================

Cycles Claude through: data scan → idea → code edit → backtest → report

Implements Ralph's control patterns plus enhanced features:
- Exit detection with completion signals
- Circuit breakers for API resilience
- Rate limiting with adaptive backoff
- State persistence and archival
"""

import json
import time
import asyncio
import logging
from datetime import datetime, timedelta
from dataclasses import dataclass, asdict
from typing import Dict, List, Optional, Callable, Any
from pathlib import Path
from enum import Enum
import hashlib


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
)
logger = logging.getLogger(__name__)


class CyclePhase(Enum):
    """Orchestration cycle phases"""
    DATA_SCAN = "data_scan"
    IDEA = "idea"
    CODE_EDIT = "code_edit"
    BACKTEST = "backtest"
    REPORT = "report"
    COMPLETE = "complete"


@dataclass
class OrchestratorConfig:
    """Configuration for the orchestrator"""
    max_iterations: int = 10
    sleep_between_iterations: float = 2.0
    state_dir: str = "./orchestrator_state"
    archive_dir: str = "./orchestrator_archive"
    branch_file: str = "branch.json"
    progress_file: str = "progress.json"
    prd_file: str = "prd.json"

    # Circuit breaker settings
    circuit_breaker_failure_threshold: int = 5
    circuit_breaker_recovery_timeout: int = 60
    circuit_breaker_expected_exception: type = Exception

    # Rate limiting settings
    requests_per_minute: int = 60
    requests_per_hour: int = 1000


@dataclass
class CycleState:
    """State of a single cycle"""
    cycle_id: str
    start_time: datetime
    end_time: Optional[datetime] = None
    current_phase: CyclePhase = CyclePhase.DATA_SCAN
    phase_results: Dict[str, Any] = None
    success: bool = False
    error_message: Optional[str] = None

    def __post_init__(self):
        if self.phase_results is None:
            self.phase_results = {}


@dataclass
class CycleResult:
    """Result of a completed cycle"""
    cycle_id: str
    duration_seconds: float
    phases_completed: List[CyclePhase]
    success: bool
    output: str
    learnings: List[str] = None

    def __post_init__(self):
        if self.learnings is None:
            self.learnings = []


class CircuitBreaker:
    """
    Circuit breaker pattern implementation
    Protects against cascading failures
    """

    def __init__(self, failure_threshold: int = 5, recovery_timeout: int = 60):
        self.failure_threshold = failure_threshold
        self.recovery_timeout = recovery_timeout
        self.failure_count = 0
        self.last_failure_time: Optional[datetime] = None
        self.state = "CLOSED"  # CLOSED, OPEN, HALF_OPEN

    def call(self, func: Callable, *args, **kwargs) -> Any:
        """
        Execute function through circuit breaker
        """
        if self.state == "OPEN":
            if self._should_attempt_reset():
                self.state = "HALF_OPEN"
                logger.info("Circuit breaker transitioning to HALF_OPEN")
            else:
                raise Exception(f"Circuit breaker is OPEN - failing fast")

        try:
            # Check if function is async
            import inspect
            if inspect.iscoroutinefunction(func):
                # This won't work - need to handle async differently
                # For now, raise an error to indicate async handling needed
                raise RuntimeError(
                    "Circuit breaker does not support async functions directly. "
                    "Use 'await execute_phase()' pattern instead."
                )
            result = func(*args, **kwargs)
            self._on_success()
            return result
        except Exception as e:
            self._on_failure()
            raise

    def _should_attempt_reset(self) -> bool:
        """Check if enough time has passed to attempt reset"""
        if self.last_failure_time is None:
            return False
        elapsed = datetime.now() - self.last_failure_time
        return elapsed.total_seconds() >= self.recovery_timeout

    def _on_success(self):
        """Handle successful execution"""
        self.failure_count = 0
        self.state = "CLOSED"

    def _on_failure(self):
        """Handle failed execution"""
        self.failure_count += 1
        self.last_failure_time = datetime.now()

        if self.failure_count >= self.failure_threshold:
            self.state = "OPEN"
            logger.warning(
                f"Circuit breaker OPEN after {self.failure_count} failures"
            )


class RateLimiter:
    """
    Rate limiter with adaptive backoff
    Implements token bucket algorithm
    """

    def __init__(self, requests_per_minute: int = 60, requests_per_hour: int = 1000):
        self.rpm_limit = requests_per_minute
        self.rph_limit = requests_per_hour

        # Token buckets
        self.minute_bucket = requests_per_minute
        self.hour_bucket = requests_per_hour
        self.last_refill = datetime.now()

        # Track requests
        self.request_times: List[datetime] = []

    async def acquire(self):
        """
        Acquire permission to make a request
        Blocks if necessary with adaptive backoff
        """
        self._refill_buckets()
        self._clean_old_requests()

        # Check if we should backoff based on recent failures
        backoff_multiplier = 1.0
        if len(self.request_times) >= 10:
            recent_requests = self.request_times[-10:]
            # If requests are too frequent, increase backoff
            time_span = (recent_requests[-1] - recent_requests[0]).total_seconds()
            if time_span < 60:  # 10 requests in less than 60 seconds
                backoff_multiplier = 2.0
                logger.warning("High request frequency detected, applying backoff")

        # Wait for available tokens
        wait_time = 0.0

        if self.minute_bucket < 1:
            # Need to wait for minute bucket refill
            elapsed_since_refill = (datetime.now() - self.last_refill).total_seconds()
            wait_time = max(wait_time, 60.0 - elapsed_since_refill)

        if self.hour_bucket < 1:
            # Need to wait for hour bucket refill
            elapsed_since_refill = (datetime.now() - self.last_refill).total_seconds()
            wait_time = max(wait_time, 3600.0 - elapsed_since_refill)

        wait_time *= backoff_multiplier

        if wait_time > 0:
            logger.info(f"Rate limiting - waiting {wait_time:.2f}s")
            await asyncio.sleep(wait_time)
            self._refill_buckets()

        # Check again after waiting
        if self.minute_bucket < 1 or self.hour_bucket < 1:
            # Still no tokens, wait a bit more
            await asyncio.sleep(1.0)

        # Consume tokens
        self.minute_bucket = max(0, self.minute_bucket - 1)
        self.hour_bucket = max(0, self.hour_bucket - 1)
        self.request_times.append(datetime.now())

    def _refill_buckets(self):
        """Refill token buckets based on elapsed time"""
        now = datetime.now()
        elapsed = (now - self.last_refill).total_seconds()

        if elapsed > 0:
            # Refill based on elapsed time
            self.minute_bucket = min(
                self.rpm_limit,
                self.minute_bucket + (elapsed / 60.0) * self.rpm_limit
            )
            self.hour_bucket = min(
                self.rph_limit,
                self.hour_bucket + (elapsed / 3600.0) * self.rph_limit
            )
            self.last_refill = now

    def _clean_old_requests(self):
        """Clean requests older than 1 hour"""
        cutoff = datetime.now() - timedelta(hours=1)
        self.request_times = [t for t in self.request_times if t > cutoff]


class StateManager:
    """Manages orchestrator state and archival (Ralph pattern)"""

    def __init__(self, config: OrchestratorConfig):
        self.config = config
        self.state_dir = Path(config.state_dir)
        self.archive_dir = Path(config.archive_dir)
        self.branch_file = self.state_dir / config.branch_file
        self.progress_file = self.state_dir / config.progress_file
        self.prd_file = self.state_dir / config.prd_file

        # Ensure directories exist
        self.state_dir.mkdir(exist_ok=True)
        self.archive_dir.mkdir(exist_ok=True)

    def get_current_branch(self) -> Optional[str]:
        """Get current branch from state"""
        if self.branch_file.exists():
            try:
                data = json.loads(self.branch_file.read_text())
                return data.get("branch")
            except:
                pass
        return None

    def set_current_branch(self, branch: str):
        """Set current branch in state"""
        self.branch_file.write_text(json.dumps({"branch": branch, "timestamp": datetime.now().isoformat()}))

    def should_archive(self, current_branch: str) -> bool:
        """Check if we should archive based on branch change (Ralph pattern)"""
        last_branch = self.get_current_branch()
        return last_branch is not None and last_branch != current_branch

    def archive_current_state(self, current_branch: str):
        """Archive current state when branch changes (Ralph pattern)"""
        if not self.should_archive(current_branch):
            return

        last_branch = self.get_current_branch()
        date_str = datetime.now().strftime("%Y-%m-%d")
        folder_name = last_branch.replace("/", "-")
        archive_folder = self.archive_dir / f"{date_str}-{folder_name}"

        logger.info(f"Archiving previous run: {last_branch} -> {archive_folder}")

        archive_folder.mkdir(exist_ok=True)

        # Copy state files
        for file_path in [self.progress_file, self.prd_file]:
            if file_path.exists():
                dest = archive_folder / file_path.name
                dest.write_text(file_path.read_text())

        # Update current branch
        self.set_current_branch(current_branch)

    def load_progress(self) -> Dict:
        """Load progress from state"""
        if self.progress_file.exists():
            try:
                return json.loads(self.progress_file.read_text())
            except:
                pass
        return {"cycles": [], "patterns": [], "start_time": datetime.now().isoformat()}

    def save_progress(self, progress: Dict):
        """Save progress to state"""
        self.progress_file.write_text(json.dumps(progress, indent=2))

    def add_pattern(self, pattern: str):
        """Add a reusable pattern (Ralph pattern)"""
        progress = self.load_progress()
        if "patterns" not in progress:
            progress["patterns"] = []
        if pattern not in progress["patterns"]:
            progress["patterns"].append(pattern)
            self.save_progress(progress)

    def add_cycle_result(self, result: CycleResult):
        """Add cycle result to progress"""
        progress = self.load_progress()
        if "cycles" not in progress:
            progress["cycles"] = []

        # Convert cycle result to dict, handling enums
        result_dict = asdict(result)
        # Convert enum values to strings
        if "phases_completed" in result_dict:
            result_dict["phases_completed"] = [p.value for p in result_dict["phases_completed"]]

        progress["cycles"].append(result_dict)
        self.save_progress(progress)


class ClaudeOrchestrator:
    """
    Main orchestrator class
    Implements Ralph's patterns with enhancements
    """

    def __init__(self, config: OrchestratorConfig):
        self.config = config
        self.state_manager = StateManager(config)
        self.circuit_breaker = CircuitBreaker(
            failure_threshold=config.circuit_breaker_failure_threshold,
            recovery_timeout=config.circuit_breaker_recovery_timeout
        )
        self.rate_limiter = RateLimiter(
            requests_per_minute=config.requests_per_minute,
            requests_per_hour=config.requests_per_hour
        )
        self.current_cycle: Optional[CycleState] = None
        self.cycles_completed = 0

        # Phase handlers
        self.phase_handlers = {
            CyclePhase.DATA_SCAN: self._handle_data_scan,
            CyclePhase.IDEA: self._handle_idea,
            CyclePhase.CODE_EDIT: self._handle_code_edit,
            CyclePhase.BACKTEST: self._handle_backtest,
            CyclePhase.REPORT: self._handle_report,
        }

    async def run(self, max_iterations: Optional[int] = None, current_branch: str = "main") -> bool:
        """
        Run orchestrator loop
        Returns True if completed, False if max iterations reached
        """
        max_iterations = max_iterations or self.config.max_iterations

        # Ralph pattern: Archive if branch changed
        self.state_manager.archive_current_state(current_branch)
        self.state_manager.set_current_branch(current_branch)

        logger.info(f"Starting orchestrator - Max iterations: {max_iterations}")
        logger.info(f"State dir: {self.state_manager.state_dir}")
        logger.info(f"Archive dir: {self.state_manager.archive_dir}")

        for i in range(1, max_iterations + 1):
            logger.info(f"\n{'='*60}")
            logger.info(f"  Cycle {i} of {max_iterations}")
            logger.info(f"{'='*60}\n")

            # Run single cycle
            success = await self._run_cycle(i)

            if success:
                logger.info(f"\n✓ Orchestrator completed all tasks!")
                logger.info(f"  Completed at cycle {i} of {max_iterations}")
                return True

            # Rate limit between cycles
            await asyncio.sleep(self.config.sleep_between_iterations)

        logger.warning(f"\n✗ Reached max iterations ({max_iterations}) without completion")
        return False

    async def _run_cycle(self, cycle_num: int) -> bool:
        """
        Run a single cycle through all phases
        Returns True if cycle completed successfully
        """
        # Create cycle state
        cycle_id = f"cycle_{cycle_num}_{int(time.time())}"
        self.current_cycle = CycleState(
            cycle_id=cycle_id,
            start_time=datetime.now()
        )

        try:
            # Execute each phase in sequence
            for phase in CyclePhase:
                if phase == CyclePhase.COMPLETE:
                    continue

                logger.info(f"\n--- Phase: {phase.value.upper()} ---")

                self.current_cycle.current_phase = phase

                # Execute phase
                # Apply rate limiting first
                await self.rate_limiter.acquire()

                # Execute the phase
                output = await self.phase_handlers[phase](phase)

                self.current_cycle.phase_results[phase.value] = output

                logger.info(f"Phase {phase.value} completed")

            # Mark cycle as complete
            self.current_cycle.end_time = datetime.now()
            self.current_cycle.success = True

            # Create result
            duration = (self.current_cycle.end_time - self.current_cycle.start_time).total_seconds()
            result = CycleResult(
                cycle_id=cycle_id,
                duration_seconds=duration,
                phases_completed=list(CyclePhase)[:-1],  # All except COMPLETE
                success=True,
                output="Cycle completed successfully",
                learnings=[
                    f"Phase {p.value} executed successfully"
                    for p in CyclePhase
                    if p != CyclePhase.COMPLETE
                ]
            )

            # Save to progress (Ralph pattern)
            self.state_manager.add_cycle_result(result)

            self.cycles_completed += 1

            # Check for completion signal (Ralph pattern)
            if self._check_completion_signal():
                logger.info("\n<promise>COMPLETE</promise>")
                return True

            return False

        except Exception as e:
            self.current_cycle.end_time = datetime.now()
            self.current_cycle.success = False
            self.current_cycle.error_message = str(e)

            logger.error(f"Cycle failed: {e}", exc_info=True)

            # Add to progress
            result = CycleResult(
                cycle_id=cycle_id,
                duration_seconds=0,
                phases_completed=[],
                success=False,
                output=f"Cycle failed: {e}",
                learnings=[f"Error in {self.current_cycle.current_phase.value}: {e}"]
            )
            self.state_manager.add_cycle_result(result)

            return False

    async def _handle_data_scan(self, phase: CyclePhase) -> Dict:
        """Phase 1: Data Scan"""
        logger.info("Scanning data sources...")

        # Simulate data scanning
        await asyncio.sleep(0.5)

        result = {
            "files_scanned": 10,
            "patterns_found": ["REST API", "GraphQL", "WebSocket"],
            "dependencies": ["pandas", "numpy", "requests"],
            "insights": [
                "Codebase uses FastAPI for APIs",
                "Heavy use of async/await patterns",
                "Database migrations in ./migrations/"
            ]
        }

        # Ralph pattern: Add reusable patterns
        for insight in result["insights"]:
            self.state_manager.add_pattern(f"Data scan: {insight}")

        logger.info(f"  Found {result['files_scanned']} files")
        logger.info(f"  Patterns: {', '.join(result['patterns_found'])}")

        return result

    async def _handle_idea(self, phase: CyclePhase) -> Dict:
        """Phase 2: Idea Generation"""
        logger.info("Generating ideas based on data...")

        # Get data from previous phase
        scan_results = self.current_cycle.phase_results.get("data_scan", {})

        # Simulate idea generation
        await asyncio.sleep(0.5)

        ideas = [
            {
                "title": "Optimize database queries",
                "priority": "high",
                "effort": "medium",
                "reasoning": "Found N+1 query patterns in data scan"
            },
            {
                "title": "Add caching layer",
                "priority": "medium",
                "effort": "low",
                "reasoning": "Multiple expensive API calls detected"
            },
            {
                "title": "Implement rate limiting",
                "priority": "high",
                "effort": "low",
                "reasoning": "Circuit breaker already implemented"
            }
        ]

        result = {
            "ideas": ideas,
            "selected_idea": ideas[0],  # First high priority
            "reasoning": "Selected based on high priority and medium effort"
        }

        logger.info(f"  Generated {len(ideas)} ideas")
        logger.info(f"  Selected: {result['selected_idea']['title']}")

        return result

    async def _handle_code_edit(self, phase: CyclePhase) -> Dict:
        """Phase 3: Code Edit"""
        logger.info("Implementing code changes...")

        # Get idea from previous phase
        idea = self.current_cycle.phase_results.get("idea", {}).get("selected_idea", {})

        # Simulate code editing
        await asyncio.sleep(0.5)

        files_changed = [
            "models/user.py",
            "api/endpoints/users.py",
            "tests/test_users.py"
        ]

        result = {
            "idea_title": idea.get("title"),
            "files_changed": files_changed,
            "lines_added": 150,
            "lines_removed": 30,
            "test_coverage": "85%",
            "changes_summary": f"Implemented {idea.get('title')} with full test coverage"
        }

        logger.info(f"  Changed {len(files_changed)} files")
        logger.info(f"  Lines: +{result['lines_added']} / -{result['lines_removed']}")

        return result

    async def _handle_backtest(self, phase: CyclePhase) -> Dict:
        """Phase 4: Backtest"""
        logger.info("Running backtests...")

        # Get code edit from previous phase
        code_edit = self.current_cycle.phase_results.get("code_edit", {})

        # Simulate backtesting
        await asyncio.sleep(0.5)

        result = {
            "tests_run": 15,
            "tests_passed": 15,
            "tests_failed": 0,
            "coverage": "85%",
            "performance": {
                "latency_ms": 45,
                "throughput_rps": 1200,
                "memory_mb": 256
            },
            "status": "PASSED"
        }

        logger.info(f"  Tests: {result['tests_passed']}/{result['tests_run']} passed")
        logger.info(f"  Coverage: {result['coverage']}")
        logger.info(f"  Status: {result['status']}")

        return result

    async def _handle_report(self, phase: CyclePhase) -> Dict:
        """Phase 5: Report"""
        logger.info("Generating report...")

        # Aggregate all phase results
        all_results = self.current_cycle.phase_results

        # Simulate report generation
        await asyncio.sleep(0.5)

        report = {
            "cycle_summary": {
                "data_scan": all_results.get("data_scan", {}),
                "idea": all_results.get("idea", {}),
                "code_edit": all_results.get("code_edit", {}),
                "backtest": all_results.get("backtest", {})
            },
            "metrics": {
                "duration_seconds": (datetime.now() - self.current_cycle.start_time).total_seconds(),
                "success": True,
                "quality_score": 9.2
            },
            "next_steps": [
                "Deploy changes to staging",
                "Monitor production metrics",
                "Plan next iteration"
            ],
            "recommendations": [
                "Consider adding integration tests",
                "Optimize database schema",
                "Implement feature flags"
            ]
        }

        logger.info(f"  Quality score: {report['metrics']['quality_score']}/10")
        logger.info(f"  Next steps: {len(report['next_steps'])} identified")

        return report

    def _check_completion_signal(self) -> bool:
        """
        Check for completion signal (Ralph pattern)
        Looks for <promise>COMPLETE</promise> or all objectives met
        """
        # Check if we have enough completed cycles
        if self.cycles_completed >= 3:  # Example: 3 successful cycles = complete
            return True

        # Check last cycle results
        progress = self.state_manager.load_progress()
        if progress.get("cycles"):
            last_cycle = progress["cycles"][-1]
            if last_cycle.get("success"):
                # Check if backtest passed with high quality
                backtest = last_cycle.get("output", "")
                if "PASSED" in backtest and "9" in str(last_cycle.get("metrics", {}).get("quality_score", "")):
                    return True

        return False


async def main():
    """Main entry point"""
    config = OrchestratorConfig(
        max_iterations=10,
        sleep_between_iterations=2.0,
        requests_per_minute=60,
        requests_per_hour=1000,
        circuit_breaker_failure_threshold=5
    )

    orchestrator = ClaudeOrchestrator(config)
    completed = await orchestrator.run()

    if completed:
        logger.info("✓ Orchestrator completed successfully")
        exit(0)
    else:
        logger.warning("✗ Orchestrator did not complete")
        exit(1)


if __name__ == "__main__":
    asyncio.run(main())
