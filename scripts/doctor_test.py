import os
import shutil
import subprocess
import tempfile
import unittest
from dataclasses import dataclass
from pathlib import Path

REPOSITORY_ROOT = Path(__file__).resolve().parent.parent
FAKE_HOOK_CHECK = """#!/usr/bin/env bash
printf 'hook tool report: %s\\n' "$*"
exit "${FAKE_HOOK_CHECK_EXIT_CODE:-0}"
"""
FAKE_ENVIRONMENT_CHECK = """import os
import sys

print("environment report")
sys.exit(int(os.environ.get("FAKE_ENVIRONMENT_CHECK_EXIT_CODE", "0")))
"""


@dataclass(frozen=True)
class DoctorRun:
    exit_code: int
    stdout: str
    stderr: str


def write_executable(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content)
    path.chmod(0o755)


class DoctorTest(unittest.TestCase):
    def setUp(self) -> None:
        self.workspace = Path(tempfile.mkdtemp(prefix="doctor-"))
        self.addCleanup(shutil.rmtree, self.workspace, ignore_errors=True)
        write_executable(
            self.workspace / "scripts/doctor.sh",
            (REPOSITORY_ROOT / "scripts/doctor.sh").read_text(),
        )
        write_executable(self.workspace / ".agent-hooks/run.sh", FAKE_HOOK_CHECK)
        (self.workspace / "scripts/development_environment_check.py").write_text(
            FAKE_ENVIRONMENT_CHECK
        )

    def run_doctor(self, hook_check_exit_code: int, environment_check_exit_code: int) -> DoctorRun:
        completed = subprocess.run(
            [str(self.workspace / "scripts/doctor.sh"), "--client", "claude"],
            env={
                **os.environ,
                "FAKE_HOOK_CHECK_EXIT_CODE": str(hook_check_exit_code),
                "FAKE_ENVIRONMENT_CHECK_EXIT_CODE": str(environment_check_exit_code),
            },
            capture_output=True,
            text=True,
            check=False,
        )

        return DoctorRun(completed.returncode, completed.stdout, completed.stderr)

    def test_prints_both_reports_and_exits_with_the_combined_outcome(self) -> None:
        for hook_check_exit_code, environment_check_exit_code, expected_exit_code in (
            (0, 0, 0),
            (1, 0, 1),
            (0, 1, 1),
            (1, 1, 1),
            (2, 1, 2),
        ):
            with self.subTest(
                hook_check_exit_code=hook_check_exit_code,
                environment_check_exit_code=environment_check_exit_code,
            ):
                doctor_run = self.run_doctor(hook_check_exit_code, environment_check_exit_code)

                self.assertEqual(doctor_run.exit_code, expected_exit_code, doctor_run.stderr)
                self.assertIn("hook tool report: doctor --client claude", doctor_run.stdout)
                self.assertIn("environment report", doctor_run.stdout)
                self.assertLess(
                    doctor_run.stdout.index("hook tool report"),
                    doctor_run.stdout.index("environment report"),
                )
                verdict = doctor_run.stdout.rstrip("\n").splitlines()[-1]
                self.assertEqual(
                    verdict,
                    "All checks passed"
                    if expected_exit_code == 0
                    else "Some checks failed, see the lines above",
                )


if __name__ == "__main__":
    unittest.main()
