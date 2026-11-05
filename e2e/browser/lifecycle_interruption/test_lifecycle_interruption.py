import os
import shlex
import signal
import subprocess
import time
from collections.abc import Callable, Iterator
from pathlib import Path

import pytest
from docker_wrapper import install_docker_wrapper_and_get_path
from leftover_check import is_image_present
from lifecycle.run_cleanup import remove_run_resources
from public_command import PublicRun, wait_for_process_to_end
from supervisor_file import SupervisorRecord, get_run_directory, read_supervisor_records

LIFECYCLE_COMMAND = ("mise", "run", "e2e:lifecycle")
PROBES_IMAGE_SUFFIX = ":probes"
BUILD_LIMIT_SECONDS = 900
RELEASE_LIMIT_SECONDS = 300


@pytest.fixture(autouse=True)
def no_named_image(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("DATABASUS_IMAGE", raising=False)


def _wait_until(description: str, condition: Callable[[], bool], limit_seconds: int) -> None:
    deadline = time.monotonic() + limit_seconds

    while not condition():
        assert time.monotonic() < deadline, f"gave up waiting for {description}"
        time.sleep(1)


def is_running_in(lifecycle_run: PublicRun, pid: int) -> bool:
    """Another worktree's lifecycle command also records a probes image, and must not get this test's signal."""
    try:
        return lifecycle_run.owns_process(pid)
    except ProcessLookupError:
        return False


def _find_built_probes_record(lifecycle_run: PublicRun) -> SupervisorRecord | None:
    for record in read_supervisor_records():
        if (
            is_running_in(lifecycle_run, record.pid)
            and record.built_image
            and record.built_image.endswith(PROBES_IMAGE_SUFFIX)
            and is_image_present(record.built_image)
        ):
            return record

    return None


@pytest.fixture
def lifecycle_run() -> Iterator[PublicRun]:
    run = PublicRun(
        command=LIFECYCLE_COMMAND,
        environment={
            "PYTEST_ADDOPTS": (
                '-k "fails_with_the_diagnostic_and_removes_everything_when_a_scenario_fails '
                'or test_fails_after_the_application_and_the_browser_are_ready"'
            ),
        },
    )

    built_probes_record = None
    try:
        built_probes_record = _wait_for_built_probes_record(run)
        yield run
    finally:
        try:
            still_running_pids = [
                record.pid for record in read_supervisor_records() if is_running_in(run, record.pid)
            ]
            run.send_signal_to_the_whole_command(signal.SIGTERM)
            run.finish()

            for pid in still_running_pids:
                wait_for_process_to_end(pid)
        finally:
            if built_probes_record is not None:
                remove_run_resources(
                    run,
                    built_probes_record,
                    unrelated_container=f"unrelated-to-e2e-{built_probes_record.run_id}",
                )


def _wait_for_built_probes_record(lifecycle_run: PublicRun) -> SupervisorRecord:
    def is_image_built() -> bool:
        assert lifecycle_run.is_running(), (
            f"the lifecycle command ended early:\n{lifecycle_run.get_output()}"
        )

        return _find_built_probes_record(lifecycle_run) is not None

    _wait_until("the lifecycle command's image", is_image_built, BUILD_LIMIT_SECONDS)

    record = _find_built_probes_record(lifecycle_run)
    assert record is not None
    return record


def _get_probe_sessions(lifecycle_pid: int) -> set[int]:
    """Each probe command leads its own session, which still names it after the kill ends its parent."""
    probe_sessions = set()

    for process_stat in Path("/proc").glob("[0-9]*/stat"):
        try:
            fields_after_command = process_stat.read_text().rpartition(")")[2].split()
        except OSError:
            continue

        if int(fields_after_command[1]) == lifecycle_pid:
            probe_sessions.add(int(process_stat.parent.name))

    return probe_sessions


def _is_in_sessions(pid: int, sessions: set[int]) -> bool:
    try:
        return os.getsid(pid) in sessions
    except ProcessLookupError:
        return False


def _have_probe_runs_released_their_resources(probe_sessions: set[int]) -> bool:
    return not [
        record
        for record in read_supervisor_records()
        if _is_in_sessions(record.pid, probe_sessions)
    ]


class TestLifecycleInterruption:
    def test_removes_the_unrelated_container_when_interrupted_during_readiness(
        self, tmp_path: Path
    ) -> None:
        readiness_marker = tmp_path / "readiness-container"
        path_with_blocked_readiness = install_docker_wrapper_and_get_path(
            tmp_path,
            f'*" pg_isready "*) printf \'%s %s\' "$2" "$PPID" > {shlex.quote(str(readiness_marker))}; exit 1;;',
        )
        run = PublicRun(
            command=(
                "uv",
                "run",
                "--locked",
                "--project",
                "e2e/browser",
                "pytest",
                "e2e/browser/lifecycle/test_failing_runs.py",
                "-k",
                "refuses_an_unknown_probe",
            ),
            environment={"PATH": path_with_blocked_readiness},
        )

        try:
            _wait_until(
                "the unrelated container's readiness check",
                readiness_marker.exists,
                RELEASE_LIMIT_SECONDS,
            )
            container_name, fixture_pid = readiness_marker.read_text().split()
            run.send_signal_to_the_whole_command(signal.SIGTERM)
            finished_run = run.finish()
            wait_for_process_to_end(int(fixture_pid))

            assert finished_run.exit_code != 0
            container_status = subprocess.run(
                ["docker", "inspect", "--format", "{{.Name}}", container_name], capture_output=True
            )
            assert container_status.returncode != 0, finished_run.output
        finally:
            run.send_signal_to_the_whole_command(signal.SIGTERM)
            run.finish()

            if readiness_marker.exists():
                subprocess.run(
                    [
                        "docker",
                        "rm",
                        "--force",
                        "--volumes",
                        readiness_marker.read_text().split()[0],
                    ],
                    capture_output=True,
                )

    def test_removes_its_image_when_interrupted(self, lifecycle_run: PublicRun) -> None:
        record = _wait_for_built_probes_record(lifecycle_run)

        os.kill(record.pid, signal.SIGTERM)
        lifecycle_run.finish()
        wait_for_process_to_end(record.pid)

        assert record.built_image is not None
        assert not is_image_present(record.built_image)
        assert not get_run_directory(record.run_id).exists()

    def test_the_next_run_removes_its_image_and_directory_when_killed(
        self, lifecycle_run: PublicRun
    ) -> None:
        record = _wait_for_built_probes_record(lifecycle_run)
        probe_sessions = _get_probe_sessions(record.pid)

        lifecycle_run.send_signal_to_the_whole_command(signal.SIGKILL)
        lifecycle_run.finish()
        _wait_until(
            "the probe runs of the killed lifecycle command to release their resources",
            lambda: _have_probe_runs_released_their_resources(probe_sessions),
            RELEASE_LIMIT_SECONDS,
        )

        PublicRun(environment={"E2E_PROBE": "scenario"}).finish()

        assert record.built_image is not None
        assert not is_image_present(record.built_image)
        assert not get_run_directory(record.run_id).exists()
