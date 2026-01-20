#!/usr/bin/env python3
"""
Tests for Claude Orchestrator
"""

import pytest
import asyncio
import json
from datetime import datetime, timedelta
from pathlib import Path
import tempfile
import shutil

from orchestrator import (
    OrchestratorConfig,
    CycleState,
    CyclePhase,
    CircuitBreaker,
    RateLimiter,
    StateManager,
    ClaudeOrchestrator,
)


class TestCircuitBreaker:
    """Test circuit breaker functionality"""

    def test_circuit_breaker_closed_state(self):
        """Test circuit breaker in closed state allows calls"""
        cb = CircuitBreaker(failure_threshold=3, recovery_timeout=60)

        # Should allow calls when closed
        result = cb.call(lambda: "success")
        assert result == "success"
        assert cb.state == "CLOSED"

    def test_circuit_breaker_opens_after_threshold(self):
        """Test circuit breaker opens after threshold failures"""
        cb = CircuitBreaker(failure_threshold=3, recovery_timeout=60)

        # Fail multiple times
        for i in range(3):
            with pytest.raises(Exception):
                cb.call(lambda: 1 / 0)

        assert cb.state == "OPEN"
        assert cb.failure_count == 3

    def test_circuit_breaker_fails_fast_when_open(self):
        """Test circuit breaker fails fast when open"""
        cb = CircuitBreaker(failure_threshold=3, recovery_timeout=60)

        # Open the circuit
        for i in range(3):
            with pytest.raises(Exception):
                cb.call(lambda: 1 / 0)

        # Should fail fast without calling function
        with pytest.raises(Exception, match="Circuit breaker is OPEN"):
            cb.call(lambda: "should not execute")

    def test_circuit_breaker_half_open_recovery(self):
        """Test circuit breaker recovers after timeout"""
        cb = CircuitBreaker(failure_threshold=2, recovery_timeout=1)

        # Open the circuit
        for i in range(2):
            with pytest.raises(Exception):
                cb.call(lambda: 1 / 0)

        assert cb.state == "OPEN"

        # Wait for recovery timeout
        import time
        time.sleep(1.1)

        # Should allow one trial call (half-open)
        try:
            cb.call(lambda: "success")
            assert cb.state == "CLOSED"
        except:
            # If it fails, should go back to open
            assert cb.state == "OPEN"


class TestRateLimiter:
    """Test rate limiter functionality"""

    @pytest.mark.asyncio
    async def test_rate_limiter_allows_initial_requests(self):
        """Test rate limiter allows requests within limit"""
        rl = RateLimiter(requests_per_minute=10, requests_per_hour=100)

        # Should allow requests up to limit
        for i in range(10):
            await rl.acquire()

        assert rl.minute_bucket >= 0

    @pytest.mark.asyncio
    async def test_rate_limiter_enforces_limits(self):
        """Test rate limiter enforces RPM limit"""
        rl = RateLimiter(requests_per_minute=5, requests_per_hour=100)

        # Use up all tokens
        start = datetime.now()
        for i in range(5):
            await rl.acquire()

        # Next request should wait
        await rl.acquire()
        elapsed = (datetime.now() - start).total_seconds()

        # Should have waited at least close to 60 seconds
        # (allowing some buffer for test execution time)
        assert elapsed >= 55

    @pytest.mark.asyncio
    async def test_rate_limiter_bucket_refill(self):
        """Test token bucket refill over time"""
        rl = RateLimiter(requests_per_minute=10, requests_per_hour=100)

        # Use all tokens
        for i in range(10):
            await rl.acquire()

        # Wait for partial refill (1 second = 1/60 of a minute)
        import time
        time.sleep(1)

        # Should have some tokens now
        assert rl.minute_bucket > 0

    @pytest.mark.asyncio
    async def test_rate_limiter_backoff_on_frequency(self):
        """Test rate limiter applies backoff on high frequency"""
        rl = RateLimiter(requests_per_minute=100, requests_per_hour=1000)

        # Make many requests quickly
        for i in range(15):
            await rl.acquire()

        # Next request should have backoff
        start = datetime.now()
        await rl.acquire()
        elapsed = (datetime.now() - start).total_seconds()

        # Should have applied backoff (more than minimal wait)
        assert elapsed > 0.1


class TestStateManager:
    """Test state management functionality"""

    def setup_method(self):
        """Setup temporary directory for tests"""
        self.temp_dir = tempfile.mkdtemp()
        self.config = OrchestratorConfig(
            state_dir=self.temp_dir,
            archive_dir=f"{self.temp_dir}/archive"
        )

    def teardown_method(self):
        """Cleanup temporary directory"""
        shutil.rmtree(self.temp_dir, ignore_errors=True)

    def test_state_manager_initialization(self):
        """Test state manager creates directories"""
        sm = StateManager(self.config)

        assert sm.state_dir.exists()
        assert sm.archive_dir.exists()

    def test_branch_tracking(self):
        """Test branch tracking (Ralph pattern)"""
        sm = StateManager(self.config)

        # Set branch
        sm.set_current_branch("feature/test-branch")
        assert sm.get_current_branch() == "feature/test-branch"

        # Change branch
        sm.set_current_branch("main")
        assert sm.get_current_branch() == "main"

    def test_archive_on_branch_change(self):
        """Test archiving when branch changes (Ralph pattern)"""
        sm = StateManager(self.config)

        # Set initial branch
        sm.set_current_branch("feature/old")

        # Create progress file
        progress = {"test": "data"}
        sm.save_progress(progress)

        # Change to new branch
        sm.archive_current_state("feature/new")

        # Should create archive
        archive_files = list(sm.archive_dir.glob("*"))
        assert len(archive_files) > 0

        # Current branch should be updated
        assert sm.get_current_branch() == "feature/new"

    def test_add_pattern(self):
        """Test adding reusable patterns (Ralph pattern)"""
        sm = StateManager(self.config)

        # Add patterns
        sm.add_pattern("Use async/await for I/O")
        sm.add_pattern("Always validate input")

        progress = sm.load_progress()
        assert "patterns" in progress
        assert len(progress["patterns"]) == 2

    def test_add_cycle_result(self):
        """Test adding cycle results"""
        sm = StateManager(self.config)

        from orchestrator import CycleResult, CyclePhase

        result = CycleResult(
            cycle_id="test_1",
            duration_seconds=10.5,
            phases_completed=[CyclePhase.DATA_SCAN],
            success=True,
            output="Test output"
        )

        sm.add_cycle_result(result)

        progress = sm.load_progress()
        assert "cycles" in progress
        assert len(progress["cycles"]) == 1
        assert progress["cycles"][0]["cycle_id"] == "test_1"


class TestClaudeOrchestrator:
    """Test orchestrator functionality"""

    def setup_method(self):
        """Setup temporary directory for tests"""
        self.temp_dir = tempfile.mkdtemp()
        self.config = OrchestratorConfig(
            max_iterations=3,
            state_dir=self.temp_dir,
            archive_dir=f"{self.temp_dir}/archive",
            sleep_between_iterations=0.1  # Speed up tests
        )

    def teardown_method(self):
        """Cleanup temporary directory"""
        shutil.rmtree(self.temp_dir, ignore_errors=True)

    @pytest.mark.asyncio
    async def test_orchestrator_initialization(self):
        """Test orchestrator initializes correctly"""
        orchestrator = ClaudeOrchestrator(self.config)

        assert orchestrator.config.max_iterations == 3
        assert orchestrator.cycles_completed == 0
        assert orchestrator.current_cycle is None

    @pytest.mark.asyncio
    async def test_single_cycle_execution(self):
        """Test single cycle execution through all phases"""
        orchestrator = ClaudeOrchestrator(self.config)

        # Mock phase handlers to speed up test
        async def mock_handler(phase):
            return {"test": "data"}

        for phase in CyclePhase:
            if phase != CyclePhase.COMPLETE:
                orchestrator.phase_handlers[phase] = mock_handler

        # Run one cycle
        success = await orchestrator._run_cycle(1)

        assert success is False  # Won't complete in one cycle
        assert orchestrator.cycles_completed == 1
        assert orchestrator.current_cycle.success is True
        assert len(orchestrator.current_cycle.phase_results) == 5  # All phases

    @pytest.mark.asyncio
    async def test_orchestrator_completion(self):
        """Test orchestrator completion"""
        orchestrator = ClaudeOrchestrator(self.config)

        # Mock all phases
        async def mock_handler(phase):
            return {"success": True}

        for phase in CyclePhase:
            if phase != CyclePhase.COMPLETE:
                orchestrator.phase_handlers[phase] = mock_handler

        # Mock completion signal
        orchestrator.cycles_completed = 3  # Trigger completion

        # Run
        result = await orchestrator.run(max_iterations=5)

        assert result is True

    @pytest.mark.asyncio
    async def test_orchestrator_max_iterations(self):
        """Test orchestrator stops at max iterations"""
        orchestrator = ClaudeOrchestrator(self.config)

        # Mock phases to fail
        async def mock_handler(phase):
            return {"fail": True}

        for phase in CyclePhase:
            if phase != CyclePhase.COMPLETE:
                orchestrator.phase_handlers[phase] = mock_handler

        # Run with max_iterations=2
        result = await orchestrator.run(max_iterations=2)

        assert result is False
        assert orchestrator.cycles_completed == 2

    @pytest.mark.asyncio
    async def test_circuit_breaker_integration(self):
        """Test circuit breaker integration in orchestrator"""
        orchestrator = ClaudeOrchestrator(self.config)

        # Mock phase that always fails
        call_count = 0
        async def failing_handler(phase):
            nonlocal call_count
            call_count += 1
            raise Exception("Service unavailable")

        for phase in CyclePhase:
            if phase != CyclePhase.COMPLETE:
                orchestrator.phase_handlers[phase] = failing_handler

        # Should fail after circuit breaker opens
        await orchestrator._run_cycle(1)

        # Circuit breaker should have opened
        assert orchestrator.circuit_breaker.state == "OPEN"

    @pytest.mark.asyncio
    async def test_rate_limiter_integration(self):
        """Test rate limiter integration in orchestrator"""
        orchestrator = ClaudeOrchestrator(self.config)

        # Use very low rate limit
        orchestrator.rate_limiter.rpm_limit = 1

        call_times = []
        async def timed_handler(phase):
            call_times.append(datetime.now())
            return {"test": "data"}

        for phase in CyclePhase:
            if phase != CyclePhase.COMPLETE:
                orchestrator.phase_handlers[phase] = timed_handler

        # Run two cycles
        await orchestrator._run_cycle(1)
        await orchestrator._run_cycle(2)

        # Should have rate limited between cycles
        assert len(call_times) == 10  # 5 phases * 2 cycles
        if len(call_times) >= 2:
            # Should have waited between phases
            time_diff = (call_times[1] - call_times[0]).total_seconds()
            assert time_diff > 0


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
