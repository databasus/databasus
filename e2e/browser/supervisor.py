"""Owns one run of the browser end-to-end suite: the image, the containers, pytest and their removal."""

import ctypes
import os
import secrets
import signal
import subprocess
import sys
from pathlib import Path

from compose_project import ComposeProject, list_remaining_resources
from leftover_sweep import sweep_dead_runs
from output_masking import OutputMask
from supervisor_file import (
    SupervisorRecord,
    get_process_start_time,
    get_project_name,
    get_run_directory,
    remove_run_directory,
    write_supervisor_record,
)

SUITE_DIRECTORY = Path(__file__).parent
REPOSITORY_ROOT = SUITE_DIRECTORY.parent.parent
IMAGE_VARIABLE = "DATABASUS_IMAGE"
EXIT_FAILED = 1
EXIT_INTERRUPTED = 130
LOG_TAIL_LINES = "200"
PR_SET_PDEATHSIG = 1
KNOWN_PROBES = {"scenario", "interrupt", "secrets", "passing", "refused-form"}
PROBE_SCENARIOS = {"refused-form": "scenarios/test_critical_path.py"}
DEFAULT_PROBE_SCENARIOS = "scenarios/test_probes.py"
APPLICATION_ADDRESS = "http://databasus:4005"


class RunInterrupted(BaseException):
    """Not an Exception, so the handler of ordinary failures cannot swallow an interruption."""


class Supervisor:
    def __init__(self) -> None:
        self.run_id = secrets.token_hex(6)
        self.probe = os.environ.get("E2E_PROBE", "")
        self.named_image = os.environ.get(IMAGE_VARIABLE, "")
        self.built_image = None if self.named_image else f"{get_project_name(self.run_id)}:candidate"
        self.account_password = (self.probe and os.environ.get("E2E_PROBE_PASSWORD")) or secrets.token_urlsafe(24)
        self.database_password = (self.probe and os.environ.get("E2E_PROBE_DATABASE_PASSWORD")) or secrets.token_urlsafe(24)
        self.output_mask = OutputMask([self.account_password, self.database_password])
        self.project = ComposeProject(
            name=get_project_name(self.run_id),
            image=self.named_image or self.built_image,
            database_password=self.database_password,
        )

    def run(self) -> int:
        exit_code = self._run_until_signals_are_ignored()
        container_logs = self._get_masked_container_logs()
        cleanup_exit_code = self._clean_up()

        if exit_code != 0 or cleanup_exit_code != 0:
            for line in container_logs.splitlines():
                self._report(line)

        return cleanup_exit_code or exit_code

    def _run_until_signals_are_ignored(self) -> int:
        """The outer handler also covers a signal that arrives between the scenarios and ignoring signals."""
        try:
            try:
                exit_code = self._run_until_cleanup()
            except Exception as failure:
                self._report(f"E2E run failed: {failure}")
                exit_code = EXIT_FAILED

            _ignore_signals()
        except RunInterrupted:
            self._report("E2E run interrupted")
            exit_code = EXIT_INTERRUPTED

        return exit_code

    def _run_until_cleanup(self) -> int:
        write_supervisor_record(
            SupervisorRecord(
                run_id=self.run_id,
                pid=os.getpid(),
                process_start_time=get_process_start_time(os.getpid()),
                built_image=self.built_image,
            )
        )
        self._report(f"E2E run: {self.run_id}; project: {self.project.name}; supervisor PID: {os.getpid()}")
        self._report(f"E2E image under test: {self.project.image}")

        sweep_dead_runs(self.run_id, self._report)

        if self.built_image:
            self._report(f"E2E building {self.built_image}")
            build_exit_code = self._run_masked(["docker", "build", "--tag", self.built_image, str(REPOSITORY_ROOT)])
            if build_exit_code != 0:
                self._report("E2E image build failed")
                return EXIT_FAILED

        up_exit_code = self._run_masked(
            self.project.get_command("up", "--detach", "--wait", "--quiet-pull"),
            environment=self.project.get_environment(),
        )
        if up_exit_code != 0:
            self._report("E2E start-up failed")
            return EXIT_FAILED

        return self._run_scenarios()

    def _run_scenarios(self) -> int:
        browser_port = self.project.get_browser_port()
        email = f"admin-{self.run_id}@example.com"
        self._report(f"E2E browser port: {browser_port}; account: {email}")

        scenarios = PROBE_SCENARIOS.get(self.probe, DEFAULT_PROBE_SCENARIOS) if self.probe else "scenarios"
        environment = {
            **self.project.get_environment(),
            "E2E_BASE_URL": APPLICATION_ADDRESS,
            "E2E_BROWSER_ENDPOINT": f"ws://127.0.0.1:{browser_port}/",
            "E2E_COMPOSE_PROJECT": self.project.name,
            "E2E_EMAIL": email,
            "E2E_PASSWORD": self.account_password,
            "PYTHONUNBUFFERED": "1",
        }

        return self._run_masked(
            [
                sys.executable, "-m", "pytest", "--capture=no",
                "--basetemp", str(get_run_directory(self.run_id) / "pytest"), scenarios,
            ],
            environment=environment,
            working_directory=SUITE_DIRECTORY,
        )

    def _get_masked_container_logs(self) -> str:
        try:
            completed = subprocess.run(
                self.project.get_command("logs", "--no-color", "--tail", LOG_TAIL_LINES),
                env=self.project.get_environment(),
                capture_output=True,
                text=True,
                errors="replace",
            )

            return self.output_mask.apply(completed.stdout + completed.stderr)
        except Exception as failure:
            return self.output_mask.apply(f"E2E could not read the container logs: {failure}")

    def _clean_up(self) -> int:
        try:
            down_exit_code = self._run_masked(
                self.project.get_command("down", "--volumes", "--remove-orphans", "--timeout", "5"),
                environment=self.project.get_environment(),
            )
            remaining_resources = list_remaining_resources(self.project.name)
        except Exception as failure:
            self._report(f"E2E cleanup failed, a later run removes {self.project.name}: {failure}")
            return EXIT_FAILED

        if self.built_image:
            image_removal = subprocess.run(["docker", "image", "rm", self.built_image], capture_output=True, text=True)
            if image_removal.returncode != 0 and "No such image" not in image_removal.stderr:
                remaining_resources.append(f"image {self.built_image}")

        if down_exit_code != 0 or remaining_resources:
            self._report(f"E2E cleanup failed, remaining: {', '.join(remaining_resources) or self.project.name}")
            return EXIT_FAILED

        remove_run_directory(self.run_id)

        return 0

    def _run_masked(
        self,
        command: list[str],
        environment: dict[str, str] | None = None,
        working_directory: Path | None = None,
    ) -> int:
        # The suite requires Python 3.14 or newer, which supports the errors argument.
        # nosemgrep: python.lang.compatibility.python36.python36-compatibility-Popen1
        child = subprocess.Popen(
            command,
            env=environment,
            cwd=working_directory,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            errors="replace",
            start_new_session=True,
            preexec_fn=_terminate_with_parent,
        )

        try:
            for line in child.stdout:
                self._report(line.rstrip("\n"))

            return child.wait()
        except RunInterrupted:
            os.killpg(child.pid, signal.SIGTERM)
            child.wait()
            raise

    def _report(self, line: str) -> None:
        try:
            print(self.output_mask.apply(line), flush=True)
        except BrokenPipeError:
            self._stop_printing()

    def _stop_printing(self) -> None:
        """The command that relayed this output is gone, and the run must still remove what it started."""
        sys.stdout = open(os.devnull, "w")
        os.dup2(sys.stdout.fileno(), 1)


def _terminate_with_parent() -> None:
    ctypes.CDLL(None).prctl(PR_SET_PDEATHSIG, signal.SIGTERM)


def _ignore_signals() -> None:
    signal.signal(signal.SIGINT, signal.SIG_IGN)
    signal.signal(signal.SIGTERM, signal.SIG_IGN)


def _raise_interrupted(_signal_number: int, _frame: object) -> None:
    _ignore_signals()

    raise RunInterrupted


def main() -> int:
    if os.environ.get("CI") and not os.environ.get(IMAGE_VARIABLE):
        print(f"E2E refuses to run in CI without {IMAGE_VARIABLE}: it tests the image CI built", flush=True)
        return EXIT_FAILED

    probe = os.environ.get("E2E_PROBE", "")
    if probe and probe not in KNOWN_PROBES:
        print(f"E2E refuses the unknown probe {probe}: no scenario would run", flush=True)
        return EXIT_FAILED

    signal.signal(signal.SIGINT, _raise_interrupted)
    signal.signal(signal.SIGTERM, _raise_interrupted)

    return Supervisor().run()


if __name__ == "__main__":
    sys.exit(main())
