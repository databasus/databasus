"""Drives the suite's public commands as subprocesses, for the tests of its lifecycle guarantees."""

import ctypes
import os
import re
import signal
import subprocess
import threading
import time
from dataclasses import dataclass
from pathlib import Path

REPOSITORY_ROOT = Path(__file__).parent.parent.parent
RUN_LIMIT_SECONDS = 900
PR_SET_PDEATHSIG = 1
_POLL_SECONDS = 0.2


_live_runs: set["PublicRun"] = set()


def _terminate_with_parent() -> None:
    ctypes.CDLL(None).prctl(PR_SET_PDEATHSIG, signal.SIGTERM)


@dataclass(frozen=True)
class FinishedRun:
    exit_code: int
    output: str


class PublicRun:
    def __init__(
        self,
        command: tuple[str, ...] = ("mise", "run", "e2e:browser"),
        environment: dict[str, str] | None = None,
        working_directory: Path = REPOSITORY_ROOT,
    ) -> None:
        run_environment = {**os.environ, **(environment or {})}

        for variable in ("VIRTUAL_ENV", "PYTEST_CURRENT_TEST"):
            run_environment.pop(variable, None)

        if not (environment or {}).get("CI"):
            run_environment.pop("CI", None)

        # The suite requires Python 3.14 or newer, which supports the errors argument.
        # nosemgrep: python.lang.compatibility.python36.python36-compatibility-Popen1
        self._process = subprocess.Popen(
            command,
            cwd=working_directory,
            env=run_environment,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            errors="replace",
            start_new_session=True,
            preexec_fn=_terminate_with_parent,
        )
        self._output_lines: list[str] = []
        _live_runs.add(self)
        self._reader = threading.Thread(target=self._read_output, daemon=True)
        self._reader.start()

    def _read_output(self) -> None:
        for line in self._process.stdout:
            self._output_lines.append(line)

    def get_output(self) -> str:
        return "".join(self._output_lines)

    def wait_for_output(self, text: str, limit_seconds: int = RUN_LIMIT_SECONDS) -> None:
        deadline = time.monotonic() + limit_seconds

        while text not in self.get_output():
            assert self._process.poll() is None, f"the run ended before printing {text}:\n{self.get_output()}"
            assert time.monotonic() < deadline, f"the run did not print {text}:\n{self.get_output()}"
            time.sleep(_POLL_SECONDS)

    def get_project_name(self) -> str:
        self.wait_for_output("project: ")

        return re.search(r"project: (\S+?);", self.get_output()).group(1)

    def get_run_id(self) -> str:
        return self.get_project_name().removeprefix("databasus-e2e-")

    def get_supervisor_pid(self) -> int:
        self.wait_for_output("supervisor PID: ")

        return int(re.search(r"supervisor PID: (\d+)", self.get_output()).group(1))

    def send_signal_to_the_whole_command(self, signal_number: signal.Signals) -> None:
        try:
            os.killpg(self._process.pid, signal_number)
        except ProcessLookupError:
            return

    def terminate_supervisor(self) -> None:
        try:
            os.kill(self.get_supervisor_pid(), signal.SIGTERM)
        except ProcessLookupError:
            return

    def owns_process(self, pid: int) -> bool:
        return os.getsid(pid) == self._process.pid

    def is_running(self) -> bool:
        return self._process.poll() is None

    def finish(self, limit_seconds: int = RUN_LIMIT_SECONDS) -> FinishedRun:
        exit_code = self._process.wait(timeout=limit_seconds)
        self._wait_for_supervisor_to_finish(limit_seconds)
        self._reader.join(timeout=limit_seconds)
        _live_runs.discard(self)

        return FinishedRun(exit_code, self.get_output())

    def _wait_for_supervisor_to_finish(self, limit_seconds: int) -> None:
        """A signal can end the relaying command first, while the supervisor is still removing what it started."""
        supervisor_pid = re.search(r"supervisor PID: (\d+)", self.get_output())

        if supervisor_pid:
            wait_for_process_to_end(int(supervisor_pid.group(1)), limit_seconds)


def get_running_image(project_name: str, service: str) -> str:
    completed = subprocess.run(
        ["docker", "inspect", "--format", "{{.Config.Image}}", f"{project_name}-{service}-1"],
        capture_output=True,
        text=True,
        check=True,
    )

    return completed.stdout.strip()


def wait_for_process_to_end(pid: int, limit_seconds: int = RUN_LIMIT_SECONDS) -> None:
    deadline = time.monotonic() + limit_seconds

    while _is_process_running(pid):
        assert time.monotonic() < deadline, f"process {pid} did not end"
        time.sleep(_POLL_SECONDS)


def _is_process_running(pid: int) -> bool:
    try:
        return Path(f"/proc/{pid}/stat").read_text().rpartition(")")[2].split()[0] != "Z"
    except OSError:
        return False


def stop_live_runs() -> None:
    for run in list(_live_runs):
        run.terminate_supervisor()

    for run in list(_live_runs):
        run.finish()
