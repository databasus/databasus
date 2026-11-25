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
loader_path = os.environ.get("LD_LIBRARY_PATH", "")
shim_directory = loader_path.split(":")[0]
print("LD_LIBRARY_PATH=" + loader_path)
print("shim provides libedit.so.2: " + str(os.path.islink(shim_directory + "/libedit.so.2")))
sys.exit(int(os.environ.get("FAKE_ENVIRONMENT_CHECK_EXIT_CODE", "0")))
"""
FAILING_LIBEDIT_SHIM = """apply_libedit_shim() {
    echo "libedit.so.0 not found" >&2
    return 1
}
"""
LDCONFIG_WITH_LIBEDIT_2 = "#!/bin/sh\nprintf '%s\\n' 'libedit.so.2 (libc6,x86-64) => /lib/x86_64-linux-gnu/libedit.so.2'\n"


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
        self.commands = self.workspace / "bin"
        write_executable(
            self.workspace / "scripts/doctor.sh",
            (REPOSITORY_ROOT / "scripts/doctor.sh").read_text(),
        )
        write_executable(self.workspace / ".agent-hooks/run.sh", FAKE_HOOK_CHECK)
        (self.workspace / "scripts/development_environment_check.py").write_text(
            FAKE_ENVIRONMENT_CHECK
        )
        (self.workspace / "scripts/fedora.sh").write_text(
            (REPOSITORY_ROOT / "scripts/fedora.sh").read_text()
        )
        write_executable(self.commands / "ldconfig", LDCONFIG_WITH_LIBEDIT_2)
        self.temporary_directory = self.workspace / "tmp"
        self.temporary_directory.mkdir()
        (self.workspace / "home").mkdir()

    def run_doctor(self, hook_check_exit_code: int, environment_check_exit_code: int) -> DoctorRun:
        completed = subprocess.run(
            [str(self.workspace / "scripts/doctor.sh"), "--client", "claude"],
            env={
                **os.environ,
                "PATH": f"{self.commands}:{os.environ['PATH']}",
                "HOME": str(self.workspace / "home"),
                "TMPDIR": str(self.temporary_directory),
                "LD_LIBRARY_PATH": "",
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

    def test_runs_the_environment_check_after_the_libedit_shim_fails(self) -> None:
        (self.workspace / "scripts/fedora.sh").write_text(FAILING_LIBEDIT_SHIM)

        doctor_run = self.run_doctor(0, 0)

        self.assertEqual(doctor_run.exit_code, 0, doctor_run.stderr)
        self.assertIn("libedit.so.0 not found", doctor_run.stderr)
        self.assertIn("environment report", doctor_run.stdout)

    def test_runs_the_environment_check_with_a_temporary_shim_it_removes_afterwards(
        self,
    ) -> None:
        fake_libedit = self.workspace / "lib/libedit.so.0"
        fake_libedit.parent.mkdir()
        fake_libedit.touch()
        write_executable(
            self.commands / "ldconfig",
            f"#!/bin/sh\nprintf '%s\\n' 'libedit.so.0 (libc6,x86-64) => {fake_libedit}'\n",
        )

        doctor_run = self.run_doctor(0, 0)

        self.assertEqual(doctor_run.exit_code, 0, doctor_run.stderr)
        self.assertIn(f"LD_LIBRARY_PATH={self.temporary_directory}/", doctor_run.stdout)
        self.assertIn("shim provides libedit.so.2: True", doctor_run.stdout)
        self.assertEqual(list(self.temporary_directory.iterdir()), [])
        self.assertEqual(list((self.workspace / "home").iterdir()), [])


if __name__ == "__main__":
    unittest.main()
