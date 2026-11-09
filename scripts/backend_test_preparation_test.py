import os
import subprocess
import tempfile
import textwrap
import unittest
from dataclasses import dataclass
from pathlib import Path

from workflow_jobs import get_jobs, get_steps

ROOT = Path(__file__).resolve().parents[1]
PREPARATION_NAMES = (
    "Install database client dependencies",
    "Prepare locked Go modules",
    "Create .env file for testing",
    "Start test containers",
    "Wait for containers to be ready",
    "Create data and temp directories",
)
PREPARATION_JOBS = ("e2e-verification-agent-podman", "test-backend")


def get_preparation_steps(job: str) -> list[str]:
    return [
        step.rstrip()
        for step in get_steps(job)
        if step.splitlines()[0].removeprefix("      - name: ") in PREPARATION_NAMES
    ]


def get_step_script(step: str) -> str:
    if "        run: |\n" in step:
        return textwrap.dedent(step.split("        run: |\n", 1)[1])

    return step.split("        run: ", 1)[1] + "\n"


@dataclass(frozen=True)
class PreparationCase:
    job_name: str
    failing_command: str = ""


@dataclass(frozen=True)
class PreparationOutcome:
    status: int
    calls: list[str]
    has_test_environment: bool
    has_backup_directory: bool
    has_temporary_directory: bool


class BackendTestPreparationTest(unittest.TestCase):
    def setUp(self) -> None:
        self.jobs = get_jobs((ROOT / ".github/workflows/ci-release.yml").read_text())

    def run_preparation(self, preparation: PreparationCase) -> PreparationOutcome:
        steps = get_preparation_steps(self.jobs[preparation.job_name])
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            (workspace / "backend").mkdir()
            (workspace / ".env.example").write_text("TEST_ENVIRONMENT=prepared\n")
            binary = workspace / "bin"
            binary.mkdir()
            command_log = workspace / "commands"
            for command in ("sudo", "go", "docker", "dpkg-architecture", "timeout"):
                executable = binary / command
                executable.write_text(
                    "#!/usr/bin/env python3\n"
                    "import os, subprocess, sys\n"
                    "from pathlib import Path\n"
                    "command = Path(sys.argv[0]).name\n"
                    'with open(os.environ["COMMAND_LOG"], "a") as calls:\n'
                    '    calls.write(command + " " + " ".join(sys.argv[1:]) + "\\n")\n'
                    'if command == os.environ["FAILING_COMMAND"]:\n'
                    '    sys.exit(124 if command == "timeout" else 17)\n'
                    'if command == "dpkg-architecture":\n'
                    '    print("test-architecture-linux-gnu")\n'
                    'if command == "go":\n'
                    '    assert Path.cwd().name == "backend"\n'
                    'if command == "timeout":\n'
                    "    sys.exit(subprocess.run(sys.argv[2:], check=False).returncode)\n"
                )
                executable.chmod(0o755)

            status = 0
            for step in steps:
                invocation = subprocess.run(
                    ["bash", "-e", "-c", get_step_script(step)],
                    cwd=workspace,
                    env={
                        **os.environ,
                        "PATH": f"{binary}:{os.environ['PATH']}",
                        "COMMAND_LOG": str(command_log),
                        "FAILING_COMMAND": preparation.failing_command,
                    },
                    capture_output=True,
                    text=True,
                    timeout=10,
                )
                status = invocation.returncode
                if status != 0:
                    break

            return PreparationOutcome(
                status=status,
                calls=command_log.read_text().splitlines(),
                has_test_environment=(workspace / ".env").exists()
                and (workspace / ".env").read_text() == (workspace / ".env.example").read_text(),
                has_backup_directory=(workspace / "databasus-data/backups").is_dir(),
                has_temporary_directory=(workspace / "databasus-data/temp").is_dir(),
            )

    def test_keeps_the_same_six_fatal_steps_in_both_jobs(self) -> None:
        podman_steps = get_preparation_steps(self.jobs["e2e-verification-agent-podman"])
        self.assertEqual(
            [step.splitlines()[0].removeprefix("      - name: ") for step in podman_steps],
            list(PREPARATION_NAMES),
        )
        for job_name in PREPARATION_JOBS:
            with self.subTest(job=job_name):
                steps = get_preparation_steps(self.jobs[job_name])
                self.assertEqual(steps, podman_steps)
                for step in steps:
                    self.assertNotRegex(
                        step, r"(?m)^        (if|continue-on-error|working-directory|shell):"
                    )
                self.assertLess(
                    self.jobs[job_name].index("Create data and temp directories"),
                    self.jobs[job_name].index(
                        "Run e2e tests on Podman" if "podman" in job_name else "Run Go tests"
                    ),
                )

    def test_prepares_database_clients_environment_and_directories_before_tests(self) -> None:
        for job_name in PREPARATION_JOBS:
            with self.subTest(job=job_name):
                prepared = self.run_preparation(PreparationCase(job_name=job_name))

                self.assertEqual(prepared.status, 0)
                self.assertTrue(prepared.has_test_environment)
                self.assertTrue(prepared.has_backup_directory)
                self.assertTrue(prepared.has_temporary_directory)
                self.assertEqual(
                    prepared.calls,
                    [
                        "sudo apt-get update -qq",
                        "sudo apt-get install -y -qq dpkg-dev libedit2 libncurses6 libtinfo6 libpq5 libmariadb3 libgnutls30 zstd netcat-openbsd",
                        "dpkg-architecture -qDEB_HOST_MULTIARCH",
                        "sudo ln -sf /usr/lib/test-architecture-linux-gnu/libncurses.so.6 /usr/lib/test-architecture-linux-gnu/libncurses.so.5",
                        "sudo ln -sf /usr/lib/test-architecture-linux-gnu/libtinfo.so.6 /usr/lib/test-architecture-linux-gnu/libtinfo.so.5",
                        "go mod tidy -diff",
                        "docker compose -f docker-compose.yml up -d backend-db-test",
                        "timeout 60 bash -c until docker exec backend-db-test pg_isready -U postgres; do sleep 2; done",
                        "docker exec backend-db-test pg_isready -U postgres",
                    ],
                )

    def test_stops_preparation_after_setup_or_database_readiness_fails(self) -> None:
        for job_name in PREPARATION_JOBS:
            for command in ("sudo", "go", "docker", "timeout"):
                with self.subTest(job=job_name, failure=command):
                    prepared = self.run_preparation(
                        PreparationCase(job_name=job_name, failing_command=command)
                    )

                    self.assertEqual(prepared.status, 124 if command == "timeout" else 17)
                    self.assertTrue(prepared.calls[-1].startswith(command + " "))
                    self.assertEqual(
                        prepared.has_test_environment, command in {"docker", "timeout"}
                    )
                    self.assertFalse(prepared.has_backup_directory)
                    self.assertFalse(prepared.has_temporary_directory)

    def test_shared_preparation_shell_passes_shellcheck(self) -> None:
        for step in get_preparation_steps(self.jobs["test-backend"]):
            with self.subTest(step=step.splitlines()[0]):
                checked = subprocess.run(
                    ["shellcheck", "--shell=bash", "-"],
                    input="set -e\n" + get_step_script(step),
                    capture_output=True,
                    text=True,
                    timeout=10,
                )

                self.assertEqual(checked.returncode, 0, checked.stdout + checked.stderr)
