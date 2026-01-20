#!/usr/bin/env python3
"""
Example: Advanced Orchestrator with Claude Code Integration
==========================================================

Demonstrates integration with Claude Code via shell commands
Shows how to use the orchestrator with real-world workflows
"""

import asyncio
import subprocess
import json
import logging
from datetime import datetime
from pathlib import Path
from typing import Dict, Any

from orchestrator import (
    OrchestratorConfig,
    ClaudeOrchestrator,
    CyclePhase,
    StateManager,
)

logger = logging.getLogger(__name__)


class ClaudeCodeOrchestrator(ClaudeOrchestrator):
    """
    Extended orchestrator with Claude Code integration
    """

    def __init__(self, config: OrchestratorConfig):
        super().__init__(config)
        self.claude_commands = {
            CyclePhase.DATA_SCAN: self._claude_data_scan,
            CyclePhase.IDEA: self._claude_idea_generation,
            CyclePhase.CODE_EDIT: self._claude_code_edit,
            CyclePhase.BACKTEST: self._claude_run_tests,
            CyclePhase.REPORT: self._claude_generate_report,
        }

    async def _run_claude_command(
        self,
        command: str,
        prompt: str,
        max_iterations: int = 1
    ) -> Dict[str, Any]:
        """
        Run a Claude Code command

        Args:
            command: Shell command to execute
            prompt: Prompt to send to Claude
            max_iterations: Max iterations for the command

        Returns:
            Dict with output, success status, etc.
        """
        logger.info(f"Executing: {command}")
        logger.info(f"Prompt: {prompt[:100]}...")

        try:
            # Execute command
            process = await asyncio.create_subprocess_shell(
                command,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                stdin=asyncio.subprocess.PIPE
            )

            stdout, stderr = await process.communicate(input=prompt.encode())

            output = stdout.decode()
            error = stderr.decode()

            result = {
                "command": command,
                "prompt": prompt,
                "stdout": output,
                "stderr": error,
                "returncode": process.returncode,
                "success": process.returncode == 0,
                "timestamp": datetime.now().isoformat()
            }

            # Check for completion signal (Ralph pattern)
            if "<promise>COMPLETE</promise>" in output:
                result["completion_signal"] = True
                logger.info("✓ Completion signal detected")

            logger.info(f"Command result: {'SUCCESS' if result['success'] else 'FAILED'}")
            if error:
                logger.warning(f"Stderr: {error[:200]}")

            return result

        except Exception as e:
            logger.error(f"Command failed: {e}")
            return {
                "command": command,
                "prompt": prompt,
                "error": str(e),
                "success": False,
                "timestamp": datetime.now().isoformat()
            }

    async def _claude_data_scan(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 1: Data Scan using Claude"""
        logger.info("Phase 1: Data Scan")

        prompt = """
        Scan the codebase and analyze:
        1. File structure and organization
        2. Technology stack and dependencies
        3. Code patterns and architecture
        4. Potential issues or improvements

        Output a JSON summary with:
        - files_scanned: count
        - tech_stack: list of technologies
        - patterns: list of patterns found
        - issues: list of potential issues
        - recommendations: list of recommendations
        """

        result = await self._run_claude_command(
            "claude -p scan --max-iterations 1",
            prompt
        )

        return {
            "phase": "data_scan",
            "result": result,
            "summary": result.get("stdout", "")
        }

    async def _claude_idea_generation(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 2: Idea Generation using Claude"""
        logger.info("Phase 2: Idea Generation")

        # Get data from previous phase
        scan_results = self.current_cycle.phase_results.get("data_scan", {})

        prompt = f"""
        Based on the data scan results, generate 3-5 actionable ideas for improvement.

        Previous scan results:
        {json.dumps(scan_results, indent=2)}

        For each idea, provide:
        1. Title
        2. Priority (high/medium/low)
        3. Effort estimate (low/medium/high)
        4. Reasoning
        5. Expected impact

        Select the best idea and explain why.
        """

        result = await self._run_claude_command(
            "claude -p idea --max-iterations 1",
            prompt
        )

        return {
            "phase": "idea",
            "result": result,
            "selected_idea": self._extract_selected_idea(result.get("stdout", "")),
            "summary": result.get("stdout", "")
        }

    async def _claude_code_edit(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 3: Code Edit using Claude"""
        logger.info("Phase 3: Code Edit")

        # Get selected idea from previous phase
        idea = self.current_cycle.phase_results.get("idea", {}).get("selected_idea", {})

        prompt = f"""
        Implement the following idea:

        {json.dumps(idea, indent=2)}

        Requirements:
        1. Make minimal, focused changes
        2. Follow existing code patterns
        3. Add comments explaining changes
        4. Ensure code compiles/builds successfully
        5. Do NOT commit changes - just implement

        Report:
        - files_changed: list of files modified
        - lines_added: count
        - lines_removed: count
        - changes_summary: description of changes
        """

        result = await self._run_claude_command(
            "claude -p edit --max-iterations 1",
            prompt
        )

        return {
            "phase": "code_edit",
            "result": result,
            "summary": result.get("stdout", "")
        }

    async def _claude_run_tests(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 4: Backtest using Claude"""
        logger.info("Phase 4: Backtest (Run Tests)")

        prompt = """
        Run tests and validate changes:

        1. Run all tests (unit, integration, etc.)
        2. Check code coverage
        3. Verify code quality (linting, type checking)
        4. Check for security issues
        5. Measure performance metrics

        Report:
        - tests_run: count
        - tests_passed: count
        - tests_failed: count
        - coverage: percentage
        - quality_score: 1-10
        - performance: metrics
        - status: PASSED/FAILED
        """

        result = await self._run_claude_command(
            "claude -p test --max-iterations 1",
            prompt
        )

        return {
            "phase": "backtest",
            "result": result,
            "summary": result.get("stdout", "")
        }

    async def _claude_generate_report(self, phase: CyclePhase) -> Dict[str, Any]:
        """Phase 5: Report using Claude"""
        logger.info("Phase 5: Generate Report")

        # Aggregate all phase results
        all_results = {
            k: v for k, v in self.current_cycle.phase_results.items()
        }

        prompt = f"""
        Generate a comprehensive report for this cycle.

        Phase results:
        {json.dumps(all_results, indent=2)}

        Include:
        1. Executive Summary
        2. Phase-by-phase results
        3. Metrics and KPIs
        4. Learnings and patterns discovered
        5. Next steps and recommendations

        Format as a markdown report.
        """

        result = await self._run_claude_command(
            "claude -p report --max-iterations 1",
            prompt
        )

        return {
            "phase": "report",
            "result": result,
            "summary": result.get("stdout", "")
        }

    def _extract_selected_idea(self, output: str) -> Dict[str, Any]:
        """Extract selected idea from Claude output"""
        # Simple extraction - in real implementation, use more robust parsing
        if "SELECTED IDEA:" in output.upper():
            return {"title": "Extracted from Claude", "output": output}
        return {"title": "Idea generation complete", "output": output}


async def main():
    """Run advanced orchestrator example with Claude Code"""

    config = OrchestratorConfig(
        max_iterations=3,
        sleep_between_iterations=2.0,
        requests_per_minute=60,
        requests_per_hour=1000,
        circuit_breaker_failure_threshold=5,
        state_dir="./advanced_example_state",
        archive_dir="./advanced_example_archive"
    )

    logger.info("="*60)
    logger.info("Advanced Orchestrator Example with Claude Code Integration")
    logger.info("="*60)

    # Check if Claude is available
    try:
        result = subprocess.run(
            ["which", "claude"],
            capture_output=True,
            text=True,
            timeout=5
        )
        if result.returncode != 0:
            logger.warning("Claude CLI not found - using mock mode")
            logger.warning("Install Claude CLI for full functionality")
    except Exception as e:
        logger.warning(f"Could not check Claude: {e}")

    # Create orchestrator with Claude integration
    orchestrator = ClaudeCodeOrchestrator(config)

    # Run orchestrator
    completed = await orchestrator.run(
        max_iterations=3,
        current_branch="main"
    )

    # Save final report
    progress = orchestrator.state_manager.load_progress()
    report_file = Path("final_report.json")
    report_file.write_text(json.dumps(progress, indent=2))
    logger.info(f"Final report saved to: {report_file}")

    if completed:
        logger.info("\n" + "="*60)
        logger.info("✓ Advanced orchestrator completed successfully!")
        logger.info("="*60)
    else:
        logger.warning("\n" + "="*60)
        logger.warning("✗ Advanced orchestrator did not complete")
        logger.warning("="*60)

    return completed


if __name__ == "__main__":
    result = asyncio.run(main())
    exit(0 if result else 1)
