import os
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SCRIPT = ROOT / "scripts/retry-on-registry-rate-limit.sh"
RATE_LIMIT = "failed to push: unexpected status from HEAD request: 429 Too Many Requests"


class RetryOnRegistryRateLimitTest(unittest.TestCase):
    def run_script(self, outcomes: list[tuple[int, str]]) -> tuple[int, int, str]:
        """Runs a command whose attempts end with the given exit codes and output, in order."""
        with tempfile.TemporaryDirectory(prefix="registry-retry-") as workspace:
            directory = Path(workspace)
            counter = directory / "attempts"
            counter.write_text("0")
            for index, (status, output) in enumerate(outcomes):
                (directory / f"status-{index}").write_text(str(status))
                (directory / f"output-{index}").write_text(output)
            command = directory / "push"
            command.write_text(
                "#!/usr/bin/env bash\n"
                'directory=$(dirname "$0")\n'
                'attempt=$(cat "$directory/attempts")\n'
                'echo $((attempt + 1)) >"$directory/attempts"\n'
                'cat "$directory/output-$attempt"\n'
                'exit "$(cat "$directory/status-$attempt")"\n'
            )
            command.chmod(0o755)

            execution = subprocess.run(
                [str(SCRIPT), str(command)],
                env={**os.environ, "REGISTRY_RATE_LIMIT_WAITS": "0 0 0"},
                capture_output=True,
                text=True,
                timeout=30,
            )
            attempts = int(counter.read_text())
        return execution.returncode, attempts, execution.stdout + execution.stderr

    def test_pushes_again_after_a_rate_limit(self) -> None:
        status, attempts, output = self.run_script([(1, RATE_LIMIT), (0, "pushed")])

        self.assertEqual(status, 0, output)
        self.assertEqual(attempts, 2)
        self.assertIn("pushed", output)

    def test_fails_after_four_attempts_when_every_attempt_is_rate_limited(self) -> None:
        status, attempts, output = self.run_script([(1, RATE_LIMIT)] * 5)

        self.assertEqual(status, 1)
        self.assertEqual(attempts, 4)
        self.assertIn("after 4 attempts", output)

    def test_fails_at_once_on_another_error(self) -> None:
        status, attempts, output = self.run_script([(3, "unauthorized: authentication required")])

        self.assertEqual(status, 3)
        self.assertEqual(attempts, 1)
        self.assertIn("unauthorized", output)

    def test_waits_60_120_and_240_seconds_by_default(self) -> None:
        self.assertIn("${REGISTRY_RATE_LIMIT_WAITS:-60 120 240}", SCRIPT.read_text())

    def test_recognizes_the_registry_error_code(self) -> None:
        status, attempts, _ = self.run_script(
            [(1, "error: toomanyrequests: rate limit exceeded"), (0, "")]
        )

        self.assertEqual(status, 0)
        self.assertEqual(attempts, 2)


if __name__ == "__main__":
    unittest.main()
